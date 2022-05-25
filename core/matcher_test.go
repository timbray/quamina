package core

import "testing"

func TestMatcherInterface(t *testing.T) {
	var m Matcher
	m = NewCoreMatcher()
	_, ok := m.(*CoreMatcher)
	if !ok {
		t.Error("Can't cast")
	}
	var x X
	x = "x"
	err := m.AddPattern(x, `{"x": [1]}`)
	if err != nil {
		t.Error("AddPattern? " + err.Error())
	}
	err = m.DeletePattern("x")
	if err == nil {
		t.Error("CoreMatcher allowed Delete!?")
	}
	event := `{"x": 1}`
	matches, err := m.MatchesForJSONEvent([]byte(event))
	if len(matches) != 1 || matches[0] != x {
		t.Error("missed match")
	}
}
