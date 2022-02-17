package quamina

import (
	"bytes"
)

//  valueMatcher represents a byte-driven automaton.  The simplest implementation would be
//   for each step to be driven by the equivalent of a map[byte]nextState.  This is done with the startTable field.
//   In some cases there is only one byte sequence forward from a state, in which case that would be provided in
//   the singletonMatch field; if it matches, the singletonTransition is the return value. This is to avoid
//   having a long of smallTables each with only one entry
type valueMatcher struct {
	startTable          *smallTable
	singletonMatch      []byte
	singletonTransition *fieldMatcher
	existsTransitions   []*fieldMatcher
}

type smallStep interface {
	SmallTable() *smallTable
	SmallTransition() *smallTransition
}

type smallTransition struct {
	smallTable      *smallTable
	fieldTransition *fieldMatcher
}

// SmallTable and SmallTransition implement smallStep
func (t *smallTransition) SmallTable() *smallTable {
	return t.smallTable
}
func (t *smallTransition) SmallTransition() *smallTransition {
	return t
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
		// we always initialize the smallTable, even in a smallTransition step, so no need to check for nil
		table = step.SmallTable()
	}

	// we only do a field-level transition if there's one in the table that the last character in val arrives at
	nextTrans := step.SmallTransition()
	if nextTrans != nil {
		transitions = append(transitions, nextTrans.fieldTransition)
	}

	return transitions
}

func (m *valueMatcher) addTransition(val typedVal) *fieldMatcher {
	valBytes := []byte(val.val)

	if val.vType == existsTrueType || val.vType == existsFalseType {
		next := newFieldMatcher()
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
		m.singletonTransition = newFieldMatcher()
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

// addSteps, one for each byte in val.
//  Most steps are just from one smallTable to the next, an occasional one needing to be decorated with a field
//  transition, so it can be a smallTransition. The smallStep interface allows each kind.
func (m *valueMatcher) addSteps(val []byte, useThisTransition *fieldMatcher) *fieldMatcher {

	table := m.startTable

	// loop through all but the last character in val
	for i := 0; i < len(val)-1; i++ {
		utf8Byte := val[i]
		step := table.step(utf8Byte)

		if step == nil {
			// nothing here, just drop in a new smallTable
			newTable := newSmallTable()
			table.addRange([]byte{utf8Byte}, newTable)
			table = newTable
		} else {
			table = step.SmallTable()
		}
	}

	// is there an existing step from the last character?
	lastCh := val[len(val)-1]
	step := table.step(lastCh)

	// if no, build a new smallTransition with an empty smallTable and the right fieldMatcher
	// Note that the useThisTransition logic only applies here because it's only used on the very
	//  first entry in a new valueMatcher's smallTable
	if step == nil {
		var newTrans *fieldMatcher
		if useThisTransition != nil {
			newTrans = useThisTransition
		} else {
			newTrans = newFieldMatcher()
		}

		newSmallTrans := &smallTransition{
			smallTable:      newSmallTable(),
			fieldTransition: newTrans,
		}
		table.addRange([]byte{lastCh}, newSmallTrans)
		return newTrans
	}

	// there is a step forward. If it's already a smallTransition, we just return the existing field transition that's there
	smallTrans, ok := step.(*smallTransition)
	if ok {
		return smallTrans.fieldTransition
	}

	// the step is just a smallTable - we need to turn it into a smallTransition
	newTrans := &smallTransition{
		smallTable:      step.(*smallTable),
		fieldTransition: newFieldMatcher(),
	}
	table.addRange([]byte{lastCh}, newTrans)
	return newTrans.fieldTransition
}
