package quamina

import "testing"

func TestMatcherInterface(t *testing.T) {
	var m matcher = newCoreMatcher()
	if _, ok := m.(*coreMatcher); !ok {
		t.Error("Can't cast")
	}
	var x X = "x"
	err := m.addPattern(x, `{"x": [1, 2]}`)
	if err != nil {
		t.Error("addPattern? " + err.Error())
	}
	err = m.deletePatterns("x")
	if err == nil {
		t.Error("coreMatcher allowed Delete!?")
	}
	event := `{"x": [3, 1]}`
	fields, _ := newJSONFlattener().Flatten([]byte(event), m)
	matches, _ := m.matchesForFields(fields)
	if len(matches) != 1 || matches[0] != x {
		t.Error("missed match")
	}
}
