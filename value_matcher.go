package quamina

import (
	"bytes"
	"fmt"
	"sync/atomic"
)

type bufpair struct {
	buf1, buf2 []*faState
}

// valueMatcher represents a byte-driven finite automaton (FA).  The table needs to be the
// equivalent of a map[byte][]nextState and is represented by smallTable.
// In this implementation all the FAs are nondeterministic, which means each
// byte can cause transfers to multiple other states. We compute the FA
// for a pattern and merge with any existing FA.
// In some (common) cases there is only one matching value present for some field,
// e.g. a string-valued field with only one string match. In this case, the FA
// will be null and the value being matched has to exactly equal the singletonMatch
// field; if so, the singletonTransition is the return value. This is to avoid
// having a long chain of smallTables each with only one entry.
// To allow for concurrent access between one thread running AddPattern and many
// others running MatchesForEvent, the valueMatcher payload is stored in an
// atomic.Pointer
type valueMatcher struct {
	updateable atomic.Pointer[vmFields]
}
type vmFields struct {
	startTable          *smallTable
	singletonMatch      []byte
	singletonTransition *fieldMatcher
	hasNumbers          bool
	isNondeterministic  bool
}

func (m *valueMatcher) fields() *vmFields {
	return m.updateable.Load()
}

func (m *valueMatcher) getFieldsForUpdate() *vmFields {
	current := m.updateable.Load()
	freshState := *current // struct copy
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

func (m *valueMatcher) transitionOn(eventField *Field, bufs *bufpair) []*fieldMatcher {
	vmFields := m.fields()
	var transitions []*fieldMatcher

	val := eventField.Val
	switch {
	case vmFields.singletonMatch != nil:
		// if there's a singleton entry here, we either match the val or we're
		// done Note: We have to check this first because addTransition might be
		// busy constructing an automaton, but it's not ready for use yet.  When
		// it's done it'll zero out the singletonMatch
		if bytes.Equal(vmFields.singletonMatch, val) {
			transitions = append(transitions, vmFields.singletonTransition)
		}
		return transitions

	case vmFields.startTable != nil:
		// if there is a potential for a numeric match, try making a Q number from the event
		if vmFields.hasNumbers && eventField.IsNumber {
			qNum, err := qNumFromBytes(val)
			if err == nil {
				if vmFields.isNondeterministic {
					return traverseNFA(vmFields.startTable, qNum, transitions, bufs)
				} else {
					return traverseDFA(vmFields.startTable, qNum, transitions)
				}
			}
		}

		// if it doesn't work as a Q number for some reason, go ahead and compare the string values
		if vmFields.isNondeterministic {
			return traverseNFA(vmFields.startTable, val, transitions, bufs)
		} else {
			return traverseDFA(vmFields.startTable, val, transitions)
		}

	default:
		// no FA, no singleton, nothing to do, this probably can't happen because a flattener
		// shouldn't preserve a field that hasn't appeared in a pattern
		return transitions
	}
}

func (m *valueMatcher) addTransition(val typedVal, printer printer) *fieldMatcher {
	valBytes := []byte(val.val)
	fields := m.getFieldsForUpdate()

	// special case - virgin state and this is a string match
	if fields.startTable == nil && fields.singletonMatch == nil && (val.vType == stringType || val.vType == literalType) {
		fields.singletonMatch = valBytes
		fields.singletonTransition = newFieldMatcher()
		m.update(fields)
		return fields.singletonTransition
	}

	// special case: singleton match is here and this value matches it
	if val.vType == stringType || val.vType == literalType {
		if bytes.Equal(fields.singletonMatch, valBytes) {
			return fields.singletonTransition
		}
	}

	// no dodges, we have to build an automaton to match this value
	var nextField *fieldMatcher
	var newFA *smallTable
	switch val.vType {
	case stringType, literalType:
		newFA, nextField = makeStringFA(valBytes, nil, false)
	case numberType:
		newFA, nextField = makeStringFA(valBytes, nil, true)
		fields.hasNumbers = true
	case anythingButType:
		newFA, nextField = makeMultiAnythingButFA(val.list)
	case shellStyleType:
		newFA, nextField = makeShellStyleFA(valBytes, printer)
		fields.isNondeterministic = true
	case wildcardType:
		newFA, nextField = makeWildcardFA(valBytes, printer)
		fields.isNondeterministic = true
	case prefixType:
		newFA, nextField = makePrefixFA(valBytes)
	case monocaseType:
		newFA, nextField = makeMonocaseFA(valBytes, printer)
	case regexpType:
		newFA, nextField = makeRegexpNFA(val.parsedRegexp, true)
		printer.labelTable(newFA, "RX start")
	default:
		panic("unknown value type")
	}

	// there's already a table, thus an out-degree > 1
	if fields.startTable != nil {
		fields.startTable = mergeFAs(fields.startTable, newFA, sharedNullPrinter)
		m.update(fields)
		return nextField
	}

	// no start table, maybe singletons …
	if fields.singletonMatch != nil {
		// singleton is here, we don't match, so our outdegree becomes 2, so we have
		// to build an automaton with two values in it
		singletonAutomaton, _ := makeStringFA(fields.singletonMatch, fields.singletonTransition, false)

		// now table is ready for use, nuke singleton to signal threads to use it
		fields.startTable = mergeFAs(singletonAutomaton, newFA, sharedNullPrinter)
		fields.singletonMatch = nil
		fields.singletonTransition = nil
	} else {
		// empty valueMatcher, no special cases, just jam in the new FA
		fields.startTable = newFA
	}
	m.update(fields)
	return nextField
}

func (m *valueMatcher) gatherMetadata(meta *nfaMetadata) {
	start := m.fields().startTable
	if start != nil {
		start.gatherMetadata(meta)
	}
}

// TODO: make these simple FA builders iterative not recursive, this will recurse as deep as the longest string match

func makePrefixFA(val []byte) (*smallTable, *fieldMatcher) {
	nextField := newFieldMatcher()
	return makeOnePrefixFAStep(val, 0, nextField), nextField
}

func makeOnePrefixFAStep(val []byte, index int, nextField *fieldMatcher) *smallTable {
	// have to stop one short to skip the closing "
	var nextState *faState
	if index == len(val)-2 {
		nextState = &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{nextField}}
	} else {
		nextState = &faState{table: makeOnePrefixFAStep(val, index+1, nextField)}
	}
	return makeSmallTable(nil, []byte{val[index]}, []*faState{nextState})
}

