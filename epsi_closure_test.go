package quamina

import (
	"testing"
)

func TestEpsilonClosure(t *testing.T) {
	var st *smallTable

	pp := newPrettyPrinter(4589)

	st = newSmallTable()
	aSa := &faState{table: st}
	pp.labelTable(aSa.table, "aSa")
	aSstar := &faState{}
	aSc := &faState{}
	st.addByteStep('b', aSstar)
	st = newSmallTable()
	st.epsilons = []*faState{aSstar}
	st.addByteStep('c', aSc)
	aSstar.table = st
	pp.labelTable(aSstar.table, "aSstar")
	aSc.table = newSmallTable()
	pp.labelTable(aSc.table, "aSc")
	aFM := newFieldMatcher()
	aSc.fieldTransitions = []*fieldMatcher{aFM}

	closureForState(aSa)
	if len(aSa.epsilonClosure) != 1 || !containsState(t, aSa.epsilonClosure, aSa) {
		t.Errorf("len(ec) = %d; want 1", len(aSa.epsilonClosure))
	}
	closureForState(aSstar)
	if len(aSstar.epsilonClosure) != 1 || !containsState(t, aSstar.epsilonClosure, aSstar) {
		t.Error("aSstar")
	}
	closureForState(aSc)
	if len(aSc.epsilonClosure) != 1 || !containsState(t, aSc.epsilonClosure, aSc) {
		t.Error("aSc")
	}

	// (b) ab|*x
	var bSa, bSb, bSsplice, bSstar, bSx *faState
	st = newSmallTable()

	bSa = &faState{table: st}
	bFM1 := newFieldMatcher()
	bSb = &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{bFM1}}
	bSa.table.addByteStep('b', bSb)
	bFM2 := newFieldMatcher()
	bSx = &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{bFM2}}

	st = newSmallTable()
	bSstar = &faState{table: st}
	st.epsilons = []*faState{bSstar}
	st.addByteStep('x', bSx)
	st.epsilons = []*faState{bSstar}

	st = newSmallTable()
	st.epsilons = []*faState{bSa, bSstar}
	bSsplice = &faState{table: st}

	// 	var bSa, bSb, bSsplice, bSstar, bSx *faState
	pp.labelTable(bSa.table, "bSa")
	pp.labelTable(bSb.table, "bSb")
	pp.labelTable(bSstar.table, "bSstar")
	pp.labelTable(bSx.table, "bSx")
	pp.labelTable(bSsplice.table, "bSsplice")

	bEcShouldBeOne := []*faState{bSa, bSb, bSx, bSstar}
	zNames := []string{"bSa", "bSb", "bSx", "bSstar"}
	for i, state := range bEcShouldBeOne {
		closureForState(state)
		if len(state.epsilonClosure) != 1 || !containsState(t, state.epsilonClosure, state) {
			t.Errorf("should be 1 for %s, isn't", zNames[i])
		}
	}

	closureForState(bSsplice)
	if len(bSsplice.epsilonClosure) != 2 || !containsState(t, bSsplice.epsilonClosure, bSa) || !containsState(t, bSsplice.epsilonClosure, bSstar) {
		t.Error("wrong EC for b")
	}

	// a?b?c?z
	var cStart, cSa, cSb, cSc, cSz *faState
	cStart = &faState{table: newSmallTable()}
	cSa = &faState{table: newSmallTable()}
	cSb = &faState{table: newSmallTable()}
	cSc = &faState{table: newSmallTable()}
	cSz = &faState{table: newSmallTable()}

	cStart.table.addByteStep('a', cSa)
	cStart.table.epsilons = []*faState{cSa}
	cSa.table.addByteStep('b', cSb)
	cSa.table.epsilons = []*faState{cSb}
	cSb.table.addByteStep('c', cSc)
	cSb.table.epsilons = []*faState{cSc}
	cSc.table.addByteStep('z', cSz)
	cSc.table.epsilons = []*faState{cSz}
	cFM := newFieldMatcher()
	cSz.fieldTransitions = []*fieldMatcher{cFM}
	names := []string{"cStart", "cSa", "cSb", "cSc", "cSz"}
	states := []*faState{cStart, cSa, cSb, cSc, cSz}
	for i, name := range names {
		st = states[i].table
		pp.labelTable(st, name)
	}

	closureForState(cStart)
	cWantInEC := []*faState{cStart, cSa, cSb, cSc, cSz}
	if len(cStart.epsilonClosure) != 5 {
		t.Errorf("len B ec %d wanted 5", len(cStart.epsilonClosure))
	}
	for i, want := range cWantInEC {
		if !containsState(t, cStart.epsilonClosure, want) {
			t.Errorf("C missed %s", names[i])
		}
	}
}
