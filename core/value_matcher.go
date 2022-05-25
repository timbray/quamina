package core

import (
	"bytes"
	"sync/atomic"
)

// valueMatcher represents a byte-driven automaton.  The table needs to be the equivalent of
//  a map[byte]nextState and is represented by smallTable. Some patterns can be represented by a deterministic
//  finite automaton (DFA) but others, particularly with a regex failure, need to be represented by a
//  nondeterministic finite automaton (NFA). NFAs trump DFAs so if a valueMatcher has one, it must be
//  used in preference to other alternatives.
// In some cases there is only one byte sequence forward from a state, i.e. a string-valued field with
//  only one string match. In this case, the DFA and NFA will b null and the value being matched has
//  to exactly equal the singletonMatch field; if so, the singletonTransition is the return value. This is
//  to avoid having a long chain of smallTables each with only one entry.
type valueMatcher struct {
	updateable atomic.Value
}
type vmFields struct {
	startDfa            *smallTable[*dfaStep]
	startNfa            *smallTable[*nfaStepList]
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
	vm.update(&vmFields{startNfa: nil})
	return &vm
}

func (m *valueMatcher) transitionOn(val []byte) []*fieldMatcher {
	var transitions []*fieldMatcher

	// exists transitions are basically a * on the value, so if we got the matcher, add 'em to the output
	fields := m.getFields()
	transitions = append(transitions, fields.existsTransitions...)

	switch {
	case fields.startNfa != nil:
		// if an NFA is provided, just use it
		return m.transitionNfa(val, transitions)

	case fields.singletonMatch != nil:
		// if there's a singleton entry here, we either match the val or we're done
		// Note: We have to check this first because addTransition might be busy
		//  constructing an automaton, but it's not ready for use yet.  When it's done
		//  it'll zero out the singletonMatch
		if bytes.Equal(fields.singletonMatch, val) {
			transitions = append(transitions, fields.singletonTransition)
		}
		return transitions

	case fields.startDfa != nil:
		return m.transitionDfa(val, transitions)

	default:
		return transitions
	}
}

// transitionNfa traverses a nondeterministic automaton - unlike a dfa, an input byte can transition
//  to multiple other nfa steps.  We could do like the top-level fieldMatcher does and add the
//  candidate next steps to a list, and then keep operating as long as there's something on the list,
//  but this is way deep into the lowest level and we'd like to avoid doing a lot of appending and
//  chopping on a slice, profiler says we're already spending almost all our time in GC and malloc.
//  So instead, we'll recurse like hell and and just follow all the links in order as we come to them,
//  on the theory that stack hammering is cheaper than slice bashing.
func (m *valueMatcher) transitionNfa(val []byte, transitions []*fieldMatcher) []*fieldMatcher {
	return oneNfaStep(m.getFields().startNfa, 0, val, transitions)
}

func oneNfaStep(table *smallTable[*nfaStepList], index int, val []byte, transitions []*fieldMatcher) []*fieldMatcher {
	var utf8Byte byte

	// fake ValueTerminator at the end of every val
	switch {
	case index == len(val):
		utf8Byte = ValueTerminator
	case index < len(val):
		utf8Byte = val[index]
	default:
		return transitions
	}
	nextSteps := table.step(utf8Byte)
	if nextSteps == nil {
		return transitions
	}
	index++
	for _, nextStep := range nextSteps.steps {
		transitions = append(transitions, nextStep.fieldTransitions...)
		transitions = oneNfaStep(nextStep.table, index, val, transitions)
	}
	return transitions
}

func (m *valueMatcher) transitionDfa(val []byte, transitions []*fieldMatcher) []*fieldMatcher {

	// step through the smallTables, byte by byte
	table := m.getFields().startDfa
	for _, utf8Byte := range val {
		step := table.step(utf8Byte)
		if step == nil {
			return transitions
		}

		transitions = append(transitions, step.fieldTransitions...)

		table = step.table
	}

	// look for terminator after exhausting bytes of val
	lastStep := table.step(ValueTerminator)

	// we only do a field-level transition if there's one in the table that the last character in val arrives at
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
	if fields.startDfa != nil || fields.startNfa != nil {

		if val.vType == shellStyleType {
			newNfa, nextField := makeShellStyleAutomaton(valBytes, nil)
			if fields.startNfa != nil {
				fields.startNfa = mergeNfas(newNfa, fields.startNfa)
			} else {
				fields.startNfa = mergeNfas(newNfa, dfa2Nfa(fields.startDfa))
				fields.startDfa = nil
			}
			m.update(fields)
			return nextField
		} else {
			newDfa, nextField := makeStringAutomaton(valBytes, nil)
			if fields.startNfa != nil {
				fields.startNfa = mergeNfas(fields.startNfa, dfa2Nfa(newDfa))
			} else {
				fields.startDfa = mergeDfas(fields.startDfa, newDfa)
			}
			m.update(fields)
			return nextField
		}
	}

	// no start table, we have to work with singletons …

	// … unless this is completely virgin, in which case put in the singleton, assuming it's just a string match
	if fields.singletonMatch == nil {
		if val.vType == shellStyleType {
			newAutomaton, nextField := makeShellStyleAutomaton(valBytes, nil)
			fields.startNfa = newAutomaton
			m.update(fields)
			return nextField
		} else {
			// at the moment this works for everything that's not a shellStyle, but this may not always be true in future
			fields.singletonMatch = valBytes
			fields.singletonTransition = newFieldMatcher()
			m.update(fields)
			return fields.singletonTransition
		}
	}

	// singleton match is here and this value matches it
	if (val.vType != shellStyleType) && bytes.Equal(fields.singletonMatch, valBytes) {
		return fields.singletonTransition
	}

	// singleton is here, we don't match, so our outdegree becomes 2, so we have to build an automaton with
	//  two values in it
	singletonAutomaton, _ := makeStringAutomaton(fields.singletonMatch, fields.singletonTransition)
	var nextField *fieldMatcher
	if val.vType == shellStyleType {
		var newNfa *smallTable[*nfaStepList]
		newNfa, nextField = makeShellStyleAutomaton(valBytes, nil)
		fields.startNfa = mergeNfas(newNfa, dfa2Nfa(singletonAutomaton))
	} else {
		var newDfa *smallTable[*dfaStep]
		newDfa, nextField = makeStringAutomaton(valBytes, nil)
		fields.startDfa = mergeDfas(singletonAutomaton, newDfa)
	}

	// now table is ready for use, nuke singleton to signal threads to use it
	fields.singletonMatch = nil
	fields.singletonTransition = nil
	m.update(fields)
	return nextField
}

// makeStringAutomaton creates a utf8-based automaton from a literal string using smallTables. Note
//  the addition of a ValueTerminator. The implementation is recursive because this allows the use of the
//  makeSmallDfaTable call, which reduces memory churn. Converting from a straightforward implementation to this
//  approximately doubled the fields/second rate in addPattern
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
		nextStep = &dfaStep{table: makeSmallDfaTable(nil, []byte{ValueTerminator}, []*dfaStep{lastStep})}
	} else {
		nextStep = &dfaStep{table: oneDfaStep(val, index+1, nextField)}
	}
	return makeSmallDfaTable(nil, []byte{val[index]}, []*dfaStep{nextStep})
}
