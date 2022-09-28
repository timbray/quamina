package quamina

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func TestLongCase(t *testing.T) {
	m := newCoreMatcher()
	pat := `{"x": [ {"shellstyle": "*abab"} ] }`
	err := m.addPattern("x", pat)
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
		matches, err := m.matchesForJSONEvent([]byte(event))
		if err != nil {
			t.Error("m4j " + err.Error())
		}
		if len(matches) != 1 {
			t.Error("MISSED: " + should)
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

	// NOTE also testing nfa2Dfa
	for i, pattern := range patterns {
		myNext := newFieldMatcher()
		a, wanted := makeShellStyleAutomaton([]byte(pattern), myNext)
		if wanted != myNext {
			t.Error("bad next on: " + pattern)
		}
		d := nfa2Dfa(a)
		vm := newValueMatcher()
		vmf := vmFields{startDfa: d}
		vm.update(&vmf)
		for _, should := range shouldsForPatterns[i] {
			var transitions []*fieldMatcher
			gotTrans := transitionDfa(d, []byte(should), transitions)
			if len(gotTrans) != 1 || gotTrans[0] != wanted {
				t.Errorf("Failure for %s on %s", pattern, should)
			}
		}
		for _, shouldNot := range shouldNotForPatterns[i] {
			var transitions []*fieldMatcher
			gotTrans := transitionDfa(d, []byte(shouldNot), transitions)
			if gotTrans != nil {
				t.Errorf("bogus DFA match for %s on %s", pattern, shouldNot)
			}
		}
	}
}

func TestShellStyleBuildTime(t *testing.T) {
	words := readWWords(t)
	starWords := make([]string, 0, len(words))
	patterns := make([]string, 0, len(words))
	for _, word := range words {
		//nolint:gosec
		starAt := rand.Int31n(6)
		starWord := string(word[:starAt]) + "*" + string(word[starAt:])
		starWords = append(starWords, starWord)
		pattern := fmt.Sprintf(`{"x": [ {"shellstyle": "%s" } ] }`, starWord)
		patterns = append(patterns, pattern)
	}
	q, _ := New()
	for i := 0; i < 32; i++ {
		// fmt.Printf("i=%d w=%s: %s\n", i, starWords[i], matcherStats(q.matcher.(*coreMatcher)))
		// fmt.Println(patterns[i])
		err := q.AddPattern(starWords[i], patterns[i])
		if err != nil {
			t.Error("AddP: " + err.Error())
		}
	}
	fmt.Println(matcherStats(q.matcher.(*coreMatcher)))
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

	stringTemplate := `{"properties": { "STREET": [ XX ] } }`
	shellTemplate := `{"properties": {"STREET":[ {"shellstyle": XX} ] } }`
	m := newCoreMatcher()
	for name := range x {
		var pat string
		if strings.Contains(name, "*") {
			pat = strings.ReplaceAll(shellTemplate, "XX", name)
		} else {
			pat = strings.ReplaceAll(stringTemplate, "XX", name)
		}

		err := m.addPattern(name, pat)
		if err != nil {
			t.Error("addPattern: " + name + ", prob=" + err.Error())
		}
	}
	got := make(map[X]int)
	lines := getCityLotsLines(t)
	for _, line := range lines {
		matches, err := m.matchesForJSONEvent(line)
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
