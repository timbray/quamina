package quamina

// closureGeneration is incremented each time epsilonClosure is called.
// Each smallTable stores the generation it was last visited in, avoiding
// the need for a visited map.
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

	// Reuse and clear the map to avoid allocations on each call
	clear(bufs.closureSet)
	if !state.table.isEpsilonOnly() {
		bufs.closureSet[state] = true
	}
	traverseEpsilons(state, state.table.epsilons, bufs.closureSet)

	closure := make([]*faState, 0, len(bufs.closureSet))
	for s := range bufs.closureSet {
		closure = append(closure, s)
	}
	state.epsilonClosure = closure
}

func traverseEpsilons(start *faState, epsilons []*faState, closureSet map[*faState]bool) {
	for _, eps := range epsilons {
		if eps == start || closureSet[eps] {
			continue
		}
		if !eps.table.isEpsilonOnly() {
			closureSet[eps] = true
		}
		traverseEpsilons(start, eps.table.epsilons, closureSet)
	}
}
