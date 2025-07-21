package quamina

import (
	"fmt"
)

// This groups the functions that traverse, merge, and debug Quamina's nondeterministic finite automata

// faState is used by the valueMatcher automaton - every step through the
// automaton requires a smallTable and for some of them, taking the step means you've matched a value and can
// transition to a new fieldMatcher, in which case the fieldTransitions slice will be non-nil
type faState struct {
	table            *smallTable
	fieldTransitions []*fieldMatcher
}

// transmap is a Set structure used to gather transitions as we work our way through the automaton
type transmap struct {
	set map[*fieldMatcher]bool
}

func newTransMap() *transmap {
	return &transmap{set: make(map[*fieldMatcher]bool)}
}

func (tm *transmap) add(fms []*fieldMatcher) {
	for _, fm := range fms {
		tm.set[fm] = true
	}
}

func (tm *transmap) all() []*fieldMatcher {
	var all []*fieldMatcher
	for fm := range tm.set {
		all = append(all, fm)
	}
	return all
}

// nfaBuffers contains the buffers that are used to traverse NFAs. Go doesn't have thread-local variables
// but Quamina does, because of the required quamina.Copy() function.  These will grow to accommodate
// the incoming event patterns and matcher structures and eventually the amount of event-matching memory
// allocation will be reduced to nearly zero.
type nfaBuffers struct {
	buf1, buf2 []*faState
	eClosure   *epsilonClosure
}

func newNfaBuffers() *nfaBuffers {
	return &nfaBuffers{
		buf1:     make([]*faState, 0, 16),
		buf2:     make([]*faState, 0, 16),
		eClosure: newEpsilonClosure(),
	}
}

func nfa2Dfa(nfaTable *smallTable) *faState {
	startNfa := []*faState{{table: nfaTable}}
	return n2dNode(startNfa, newStateLists())
}

// n2dNode input is a list of NFA states, which are all the states that are either the
// singleton start state or the states that can be reached from a previous state on
// a byte transition.
// It returns a DFA state (i.e. no epsilons) that corresponds to this aggregation of
// NFA states.
func n2dNode(rawNStates []*faState, sList *stateLists) *faState {
	// we expand the raw list of states by adding the epsilon closure of each
	nStates := make([]*faState, 0, len(rawNStates))
	for _, rawNState := range rawNStates {
		nStates = append(nStates, getEpsilonClosure(rawNState)...)
	}

	// the collection of states may have duplicates and, deduplicated, considered'
	// as a set, may be equal to a previous set of states, in which case the
	// corresponding DFA will have already been constructed.
	ingredients, dfaState, alreadyExists := sList.intern(nStates)
	if alreadyExists {
		return dfaState
	}

	// OK, this is a new set of states, so we have to consider all the possible byte
	// transitions and, for each, aggregate all the states that could be reached on seeing
	// that byte, then recurse to turn that aggregation into a DFA state

	// to simplify, let's unpack all the ingredients
	nUnpacked := make([]*unpackedTable, len(ingredients))
	for i, nState := range ingredients {
		nUnpacked[i] = unpackTable(nState.table)
	}

	// for each byte value
	for utf8byte := 0; utf8byte < byteCeiling; utf8byte++ {
		var rawStates []*faState

		// for each of the unique states
		for ingredient, unpackedNState := range nUnpacked {
			if unpackedNState[utf8byte] != nil {
				rawStates = append(rawStates, unpackedNState[utf8byte])
			}
			rawStates = append(rawStates, ingredients[ingredient].table.epsilons...)
		}
		if len(rawStates) > 0 {
			dfaState.table.addByteStep(byte(utf8byte), n2dNode(rawStates, sList))
		}
	}

	// load up transitions
	trans := newTransMap()
	for _, state := range ingredients {
		trans.add(state.fieldTransitions)
	}
	dfaState.fieldTransitions = trans.all()
	return dfaState
}

