package quamina

// precomputeEpsilonClosures walks the automaton starting from the given table
// and precomputes the epsilon closure for every reachable faState.
func precomputeEpsilonClosures(table *smallTable) {
	visited := make(map[*smallTable]bool)
	precomputeClosuresRecursive(table, visited)
}

func precomputeClosuresRecursive(table *smallTable, visited map[*smallTable]bool) {
	if visited[table] {
		return
	}
	visited[table] = true

	// Process each faState reachable via byte transitions
	for _, state := range table.steps {
		if state != nil {
			computeClosureForState(state)
			precomputeClosuresRecursive(state.table, visited)
		}
	}
	// Process each faState reachable via epsilon transitions
	for _, eps := range table.epsilons {
		computeClosureForState(eps)
		precomputeClosuresRecursive(eps.table, visited)
	}
}

func computeClosureForState(state *faState) {
	if state.epsilonClosure != nil {
		return // already computed
	}

	if len(state.table.epsilons) == 0 {
		state.epsilonClosure = []*faState{state}
		return
	}

	closureSet := make(map[*faState]bool)
	if !state.table.isEpsilonOnly() {
		closureSet[state] = true
	}
	traverseEpsilons(state, state.table.epsilons, closureSet)

	closure := make([]*faState, 0, len(closureSet))
	for s := range closureSet {
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
