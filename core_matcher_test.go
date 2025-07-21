package quamina

import (
	"fmt"
	"testing"
)

func TestBasicMatching(t *testing.T) {
	var x X = "testing"
	pattern := `{"a": [1, 2], "b": [1, "3"]}`
	m := newCoreMatcher()
	err := m.addPattern(x, pattern)
	if err != nil {
		t.Error(err.Error())
	}
	shouldMatch := []string{
		`{"b": "3", "a": 1}`,
		`{"a": 2, "b": "3", "x": 33}`,
	}
	shouldNotMatch := []string{
		`{"b": "3", "a": 6}`,
		`{"a": 2}`,
		`{"b": "3"}`,
	}
	for _, should := range shouldMatch {
		var matches []X
		matches, err = m.matchesForJSONEvent([]byte(should))
		if err != nil {
			t.Error(err.Error())
		}
		if len(matches) != 1 {
			t.Errorf("event %s, LM %d", should, len(matches))
		}
	}
	for _, shouldNot := range shouldNotMatch {
		var matches []X
		matches, _ = m.matchesForJSONEvent([]byte(shouldNot))
		if len(matches) != 0 {
			t.Error("Matched: " + shouldNot)
		}
	}
}

// thanks to @kylemcc
func TestExistsFalseOrder(t *testing.T) {
	j := `{"aField": "a","bField": "b",	"cField": "c"}`

	// make sure exists:false properly disqualifies a match regardless of where
	// it occurs (lexicographically) in the pattern
	shouldNotPatterns := []string{
		`{"aField": ["a"], "bField": [{ "exists": false }], "cField": ["c"]}`,
		`{"aField": [{ "exists": false }], "bField": ["b"], "cField": ["c"]}`,
		`{"aField": ["a"], "bField": ["b"], "cField": [{ "exists": false }]}`,
	}
	matchesForSNPatterns := []string{
		`{ "$field": "$", "aField": "a", "cField": "c" }`,
		`{ "$field": "$", "bField": "b", "cField": "c" }`,
		`{ "$field": "$", "aField": "a", "bField": "b" }`,
	}

	for i, shouldNot := range shouldNotPatterns {
		m := newCoreMatcher()
		err := m.addPattern(fmt.Sprintf("should NOT %d", i), shouldNot)
		if err != nil {
			t.Error("addPattern: " + shouldNot + ": " + err.Error())
		}
		// fmt.Println("Try to match: " + j + " to " + shouldNot)
		matches, err := m.matchesForJSONEvent([]byte(j))
		if err != nil {
			t.Error("ShouldNot " + shouldNot + ": " + err.Error())
		}
		if len(matches) != 0 {
			t.Errorf("YES p=%s e=%s", shouldNot, j)
		}

		// fmt.Println("Try to match: " + matchesForSNPatterns[i] + " to " + shouldNot)
		matches, err = m.matchesForJSONEvent([]byte(matchesForSNPatterns[i]))
		if err != nil {
			t.Error("matchesForSNPatterns: + ", err.Error())
		}
		if len(matches) == 0 {
			t.Errorf("NO p=%s e=%s", shouldNot, matchesForSNPatterns[i])
		}
	}

	mm := newCoreMatcher()
	for _, pattern := range shouldNotPatterns {
		err := mm.addPattern(pattern, pattern)
		if err != nil {
			t.Error("AddP: " + err.Error())
		}
	}
	matches, err := mm.matchesForJSONEvent([]byte(j))
	if err != nil {
		t.Error("match: " + err.Error())
	}
	if len(matches) != 0 {
		msg := fmt.Sprintf("all patterns, too many matches (%d) for %s\n", len(matches), j)
		for _, match := range matches {
			msg += fmt.Sprintf(" %s\n", match)
		}
		t.Error(msg)
	}
}