// While some Quamina patterns require the use of NFAs, many (most?) don't, and while we're still using a
// NFA-capable data structure, we can traverse it deterministically if we know in advance that every
// combination of an faState with a byte will transition to at most one other faState.

func traverseDFA(table *smallTable, val []byte, transitions []*fieldMatcher) []*fieldMatcher {
	for index := 0; index <= len(val); index++ {
		var utf8Byte byte
		if index < len(val) {
			utf8Byte = val[index]
		} else {
			utf8Byte = valueTerminator
		}
		next := table.dStep(utf8Byte)
		if next == nil {
			break
		}
		transitions = append(transitions, next.fieldTransitions...)
		table = next.table
	}
	return transitions
}

// traverseNFA attempts efficient traversal of an NFA. Each step processes currentList, a list of the
// automaton states currently active. For each element of the list, we compute its epsilon closure
// and apply the current input byte to each state in the resulting list. The results, if any, are
// collected in the nextStates list.  The bufs structure contains three buffers, one each for
// currentStates, nextStates, and the epsilon closure of one particular state. These are re-used
// and should grow with use and minimize the need for memory allocation.
func traverseNFA(table *smallTable, val []byte, transitions []*fieldMatcher, bufs *nfaBuffers, _ printer) []*fieldMatcher {
	currentStates := bufs.buf1
	currentStates = append(currentStates, &faState{table: table})
	nextStates := bufs.buf2

	// a lot of the transitions stuff is going to be empty, but on the other hand
	// a * entry with a transition could end up getting added a lot. While this
	// involves memory allocation, in the vast majority of cases matching an event
	// will turn up a tiny number of unique matches, so allocation should be minimal
	newTransitions := newTransMap()
	newTransitions.add(transitions)

	stepResult := &stepOut{}
	for index := 0; len(currentStates) != 0 && index <= len(val); index++ {
		var utf8Byte byte
		if index < len(val) {
			utf8Byte = val[index]
		} else {
			utf8Byte = valueTerminator
		}
		for _, state := range currentStates {
			closure := bufs.eClosure.getAndCacheClosure(state)
			for _, ecState := range closure {
				newTransitions.add(ecState.fieldTransitions)
				ecState.table.step(utf8Byte, stepResult)
				if stepResult.step != nil {
					nextStates = append(nextStates, stepResult.step)
				}
				// TODO: Figure out why this works
				// follow loopback epsilon transitions
				for _, state := range ecState.table.epsilons {
					if state == ecState {
						nextStates = append(nextStates, state)
					}
				}
			}
		}
		// re-use these
		swapStates := currentStates
		currentStates = nextStates
		nextStates = swapStates[:0]
	}

	// we've run out of input bytes so we need to check the current states and their
	// epsilon closures for matches
	for _, state := range currentStates {
		closure := bufs.eClosure.getAndCacheClosure(state)
		for _, ecState := range closure {
			newTransitions.add(ecState.fieldTransitions)
		}
	}

	bufs.buf1 = currentStates[:0]
	bufs.buf2 = nextStates[:0]
	return newTransitions.all()
}

type faStepKey struct {
	step1 *faState
	step2 *faState
}

// mergeFAs compute the union of two valueMatch automata.  If you look up the textbook theory about this,
// they say to compute the set product for automata A and B and build A0B0, A0B1 … A1BN, A1B0 … but if you look
// at that you realize that many of the product states aren't reachable. So you compute A0B0 and then keep
// recursing on the transitions coming out, I'm pretty sure you get a correct result. I don't know if it's
// minimal or even avoids being wasteful.
// INVARIANT: neither argument is nil
// INVARIANT: To be thread-safe, no existing table can be updated except when we're building it
func mergeFAs(table1, table2 *smallTable, pp printer) *smallTable {
	state1 := &faState{table: table1}
	state2 := &faState{table: table2}
	return mergeFAStates(state1, state2, make(map[faStepKey]*faState), false, pp).table
}

