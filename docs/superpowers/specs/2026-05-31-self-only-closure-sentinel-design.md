# Self-only epsilon-closure sentinel

Date: 2026-05-31
Branch: embed-smalltable-research
Status: approved design, pre-implementation

## Problem

After Phase 1 (incremental closure walk pruning, 3× build throughput), the
residual build cost is closure *materialization* in `closureForState`: the
per-state `make([]*faState, …)` + `append`, plus the GC churn from those
allocations. Measured closure-size distribution (shellstyle, N=2000):

| metric | value |
|---|---|
| states | 14,573 |
| total closure mass | 98,419 |
| avg / p50 / p90 / max | 6.8 / 1 / 2 / 1,236 |

The distribution is wildly skewed: **most states' closure is exactly `{self}`**
(p50 = 1), yet each currently allocates a 1-element `[]*faState{state}` slice. A
few hub states carry almost all the mass (max 1,236).

A prior spike that dropped `self` from *all* closures and processed it
explicitly at match time **hung** `TestToxicStack` / `TestRegexpValidity`:
pulling `self` out of a multi-member closure defeats the table-pointer dedup for
`self` (it shares a `smallTable` with closure members), reintroducing the
next-state duplication that dedup prevents → exponential blowup on adversarial
regexps. Lesson: `self` must stay inside the dedup **when there are other
members**. But a self-only closure has nothing to dedup against, so that case is
safe to specialize.

## Goal

Specialize the `{self}` case to avoid its allocation (and resident backing array
+ GC scan) for the majority of states, **without** changing multi-member
closures (so the dedup stays intact and the spike's hang cannot recur). Primary
win: fewer allocations and lower resident memory for the common case. Build-curve
flattening is a measured outcome, not a guarantee (the few huge hub closures are
unaffected).

## Representation

`epsilonClosure []*faState` carries three meanings via length:

- **`nil`** — not yet computed. Unchanged; the Phase-1 prune (`closureForNfa`)
  and the `closureForState` memoization both key on `!= nil`.
- **non-nil, `len == 0`** — "self-only" (`{self}`). Stored as a shared
  package-level empty slice `var selfOnlyClosure = []*faState{}` — non-nil,
  zero-allocation (Go empty composite literal points at runtime zerobase).
- **`len >= 2`** — an explicit closure that **includes self**, deduped, exactly
  as today.

`len == 1` is never stored: a closure that reduces to `[self]` collapses to
`selfOnlyClosure`. So `len == 0` ⇔ self-only is an unambiguous discriminator.

## Build changes (`epsi_closure.go`, `closureForState`)

1. Fast path (`len(state.table.epsilons) == 0`): assign `selfOnlyClosure`
   instead of `[]*faState{state}`.
2. Main path: build `closureList` (self + `traverseEpsilons`) as now. If after
   traversal `len(closureList) == 1` (just self), assign `selfOnlyClosure` and
   skip the `make`/dedup post-pass. Otherwise run the existing dedup post-pass
   and store the full slice (which includes self). If the dedup post-pass
   reduces the result to a single element (`[self]`), assign `selfOnlyClosure`
   rather than the 1-element slice.

`nfa.go` `nfa2Dfa` start init (`nfaStart.epsilonClosure = []*faState{nfaStart}`)
→ `selfOnlyClosure` (dead path, kept consistent).

## Consumer changes (the discriminator branch)

All readers branch on `len(ec) == 0`:

- `traverseNFA` (`nfa.go`), both the per-byte loop and the end-of-input loop:
  ```
  for _, state := range currentStates {
      if len(state.epsilonClosure) == 0 {
          // self-only: process self (its fieldTransitions, and its step in the per-byte loop)
      } else {
          for _, ec := range state.epsilonClosure { /* process ec — includes self */ }
      }
  }
  ```
  **Multi-member closures are iterated whole; `self` is never processed
  separately for them.** This is the invariant that keeps the dedup effective
  and prevents the spike's blowup.
- `n2dNode` (`nfa.go`, dead `nfa2Dfa` path, must still be correct):
  `if len(ec) == 0 { nStates = append(nStates, rawNState) } else { nStates = append(nStates, ec...) }`.
- `memory_cost.go` `cmStateStats`: `len(epsilonClosure)` feeds the `fanouts`
  stat; self-only now contributes 0, so byte/fanout assertions in
  `memory_cost_test.go` (and the `[N in closure]` display in
  `prettyprinter_test.go`) are recalibrated.

## Correctness

- Self-only state: process self once. No other members ⇒ no duplication. Match
  result identical to iterating the old `[self]`.
- Multi-member state: iterate the same deduped set (incl self) the same way as
  today. No separate self-processing ⇒ no duplication ⇒ no hang.
- `nil` still means "not computed", so memoization and the Phase-1 prune are
  untouched.

## Validation

1. **Full suite green** (`go test ./...`), including **`TestToxicStack` and
   `TestRegexpValidity`** — these hung under the no-self spike and are the
   explicit gate proving this approach avoids that failure mode.
2. **Order-independence guard** (`TestIncrementalClosureOrderIndependence`).
3. Recalibrate the size/fanout assertions in `memory_cost_test.go` and
   `prettyprinter_test.go` to the new self-only=0 fanout contribution (and verify
   the new numbers by inspection, not by blindly pasting whatever the code prints).

## Measurement

- Closure-mass / allocation probe: confirm self-only states allocate no backing
  array; total closure-slice allocation count drops to roughly the multi-member
  state count.
- Build-scale probe (N = 250 → 2000) and the 10k research harness: record
  patterns/sec and the per-word curve (flattening is the hoped-for, unproven
  outcome).
- Resident-memory probe (the `GetMatcherStats`/heap measure used earlier):
  confirm a reduction from dropping the majority's 1-element backing arrays.
- Match benchmarks (`BenchmarkCityLots` + a workload subset): confirm the
  discriminator branch does not regress match throughput (expected flat-to-faster
  for self-only states, which now skip a 1-element slice iteration).

## Scope

In scope: the `selfOnlyClosure` sentinel + the `closureForState` build changes +
the `len==0` discriminator in `traverseNFA`/`n2dNode` + test recalibration. Out
of scope: inlining size-2 closures (Approach B), any sharing of multi-member or
hub closures (shown infeasible — closures are unique by self and self must stay
in the dedup), lazy/on-demand construction.
