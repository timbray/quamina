# Incremental Epsilon Closure (Walk Pruning) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `AddPattern`'s epsilon-closure pass incremental by pruning the closure walk at already-closed states, eliminating the O(N²) full-NFA re-walk on every add.

**Architecture:** `closureForNfa` gains one guard — if a state's `epsilonClosure` is already computed, don't recurse into it. Because merges create new states and reuse closed subtrees wholesale, the new (nil-closure) states form a connected region at the root bounded by closed subtrees, so the pruned walk visits exactly the states this add created. Behavior is unchanged; only the walk is smaller.

**Tech Stack:** Go, package `quamina`, files `epsi_closure.go` / `nfa.go`. Spec: `docs/superpowers/specs/2026-05-31-incremental-epsilon-closure-design.md`.

---

## File Structure

- `epsi_closure.go` — modify `closureForNfa` (the prune guard) and `closureBuffers` (add a test-observability walk counter).
- `incremental_closure_test.go` — **create**; holds both the order-independence correctness guard and the walk-prune red-green test.
- No match-time files change. No public API changes.

---

### Task 1: Verify the immutability invariant (correctness gate)

The prune is only safe if no code path mutates an **existing** state's `table.epsilons` / `table.steps` after that state's closure was computed. This task confirms that. **It is a gate: if a mutation is found, STOP and revisit the design — do not proceed to Task 3.**

**Files:** none modified (investigation only).

- [ ] **Step 1: Grep for epsilon/step/ceiling mutation in build code**

Run:
```bash
grep -nE '\.(epsilons|steps|ceilings)\s*=' nfa.go value_matcher.go shell_style.go wildcard.go regexp_nfa.go small_table.go
grep -nE 'append\([^,]*\.(epsilons|steps)' nfa.go value_matcher.go shell_style.go wildcard.go regexp_nfa.go small_table.go
```

- [ ] **Step 2: Classify each hit**

For every hit, confirm the left-hand side is a **freshly-created** state/table (e.g. `combined := &faState{table: newSmallTable()}`, or a `*smallTable` returned by a `make…FA` constructor before any closure runs), not a state that could already have a computed `epsilonClosure`. Pay specific attention to:
- `mergeFAStates` (the `combined.table.epsilons = simplifySplices(...)` and `combined.table.steps = steps` lines — `combined` is new ✓)
- `symmetricSpinnerMerge` / `asymmetricSpinnerMerge` (confirm they assign to a new `combined`, never to `spinner`/`nonSpinner`)
- `simplifySplices` (confirm it returns a new slice and does not assign into an input state)

- [ ] **Step 3: Record the finding**

Append a short note to the spec file under a new `## Implementation note: immutability verification` heading stating which functions assign epsilons/steps and that each target is a newly-created state (or, if a counterexample is found, STOP).

Run:
```bash
git add docs/superpowers/specs/2026-05-31-incremental-epsilon-closure-design.md
git commit -m "docs: record epsilon/step immutability verification for closure prune"
```

---

### Task 2: Order-independence correctness guard

A correct matcher's behavior is independent of `AddPattern` order. A closure-prune bug (a stale or skipped closure) would almost certainly surface as order-dependent match results. This test passes on the current (un-pruned) code and must keep passing after the prune — it is the safety net for Task 3.

**Files:**
- Create: `incremental_closure_test.go`

- [ ] **Step 1: Write the order-independence test**

