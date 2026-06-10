package quamina

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// testTransitionOn wraps valueMatcher.transitionOn with the push/pop that
// tryToMatch normally provides.
func testTransitionOn(vm *valueMatcher, val []byte, bufs *nfaBuffers) []*fieldMatcher {
	tm := bufs.getTransmap()
	tm.push()
	result := vm.transitionOn(&Field{Val: val}, bufs)
	tm.pop()
	return result
}

func TestInvalidValueTypes(t *testing.T) {
	var before []typedVal
	addInvalid(t, before)

	before = append(before, typedVal{vType: stringType, val: "foo"})
	addInvalid(t, before)

	before = append(before, typedVal{vType: stringType, val: "bar"})
	addInvalid(t, before)
}
func addInvalid(t *testing.T, before []typedVal) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("TestAddInvalidTransition should have panicked")
		}
	}()

	panicType := valType(999)

	// empty value matcher
	m := newValueMatcher()
	invalidField := typedVal{
		vType: panicType,
		val:   "one",
	}
	for _, addBefore := range before {
		_ = m.addTransition(addBefore, &nullPrinter{}, newClosureBuffers())
	}
	_ = m.addTransition(invalidField, &nullPrinter{}, newClosureBuffers())
}

func TestNoOpTransition(t *testing.T) {
	vm := newValueMatcher()
	tr := vm.transitionOn(&Field{Val: []byte("foo")}, &nfaBuffers{})
	if len(tr) != 0 {
		t.Error("matched on empty valuematcher")
	}
}

func TestAddTransition(t *testing.T) {
	m := newValueMatcher()
	v1 := typedVal{
		vType: stringType,
		val:   "one",
	}
	t1 := m.addTransition(v1, &nullPrinter{}, newClosureBuffers())
	if t1 == nil {
		t.Error("nil addTrans")
	}
	t1x := m.transitionOn(&Field{Val: []byte("one")}, &nfaBuffers{})
	if len(t1x) != 1 || t1x[0] != t1 {
		t.Error("Retrieve failed")
	}

	tXtra := m.addTransition(v1, &nullPrinter{}, newClosureBuffers())
	if tXtra != t1 {
		t.Error("dupe trans missed")
	}

	v2 := typedVal{
		vType: stringType,
		val:   "two",
	}
	t2 := m.addTransition(v2, &nullPrinter{}, newClosureBuffers())

	t2x := m.transitionOn(&Field{Val: []byte("two")}, &nfaBuffers{})
	if len(t2x) != 1 || t2x[0] != t2 {
		t.Error("trans failed T2")
	}
	t1x = m.transitionOn(&Field{Val: []byte("one")}, &nfaBuffers{})
	if len(t1x) != 1 || t1x[0] != t1 {
		t.Error("Retrieve failed")
	}
	v3 := typedVal{
		vType: stringType,
		val:   "three",
	}
	t3 := m.addTransition(v3, &nullPrinter{}, newClosureBuffers())
	t3x := m.transitionOn(&Field{Val: []byte("three")}, &nfaBuffers{})
	if len(t3x) != 1 || t3x[0] != t3 {
		t.Error("Match failed T3")
	}
	t2x = m.transitionOn(&Field{Val: []byte("two")}, &nfaBuffers{})
	if len(t2x) != 1 || t2x[0] != t2 {
		t.Error("trans failed T2")
	}
	t1x = m.transitionOn(&Field{Val: []byte("one")}, &nfaBuffers{})
	if len(t1x) != 1 || t1x[0] != t1 {
		t.Error("Retrieve failed")
	}
}

func TestMultiTransitions(t *testing.T) {
	patX := `{"foo": [ { "shellstyle": "*x*b" } ]}`
	patY := `{"foo": [ { "shellstyle": "*y*b" } ]}`

	m := newCoreMatcher()
	if m.addPattern("X", patX) != nil {
		t.Error("add patX")
	}
	if m.addPattern("Y", patY) != nil {
		t.Error("add patY")
	}
	e := `{"foo": "axyb"}`
	matches, err := m.matchesForJSONEvent([]byte(e))
	if err != nil {
		t.Error("m4: " + err.Error())
	}
	if len(matches) != 2 {
		t.Error("just one")
	}
}

