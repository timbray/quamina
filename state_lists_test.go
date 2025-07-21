package quamina

import "testing"

func TestStateLists(t *testing.T) {
	f1 := &faState{}
	f2 := &faState{}
	f3 := &faState{}
	f4 := &faState{}
	f5 := &faState{}

	list135A := []*faState{f3, f5, f1}
	list135B := []*faState{f5, f1, f3}

	lists := newStateLists()
	list1, dfa1, alreadyExisted := lists.intern(list135A)
	if dfa1 == nil {
		t.Error("DFA1 nil")
	}
	if alreadyExisted {
		t.Error("No it didn't")
	}
	list2, dfa2, alreadyExisted := lists.intern(list135B)
	if dfa2 != dfa1 {
		t.Error("DFA 1/2 differ")
	}
	if !alreadyExisted {
		t.Error("Yes it did")
	}
	if !stateListsEquals(t, list1, list2) {
		t.Error("Lists l1/l2 didn't match")
	}

	listAll1 := []*faState{f1, f5, f1, f4, f3, f2, f2, f4}
	listAll2 := []*faState{f1, f1, f1, f4, f3, f2, f2, f5, f4}
	listAll3 := []*faState{f5, f1, f4, f3, f2, f3, f5, f4}
	u1, dfa1, alreadyExisted := lists.intern(listAll1)
	if dfa1 == nil {
		t.Error("DFA1 nil")
	}
	if dfa1 == dfa2 {
		t.Error("DFA2 should be different")
	}
	if alreadyExisted {
		t.Error("No it didn't")
	}
	u2, dfa2, alreadyExisted := lists.intern(listAll2)
	if dfa2 != dfa1 {
		t.Error("DFA 1/2 differ")
	}
	if !alreadyExisted {
		t.Error("Yes it did")
	}
	u3, dfa3, alreadyExisted := lists.intern(listAll3)
	if dfa3 != dfa2 {
		t.Error("DFA 2/3 differ")
	}
	if !alreadyExisted {
		t.Error("Yes it did")
	}
	if !stateListsEquals(t, u1, u2) || !stateListsEquals(t, u3, u1) || !stateListsEquals(t, u2, u3) {
		t.Error("Ouch")
	}
}

func stateListsEquals(t *testing.T, list1, list2 []*faState) bool {
	t.Helper()
	return &list1[0] == &list2[0]
}
