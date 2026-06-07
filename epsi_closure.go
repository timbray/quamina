package quamina

import "sync"

// Epsilon-closure construction
// =============================
//
// An NFA state's epsilon closure is the set of states reachable from it by
// following epsilon (zero-input) transitions, plus itself. At match time
// traverseNFA consumes a state's precomputed closure instead of chasing
// epsilons live, so the closures must be built eagerly: AddPattern merges the
// new value's FA into the existing one and then calls epsilonClosure on the
// start state, which closes every state the merge created. (This is an eager
// architecture on purpose — closures sit on the match hot path; lazy/on-demand
// construction is a separate line of work.)
//
// Construction is three nested passes:
//
//   epsilonClosure   — entry point; grabs pooled scratch and starts the walk.
//   closureForNfa    — walks the reachable NFA once, calling closureForState on
//                      each not-yet-closed state, then recursing over its steps
//                      and epsilons.
//   closureForState  — computes one state's closure: collects the epsilon-
//                      reachable states (traverseEpsilons), then dedups them.
//
// Three optimizations keep this cheap, and they are why the scratch in
// closureBuffers looks busier than the core idea:
//
//  1. Incremental walk (closureForNfa). A state whose epsilonClosure is already
//     non-nil was closed by a previous AddPattern; merges only ever add new
//     states as ancestors of closed subtrees, so the walk can prune at any
//     closed state. Without this the walk re-traverses the whole, ever-growing
//     NFA on every add — O(N²) over N adds.
//
//  2. Self-only sentinel (selfOnlyClosure). The vast majority of states close to
//     just {self}; storing that as a shared zero-length slice avoids a heap
//     allocation (and resident, GC-scanned backing array) per such state. See
//     selfOnlyClosure below for the nil / len==0 / len>=2 encoding.
//
//  3. Table-pointer dedup (the post-pass in closureForState). States that share
//     a steps backing array have identical byte transitions, so only one
//     representative per share group is kept — see dedup_key.go and tableMark.
//     self must stay inside multi-member closures (pulling it out defeats this
//     dedup and reintroduces exponential blowup on adversarial regexps), which
//     is why only the self-only case is specialized, not "drop self everywhere".
//
// Generation counters avoid clearing the scratch maps between the many closures
// a build computes. Every "visited in this pass?" check compares a stored
// generation against the current one; bumping the current generation logically
// empties the map in O(1). closureBuffers tracks several independent passes
// (the NFA walk, the per-closure epsilon traversal, and the dedup post-pass),
// each with its own generation snapshot, which is what the fields below encode.

// selfOnlyClosure is the shared sentinel for a closure equal to {self}: non-nil
// (distinct from nil = "not computed") and zero-length (so consumers take their
// len==0 self-only branch). The empty composite literal points at runtime
// zerobase, so this allocates nothing.
var selfOnlyClosure = []*faState{}

// tableMark carries the per-table-share-group scratch used by the closure
// post-pass that collapses states sharing a smallTable. It used to live as
// fields on smallTable itself, but that is purely build-time state whose
// permanent presence was wasted steady-state memory; it now lives in a
// pooled side table (closureBuffers.tables).
//
// tableMark is stored by value so marking a share group costs no per-entry
// heap allocation.
type tableMark struct {
	// closureGen is the dedup generation in which this mark was last written.
	// The map is never cleared between closures, so a mark only counts when its
	// closureGen equals the current dedupGen; older values are stale and ignored.
	closureGen uint64
	// closureRep is the representative faState for this table-share group: the
	// first state in the current closure found to use this smallTable. Later
	// states that share the table collapse onto closureRep when their
	// fieldTransitions match it (sameFieldTransitions), so only the
	// representative survives into the deduped closure.
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
	// nfaWalkCount counts states actually processed by closureForNfa (past the
	// prune + dedup guards). Reset to zero at the start of each epsilonClosure
	// call (per-walk count, not a pooled lifetime total). Used by tests to
	// assert the walk is incremental; negligible in production and never read
	// there.
	nfaWalkCount uint64
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
// A single build (everything under one AddPattern) is single-threaded, so the
// buffers are never shared concurrently within a build. sync.Pool is used
// rather than a plain free list for two reasons: this is a package-level
// global, so independent matcher builds running in separate goroutines draw
// from it at once and need a safe hand-off; and sync.Pool drops its contents
// on GC, so the maps do not become permanent steady-state memory.
var closureBufferPool = sync.Pool{
	New: func() any { return newClosureBuffers() },
}

// epsilonClosure walks the automaton starting from the given state
// and precomputes the epsilon closure for every reachable faState.
func epsilonClosure(start *faState) {
	bufs := closureBufferPool.Get().(*closureBuffers)
	bufs.nfaWalkCount = 0
	// Take a fresh generation for this walk. closureForState bumps bufs.gen
	// for its own dedup phases, but it never touches walkGen, so the state
	// dedup in closureForNfa compares against a value that stays fixed for
	// the whole walk.
	bufs.gen++
	bufs.walkGen = bufs.gen
	closureForNfa(start, bufs)
	closureBufferPool.Put(bufs)
}

// closureForNfa dedups by faState identity, not table-share key: each state
// must be walked once. (Share-key dedup is unsafe here — distinct states can
// share a steps backing array yet have different epsilons, and the zero key
// collapses all no-byte tables; the dedup post-pass in closureForState
// re-checks fieldTransitions on collision, but the walk has no such guard.)
func closureForNfa(state *faState, bufs *closureBuffers) {
	if bufs.walkVisited[state] == bufs.walkGen {
		return
	}
	bufs.walkVisited[state] = bufs.walkGen
	if state.epsilonClosure != nil {
		// Closed by a previous epsilonClosure call: everything reachable from
		// here was built and closed then and is unchanged, so there is nothing
		// new to compute below it. This makes each add's walk incremental.
		// Pruning is safe because mergeFAs only ever creates new states as
		// ancestors of (never descendants of) already-closed states, so no
		// nil-closure state is reachable exclusively through a closed one.
		return
	}
	bufs.nfaWalkCount++
	closureForState(state, bufs)
	for _, s := range state.table.steps {
		if s != nil {
			closureForNfa(s, bufs)
		}
	}
	for _, eps := range state.table.epsilons {
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
		state.epsilonClosure = selfOnlyClosure
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

	// Self-only closure (no other reachable non-epsilon-only state): use the
	// shared sentinel instead of allocating a 1-element slice. closureList has
	// length 1 exactly when only `self` was collected — but only when self is
	// non-epsilon-only (so it was added to closureList). Epsilon-only states
	// are not added to closureList, so length 1 there means a single other
	// state was found (not self), and we must not conflate that with self-only.
	if !state.table.isEpsilonOnly() && len(bufs.closureList) == 1 {
		state.epsilonClosure = selfOnlyClosure
		return
	}

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
	if !state.table.isEpsilonOnly() && len(closure) == 1 {
		// dedup collapsed everything into self (self was the sole surviving
		// representative); use the sentinel. Guard: epsilon-only states are
		// not self-added to closureList, so a singleton closure there means
		// one other state survived, not self.
		state.epsilonClosure = selfOnlyClosure
		return
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
