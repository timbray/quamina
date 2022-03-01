package quamina

import (
	"bytes"
)

// valueMatcher represents a byte-driven automaton.  The simplest implementation would be
//  for each step to be driven by the equivalent of a map[byte]nextState.  This is done with the startTable field.
//  In some cases there is only one byte sequence forward from a state, in which case that would be provided in
//  the singletonMatch field; if it matches, the singletonTransition is the return value. This is to avoid
//  having a long of smallTables each with only one entry
// Extra work is being done here with the goal of not wasting memory.  Since Quamina is happy to match basically
//  any number of patterns, users might reasonably do crazy-sounding things like add a million patterns to a
//  matcher, then complain about how much memory this takes.
type valueMatcher struct {
	startTable          *smallTable
	singletonMatch      []byte
	singletonTransition *fieldMatcher
	existsTransitions   []*fieldMatcher
}

type smallStep interface {
	SmallTable() *smallTable
	SmallTransition() *smallTransition
	HasTransition() bool
}

// ValueTerminator - whenever we're trying to match a value, we virtually add one of these as the last character, both
//  when building the automaton and matching a value.  This simplifies things because you can write a pattern to
//  match for example "foo" by writing it as
const ValueTerminator byte = 0xf5

type smallTransition struct {
	smallTable    *smallTable
	fieldMatchers []*fieldMatcher
}

func newSmallTransition(matcher *fieldMatcher) *smallTransition {
	return &smallTransition{
		smallTable:    newSmallTable(),
		fieldMatchers: []*fieldMatcher{matcher},
	}
}

func newSmallMultiTransition(matchers []*fieldMatcher) *smallTransition {
	return &smallTransition{
		smallTable:    newSmallTable(),
		fieldMatchers: matchers,
	}
}

// SmallTable and SmallTransition implement smallStep
func (t *smallTransition) SmallTable() *smallTable {
	return t.smallTable
}
func (t *smallTransition) SmallTransition() *smallTransition {
	return t
}
func (t *smallTransition) HasTransition() bool {
	return true
}

func newValueMatcher() *valueMatcher {
	return &valueMatcher{}
}

func (m *valueMatcher) transitionOn(val []byte) []*fieldMatcher {
	var transitions []*fieldMatcher
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
	var step smallStep
	for _, utf8Byte := range val {
		step = table.step(utf8Byte)
		if step == nil {
			return transitions
		}

		if step.HasTransition() {
			transitions = append(transitions, step.SmallTransition().fieldMatchers...)
		}

		// we always initialize the smallTable, even in a smallTransition step, so no need to check for nil
		table = step.SmallTable()
	}

	// look for terminator
	lastStep := table.step(ValueTerminator)

	// we only do a field-level transition if there's one in the table that the last character in val arrives at
	if lastStep != nil && lastStep.HasTransition() {
		transitions = append(transitions, lastStep.SmallTransition().fieldMatchers...)
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
	if m.startTable != nil {
		var nextField *fieldMatcher

		var newAutomaton smallStep
		if val.vType == shellStyleType {
			newAutomaton, nextField = makeShellStyleAutomaton(valBytes, nil)
		} else {
			newAutomaton, nextField = makeStringAutomaton(valBytes, nil)
		}
		m.startTable = mergeAutomata(m.startTable, newAutomaton)
		return nextField
	}

	// no start table, we have to work with singletons …

	// … unless this is completely virgin, in which case put in the singleton, assuming it's just a string match
	if m.singletonMatch == nil {
		if val.vType == shellStyleType {
			newAutomaton, nextField := makeShellStyleAutomaton(valBytes, nil)
			m.startTable = newAutomaton
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
	var singletonAutomaton, newAutomaton *smallTable
	singletonAutomaton, _ = makeStringAutomaton(m.singletonMatch, m.singletonTransition)

	var nextField *fieldMatcher
	if val.vType == shellStyleType {
		newAutomaton, nextField = makeShellStyleAutomaton(valBytes, nil)
	} else {
		newAutomaton, nextField = makeStringAutomaton(valBytes, nil)
	}
	m.startTable = mergeAutomata(singletonAutomaton, newAutomaton)

	// now table is ready for use, nuke singleton to signal threads to use it
	m.singletonMatch = nil
	m.singletonTransition = nil
	return nextField
}

// makeStringAutomaton is the simplest-case way to create a utf8-based automaton based on smallTables. Note
//  the addition of a ValueTerminator
func makeStringAutomaton(val []byte, useThisTransition *fieldMatcher) (start *smallTable, nextField *fieldMatcher) {
	table := newSmallTable()
	start = table

	for _, ch := range val {
		next := newSmallTable()
		table.addByteStep(ch, next)
		table = next
	}

	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}

	table.addByteStep(ValueTerminator, newSmallTransition(nextField))
	return
}
