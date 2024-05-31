package quamina

// This groups the functions that traverse, merge, and debug Quamina's nondeterministic finite automata

func traverseFA(table *smallTable, val []byte, transitions []*fieldMatcher) []*fieldMatcher {
	return traverseOneFAStep(table, 0, val, transitions)
}

func traverseOneFAStep(table *smallTable, index int, val []byte, transitions []*fieldMatcher) []*fieldMatcher {
	var utf8Byte byte
	switch {
	case index < len(val):
		utf8Byte = val[index]
	case index == len(val):
		utf8Byte = valueTerminator
	default:
		return transitions
	}
	nextSteps := table.step(utf8Byte)
	if nextSteps == nil {
		return transitions
	}
	index++
	// 1. Note no effort to traverse multiple next-steps in parallel. The traversal compute is tiny and the
	//    necessary concurrency apparatus would almost certainly outweigh it
	// 2. TODO: It would probably be better to implement this iteratively rather than recursively.
	//    The recursion will potentially go as deep as the val argument is long.
	for _, nextStep := range nextSteps.steps {
		transitions = append(transitions, nextStep.fieldTransitions...)
		transitions = traverseOneFAStep(nextStep.table, index, val, transitions)
	}
	return transitions
}

// mergeFAs compute the union of two valueMatch automata.  If you look up the textbook theory about this,
// they say to compute the set product for automata A and B and build A0B0, A0B1 … A1BN, A1B0 … but if you look
// at that you realize that many of the product states aren't reachable. So you compute A0B0 and then keep
// recursing on the transitions coming out, I'm pretty sure you get a correct result. I don't know if it's
// minimal or even avoids being wasteful.
// INVARIANT: neither argument is nil
// INVARIANT: To be thread-safe, no existing table can be updated except when we're building it

type faStepKey struct {
	step1 *faState
	step2 *faState
}

func mergeFAs(table1, table2 *smallTable) *smallTable {
	state1 := &faState{table: table1}
	state2 := &faState{table: table2}
	return mergeFAStates(state1, state2, make(map[faStepKey]*faState)).table
}

// TODO: maybe memoize these based on the string of characters you matched to get here?
// TODO: recursion seems way too deep
func mergeFAStates(state1, state2 *faState, keyMemo map[faStepKey]*faState) *faState {
	var combined *faState
	mKey := faStepKey{state1, state2}
	combined, ok := keyMemo[mKey]
	if ok {
		return combined
	}

	newTable := newSmallTable()

	fieldTransitions := append(state1.fieldTransitions, state2.fieldTransitions...)
	combined = &faState{table: newTable, fieldTransitions: fieldTransitions}
	//DEBUG combined.table.label = fmt.Sprintf("(%s ∎ %s)", state1.table.label, state2.table.label)
	keyMemo[mKey] = combined
	u1 := unpackTable(state1.table)
	u2 := unpackTable(state2.table)
	var uComb unpackedTable

	for i, next1 := range u1 {
		next2 := u2[i]
		switch {
		case next1 == nil && next2 == nil:
			uComb[i] = nil
		case next1 != nil && next2 == nil:
			uComb[i] = u1[i]
		case next1 == nil && next2 != nil:
			uComb[i] = u2[i]
		case next1 != nil && next2 != nil:
			//fmt.Printf("MERGE %s & %s i=%d d=%d: ", next1, next2, i, depth)
			if next1 == next2 {
				//	fmt.Println("n1 == n2")
				uComb[i] = next1
			} else if i > 0 && next1 == u1[i-1] && next2 == u2[i-1] {
				uComb[i] = uComb[i-1]
				//	fmt.Printf("SEQ %s\n", uComb[i].steps[0].table.shortDump())
			} else {
				//	fmt.Println("RECURSE!")
				var comboNext []*faState
				for _, nextStep1 := range next1.steps {
					for _, nextStep2 := range next2.steps {
						comboNext = append(comboNext, mergeFAStates(nextStep1, nextStep2, keyMemo))
					}
				}
				uComb[i] = &faNext{steps: comboNext}
				//DEBUG uComb[i].serial = *serial
			}
		}
	}
	combined.table.pack(&uComb)

	return combined
}
