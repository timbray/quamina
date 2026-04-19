//go:build go1.24

package quamina

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func BenchmarkMemoryBoundaries(b *testing.B) {
	words := readWWords(b, 0)

	fmt.Printf("WC %d\n", len(words))
	patterns := make([]string, 0, len(words))
	source := rand.NewSource(293591)

	// Budgets scaled for HeapAlloc-based accounting (retained memory,
	// not cumulative alloc traffic). 8 MiB matches RE2's default cache
	// ceiling; the sweep spans an order of magnitude around that.
	meg := uint64(1 << 20)
	k := 1024.0
	budgets := []uint64{2 * meg, 4 * meg, 8 * meg, 16 * meg, 32 * meg, 64 * meg}
	for _, word := range words {
		//nolint:gosec
		starAt := source.Int63() % 6
		starWord := string(word[:starAt]) + "*" + string(word[starAt:])
		pattern := fmt.Sprintf(`{"x": [ {"shellstyle": "%s" } ] }`, starWord)
		patterns = append(patterns, pattern)
	}
	b.ReportAllocs()
	totalPatternsAdded := 0
	for b.Loop() {
		for _, budget := range budgets {
			fmt.Printf("%dM memory budget ", budget/meg)
			cm := newCoreMatcher()
			_, _ = cm.setMemoryBudget(budget)
			for i, pattern := range patterns {
				err := cm.addPattern("x", pattern)
				if err == nil {
					totalPatternsAdded++
				} else {
					if strings.Contains(err.Error(), "Memory") {
						//golint: nosec
						perPattern := float64(budget) / float64(i) / k
						fmt.Printf("supports %d patterns, %.1fK bytes/pattern\n", i, perPattern)
					} else {
						b.Error("Weird error: " + err.Error())
					}
					break
				}
			}
			fmt.Println(stStats(b, cm) + "\n")
		}
	}
	fmt.Printf("Total patterns: %d\n", totalPatternsAdded)
}

func stStats(b *testing.B, m *coreMatcher) string {
	b.Helper()
	s := statsAccum{
		fmVisited: make(map[*fieldMatcher]bool),
		vmVisited: make(map[*valueMatcher]bool),
		stVisited: make(map[*smallTable]bool),
	}
	fmStats(m.fields().state, &s)
	avgStSize := "n/a"
	avgEpSize := "n/a"
	if s.stTblCount > 0 {
		avgStSize = fmt.Sprintf("%.3f", float64(s.stEntries)/float64(s.stTblCount))
	}
	if s.stEpsilon > 0 {
		avgEpSize = fmt.Sprintf("%.3f", float64(s.stEpsilon)/float64(s.stTblCount))
	}
	return fmt.Sprintf("SmallTables %d (splices %d, avg %s, max %d, epsilons avg %s, max %d) singletons %d",
		len(s.stVisited), s.splices, avgStSize, s.stMax, avgEpSize, s.stepMax, s.siCount)
}
