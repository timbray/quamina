package quamina

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
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
func TestMakeShellStyleFA(t *testing.T) {
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
		a, wanted := makeShellStyleFA([]byte(pattern), sharedNullPrinter)
		vm := newValueMatcher()
		vmf := vmFields{startTable: a}
		vm.update(&vmf)
		var bufs bufpair
		for _, should := range shouldsForPatterns[i] {
			var transitions []*fieldMatcher
			gotTrans := traverseFA(a, []byte(should), transitions, &bufs)
			if len(gotTrans) != 1 || gotTrans[0] != wanted {
				t.Errorf("Failure for %s on %s", pattern, should)
			}
		}
		for _, shouldNot := range shouldNotForPatterns[i] {
			var transitions []*fieldMatcher
			gotTrans := traverseFA(a, []byte(shouldNot), transitions, &bufs)
			if gotTrans != nil {
				t.Errorf("bogus match for %s on %s", pattern, shouldNot)
			}
		}
	}
}

func TestWildCardRuler(t *testing.T) {
	rule1 := "{ \"a\" : [ { \"shellstyle\": \"*bc\" } ] }"
	rule2 := "{ \"b\" : [ { \"shellstyle\": \"d*f\" } ] }"
	rule3 := "{ \"b\" : [ { \"shellstyle\": \"d*ff\" } ] }"
	rule4 := "{ \"c\" : [ { \"shellstyle\": \"xy*\" } ] }"
	rule5 := "{ \"c\" : [ { \"shellstyle\": \"xy*\" } ] }"
	rule6 := "{ \"d\" : [ { \"shellstyle\": \"12*4*\" } ] }"

	cm := newCoreMatcher()
	_ = cm.addPattern("r1", rule1)
	_ = cm.addPattern("r2", rule2)
	_ = cm.addPattern("r3", rule3)
	_ = cm.addPattern("r4", rule4)
	_ = cm.addPattern("r5", rule5)
	_ = cm.addPattern("r6", rule6)

	var matches []X
	matches, _ = cm.matchesForJSONEvent([]byte("{\"a\" : \"bc\"}"))
	if len(matches) != 1 || matches[0] != "r1" {
		t.Error("Missed on r1")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"a\" : \"abc\"}"))
	if len(matches) != 1 || matches[0] != "r1" {
		t.Error("Missed on r1")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"b\" : \"dexef\"}"))
	if len(matches) != 1 || matches[0] != "r2" {
		t.Error("Missed on r2")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"b\" : \"dexeff\"}"))
	if len(matches) != 2 || (!containsX(matches, "r2")) || !containsX(matches, "r3") {
		t.Error("Missed on r2/r3")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"c\" : \"xyzzz\"}"))
	if len(matches) != 2 || (!containsX(matches, "r4")) || !containsX(matches, "r5") {
		t.Error("Missed on r4/r5")
	}
	matches, _ = cm.matchesForJSONEvent([]byte("{\"d\" : \"12345\"}"))
	if len(matches) != 1 || matches[0] != "r6" {
		t.Error("Missed on r6")
	}

	shouldNots := []string{
		"{\"c\" : \"abc\"}",
		"{\"a\" : \"xyz\"}",
		"{\"c\" : \"abcxyz\"}",
		"{\"b\" : \"ef\"}",
		"{\"b\" : \"de\"}",
		"{\"d\" : \"1235\"}",
	}
	for _, shouldNot := range shouldNots {
		matches, _ := cm.matchesForJSONEvent([]byte(shouldNot))
		if len(matches) != 0 {
			t.Error("shouldn't have matched: " + shouldNot)
		}
	}
}

func containsX(matches []X, wanteds ...string) bool {
	var sMatches []string
	for _, x := range matches {
		sMatches = append(sMatches, x.(string))
	}
	for _, wanted := range wanteds {
		for _, sMatch := range sMatches {
			if wanted == sMatch {
				return true
			}
		}
	}
	return false
}

func TestShellStyleBuildTime(t *testing.T) {
	words := readWWords(t)
	fmt.Printf("WC %d\n", len(words))
	starWords := make([]string, 0, len(words))
	patterns := make([]string, 0, len(words))
	source := rand.NewSource(293591)
	for _, word := range words {
		//nolint:gosec
		starAt := source.Int63() % 6
		starWord := string(word[:starAt]) + "*" + string(word[starAt:])
		starWords = append(starWords, starWord)
		pattern := fmt.Sprintf(`{"x": [ {"shellstyle": "%s" } ] }`, starWord)
		patterns = append(patterns, pattern)
	}
	q, _ := New()
	for i := range words {
		err := q.AddPattern(starWords[i], patterns[i])
		if err != nil {
			t.Error("AddP: " + err.Error())
		}
	}
	fmt.Println(matcherStats(q.matcher.(*coreMatcher)))
	// make sure that all the words actually are matched
	before := time.Now()
	for _, word := range words {
		record := fmt.Sprintf(`{"x": "%s"}`, word)
		matches, err := q.MatchesForEvent([]byte(record))
		if err != nil {
			t.Error("M4E on " + string(word))
		}
		if len(matches) == 0 {
			t.Error("no matches for " + string(word))
		}
		if len(matches) > 1 {
			fmt.Printf("%d matches for %s\n", len(matches), word)
		}
	}
	elapsed := float64(time.Since(before).Milliseconds())
	eps := float64(len(words)) / (elapsed / 1000.0)
	fmt.Printf("Huge-machine events/sec: %.1f\n", eps)
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
	fmt.Println("M: " + matcherStats(m))

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
