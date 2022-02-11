package quamina

import (
	"testing"
)

func TestAddTransition(t *testing.T) {
	m := newValueMatcher()
	v1 := typedVal{
		vType: stringType,
		val:   "one",
	}
	t1 := m.addTransition(v1)
	if t1 == nil {
		t.Error("nil addTrans")
	}
	t1x := m.transitionOn([]byte("one"))
	if len(t1x) != 1 || t1x[0] != t1 {
		t.Error("Retrieve failed")
	}

	tXtra := m.addTransition(v1)
	if tXtra != t1 {
		t.Error("dupe trans missed")
	}

	v2 := typedVal{
		vType: stringType,
		val:   "two",
	}
	t2 := m.addTransition(v2)
	tXtra = m.addTransition(v2)
	if tXtra != t2 {
		t.Error("dupe trans missed")
	}

	t2x := m.transitionOn([]byte("two"))
	if len(t2x) != 1 || t2x[0] != t2 {
		t.Error("trans failed T2")
	}
	t1x = m.transitionOn([]byte("one"))
	if len(t1x) != 1 || t1x[0] != t1 {
		t.Error("Retrieve failed")
	}
	v3 := typedVal{
		vType: stringType,
		val:   "three",
	}
	t3 := m.addTransition(v3)
	t3x := m.transitionOn([]byte("three"))
	if len(t3x) != 1 || t3x[0] != t3 {
		t.Error("Match failed T3")
	}
	t2x = m.transitionOn([]byte("two"))
	if len(t2x) != 1 || t2x[0] != t2 {
		t.Error("trans failed T2")
	}
	t1x = m.transitionOn([]byte("one"))
	if len(t1x) != 1 || t1x[0] != t1 {
		t.Error("Retrieve failed")
	}

	v4 := typedVal{
		vType: existsTrueType,
		val:   "",
	}
	t4 := m.addTransition(v4)
	t4x := m.transitionOn([]byte("foo"))
	if len(t4x) != 1 || t4x[0] != t4 {
		t.Error("Trans failed T4")
	}
	t4x = m.transitionOn([]byte("two"))
	if len(t4x) != 2 {
		t.Error("Should get 2 results")
	}
	if !contains(t4x, t4) || !contains(t4x, t2) {
		t.Error("missing contains")
	}

}

func contains(list []*fieldMatchState, s *fieldMatchState) bool {
	for _, l := range list {
		if l == s {
			return true
		}
	}
	return false
}
