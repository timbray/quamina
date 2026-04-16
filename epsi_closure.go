package quamina

// tableMark carries the per-smallTable scratch used only during epsilon
// closure computation (lastVisitedGen for NFA walk dedup, and closureGen /
// closureRep for table-pointer dedup). These used to live as fields on
// smallTable itself, but they are purely build-time state and their
// permanent presence on every smallTable was wasted steady-state memory.
// They now live in a per-call side table that is discarded when
// epsilonClosure returns.
type tableMark struct {
	lastVisitedGen uint32
	closureGen     uint32
	closureRep     *faState
}

// closureBuffers carries per-epsilonClosure-call scratch. The two maps
// replace build-time fields that used to sit on smallTable/faState;
// they live only for the duration of the closure computation.
type closureBuffers struct {
	gen           uint32
	closureSetGen uint32
	closureList   []*faState
	tables        map[*smallTable]*tableMark
	states        map[*faState]uint32
}

func newClosureBuffers() *closureBuffers {
	return &closureBuffers{
		gen:    1,
		tables: make(map[*smallTable]*tableMark),
		states: make(map[*faState]uint32),
	}
}

// tableMarkOf returns the tableMark for t, creating one on first access.
func (b *closureBuffers) tableMarkOf(t *smallTable) *tableMark {
	m, ok := b.tables[t]
	if !ok {
		m = &tableMark{}
		b.tables[t] = m
	}
	return m
}

// epsilonClosure walks the automaton starting from the given table
// and precomputes the epsilon closure for every reachable faState.
func epsilonClosure(table *smallTable) {
	bufs := newClosureBuffers()
	closureForNfa(table, bufs)
}

func closureForNfa(table *smallTable, bufs *closureBuffers) {
	mark := bufs.tableMarkOf(table)
	if mark.lastVisitedGen == bufs.gen {
		return
	}
	mark.lastVisitedGen = bufs.gen

	for _, state := range table.steps {
		if state != nil {
			closureForState(state, bufs)
			closureForNfa(state.table, bufs)
		}
	}
	for _, eps := range table.epsilons {
		closureForState(eps, bufs)
		closureForNfa(eps.table, bufs)
	}
}

// closureForStateNoBufs computes the epsilon closure for a single state.
// Used directly in tests; production code uses closureForState.
func closureForStateNoBufs(state *faState) {
	bufs := newClosureBuffers()
	closureForState(state, bufs)
}

func closureForState(state *faState, bufs *closureBuffers) {
	if state.epsilonClosure != nil {
		return
	}

	if len(state.table.epsilons) == 0 {
		state.epsilonClosure = []*faState{state}
		return
	}

	// Use generation-based visited tracking instead of a fresh map per
	// traversal. bufs.states records which gen last visited each state.
	bufs.gen++
	bufs.closureSetGen = bufs.gen
	bufs.closureList = bufs.closureList[:0]
	if !state.table.isEpsilonOnly() {
		bufs.states[state] = bufs.closureSetGen
		bufs.closureList = append(bufs.closureList, state)
	}
	traverseEpsilons(state, state.table.epsilons, bufs)

	// Table-pointer dedup: when multiple states in the closure share the
	// same *smallTable, their byte transitions are identical, so only one
	// representative is needed. This is done as a post-pass over the
	// closure list rather than during traversal to keep traverseEpsilons
	// zero-overhead. States with different fieldTransitions are preserved.
	bufs.gen++
	dedupGen := bufs.gen
	closure := make([]*faState, 0, len(bufs.closureList))
	for _, s := range bufs.closureList {
		mark := bufs.tableMarkOf(s.table)
		if mark.closureGen == dedupGen {
			if sameFieldTransitions(mark.closureRep, s) {
				continue
			}
		} else {
			mark.closureGen = dedupGen
			mark.closureRep = s
		}
		closure = append(closure, s)
	}
	state.epsilonClosure = closure
}

// traverseEpsilons recursively collects non-epsilon-only states reachable
// via epsilon transitions into bufs.closureList.
func traverseEpsilons(start *faState, epsilons []*faState, bufs *closureBuffers) {
	for _, eps := range epsilons {
		if eps == start || bufs.states[eps] == bufs.closureSetGen {
			continue
		}
		bufs.states[eps] = bufs.closureSetGen
		if !eps.table.isEpsilonOnly() {
			bufs.closureList = append(bufs.closureList, eps)
		}
		traverseEpsilons(start, eps.table.epsilons, bufs)
	}
}

// sameFieldTransitions reports whether two states have identical fieldTransitions.
// This does an order-dependent comparison. If the same field matchers appear in
// different order, we'll miss the dedup — but that just keeps an extra state in
// the closure (a missed optimization, not a correctness bug). In practice,
// fieldTransitions almost always has 0 or 1 element, so ordering doesn't matter.
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
