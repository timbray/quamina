package core

import (
	"github.com/timbray/quamina/flattener"
	"testing"
)

func TestMatcherInterface(t *testing.T) {
	var m Matcher = NewCoreMatcher()
	if _, ok := m.(*CoreMatcher); !ok {
		t.Error("Can't cast")
	}
	var x X = "x"
	err := m.AddPattern(x, `{"x": [1, 2]}`)
	if err != nil {
		t.Error("AddPattern? " + err.Error())
	}
	err = m.DeletePattern("x")
	if err == nil {
		t.Error("CoreMatcher allowed Delete!?")
	}
	event := `{"x": [3, 1]}`
	fj := flattener.NewFJ()
	fields, err := fj.Flatten([]byte(event), m)
	if err != nil {
		t.Error("Flatten: " + err.Error())
	}
	matches, err := m.MatchesForFields(fields)
	if len(matches) != 1 || matches[0] != x {
		t.Error("missed match")
	}
}
