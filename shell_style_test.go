package quamina

import (
	"fmt"
	"strings"
	"testing"
)

func TestLongCase(t *testing.T) {
	m := NewCoreMatcher()
	pat := `{"x": [ {"shellstyle": "*abab"} ] }`
	err := m.AddPattern("x", pat)
	if err != nil {
		t.Error("addPat? " + err.Error())
	}
	shoulds := []string{
		"abaabab",
		"ababab",
		"ababaabab",
	}
	for _, should := range shoulds {
		event := fmt.Sprintf(`{"x": "%s"}`, should)
		matches, err := m.MatchesForJSONEvent([]byte(event))
		if err != nil {
			t.Error("m4j " + err.Error())
		}
		if len(matches) != 1 {
			t.Error("MISSED: " + should)
		}
	}
}
func newNfaWithStart(start *smallTable[*nfaStepList]) *valueMatcher {
	vm := newValueMatcher()
	state := &vmFields{startNfa: start}
	vm.update(state)
	return vm
}
func TestNfaMerging(t *testing.T) {
	aMatches := []string{
		`"Afoo"`,
		`"ABA"`,
	}
	bMatches := []string{
		`"BAB"`,
		`"Bbar"`,
	}
	var f1 = &fieldMatcher{}
	var f2 = &fieldMatcher{}
	nfa1, _ := makeShellStyleAutomaton([]byte(`"A*"`), f1)
	nfa2, _ := makeShellStyleAutomaton([]byte(`"B*"`), f2)

	v1 := newNfaWithStart(nfa1)
	v2 := newNfaWithStart(nfa2)

	for _, aMatch := range aMatches {
		t1 := v1.transitionOn([]byte(aMatch))
		if len(t1) != 1 || t1[0] != f1 {
			t.Error("mismatch on " + aMatch)
		}
	}
	for _, bMatch := range bMatches {
		t1 := v2.transitionOn([]byte(bMatch))
		if len(t1) != 1 || t1[0] != f2 {
			t.Error("mismatch on " + bMatch)
		}
	}

	combo := mergeNfas(nfa1, nfa2)
	v3 := newNfaWithStart(combo)
	ab := append(aMatches, bMatches...)
	for _, match := range ab {
		t3 := v3.transitionOn([]byte(match))
		if len(t3) != 1 {
			t.Error("Fail on " + match)
		}
	}

}

func TestMakeShellStyleAutomaton(t *testing.T) {
	patterns := []string{
		`"*ST"`,
		`"foo*"`,
		`"*foo"`,
		`"*foo*"`,
		`"xx*yy*zz"`,
		`"*xx*yy*"`,
	}
	shouldsForPatterns := [][]string{
		{`"STA ST"`, `"1ST"`},
		{`"fooabc"`, `"foo"`},
		{`"afoo"`, `"foo"`},
		{`"xxfooyy"`, `"fooyy"`, `"xxfoo"`, `"foo"`},
		{`"xxabyycdzz"`, `"xxyycdzz"`, `"xxabyyzz"`, `"xxyyzz"`},
		{`"abxxcdyyef"`, `"xxcdyyef"`, `"abxxyyef"`, `"abxxcdyy"`, `"abxxyy"`, `"xxcdyy"`, `"xxyyef"`, `"xxyy"`},
	}
	shouldNotForPatterns := [][]string{
		{`"STA"`, `"STAST "`},
		{`"afoo"`, `"fofo"`},
		{`"foox"`, `"afooo"`},
		{`"afoa"`, `"fofofoxooxoo"`},
		{`"xyzyxzy yy zz"`, `"zz yy xx"`},
		{`"ayybyyzxx"`},
	}

	for i, pattern := range patterns {
		myNext := newFieldMatcher()
		a, wanted := makeShellStyleAutomaton([]byte(pattern), myNext)
		if wanted != myNext {
			t.Error("bad next on: " + pattern)
		}
		for _, should := range shouldsForPatterns[i] {
			var transitions []*fieldMatcher
			gotTrans := oneNfaStep(a, 0, []byte(should), transitions)
			if len(gotTrans) != 1 || gotTrans[0] != wanted {
				t.Errorf("Failure for %s on %s", pattern, should)
			}
		}
		for _, shouldNot := range shouldNotForPatterns[i] {
			var transitions []*fieldMatcher
			gotTrans := oneNfaStep(a, 0, []byte(shouldNot), transitions)
			if gotTrans != nil {
				t.Errorf("bogus match for %s on %s", pattern, shouldNot)
			}
		}
	}
}

func TestMixedPatterns(t *testing.T) {
	// let's mix up some prefix, infix, suffix, and exact-match searches
	x := map[string]int{
		`"*ST"`:     5754,
		`"*TH"`:     34310,
		`"B*K"`:     746,
		`"C*L"`:     1022,
		`"CH*"`:     2226,
		`"Z*"`:      25,
		`"BANNOCK"`: 22,
		`"21ST"`:    1370,
		`"ZOE"`:     19,
		`"CRYSTAL"`: 6,
	}
	x1, _ := makeShellStyleAutomaton([]byte(`"*ST"`), nil)
	x2, _ := makeShellStyleAutomaton([]byte(`"*TH"`), nil)
	mergeNfas(x1, x2)

	stringTemplate := `{"properties": { "STREET": [ XX ] } }`
	shellTemplate := `{"properties": {"STREET":[ {"shellstyle": XX} ] } }`
	m := NewCoreMatcher()
	for name := range x {
		var pat string
		if strings.Contains(name, "*") {
			pat = strings.ReplaceAll(shellTemplate, "XX", name)
		} else {
			pat = strings.ReplaceAll(stringTemplate, "XX", name)
		}

		err := m.AddPattern(name, pat)
		if err != nil {
			t.Error("addPattern: " + name + ", prob=" + err.Error())
		}
	}
	got := make(map[X]int)
	lines := getCityLotsLines(t)
	for _, line := range lines {
		matches, err := m.MatchesForJSONEvent(line)
		if err != nil {
			t.Error("Matches4JSON: " + err.Error())
		}
		for _, match := range matches {
			count, ok := got[match]
			if !ok {
				got[match] = 1
			} else {
				got[match] = count + 1
			}
		}
	}
	for match, count := range got {
		sm := match.(string)
		if x[sm] != count {
			t.Errorf("For %s wanted %d got %d", sm, x[sm], count)
		}

	}
}
