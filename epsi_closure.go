package quamina

// closureGeneration is incremented each time epsilonClosure is called.
// Each smallTable stores the generation it was last visited in, avoiding
// the need for a visited map.
var closureGeneration uint64
var closureRepGeneration uint64

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

	// Reuse and clear the set; bump global generation for table-pointer dedup
	clear(bufs.closureSet)
	closureRepGeneration++
	if !state.table.isEpsilonOnly() {
		bufs.closureSet[state] = true
		state.table.closureRepGen = closureRepGeneration
		state.table.closureRep = state
	}
	traverseEpsilons(state, state.table.epsilons, bufs)

	closure := make([]*faState, 0, len(bufs.closureSet))
	for s := range bufs.closureSet {
		closure = append(closure, s)
	}
	state.epsilonClosure = closure
}

// traverseEpsilons recursively collects non-epsilon-only states reachable
// via epsilon transitions into bufs.closureSet. Table-pointer dedup skips
// states whose *smallTable is already represented, avoiding redundant byte
// transitions in the closure. When a table collision has different
// fieldTransitions, the state is still added (correctness over speed) but
// recursion is skipped (same table = same epsilon edges).
func traverseEpsilons(start *faState, epsilons []*faState, bufs *closureBuffers) {
	for _, eps := range epsilons {
		if eps == start || bufs.closureSet[eps] {
			continue
		}
		if !eps.table.isEpsilonOnly() {
			if eps.table.closureRepGen == closureRepGeneration {
				if sameFieldTransitions(eps.table.closureRep, eps) {
					continue
				}
				// Different fieldTransitions on same table: include state
				// to preserve match correctness, but skip recursion since
				// the table's epsilons have already been traversed.
				bufs.closureSet[eps] = true
				continue
			}
			bufs.closureSet[eps] = true
			eps.table.closureRepGen = closureRepGeneration
			eps.table.closureRep = eps
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