func TestFieldNameOrdering(t *testing.T) {
	j := `{
		"b": 1
      }`
	patterns := []string{
		`{ "b": [1], "a": [ { "exists":false } ] }"`,
		`{ "b": [1], "c": [ { "exists":false } ] }"`,
		`{ "b": [1]}"`,
		`{ "a": [ { "exists":false } ] }"`,
	}
	wanted := make(map[string]int)
	for _, pattern := range patterns {
		wanted[pattern] = 0
	}
	m := newCoreMatcher()
	for _, pattern := range patterns {
		err := m.addPattern(pattern, pattern)
		if err != nil {
			t.Error("addPattern: " + err.Error())
		}
	}
	matches, err := m.matchesForJSONEvent([]byte(j))
	if err != nil {
		t.Error("M4J: " + err.Error())
	}
	for _, match := range matches {
		smatch := match.(string)
		wanted[smatch]++
	}
	for want, count := range wanted {
		if count != 1 {
			t.Error("missed: " + want)
		}
	}
}

func TestSuffixBug(t *testing.T) {
	var err error
	j := `{"Url":    "xy9"}`
	patterns := []string{
		`{ "Url": [ { "shellstyle": "*9" } ] }`,
		`{ "Url": [ { "shellstyle": "x*9" } ] }`,
	}

	// make sure each works individually
	m := newCoreMatcher()
	_ = m.addPattern("p0", patterns[0])
	matches, _ := m.matchesForJSONEvent([]byte(j))
	if len(matches) != 1 || matches[0] != "p0" {
		t.Error("p0 didn't match")
	}

	m = newCoreMatcher()
	_ = m.addPattern("p1", patterns[1])
	matches, _ = m.matchesForJSONEvent([]byte(j))
	if len(matches) != 1 || matches[0] != "p1" {
		t.Error("p1 didn't match")
	}

	// now let's see if they work merged
	m = newCoreMatcher()
	wanted := make(map[X]int)
	for _, should := range patterns {
		wanted[should] = 0
		err = m.addPattern(should, should)
		if err != nil {
			t.Error("add one of many: " + err.Error())
		}
	}
	matches, err = m.matchesForJSONEvent([]byte(j))
	if err != nil {
		t.Error("m4J on all: " + err.Error())
	}
	if len(matches) != len(patterns) {
		for _, match := range matches {
			wanted[match]++
		}
		for want := range wanted {
			if wanted[want] == 0 {
				t.Errorf("Missed: %s", want.(string))
			} else {
				fmt.Printf("Matched %s\n", want)
			}
		}
		fmt.Println()
	}
}

