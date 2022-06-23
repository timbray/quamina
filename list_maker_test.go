package quamina

import (
	"testing"
)

func TestDfaMemory(t *testing.T) {
	d1 := &dfaStep{}
	d3 := &dfaStep{}
	d12 := &dfaStep{}
	d13 := &dfaStep{}
	d123 := &dfaStep{}
	ns1 := &nfaStep{}
	ns2 := &nfaStep{}
	ns3 := &nfaStep{}
	l1 := []*nfaStep{ns1}
	l3 := []*nfaStep{ns3}
	l12 := []*nfaStep{ns1, ns2}
	l13 := []*nfaStep{ns1, ns3}
	l123 := []*nfaStep{ns1, ns2, ns3}

	mem := newDfaMemory()
	mem.rememberDfaForList(d1, l1...)
	mem.rememberDfaForList(d3, l3...)
	mem.rememberDfaForList(d12, l12...)
	mem.rememberDfaForList(d13, l13...)
	mem.rememberDfaForList(d123, l123...)

	var ok bool
	var d *dfaStep
	d, ok = mem.dfaForNfas(l1...)
	if ok == false || d != d1 {
		t.Error("failed d1")
	}
	d, ok = mem.dfaForNfas(l3...)
	if ok == false || d != d3 {
		t.Error("failed d1")
	}
	var shouldMatches [][]*nfaStep
	shouldMatches = [][]*nfaStep{{ns1, ns2}, {ns2, ns1}}
	for i, should := range shouldMatches {
		d, ok := mem.dfaForNfas(should...)
		if ok == false || d != d12 {
			t.Errorf("no match on %d", i)
		}
	}
	shouldMatches = [][]*nfaStep{{ns1, ns3}, {ns3, ns1}}
	for i, should := range shouldMatches {
		d, ok := mem.dfaForNfas(should...)
		if ok == false || d != d13 {
			t.Errorf("no match on %d", i)
		}
	}
	shouldMatches = [][]*nfaStep{{ns1, ns2, ns3}, {ns1, ns3, ns2}, {ns3, ns1, ns2}, {ns3, ns2, ns1}}
	for i, should := range shouldMatches {
		d, ok := mem.dfaForNfas(should...)
		if ok == false || d != d123 {
			t.Errorf("no match on %d", i)
		}
	}

	noDfaFor := [][]*nfaStep{
		{&nfaStep{}},
		{ns2},
		{ns3, ns2},
		{ns1, ns2, &nfaStep{}},
		{ns1, ns2, ns3, &nfaStep{}},
	}

	for i, no := range noDfaFor {
		_, ok = mem.dfaForNfas(no...)
		if ok {
			t.Errorf("bogus match %d", i)
		}
	}
}

func TestListMaker(t *testing.T) {
	steps := []*nfaStep{
		{},
		{},
		{},
	}
	multi := [][]*nfaStep{
		{steps[0]},
		{steps[0], steps[1]},
		{steps[0], steps[1], steps[2]},
		{steps[0], steps[2]},
		{steps[1]},
		{steps[1], steps[2]},
		{steps[2]},
	}
	lm := newListMaker()
	lists := make(map[*nfaStepList]bool)
	for _, step := range steps {
		lists[lm.getSingleton(step)] = true
	}
	if len(lists) != 3 {
		t.Error("length should be 3")
	}
	for _, step := range steps {
		lists[lm.getSingleton(step)] = true
	}
	if len(lists) != 3 {
		t.Error("length STILL should be 3")
	}
	lm = newListMaker()
	lists = make(map[*nfaStepList]bool)
	for _, plural := range multi {
		lists[lm.getList(plural...)] = true
	}
	wanted := len(multi)
	if len(lists) != wanted {
		t.Errorf("Got %d wanted %d", len(lists), wanted)
	}
	for _, plural := range multi {
		lists[lm.getList(plural...)] = true
	}
	if len(lists) != wanted {
		t.Errorf("Got %d STILL wanted %d", len(lists), wanted)
	}
}
