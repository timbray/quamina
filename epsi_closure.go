package quamina

type epsilonClosure struct {
	closures map[*faState][]*faState
	slab     []*faState
}

func newEpsilonClosure() *epsilonClosure {
	return &epsilonClosure{
		closures: make(map[*faState][]*faState),
		slab:     make([]*faState, 0, 1024),
	}
}

func (ec *epsilonClosure) reset() {
	clear(ec.closures)
	ec.slab = ec.slab[:0]
}

func (ec *epsilonClosure) getClosure(state *faState) []*faState {
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
		start := len(ec.slab)
		ec.slab = append(ec.slab, state)
		justMe := ec.slab[start : start+1]

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

	start := len(ec.slab)
	for s := range closureStates {
		ec.slab = append(ec.slab, s)
	}
	closure = ec.slab[start:]

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
