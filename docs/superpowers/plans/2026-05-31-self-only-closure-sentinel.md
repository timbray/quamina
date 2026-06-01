# Self-Only Closure Sentinel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Store the common `{self}` epsilon closure as a zero-allocation, non-nil sentinel (empty slice) so the majority of states allocate no closure backing array, without changing multi-member closures.

**Architecture:** `epsilonClosure` length now discriminates: `nil` = not computed; non-nil `len 0` = self-only; `len >= 2` = explicit closure including self. Consumers branch on `len == 0` to process `self` directly; multi-member closures are iterated whole (self stays inside the table-pointer dedup, so the prior no-self spike's adversarial blowup cannot recur).

**Tech Stack:** Go, package `quamina`, files `epsi_closure.go` / `nfa.go`. Spec: `docs/superpowers/specs/2026-05-31-self-only-closure-sentinel-design.md`.

---

## File Structure

- `nfa.go` — the `len==0` discriminator in `traverseNFA` (two sites) and `n2dNode`; `nfa2Dfa` start init. (Modified.)
- `epsi_closure.go` — `selfOnlyClosure` package var + `closureForState` self-only handling. (Modified.)
- `incremental_closure_test.go` — add the white-box sentinel red-green test. (Modified.)
- `epsi_closure_test.go`, `value_matcher_test.go`, `memory_cost_test.go` — recalibrate the assertions that read `len(epsilonClosure)`/closure-derived byte counts. (Modified.)

No new files; no public API change.

---

### Task 1: Consumer discriminator (behavior-preserving prep)

Add the `len(ec)==0 → process self` branch to every `epsilonClosure` consumer. This is a pure refactor: until Task 2 introduces the sentinel, no stored closure has `len 0` (the minimum is `[self]`, len 1), so the new branch is never taken and behavior is identical. Landing it first keeps the suite green and isolates the representation flip in Task 2.

**Files:**
- Modify: `nfa.go` (`traverseNFA` two loops, `n2dNode`)

- [ ] **Step 1: Update the `traverseNFA` per-byte loop**

Replace
```go
		for _, state := range currentStates {
			for _, ecState := range state.epsilonClosure {
				for _, fm := range ecState.fieldTransitions {
					fieldSet[fm] = true
				}
				if nextStep := ecState.table.step(utf8Byte); nextStep != nil {
					nextStates = append(nextStates, nextStep)
				}
			}
		}
```
with
```go
		for _, state := range currentStates {
			if len(state.epsilonClosure) == 0 {
				// self-only closure: process the state itself
				for _, fm := range state.fieldTransitions {
					fieldSet[fm] = true
				}
				if nextStep := state.table.step(utf8Byte); nextStep != nil {
					nextStates = append(nextStates, nextStep)
				}
				continue
			}
			for _, ecState := range state.epsilonClosure {
				for _, fm := range ecState.fieldTransitions {
					fieldSet[fm] = true
				}
				if nextStep := ecState.table.step(utf8Byte); nextStep != nil {
					nextStates = append(nextStates, nextStep)
				}
			}
		}
```

- [ ] **Step 2: Update the `traverseNFA` end-of-input loop**

Replace
```go
	for _, state := range currentStates {
		for _, ecState := range state.epsilonClosure {
			for _, fm := range ecState.fieldTransitions {
				fieldSet[fm] = true
			}
		}
	}
```
with
```go
	for _, state := range currentStates {
		if len(state.epsilonClosure) == 0 {
			for _, fm := range state.fieldTransitions {
				fieldSet[fm] = true
			}
			continue
		}
		for _, ecState := range state.epsilonClosure {
			for _, fm := range ecState.fieldTransitions {
				fieldSet[fm] = true
			}
		}
	}
```

- [ ] **Step 3: Update `n2dNode`**

Replace
```go
	for _, rawNState := range rawNStates {
		nStates = append(nStates, rawNState.epsilonClosure...) // a state's closure includes itself
	}
```
with
```go
	for _, rawNState := range rawNStates {
		if len(rawNState.epsilonClosure) == 0 {
			nStates = append(nStates, rawNState) // self-only closure: self is implicit
		} else {
			nStates = append(nStates, rawNState.epsilonClosure...) // includes self
		}
	}
```

- [ ] **Step 4: Verify behavior is unchanged**

Run: `go vet . && go test ./...`
Expected: all PASS (`ok quamina.net/go/quamina/v2`). No assertion changes yet because no closure has `len 0` until Task 2.

- [ ] **Step 5: Commit**

```bash
git add nfa.go
git commit -m "nfa: add len==0 self-only discriminator to closure consumers

Behavior-preserving prep: until the self-only sentinel exists, no stored
closure has length 0, so the new branch is never taken."
```

---

### Task 2: Self-only sentinel + recalibrate size assertions

Introduce the sentinel in `closureForState`, flipping self-only states to a zero-length non-nil closure. This makes the Task-1 branches live. The only tests that must change are those reading `len(epsilonClosure)` or closure-derived byte counts; matching tests, `TestTablePointerDedup`, `TestToxicStack`, and `TestRegexpValidity` must stay green (proving behavior is preserved and the no-self hang cannot recur).

**Files:**
- Modify: `epsi_closure.go` (package var + `closureForState`), `nfa.go` (`nfa2Dfa` start)
- Test: `incremental_closure_test.go` (new red-green test)
- Recalibrate: `epsi_closure_test.go`, `value_matcher_test.go`, `memory_cost_test.go`

- [ ] **Step 1: Write the failing sentinel test**

Append to `incremental_closure_test.go`:
```go
// TestSelfOnlyClosureSentinel: a state with no epsilon transitions has closure
// {self}, which must be stored as the non-nil, zero-length self-only sentinel
// (not a 1-element slice). nil still means "not computed".
func TestSelfOnlyClosureSentinel(t *testing.T) {
	s := &faState{table: *newSmallTable()} // no epsilons
	closureForStateNoBufs(s)
	if s.epsilonClosure == nil {
		t.Fatal("self-only closure must be non-nil (nil means 'not computed')")
	}
	if len(s.epsilonClosure) != 0 {
		t.Errorf("self-only closure: want len-0 sentinel, got len %d", len(s.epsilonClosure))
	}
}
```

- [ ] **Step 2: Run it; verify it FAILS**

Run: `go test -run TestSelfOnlyClosureSentinel -v .`
Expected: FAIL — `want len-0 sentinel, got len 1` (the current fast path stores `[]*faState{state}`).

- [ ] **Step 3: Add the `selfOnlyClosure` package var**

In `epsi_closure.go`, just below the `import` line, add:
```go
// selfOnlyClosure is the shared sentinel for a closure equal to {self}: non-nil
// (so it is distinct from nil = "not computed") and zero-length (so consumers
// take their len==0 self-only branch). The empty composite literal points at
// runtime zerobase, so this allocates nothing.
var selfOnlyClosure = []*faState{}
```

- [ ] **Step 4: Use the sentinel in `closureForState`**

In `closureForState`, change the fast path
```go
	if len(state.table.epsilons) == 0 {
		state.epsilonClosure = []*faState{state}
		return
	}
```
to
```go
	if len(state.table.epsilons) == 0 {
		state.epsilonClosure = selfOnlyClosure
		return
	}
```
Then, in the same function, replace the self-append block and final assignment. Change
```go
	bufs.closureList = bufs.closureList[:0]
	if !state.table.isEpsilonOnly() {
		bufs.states[state] = bufs.closureSetGen
		bufs.closureList = append(bufs.closureList, state)
	}
	traverseEpsilons(state, state.table.epsilons, bufs)
```
to
```go
	bufs.closureList = bufs.closureList[:0]
	if !state.table.isEpsilonOnly() {
		bufs.states[state] = bufs.closureSetGen
		bufs.closureList = append(bufs.closureList, state)
	}
	traverseEpsilons(state, state.table.epsilons, bufs)

	// Self-only closure (no other reachable non-epsilon-only states): use the
	// shared sentinel instead of allocating a 1-element slice.
	if len(bufs.closureList) == 1 {
		state.epsilonClosure = selfOnlyClosure
		return
	}
```
(The existing dedup post-pass and `state.epsilonClosure = closure` remain below, for the multi-member case. Note: `closureList` has length 1 exactly when only `self` was collected, since `self` is appended first and is the only non-epsilon-only state in its own closure.)

- [ ] **Step 5: Collapse a dedup-to-self result to the sentinel**

The dedup post-pass can reduce a multi-member `closureList` to just `[self]` (when the other members share self's table and field transitions). Guard the final assignment. Change
```go
	state.epsilonClosure = closure
```
to
```go
	if len(closure) == 1 {
		// dedup collapsed everything into self; use the sentinel
		state.epsilonClosure = selfOnlyClosure
		return
	}
	state.epsilonClosure = closure
```

- [ ] **Step 6: Update the `nfa2Dfa` start init**

In `nfa.go`, change
```go
	nfaStart.epsilonClosure = []*faState{nfaStart}
```
to
```go
	nfaStart.epsilonClosure = selfOnlyClosure
```

- [ ] **Step 7: Run the sentinel test; verify it PASSES**

Run: `go test -run TestSelfOnlyClosureSentinel -v .`
Expected: PASS.

- [ ] **Step 8: Run the full suite; identify the assertions to recalibrate**

Run: `go test ./... 2>&1 | tail -40`
Expected failures are ONLY tests that read closure lengths / closure-derived byte counts:
`TestEpsilonClosure` (`epsi_closure_test.go`), `TestEpsilonClosureRequired` (`value_matcher_test.go`), `TestMcNfaSizes` and `TestQuaminaMemoryCost` (`memory_cost_test.go`).

**GATE — if any other test fails, STOP and report BLOCKED.** In particular `TestTablePointerDedup`, `TestToxicStack`, `TestRegexpValidity`, and all matching tests MUST pass: this approach keeps multi-member closures (incl. self) and feeds `n2dNode` the identical state sets, so any failure there means the implementation changed behavior and is wrong.

- [ ] **Step 9: Recalibrate `TestEpsilonClosure` (`epsi_closure_test.go`)**

This test builds states and asserts `len(state.epsilonClosure)`. Every assertion of the form `len(x.epsilonClosure) != 1` / `... want 1` for a **self-only** state (one whose closure was `{self}`) becomes `len 0`. Update each such assertion to expect `0` and drop the now-wrong `containsState(t, x.epsilonClosure, x)` checks for self-only states (self is no longer stored). For genuinely multi-member closures (e.g. the splice state with closure size 2, and the `a?b?c?z` chain with closure size 5) the expected counts are **unchanged** and self is still present. Work assertion-by-assertion using the test's own comments; verify each changed state truly has no other non-epsilon-only state in its closure before setting it to 0.

- [ ] **Step 10: Recalibrate `TestEpsilonClosureRequired` (`value_matcher_test.go`)**

This test asserts the test setup *requires* closures. Re-read its body: it builds two patterns and checks matching depends on closures. With self processed explicitly, update the assertion(s) so the test still verifies the intended property; if the test's premise ("matched without closures") is now structurally different, adjust the assertion to reflect that multi-member closures (not self-only) are what carry the dependency. Confirm the test still fails if closures were genuinely broken (e.g., by reasoning about what it would report if a multi-member closure were dropped).

- [ ] **Step 11: Recalibrate `TestMcNfaSizes` and `TestQuaminaMemoryCost` (`memory_cost_test.go`)**

`stats.fanouts` is `sum(len(epsilonClosure))`; self-only states now contribute 0 instead of 1, so `fanouts` drops by exactly the count of self-only states in the FA. `stats.bytes` includes `mcPointer * cap(epsilonClosure)`; self-only states now contribute 0 instead of `mcPointer`. `maxFanout` (the largest single closure) is a multi-member closure and is unchanged.
- Run each test, read the `got` value, and update the `want` constant to the `got`.
- **Verify, don't blind-paste:** confirm `bytes` decreased and the decrease equals `mcPointer × (number of self-only states)`; confirm `fanouts` decreased by the same state count; confirm `maxFanout` is unchanged. If `maxFanout` changed or `bytes` moved in the wrong direction, STOP — something is off.

- [ ] **Step 12: Full suite green**

Run: `go vet . && go test ./...`
Expected: all PASS, including `TestToxicStack`, `TestRegexpValidity`, `TestTablePointerDedup`, `TestIncrementalClosureOrderIndependence`, `TestSelfOnlyClosureSentinel`.

- [ ] **Step 13: Commit**

```bash
git add epsi_closure.go nfa.go incremental_closure_test.go epsi_closure_test.go value_matcher_test.go memory_cost_test.go
git commit -m "epsi_closure: store {self} closures as a zero-alloc sentinel

The common self-only closure is now a shared non-nil empty slice instead of a
1-element backing array, so the majority of states allocate nothing for their
closure. Multi-member closures are unchanged (self stays inside the table dedup;
consumers iterate them whole), so matching, nfa2Dfa, and the adversarial regexp
tests are byte-for-byte preserved. Size/fanout assertions recalibrated for the
self-only=0 fanout contribution."
```

---

### Task 3: Measure (report, not pass/fail)

**Files:** throwaway probes (created then deleted).

- [ ] **Step 1: Build-scale curve**

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
		t.Logf("N=%4d build=%9.2fms per-word=%7.1fus", len(words),
			float64(el.Microseconds())/1000.0, float64(el.Microseconds())/float64(len(words)))
	}
}
```
Run: `go test -run TestBuildScaleProbe -v .` and record the numbers. Compare per-word time to the pre-change Phase-1 baseline (≈150µs@250 → 400µs@2000).

- [ ] **Step 2: Allocation count (the primary expected win)**

Run the build-scale probe with `-benchmem` is not meaningful here (build is in setup); instead confirm the allocation reduction with the research harness profile:
```bash
go build -o /tmp/researchbin ./research
/tmp/researchbin -cpuprofile=/tmp/research_selfonly.prof
go tool pprof -top -cum -nodecount=20 /tmp/researchbin /tmp/research_selfonly.prof
```
Record `Patterns/sec` (compare to the Phase-1 baseline of ~418/sec) and check whether `closureForState`'s `make`/`append` and the GC (`madvise`, `gcBgMarkWorker`) shares dropped.

- [ ] **Step 3: Match throughput (no-regression gate)**

Run: `go test -run='^$' -bench='BenchmarkCityLots$|BenchmarkWorkload_ManyOverlappingWildcards|BenchmarkWorkload_CacheThrashing' -benchmem -count=6 . > /tmp/selfonly_after.txt`
Compare against an equivalent baseline run on the parent commit (check out `HEAD~1` in a scratch checkout or run before Task 2 was applied). Confirm CityLots and the workloads are flat-or-better (the self-only branch replaces a 1-element slice iteration with a direct field/step access).

- [ ] **Step 4: Remove the probe**

Run: `rm -f buildscale_probe_test.go`
(Do NOT commit the probe.)

- [ ] **Step 5: Report**

Summarize: build-scale before/after, patterns/sec before/after, allocation/GC delta from the profile, resident-memory change if measured, and match-benchmark deltas. State plainly whether the build curve flattened (the unproven hypothesis) or whether the residual remains the huge hub closures.

---

## Self-Review

- **Spec coverage:** sentinel representation (Task 2 Steps 3-6) ✓; `closureForState` fast-path + main-path collapse + dedup-collapse (Steps 4-5) ✓; `nil`/`len0`/`len>=2` discriminator in `traverseNFA`×2 + `n2dNode` (Task 1) ✓; `nfa2Dfa` start (Step 6) ✓; recalibration of size/fanout assertions (Steps 9-11) ✓; the `TestToxicStack`/`TestRegexpValidity`/`TestTablePointerDedup` regression gate (Step 8, Step 12) ✓; measurement incl. match no-regression (Task 3) ✓. Spec's out-of-scope items (size-2 inlining, hub sharing, lazy) correctly have no task.
- **Placeholder scan:** recalibration constants (Steps 9-11) are inherently run-derived; each step gives concrete verification criteria (direction + magnitude tied to self-only state count), not a blind "update as needed".
- **Type/name consistency:** `selfOnlyClosure` defined once (Task 2 Step 3), used in `closureForState` (Steps 4-5) and `nfa2Dfa` (Step 6); the `len==0` discriminator in Task 1 matches the sentinel's `len 0`; `closureForStateNoBufs`/`newSmallTable`/`faState`/`readWWords` are existing symbols.
