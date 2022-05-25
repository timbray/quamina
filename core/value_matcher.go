package core

import (
	"bytes"
)

// valueMatcher represents a byte-driven automaton.  The simplest implementation would be
//  for each step to be driven by the equivalent of a map[byte]nextState.  This is done with the startStep field.
//  In some cases there is only one byte sequence forward from a state, in which case that would be provided in
//  the singletonMatch field; if it matches, the singletonTransition is the return value. This is to avoid
//  having a long of smallTables each with only one entry. The isNFA field is true if the startStep points to
//  a smallDfaTable in which the values are nfaSteps, i.e. slices of smallSteps.
// Extra work is being done here with the goal of not wasting memory.  Since Quamina is happy to match basically
//  any number of patterns, users might reasonably do crazy-sounding things like add a million patterns to a
//  matcher, then complain about how much memory this takes.
type valueMatcher struct {
	startDfa            *smallTable[DS]
	startNfa            *smallTable[NSL]
	singletonMatch      []byte
	singletonTransition *fieldMatcher
	existsTransitions   []*fieldMatcher
}

func newValueMatcher() *valueMatcher {
	return &valueMatcher{startNfa: nil}
}

func (m *valueMatcher) transitionOn(val []byte) []*fieldMatcher {
	var transitions []*fieldMatcher
	transitions = append(transitions, m.existsTransitions...)
	if m.startNfa != nil {
		return m.transitionNfa(val, transitions)
	}

	// if there's a singleton entry here, we either match the val or we're done
	// Note: We have to check this first because addTransition might be busy
	//  constructing the table, but it's not ready for use yet.  When it's done
	//  it'll zero out the singletonMatch
	if m.singletonMatch != nil {
		if bytes.Equal(m.singletonMatch, val) {
			transitions = append(transitions, m.singletonTransition)
		}
		return transitions
	}

	// there's no singleton. If there's also no table, there's nowhere to go
	if m.startDfa == nil {
		return transitions
	}

	return m.transitionDfa(val, transitions)
}

// transitionNfa traverses a nondeterministic automaton - unlike a dfa, an input byte can transition
//  to multiple other nfa steps.  We could do like the top-level fieldMatcher does and add the
//  candidate next steps to a list, and then keep operating as long as there's something on the list,
//  but this is way deep into the lowest level and we'd like to avoid doing a lot of appending and
//  chopping on a slice, profiler says we're already spending almost all our time in GC and malloc.
//  So instead, we'll recurse like hell and and just follow all the links in order as we come to them,
//  on the theory that stack hammering is cheaper than slice bashing.
func (m *valueMatcher) transitionNfa(val []byte, transitions []*fieldMatcher) []*fieldMatcher {
	return oneNfaStep(m.startNfa, 0, val, transitions)
}

func oneNfaStep(table *smallTable[NSL], index int, val []byte, transitions []*fieldMatcher) []*fieldMatcher {
	var utf8Byte byte
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
	table := m.startDfa
	for _, utf8Byte := range val {
		step := table.step(utf8Byte)
		if step == nil {
			return transitions
		}

		transitions = append(transitions, step.fieldTransitions...)

		// we always initialize the smallDfaTable, even in a dfaTransition step, so no need to check for nil
		table = step.table
	}

	// look for terminator
	lastStep := table.step(ValueTerminator)

	// we only do a field-level transition if there's one in the table that the last character in val arrives at
	if lastStep != nil && lastStep.fieldTransitions != nil {
		transitions = append(transitions, lastStep.fieldTransitions...)
	}

	return transitions
}

func (m *valueMatcher) addTransition(val typedVal) *fieldMatcher {
	valBytes := []byte(val.val)

	// TODO: Shouldn't these all point to the same fieldMatcher?
	if val.vType == existsTrueType || val.vType == existsFalseType {
		next := newFieldMatcher()
		m.existsTransitions = append(m.existsTransitions, next)
		return next
	}

	// there's already a table, thus an out-degree > 1
	if m.startDfa != nil || m.startNfa != nil {

		if val.vType == shellStyleType {
			newNfa, nextField := makeShellStyleAutomaton(valBytes, nil)
			if m.startNfa != nil {
				m.startNfa = mergeNfas(newNfa, m.startNfa)
			} else {
				m.startNfa = mergeNfas(newNfa, dfa2Nfa(m.startDfa))
			}
			return nextField
		} else {
			newDfa, nextField := makeStringAutomaton(valBytes, nil)
			if m.startNfa != nil {
				m.startNfa = mergeNfas(m.startNfa, dfa2Nfa(newDfa))
			} else {
				m.startDfa = mergeDfas(m.startDfa, newDfa)
			}
			return nextField
		}
	}

	// no start table, we have to work with singletons …

	// … unless this is completely virgin, in which case put in the singleton, assuming it's just a string match
	if m.singletonMatch == nil {
		if val.vType == shellStyleType {
			newAutomaton, nextField := makeShellStyleAutomaton(valBytes, nil)
			m.startNfa = newAutomaton
			return nextField
		} else {
			// at the moment this works for everything that's not a shellStyle, but this may not always be true in future
			m.singletonMatch = valBytes
			m.singletonTransition = newFieldMatcher()
			return m.singletonTransition
		}
	}

	// singleton match is here and this value matches it
	if (val.vType != shellStyleType) && bytes.Equal(m.singletonMatch, valBytes) {
		return m.singletonTransition
	}

	// singleton is here, we don't match, so our outdegree becomes 2, so we have to build an automaton with
	//  two values in it
	var singletonAutomaton *smallTable[DS]
	singletonAutomaton, _ = makeStringAutomaton(m.singletonMatch, m.singletonTransition)
	var nextField *fieldMatcher
	if val.vType == shellStyleType {
		var newNfa *smallTable[NSL]
		newNfa, nextField = makeShellStyleAutomaton(valBytes, nil)
		m.startNfa = mergeNfas(newNfa, dfa2Nfa(singletonAutomaton))
	} else {
		var newDfa *smallTable[DS]
		newDfa, nextField = makeStringAutomaton(valBytes, nil)
		m.startDfa = mergeDfas(singletonAutomaton, newDfa)
	}

	// now table is ready for use, nuke singleton to signal threads to use it
	m.singletonMatch = nil
	m.singletonTransition = nil
	return nextField
}

// makeStringAutomaton creates a utf8-based automaton from a literal string using smallTables. Note
//  the addition of a ValueTerminator. The implementation is recursive because this allows the use of the
//  makeSmallDfaTable call, which reduces memory churn. Converting from a straightforward implementation to this
//  approximately doubled the fields/second rate in addPattern
func makeStringAutomaton(val []byte, useThisTransition *fieldMatcher) (*smallTable[DS], *fieldMatcher) {
	var nextField *fieldMatcher
	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}
	return oneDfaStep(val, 0, nextField), nextField
}

func oneDfaStep(val []byte, index int, nextField *fieldMatcher) *smallTable[DS] {
	var nextStep DS
	if index == len(val)-1 {
		lastStep := &dfaStep{table: newSmallTable[DS](), fieldTransitions: []*fieldMatcher{nextField}}
		nextStep = &dfaStep{table: makeSmallDfaTable(nil, []byte{ValueTerminator}, []DS{lastStep})}
	} else {
		nextStep = &dfaStep{table: oneDfaStep(val, index+1, nextField)}
	}
	return makeSmallDfaTable(nil, []byte{val[index]}, []DS{nextStep})
}
