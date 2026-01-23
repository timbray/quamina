package quamina

type epsilonClosure struct {
	closures  map[*faState][]*faState
	hotCache  [8]*faState   // Hot states (keys)
	hotValues [8][]*faState // Corresponding closures (values)
	hotIndex  int           // Round-robin index for cache replacement
}

func newEpsilonClosure() *epsilonClosure {
	return &epsilonClosure{
		closures: make(map[*faState][]*faState),
	}
}

func (ec *epsilonClosure) reset() {
	clear(ec.closures)
	// Clear hot cache
	for i := range ec.hotCache {
		ec.hotCache[i] = nil
		ec.hotValues[i] = nil
	}
	ec.hotIndex = 0
}

func (ec *epsilonClosure) getClosure(state *faState) []*faState {
	// Check hot cache first (unrolled for performance)
	if ec.hotCache[0] == state {
		return ec.hotValues[0]
	}
	if ec.hotCache[1] == state {
		return ec.hotValues[1]
	}
	if ec.hotCache[2] == state {
		return ec.hotValues[2]
	}
	if ec.hotCache[3] == state {
		return ec.hotValues[3]
	}
	if ec.hotCache[4] == state {
		return ec.hotValues[4]
	}
	if ec.hotCache[5] == state {
		return ec.hotValues[5]
	}
	if ec.hotCache[6] == state {
		return ec.hotValues[6]
	}
	if ec.hotCache[7] == state {
		return ec.hotValues[7]
	}

	// Check main map cache
	var closure []*faState
	var ok bool
	if ec.closures != nil {
		closure, ok = ec.closures[state]
		if ok {
			// Promote to hot cache
			ec.hotCache[ec.hotIndex] = state
			ec.hotValues[ec.hotIndex] = closure
			ec.hotIndex = (ec.hotIndex + 1) & 7 // Wrap around using mask
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
		// Add to hot cache
		ec.hotCache[ec.hotIndex] = state
		ec.hotValues[ec.hotIndex] = closure
		ec.hotIndex = (ec.hotIndex + 1) & 7 // Wrap around using mask
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
