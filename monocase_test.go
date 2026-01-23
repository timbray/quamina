package quamina

import (
	"testing"
)

func TestABCDMono(t *testing.T) {
	permuteAndTest(t, "abcd", "ABCD")
}
func TestHungarianMono(t *testing.T) {
	orig := []rune{0x10C80, 0x10C9D, 0x10C95, 0x10C8B}
	alts := []rune{0x10CC0, 0x10CDD, 0x10CD5, 0x10CCB}
	permuteAndTest(t, string(orig), string(alts))
}
func TestIntermittentMono(t *testing.T) {
	permuteAndTest(t, "a,8899bc d", "A,8899BC D")
}

func permuteAndTest(t *testing.T, origS, altsS string) {
	t.Helper()
	orig := []byte(origS)
	alts := []byte(altsS)
	t.Helper()
	permutations := permuteCase(t, orig, alts, nil, 0, nil)
	pp := newPrettyPrinter(98987)
	fa, fm := makeMonocaseFA(orig, pp)

	for _, p := range permutations {
		ff := traverseDFAForTest(fa, p, nil)
		if len(ff) != 1 || ff[0] != fm {
			t.Error("FfFfAIL")
		}
	}
}
func permuteCase(t *testing.T, orig []byte, alts []byte, sofar []byte, index int, permutations [][]byte) [][]byte {
	t.Helper()
	if index == len(orig) {
		next := make([]byte, len(sofar))
		copy(next, sofar)
		permutations = append(permutations, next)
	} else {
		permutations = permuteCase(t, orig, alts, append(sofar, orig[index]), index+1, permutations)
		permutations = permuteCase(t, orig, alts, append(sofar, alts[index]), index+1, permutations)
	}
	return permutations
}

func TestSingletonMonocaseMerge(t *testing.T) {
	cm := newCoreMatcher()
	var err error
	err = cm.addPattern("singleton", `{"x": ["singleton"] }`)
	if err != nil {
		t.Error("add singleton: " + err.Error())
	}
	err = cm.addPattern("mono", `{"x": [ {"equals-ignore-case": "foo"}]}`)
	if err != nil {
		t.Error("add mono")
	}
	matches, _ := cm.matchesForJSONEvent([]byte(`{"x": "singleton"}`))
	if len(matches) != 1 && !containsX(matches, "singleton") {
		t.Error("singleton match failed")
	}
	matches, _ = cm.matchesForJSONEvent([]byte(`{"x": "FoO"}`))
	if len(matches) != 1 && !containsX(matches, "mono") {
		t.Error("singleton match failed")
	}
}

func TestEqualsIgnoreCaseMatching(t *testing.T) {
	rule1 := "{ \"a\" : [ { \"equals-ignore-case\": \"aBc\" } ] }"
	rule2 := "{ \"b\" : [ { \"equals-ignore-case\": \"XyZ\" } ] }"
	rule3 := "{ \"b\" : [ { \"equals-ignore-case\": \"xyZ\" } ] }"

	var err error
	cm := newCoreMatcher()
	err = cm.addPattern("r1", rule1)
	if err != nil {
		t.Error("AddPattern: " + err.Error())
	}
	err = cm.addPattern("r2", rule2)
	if err != nil {
		t.Error("AddPattern: " + err.Error())
	}
	err = cm.addPattern("r3", rule3)
	if err != nil {
		t.Error("AddPattern: " + err.Error())
	}
	matches, _ := cm.matchesForJSONEvent([]byte("{\"a\" : \"abc\"}"))
	if len(matches) != 1 || matches[0] != "r1" {
		t.Error("wrong on rule1")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"b\" : \"XYZ\"}"))
	if len(matches) != 2 || !containsX(matches, "r2", "r3") {
		t.Error("wrong on XYZ")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"a\" : \"AbC\"}"))
	if len(matches) != 1 || !containsX(matches, "r1") {
		t.Error("wrong on AbC")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"b\" : \"xyzz\"}"))
	if len(matches) != 0 {
		t.Error("wrong on xyzz")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"b\" : \"aabc\"}"))
	if len(matches) != 0 {
		t.Error("wrong on aabc")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"b\" : \"ABCXYZ\"}"))
	if len(matches) != 0 {
		t.Error("wrong on ABCXYZ")
	}
}
