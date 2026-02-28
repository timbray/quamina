//go:build go1.24

package quamina

import (
	"fmt"
	"testing"
)

// BenchmarkNfa2Dfa measures the cost of nfa2Dfa conversion, where intern()
// in state_lists.go typically dominates. Patterns with more wildcards produce
// larger epsilon closures and more intern() calls.
func BenchmarkNfa2Dfa(b *testing.B) {
	patterns := []struct {
		name    string
		pattern string
	}{
		{"single_star", "*foo*"},
		{"two_stars", "*foo*bar*"},
		{"three_stars", "*a*b*c*"},
		{"five_stars", "*a*b*c*d*e*"},
		{"eight_stars", "*a*b*c*d*e*f*g*h*"},
	}

	pp := newPrettyPrinter(12345)
	for _, tc := range patterns {
		b.Run(tc.name, func(b *testing.B) {
			nfa, _ := makeShellStyleFA([]byte(fmt.Sprintf(`"%s"`, tc.pattern)), pp)
			epsilonClosure(nfa)
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				nfa2Dfa(nfa)
			}
		})
	}
}