// makeStringFA creates a utf8-based automaton from a literal string
// using smallTables. Note the addition of a valueTerminator. The implementation
// is recursive because this allows the use of the makeSmallTable call, which
// reduces memory churn. Converting from a straightforward implementation to
// this approximately doubled the fields/second rate in addPattern
func makeStringFA(val []byte, useThisTransition *fieldMatcher, isNumber bool) (*smallTable, *fieldMatcher) {
	var nextField *fieldMatcher
	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}

	stringFA := makeOneStringFAStep(val, 0, nextField)

	// if the field is numeric, *and* if it can be converted to a float, *and* can be
	// made into a Q number, equip the NFA with the Q number form
	if isNumber {
		qNum, err := qNumFromBytes(val)
		if err == nil {
			numberFA := makeOneStringFAStep(qNum, 0, nextField)
			stringFA = mergeFAs(stringFA, numberFA, sharedNullPrinter)
		}
	}
	return stringFA, nextField
}

// makeFAFragment makes the simplest possible byte-chain FA with its last transition being to the provided
// endAt value. It is designed to help higher-level automaton builders.
// suppose you need a few steps to match "cat". You call makeFAFragment and it'll make two *faState instances, one
// which matches 'a' and transitions to the second, which matches 't' and transitions to the provided endAt
// argument. Then you transition to what makeFAFragment returns on 'c' from your current faState.
func makeFAFragment(val []byte, endAt *faState, pp printer) *faState {
	firstStep := &faState{}
	step := firstStep
	// no-op on one-byte values, but should still work so caller can just call this without worrying
	// about slice length
	if len(val) == 1 {
		return endAt
	}
	for index := 1; index < len(val); index++ {
		if index == len(val)-1 {
			table := makeSmallTable(nil, []byte{val[index]}, []*faState{endAt})
			pp.labelTable(table, fmt.Sprintf("exiting on %v", val[index]))
			step.table = table
		} else {
			nextState := &faState{}
			table := makeSmallTable(nil, []byte{val[index]}, []*faState{nextState})
			pp.labelTable(table, fmt.Sprintf("stepping on %c", val[index]))
			step.table = table
			step = nextState
		}
	}
	return firstStep
}

func makeOneStringFAStep(val []byte, index int, nextField *fieldMatcher) *smallTable {
	// TODO: Turn this into a simple back-to-front construction and remove the recursion
	var nextStep *faState
	if index == len(val)-1 {
		lastStep := &faState{
			table:            newSmallTable(),
			fieldTransitions: []*fieldMatcher{nextField},
		}
		nextStep = &faState{
			table: makeSmallTable(nil, []byte{valueTerminator}, []*faState{lastStep}),
		}
	} else {
		nextStep = &faState{table: makeOneStringFAStep(val, index+1, nextField)}
	}
	return makeSmallTable(nil, []byte{val[index]}, []*faState{nextStep})
}
