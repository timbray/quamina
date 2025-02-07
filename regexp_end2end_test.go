package quamina

import (
	"fmt"
	"testing"
)

type rxTest struct {
	rx         string
	matches    []string
	nonMatches []string
}

func TestRegexpEnd2End(t *testing.T) {
	// somewhat duplicative of the samples-based regexp_validity_test but worth
	// doing just to check for merge problems.
	allPatternsCM := newCoreMatcher()

	tests := []rxTest{
		{rx: "a|b", matches: []string{"a", "b"}, nonMatches: []string{"x", "Á"}},
		{rx: "a", matches: []string{"a"}, nonMatches: []string{"b", ""}},
		{rx: "a.b", matches: []string{"axb", "a.b", "aÉb"}, nonMatches: []string{"ab", "axxb"}},
		{rx: "abc|def", matches: []string{"abc", "def"}, nonMatches: []string{"x", "Á"}},
		{rx: "[hij]", matches: []string{"h", "i", "j"}, nonMatches: []string{"x", "Á"}},
		{rx: "a[e-g]x", matches: []string{"aex", "afx", "agx"}, nonMatches: []string{"ax", "axx"}},
		{rx: "[ae-gx]", matches: []string{"a", "e", "f", "g", "x"}, nonMatches: []string{"b", "Á"}},
		{rx: "[-ab]", matches: []string{"-", "a", "b"}, nonMatches: []string{"c", "Á"}},
		{rx: "[ab-]", matches: []string{"-", "a", "b"}, nonMatches: []string{"c", "Á"}},
		{rx: "[~[~]]", matches: []string{"[", "]"}, nonMatches: []string{"", "Á"}},
		{rx: "[~r~t~n]", matches: []string{"\\r", "\\t", "\\n"}, nonMatches: []string{"c", "Á"}},
		{rx: "[a-c]|[xz]", matches: []string{"a", "b", "c", "x", "z"}, nonMatches: []string{"", "Á", "w"}},
		{rx: "[ac-e]h|p[xy]", matches: []string{"ah", "ch", "dh", "eh", "px", "py"}, nonMatches: []string{"", "Á", "xp"}},
		{rx: "[0-9][0-9][rtn][dh]", matches: []string{"11th", "23rd", "22nd"}, nonMatches: []string{"first", "9th"}},
		{rx: "a(h|i)z", matches: []string{"ahz", "aiz"}, nonMatches: []string{"a.z", "Á"}},
		{rx: "a([1-3]|ac)z", matches: []string{"a1z", "a2z", "a3z", "aacz"}, nonMatches: []string{"a.z", "Á", "a0^z"}},
		{rx: "a(h|([x-z]|(1|2)))z", matches: []string{"ahz", "axz", "a1z", "a2z"}, nonMatches: []string{"a.z", "Á"}},
	}

	for _, test := range tests {
		cm := newCoreMatcher()
		pattern := fmt.Sprintf(`{"a": [{"regexp": "%s"}]}`, test.rx)
		err := cm.addPattern("a", pattern)
		if err != nil {
			t.Error("addP: " + err.Error())
			continue
		}
		err = allPatternsCM.addPattern(pattern, pattern)
		if err != nil {
			t.Error("addPAll" + err.Error())
			continue
		}
		for _, match := range test.matches {
			event := fmt.Sprintf(`{"a": "%s"}`, match)
			matches, err := cm.matchesForJSONEvent([]byte(event))
			if err != nil {
				t.Error("M4JE: " + err.Error())
			}
			if len(matches) != 1 || matches[0] != "a" {
				t.Errorf("%s didn't match /%s/", match, test.rx)
			}
		}
		for _, match := range test.nonMatches {
			event := fmt.Sprintf(`{"a": "%s"}`, match)
			matches, err := cm.matchesForJSONEvent([]byte(event))
			if err != nil {
				t.Error("M4JE: " + err.Error())
			}
			if len(matches) != 0 {
				t.Errorf("%s matched /%s/", match, test.rx)
			}
		}
	}
	// now let's see if the merged FA's work
	for _, test := range tests {
		for _, match := range test.matches {
			event := fmt.Sprintf(`{"a": "%s"}`, match)
			matches, err := allPatternsCM.matchesForJSONEvent([]byte(event))
			if err != nil {
				t.Error("M4JE: " + err.Error())
			}
			if len(matches) == 0 {
				t.Errorf("%s didn't match in merge FA", match)
			}
			pattern := fmt.Sprintf(`{"a": [{"regexp": "%s"}]}`, test.rx)
			if !containsX(matches, pattern) {
				t.Errorf("event %s should match %s", event, pattern)
			}
		}
	}
}