func TestAY(t *testing.T) {
	m := newCoreMatcher()
	pat := `{"x": [ { "shellstyle": "*ay*"} ] }`
	err := m.addPattern("AY", pat)
	if err != nil {
		t.Error("AY: " + err.Error())
	}
	shouldMatch := []string{"ay", "aay", "aaaayyyyy", "xyzay", "ayxxxx"}
	e := `{"x": "X"}`
	for _, sm := range shouldMatch {
		p := strings.ReplaceAll(e, "X", sm)
		matches, err := m.matchesForJSONEvent([]byte(p))
		if err != nil {
			t.Error("bad JSON: " + err.Error())
		}
		if len(matches) != 1 || matches[0] != "AY" {
			t.Errorf("%s didn't match", sm)
		}
	}
}

func TestOverlappingValues(t *testing.T) {
	m := newCoreMatcher()
	p1 := `{"a": ["foo"]}`
	p2 := `{"a": ["football"]}`
	p3 := `{"a": ["footballer"]}`
	var err error
	var wantP1 X = "p1"
	err = m.addPattern(wantP1, p1)
	if err != nil {
		t.Error("Ouch p1")
	}
	var wantP2 X = "p2"
	err = m.addPattern(wantP2, p2)
	if err != nil {
		t.Error("Ouch p2")
	}
	var wantP3 X = "p3"
	err = m.addPattern(wantP3, p3)
	if err != nil {
		t.Error("Ouch p3")
	}
	e1 := `{"x": 3, "a": "foo"}`
	e2 := `{"x": 3, "a": "football"}`
	e3 := `{"x": 3, "a": "footballer"}`
	matches, err := m.matchesForJSONEvent([]byte(e1))
	if err != nil {
		t.Error("Error on e1: " + err.Error())
	}
	if len(matches) != 1 {
		t.Errorf("bad len %d", len(matches))
	} else if matches[0] != wantP1 {
		t.Errorf("Failure on e1 - want %v got %v", wantP1, matches[0])
	}

	matches, err = m.matchesForJSONEvent([]byte(e2))
	if err != nil {
		t.Error("Error on e2: " + err.Error())
	}
	if len(matches) != 1 || matches[0] != wantP2 {
		t.Error("Failure on e2")
	}

	matches, err = m.matchesForJSONEvent([]byte(e3))
	if err != nil {
		t.Error("Error on e3: " + err.Error())
	}
	if len(matches) != 1 || matches[0] != wantP3 {
		t.Error("Failure on e3")
	}
}

func TestFuzzValueMatcher(t *testing.T) {
	source := rand.NewSource(98543)

	m := newCoreMatcher()
	var pNames []X
	bytes := "abcdefghijklmnopqrstuvwxyz"
	lb := int64(len(bytes))
	strLen := 12
	used := make(map[X]bool)

	// make ten thousand 12-char strings, randomized
	for i := 0; i < 10000; i++ {
		str := ""
		for j := 0; j < strLen; j++ {
			//nolint:gosec
			ch := bytes[source.Int63()%lb]
			str += string([]byte{ch})
		}
		pNames = append(pNames, str)
		used[str] = true
	}

	// add a pattern for each string
	pBase := `{"a": [ "999" ]}`
	for _, pName := range pNames {
		err := m.addPattern(pName, strings.ReplaceAll(pBase, "999", pName.(string)))
		if err != nil {
			t.Errorf("addPattern %s: %s", pName, err.Error())
		}
	}

	// make sure all the patterns match
	eBase := `{"a": "999"}`
	for _, pName := range pNames {
		event := strings.ReplaceAll(eBase, "999", pName.(string))
		matches, err := m.matchesForJSONEvent([]byte(event))
		if err != nil {
			t.Errorf("m4J botch on %s: %s", event, err.Error())
		}
		if len(matches) != 1 {
			t.Errorf("M=%d for %s", len(matches), pName)
		} else {
			if matches[0] != pName {
				t.Errorf("wanted %s got %s", pName, matches[0])
			}
		}
	}

	// now let's run 1000 more random strings that shouldn't match
	shouldNot := 0
	for shouldNot < len(pNames) {
		str := ""
		for j := 0; j < strLen; j++ {
			//nolint:gosec
			ch := bytes[source.Int63()%lb]
			str += string([]byte{ch})
		}
		_, ok := used[str]
		if ok {
			continue
		}
		shouldNot++
		event := strings.ReplaceAll(eBase, "999", str)
		matches, err := m.matchesForJSONEvent([]byte(event))
		if err != nil {
			t.Errorf("shouldNot botch on %s: %s", event, err.Error())
		}
		if len(matches) != 0 {
			t.Errorf("OUCH %d matches to %s", len(matches), str)
		}
	}
}

