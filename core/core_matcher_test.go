package core

import (
	"fmt"
	"github.com/timbray/quamina/flattener"
	"testing"
)

func TestBasicMatching(t *testing.T) {
	var x X = "testing"
	pattern := `{"a": [1, 2], "b": [1, "3"]}`
	m := NewCoreMatcher()
	err := m.AddPattern(x, pattern)
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
	fj := flattener.NewFJ()
	for _, should := range shouldMatch {
		var matches []X
		fields, err := fj.Flatten([]byte(should), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err = m.MatchesForFields(fields)
		if err != nil {
			t.Error(err.Error())
		}
		if len(matches) != 1 {
			t.Errorf("event %s, LM %d", should, len(matches))
		}
	}
	for _, shouldNot := range shouldNotMatch {
		var matches []X
		fields, err := fj.Flatten([]byte(shouldNot), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err = m.MatchesForFields(fields)
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
	shouldMatches := []string{
		`{"Foo": [ { "exists": false } ] }"`,
		`{"Image": {"Width": [800]}}`,
		`{"Image": { "Animated": [ false], "Thumbnail": { "Height": [ 125 ] } } }}, "IDs": [943]}`,
		`{"Image": { "Title": [ { "exists": true } ] } }`,
		`{"Image": { "Width": [800], "Title": [ { "exists": true } ], "Animated": [ false ] } }`,
		`{"Image": { "Width": [800], "IDs": [ { "exists": true } ] } }`,
		//`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "https://*.example.com/*" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "*9943" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "https://www.example.com/*" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "shellstyle": "https://www.example.com/*9943" } ] } } }`,
	}

	var err error
	for i, should := range shouldMatches {
		m := NewCoreMatcher()
		err = m.AddPattern(fmt.Sprintf("should %d", i), should)
		if err != nil {
			t.Error("addPattern " + should + ": " + err.Error())
		}
		fj := flattener.NewFJ()
		fields, err := fj.Flatten([]byte(j), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err := m.MatchesForFields(fields)
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
	fj := flattener.NewFJ()
	for i, shouldNot := range shouldNotMatches {
		m := NewCoreMatcher()
		err = m.AddPattern(fmt.Sprintf("should NOT %d", i), shouldNot)
		if err != nil {
			t.Error("addPattern: " + shouldNot + ": " + err.Error())
		}
		fields, err := fj.Flatten([]byte(j), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err := m.MatchesForFields(fields)
		if err != nil {
			t.Error("ShouldNot " + shouldNot + ": " + err.Error())
		}
		if len(matches) != 0 {
			t.Error(shouldNot + " matched but shouldn't have")
		}
	}
}

func TestSimpleAddPattern(t *testing.T) {
	// laboriously hand-check the simplest possible automaton
	var x X = "testing"
	pattern := `{"a": [1, 2], "b": [1, "3"]}`
	m := NewCoreMatcher()
	err := m.AddPattern(x, pattern)
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
