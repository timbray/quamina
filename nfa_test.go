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
			matched := testTraverseNFA(nfa, asQuotedBytes(t, should), transitions, bufs, pp)
			if len(matched) != 1 {
				t.Errorf("NFA %s didn't %s: ", test.pattern, should)
			}
		}
		for _, nope := range test.nopes {
			matched := testTraverseNFA(nfa, asQuotedBytes(t, nope), transitions, bufs, pp)
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

// testTraverseNFA wraps traverseNFA with the push/pop that tryToMatch
// normally provides. Test-only convenience so direct callers don't need
// to manage the transmap stack themselves.
func testTraverseNFA(table *smallTable, val []byte, transitions []*fieldMatcher, bufs *nfaBuffers, pp printer) []*fieldMatcher {
	tm := bufs.getTransmap()
	tm.push()
	result := traverseNFA(table, val, transitions, bufs, pp)
	tm.pop()
	return result
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

// TestOverlappingShellStyleNesting validates that the transmap's stack-based
// buffer management is necessary for correct results. It constructs a scenario
// where traverseNFA is called for field "a" and returns 2 fieldMatchers (because
// overlapping shellstyle patterns "*" and "foo*" both match). During iteration
// of those results, recursive tryToMatch calls traverseNFA again for field "b",
// which also returns 2+ results (from "*" and "bar*"). With a naive single-buffer
// transmap, the inner pop() overwrites both positions of the outer buffer, so the
// second fieldMatcher from field "a" is never properly processed. This causes
// patterns reachable only through that second fieldMatcher to be missed.
//
// The test is designed so that BOTH possible map iteration orderings of the outer
// result cause corruption: whichever fieldMatcher is processed first, its inner
// traversal produces 2 results that overwrite position [1], corrupting the other.
// With a correct stack-based transmap, all 4 patterns match. With a single shared
// buffer, only 2 of 4 are found.
func TestOverlappingShellStyleNesting(t *testing.T) {
	q, err := New()
	if err != nil {
		t.Fatal(err)
	}

	// Two patterns go through a:* (sharing one fieldMatcher after field "a")
	// with overlapping b patterns, so the inner traverseNFA returns 2 results.
	err = q.AddPattern("P1", `{"a": [{"shellstyle": "*"}], "b": [{"shellstyle": "*"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	err = q.AddPattern("P2", `{"a": [{"shellstyle": "*"}], "b": [{"shellstyle": "bar*"}]}`)
	if err != nil {
		t.Fatal(err)
	}

	// Two patterns go through a:foo* (sharing a different fieldMatcher after "a")
	// with overlapping b patterns, so this branch also produces 2 inner results.
	err = q.AddPattern("P3", `{"a": [{"shellstyle": "foo*"}], "b": [{"shellstyle": "*"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	err = q.AddPattern("P4", `{"a": [{"shellstyle": "foo*"}], "b": [{"shellstyle": "bar*"}]}`)
	if err != nil {
		t.Fatal(err)
	}

	event := []byte(`{"a": "fooX", "b": "barY"}`)
	want := map[string]bool{"P1": true, "P2": true, "P3": true, "P4": true}

	matches, err := q.MatchesForEvent(event)
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]bool, len(matches))
	for _, m := range matches {
		got[m.(string)] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("missing expected match %s, got %v", name, matches)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("unexpected match %s", name)
		}
	}
}

// TestThreeLevelNesting exercises 3 levels of nested traverseNFA calls. Field
// "a" has overlapping patterns producing 2 outer fieldMatchers. One branch goes
// through fields "b" then "c" (each with overlapping patterns), creating depth-3
// nesting. A separate branch through a:foo* reaches field "d" only if the outer
// buffer survives the nested corruption.
func TestThreeLevelNesting(t *testing.T) {
	q, err := New()
	if err != nil {
		t.Fatal(err)
	}

	// Branch through a:* → b → c (3 levels of NFA nesting)
	err = q.AddPattern("deep-1", `{"a": [{"shellstyle": "*"}], "b": [{"shellstyle": "*"}], "c": [{"shellstyle": "cat*"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	err = q.AddPattern("deep-2", `{"a": [{"shellstyle": "*"}], "b": [{"shellstyle": "bar*"}], "c": [{"shellstyle": "cow*"}]}`)
	if err != nil {
		t.Fatal(err)
	}

	// Branch through a:foo* → d (only reachable if outer buffer is intact)
	err = q.AddPattern("side", `{"a": [{"shellstyle": "foo*"}], "d": [{"shellstyle": "dog*"}]}`)
	if err != nil {
		t.Fatal(err)
	}

	event := []byte(`{"a": "fooX", "b": "barY", "c": "catZ", "d": "dogW"}`)
	want := map[string]bool{"deep-1": true, "side": true}

	// Run multiple iterations to exercise both map iteration orderings.
	for i := 0; i < 100; i++ {
		matches, err := q.MatchesForEvent(event)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}

		got := make(map[string]bool, len(matches))
		for _, m := range matches {
			got[m.(string)] = true
		}
		for name := range want {
			if !got[name] {
				t.Fatalf("iter %d: missing %s, got %v", i, name, matches)
			}
		}
		if got["deep-2"] {
			t.Fatalf("iter %d: unexpected deep-2 (c=catZ should not match cow*)", i)
		}
	}
}

// TestTransmapBufferReuse directly tests that the transmap buffer reuse is safe.
// The new API: push() in the caller (tryToMatch), traverseNFA writes into the
// current level's buffer via levels[depth]. Nested push() at a higher depth
// must not corrupt the outer level's buffer.
func TestTransmapBufferReuse(t *testing.T) {
	fm1 := &fieldMatcher{}
	fm2 := &fieldMatcher{}
	fm3 := &fieldMatcher{}

	tm := newTransMap()
	tm.resetDepth()

	// Simulate outer tryToMatch: push, then traverseNFA writes into levels[depth]
	tm.push() // depth 0
	buf := tm.levels[tm.depth][:0]
	buf = append(buf, fm1, fm2)
	tm.levels[tm.depth] = buf
	outerResult := tm.levels[tm.depth]

	if len(outerResult) != 2 {
		t.Fatalf("outer result before inner: got %d, want 2", len(outerResult))
	}

	// Simulate inner tryToMatch: push at depth 1, traverseNFA writes there
	tm.push() // depth 1
	innerBuf := tm.levels[tm.depth][:0]
	innerBuf = append(innerBuf, fm3)
	tm.levels[tm.depth] = innerBuf
	innerResult := tm.levels[tm.depth]

	if len(innerResult) != 1 || innerResult[0] != fm3 {
		t.Errorf("inner result: got %v, want [fm3]", innerResult)
	}
	tm.pop() // back to depth 0

	// BUG CHECK: outerResult must still have fm1, fm2
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
	tm.pop() // back to depth -1
}
