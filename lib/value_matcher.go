package quamina

import (
	"bytes"
)

//  SmallValueMatcher represents a byte-driven automaton.  The simplest implementation would be
//   for each step to be driven by the equivalent of a map[byte]nextState.  This is done with the startTable field.
//   In some cases there is only one byte sequence forward from a state, in which case that would be provided in
//   the singletonMatch field; if it matches, the singletonTransition is the return value. This is to avoid
//   having a long of smallTables each with only one entry
type valueMatcher struct {
	startTable          *smallTable
	singletonMatch      []byte
	singletonTransition *fieldMatchState
	existsTransitions   []*fieldMatchState
}

func newValueMatcher() *valueMatcher {
	return &valueMatcher{}
}

func (m *valueMatcher) transitionOn(val []byte) []*fieldMatchState {
	var transitions []*fieldMatchState
	transitions = append(transitions, m.existsTransitions...)

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
	if m.startTable == nil {
		return transitions
	}

	// step through the smallTables, byte by byte
	table := m.startTable
	for _, utf8Byte := range val {
		table = table.step(utf8Byte)
		if table == nil {
			return transitions
		}
	}

	// we only do a field-level transition if there's one in the table that the last character in val arrives at
	if table.transition != nil {
		transitions = append(transitions, table.transition)
	}

	return transitions
}

func (m *valueMatcher) addTransition(val typedVal) *fieldMatchState {
	valBytes := []byte(val.val)

	if val.vType == existsTrueType || val.vType == existsFalseType {
		next := newFieldMatchState()
		m.existsTransitions = append(m.existsTransitions, next)
		return next
	}

	// there's already a table, thus an out-degree > 1
	if m.startTable != nil {
		return m.addSteps(valBytes, nil)
	}

	// no start table, we have to work with singletons …

	// … unless this is completely virgin, in which case put in the singleton
	if m.singletonMatch == nil {
		m.singletonMatch = valBytes
		m.singletonTransition = newFieldMatchState()
		return m.singletonTransition
	}

	// singleton match is here and this value matches it
	if bytes.Equal(m.singletonMatch, valBytes) {
		return m.singletonTransition
	}

	// singleton is here, we don't match, so our outdegree becomes 2, so we have to build two smallTable chains
	m.startTable = newSmallTable()
	_ = m.addSteps(m.singletonMatch, m.singletonTransition) // be careful to re-use singleton transition

	// now table is ready for use, nuke singleton to signal threads to use it
	m.singletonMatch = nil
	m.singletonTransition = nil
	return m.addSteps(valBytes, nil)
}

func (m *valueMatcher) addSteps(val []byte, useThisTransition *fieldMatchState) *fieldMatchState {
	table := m.startTable
	for _, utf8Byte := range val {
		next := table.step(utf8Byte)
		if next == nil {
			next = newSmallTable()
			table.addRange([]byte{utf8Byte}, next)
		}
		table = next
	}
	if useThisTransition != nil {
		table.transition = useThisTransition
	} else if table.transition == nil {
		table.transition = newFieldMatchState()
	}
	return table.transition
}
