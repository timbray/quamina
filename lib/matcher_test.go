package quamina

import (
	"fmt"
	"testing"
)

func TestBasicMatching(t *testing.T) {
	var x X
	x = X("testing")
	pattern := `{"a": [1, 2], "b": [1, "3"]}`
	m := NewMatcher()
	err := m.AddPattern(x, pattern)
	if err != nil {
		t.Error(err.Error())
	}
	shouldMatch := []string{
		`{"a": 2, "b": "3", "x": 33}`,
		`{"b": "3", "a": 1}`,
	}
	shouldNotMatch := []string{
		`{"b": "3", "a": 6}`,
		`{"a": 2}`,
		`{"b": "3"}`,
	}
	for _, shouldNot := range shouldNotMatch {
		var matches []X
		matches, err = m.MatchesForJSONEvent([]byte(shouldNot))
		if len(matches) != 0 {
			t.Error("Matched: " + shouldNot)
		}
	}
	for _, should := range shouldMatch {
		var matches []X
		matches, err = m.MatchesForJSONEvent([]byte(should))
		if err != nil {
			t.Error(err.Error())
		}
		if len(matches) != 1 {
			t.Errorf("event %s, LM %d", should, len(matches))
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
                "Url":    "http://www.example.com/image/481989943",
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
	}

	var err error
	for i, should := range shouldMatches {
		m := NewMatcher()
		err = m.AddPattern(fmt.Sprintf("should %d", i), should)
		if err != nil {
			t.Error("addPattern " + should + ": " + err.Error())
		}
		matches, err := m.MatchesForJSONEvent([]byte(j))
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
		m := NewMatcher()
		err = m.AddPattern(fmt.Sprintf("should NOT %d", i), shouldNot)
		if err != nil {
			t.Error("addPattern: " + shouldNot + ": " + err.Error())
		}
		matches, err := m.MatchesForJSONEvent([]byte(j))
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
	var x X
	x = X("testing")
	pattern := `{"a": [1, 2], "b": [1, "3"]}`
	m := NewMatcher()
	err := m.AddPattern(x, pattern)
	if err != nil {
		t.Error(err.Error())
	}
	if len(m.namesUsed) != 2 {
		t.Errorf("nameUsed = %d", len(m.namesUsed))
	}
	if !m.IsNameUsed([]byte("a")) {
		t.Error("'a' not showing as used")
	}
	if !m.IsNameUsed([]byte("b")) {
		t.Error("'b' not showing as used")
	}
	s0 := m.startState
	if len(s0.transitions) != 1 {
		t.Errorf("s0 trans len %d", len(s0.transitions))
	}

	v0, ok := s0.transitions["a"]
	if !ok {
		t.Error("No trans from start on 'a'")
	}
	if len(v0.valueTransitions) != 2 {
		t.Errorf("v1 trans %d wanted 2", len(v0.valueTransitions))
	}
	s1, ok := v0.valueTransitions["1"]
	if !ok {
		t.Error("no trans on 1 fro s1")
	}
	s2, ok := v0.valueTransitions["2"]
	if !ok {
		t.Error("no trans on 2 from s2")
	}
	if len(s1.transitions) != 1 {
		t.Errorf("s1 trans len %d", len(s1.transitions))
	}
	if len(s2.transitions) != 1 {
		t.Errorf("s2 trans len %d", len(s2.transitions))
	}
	v1, ok := s1.transitions["b"]
	if !ok {
		t.Error("no trans on b from s1")
	}
	v2, ok := s2.transitions["b"]
	if !ok {
		t.Error("no trans on b from s2")
	}
	for _, v := range []*valueMatchState{v1, v2} {
		if len(v.valueTransitions) != 2 {
			t.Errorf("trans len on %v = %d", v, len(v.valueTransitions))
		}
		s3, ok := v.valueTransitions["1"]
		if !ok {
			t.Error("no trans on 1 at s3")
		}
		if len(s3.transitions) != 0 {
			t.Errorf("len trans s3 = %d", len(s3.transitions))
		}
		if len(s3.matches) != 1 {
			t.Errorf("s3 matches %d", len(s3.matches))
		}
		if s3.matches[0] != x {
			t.Error("s3 match mismatch")
		}
		s4, ok := v.valueTransitions[`"3"`]
		if !ok {
			t.Error(`no trans on "3" at s4`)
		}
		if len(s4.transitions) != 0 {
			t.Errorf("len trans s4 = %d", len(s4.transitions))
		}
		if len(s4.matches) != 1 {
			t.Errorf("s4 matches %d", len(s4.matches))
		}
		if s4.matches[0] != x {
			t.Error("s4 match mismatch")
		}
	}
}
