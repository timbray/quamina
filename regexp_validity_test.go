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
	oneRegexp(t, "(~p{Ll}~p{Cc}~p{Nd})*", true)
}

func TestRegexpValidity(t *testing.T) {
	problems := 0
	tests := 0
	for _, sample := range regexpSamples {
		tests++
		_, err := readRegexp(sample.regex)
		if sample.valid {
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
		if tests%100 == 0 {
			fmt.Printf("Tests: %d\n", tests)
		}
		if problems == 10 {
			return
		}
	}
}
