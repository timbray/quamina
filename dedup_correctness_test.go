package quamina

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestPathologicalCorrectness verifies that the pathological pattern mix from
// BenchmarkPathologicalEpsilon produces correct match results. The patterns
// exercise merged shell-style wildcards and pathological regexps that create
// large epsilon closures with shared table pointers.
func TestPathologicalCorrectness(t *testing.T) {
	q, _ := New()

	shellPatterns := []struct {
		name, glob string
	}{
		{"shell0", "*a*b*c*"},
		{"shell1", "*x*y*z*"},
		{"shell2", "*e*f*g*"},
		{"shell3", "*m*n*o*"},
		{"shell4", "*p*q*r*"},
		{"shell5", "*s*t*u*"},
		{"shell6", "*a*e*i*"},
		{"shell7", "*b*d*f*"},
		{"shell8", "*c*g*k*"},
		{"shell9", "*d*h*l*"},
		{"shell10", "*i*o*u*"},
		{"shell11", "*r*s*t*"},
	}
	for _, sp := range shellPatterns {
		pattern := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, sp.glob)
		if err := q.AddPattern(sp.name, pattern); err != nil {
			t.Fatal(err)
		}
	}

	regexPatterns := []struct {
		name, re string
	}{
		{"re0", "(([abc]?)*)+"},
		{"re1", "([abc]+)*d"},
		{"re2", "(a*)*b"},
		{"re3", "([xyz]?)*end"},
		{"re4", "(([mno]?)*)+"},
		{"re5", "([pqr]+)*s"},
	}
	for _, rp := range regexPatterns {
		pattern := fmt.Sprintf(`{"val": [{"regexp": "%s"}]}`, rp.re)
		if err := q.AddPattern(rp.name, pattern); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		event string
		want  []string
	}{
		{`{"val": "abc"}`, []string{"re0", "shell0"}},
		{`{"val": "abcd"}`, []string{"re1", "shell0"}},
		{`{"val": "aaab"}`, []string{"re0", "re2"}},
		{`{"val": "mno"}`, []string{"re4", "shell3"}},
		{`{"val": "pqrs"}`, []string{"re5", "shell4"}},
		{`{"val": "xyzend"}`, []string{"re3", "shell1"}},
		{`{"val": "abcxyz"}`, []string{"shell0", "shell1"}},
		{`{"val": "mnopqr"}`, []string{"shell3", "shell4"}},
		{`{"val": "aeiou"}`, []string{"shell10", "shell6"}},
		{`{"val": "rstuvwxyz"}`, []string{"shell1", "shell11", "shell5"}},
		{`{"val": "abcdefghijklmno"}`, []string{"shell0", "shell2", "shell3", "shell6", "shell7", "shell8", "shell9"}},
		{`{"val": "abcabcabcd"}`, []string{"re1", "shell0"}},
		{`{"val": "aaaaaab"}`, []string{"re0", "re2"}},
	}

	for _, tc := range tests {
		matches, err := q.MatchesForEvent([]byte(tc.event))
		if err != nil {
			t.Errorf("%s: error: %s", tc.event, err)
			continue
		}
		got := make([]string, len(matches))
		for i, m := range matches {
			got[i] = m.(string)
		}
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%s:\n  want: %v\n  got:  %v", tc.event, tc.want, got)
		}
	}
}