func TestFuzzWithNumbers(t *testing.T) {
	source := rand.NewSource(98543)
	m := newCoreMatcher()
	var pNames []X
	used := make(map[X]bool)

	// make ten thousand random numbers between 1 and 100K. There are probably dupes?
	for i := 0; i < 10000; i++ {
		//nolint:gosec
		n := source.Int63()
		ns := fmt.Sprintf("%d", n)
		pNames = append(pNames, ns)
		used[ns] = true
	}

	// add a pattern for each number
	pBase := `{"a": [ 999 ]}`
	for _, pName := range pNames {
		err := m.addPattern(pName, strings.ReplaceAll(pBase, "999", pName.(string)))
		if err != nil {
			t.Errorf("addPattern %s: %s", pName, err.Error())
		}
	}

	// make sure all the patterns match
	eBase := `{"a": 999}`
	for _, pName := range pNames {
		event := strings.ReplaceAll(eBase, "999", pName.(string))
		matches, err := m.matchesForJSONEvent([]byte(event))
		if err != nil {
			t.Errorf("m4J botch on %s: %s", event, err.Error())
		}
		if len(matches) != 1 {
			t.Errorf("M=%d for %s", len(matches), pName)
		} else {
			if matches[0] != pName {
				t.Errorf("wanted %s got %s", pName, matches[0])
			}
		}
	}

	// now let's run 1000 more random numbers that shouldn't match
	shouldNot := 0
	for shouldNot < len(pNames) {
		//nolint:gosec
		n := rand.Int63n(1000000)
		ns := fmt.Sprintf("%d", n)
		_, ok := used[ns]
		if ok {
			continue
		}
		shouldNot++
		event := strings.ReplaceAll(eBase, "999", ns)
		// breaks on 98463
		// fmt.Println("Event: " + event)
		matches, err := m.matchesForJSONEvent([]byte(event))
		if err != nil {
			t.Errorf("shouldNot botch on %s: %s", event, err.Error())
		}
		if len(matches) != 0 {
			t.Errorf("OUCH %d matches to %s", len(matches), ns)
		}
	}
}

