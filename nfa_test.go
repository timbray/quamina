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

func TestNfa2Dfa(t *testing.T) {
	type n2dtest struct {
		pattern string
		shoulds []string
		nopes   []string
	}
	tests := []n2dtest{
		{
			pattern: "*abc",
			shoulds: []string{"abc", "fooabc", "abcabc"},
			nopes:   []string{"abd", "fooac"},
		},
		{
			pattern: "a*bc",
			shoulds: []string{"abc", "axybc", "abcbc"},
			nopes:   []string{"abd", "fooac"},
		},
		{
			pattern: "ab*c",
			shoulds: []string{"abc", "abxyxc", "abbbbbc"},
			nopes:   []string{"abd", "abcxy"},
		},
		{
			pattern: "abc*",
			shoulds: []string{"abc", "abcfoo"},
			nopes:   []string{"xabc", "abxbar"},
		},
	}
	pp := newPrettyPrinter(4567)
	transitions := []*fieldMatcher{}
	bufs := newNfaBuffers()
	for _, test := range tests {
		nfa, _ := makeShellStyleFA(asQuotedBytes(t, test.pattern), pp)
		//fmt.Println("NFA: " + pp.printNFA(nfa))

		for _, should := range test.shoulds {
			matched := traverseNFA(nfa, asQuotedBytes(t, should), transitions, bufs, pp)
			if len(matched) != 1 {
				t.Errorf("NFA %s didn't %s: ", test.pattern, should)
			}
		}
		for _, nope := range test.nopes {
			matched := traverseNFA(nfa, asQuotedBytes(t, nope), transitions, bufs, pp)
			if len(matched) != 0 {
				t.Errorf("NFA %s matched %s", test.pattern, nope)
			}
		}
		dfa := nfa2Dfa(nfa)
		// fmt.Println("DFA: " + pp.printNFA(dfa.table))
		for _, should := range test.shoulds {
			matched := traverseDFA(dfa.table, asQuotedBytes(t, should), transitions)
			if len(matched) != 1 {
				t.Errorf("DFA %s didn't match %s ", test.pattern, should)
			}
		}
		for _, nope := range test.nopes {
			matched := traverseDFA(dfa.table, asQuotedBytes(t, nope), transitions)
			if len(matched) != 0 {
				t.Errorf("DFA %s matched %s", test.pattern, nope)
			}
		}
	}
}
func asQuotedBytes(t *testing.T, s string) []byte {
	t.Helper()
	s = `"` + s + `"`
	return []byte(s)
}

// TestNestedTransmapSafety verifies that the transmap handles nested traverseNFA calls correctly.
// The bug scenario: tryToMatch iterates over fieldMatchers returned from transitionOn (which uses
// the transmap buffer). During iteration, recursive tryToMatch calls transitionOn again, which
// would clobber the buffer if not handled properly. The stack-based transmap prevents this.
func TestNestedTransmapSafety(t *testing.T) {
	// Create patterns with shellstyle on multiple fields to force NFA mode and nested calls.
	// Field "a" comes before "b" lexically, so tryToMatch processes "a" first, then recurses for "b".
	patterns := []string{
		`{"a": [{"shellstyle": "foo*"}], "b": [{"shellstyle": "bar*"}]}`,
		`{"a": [{"shellstyle": "foo*"}], "b": [{"shellstyle": "baz*"}]}`,
		`{"a": [{"shellstyle": "fox*"}], "b": [{"shellstyle": "bar*"}]}`,
	}

	q, err := New()
	if err != nil {
		t.Fatal(err)
	}

	for i, p := range patterns {
		err = q.AddPattern(fmt.Sprintf("P%d", i), p)
		if err != nil {
			t.Fatalf("AddPattern %d: %v", i, err)
		}
	}

	// Events that match different combinations
	tests := []struct {
		event   string
		matches []string
	}{
		// Matches P0: a=foo*, b=bar*
		{`{"a": "fooXYZ", "b": "barXYZ"}`, []string{"P0"}},
		// Matches P1: a=foo*, b=baz*
		{`{"a": "fooABC", "b": "bazABC"}`, []string{"P1"}},
		// Matches P2: a=fox*, b=bar*
		{`{"a": "foxDEF", "b": "barDEF"}`, []string{"P2"}},
		// Matches P0 and P1: a=foo*, b matches both bar* and baz*
		{`{"a": "fooXYZ", "b": "bar"}`, []string{"P0"}},
		{`{"a": "fooXYZ", "b": "baz"}`, []string{"P1"}},
		// No match
		{`{"a": "nomatch", "b": "nomatch"}`, []string{}},
	}

	for _, tc := range tests {
		matches, err := q.MatchesForEvent([]byte(tc.event))
		if err != nil {
			t.Errorf("MatchesForEvent(%s): %v", tc.event, err)
			continue
		}

		if len(matches) != len(tc.matches) {
			t.Errorf("Event %s: got %d matches %v, want %d matches %v",
				tc.event, len(matches), matches, len(tc.matches), tc.matches)
			continue
		}

		// Verify expected matches
		matchSet := make(map[string]bool)
		for _, m := range matches {
			matchSet[m.(string)] = true
		}
		for _, want := range tc.matches {
			if !matchSet[want] {
				t.Errorf("Event %s: missing expected match %s, got %v", tc.event, want, matches)
			}
		}
	}
}

// TestTransmapBufferReuse directly tests that the transmap buffer reuse is safe.
// With a buggy single-buffer implementation, nested reset/all calls corrupt the outer buffer.
func TestTransmapBufferReuse(t *testing.T) {
	// Create dummy fieldMatchers for testing
	fm1 := &fieldMatcher{}
	fm2 := &fieldMatcher{}
	fm3 := &fieldMatcher{}

	tm := newTransMap()

	// Simulate start of matchesForFields - reset depth
	tm.resetDepth()

	// Simulate outer traverseNFA call
	tm.reset()
	tm.add([]*fieldMatcher{fm1, fm2})

	// Get outer result - this returns a buffer
	outerResult := tm.all()

	// Verify outer result before inner call
	if len(outerResult) != 2 {
		t.Fatalf("outer result before inner: got %d, want 2", len(outerResult))
	}

	// Remember which fieldMatchers we expect
	expectFM1 := outerResult[0] == fm1 || outerResult[1] == fm1
	expectFM2 := outerResult[0] == fm2 || outerResult[1] == fm2
	if !expectFM1 || !expectFM2 {
		t.Fatalf("outer result should have fm1 and fm2")
	}

	// Simulate inner traverseNFA call (would happen during iteration in tryToMatch)
	tm.reset()
	tm.add([]*fieldMatcher{fm3})
	innerResult := tm.all()

	// Inner should have fm3
	if len(innerResult) != 1 || innerResult[0] != fm3 {
		t.Errorf("inner result: got %v, want [fm3]", innerResult)
	}

	// THIS IS THE BUG CHECK: With single-buffer impl, outerResult would be corrupted.
	// The inner reset/all would overwrite the same buffer that outerResult points to.
	// With stack-based impl, they use different buffers at different depths.
	//
	// After inner call, check outerResult DIRECTLY (not a copy).
	// With buggy impl: outerResult[0] is now fm3 (corrupted!)
	// With stack impl: outerResult still has fm1, fm2

	foundFM1 := false
	foundFM2 := false
	for _, fm := range outerResult {
		if fm == fm1 {
			foundFM1 = true
		}
		if fm == fm2 {
			foundFM2 = true
		}
	}

	if !foundFM1 || !foundFM2 {
		t.Errorf("outer result was corrupted after inner call: expected fm1 and fm2, got %v", outerResult)
	}
}