func TestExerciseMatching(t *testing.T) {
	j := `{
        "Image": {
            "Width":  800,
            "Height": 600,
            "Title":  "View from 15th Floor",
            "Thumbnail": {
                "Url":    "https://www.example.com/image/481989943",
                "Height": 125,
                "Width":  100
            },
            "Animated" : false,
            "IDs": [116, 943, 234, 38793]
          }
      }`
	patternsFromReadme := []string{
		`{"Image": { "Title": [ { "exists": true } ] } }`,
		`{"Foo": [ { "exists": false } ] }"`,
		`{"Image": {"Width": [800]}}`,
		`{"Image": { "Animated": [ false], "Thumbnail": { "Height": [ 125 ] } } }}, "IDs": [943]}`,
		`{"Image": { "Width": [800], "Title": [ { "exists": true } ], "Animated": [ false ] } }`,
		`{"Image": { "Width": [800], "IDs": [ { "exists": true } ] } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "*9943" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "https://www.example.com/*" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "https://www.example.com/*9943" } ] } } }`,
		`{"Image": { "Title": [ {"anything-but":  ["Pikachu", "Eevee"] } ]  } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "prefix": "https:" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ "a", { "prefix": "https:" } ] } } }`,
		`{"Image": { "Title": [ { "equals-ignore-case": "VIEW FROM 15th FLOOR" } ] } }`,
		`{"Image": { "Title": [ { "regexp": "View from .... Floor" } ]  } }`,
		`{"Image": { "Title": [ { "regexp": "View from [0-9][0-9][rtn][dh] Floor" } ]  } }`,
		`{"Image": { "Title": [ { "regexp": "View from 15th (Floor|Storey)" } ]  } }`,
	}

	var err error
	blankMatcher := newCoreMatcher()
	empty, err := blankMatcher.matchesForJSONEvent([]byte(j))
	if err != nil {
		t.Error("blank: " + err.Error())
	}
	if len(empty) != 0 {
		t.Error("matches on blank matcher")
	}

	for i, should := range patternsFromReadme {
		m := newCoreMatcher()
		err = m.addPattern(fmt.Sprintf("should %d", i), should)
		if err != nil {
			t.Error("addPattern " + should + ": " + err.Error())
		}
		matches, err := m.matchesForJSONEvent([]byte(j))
		if err != nil {
			t.Error("M4J: " + err.Error())
		}
		if len(matches) != 1 {
			t.Errorf("Matches %s Length %d", should, len(matches))
		}
	}

	shouldNotMatches := []string{
		`{"Image": { "Animated": [ { "exists": false } ] } }`,
		`{"Image": { "NotThere": [ { "exists": true } ] } }`,
		`{"Image": { "IDs": [ { "exists": false } ], "Animated": [ false ] } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "prefix": "http:" } ] } } }`,
	}
	for i, shouldNot := range shouldNotMatches {
		m := newCoreMatcher()
		err = m.addPattern(fmt.Sprintf("should NOT %d", i), shouldNot)
		if err != nil {
			t.Error("addPattern: " + shouldNot + ": " + err.Error())
		}
		matches, err := m.matchesForJSONEvent([]byte(j))
		if err != nil {
			t.Error("ShouldNot " + shouldNot + ": " + err.Error())
		}
		if len(matches) != 0 {
			t.Error(shouldNot + " matched but shouldn't have")
		}
	}
	// now add them all
	m := newCoreMatcher()
	wanted := make(map[X]int)
	for _, should := range patternsFromReadme {
		wanted[should] = 0
		err = m.addPattern(should, should)
		if err != nil {
			t.Error("add one of many: " + err.Error())
		}
	}
	fmt.Println("MS: " + matcherStats(m))
	matches, err := m.matchesForJSONEvent([]byte(j))
	if err != nil {
		t.Error("m4J on all: " + err.Error())
	}
	if len(matches) != len(patternsFromReadme) {
		for _, match := range matches {
			wanted[match]++
		}
		for want := range wanted {
			if wanted[want] == 0 {
				t.Errorf("Missed: %s", want.(string))
			}
		}
		fmt.Println()
	}
}

func TestTacos(t *testing.T) {
	pat := `{"like":["tacos","queso"],"want":[0]}`
	m := newCoreMatcher()
	err := m.addPattern(pat, pat)
	if err != nil {
		t.Error("Tacos: " + err.Error())
	}
}

func TestSimpleaddPattern(t *testing.T) {
	// laboriously hand-check the simplest possible automaton
	var x X = "testing"
	pattern := `{"a": [1, 2], "b": [1, "3"]}`
	m := newCoreMatcher()
	err := m.addPattern(x, pattern)
	if err != nil {
		t.Error(err.Error())
	}
	s0 := m.fields().state
	if len(s0.fields().transitions) != 1 {
		t.Errorf("s0 trans len %d", len(s0.fields().transitions))
	}

	_, ok := s0.fields().transitions["a"]
	if !ok {
		t.Error("No trans from start on 'a'")
	}
}

// a lot of tests add and test patterns through the top-level coreMatcher interfaces,
// which means the finite automata are hidden deep inside the coreMatcher instance
// and hard to get at.  This helper routine fetches the value-matcher automaton
// corresponding to the "path" argument
func fetchFAForPath(t *testing.T, cm *coreMatcher, path string) *smallTable {
	t.Helper()
	vm := cm.fields().state.fields().transitions[path]
	return vm.fields().startTable
}
