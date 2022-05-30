package core

import (
	"fmt"
	"github.com/timbray/quamina/flattener"
	"math/rand"
	"strings"
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

/* - restore this one when we get multi-glob working
func TestMultiTransitions(t *testing.T) {
	patX := `{"foo": [ { "shellstyle": "*x*b" } ]}`
	patY := `{"foo": [ { "shellstyle": "*y*b" } ]}`

	m := NewCoreMatcher()
	if m.AddPattern("X", patX) != nil {
		t.Error("add patX")
	}
	if m.AddPattern("Y", patY) != nil {
		t.Error("add patY")
	}
	e := `{"foo": "axyb"}`
	matches, err := m.MatchesForJSONEvent([]byte(e))
	if err != nil {
		t.Error("m4: " + err.Error())
	}
	if len(matches) != 2 {
		t.Error("just one")
	}
}

func TestAY(t *testing.T) {
	m := NewCoreMatcher()
	pat := `{"x": [ { "shellstyle": "*ay*"} ] }`
	err := m.AddPattern("AY", pat)
	if err != nil {
		t.Error("AY: " + err.Error())
	}
	shouldMatch := []string{"ay", "aay", "aaaayyyyy", "xyzay", "ayxxxx"}
	e := `{"x": "X"}`
	for _, sm := range shouldMatch {
		p := strings.ReplaceAll(e, "X", sm)
		matches, err := m.MatchesForJSONEvent([]byte(p))
		if err != nil {
			t.Error("bad JSON: " + err.Error())
		}
		if len(matches) != 1 || matches[0] != "AY" {
			t.Errorf("%s didn't match", sm)
		}
	}
}
*/

func TestOverlappingValues(t *testing.T) {
	m := NewCoreMatcher()
	p1 := `{"a": ["foo"]}`
	p2 := `{"a": ["football"]}`
	p3 := `{"a": ["footballer"]}`
	var err error
	var wantP1 X = "p1"
	err = m.AddPattern(wantP1, p1)
	if err != nil {
		t.Error("Ouch p1")
	}
	var wantP2 X = "p2"
	err = m.AddPattern(wantP2, p2)
	if err != nil {
		t.Error("Ouch p2")
	}
	var wantP3 X = "p3"
	err = m.AddPattern(wantP3, p3)
	if err != nil {
		t.Error("Ouch p3")
	}
	e1 := `{"x": 3, "a": "foo"}`
	e2 := `{"x": 3, "a": "football"}`
	e3 := `{"x": 3, "a": "footballer"}`
	fj := flattener.NewFJ()
	fields, err := fj.Flatten([]byte(e1), m)
	if err != nil {
		t.Error("Flatten: " + err.Error())
	}
	matches, err := m.MatchesForFields(fields)
	if err != nil {
		t.Error("Error on e1: " + err.Error())
	}
	if len(matches) != 1 {
		t.Errorf("bad len %d", len(matches))
	} else if matches[0] != wantP1 {
		t.Errorf("Failure on e1 - want %v got %v", wantP1, matches[0])
	}

	fields, err = fj.Flatten([]byte(e2), m)
	if err != nil {
		t.Error("Flatten: " + err.Error())
	}
	matches, err = m.MatchesForFields(fields)
	if err != nil {
		t.Error("Error on e2: " + err.Error())
	}
	if len(matches) != 1 || matches[0] != wantP2 {
		t.Error("Failure on e2")
	}

	fields, err = fj.Flatten([]byte(e3), m)
	if err != nil {
		t.Error("Flatten: " + err.Error())
	}
	matches, err = m.MatchesForFields(fields)
	if err != nil {
		t.Error("Error on e3: " + err.Error())
	}
	if len(matches) != 1 || matches[0] != wantP3 {
		t.Error("Failure on e3")
	}
}

func TestFuzzValueMatcher(t *testing.T) {
	rand.Seed(98543)
	m := NewCoreMatcher()
	var pNames []X
	bytes := "abcdefghijklmnopqrstuvwxyz"
	lb := len(bytes)
	strLen := 12
	used := make(map[X]bool)

	// make ten thousand 12-char strings, randomized
	for i := 0; i < 10000; i++ {
		str := ""
		for j := 0; j < strLen; j++ {
			ch := bytes[rand.Int()%lb]
			str += string([]byte{ch})
		}
		pNames = append(pNames, str)
		used[str] = true
	}

	// add a pattern for each string
	pBase := `{"a": [ "999" ]}`
	for _, pName := range pNames {
		err := m.AddPattern(pName, strings.ReplaceAll(pBase, "999", pName.(string)))
		if err != nil {
			t.Errorf("addPattern %s: %s", pName, err.Error())
		}
	}

	// make sure all the patterns match
	eBase := `{"a": "999"}`
	fj := flattener.NewFJ()
	for _, pName := range pNames {
		event := strings.ReplaceAll(eBase, "999", pName.(string))
		fields, err := fj.Flatten([]byte(event), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err := m.MatchesForFields(fields)
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
	fj = flattener.NewFJ()
	for shouldNot < len(pNames) {
		str := ""
		for j := 0; j < strLen; j++ {
			ch := bytes[rand.Int()%lb]
			str += string([]byte{ch})
		}
		_, ok := used[str]
		if ok {
			continue
		}
		shouldNot++
		event := strings.ReplaceAll(eBase, "999", str)
		fields, err := fj.Flatten([]byte(event), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err := m.MatchesForFields(fields)
		if err != nil {
			t.Errorf("shouldNot botch on %s: %s", event, err.Error())
		}
		if len(matches) != 0 {
			t.Errorf("OUCH %d matches to %s", len(matches), str)
		}
	}
}

func TestFuzzWithNumbers(t *testing.T) {
	rand.Seed(98543)
	m := NewCoreMatcher()
	var pNames []X
	used := make(map[X]bool)

	// make ten thousand random numbers between 1 and 100K. There are probably dupes?
	for i := 0; i < 10000; i++ {
		n := rand.Int63n(1000000)
		ns := fmt.Sprintf("%d", n)
		pNames = append(pNames, ns)
		used[ns] = true
	}

	// add a pattern for each number
	pBase := `{"a": [ 999 ]}`
	for _, pName := range pNames {
		err := m.AddPattern(pName, strings.ReplaceAll(pBase, "999", pName.(string)))
		if err != nil {
			t.Errorf("addPattern %s: %s", pName, err.Error())
		}
	}

	// make sure all the patterns match
	eBase := `{"a": 999}`
	fj := flattener.NewFJ()
	for _, pName := range pNames {
		event := strings.ReplaceAll(eBase, "999", pName.(string))
		fields, err := fj.Flatten([]byte(event), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err := m.MatchesForFields(fields)
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
		fields, err := fj.Flatten([]byte(event), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err := m.MatchesForFields(fields)
		if err != nil {
			t.Errorf("shouldNot botch on %s: %s", event, err.Error())
		}
		if len(matches) != 0 {
			t.Errorf("OUCH %d matches to %s", len(matches), ns)
		}
	}
}

func contains(list []*fieldMatcher, s *fieldMatcher) bool {
	for _, l := range list {
		if l == s {
			return true
		}
	}
	return false
}