func TestMakeFAFragment(t *testing.T) {
	data := []string{"ca", "cat", "longer"}
	targetFA := &fieldMatcher{}
	targetState := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{targetFA}}
	pp := newPrettyPrinter(3234)
	for _, datum := range data {
		frag := makeFAFragment([]byte(datum), targetState, pp)
		var transIn []*fieldMatcher
		transOut := traverseDFA(frag, []byte(datum)[1:], transIn)
		if len(transOut) != 1 || transOut[0] != targetFA {
			t.Error("fail on ", datum)
		}
	}
}
func TestExerciseSingletonReplacement(t *testing.T) {
	cm := newCoreMatcher()
	err := cm.addPattern("x", `{"x": [ "a"]}`)
	if err != nil {
		t.Error("AP: " + err.Error())
	}
	err = cm.addPattern("x", `{"x": [1]}`)
	if err != nil {
		t.Error("AP: " + err.Error())
	}
	events := []string{`{"x": 1}`, `{"x": "a"}`}
	for _, e := range events {
		matches, err := cm.matchesForJSONEvent([]byte(e))
		if err != nil {
			t.Error("m4: " + err.Error())
		}
		if len(matches) != 1 || matches[0] != "x" {
			t.Error("match failed on: " + e)
		}
	}
	events = []string{`{"x": 1}`, `{"x": "a"}`}
	for _, e := range events {
		matches, err := cm.matchesForJSONEvent([]byte(e))
		if err != nil {
			t.Error("m4: " + err.Error())
		}
		if len(matches) != 1 || matches[0] != "x" {
			t.Error("match failed on: " + e)
		}
	}
	cm = newCoreMatcher()
	err = cm.addPattern("x", `{"x": ["x"]}`)
	if err != nil {
		t.Error("AP: " + err.Error())
	}
	err = cm.addPattern("x", `{"x": [ {"wildcard": "x*y"}]}`)
	if err != nil {
		t.Error("AP: " + err.Error())
	}
	events = []string{`{"x": "x"}`, `{"x": "x..y"}`}
	for _, e := range events {
		matches, err := cm.matchesForJSONEvent([]byte(e))
		if err != nil {
			t.Error("m4: " + err.Error())
		}
		if len(matches) != 1 || matches[0] != "x" {
			t.Error("match failed on: " + e)
		}
	}
}

func TestMergeNfaAndNumeric(t *testing.T) {
	cm := newCoreMatcher()
	err := cm.addPattern("x", `{"x": [{"wildcard":"x*y"}]}`)
	if err != nil {
		t.Error("AP: " + err.Error())
	}
	err = cm.addPattern("x", `{"x": [3]}`)
	if err != nil {
		t.Error("AP: " + err.Error())
	}
	events := []string{`{"x": 3}`, `{"x": "xasdfy"}`}
	for _, e := range events {
		matches, err := cm.matchesForJSONEvent([]byte(e))
		if err != nil {
			t.Error("M4: " + err.Error())
		}
		if len(matches) != 1 || matches[0] != "x" {
			t.Error("Match failed on " + e)
		}
	}
}

// TestEpsilonClosureAfterMerge verifies that when a deterministic pattern
// is merged into an NFA that already has epsilon transitions, the newly
// created splice states get their epsilon closures computed.
func TestEpsilonClosureAfterMerge(t *testing.T) {
	vm := newValueMatcher()

	// Add a wildcard pattern first - this sets isNondeterministic=true
	// and creates an NFA with epsilon transitions
	wildcardVal := typedVal{
		vType: wildcardType,
		val:   "a*b",
	}
	_ = vm.addTransition(wildcardVal, sharedNullPrinter, newClosureBuffers())

	fields := vm.fields()
	if !fields.isNondeterministic {
		t.Error("expected isNondeterministic=true after wildcard")
	}

	// Now add a simple string pattern - this will merge into the existing NFA
	// and create new splice states that need epsilon closure computation
	stringVal := typedVal{
		vType: stringType,
		val:   "xyz",
	}
	_ = vm.addTransition(stringVal, sharedNullPrinter, newClosureBuffers())

	fields = vm.fields()
	if !fields.isNondeterministic {
		t.Error("expected isNondeterministic=true to remain set")
	}

	// Walk the automaton and verify all states have epsilon closures computed
	visited := make(map[*faState]bool)
	missingClosures := checkEpsilonClosures(fields.start, visited)
	if len(missingClosures) > 0 {
		t.Errorf("found %d states with missing epsilon closures", len(missingClosures))
	}

	// Verify the matcher actually works
	bufs := newNfaBuffers()
	// Should match wildcard pattern "a*b"
	trans := testTransitionOn(vm, []byte("aXXXb"), bufs)
	if len(trans) != 1 {
		t.Errorf("expected 1 transition for 'aXXXb', got %d", len(trans))
	}
	// Should match string pattern "xyz"
	trans = testTransitionOn(vm, []byte("xyz"), bufs)
	if len(trans) != 1 {
		t.Errorf("expected 1 transition for 'xyz', got %d", len(trans))
	}
	// Should not match
	trans = testTransitionOn(vm, []byte("nomatch"), bufs)
	if len(trans) != 0 {
		t.Errorf("expected 0 transitions for 'nomatch', got %d", len(trans))
	}
}

