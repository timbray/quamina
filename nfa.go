package quamina

import "fmt"

// This groups the functions that traverse, merge, and debug Quamina's nondeterministic finite automata

// faState is used by the valueMatcher automaton - every step through the
// automaton requires a smallTable and for some of them, taking the step means you've matched a value and can
// transition to a new fieldMatcher, in which case the fieldTransitions slice will be non-nil
type faState struct {
	table            *smallTable
	fieldTransitions []*fieldMatcher
}

type nfaMetadata struct {
	maxOutDegree int
}

// transmap is a Set structure used to gather transitions as we work our way through the automaton
type transmap struct {
	set map[*fieldMatcher]bool
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

func traverseNFA(table *smallTable, val []byte, transitions []*fieldMatcher, bufs *bufpair) []*fieldMatcher {
	currentStates := bufs.buf1
	currentStates = append(currentStates, &faState{table: table})
	nextStates := bufs.buf2

	// a lot of the transitions stuff is going to be empty, but on the other hand
	// a * entry with a transition could end up getting added a lot.
	newTransitions := &transmap{set: make(map[*fieldMatcher]bool, len(transitions))}
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
			state.table.step(utf8Byte, stepResult)
			if stepResult.step != nil {
				newTransitions.add(stepResult.step.fieldTransitions)
				nextStates = append(nextStates, stepResult.step)
			}
			for _, nextStep := range stepResult.epsilon {
				newTransitions.add(nextStep.fieldTransitions)
				nextStates = append(nextStates, nextStep)
			}
		}
		// re-use these
		swapStates := currentStates
		currentStates = nextStates
		nextStates = swapStates[:0]
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
func mergeFAs(table1, table2 *smallTable, printer printer) *smallTable {
	state1 := &faState{table: table1}
	state2 := &faState{table: table2}
	return mergeFAStates(state1, state2, make(map[faStepKey]*faState), printer).table
}

func mergeFAStates(state1, state2 *faState, keyMemo map[faStepKey]*faState, printer printer) *faState {
	// try to memo-ize
	mKey := faStepKey{state1, state2}
	combined, ok := keyMemo[mKey]
	if ok {
		return combined
	}

	fieldTransitions := append(state1.fieldTransitions, state2.fieldTransitions...)
	combined = &faState{table: newSmallTable(), fieldTransitions: fieldTransitions}

	pretty, ok := printer.(*prettyPrinter)
	if ok {
		printer.labelTable(combined.table, fmt.Sprintf("%d∎%d",
			pretty.tableSerial(state1.table), pretty.tableSerial(state2.table)))
	}

	keyMemo[mKey] = combined
	u1 := unpackTable(state1.table)
	u2 := unpackTable(state2.table)
	var uComb unpackedTable
	for i, next1 := range u1 {
		next2 := u2[i]
		switch {
		case next1 == next2: // no need to merge
			uComb[i] = next1
		case next2 == nil: // u1 must be non-nil
			uComb[i] = next1
		case next1 == nil: // u2 must be non-nil
			uComb[i] = next2
		case i > 0 && next1 == u1[i-1] && next2 == u2[i-1]: // dupe of previous step - happens a lot
			uComb[i] = uComb[i-1]
		default: // have to recurse & merge
			uComb[i] = mergeFAStates(next1, next2, keyMemo, printer)
		}
	}
	combined.table.pack(&uComb)
	combined.table.epsilon = append(state1.table.epsilon, state2.table.epsilon...)

	return combined
}
