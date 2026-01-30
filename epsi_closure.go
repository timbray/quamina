package quamina

// epsilonClosure walks the automaton starting from the given table
// and precomputes the epsilon closure for every reachable faState.
// It returns an faState wrapping the start table with its closure computed.
func epsilonClosure(table *smallTable) *faState {
	startState := &faState{table: table}
	closureForState(startState)
	closureForNfa(table, make(map[*smallTable]bool))
	return startState
}

func closureForNfa(table *smallTable, visited map[*smallTable]bool) {
	if visited[table] {
		return
	}
	visited[table] = true

	for _, state := range table.steps {
		if state != nil {
			closureForState(state)
			closureForNfa(state.table, visited)
		}
	}
	for _, eps := range table.epsilons {
		closureForState(eps)
		closureForNfa(eps.table, visited)
	}
}

// closureForState computes the epsilon closure for a single state.
func closureForState(state *faState) {
	if state.epsilonClosure != nil {
		return
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
