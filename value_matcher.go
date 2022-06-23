package quamina

import (
	"bytes"
	"sync/atomic"
)

// valueMatcher represents a byte-driven automaton.  The table needs to be the
// equivalent of a map[byte]nextState and is represented by smallTable. Some
// patterns can be represented by a deterministic finite automaton (DFA) but
// others, particularly with a regex failure, need to be represented by a
// nondeterministic finite automaton (NFA). NFAs trump DFAs so if a valueMatcher
// has one, it must be used in preference to other alternatives. In some cases
// there is only one byte sequence forward from a state, i.e. a string-valued
// field with only one string match. In this case, the DFA and NFA will b null
// and the value being matched has to exactly equal the singletonMatch field; if
// so, the singletonTransition is the return value. This is to avoid having a
// long chain of smallTables each with only one entry.
type valueMatcher struct {
	updateable atomic.Value
}
type vmFields struct {
	startDfa            *smallTable[*dfaStep]
	singletonMatch      []byte
	singletonTransition *fieldMatcher
	existsTransitions   []*fieldMatcher
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

	// exists transitions are basically a * on the value, so if we got the
	// matcher, add 'em to the output
	fields := m.getFields()
	transitions = append(transitions, fields.existsTransitions...)

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
		// no dfa, no singleton, nothing to do
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

	// TODO: Shouldn't these all point to the same fieldMatcher?
	if val.vType == existsTrueType || val.vType == existsFalseType {
		next := newFieldMatcher()
		fields.existsTransitions = append(fields.existsTransitions, next)
		m.update(fields)
		return next
	}

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
	default:
		panic("unknown val type")
	}

	// now table is ready for use, nuke singleton to signal threads to use it
	fields.startDfa = mergeDfas(singletonAutomaton, newDfa)
	fields.singletonMatch = nil
	fields.singletonTransition = nil
	m.update(fields)
	return nextField
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
