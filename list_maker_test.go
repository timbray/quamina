package quamina

import (
	"testing"
)

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
