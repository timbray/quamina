//go:build go1.24

package quamina

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// Benchmarks designed to work with Go's 1.24 testing.B.Loop().  Note: When doing this kind of benchmarking, always
// call quamina.MatchesForEvent, as opposed to working directly with the coreMatcher, because the top-level function
// is clever about re-using the nfaBuffers structure.

func Benchmark8259Example(b *testing.B) {
	j := `{
        "Image": {
            "Width":  800,
            "Height": 600,
            "Title":  "View from 15th Floor",
            "Thumbnail": {
                "Url":    "https://www.example.com/image/481989943",
                "Height": 125,
                "Width":  100
            },
            "Animated" : false,
            "IDs": [116, 943, 234, 38793]
          }
      }`
	patternsFromReadme := []string{
		`{"Image": {"Width": [800]}}`,
		`{"Image": { "Animated": [ false], "Thumbnail": { "Height": [ 125 ] } } }}, "IDs": [943]}`,
		`{"Image": { "Title": [ { "exists": true } ] } }`,
		`{"Image": { "Width": [800], "Title": [ { "exists": true } ], "Animated": [ false ] } }`,
		`{"Image": { "Width": [800], "IDs": [ { "exists": true } ] } }`,
		`{"Foo": [ { "exists": false } ] }"`,
		`{"Image": { "Thumbnail": { "Url": [ { "wildcard": "*9943" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "wildcard": "https://www.example.*/*9943" } ] } } }`,
		`{"Image": { "Title": [ {"anything-but":  ["Pikachu", "Eevee"] } ]  } }`,
		`{"Image": { "Thumbnail": { "Url": [ "a", { "prefix": "https:" } ] } } }`,
		`{"Image": { "Title": [ { "equals-ignore-case": "VIEW FROM 15th FLOOR" } ] } }`,
		`{"Image": { "Title": [ { "regexp": "View .... [0-9][0-9][rtn][dh] Floor" } ]  } }`,
		`{"Image": { "Title": [ { "regexp": "(View)?( down)? from 15th (Floor|Storey)" } ]  } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "regexp": "https://www.example.com/[^0-9/]+/[1-9]+" } ] } } }`,
		`{"Image": { "Title": [ { "regexp": "[~p{L}~p{Zs}~p{Nd}]*"} ] } }"`,
	}

	var err error
	q, _ := New()
	before := time.Now()
	m := q.matcher.(*coreMatcher)
	for _, should := range patternsFromReadme {
		err = q.AddPattern(should, should)
		if err != nil {
			b.Error("add one of many: " + err.Error())
		}
	}
	elapsed := time.Since(before).Seconds()
	b.Logf("Adds/sec %.1f", float64(len(patternsFromReadme))/elapsed)
	b.Logf("FA: %s", matcherStats(m))
	bytes := []byte(j)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		matches, _ := q.MatchesForEvent(bytes)
		if len(matches) == 0 {
			b.Errorf("No matches")
		}
	}
	elapsed = float64(b.Elapsed().Seconds())
	count := float64(b.N)
	b.Logf("%.0f/sec", count/elapsed)
}

func BenchmarkShellStyleBuildTime(b *testing.B) {
	words := readWWords(b, 1000)

	source := rand.NewSource(293591)
	starWords := make([]string, 0, len(words))
	expandedWords := make([]string, 0, len(words))
	patterns := make([]string, 0, len(words))
	for _, word := range words {
		//nolint:gosec
		starAt := source.Int63() % 6
		starWord := string(word[:starAt]) + "*" + string(word[starAt:])
		expandedWord := string(word[:starAt]) + "ÉÉÉÉ" + string(word[starAt:])
		starWords = append(starWords, starWord)
		expandedWords = append(expandedWords, expandedWord)
		pattern := fmt.Sprintf(`{"x": [ {"shellstyle": "%s" } ] }`, starWord)
		patterns = append(patterns, pattern)
	}

	q, _ := New()
	for i := range words {
		err := q.AddPattern(starWords[i], patterns[i])
		if err != nil {
			b.Fatal("AddP: " + err.Error())
		}
	}

	// Build events: original words and expanded words
	type event struct {
		json []byte
		word string
	}
	events := make([]event, 0, len(words)*2)
	for i, word := range words {
		events = append(events,
			event{[]byte(fmt.Sprintf(`{"x": "%s"}`, word)), string(word)},
			event{[]byte(fmt.Sprintf(`{"x": "%s"}`, expandedWords[i])), expandedWords[i]},
		)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		for _, ev := range events {
			matches, err := q.MatchesForEvent(ev.json)
			if err != nil {
				b.Fatal("M4E on " + ev.word)
			}
			if len(matches) == 0 {
				b.Fatal("no matches for " + ev.word)
			}
		}
	}
}

// BenchmarkTablePointerDedup benchmarks matching speed for workloads where
// table-pointer dedup in epsilon closures has the most impact: nested
// quantifier regexps with overlapping character classes.
func BenchmarkTablePointerDedup(b *testing.B) {
	for _, wl := range dedupWorkloads {
		b.Run(wl.name, func(b *testing.B) {
			q := buildDedupMatcher(b, wl)
			events := dedupEvents()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				for _, event := range events {
					_, _ = q.MatchesForEvent(event)
				}
			}
		})
	}
}
