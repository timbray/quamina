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
		tableRep:  make(map[*smallTable]*faState, 64),
	}
	closureForNfa(table, bufs)
}

type closureBuffers struct {
	generation uint64
	closureSet map[*faState]bool
	tableRep   map[*smallTable]*faState
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
		tableRep:  make(map[*smallTable]*faState, 64),
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

	// Reuse and clear the maps to avoid allocations on each call
	clear(bufs.closureSet)
	clear(bufs.tableRep)
	if !state.table.isEpsilonOnly() {
		bufs.closureSet[state] = true
		bufs.tableRep[state.table] = state
	}
	traverseEpsilons(state, state.table.epsilons, bufs)

	closure := make([]*faState, 0, len(bufs.closureSet))
	for s := range bufs.closureSet {
		closure = append(closure, s)
	}
	state.epsilonClosure = closure
}

func traverseEpsilons(start *faState, epsilons []*faState, bufs *closureBuffers) {
	for _, eps := range epsilons {
		if eps == start || bufs.closureSet[eps] {
			continue
		}
		if !eps.table.isEpsilonOnly() {
			if rep, ok := bufs.tableRep[eps.table]; ok {
				// Same table already in closure. Safe to skip only if
				// fieldTransitions match — otherwise we'd lose matches.
				if sameFieldTransitions(rep, eps) {
					continue
				}
				bufs.closureSet[eps] = true
				continue // same table means same epsilons, skip recursion
			}
			bufs.closureSet[eps] = true
			bufs.tableRep[eps.table] = eps
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
