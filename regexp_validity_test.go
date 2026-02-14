package quamina

import (
	"fmt"
	"testing"
)

func oneRegexp(t *testing.T, re string, valid bool) {
	t.Helper()
	_, err := readRegexp(re)
	if valid && err != nil {
		t.Errorf("should be valid: /%s/, but <%s>", re, err.Error())
	}
	if (!valid) && err == nil {
		t.Errorf("should NOT be valid: /%s/", re)
	}
}

func TestDebugRegexp(t *testing.T) {
	oneRegexp(t, "[~]", false)
}

func TestEmptyRegexp(t *testing.T) {
	parse := newRxParseState([]byte{})
	parse, err := readRegexpWithParse(parse)
	if err != nil {
		fmt.Println("OOPS: " + err.Error())
	}
	table, fm := makeRegexpNFA(parse.tree, sharedNullPrinter)
	// empty quoted string should match empty regexp
	var transitions []*fieldMatcher
	bufs := newNfaBuffers()
	fields := testTraverseNFA(table, []byte(`""`), transitions, bufs, sharedNullPrinter)
	if len(fields) != 1 || fields[0] != fm {
		t.Error("Failed to match empty string")
	}

	// matching on a field SHOULD match
	pattern := `{"a": [{"regexp": ""}]}`
	cm := newCoreMatcher()
	err = cm.addPattern("a", pattern)
	if err != nil {
		t.Error("addPattern: " + err.Error())
	}
	event := `{"a": ""}`
	mm, err := cm.matchesForJSONEvent([]byte(event))
	if err != nil {
		t.Error("M4J: " + err.Error())
	}
	if len(mm) == 0 {
		t.Error("Didn't match empty to empty")
	}
}

func TestToxicStack(t *testing.T) {
	var table *smallTable
	pp := newPrettyPrinter(34897)

	re3 := "(([~.~~~?~*~+~{~}~[~]~(~)~|]?)*)+"
	parse := newRxParseState([]byte(re3))

	str := `".~?*+{}[]()|.~?*+{}[]()|.~?*+{}[]()|"`

	parse, err := readRegexpWithParse(parse)
	if err != nil {
		t.Error("OOPS: " + err.Error())
	}
	table, _ = makeRegexpNFA(parse.tree, pp)

	var transitions []*fieldMatcher
	bufs := newNfaBuffers()
	trans := testTraverseNFA(table, []byte(str), transitions, bufs, sharedNullPrinter)
	if len(trans) != 1 {
		t.Error("Toxic stack failure")
	}
}

func TestRegexpValidity(t *testing.T) {
	t.Helper()
	problems := 0
	tests := 0
	implemented := 0
	correctlyMatched := 0
	correctlyNotMatched := 0

	var starSamplesMatchingEmpty = map[string]bool{
		"(([~.~~~?~*~+~{~}~[~]~(~)~|]?)*)+":     true,
		"[~~~|~.~?~*~+~(~)~{~}~-~[~]~^]*":       true,
		"[~*a]*":                                true,
		"[a-]*":                                 true,
		"[~n~r~t~~~|~.~-~^~?~*~+~{~}~[~]~(~)]*": true,
		"[a~*]*":                                true,
		"[0-9]*":                                true,
		"(([a-d]*)|([a-z]*))":                   true,
		"(([d-f]*)|([c-e]*))":                   true,
		"(([c-e]*)|([d-f]*))":                   true,
		"(([a-d]*)|(.*))":                       true,
		"(([d-f]*)|(.*))":                       true,
		"(([c-e]*)|(.*))":                       true,
		"(.*)":                                  true,
		"([^~?])*":                              true,
		"~p{So}*":                               true,
		"(~p{Co})*":                             true,
		"~p{Cn}*":                               true,
		"~P{Cc}*":                               true,
		"|":                                     true,
	}

	featureMatchTests := make(map[regexpFeature]int)
	featureNotMatchTests := make(map[regexpFeature]int)

	// TODO: Was 6.42s user 0.80s system 242% cpu 2.979 total from command line
	// TestRegexpValidity (2.53s) in JetBrains

	for _, sample := range regexpSamples {
		tests++
		parse := newRxParseState([]byte(sample.regex))

		parse, err := readRegexpWithParse(parse)
		if sample.valid {
			// fmt.Println("Sample: " + sample.regex)
			if len(parse.features.foundUnimplemented()) == 0 {
				implemented++
				table, dest := makeRegexpNFA(parse.tree, sharedNullPrinter)
				for _, should := range sample.matches {
					var transitions []*fieldMatcher
					bufs := newNfaBuffers()
					fields := testTraverseNFA(table, []byte(`"`+should+`"`), transitions, bufs, sharedNullPrinter)

					if !containsFM(t, fields, dest) {
						// the sample regexp tests think the empty string matches lots of regexps with which
						// I don't think it should
						if should != "" {
							t.Errorf("<%s> failed to match /%s/", should, sample.regex)
							problems++
						}
					} else {
						correctlyMatched++
						for feature := range parse.features.found {
							count, ok := featureMatchTests[feature]
							if !ok {
								count = 1
							} else {
								count++
							}
							featureMatchTests[feature] = count
						}
					}
				}
				for _, shouldNot := range sample.nomatches {
					var transitions []*fieldMatcher
					bufs := newNfaBuffers()
					fields := testTraverseNFA(table, []byte(`"`+shouldNot+`"`), transitions, bufs, sharedNullPrinter)
					if len(fields) != 0 {
						// similarly, it says quite a lot of empty strins should not match regexps that
						// have stars and *should* match them
						if len(shouldNot) == 0 && !starSamplesMatchingEmpty[sample.regex] {
							t.Errorf("<%s> matched /%s/", shouldNot, sample.regex)
							problems++
						}
					} else {
						correctlyNotMatched++
						for feature := range parse.features.found {
							count, ok := featureNotMatchTests[feature]
							if !ok {
								count = 1
							} else {
								count++
							}
							featureNotMatchTests[feature] = count
						}
					}
				}
			}
			if err != nil {
				t.Errorf("should be valid: /%s/, but <%s> (after %d lines) ", sample.regex, err.Error(), tests)
				problems++
			}
		} else {
			if err == nil {
				t.Errorf("should NOT be valid: /%s/ (after %d lines) ", sample.regex, tests)
				problems++
			}
		}
		if problems == 10 {
			return
		}
	}
	fmt.Println("Feature match test counts:")
	for feature, count := range featureMatchTests {
		fmt.Printf(" %d %s\n", count, feature)
	}
	fmt.Println("Feature non-match test counts:")
	for feature, count := range featureNotMatchTests {
		fmt.Printf(" %d %s\n", count, feature)
	}
	fmt.Printf("tests: %d, implemented: %d, matches/nonMatches: %d/%d\n", tests, implemented,
		correctlyMatched, correctlyNotMatched)
}