// checkEpsilonClosures walks the automaton and returns states that have
// epsilon transitions but no computed epsilon closure.
func checkEpsilonClosures(start *faState, visited map[*faState]bool) []*faState {
	var missing []*faState
	if visited[start] {
		return missing
	}
	visited[start] = true

	for _, state := range start.table.steps {
		if state != nil {
			if state.epsilonClosure == nil {
				missing = append(missing, state)
			}
			missing = append(missing, checkEpsilonClosures(state, visited)...)
		}
	}
	for _, eps := range start.table.epsilons {
		if eps.epsilonClosure == nil {
			missing = append(missing, eps)
		}
		missing = append(missing, checkEpsilonClosures(eps, visited)...)
	}
	return missing
}

// TestEpsilonClosureRequired demonstrates that epsilonClosure must be called
// after merging into an NFA. The wildcard "a*z" has its start matching 'a', so
// the '*' spinner is an interior state, not the start. Merging it with the
// string "az" (which the wildcard also matches, with '*' matching the empty
// string) creates a splice whose accepting states are reachable only through a
// multi-member epsilon closure. Clearing the closures — simulating a skipped
// epsilonClosure call — therefore makes the "az" match disappear.
//
// Note: on this branch a self-only closure is the zero-length sentinel, so
// clearing a closure to nil reads as "self-only" (process self) rather than
// "no closure". The match must therefore be lost at a splice state that has no
// useful self-transition, which is what the "az" path through the merge gives.
func TestEpsilonClosureRequired(t *testing.T) {
	vm := newValueMatcher()

	// Add a wildcard pattern first - creates NFA with epsilon transitions
	_ = vm.addTransition(typedVal{vType: wildcardType, val: "a*z"}, sharedNullPrinter, newClosureBuffers())

	// Add a string pattern - this triggers merge and epsilonClosure call
	_ = vm.addTransition(typedVal{vType: stringType, val: "az"}, sharedNullPrinter, newClosureBuffers())

	bufs := newNfaBuffers()

	// "az" matches both patterns ('*' matching the empty string for "a*z"),
	// which requires traversing the merged splice's epsilon closure.
	if got := len(testTransitionOn(vm, []byte("az"), bufs)); got != 2 {
		t.Fatalf("with closures: expected 2 transitions for \"az\", got %d", got)
	}

	// Clear all closures to simulate a skipped epsilonClosure call. The match
	// disappears: reaching the accepting states required the multi-member
	// closure, which self-processing of the splice state cannot replace.
	clearEpsilonClosures(vm.fields().start, make(map[*faState]bool))
	if got := len(testTransitionOn(vm, []byte("az"), bufs)); got != 0 {
		t.Fatalf("without closures: expected 0 transitions for \"az\" (closure required), got %d", got)
	}

	// Restore closures; matching works again.
	epsilonClosure(vm.fields().start)
	if got := len(testTransitionOn(vm, []byte("az"), bufs)); got != 2 {
		t.Errorf("after restore: expected 2 transitions for \"az\", got %d", got)
	}
}

// clearEpsilonClosures walks the automaton and sets all epsilonClosure fields to nil
func clearEpsilonClosures(start *faState, visited map[*faState]bool) {
	if visited[start] {
		return
	}
	visited[start] = true
	start.epsilonClosure = nil

	for _, state := range start.table.steps {
		if state != nil {
			clearEpsilonClosures(state, visited)
		}
	}
	for _, eps := range start.table.epsilons {
		clearEpsilonClosures(eps, visited)
	}
}
