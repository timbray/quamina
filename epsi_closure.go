package quamina

// closureGeneration is a global counter used for generation-based visited
// tracking. It is incremented by epsilonClosure (for NFA walk dedup via
// lastVisitedGen) and by closureForStateWithBufs (for table-pointer dedup
// via closureRepGen). Each smallTable stores the generation it was last
// visited in, avoiding the need for a visited map. This works because
// epsilonClosure snapshots the counter into bufs.generation before the
// walk begins, so subsequent increments by the dedup pass don't interfere.
var closureGeneration uint64

// epsilonClosure walks the automaton starting from the given table
// and precomputes the epsilon closure for every reachable faState.
func epsilonClosure(table *smallTable) {
	closureGeneration++
	bufs := &closureBuffers{
		generation: closureGeneration,
	}
	closureForNfa(table, bufs)
}

type closureBuffers struct {
	generation    uint64
	closureSetGen uint64
	closureList   []*faState
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
	bufs := &closureBuffers{}
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

	// Use generation-based visited tracking instead of a map
	closureGeneration++
	bufs.closureSetGen = closureGeneration
	bufs.closureList = bufs.closureList[:0]
	if !state.table.isEpsilonOnly() {
		state.closureSetGen = bufs.closureSetGen
		bufs.closureList = append(bufs.closureList, state)
	}
	traverseEpsilons(state, state.table.epsilons, bufs)

	// Table-pointer dedup: when multiple states in the closure share the
	// same *smallTable, their byte transitions are identical, so only one
	// representative is needed. This is done as a post-pass over the
	// closure list rather than during traversal to keep traverseEpsilons
	// zero-overhead. States with different fieldTransitions are preserved.
	closureGeneration++
	closure := make([]*faState, 0, len(bufs.closureList))
	for _, s := range bufs.closureList {
		if s.table.closureRepGen == closureGeneration {
			if sameFieldTransitions(s.table.closureRep, s) {
				continue
			}
		} else {
			s.table.closureRepGen = closureGeneration
			s.table.closureRep = s
		}
		closure = append(closure, s)
	}
	state.epsilonClosure = closure
}

// traverseEpsilons recursively collects non-epsilon-only states reachable
// via epsilon transitions into bufs.closureList.
func traverseEpsilons(start *faState, epsilons []*faState, bufs *closureBuffers) {
	for _, eps := range epsilons {
		if eps == start || eps.closureSetGen == bufs.closureSetGen {
			continue
		}
		eps.closureSetGen = bufs.closureSetGen
		if !eps.table.isEpsilonOnly() {
			bufs.closureList = append(bufs.closureList, eps)
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
