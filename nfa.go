package quamina

import (
	"cmp"
	"fmt"
	"slices"
	"unsafe"
)

// This groups the functions that traverse, merge, and debug Quamina's nondeterministic finite automata

// faState is used by the valueMatcher automaton - every step through the
// automaton requires a smallTable and for some of them, taking the step means you've matched a value and can
// transition to a new fieldMatcher, in which case the fieldTransitions slice will be non-nil
type faState struct {
	table            *smallTable
	fieldTransitions []*fieldMatcher
	isSpinner        bool
	epsilonClosure   []*faState // precomputed epsilon closure including self
}

/*
Here's the problem. When you have the shellstyle *, which really means ".*", there are options on how
to implement, and they have effect on what you can do while merging, with the results highlighted by
TestShellStyleBuildTime().

Building a smallTable where all the entries link back to the table is easy, so how do you build an
"escape" transition, e.g. on C for A*C?

Plan A: Go back to the era of *faNext, then it's easy to link to multiple things.  Not obvious to
see how to optimize merging though.

Plan B: Have the everything-points-to-itself smallTable and to support exit-on-C, transition
to a state that has whatever comes after C and also an epsilon linking back to the spin state.
This would require some sort of marker in the spin state so that when merging, you can
spot whether either of the states being merged is a spin state and optimize.  Advantage: no
special-casing in traverseNFA

Plan C: Have a boolean isSpinState in either the faState or smallTable, and teach traverseNfa()
to always take that step. Advantage: Save the Plan-B epsilon voodoo.

OK, went with Plan B, go test passes but TestBuildShellStyle is down to only 2k events/second with
huge numbers of states and splices. So, we want to optimize merging.

*/

// transmap is a Set structure used to gather transitions as we work our way through the automaton
type transmap struct {
	set map[*fieldMatcher]bool
}

func newTransMap() *transmap {
	return &transmap{set: make(map[*fieldMatcher]bool)}
}

func (tm *transmap) reset() {
	clear(tm.set)
}

func (tm *transmap) add(fms []*fieldMatcher) {
	for _, fm := range fms {
		tm.set[fm] = true
	}
}

func (tm *transmap) all() []*fieldMatcher {
	if len(tm.set) == 0 {
		return nil
	}
	all := make([]*fieldMatcher, 0, len(tm.set))
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
	buf1, buf2     []*faState
	matches        *matchSet
	transitionsBuf []*fieldMatcher
	resultBuf      []X
	transmap       *transmap
}

func newNfaBuffers() *nfaBuffers {
	return &nfaBuffers{
		transitionsBuf: make([]*fieldMatcher, 0, 16),
		resultBuf:      make([]X, 0, 16),
	}
}

func (nb *nfaBuffers) getBuf1() []*faState {
	if nb.buf1 == nil {
		nb.buf1 = make([]*faState, 0, 16)
	}
	return nb.buf1
}

func (nb *nfaBuffers) getBuf2() []*faState {
	if nb.buf2 == nil {
		nb.buf2 = make([]*faState, 0, 16)
	}
	return nb.buf2
}

func (nb *nfaBuffers) getMatches() *matchSet {
	if nb.matches == nil {
		nb.matches = newMatchSet()
	}
	return nb.matches
}

func (nb *nfaBuffers) getTransmap() *transmap {
	if nb.transmap == nil {
		nb.transmap = newTransMap()
	}
	return nb.transmap
}