```go
package quamina

import (
	"reflect"
	"sort"
	"testing"
)

// Patterns chosen to maximize merge interleaving on a single field: shared
// prefixes/suffixes, multiple stars, and an exact value.
var incrClosurePatterns = []struct{ x, p string }{
	{"p1", `{"x":[{"shellstyle":"*foo*"}]}`},
	{"p2", `{"x":[{"shellstyle":"*foobar*"}]}`},
	{"p3", `{"x":[{"shellstyle":"foo*"}]}`},
	{"p4", `{"x":[{"shellstyle":"*bar"}]}`},
	{"p5", `{"x":[{"shellstyle":"a*b*c"}]}`},
	{"p6", `{"x":[{"shellstyle":"*x*y*"}]}`},
	{"p7", `{"x":["foobar"]}`},
}

var incrClosureEvents = []string{
	`{"x":"foobar"}`, `{"x":"afoob"}`, `{"x":"foo"}`, `{"x":"xxbar"}`,
	`{"x":"abc"}`, `{"x":"axbyc"}`, `{"x":"nomatch"}`, `{"x":"foobarbaz"}`,
	`{"x":"axxbyyc"}`, `{"x":"bar"}`,
}

func buildAndMatch(t *testing.T, order []int) map[string][]string {
	t.Helper()
	q, _ := New()
	for _, i := range order {
		if err := q.AddPattern(incrClosurePatterns[i].x, incrClosurePatterns[i].p); err != nil {
			t.Fatalf("AddPattern %s: %v", incrClosurePatterns[i].x, err)
		}
	}
	out := make(map[string][]string, len(incrClosureEvents))
	for _, ev := range incrClosureEvents {
		matches, err := q.MatchesForEvent([]byte(ev))
		if err != nil {
			t.Fatalf("MatchesForEvent %s: %v", ev, err)
		}
		ss := make([]string, 0, len(matches))
		for _, x := range matches {
			ss = append(ss, x.(string))
		}
		sort.Strings(ss)
		out[ev] = ss
	}
	return out
}

func TestIncrementalClosureOrderIndependence(t *testing.T) {
	forward := []int{0, 1, 2, 3, 4, 5, 6}
	reverse := []int{6, 5, 4, 3, 2, 1, 0}
	shuffled := []int{3, 0, 6, 1, 5, 2, 4}

	base := buildAndMatch(t, forward)
	if got := buildAndMatch(t, reverse); !reflect.DeepEqual(base, got) {
		t.Errorf("reverse-order matches differ from forward:\nforward=%v\nreverse=%v", base, got)
	}
	if got := buildAndMatch(t, shuffled); !reflect.DeepEqual(base, got) {
		t.Errorf("shuffled-order matches differ from forward:\nforward=%v\nshuffled=%v", base, got)
	}

	// Sanity: the exact-value event must at least match p7 and the wildcards.
	if len(base[`{"x":"foobar"}`]) == 0 {
		t.Errorf(`expected matches for {"x":"foobar"}, got none`)
	}
}
```

- [ ] **Step 2: Run on current code — confirm it PASSES**

Run: `go test -run TestIncrementalClosureOrderIndependence -v .`
Expected: PASS. (If it fails on the current code, the test is wrong or there is a pre-existing bug — fix before proceeding.)

- [ ] **Step 3: Commit**

```bash
git add incremental_closure_test.go
git commit -m "test: order-independence guard for incremental epsilon closure"
```

---

### Task 3: Walk counter + prune (red-green)

**Files:**
- Modify: `epsi_closure.go` (`closureBuffers` struct, `closureForNfa`)
- Modify: `incremental_closure_test.go` (add the walk-prune test)

- [ ] **Step 1: Add a walk counter to `closureBuffers`**

In `epsi_closure.go`, add the field to the `closureBuffers` struct (alongside `walkVisited`):

```go
	walkVisited   map[*faState]uint64       // per-faState last-walked gen, used by closureForNfa
	// nfaWalkCount counts states actually processed by closureForNfa (past the
	// prune + dedup guards). Used by tests to assert the walk is incremental;
	// negligible in production and never read there.
	nfaWalkCount uint64
```

- [ ] **Step 2: Increment the counter in `closureForNfa` (no prune yet)**

In `epsi_closure.go`, add the increment right after the `walkVisited` mark, leaving everything else as-is for now:

```go
func closureForNfa(state *faState, bufs *closureBuffers) {
	if bufs.walkVisited[state] == bufs.walkGen {
		return
	}
	bufs.walkVisited[state] = bufs.walkGen
	bufs.nfaWalkCount++
	// ... rest unchanged ...
```

- [ ] **Step 3: Write the walk-prune test**

Append to `incremental_closure_test.go`:

```go
// TestClosureWalkPrunesClosedSubtree builds and fully closes an NFA, then
// re-runs the closure walk (as epsilonClosure does) with an inspectable buffer.
// A fully-closed NFA must be walked in zero state-visits: every state should be
// pruned because its epsilonClosure is already computed.
func TestClosureWalkPrunesClosedSubtree(t *testing.T) {
	pp := newPrettyPrinter(99)
	nfa, _ := makeShellStyleFA([]byte(`"*foo*bar*"`), pp)
	epsilonClosure(nfa) // fully close every reachable state

	// Mirror epsilonClosure's body, but with a buffer we can inspect.
	bufs := newClosureBuffers()
	bufs.gen++
	bufs.walkGen = bufs.gen
	closureForState(nfa, bufs)
	closureForNfa(nfa, bufs)

	if bufs.nfaWalkCount != 0 {
		t.Errorf("re-walking a fully-closed NFA should process 0 states, processed %d", bufs.nfaWalkCount)
	}
}
```

- [ ] **Step 4: Run the prune test — confirm it FAILS (red)**

Run: `go test -run TestClosureWalkPrunesClosedSubtree -v .`
Expected: FAIL — `processed N states` with N > 0, because the un-pruned walk re-traverses the whole closed NFA.

