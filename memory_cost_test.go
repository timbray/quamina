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
	tableGot := mcSmallTable(table)
	if want != tableGot {
		t.Errorf("Table wanted %d got %d", want, mcSmallTable(table))
	}
	stateBase := int64(unsafe.Sizeof(faState{}))
	state := faState{table: table}
	// want base + tableActual
	want = stateBase + tableGot
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
	if bytes != 1321 {
		t.Error("WRONG NUMBERS")
	}
	err = q.AddPattern("x", `{"y":[{"wildcard": "*y"}]}}`)
	if err != nil {
		t.Error(err)
	}
	bytes = q.GetMatcherStats()["bytes"]
	if bytes != 2*1321 {
		t.Error("WRONG NUMBERS")
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
	cmStateStats(&faState{table: fa1}, stats, pp)
	wantedBytes := int64(1321) // laboriously hand-calculated
	wantedFanout := int64(5)
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
