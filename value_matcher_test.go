package quamina

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

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
		m.addTransition(addBefore, &nullPrinter{})
	}
	m.addTransition(invalidField, &nullPrinter{})
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
	t1 := m.addTransition(v1, &nullPrinter{})
	if t1 == nil {
		t.Error("nil addTrans")
	}
	t1x := m.transitionOn(&Field{Val: []byte("one")}, &nfaBuffers{})
	if len(t1x) != 1 || t1x[0] != t1 {
		t.Error("Retrieve failed")
	}

	tXtra := m.addTransition(v1, &nullPrinter{})
	if tXtra != t1 {
		t.Error("dupe trans missed")
	}

	v2 := typedVal{
		vType: stringType,
		val:   "two",
	}
	t2 := m.addTransition(v2, &nullPrinter{})

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
	t3 := m.addTransition(v3, &nullPrinter{})
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
		startTable := frag.table
		var transIn []*fieldMatcher
		transOut := traverseDFAForTest(startTable, []byte(datum)[1:], transIn)
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
