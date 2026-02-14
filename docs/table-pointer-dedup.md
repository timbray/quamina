# Table-pointer dedup in epsilon closure

## The problem

In `regexp_nfa.go`, the `+` case of `faFromQuantifiedAtom` (line 80-87) creates
a new `faState` wrapping a `*smallTable` that already belongs to an inner state:

```go
case atom.isPlus():
    plusLoopback := &faState{table: newSmallTable()}
    state = &faState{table: atom.makeFA(plusLoopback, pp)}  // wraps same table
    plusLoopback.table.epsilons = []*faState{nextState, state}
```

`atom.makeFA()` returns a `*smallTable`, not an `*faState`. Every path through
it creates a table that internally belongs to some `faState` built deeper in the
call chain (e.g. `makeNFAFromBranches` -> `faFromBranch` -> `faFromQuantifiedAtom`
creates inner states). But `makeFA` discards those states and returns only the
table.

Back in the `+` case, line 84 wraps the returned table in a **new** `faState`.
That table already belongs to the inner `faState` created during `makeFA`. Now
two distinct `faState` pointers reference the same `*smallTable` -- and since
epsilons live on the `*smallTable` (not on `*faState`), both states have
identical byte transitions AND identical epsilon transitions.

During epsilon closure computation, both states get added to the closure (they're
different pointers). During NFA traversal, both states step the same byte through
the same table, producing duplicate destinations. These compound exponentially
across steps.

## The old fix: runtime dedup in traverseNFA

The original code let duplicate states into the closures and tried to clean them
up during matching:

```go
if len(nextStates) > 500 {
    slices.SortFunc(nextStates, func(a, b *faState) int {
        return cmp.Compare(uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b)))
    })
    nextStates = slices.Compact(nextStates)
}
```

This was reactive and lossy. It only kicked in after `nextStates` grew past a
threshold. Below that threshold, duplicates silently compounded -- two duplicates
at step N become four at step N+1, eight at N+2. The sort+compact dedup trimmed
the list back down, but the exponential blowup had already consumed CPU doing
redundant `step()` calls on identical tables. And every threshold is arbitrary --
too high wastes work, too low adds overhead on inputs that don't need it.

## The new fix: table-pointer dedup in epsilon closure

The new approach is structural. Since two `faState` pointers sharing the same
`*smallTable` are functionally identical (same byte transitions, same epsilon
transitions), we keep only one representative per table during epsilon closure
computation.

### Implementation

A global generation counter (`tableSeenGeneration`) is incremented each time
`closureForStateWithBufs` is called. Each `smallTable` has a `tableSeenGen`
field. When `traverseEpsilons` encounters a non-epsilon-only state:

1. If `eps.table.tableSeenGen == bufs.tableGen`, the table was already seen in
   this closure computation -- skip it entirely (don't add to closure, don't
   recurse into its epsilons, since they're identical to the first state's).
2. Otherwise, add the state to the closure and stamp the table.

This eliminates duplicates at the source. `traverseNFA` gets clean, minimal
closure lists with no duplicates, so it does no redundant work and needs no
cleanup pass.

### Why a generation counter instead of a map

The first implementation used a `map[*smallTable]bool` in `closureBuffers`. This
worked correctly but added per-closure overhead from map operations (clear, lookup,
insert). Replacing it with a generation counter on `smallTable` makes the check a
single integer comparison -- effectively free.

### Files changed

- `small_table.go` -- added `tableSeenGen uint64` field to `smallTable`
- `epsi_closure.go` -- added `tableSeenGeneration` global counter, `tableGen`
  field in `closureBuffers`, generation-based dedup in `traverseEpsilons`
- `nfa.go` -- removed `seen map[*faState]uint64`, `stepGen uint64`, `getSeen()`
  method, and the runtime dedup block from `traverseNFA`

## transmap activeSet optimization

During profiling, the shellstyle benchmark showed a regression from the
transmap push/pop refactor. The `add()` method was doing
`tm.levels[tm.depth].set` on every call -- a slice index plus struct field
dereference on the hottest path in NFA traversal. Caching the active set pointer
in `transmap.activeSet` on `push()` eliminated this indirection and recovered
the regression.

## Benchmark results

| Benchmark | main | dedup_fix | delta |
|---|---|---|---|
| CityLots | ~6,294 ns, 347 B, 31 allocs | ~5,798 ns, 14 B, 0 allocs | -7.9% ns, -96% B, -100% allocs |
| Shellstyle | ~32,558 ns, 312 B, 31 allocs | ~32,507 ns, 0 B, 0 allocs | flat ns, -100% B, -100% allocs |
| 8259 | ~5,837 ns, 108 B, 9 allocs | ~5,621 ns, 0 B, 0 allocs | -3.7% ns, -100% B, -100% allocs |
