package quamina

import (
	"testing"
	"time"
)

func BenchmarkCityLots(b *testing.B) {
	var localMatches []X

	patterns := []string{
		`{ "properties": { "STREET": [ "CRANLEIGH" ] } }`,
		`{ "properties": { "STREET": [ "17TH" ], "ODD_EVEN": [ "E"] } }`,
		`{ "geometry": { "coordinates": [ 37.807807921694092 ] } }`,
		`{ "properties": { "MAPBLKLOT": ["0011008"], "BLKLOT": ["0011008"]},  "geometry": { "coordinates": [ 37.807807921694092 ] } } `,
	}
	names := []string{
		"CRANLEIGH",
		"17TH Even",
		"Geometry",
		"0011008",
	}

	var err error
	q, err := New()
	if err != nil {
		b.Fatalf("New(): %s", err.Error())
	}
	for i := range names {
		err = q.AddPattern(names[i], patterns[i])
		if err != nil {
			b.Fatalf("AddPattern failed: %s", err.Error())
		}
	}
	b.Log(matcherStats(q.matcher.(*coreMatcher)))

	lines := getCityLotsLines(b)

	b.ResetTimer()
	before := time.Now()

	for i := 0; i < b.N; i++ {
		lineIndex := i
		if i >= len(lines) {
			lineIndex = 0
		}

		matches, err := q.MatchesForEvent(lines[lineIndex])
		if err != nil {
			b.Errorf("Matches4JSON: %s", err.Error())
		}

		localMatches = matches
	}

	topMatches = localMatches
	elapsed := float64(time.Since(before).Milliseconds())
	perSecond := float64(b.N) / (elapsed / 1000.0)
	b.Logf("%.2f matches/second, total lines: %d", perSecond, b.N)
}