- [ ] **Step 5: Add the prune guard**

In `epsi_closure.go`, add the guard at the very top of `closureForNfa`:

```go
func closureForNfa(state *faState, bufs *closureBuffers) {
	if state.epsilonClosure != nil { // already-closed subtree: don't re-walk
		return
	}
	if bufs.walkVisited[state] == bufs.walkGen {
		return
	}
	bufs.walkVisited[state] = bufs.walkGen
	bufs.nfaWalkCount++
	// ... rest unchanged ...
```

- [ ] **Step 6: Run the prune test — confirm it PASSES (green)**

Run: `go test -run TestClosureWalkPrunesClosedSubtree -v .`
Expected: PASS (`nfaWalkCount == 0`).

- [ ] **Step 7: Run the correctness guard + full suite**

Run:
```bash
go test -run TestIncrementalClosureOrderIndependence -v .
go vet .
go test ./...
```
Expected: all PASS (`ok quamina.net/go/quamina/v2`).

- [ ] **Step 8: Commit**

```bash
git add epsi_closure.go incremental_closure_test.go
git commit -m "epsi_closure: prune closure walk at already-closed states

Makes AddPattern's epsilon closure incremental: closureForNfa no longer
re-walks already-closed subtrees, so each add visits only the states it
created instead of the whole growing NFA (was O(N^2))."
```

---

### Task 4: Measure the win (report, not pass/fail)

**Files:** throwaway probe (created then deleted).

- [ ] **Step 1: Record the build-scale curve**

Create `buildscale_probe_test.go`:

```go
package quamina

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestBuildScaleProbe(t *testing.T) {
	for _, n := range []int{250, 500, 1000, 2000} {
		words := readWWords(t, n)
		source := rand.NewSource(293591)
		starWords := make([]string, 0, len(words))
		patterns := make([]string, 0, len(words))
		for _, word := range words {
			//nolint:gosec
			starAt := source.Int63() % 6
			starWord := string(word[:starAt]) + "*" + string(word[starAt:])
			starWords = append(starWords, starWord)
			patterns = append(patterns, fmt.Sprintf(`{"x": [ {"shellstyle": "%s" } ] }`, starWord))
		}
		q, _ := New()
		start := time.Now()
		for i := range words {
			if err := q.AddPattern(starWords[i], patterns[i]); err != nil {
				t.Fatal(err)
			}
		}
		el := time.Since(start)
		t.Logf("N=%4d build=%8.2fms per-word=%6.1fus", len(words),
			float64(el.Microseconds())/1000.0, float64(el.Microseconds())/float64(len(words)))
	}
}
```

Run: `go test -run TestBuildScaleProbe -v .`
Expected: per-word build time should be roughly flat across N (the pre-change baseline climbed steeply: ~197µs→489µs from N=125→1000). Record the numbers.

- [ ] **Step 2: Run Tim's 10k harness and compare patterns/sec**

Run: `go run ./research`
Expected: `Patterns/sec` substantially higher than the pre-change baseline of ~133–140/sec.

- [ ] **Step 3: Re-profile to confirm walkVisited traffic is gone**

Run:
```bash
go build -o /tmp/researchbin ./research
/tmp/researchbin -cpuprofile=/tmp/research_after.prof
go tool pprof -top -cum -nodecount=20 /tmp/researchbin /tmp/research_after.prof
```
Expected: `closureForNfa` / `walkVisited` map lookups should no longer dominate; the build’s former O(N²) walk cost should be largely gone.

- [ ] **Step 4: Remove the throwaway probe**

Run:
```bash
rm -f buildscale_probe_test.go
```
(Do NOT commit the probe. The generated `research/*.csv` is also untracked — leave or delete as desired.)

- [ ] **Step 5: Report**

Summarize: build-scale curve before/after, patterns/sec before/after, and the profile delta. Then revisit the spec's Phase-2 gate (closure-storage sharing) using the new profile — decide if the residual `append`/GC cost is worth pursuing.

---

## Self-Review

- **Spec coverage:** prune guard (Task 3) ✓; immutability verification (Task 1) ✓; order-independence/differential validation (Task 2) ✓; full-suite validation (Task 3 Step 7) ✓; measurement + re-profile + Phase-2 gate (Task 4) ✓. `walkVisited` denser-structure explicitly dropped per spec — no task, correct.
- **Placeholder scan:** none — all steps carry concrete code/commands.
- **Type consistency:** `nfaWalkCount uint64` defined in Task 3 Step 1, incremented Step 2, read in the test Step 3 and Step 6; `closureForNfa`/`closureForState`/`newClosureBuffers`/`makeShellStyleFA`/`readWWords`/`newPrettyPrinter` are all existing symbols used consistently.
