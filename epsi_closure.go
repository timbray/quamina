package quamina

// closureGeneration is a global counter used for generation-based visited
// tracking. The NFA walk in closureForNfa snapshots it into bufs.generation
// and compares against that; the dedup pass in closureForStateWithBufs
// increments it and compares against the new value. Both use the single
// lastVisitedGen field on smallTable. If a dedup pass overwrites a table's
// lastVisitedGen, the NFA walk may revisit that table harmlessly since
// closureForStateWithBufs early-returns when epsilonClosure is already set.
var closureGeneration uint64

// epsilonClosure walks the automaton starting from the given table
// and precomputes the epsilon closure for every reachable faState.
func epsilonClosure(table *smallTable) {
	closureGeneration++
	bufs := &closureBuffers{
		generation: closureGeneration,
		closureSet: make(map[*faState]bool, 64),
	}
	closureForNfa(table, bufs)
}

type closureBuffers struct {
	generation uint64
	closureSet map[*faState]bool
}

func closureForNfa(table *smallTable, bufs *closureBuffers) {
	if table.lastVisitedGen == bufs.generation {
		return
	}
	table.lastVisitedGen = bufs.generation

	for _, state := range table.steps {
		if state != nil {
			closureForStateWithBufs(state, bufs)
			closureForNfa(state.table, bufs)
		}
	}
	for _, eps := range table.epsilons {
		closureForStateWithBufs(eps, bufs)
		closureForNfa(eps.table, bufs)
	}
}

// closureForState computes the epsilon closure for a single state.
// Used directly in tests; production code uses closureForStateWithBufs.
func closureForState(state *faState) {
	bufs := &closureBuffers{
		closureSet: make(map[*faState]bool, 64),
	}
	closureForStateWithBufs(state, bufs)
}

func closureForStateWithBufs(state *faState, bufs *closureBuffers) {
	if state.epsilonClosure != nil {
		return
	}

	if len(state.table.epsilons) == 0 {
		state.epsilonClosure = []*faState{state}
		return
	}

	// Reuse and clear the set
	clear(bufs.closureSet)
	if !state.table.isEpsilonOnly() {
		bufs.closureSet[state] = true
	}
	traverseEpsilons(state, state.table.epsilons, bufs)

	// Table-pointer dedup: when multiple states in the closure share the
	// same *smallTable, their byte transitions are identical, so only one
	// representative is needed. This is done as a post-pass over the
	// closure set rather than during traversal to keep traverseEpsilons
	// zero-overhead. States with different fieldTransitions are preserved.
	closureGeneration++
	closure := make([]*faState, 0, len(bufs.closureSet))
	for s := range bufs.closureSet {
		if s.table.lastVisitedGen == closureGeneration {
			if sameFieldTransitions(s.table.closureRep, s) {
				continue
			}
		} else {
			s.table.lastVisitedGen = closureGeneration
			s.table.closureRep = s
		}
		closure = append(closure, s)
	}
	state.epsilonClosure = closure
}

// traverseEpsilons recursively collects non-epsilon-only states reachable
// via epsilon transitions into bufs.closureSet.
func traverseEpsilons(start *faState, epsilons []*faState, bufs *closureBuffers) {
	for _, eps := range epsilons {
		if eps == start || bufs.closureSet[eps] {
			continue
		}
		if !eps.table.isEpsilonOnly() {
			bufs.closureSet[eps] = true
		}
		traverseEpsilons(start, eps.table.epsilons, bufs)
	}
}

// sameFieldTransitions reports whether two states have identical fieldTransitions.
func sameFieldTransitions(a, b *faState) bool {
	if len(a.fieldTransitions) != len(b.fieldTransitions) {
		return false
	}
	for i, fm := range a.fieldTransitions {
		if fm != b.fieldTransitions[i] {
			return false
		}
	}
	return true
}
