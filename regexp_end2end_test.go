package quamina

import (
	"fmt"
	"testing"
)

func TestRegexpEnd2End(t *testing.T) {
	// somewhat duplicative of the samples-based regexp_validity_test but worth
	// doing just to check for merge problems.

	tests := []regexpSample{
		{
			regex:     "(xyz)?a?b",
			matches:   []string{"xyzb", "xyzab", "ab", "b"},
			nomatches: []string{"xyzc", "c", "xyza"},
		},
		{regex: "a|b", matches: []string{"a", "b"}, nomatches: []string{"x", "Á"}},
		{regex: "a", matches: []string{"a"}, nomatches: []string{"b", ""}},
		{regex: "a.b", matches: []string{"axb", "a.b", "aÉb"}, nomatches: []string{"ab", "axxb"}},
		{regex: "abc|def", matches: []string{"abc", "def"}, nomatches: []string{"x", "Á"}},
		{regex: "[hij]", matches: []string{"h", "i", "j"}, nomatches: []string{"x", "Á"}},
		{regex: "a[e-g]x", matches: []string{"aex", "afx", "agx"}, nomatches: []string{"ax", "axx"}},
		{regex: "[ae-gx]", matches: []string{"a", "e", "f", "g", "x"}, nomatches: []string{"b", "Á"}},
		{regex: "[-ab]", matches: []string{"-", "a", "b"}, nomatches: []string{"c", "Á"}},
		{regex: "[ab-]", matches: []string{"-", "a", "b"}, nomatches: []string{"c", "Á"}},
		{regex: "[~[~]]", matches: []string{"[", "]"}, nomatches: []string{"", "Á"}},
		{regex: "[~r~t~n]", matches: []string{"\\r", "\\t", "\\n"}, nomatches: []string{"c", "Á"}},
		{regex: "[a-c]|[xz]", matches: []string{"a", "b", "c", "x", "z"}, nomatches: []string{"", "Á", "w"}},
		{regex: "[ac-e]h|p[xy]", matches: []string{"ah", "ch", "dh", "eh", "px", "py"}, nomatches: []string{"", "Á", "xp"}},
		{regex: "[0-9][0-9][rtn][dh]", matches: []string{"11th", "23rd", "22nd"}, nomatches: []string{"first", "9th"}},
		{regex: "a(h|i)z", matches: []string{"ahz", "aiz"}, nomatches: []string{"a.z", "Á"}},
		{regex: "a([1-3]|ac)z", matches: []string{"a1z", "a2z", "a3z", "aacz"}, nomatches: []string{"a.z", "Á", "a0^z"}},
		{regex: "a(h|([x-z]|(1|2)))z", matches: []string{"ahz", "axz", "a1z", "a2z"}, nomatches: []string{"a.z", "Á"}},
	}
	testRegexpMatches(t, tests)
}

func testRegexpMatches(t *testing.T, tests []regexpSample) {
	t.Helper()

	allPatternsCM := newCoreMatcher()
	// not using runRegexpTests because we also want to add all the patterns to check for merging
	for _, test := range tests {
		cm := newCoreMatcher()
		pattern := fmt.Sprintf(`{"a": [{"regexp": "%s"}]}`, test.regex)
		err := cm.addPattern("a", pattern)
		if err != nil {
			t.Error("addP: " + err.Error())
			continue
		}
		fa := fetchFAForPath(t, cm, "a")
		if fa == nil {
			t.Error("FETCH failed")
		}
		// pp := newPrettyPrinter(8899)
		// fmt.Printf("FA for %s: %s\n", test.regex, pp.printNFA(fa))
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
				t.Errorf("single <%s> didn't match /%s/", match, test.regex)
			}
		}
		for _, match := range test.nomatches {
			event := fmt.Sprintf(`{"a": "%s"}`, match)
			matches, err := cm.matchesForJSONEvent([]byte(event))
			if err != nil {
				t.Error("M4JE: " + err.Error())
			}
			if len(matches) != 0 {
				t.Errorf("singlex <%s> matched /%s/", match, test.regex)
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
				t.Errorf("<%s> didn't match in merge FA", match)
			}
			pattern := fmt.Sprintf(`{"a": [{"regexp": "%s"}]}`, test.regex)
			if !containsX(matches, pattern) {
				t.Errorf("event %s should match %s", event, pattern)
			}
		}
	}
}
