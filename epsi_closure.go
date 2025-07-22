package quamina

type epsilonClosure struct {
	closures map[*faState][]*faState
}

func newEpsilonClosure() *epsilonClosure {
	return &epsilonClosure{make(map[*faState][]*faState)}
}

func getEpsilonClosure(state *faState) []*faState {
	emptyEC := &epsilonClosure{closures: nil}
	return emptyEC.getAndCacheClosure(state)
}

func (ec *epsilonClosure) getAndCacheClosure(state *faState) []*faState {
	var closure []*faState
	var ok bool
	if ec.closures != nil {
		closure, ok = ec.closures[state]
		if ok {
			return closure
		}
	}

	// not already known
	if len(state.table.epsilons) == 0 {
		justMe := []*faState{state}
		if ec.closures != nil {
			ec.closures[state] = justMe
		}
		return justMe
	}

	var closureStates = make(map[*faState]bool)
	if !state.table.isEpsilonOnly() {
		closureStates[state] = true
	}
	traverseEpsilons(state, state.table.epsilons, closureStates)
	for s := range closureStates {
		closure = append(closure, s)
	}
	if ec.closures != nil {
		ec.closures[state] = closure
	}
	return closure
}

func traverseEpsilons(start *faState, epsilons []*faState, closureStates map[*faState]bool) {
	for _, eps := range epsilons {
		if eps == start || closureStates[eps] {
			continue
		}
		if !eps.table.isEpsilonOnly() {
			closureStates[eps] = true
		}
		traverseEpsilons(start, eps.table.epsilons, closureStates)
	}
}
