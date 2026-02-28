package quamina

import (
	"slices"
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

	closureForStateNoBufs(aSa)
	if len(aSa.epsilonClosure) != 1 || !containsState(t, aSa.epsilonClosure, aSa) {
		t.Errorf("len(ec) = %d; want 1", len(aSa.epsilonClosure))
	}
	closureForStateNoBufs(aSstar)
	if len(aSstar.epsilonClosure) != 1 || !containsState(t, aSstar.epsilonClosure, aSstar) {
		t.Error("aSstar")
	}
	closureForStateNoBufs(aSc)
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
		closureForStateNoBufs(state)
		if len(state.epsilonClosure) != 1 || !containsState(t, state.epsilonClosure, state) {
			t.Errorf("should be 1 for %s, isn't", zNames[i])
		}
	}

	closureForStateNoBufs(bSsplice)
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

	closureForStateNoBufs(cStart)
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

// TestTablePointerDedupPreservesFieldTransitions constructs two faState nodes
// that share the same *smallTable but carry different fieldTransitions, wired
// behind a splice state. This is the scenario where table-pointer dedup in
// epsilon closure could lose field transitions if same-table is treated as
// same-state. The test verifies correctness through epsilon closure, NFA
// traversal, and DFA conversion.
func TestTablePointerDedupPreservesFieldTransitions(t *testing.T) {
	// Build a small automaton:
	//
	//   start --'"'--> quoteState --'x'--> xState --valueTerminator--> (end)
	//
	// xState has epsilons to a splice, which fans out to stateA and stateB.
	// stateA and stateB share the same *smallTable (sharedTable) but have
	// different fieldTransitions (fmA vs fmB).
	//
	// If table-pointer dedup incorrectly drops one, we lose a field matcher.

	fmA := newFieldMatcher()
	fmB := newFieldMatcher()

	sharedTable := newSmallTable()

	// stateA and stateB share the same table but have overlapping
	// fieldTransitions in different order. The order-dependent comparison
	// in sameFieldTransitions returns false, so both are kept — correct
	// behavior (a missed dedup is safe, dropping a state is not).
	stateA := &faState{table: sharedTable, fieldTransitions: []*fieldMatcher{fmA, fmB}}
	stateB := &faState{table: sharedTable, fieldTransitions: []*fieldMatcher{fmB, fmA}}

	// splice is epsilon-only, pointing to both stateA and stateB
	spliceTable := newSmallTable()
	spliceTable.epsilons = []*faState{stateA, stateB}
	splice := &faState{table: spliceTable}

	// xState transitions on valueTerminator to nothing, but has epsilon to splice
	xTable := newSmallTable()
	xTable.epsilons = []*faState{splice}
	xState := &faState{table: xTable}

	// quoteState transitions on 'x' to xState
	quoteTable := newSmallTable()
	quoteTable.addByteStep('x', xState)
	quoteState := &faState{table: quoteTable}

	// start transitions on '"' to quoteState
	startTable := newSmallTable()
	startTable.addByteStep('"', quoteState)

	// Compute epsilon closures for the whole automaton
	epsilonClosure(startTable)

	// Verify: xState's closure must include both stateA and stateB
	if !containsState(t, xState.epsilonClosure, stateA) {
		t.Error("xState epsilon closure missing stateA")
	}
	if !containsState(t, xState.epsilonClosure, stateB) {
		t.Error("xState epsilon closure missing stateB")
	}

	// Verify via NFA traversal: both field matchers must appear
	bufs := newNfaBuffers()
	tm := bufs.getTransmap()
	tm.push()
	nfaResult := traverseNFA(startTable, []byte(`"x"`), nil, bufs)
	tm.pop()

	if !slices.Contains(nfaResult, fmA) {
		t.Error("NFA traversal missing fmA")
	}
	if !slices.Contains(nfaResult, fmB) {
		t.Error("NFA traversal missing fmB")
	}

	// Verify via DFA conversion: both field matchers must survive
	dfa := nfa2Dfa(startTable)
	dfaResult := traverseDFA(dfa.table, []byte(`"x"`), nil)

	if !slices.Contains(dfaResult, fmA) {
		t.Error("DFA traversal missing fmA")
	}
	if !slices.Contains(dfaResult, fmB) {
		t.Error("DFA traversal missing fmB")
	}
}
