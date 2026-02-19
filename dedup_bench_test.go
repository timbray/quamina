package quamina

import (
	"fmt"
	"testing"
)

// BenchmarkPathologicalEpsilon exercises the epsilon closure dedup path by
// merging many shell-style and regexp patterns on the same field. This creates
// large epsilon closures with shared table pointers, which is the case the
// table-pointer dedup in traverseEpsilons is designed to optimize.
func BenchmarkPathologicalEpsilon(b *testing.B) {
	q, _ := New()

	// Multiple multi-wildcard shell-style patterns on the same field.
	// Each merge creates splice states with epsilon transitions, and the
	// overlapping wildcards cause table sharing in the resulting closures.
	shellPatterns := []string{
		"*a*b*c*",
		"*x*y*z*",
		"*e*f*g*",
		"*m*n*o*",
		"*p*q*r*",
		"*s*t*u*",
		"*a*e*i*",
		"*b*d*f*",
		"*c*g*k*",
		"*d*h*l*",
		"*i*o*u*",
		"*r*s*t*",
	}
	for i, ss := range shellPatterns {
		pattern := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, ss)
		if err := q.AddPattern(fmt.Sprintf("shell%d", i), pattern); err != nil {
			b.Fatal(err)
		}
	}

	// Pathological regexps that create epsilon loops
	regexPatterns := []string{
		"(([abc]?)*)+",
		"([abc]+)*d",
		"(a*)*b",
		"([xyz]?)*end",
		"(([mno]?)*)+",
		"([pqr]+)*s",
	}
	for i, re := range regexPatterns {
		pattern := fmt.Sprintf(`{"val": [{"regexp": "%s"}]}`, re)
		if err := q.AddPattern(fmt.Sprintf("re%d", i), pattern); err != nil {
			b.Fatal(err)
		}
	}

	m := q.matcher.(*coreMatcher)
	b.Log(matcherStats(m))

	events := [][]byte{
		[]byte(`{"val": "abcxyz"}`),
		[]byte(`{"val": "mnopqr"}`),
		[]byte(`{"val": "aeiou"}`),
		[]byte(`{"val": "rstuvwxyz"}`),
		[]byte(`{"val": "abcdefghijklmno"}`),
		[]byte(`{"val": "xyzend"}`),
		[]byte(`{"val": "abcabcabcd"}`),
		[]byte(`{"val": "aaaaaab"}`),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, event := range events {
			_, err := q.MatchesForEvent(event)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
