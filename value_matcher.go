package quamina

import (
	"bytes"
	"sync/atomic"
)

// valueMatcher represents a byte-driven finite automaton (FA).  The table needs to be the
// equivalent of a map[byte]nextState and is represented by smallTable.
// In this implementation all the FAs are nondeterministic, which means each
// byte can cause transfers to multiple other states. The basic algorithm is to compute the FA
// for a pattern and merge with any existing FA.
// In some (common) cases there is only one byte sequence forward from a state,
// i.e. a string-valued field with only one string match. In this case, the FA
// will be null and the value being matched has to exactly equal the singletonMatch
// field; if so, the singletonTransition is the return value. This is to avoid
// having a long chain of smallTables each with only one entry.
// To allow for concurrent access between one thread running AddPattern and many
// others running MatchesForEvent, the valueMatcher payload is stored in an
// atomic.Value
type valueMatcher struct {
	updateable atomic.Value // always contains *vmFields
}
type vmFields struct {
	startTable          *smallTable
	singletonMatch      []byte
	singletonTransition *fieldMatcher
}

func (m *valueMatcher) getFields() *vmFields {
	return m.updateable.Load().(*vmFields)
}

func (m *valueMatcher) getFieldsForUpdate() *vmFields {
	current := m.updateable.Load().(*vmFields)
	freshState := *current
	return &freshState
}

func (m *valueMatcher) update(state *vmFields) {
	m.updateable.Store(state)
}

func newValueMatcher() *valueMatcher {
	var vm valueMatcher
	vm.update(&vmFields{})
	return &vm
}

func (m *valueMatcher) transitionOn(val []byte) []*fieldMatcher {
	var transitions []*fieldMatcher

	fields := m.getFields()

	switch {
	case fields.singletonMatch != nil:
		// if there's a singleton entry here, we either match the val or we're
		// done Note: We have to check this first because addTransition might be
		// busy constructing an automaton, but it's not ready for use yet.  When
		// it's done it'll zero out the singletonMatch
		if bytes.Equal(fields.singletonMatch, val) {
			transitions = append(transitions, fields.singletonTransition)
		}
		return transitions

	case fields.startTable != nil:
		return traverseFA(fields.startTable, val, transitions)

	default:
		// no FA, no singleton, nothing to do, this probably can't happen because a flattener
		// shouldn't preserve a field that hasn't appeared in a pattern
		return transitions
	}
}

func (m *valueMatcher) addTransition(val typedVal, _ printer) *fieldMatcher {
	valBytes := []byte(val.val)
	fields := m.getFieldsForUpdate()

	// there's already a table, thus an out-degree > 1
	if fields.startTable != nil {
		var newFA *smallTable
		var nextField *fieldMatcher
		switch val.vType {
		case stringType, numberType, literalType:
			newFA, nextField = makeStringFA(valBytes, nil)
		case anythingButType:
			newFA, nextField = makeMultiAnythingButFA(val.list)
		case shellStyleType:
			newFA, nextField = makeShellStyleAutomaton(valBytes, &nullPrinter{})
		case prefixType:
			newFA, nextField = makePrefixAutomaton(valBytes)
		default:
			panic("unknown value type")
		}
		fields.startTable = mergeFAs(fields.startTable, newFA)
		m.update(fields)
		return nextField
	}

	// no start table, we have to work with singletons …

	// … unless this is completely virgin, in which case put in the singleton,
	// assuming it's just a string match
	if fields.singletonMatch == nil {
		switch val.vType {
		case stringType, numberType, literalType:
			fields.singletonMatch = valBytes
			fields.singletonTransition = newFieldMatcher()
			m.update(fields)
			return fields.singletonTransition
		case anythingButType:
			newFA, nextField := makeMultiAnythingButFA(val.list)
			fields.startTable = newFA
			m.update(fields)
			return nextField
		case shellStyleType:
			newAutomaton, nextField := makeShellStyleAutomaton(valBytes, &nullPrinter{})
			fields.startTable = newAutomaton
			m.update(fields)
			return nextField
		case prefixType:
			newFA, nextField := makePrefixAutomaton(valBytes)
			fields.startTable = newFA
			m.update(fields)
			return nextField
		default:
			panic("unknown value type")
		}
	}

	// singleton match is here and this value matches it
	if val.vType == stringType || val.vType == numberType || val.vType == literalType {
		if bytes.Equal(fields.singletonMatch, valBytes) {
			return fields.singletonTransition
		}
	}

	// singleton is here, we don't match, so our outdegree becomes 2, so we have
	// to build an automaton with two values in it
	singletonAutomaton, _ := makeStringFA(fields.singletonMatch, fields.singletonTransition)
	var nextField *fieldMatcher
	var newFA *smallTable
	switch val.vType {
	case stringType, numberType, literalType:
		newFA, nextField = makeStringFA(valBytes, nil)
	case anythingButType:
		newFA, nextField = makeMultiAnythingButFA(val.list)
	case shellStyleType:
		newFA, nextField = makeShellStyleAutomaton(valBytes, &nullPrinter{})
	case prefixType:
		newFA, nextField = makePrefixAutomaton(valBytes)
	default:
		panic("unknown value type")
	}

	// now table is ready for use, nuke singleton to signal threads to use it
	fields.startTable = mergeFAs(singletonAutomaton, newFA)
	// fmt.Println("Merged: " + fields.startTable.dump())
	fields.singletonMatch = nil
	fields.singletonTransition = nil
	m.update(fields)
	return nextField
}

func makePrefixAutomaton(val []byte) (*smallTable, *fieldMatcher) {
	nextField := newFieldMatcher()
	return onePrefixStep(val, 0, nextField), nextField
}

func onePrefixStep(val []byte, index int, nextField *fieldMatcher) *smallTable {
	var nextStep *faNext

	// have to stop one short to skip the closing "
	var nextState *faState

	if index == len(val)-2 {
		nextState = &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{nextField}}
	} else {
		nextState = &faState{table: onePrefixStep(val, index+1, nextField)}
	}
	nextStep = &faNext{steps: []*faState{nextState}}
	return makeSmallTable(nil, []byte{val[index]}, []*faNext{nextStep})
}

// makeStringFA creates a utf8-based automaton from a literal string
// using smallTables. Note the addition of a valueTerminator. The implementation
// is recursive because this allows the use of the makeSmallTable call, which
// reduces memory churn. Converting from a straightforward implementation to
// this approximately doubled the fields/second rate in addPattern
func makeStringFA(val []byte, useThisTransition *fieldMatcher) (*smallTable, *fieldMatcher) {
	var nextField *fieldMatcher
	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}

	return makeOneFAStep(val, 0, nextField), nextField
}

func makeOneFAStep(val []byte, index int, nextField *fieldMatcher) *smallTable {
	var nextStepList *faNext
	if index == len(val)-1 {
		lastStep := &faState{
			table:            newSmallTable(),
			fieldTransitions: []*fieldMatcher{nextField},
		}
		lastStepList := &faNext{steps: []*faState{lastStep}}
		nextStep := &faState{
			table: makeSmallTable(nil, []byte{valueTerminator}, []*faNext{lastStepList}),
		}
		nextStepList = &faNext{steps: []*faState{nextStep}}
	} else {
		nextStep := &faState{table: makeOneFAStep(val, index+1, nextField)}
		nextStepList = &faNext{steps: []*faState{nextStep}}
	}
	var u unpackedTable
	u[val[index]] = nextStepList
	// return stepper.buildTable(&u)
	return makeSmallTable(nil, []byte{val[index]}, []*faNext{nextStepList})
}
