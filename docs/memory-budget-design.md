# Memory Budget: Accounting Choices

Design notes for Quamina's `SetMemoryBudget` accountant, motivated by the
interaction between issue #518 (steady-state memory reduction) and the
planned lazy-DFA cache with aging eviction.

## The three candidate fields from `runtime.MemStats`

### `TotalAlloc`
Cumulative bytes allocated over the process lifetime. Monotonically
increasing; never shrinks. This is what `memory_cost.go` currently uses.

### `HeapAlloc`
Bytes occupied by live-plus-unreclaimed heap objects. Grows on every
allocation; shrinks the moment GC frees an object.

### `HeapInuse`
Bytes in heap spans that contain at least one object. Grows when a new
span is claimed; shrinks when a span becomes fully empty.

## `HeapAlloc` vs `HeapInuse` — span granularity

Go's allocator groups objects by size class into fixed-size spans
(typically 8 KB). A span is "in use" as long as any slot is occupied.

Example:
- Allocate one 16-byte object. Go carves a slot from a 16-byte-class span
  (~8192 bytes, ~512 slots).
- `HeapAlloc` = 16 bytes.
- `HeapInuse` = 8192 bytes.

As that span fills with more 16-byte objects, `HeapAlloc` grows
linearly while `HeapInuse` stays flat at 8192 until the span is full.

Inversely: allocate 512 objects, free 511 — `HeapInuse` still = 8192
(one live object keeps the span in-use), `HeapAlloc` drops to 16.

So `HeapInuse >= HeapAlloc` always, and the delta is **fragmentation
overhead** — committed slots currently unoccupied.

### Which to pick, by question
| Question the user is asking | Right field |
|---|---|
| "How much of my data is currently live?" | `HeapAlloc` |
| "How much memory is my process actually retaining from the OS?" | `HeapInuse` (closer) |
| "Will I OOM?" | `HeapInuse` or `HeapSys` |

## `TotalAlloc` vs a retained-memory field

| Axis | `TotalAlloc` | `HeapInuse` / `HeapAlloc` |
|---|---|---|
| Semantic | "memory churn" | "memory retained right now" |
| Matches user intent | No — reflects allocation traffic | Yes — reflects what's held |
| Transient allocations | Counted (penalizes side-tables, scratch buffers) | Ignored after GC |
| Peak-during-swap behavior | Counts old+new matcher both ✓ | Also counts both ✓ |
| Sample cost | `ReadMemStats` only, ~µs | Needs `runtime.GC()` to be accurate, ~ms on a large heap |
| Timing sensitivity | None | Stale without a recent GC |
| Parallel-goroutine app alloc | Pollutes the counter | Also pollutes the heap |
| Drift over process lifetime | Unbounded; eventually trips even without growth | Stable; reflects current state |

### Where the real difference lives
- `TotalAlloc` = "what have I allocated on the way to my current state"
  — blind to GC's success.
- `HeapInuse` / `HeapAlloc` = "where am I now" — which is what a
  `SetMemoryBudget(200 MB)` user is actually asking about.

### The `runtime.GC()` cost
A single GC can be 1–10 ms on a 100 MB heap; worse on bigger ones. Doing
it inside the sampling interval (every 100 pattern-adds) is probably
fine — one pattern-add on a big matcher is already multi-ms. Doing it on
every `check()` would be ruinous.

Compromise available: `TotalAlloc` as a cheap early warning, then
`runtime.GC() + HeapInuse` only when the cheap warning trips. Cheap
common-case, correct at the boundary.

## Redis-style lazy DFA cache changes the picture

The planned lazy-DFA cache builds DFA states on demand, caches them, and
ages out entries under pressure. That reframes the budget mechanism:

1. **The budget is enforced by the cache, not by rejecting writes.**
   Unlike pattern-add (where crossing the budget = error), the cache
   responds by evicting. Accounting just needs to stay roughly in sync
   with reality — occasional staleness is fine because the eviction loop
   will catch up on the next pressure tick.

2. **Explicit GC at sample time is probably wrong for the cache path.**
   Forcing `runtime.GC()` every N operations punishes a workload whose
   normal rhythm is "allocate small DFA node, evict least-recent." Let
   the runtime's GC run on its own schedule.

