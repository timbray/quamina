package quamina

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// buildHeavyMatcher creates a Quamina instance loaded with 120+ shell-style
// and pathological regexp patterns on the same field, producing large epsilon
// closures with shared table pointers.
func buildHeavyMatcher(t *testing.T) *Quamina {
	t.Helper()
	q, _ := New()

	letters := "abcdefghijklmnopqrstuvwxyz"
	patCount := 0

	// 3-wildcard patterns
	for i := 0; i < len(letters)-2; i++ {
		for j := i + 1; j < len(letters)-1; j += 3 {
			ss := fmt.Sprintf("*%c*%c*%c*", letters[i], letters[j], letters[j+1])
			pat := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, ss)
			if err := q.AddPattern(fmt.Sprintf("s%d", patCount), pat); err != nil {
				t.Fatal(err)
			}
			patCount++
		}
	}

	// 4-wildcard patterns
	for i := 0; i < len(letters)-3; i += 2 {
		ss := fmt.Sprintf("*%c*%c*%c*%c*", letters[i], letters[i+1], letters[i+2], letters[i+3])
		pat := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, ss)
		if err := q.AddPattern(fmt.Sprintf("s%d", patCount), pat); err != nil {
			t.Fatal(err)
		}
		patCount++
	}

	// Pathological regexps
	for i := 0; i < len(letters)-2; i += 3 {
		triple := letters[i : i+3]
		re := fmt.Sprintf("(([%s]?)*)+", triple)
		pat := fmt.Sprintf(`{"val": [{"regexp": "%s"}]}`, re)
		if err := q.AddPattern(fmt.Sprintf("r%d", i/3), pat); err != nil {
			t.Fatal(err)
		}
	}

	t.Logf("Added %d shell patterns + %d regexp patterns", patCount, (len(letters)-2)/3+1)
	m := q.matcher.(*coreMatcher)
	t.Log(matcherStats(m))
	return q
}

func heavyEvents() [][]byte {
	return [][]byte{
		[]byte(fmt.Sprintf(`{"val": "%s"}`, strings.Repeat("abcdefghijklmnopqrstuvwxyz", 4))),
		[]byte(fmt.Sprintf(`{"val": "%s"}`, strings.Repeat("aebfcgdheifjgkhlijmknlo", 4))),
		[]byte(fmt.Sprintf(`{"val": "%s"}`, strings.Repeat("zywvutsrqponmlkjihgfedcba", 4))),
	}
}

func sortedMatchStrings(matches []X) []string {
	got := make([]string, len(matches))
	for i, m := range matches {
		got[i] = m.(string)
	}
	sort.Strings(got)
	return got
}

// TestHeavyPatternCorrectness checks that the heavy pattern mix produces
// correct match results. The expected values were verified on the main branch.
func TestHeavyPatternCorrectness(t *testing.T) {
	q := buildHeavyMatcher(t)
	events := heavyEvents()

	for _, event := range events {
		start := time.Now()
		matches, err := q.MatchesForEvent(event)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("match error: %s", err)
		}
		got := sortedMatchStrings(matches)
		t.Logf("%-20s → %d matches in %v: %v", event[:40], len(got), elapsed, got)
	}
}

// TestHeavyPatternTimeout verifies that matching completes within 10 seconds.
func TestHeavyPatternTimeout(t *testing.T) {
	q := buildHeavyMatcher(t)
	events := heavyEvents()

	start := time.Now()
	deadline := time.After(10 * time.Second)
	done := make(chan bool)

	go func() {
		for _, event := range events {
			_, err := q.MatchesForEvent(event)
			if err != nil {
				t.Errorf("match error: %s", err)
			}
		}
		done <- true
	}()

	select {
	case <-done:
		t.Logf("completed in %v", time.Since(start))
	case <-deadline:
		t.Fatal("matching timed out after 10s — NFA traversal blowup")
	}
}