func mergeFAStates(state1, state2 *faState, keyMemo map[faStepKey]*faState, isSpinout bool, pp printer) *faState {
	// try to memo-ize
	mKey := faStepKey{state1, state2}
	combined, ok := keyMemo[mKey]
	if ok {
		return combined
	}
	combined = &faState{table: newSmallTable()}

	// If neither side has any epsilons, we proceed with a per-byte-value merge.
	// if both sides are spinouts, or if one side is a spinout and the other has no epsilons,
	// we can re-use the spinout.
	// If either side has epsilons but isn't a spinout, then per the Thompson procedure we just
	// make a splice, an empty state with two epsilons, one branching to each state
	// Now, about the spinouts. Suppose we get a*b then ab then a*z. After the first,
	// the "a" leads to a state, call it "onA" with no transitions but a spinout, the
	// escape-from-spinout "b" transition is on the spinout state. Then we see ab and we
	// put the "b" transition right on the "onA" state. Then when a*z arrives, we know
	// that the "z" transition goes on the spinout state.

	s1HasSpinout := state1.table.spinout != nil && len(state1.table.epsilons) == 1
	s2HasSpinout := state2.table.spinout != nil && len(state2.table.epsilons) == 1
	s1HasEpsilons := len(state1.table.epsilons) > 0
	s2HasEpsilons := len(state2.table.epsilons) > 0

	switch {
	case isSpinout:
		// caller knows that both sides are spinouts so set up for that
		combined.table.spinout = combined
		combined.table.epsilons = []*faState{combined}
	case s1HasSpinout && s2HasSpinout:
		// both have spinouts so we need to merge them
		combined.table.spinout = mergeFAStates(state1.table.spinout, state2.table.spinout, keyMemo, true, pp)
		combined.table.epsilons = []*faState{combined.table.spinout}
	case s1HasSpinout && !s2HasEpsilons:
		// merge the states byte-wise, adopt s1's spinout
		combined.table.spinout = state1.table.spinout
		combined.table.epsilons = []*faState{combined.table.spinout}
	case s2HasSpinout && !s1HasEpsilons:
		// merge the states byte-wise, adopt s2's spinout
		combined.table.spinout = state2.table.spinout
		combined.table.epsilons = []*faState{combined.table.spinout}
	case s1HasEpsilons || s2HasEpsilons:
		// return a splice
		pp.labelTable(combined.table, "Splice")
		combined.table.epsilons = []*faState{state1, state2}
		keyMemo[mKey] = combined
		return combined
	}

	combined.fieldTransitions = append(state1.fieldTransitions, state2.fieldTransitions...)

	// TODO: Clean this up
	pretty, ok := pp.(*prettyPrinter)
	if ok {
		pp.labelTable(combined.table, fmt.Sprintf("%d∎%d",
			pretty.tableSerial(state1.table), pretty.tableSerial(state2.table)))
	}

	keyMemo[mKey] = combined

	var iter1, iter2 stIterator
	iter1.table = state1.table
	iter2.table = state2.table
	var uComb unpackedTable
	var merged *faState

	for i := 0; i < byteCeiling; i++ {
		next1 := iter1.nextState()
		next2 := iter2.nextState()
		switch {
		case next1 == next2: // no need to merge
			merged = next1
		case next2 == nil: // u1 must be non-nil
			merged = next1
		case next1 == nil: // u2 must be non-nil
			merged = next2
		default: // have to recurse & merge
			merged = mergeFAStates(next1, next2, keyMemo, false, pp)
		}
		uComb[i] = merged
	}

	ceilings := make([]byte, 0, 16)
	steps := make([]*faState, 0, 16)
	lastStep := uComb[0]

	for unpackedIndex, ss := range uComb {
		if ss != lastStep {
			ceilings = append(ceilings, byte(unpackedIndex))
			steps = append(steps, lastStep)
		}
		lastStep = ss
	}
	ceilings = append(ceilings, byte(byteCeiling))
	steps = append(steps, lastStep)
	combined.table.ceilings = ceilings
	combined.table.steps = steps
	return combined
}