3. **A retained-memory reading without a forced GC is sufficient *if*
   it only needs to be approximately right.** For LRU-style eviction,
   "am I over? evict 10% and re-check" tolerates a lot of measurement
   lag.

4. **Explicit per-entry byte tracking is worth considering for the
   cache.** Redis gets clean numbers because it accounts allocations
   against `maxmemory` at the allocator layer — not via OS-level metrics.
   A per-entry byte count (sum sizes on insert, subtract on evict)
   gives deterministic numbers with zero dependency on `runtime.MemStats`
   or GC timing. A `runtime.MemStats` reading would still make sense
   for the *build* path (unknown alloc shape), but the cache's budget
   could be tracked intrinsically.

5. **Two budgets, one ceiling.** If the user sets
   `SetMemoryBudget(200M)`, the natural split is: NFA construction uses
   up to `budget − current_retained`; the DFA cache's ceiling is
   "whatever's left after the NFA stops growing." Both paths compare
   against the same retained-memory ceiling but answer differently —
   NFA rejects on overflow, cache evicts.

6. **Eviction lag.** When an LRU entry is evicted, its bytes don't
   leave `HeapAlloc` (or `HeapInuse`) until a GC cycle marks and sweeps.
   So if you evict in a tight loop measuring either, you may need to
   yield to let at least one sweep happen between eviction and the
   next measurement. Pragmatic: evict batches, not singletons.

## Recommended direction (revised after experiment)

Initial direction was `HeapInuse`, but running the experiment on
Quamina's workload revealed a problem: Quamina allocates large numbers
of small objects (`*faState`, `*smallTable`, small slices), and in a
warm process these allocations drop into slots of already-claimed size-
class spans without claiming new spans. HeapInuse therefore stays flat
across many real allocations, and `currentMemory` can fail to grow
even as the matcher grows significantly. Budget enforcement ends up
under-sampling retention on the very workload it's meant to bound.

**Landing on `HeapAlloc` instead.** Byte-precise, drops on GC, but
doesn't round to span boundaries. Tradeoff: it omits span-overhead
("fragmentation") from the budget, so the user's ceiling is "bytes my
live objects occupy," not "bytes my process is holding in spans." For a
workload that's mostly many-small-objects, those two numbers are close;
for a small-number-of-big-objects workload, `HeapInuse` would be more
honest, but Quamina isn't that workload.

- Switch `bytesAllocated()` from `TotalAlloc` to `HeapAlloc`.
- No forced `runtime.GC()` — occasional lag from deferred collection
  is tolerable for a budget heuristic.
- For the DFA cache: track per-entry bytes explicitly so eviction
  decisions don't depend on GC timing. The `HeapAlloc` reading becomes
  a fleet-level ceiling, while the cache's own counter drives eviction.
- `SetMemoryBudget` reflects live-object retention. The
  build-context-extract refactor (which traded permanent per-struct
  bloat for transient build-time maps) gets fair credit instead of
  being penalized by `TotalAlloc`'s cumulative view.

## Why not `HeapInuse`

For Quamina's workload — where old and new matchers coexist during the
atomic swap, and the cache plus NFA together approach a process-level
ceiling — the user cares about retained memory *including* fragmentation,
because that's what drives RSS and the OOM killer. `HeapAlloc` omits
the span overhead.

In the experiment, though, the span-rounding of `HeapInuse` proved to
be a bigger problem than its slightly-more-honest accounting of
retained memory: many small allocations slip in under the radar and
`currentMemory` stays flat. A real matcher growing in span-sized
increments would still work fine under `HeapInuse`, but the tests
flake and the user-facing budget under-counts common workloads. Until
the allocation pattern changes (e.g. if large slabs of smallTables
start getting allocated together), `HeapAlloc` wins on balance.

`HeapInuse` remains the right choice if the goal were truly "RSS /
OOM-killer proxy" and the workload had large-enough allocations that
span granularity was noise. For a budget feature whose
purpose is to prevent OOM in a cache-migrating system, `HeapInuse` is a
better fit.
