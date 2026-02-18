//go:build stress

package quamina

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestBreak500Limit creates enough overlapping wildcard patterns to push
// the unique NFA state count well past 500 per step.
func TestBreak500Limit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping heavy pattern test in short mode")
	}
	q, _ := New()
	patCount := 0

	letters := "abcdefghijklmnopqrstuvwxyz"

	// All 2-letter pairs: *X*Y* — C(26,2) = 325 patterns
	for i := 0; i < len(letters); i++ {
		for j := i + 1; j < len(letters); j++ {
			ss := fmt.Sprintf("*%c*%c*", letters[i], letters[j])
			pat := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, ss)
			if err := q.AddPattern(fmt.Sprintf("p%d", patCount), pat); err != nil {
				t.Fatal(err)
			}
			patCount++
		}
	}

	// All 3-letter triples: *X*Y*Z* — C(26,3) = 2600 patterns
	for i := 0; i < len(letters); i++ {
		for j := i + 1; j < len(letters); j++ {
			for k := j + 1; k < len(letters); k++ {
				ss := fmt.Sprintf("*%c*%c*%c*", letters[i], letters[j], letters[k])
				pat := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, ss)
				if err := q.AddPattern(fmt.Sprintf("p%d", patCount), pat); err != nil {
					t.Fatal(err)
				}
				patCount++
			}
		}
	}

	t.Logf("Added %d patterns", patCount)
	m := q.matcher.(*coreMatcher)
	t.Log(matcherStats(m))

	// Different input strategies to maximize active NFA states:
	events := []struct {
		name  string
		event []byte
	}{
		{
			// Alphabet repeated: every char triggers branching for many patterns
			"alpha-repeat",
			[]byte(fmt.Sprintf(`{"val": "%s"}`, strings.Repeat("abcdefghijklmnopqrstuvwxyz", 4))),
		},
		{
			// Only early letters repeated: maximizes partial matches for
			// patterns with early triggers, spinners never see completing chars
			"early-only",
			[]byte(fmt.Sprintf(`{"val": "%s"}`, strings.Repeat("abcabc", 30))),
		},
		{
			// Interleaved early/late: forces maximum simultaneous branching
			// as each char triggers a different subset of patterns
			"interleaved",
			[]byte(fmt.Sprintf(`{"val": "%s"}`, strings.Repeat("azbyxcwdveu", 16))),
		},
		{
			// Near-misses: lots of spinner work, chars almost complete patterns
			// but the completing char comes late
			"near-miss",
			[]byte(fmt.Sprintf(`{"val": "%s"}`,
				strings.Repeat("a", 50)+"b"+strings.Repeat("c", 50)+"d"+
					strings.Repeat("e", 50)+"f"+strings.Repeat("g", 50)+"h")),
		},
		{
			// Single char repeated: all *a* spinners stay active, nothing completes
			"single-repeat",
			[]byte(fmt.Sprintf(`{"val": "%s"}`, strings.Repeat("m", 200))),
		},
	}

	start := time.Now()
	deadline := time.After(60 * time.Second)
	done := make(chan bool)

	go func() {
		for _, tc := range events {
			eStart := time.Now()
			matches, err := q.MatchesForEvent(tc.event)
			if err != nil {
				t.Errorf("match error: %s", err)
			}
			t.Logf("%-16s %d matches in %v", tc.name, len(matches), time.Since(eStart))
		}
		done <- true
	}()

	select {
	case <-done:
		t.Logf("all events completed in %v", time.Since(start))
	case <-deadline:
		t.Fatal("matching timed out after 60s")
	}
}
