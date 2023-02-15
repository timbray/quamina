package quamina

import (
	"bytes"
	"sync/atomic"
)

// valueMatcher represents a byte-driven automaton.  The table needs to be the
// equivalent of a map[byte]nextState and is represented by smallTable. Some
// patterns can be represented by a deterministic finite automaton (DFA) but
// others, particularly with a regex flavor, need to be represented by a
// nondeterministic finite automaton (NFA).  NFAs are converted to DFAs for
// simplicity and efficiency. The basic algorithm is to compute the automaton
// for a pattern, convert it to a DFA if necessary, and merge with any
// existing DFA.
// In some (common) cases there is only one byte sequence forward from a state,
// i.e. a string-valued field with only one string match. In this case, the DFA
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
	startDfa            *smallTable[*dfaStep]
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

	case fields.startDfa != nil:
		return transitionDfa(fields.startDfa, val, transitions)

	default:
		// no dfa, no singleton, nothing to do, this probably can't happen because a flattener
		// shouldn't preserve a field that hasn't appeared in a pattern
		return transitions
	}
}

func transitionDfa(table *smallTable[*dfaStep], val []byte, transitions []*fieldMatcher) []*fieldMatcher {
	// step through the smallTables, byte by byte
	for _, utf8Byte := range val {
		step := table.step(utf8Byte)
		if step == nil {
			return transitions
		}

		transitions = append(transitions, step.fieldTransitions...)
		table = step.table
	}

	// look for terminator after exhausting bytes of val
	lastStep := table.step(valueTerminator)

	// we only do a field-level transition if there's one in the table that the
	// last character in val arrives at
	if lastStep != nil {
		transitions = append(transitions, lastStep.fieldTransitions...)
	}

	return transitions
}

func (m *valueMatcher) addTransition(val typedVal) *fieldMatcher {
	valBytes := []byte(val.val)
	fields := m.getFieldsForUpdate()

	// there's already a table, thus an out-degree > 1
	if fields.startDfa != nil {
		var newDfa *smallTable[*dfaStep]
		var nextField *fieldMatcher
		switch val.vType {
		case stringType, numberType, literalType:
			newDfa, nextField = makeStringAutomaton(valBytes, nil)
		case anythingButType:
			newDfa, nextField = makeMultiAnythingButAutomaton(val.list, nil)
		case shellStyleType:
			var newNfa *smallTable[*nfaStepList]
			newNfa, nextField = makeShellStyleAutomaton(valBytes, nil)
			newDfa = nfa2Dfa(newNfa)
		case prefixType:
			newDfa, nextField = makePrefixAutomaton(valBytes, nil)
		default:
			panic("unknown value type")
		}
		fields.startDfa = mergeDfas(fields.startDfa, newDfa)
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
			newAutomaton, nextField := makeMultiAnythingButAutomaton(val.list, nil)
			fields.startDfa = newAutomaton
			m.update(fields)
			return nextField
		case shellStyleType:
			newAutomaton, nextField := makeShellStyleAutomaton(valBytes, nil)
			fields.startDfa = nfa2Dfa(newAutomaton)
			m.update(fields)
			return nextField
		case prefixType:
			newAutomaton, nextField := makePrefixAutomaton(valBytes, nil)
			fields.startDfa = newAutomaton
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
	singletonAutomaton, _ := makeStringAutomaton(fields.singletonMatch, fields.singletonTransition)
	var nextField *fieldMatcher
	var newDfa *smallTable[*dfaStep]
	switch val.vType {
	case stringType, numberType, literalType:
		newDfa, nextField = makeStringAutomaton(valBytes, nil)
	case anythingButType:
		newDfa, nextField = makeMultiAnythingButAutomaton(val.list, nil)
	case shellStyleType:
		var newNfa *smallTable[*nfaStepList]
		newNfa, nextField = makeShellStyleAutomaton(valBytes, nil)
		newDfa = nfa2Dfa(newNfa)
	case prefixType:
		newDfa, nextField = makePrefixAutomaton(valBytes, nil)
	default:
		panic("unknown value type")
	}

	// now table is ready for use, nuke singleton to signal threads to use it
	fields.startDfa = mergeDfas(singletonAutomaton, newDfa)
	fields.singletonMatch = nil
	fields.singletonTransition = nil
	m.update(fields)
	return nextField
}

func makePrefixAutomaton(val []byte, useThisTransition *fieldMatcher) (*smallTable[*dfaStep], *fieldMatcher) {
	var nextField *fieldMatcher

	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}
	return onePrefixStep(val, 0, nextField), nextField
}

func onePrefixStep(val []byte, index int, nextField *fieldMatcher) *smallTable[*dfaStep] {
	var nextStep *dfaStep

	// have to stop one short to skip the closing "
	if index == len(val)-2 {
		nextStep = &dfaStep{table: newSmallTable[*dfaStep](), fieldTransitions: []*fieldMatcher{nextField}}
	} else {
		nextStep = &dfaStep{table: onePrefixStep(val, index+1, nextField)}
	}
	return makeSmallDfaTable(nil, []byte{val[index]}, []*dfaStep{nextStep})
}

// makeStringAutomaton creates a utf8-based automaton from a literal string
// using smallTables. Note the addition of a valueTerminator. The implementation
// is recursive because this allows the use of the makeSmallDfaTable call, which
// reduces memory churn. Converting from a straightforward implementation to
// this approximately doubled the fields/second rate in addPattern
func makeStringAutomaton(val []byte, useThisTransition *fieldMatcher) (*smallTable[*dfaStep], *fieldMatcher) {
	var nextField *fieldMatcher
	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}
	return oneDfaStep(val, 0, nextField), nextField
}

func oneDfaStep(val []byte, index int, nextField *fieldMatcher) *smallTable[*dfaStep] {
	var nextStep *dfaStep
	if index == len(val)-1 {
		lastStep := &dfaStep{table: newSmallTable[*dfaStep](), fieldTransitions: []*fieldMatcher{nextField}}
		nextStep = &dfaStep{table: makeSmallDfaTable(nil, []byte{valueTerminator}, []*dfaStep{lastStep})}
	} else {
		nextStep = &dfaStep{table: oneDfaStep(val, index+1, nextField)}
	}
	return makeSmallDfaTable(nil, []byte{val[index]}, []*dfaStep{nextStep})
}
