package quamina

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
)

// stabilizeHeap runs GC a couple of times so HeapAlloc reflects a
// relatively clean state before the test creates its coreMatcher.
// Without this, tests that run after others in the same process see a
// heap that still holds unreclaimed garbage, which moves the baseline
// around enough to flap budget assertions.
func stabilizeHeap(t *testing.T) {
	t.Helper()
	runtime.GC()
	runtime.GC()
}

func TestMemoryBudgetBasic(t *testing.T) {
	// Budgets are measured against runtime.MemStats.HeapAlloc; numbers
	// here are chosen with enough slack that GC-driven baseline drift
	// between tests can't flip the expected outcomes.
	stabilizeHeap(t)
	cm := newCoreMatcher()
	var err error
	i1, i2 := cm.getMemoryBudget()
	if i1 != 0 || i2 != 0 {
		t.Error("Ouch 1")
	}
	// 64 KiB is comfortably larger than the few spans a single short
	// pattern claims.
	var budget uint64 = 64 * 1024
	i2, err = cm.setMemoryBudget(budget)
	if err != nil {
		t.Error(err)
	}
	if i2 != 0 {
		t.Errorf("i1/2 %d/%d\n", i1, i2)
	}
	err = cm.addPattern("x", `{"x":["abc"]}`)
	if err != nil {
		t.Error(err)
	}
	i1, _ = cm.getMemoryBudget()
	if i1 != budget {
		t.Errorf("i1: %d\n", i1)
	}

	// After adding a pattern currentMemory is >= 1 span (8 KiB), so
	// requesting a 16-byte budget must be rejected.
	_, err = cm.setMemoryBudget(0x10)
	if err == nil {
		t.Error("allowed invalid memory budget reduction")
	}
	i200 := iString(t, 200)
	cm = newCoreMatcher()
	// 32 KiB fits a tiny pattern (~2 spans) but not a 200-char pattern
	// (the extra FA alone needs ~12 spans).
	_, err = cm.setMemoryBudget(32 * 1024)
	if err != nil {
		t.Error(err)
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, "xyz"))
	if err != nil {
		t.Error(err)
	}
	_, _ = cm.getMemoryBudget()

	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, "abc"))
	if err != nil {
		t.Error(err)
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, i200))
	if err == nil {
		t.Error("Accepted overly long string")
	} else {
		fmt.Println("E: " + err.Error())
	}
}

func TestMemoryStress(t *testing.T) {
	// Verify two things about budget enforcement: (1) a generous
	// budget lets all patterns load, (2) a budget set well below what
	// step (1) required halts pattern loading somewhere in the middle.
	stabilizeHeap(t)
	words := readWWords(t, 20)
	source := rand.NewSource(293591)
	patterns := make([]string, 0, len(words))
	for _, word := range words {
		//nolint:gosec
		starAt := source.Int63() % 6
		starWord := string(word[:starAt]) + "*" + string(word[starAt:])
		patterns = append(patterns, fmt.Sprintf(`{"x": ["%s"]}`, starWord))
	}

	// Phase 1: generous budget; all patterns should fit.
	q, _ := New()
	if _, err := q.SetMemoryBudget(100 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	for _, p := range patterns {
		if err := q.AddPattern("x", p); err != nil {
			t.Fatalf("unexpected failure under generous budget: %s", err)
		}
	}
	_, totalNeeded := q.GetMemoryBudget()
	if totalNeeded == 0 {
		t.Fatal("no memory accumulated — accounting is stuck")
	}

	// Phase 2: tight budget — half of the measured total. The loader
	// should trip somewhere before the end.
	q2, _ := New()
	if _, err := q2.SetMemoryBudget(totalNeeded / 2); err != nil {
		t.Fatal(err)
	}
	var firstFailAt = -1
	for i, p := range patterns {
		if err := q2.AddPattern("x", p); err != nil {
			firstFailAt = i
			break
		}
	}
	if firstFailAt == -1 {
		t.Errorf("tight budget (%d) allowed all %d patterns; expected trip",
			totalNeeded/2, len(patterns))
	}
}

func iString(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	for i := range b {
		b[i] = 'i'
	}
	return string(b)
}

func TestStringFA(t *testing.T) {
	// Budgets are chosen in roughly-8-KiB increments for margin —
	// HeapAlloc gives byte-precise readings but addPattern's retained
	// bytes still vary a bit run-to-run with allocator state, so a
	// generous spread between the pass/fail thresholds keeps the test
	// stable.
	stabilizeHeap(t)
	const span uint64 = 8192
	cm := newCoreMatcher()
	var err error
	// Between 2 spans and 3 spans: fits a tiny pattern but not the
	// 100-char FA on top of it.
	_, err = cm.setMemoryBudget(2*span + span/2)
	if err != nil {
		t.Error("SMB")
	}
	// force it to build an FA
	err = cm.addPattern("x", `{"x": ["x"]}`)
	if err != nil {
		t.Error("x?")
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, iString(t, 100)))
	if err == nil {
		t.Error("should not succeed")
	}
	_, err = cm.setMemoryBudget(10_000_000)
	if err != nil {
		t.Error(err)
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, iString(t, 100)))
	if err != nil {
		t.Error("should succeed")
	}
	_, current := cm.getMemoryBudget()
	cm = newCoreMatcher()
	_, _ = cm.setMemoryBudget(current)
	err = cm.addPattern("x", `{"x": ["x"]}`)
	if err != nil {
		t.Error("x?")
	}
	i100 := iString(t, 100)
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, i100))
	if err != nil {
		t.Error("should succeed")
	}

	_, current = cm.getMemoryBudget()
	cm = newCoreMatcher()
	// Drop by one span: under HeapAlloc with per-byte resolution, a
	// sub-span reduction would still leave plenty of headroom for the
	// second add to slip through. A full span's margin is comfortably
	// larger than typical HeapAlloc jitter between runs.
	_, _ = cm.setMemoryBudget(current - span)
	err = cm.addPattern("x", `{"x": ["x"]}`)
	if err != nil {
		t.Error("x?")
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, i100))
	if err == nil {
		t.Error("should fail")
	}
}
