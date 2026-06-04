# Incremental epsilon closure via walk pruning

Date: 2026-05-31
Branch: embed-smalltable-research
Status: approved design, pre-implementation

## Problem

`AddPattern` on a nondeterministic value matcher calls
`epsilonClosure(fields.startState)` after each merge (`value_matcher.go`).
`epsilonClosure` walks the *entire* reachable NFA via `closureForNfa`, calling
`closureForState` on every state. `closureForState` already memoizes
(`if state.epsilonClosure != nil { return }`), so closures are not recomputed —
but the **walk itself re-traverses the whole, ever-growing NFA on every add**.

Over N adds that is O(N²). Profiling Tim's research harness (10k shellstyle
patterns) on this branch attributed the dominant build cost to exactly this:

- `closureForNfa` L71 `bufs.walkVisited[state] == walkGen` (the walk-dedup map
  lookup, one per edge per walk): ~12.2 s
- `closureForNfa` L74 walkVisited assign: ~3.1 s
- `closureForState` L141 closure `append` (materializing result slices): ~12.1 s
- plus ~30% of the run in GC (`madvise` 22 s) from the allocation churn.

Goal: kill the O(N²) build walk while keeping the eager-closure architecture
(closures are consumed on the match hot path in `traverseNFA`, so we are NOT
moving to lazy/on-demand construction — that is the separate `lazy_dfa` line of
work). Must not change match-time behavior or regress the branch's ~10% resident
memory win.

## The change

Add one guard at the top of `closureForNfa`:

```go
func closureForNfa(state *faState, bufs *closureBuffers) {
    if state.epsilonClosure != nil { // already-closed subtree: don't re-walk
        return
    }
    if bufs.walkVisited[state] == bufs.walkGen {
        return
    }
    bufs.walkVisited[state] = bufs.walkGen
    // ... unchanged: recurse over steps and epsilons ...
}
```

`epsilonClosure(start)` is still called after each merge, but the walk now
descends only through nil-closure (new) states and stops at the closed
boundary — visiting exactly the states this `AddPattern` created instead of the
whole NFA. The build walk drops from O(N²) toward O(total new states) = ~O(N).

## Correctness

**Invariant (load-bearing):** No new (nil-closure) state reachable from `start`
is reachable *only* through an already-closed state. Equivalently, the
nil-closure states form a connected region anchored at the new root, bounded by
reused (closed) subtrees.

**Why it holds:**

1. A state's `epsilonClosure` is set exactly once, and a closed state's `table`
   (`steps` + `epsilons`) is never mutated afterward. Merges create **new**
   `combined` states (`mergeFAStates`: `combined = &faState{table: newSmallTable()}`)
   and only *read* the input states; they never append to an existing state's
   epsilons/steps.
2. `mergeFAStates` reuses a subtree **wholesale** (`merged = next1` when one side
   is nil or both agree) or creates a new `combined` state. It never grafts a new
   state *underneath* a reused one — a reused state's children are its original
   (closed) children.

From (1) and (2): a reused-closed subtree is entirely unchanged and entirely
closed, so pruning the walk at it skips nothing that needs computation. A new
state can never sit behind a closed state, so the prune never skips a state that
still needs a closure. A reused state's closure, computed in a prior call over an
unchanged subtree, remains valid.

**The single failure mode** (and the first implementation step): any code path
that mutates an *existing* state's `table.epsilons`/`table.steps` after its
closure was computed would leave a stale closure that the prune would never
recompute → missed matches. Before trusting the prune, grep the build code
(including `symmetricSpinnerMerge` / `asymmetricSpinnerMerge` and
`simplifySplices`) and confirm no post-creation mutation of an existing state's
epsilons/steps exists. If any is found, this design must be revisited.

**Non-issues, confirmed:**
- First add: `start` is new, nothing is closed, so no pruning occurs and behavior
  is identical to today.
- Splice states: `simplifySplices` returns existing state pointers as epsilon
  targets and does not create states; new splice states are themselves
  nil-closure and walked normally, reaching any new epsilon targets.
- `fieldTransitions` appends in merges do not affect closures (closures are
  epsilon-reachability over `table.epsilons`; field transitions are read fresh at
  match time).

## Validation

1. **Full test suite** — `go test ./...` (heavy shellstyle/wildcard/regexp match
   coverage). Must stay green.
2. **Adversarial differential test (new):** build a matcher by adding a set of
   overlapping shellstyle + regexp patterns incrementally; build a second matcher
   with the same patterns; assert identical match results across a battery of
   matching and non-matching events. This directly catches a missed-closure
   regression that unit coverage might miss. Include patterns chosen to maximize
   merge interleaving (shared prefixes/suffixes, multiple stars).

## Measurement

- Re-run the build-scale probe (N = 250 → 1000) and Tim's 10k research harness;
  expect per-pattern build time to flatten (no O(N²) climb) and overall
  patterns/sec to rise substantially.
- Re-profile (`research -cpuprofile`); the `walkVisited` map lookups/assigns
  should largely vanish from the top.

## Phase-2 gate (#3, deferred — not part of this change)

After Phase 1 lands and is re-profiled, decide whether the residual closure
materialization cost (the `append` + GC churn) justifies closure storage sharing.
Note the caveat: each state's closure includes itself, so full-slice interning is
expected to buy little; any benefit would require suffix/structural sharing.
Driven by measured closure overlap, not assumption. The "denser visited
structure" idea is **dropped**: pruning shrinks `walkVisited` to the new region,
so it is no longer a hot path.

## Scope

In scope: `closureForNfa` prune + the immutability verification + the
differential test. Out of scope: lazy/on-demand construction, closure storage
sharing (Phase-2 gate), any change to the match-time traversal.

## Implementation note: immutability verification

Verified (2026-05-31) that the load-bearing invariant holds. Every assignment to
`.epsilons` / `.steps` / `.ceilings` in the build code targets a **freshly
created** state or table, never one that could already carry a computed
`epsilonClosure`:

- `nfa.go`: `mergeFAStates` L411/458/459 (`combined`), `asymmetricSpinnerMerge`
  L506/526/527, `symmetricSpinnerMerge` L583/601/602 — all target the new
  `combined`/`mergedState` returned by `mergeFAStates` (`&faState{table:
  newSmallTable()}`). The spinner appends at L506/L583 append onto that fresh
  merge result, not onto an input state.
- `regexp_nfa.go` L88/97/110/119/124/144, `shell_style.go` L63, `wildcard.go`
  L99 — each targets a state allocated 1–2 lines earlier in the same FA
  constructor.
- `small_table.go` `makeSmallTable`/`pack` — operate on a local table being
  built (and `pack` is only on the unused `nfa2Dfa` path).

Structural reason: `epsilonClosure()` is called only in `value_matcher.go`
`addTransition` (L167/187/195), strictly *after* `mergeFAs` has fully returned
and the new FA tree is built; `keyMemo` is scoped to one `mergeFAs` call, so even
memoized `combined` states have `epsilonClosure == nil` when they receive an
epsilon append. The prune is therefore safe.
