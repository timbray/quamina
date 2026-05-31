package quamina

import "sync"

// tableMark carries the per-table-share-group scratch used by the closure
// post-pass that collapses states sharing a smallTable. It used to live as
// fields on smallTable itself, but that is purely build-time state whose
// permanent presence was wasted steady-state memory; it now lives in a
// pooled side table (closureBuffers.tables).
//
// tableMark is stored by value so marking a share group costs no per-entry
// heap allocation.
type tableMark struct {
	closureGen uint64
	closureRep *faState
}

// closureBuffers carries the scratch for epsilon closure computation. It is
// pooled (see closureBufferPool) and reused across epsilonClosure calls, so
// the maps are allocated once and grown, not rebuilt per call. Visited
// tracking is generation-based: gen only ever increases, so stale map
// entries from a previous use are simply older than the current generation
// and need no clearing.
type closureBuffers struct {
	gen           uint64                      // monotonic counter; bumped by closureForState's two dedup phases
	walkGen       uint64                      // snapshot of gen for the current closureForNfa walk (NFA state dedup)
	closureSetGen uint64                      // snapshot of gen for the current closureForState faState dedup
	closureList   []*faState                  // reusable accumulator for the state list before the dedup post-pass
	tables        map[tableShareKey]tableMark // share-group scratch for the post-pass (closureGen, closureRep)
	states        map[*faState]uint64         // per-faState last-visited gen, used by traverseEpsilons
	walkVisited   map[*faState]uint64         // per-faState last-walked gen, used by closureForNfa
}

func newClosureBuffers() *closureBuffers {
	return &closureBuffers{
		tables:      make(map[tableShareKey]tableMark),
		states:      make(map[*faState]uint64),
		walkVisited: make(map[*faState]uint64),
	}
}

// closureBufferPool reuses closureBuffers (and their maps) across the many
// epsilonClosure calls a build performs, eliminating per-call map allocation.
// The pool is concurrency-safe, and sync.Pool drops its contents on GC, so
// the maps do not become permanent steady-state memory.
var closureBufferPool = sync.Pool{
	New: func() any { return newClosureBuffers() },
}

// epsilonClosure walks the automaton starting from the given state
// and precomputes the epsilon closure for every reachable faState.
func epsilonClosure(start *faState) {
	bufs := closureBufferPool.Get().(*closureBuffers)
	// Take a fresh generation for this walk. closureForState bumps bufs.gen
	// for its own dedup phases, but it never touches walkGen, so the state
	// dedup in closureForNfa compares against a value that stays fixed for
	// the whole walk.
	bufs.gen++
	bufs.walkGen = bufs.gen
	closureForState(start, bufs)
	closureForNfa(start, bufs)
	closureBufferPool.Put(bufs)
}

// closureForNfa dedups by faState identity, not table-share key: each state
// must be walked once. (Share-key dedup is unsafe here — distinct states can
// share a steps backing array yet have different epsilons, and the zero key
// collapses all no-byte tables; the post-pass below re-checks fieldTransitions
// on collision, but the walk has no such guard.)
func closureForNfa(state *faState, bufs *closureBuffers) {
	if bufs.walkVisited[state] == bufs.walkGen {
		return
	}
	bufs.walkVisited[state] = bufs.walkGen

	for _, s := range state.table.steps {
		if s != nil {
			closureForState(s, bufs)
			closureForNfa(s, bufs)
		}
	}
	for _, eps := range state.table.epsilons {
		closureForState(eps, bufs)
		closureForNfa(eps, bufs)
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

	// Generation-based visited tracking: bufs.states records which gen last
	// visited each state, so we never clear the map between traversals.
	bufs.gen++
	bufs.closureSetGen = bufs.gen
	bufs.closureList = bufs.closureList[:0]
	if !state.table.isEpsilonOnly() {
		bufs.states[state] = bufs.closureSetGen
		bufs.closureList = append(bufs.closureList, state)
	}
	traverseEpsilons(state, state.table.epsilons, bufs)

	// Table-pointer dedup: when multiple states in the closure share the
	// same smallTable (steps backing array), their byte transitions are
	// identical, so only one representative is needed. Done as a post-pass
	// over the closure list to keep traverseEpsilons zero-overhead. The
	// zero key (no byte transitions) is never deduped, and states with
	// different fieldTransitions are preserved.
	bufs.gen++
	dedupGen := bufs.gen
	closure := make([]*faState, 0, len(bufs.closureList))
	for _, s := range bufs.closureList {
		key := newTableShareKey(&s.table)
		if (key == tableShareKey{}) {
			closure = append(closure, s)
			continue
		}
		mark := bufs.tables[key]
		if mark.closureGen == dedupGen {
			if sameFieldTransitions(mark.closureRep, s) {
				continue
			}
		} else {
			mark.closureGen = dedupGen
			mark.closureRep = s
			bufs.tables[key] = mark
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
