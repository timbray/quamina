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
		`{"Foo": [ { "exists": false } ] }"`,
		`{"Image": {"Width": [800]}}`,
		`{"Image": { "Animated": [ false], "Thumbnail": { "Height": [ 125 ] } } }}, "IDs": [943]}`,
		`{"Image": { "Title": [ { "exists": true } ] } }`,
		`{"Image": { "Width": [800], "Title": [ { "exists": true } ], "Animated": [ false ] } }`,
		`{"Image": { "Width": [800], "IDs": [ { "exists": true } ] } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "*9943" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "https://www.example.com/*" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "https://www.example.com/*9943" } ] } } }`,
		`{"Image": { "Title": [ {"anything-but":  ["Pikachu", "Eevee"] } ]  } }`,
	}

	var err error
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
	for _, should := range patternsFromReadme {
		err = m.addPattern(should, should)
		if err != nil {
			t.Error("add one of many: " + err.Error())
		}
	}
	matches, err := m.matchesForJSONEvent([]byte(j))
	if err != nil {
		t.Error("m4J on all: " + err.Error())
	}
	if len(matches) != len(patternsFromReadme) {
		t.Errorf("on mix wanted %d got %d", len(patternsFromReadme), len(matches))
	}
	fmt.Println(matcherStats(m))
}

func TestExistsFalseOrder(t *testing.T) {
	j := `{
		"aField": "a",
		"bField": "b",
		"cField": "c"
	}`

	// make sure exists:false properly disqualifies a match regardless of where
	// it occurs (lexicographically) in the pattern
	shouldNotMatches := []string{
		`{ "aField": [ { "exists": false } ], "bField": [ "b" ], "cField": [ "c" ] }`,
		`{ "aField": [ "a" ], "bField": [ { "exists": false } ], "cField": [ "c" ] }`,
		`{ "aField": [ "a" ], "bField": [ "b" ], "cField": [ { "exists": false } ] }`,
	}

	for i, shouldNot := range shouldNotMatches {
		m := newCoreMatcher()
		err := m.addPattern(fmt.Sprintf("should NOT %d", i), shouldNot)
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
	if len(m.start().namesUsed) != 2 {
		t.Errorf("nameUsed = %d", len(m.start().namesUsed))
	}
	if !m.IsNameUsed([]byte("a")) {
		t.Error("'a' not showing as used")
	}
	if !m.IsNameUsed([]byte("b")) {
		t.Error("'b' not showing as used")
	}
	s0 := m.start().state
	if len(s0.fields().transitions) != 1 {
		t.Errorf("s0 trans len %d", len(s0.fields().transitions))
	}

	_, ok := s0.fields().transitions["a"]
	if !ok {
		t.Error("No trans from start on 'a'")
	}
}
