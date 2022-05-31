package quamina

import (
	"fmt"
	"testing"
	"time"
)

// reduced to allow unit tests in slow GitHub actions to pass
// const thresholdPerformance = 120000.0
const thresholdPerformance = 1.0

// TestCityLots is the benchmark that was used in most of Quamina's performance tuning.  It's fairly pessimal in
//  that it uses geometry/co-ordintes, which will force the fj flattener to process the big arrays of numbers in
//  each line.  A high proportion of typical Quamina workloads should run faster.
func TestCityLots(t *testing.T) {
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
	wanted := map[X]int{
		"CRANLEIGH": 7,
		"17TH Even": 836,
		"Geometry":  2,
		"0011008":   1,
	}

	var err error
	q := New()
	for i := range names {
		err = q.AddPattern(names[i], patterns[i])
		if err != nil {
			t.Error("Addpattern: " + err.Error())
		}
	}
	results := make(map[X]int)

	lines := getCityLotsLines(t)
	before := time.Now()
	for _, line := range lines {
		matches, err := q.MatchesForJSONEvent(line)
		if err != nil {
			t.Error("Matches4JSON: " + err.Error())
		}
		for _, match := range matches {
			count, ok := results[match]
			if !ok {
				count = 0
			}
			results[match] = count + 1
		}
	}
	fmt.Println()

	elapsed := float64(time.Since(before).Milliseconds())
	perSecond := float64(cityLotsLineCount) / (elapsed / 1000.0)
	fmt.Printf("%.2f matches/second\n\n", perSecond)

	if perSecond < thresholdPerformance {
		message1 := fmt.Sprintf("Events-per-second benchmark ran at %.0f events per second, below threshold of %.0f.",
			perSecond, thresholdPerformance)
		message2 := `
			It may be that re-running the benchmark test will address this, or it may be that you're running on a machine
			that is slower than the one the software was developed on, in which case you might want to readjust the
			"thresholdPerformance" constant. However, it may be that you made a change that reduced the throughput of the
			library, which would be unacceptable.`
		t.Errorf(message1 + message2)
	}

	if len(results) != len(wanted) {
		t.Errorf("got %d results, wanted %d", len(results), len(wanted))
	}
	for match, count := range results {
		if count != wanted[match] {
			t.Errorf("For %s, wanted=%d, result=%d", match, wanted[match], count)
		}
	}
}