// nfa2Dfa does what the name says, but as of 2025/12 is not used.
func nfa2Dfa(nfaTable *smallTable) *faState {
	// The start state always has a trivial epsilon closure (just itself) because
	// all Quamina automata begin by matching the opening quote (0x22). The start
	// table therefore has a single transition on `"` and never has epsilons.
	startState := &faState{table: nfaTable}
	startState.epsilonClosure = []*faState{startState}
	startNfa := []*faState{startState}
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
		nStates = append(nStates, rawNState.epsilonClosure...)
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
	currentStates := bufs.getBuf1()
	// The start state always has a trivial epsilon closure (just itself) because
	// all Quamina automata begin by matching the opening quote (0x22). The start
	// table therefore has a single transition on `"` and never has epsilons.
	startState := &faState{table: table}
	startState.epsilonClosure = []*faState{startState}
	currentStates = append(currentStates, startState)
	nextStates := bufs.getBuf2()

	// a lot of the transitions stuff is going to be empty, but on the other hand
	// a * entry with a transition could end up getting added a lot. While this
	// involves memory allocation, in the vast majority of cases matching an event
	// will turn up a tiny number of unique matches, so allocation should be minimal
	newTransitions := bufs.getTransmap()
	newTransitions.reset()
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
			for _, ecState := range state.epsilonClosure {
				newTransitions.add(ecState.fieldTransitions)
				ecState.table.step(utf8Byte, stepResult)
				if stepResult.step != nil {
					nextStates = append(nextStates, stepResult.step)
				}
			}
		}

		// for toxically-complex regexps like (([abc]?)*)+ you can get a FA with epsilon loops,
		// direct and indirect, which can lead to huge nextState buildups.  Could solve this with
		// making it a set, but this seems to work well enough
		// TODO: Investigates slices.Compact()
		if len(nextStates) > 500 {
			slices.SortFunc(nextStates, func(a, b *faState) int {
				return cmp.Compare(uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b)))
			})
			uniques := 0
			for maybes := 1; maybes < len(nextStates); maybes++ {
				if nextStates[maybes] != nextStates[uniques] {
					uniques++
					nextStates[uniques] = nextStates[maybes]
				}
			}
			uniques++
			nextStates = nextStates[:uniques]
		}

		// re-use these
		swapStates := currentStates
		currentStates = nextStates
		nextStates = swapStates[:0]
	}

	// we've run out of input bytes so we need to check the current states and their
	// epsilon closures for matches
	for _, state := range currentStates {
		for _, ecState := range state.epsilonClosure {
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

func makeFaStepKey(s1, s2 *faState) faStepKey {
	if uintptr(unsafe.Pointer(s1)) < uintptr(unsafe.Pointer(s2)) {
		return faStepKey{s1, s2}
	} else {
		return faStepKey{s2, s1}
	}
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
	return mergeFAStates(state1, state2, make(map[faStepKey]*faState), pp).table
}

func mergeFAStates(state1, state2 *faState, keyMemo map[faStepKey]*faState, pp printer) *faState {
	// try to memo-ize
	mKey := makeFaStepKey(state1, state2)
	combined, ok := keyMemo[mKey]
	if ok {
		return combined
	}
	combined = &faState{table: newSmallTable()}

	// special casing for loopback states as found in shellStyle and wildcard patterns.
	// The idea is that when either state being merged has epsilons, we have to splice them. But if
	// the epsilon is implementing the special case of a "spinner" state that needs to branch back
	// to itself, we can merge these without creating a splice
	// TODO: This is still creating way too many splice states and slowing down traversal. Fix that.
	switch {
	case state1.isSpinner && state2.isSpinner:
		pp.labelTable(combined.table, "2Spinners")
		combined = symmetricSpinnerMerge(state1, state2, keyMemo, pp)
		keyMemo[mKey] = combined
		return combined

	case state1.isSpinner && (len(state2.table.epsilons) == 0):
		// state2 isn't
		combined = asymmetricSpinnerMerge(state1, state2, keyMemo, pp)
		keyMemo[mKey] = combined
		return combined

	case state2.isSpinner && len(state1.table.epsilons) == 0:
		// state1 isn't
		combined = asymmetricSpinnerMerge(state2, state1, keyMemo, pp)
		keyMemo[mKey] = combined
		return combined
	}

	// If either of the states to be merged has epsilons we have to do a splice
	// TODO: Find more cases when we don't have to
	if len(state1.table.epsilons) != 0 || len(state2.table.epsilons) != 0 {
		pp.labelTable(combined.table, "Splice")
		combined.table.epsilons = []*faState{state1, state2}
		keyMemo[mKey] = combined
		return combined
	}

	combined.fieldTransitions = append(state1.fieldTransitions, state2.fieldTransitions...)

	pp.labelTable(combined.table, fmt.Sprintf("%d∎%d",
		pp.tableSerial(state1.table), pp.tableSerial(state2.table)))

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
			merged = mergeFAStates(next1, next2, keyMemo, pp)
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

func asymmetricSpinnerMerge(spinner, nonSpinner *faState, keyMemo map[faStepKey]*faState, pp printer) *faState {
	mKey := makeFaStepKey(spinner, nonSpinner)
	combined := &faState{table: newSmallTable()}
	combined.fieldTransitions = append(spinner.fieldTransitions, nonSpinner.fieldTransitions...)

	pp.labelTable(combined.table, fmt.Sprintf("%d∎%d",
		pp.tableSerial(spinner.table), pp.tableSerial(nonSpinner.table)))

	keyMemo[mKey] = combined

	var iter1, iter2 stIterator
	iter1.table = spinner.table
	iter2.table = nonSpinner.table
	var uComb unpackedTable
	var mergedState *faState

	for utf8byte := 0; utf8byte < byteCeiling; utf8byte++ {
		spinnerNext := iter1.nextState()
		nonSpinnernext := iter2.nextState()

		switch {
		case spinnerNext == nil:
			// illegal UTF-8
			mergedState = nil

		case nonSpinnernext == nil:
			mergedState = spinnerNext

		case spinnerNext == spinner:
			// nonspinner has a branch here
			// if the current spinner value is a loopback, we need to make a new state whose value
			// is the nonspinner with the addition of the epsilon link back to the spinner
			mergedTable := &smallTable{
				steps:    nonSpinnernext.table.steps,
				ceilings: nonSpinnernext.table.ceilings,
				epsilons: append(nonSpinnernext.table.epsilons, spinner),
			}
			mergedState = &faState{table: mergedTable, fieldTransitions: spinner.fieldTransitions}

		default:
			// if spinner's branch isn't a loopback, we need to merge its target with the nonspinner
			// while preserving the epsilon to the spinner
			mergedState = mergeFAStates(spinnerNext, nonSpinnernext, keyMemo, pp)
			mergedState.table.epsilons = append(mergedState.table.epsilons, spinner)
		}
		uComb[utf8byte] = mergedState
	}

	// the following inlines the smallTable pack() function for efficiency
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

func symmetricSpinnerMerge(state1, state2 *faState, keyMemo map[faStepKey]*faState, pp printer) *faState {
	combined := &faState{table: newSmallTable()}
	combined.fieldTransitions = append(state1.fieldTransitions, state2.fieldTransitions...)

	pp.labelTable(combined.table, fmt.Sprintf("%d∎%d",
		pp.tableSerial(state1.table), pp.tableSerial(state2.table)))

	keyMemo[makeFaStepKey(state1, state2)] = combined

	var iter1, iter2 stIterator
	iter1.table = state1.table
	iter2.table = state2.table
	var uComb unpackedTable
	var mergedState *faState

	for i := 0; i < byteCeiling; i++ {
		next1 := iter1.nextState()
		next2 := iter2.nextState()

		switch {
		case next1 == nil:
			// illegal UTF-8
			mergedState = nil
		case next1 == state1 && next2 == state2:
			// both otherwise empty spin steps
			mergedState = combined

		case next1 == state1 && next2 != state2:
			// next2 is an actual branch, so we will have to install the spin pointer in the target
			table := &smallTable{
				ceilings: next2.table.ceilings,
				steps:    next2.table.steps,
				epsilons: append(state2.table.epsilons, combined),
			}
			mergedState = &faState{
				table:            table,
				fieldTransitions: next2.fieldTransitions,
			}
		case next2 == state2 && next1 != state1:
			// next1 is an actual branch, so we will have to install the spin pointer in the target
			table := &smallTable{
				ceilings: next1.table.ceilings,
				steps:    next1.table.steps,
				epsilons: append(state1.table.epsilons, combined),
			}
			mergedState = &faState{
				table:            table,
				fieldTransitions: next1.fieldTransitions,
			}
		default:
			// neither is a spin link
			mergedState = mergeFAStates(next1, next2, keyMemo, pp)
			mergedState.table.epsilons = append(mergedState.table.epsilons, combined)
		}
		uComb[i] = mergedState
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
