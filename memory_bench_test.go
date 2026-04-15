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

	meg := uint64(1e6)
	gig := uint64(1e9)
	k := 1024.0
	budgets := []uint64{100 * meg, 200 * meg, 400 * meg, 800 * meg, 2 * gig}
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
		}
	}
	fmt.Printf("Total patterns: %d\n", totalPatternsAdded)
}
