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
	//fmt.Println("ERR: " + err.Error())
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
	table, _ := makeRegexpNFA(parse.tree, false)
	// raw empty string should NOT match
	var transitions []*fieldMatcher
	bufs := &bufpair{}
	fields := traverseNFA(table, []byte(""), transitions, bufs)
	if len(fields) != 0 {
		t.Error("Matched empty string")
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

func TestRegexpValidity(t *testing.T) {
	problems := 0
	tests := 0
	implemented := 0
	for _, sample := range regexpSamples {
		tests++
		parse := newRxParseState([]byte(sample.regex))

		parse, err := readRegexpWithParse(parse)
		if sample.valid {
			if len(parse.features.foundUnimplemented()) == 0 {
				implemented++
				table, dest := makeRegexpNFA(parse.tree, false)
				for _, should := range sample.matches {
					// the sample regexp tests think the empty string matches lots of regexps with which
					// I don't think it should
					if should == "" {
						continue
					}
					var transitions []*fieldMatcher
					bufs := &bufpair{}
					fields := traverseNFA(table, []byte(should), transitions, bufs)
					if !containsFM(t, fields, dest) {
						t.Errorf("<%s> failed to match /%s/", should, sample.regex)
						problems++
					}
				}
				for _, shouldNot := range sample.nomatches {
					var transitions []*fieldMatcher
					bufs := &bufpair{}
					fields := traverseNFA(table, []byte(shouldNot), transitions, bufs)
					if len(fields) != 0 {
						t.Errorf("<%s> matched /%s/", shouldNot, sample.regex)
						problems++
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
	fmt.Printf("tests: %d, implemented: %d\n", tests, implemented)
}
