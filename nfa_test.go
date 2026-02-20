package quamina

import (
	"fmt"
	"strings"
	"testing"
	"time"
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

// collectClosureStats walks an NFA and reports epsilon closure size statistics.
func collectClosureStats(startTable *smallTable) (stateCount, totalEntries, maxClosure int, tableSharing int) {
	visitedTables := make(map[*smallTable]bool)
	visitedStates := make(map[*faState]bool)
	tableCounts := make(map[*smallTable]int)

	var walkTable func(t *smallTable)
	walkTable = func(t *smallTable) {
		if t == nil || visitedTables[t] {
			return
		}
		visitedTables[t] = true
		for _, state := range t.steps {
			if state != nil && !visitedStates[state] {
				visitedStates[state] = true
				tableCounts[state.table]++
				ec := len(state.epsilonClosure)
				totalEntries += ec
				if ec > maxClosure {
					maxClosure = ec
				}
				walkTable(state.table)
			}
		}
		for _, eps := range t.epsilons {
			if !visitedStates[eps] {
				visitedStates[eps] = true
				tableCounts[eps.table]++
				ec := len(eps.epsilonClosure)
				totalEntries += ec
				if ec > maxClosure {
					maxClosure = ec
				}
				walkTable(eps.table)
			}
		}
	}
	walkTable(startTable)

	for _, count := range tableCounts {
		if count > 1 {
			tableSharing += count - 1
		}
	}
	return len(visitedStates), totalEntries, maxClosure, tableSharing
}

// TestEpsilonClosureSizes measures epsilon closure sizes and matching speed
// for pattern workloads that exercise table-pointer dedup. Nested quantifiers
// over overlapping character classes create the most table sharing.
func TestEpsilonClosureSizes(t *testing.T) {
	type workload struct {
		name     string
		patterns []string
		regexps  []string
	}

	workloads := []workload{
		{
			name: "6-regexps-12-shell",
			patterns: []string{
				"*a*b*c*", "*x*y*z*", "*e*f*g*", "*m*n*o*",
				"*p*q*r*", "*s*t*u*", "*a*e*i*", "*b*d*f*",
				"*c*g*k*", "*d*h*l*", "*i*o*u*", "*r*s*t*",
			},
			regexps: []string{
				"(([abc]?)*)+", "([abc]+)*d", "(a*)*b",
				"([xyz]?)*end", "(([mno]?)*)+", "([pqr]+)*s",
			},
		},
		{
			name: "20-nested-regexps",
			regexps: []string{
				"(([abc]?)*)+", "([abc]+)*d", "(a*)*b", "([xyz]?)*end",
				"(([mno]?)*)+", "([pqr]+)*s", "(([def]?)*)+", "([ghi]+)*j",
				"(([stu]?)*)+", "([vwx]+)*y", "(b*)*c", "(d*)*e",
				"(([fg]?)*)+", "([hi]+)*k", "(([jk]?)*)+", "([lm]+)*n",
				"(([op]?)*)+", "([qr]+)*t", "(e*)*f", "(g*)*h",
			},
		},
		{
			name: "deeply-nested",
			regexps: []string{
				"(((a?)*b?)*c?)*",
				"(((x?)*y?)*z?)*",
				"(((d?)*e?)*f?)*",
				"(((m?)*n?)*o?)*",
				"((((a?)*b?)*c?)*d?)*",
				"((((x?)*y?)*z?)*w?)*",
			},
		},
		{
			name: "overlapping-char-classes",
			regexps: []string{
				"(([abc]?)*)+", "(([bcd]?)*)+", "(([cde]?)*)+",
				"(([def]?)*)+", "(([efg]?)*)+", "(([fgh]?)*)+",
				"(([ghi]?)*)+", "(([hij]?)*)+", "(([ijk]?)*)+",
				"(([jkl]?)*)+", "(([klm]?)*)+", "(([lmn]?)*)+",
			},
		},
		{
			name: "shell+deep-overlap",
			patterns: []string{
				"*a*b*", "*b*c*", "*c*d*", "*d*e*", "*e*f*",
				"*a*c*", "*b*d*", "*c*e*", "*d*f*", "*a*d*",
			},
			regexps: []string{
				"(((a?)*b?)*c?)*", "(((b?)*c?)*d?)*", "(((c?)*d?)*e?)*",
				"(((d?)*e?)*f?)*", "(([abcd]?)*)+", "(([cdef]?)*)+",
			},
		},
	}

	for _, wl := range workloads {
		t.Run(wl.name, func(t *testing.T) {
			q, _ := New()
			m := q.matcher.(*coreMatcher)
			i := 0
			for _, ss := range wl.patterns {
				pattern := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, ss)
				if err := q.AddPattern(fmt.Sprintf("s%d", i), pattern); err != nil {
					t.Fatal(err)
				}
				i++
			}
			for _, re := range wl.regexps {
				pattern := fmt.Sprintf(`{"val": [{"regexp": "%s"}]}`, re)
				if err := q.AddPattern(fmt.Sprintf("r%d", i), pattern); err != nil {
					t.Fatal(err)
				}
				i++
			}

			vm := m.fields().state.fields().transitions["val"]
			nfaStart := vm.fields().startTable
			stateCount, totalEntries, maxClosure, tableSharing := collectClosureStats(nfaStart)
			avg := float64(totalEntries) / float64(stateCount)
			t.Logf("states=%d, closure_entries=%d, avg=%.1f, max=%d, table_sharing=%d",
				stateCount, totalEntries, avg, maxClosure, tableSharing)

			events := [][]byte{
				[]byte(`{"val": "abcdefgh"}`),
				[]byte(`{"val": "` + strings.Repeat("abcdef", 5) + `"}`),
				[]byte(`{"val": "` + strings.Repeat("abcdefghijklmnop", 3) + `"}`),
			}
			start := time.Now()
			for iter := 0; iter < 1000; iter++ {
				for _, event := range events {
					_, _ = q.MatchesForEvent(event)
				}
			}
			elapsed := time.Since(start)
			t.Logf("3000 matches in %v (%.0f/sec)", elapsed, 3000.0/elapsed.Seconds())
		})
	}
}
