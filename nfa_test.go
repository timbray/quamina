package quamina

import (
	"fmt"
	"testing"
	"unsafe"
)

// TestArrayBehavior is here prove that (a) you can index a map with an array and
// the indexing actually relies on the values in the array. This has nothing to do with
// Quamina, but I'm leaving it here because I had to write this stupid test after failing
// to find a straightforward question of whether this works as expected anywhere in the
// Golang docs.
func TestArrayBehavior(t *testing.T) {
	type gpig [4]int
	pigs := []gpig{
		{1, 2, 3, 4},
		{4, 3, 2, 1},
	}
	nonPigs := []gpig{
		{3, 4, 3, 4},
		{99, 88, 77, 66},
	}
	m := make(map[gpig]bool)
	for _, pig := range pigs {
		m[pig] = true
	}
	for _, pig := range pigs {
		_, ok := m[pig]
		if !ok {
			t.Error("missed pig")
		}
	}
	pigs[0][0] = 111
	pigs[1][3] = 777
	pigs = append(pigs, nonPigs...)
	for _, pig := range pigs {
		_, ok := m[pig]
		if ok {
			t.Error("mutant pig")
		}
	}
	newPig := gpig{1, 2, 3, 4}
	_, ok := m[newPig]
	if !ok {
		t.Error("Newpig")
	}
}

func TestFocusedMerge(t *testing.T) {
	shellStyles := []string{
		"a*b",
		"ab*",
		"*ab",
	}
	var automata []*smallTable
	var matchers []*fieldMatcher

	for _, shellStyle := range shellStyles {
		str := `"` + shellStyle + `"`
		automaton, matcher := makeShellStyleFA([]byte(str), &nullPrinter{})
		automata = append(automata, automaton)
		matchers = append(matchers, matcher)
	}

	var cab uintptr
	for _, mm := range matchers {
		uu := uintptr(unsafe.Pointer(mm))
		cab = cab ^ uu
	}

	merged := newSmallTable()
	for _, automaton := range automata {
		merged = mergeFAs(merged, automaton, sharedNullPrinter)

		s := statsAccum{
			fmVisited: make(map[*fieldMatcher]bool),
			vmVisited: make(map[*valueMatcher]bool),
			stVisited: make(map[*smallTable]bool),
		}
		faStats(merged, &s)
		fmt.Println(s.stStats())
	}
}
