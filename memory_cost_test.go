package quamina

import (
	"testing"
	"unsafe"
)

func TestMcBasicSizes(t *testing.T) {
	tableBase := int64(unsafe.Sizeof(smallTable{}))
	table := newSmallTable()
	// NewSmallTable output: base + 1 byte of ceiling + 1 pointer of steps (8b) +
	want := tableBase + 1 + mcPointer
	tableGot := mcSmallTable(&table)
	if want != tableGot {
		t.Errorf("Table wanted %d got %d", want, mcSmallTable(&table))
	}
	stateBase := int64(unsafe.Sizeof(faState{}))
	state := faState{table: table}
	// faState embeds smallTable, so stateBase already covers the smallTable struct.
	// Add only the slice-backing bytes (1 ceiling byte + 1 step pointer), not tableGot
	// (which includes mcSmallTableBase again and would double-count the struct overhead).
	want = stateBase + 1 + mcPointer
	stateGot := mcFaState(&state)
	if stateGot != want {
		t.Errorf("State wanted %d got %d", want, stateGot)
	}
}

func TestQuaminaMemoryCost(t *testing.T) {
	q, _ := New()
	err := q.AddPattern("x", `{"x":[{"wildcard": "*z"}]}}`)
	if err != nil {
		t.Error(err)
	}
	bytes := q.GetMatcherStats()["bytes"]
	if bytes != 1257 {
		t.Error("WRONG NUMBERS")
	}
	err = q.AddPattern("x", `{"y":[{"wildcard": "*y"}]}}`)
	if err != nil {
		t.Error(err)
	}
	bytes = q.GetMatcherStats()["bytes"]
	if bytes != 2*1257 {
		t.Error("WRONG NUMBERS")
	}
}

// Regression: GetMatcherStats panicked when a valueMatcher used the
// singleton-match optimization (singletonMatch set, start nil).
// That optimization fires for any field with a single string or literal
// value — the matcher uses bytes.Compare instead of building an FA.
// Minimal repro: {"Animated": [false]}. cmFieldMatcherStats now skips
// the nil start rather than building a faState with state.table == nil.
func TestQuaminaMemoryCostSingleton(t *testing.T) {
	q, _ := New()
	if err := q.AddPattern("p", `{"Animated": [false]}`); err != nil {
		t.Fatal(err)
	}
	s := q.GetMatcherStats()
	if s["bytes"] == 0 {
		t.Errorf("expected bytes > 0 for singleton matcher, got %v", s["bytes"])
	}
}

func TestMcNfaSizes(t *testing.T) {
	pp := newPrettyPrinter(2355)
	wc1 := `"*z"`
	fa1, _ := makeWildCardFA([]byte(wc1), pp)
	epsilonClosure(fa1)
	//fmt.Println("FA1: " + pp.printNFA(fa1))

	stats := &matcherStats{
		seenStates: make(map[*faState]bool),
	}
	cmStateStats(fa1, stats, pp)
	wantedBytes := int64(1257) // recalibrated after self-only closure sentinel (no backing array)
	wantedFanout := int64(2)
	wantedMaxFanout := int64(2)
	if stats.bytes != wantedBytes {
		t.Errorf("Wanted %d bytes, got %d", wantedBytes, stats.bytes)
	}
	if stats.fanouts != wantedFanout {
		t.Errorf("wanted %d fanout, got %d", wantedFanout, stats.fanouts)
	}
	if stats.maxFanout != wantedMaxFanout {
		t.Errorf("wanted %d maxFanout, got %d", wantedMaxFanout, stats.maxFanout)
	}
}
