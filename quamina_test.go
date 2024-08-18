package quamina

import (
	"fmt"
	"testing"
	"time"
)

func TestCopy(t *testing.T) {
	q, err := New()
	if err != nil {
		t.Error("New? " + err.Error())
	}
	q2 := q.Copy()
	if q2.matcher != q.matcher || q2.flattener == q.flattener {
		t.Error("improper copy")
	}
}

func TestNewQOptions(t *testing.T) {
	var q *Quamina
	var err error
	var ok bool
	q, err = New(WithMediaType("application/json"))
	if err != nil {
		t.Error(err.Error())
	}
	_, ok = q.flattener.(*flattenJSON)
	if !ok {
		t.Error("Should be flattenJSON")
	}
	_, err = New(WithMediaType("text/html"))
	if err == nil {
		t.Error("accepted text/html")
	}
	q, err = New(WithFlattener(newJSONFlattener()))
	if err != nil {
		t.Error(err.Error())
	}
	_, ok = q.flattener.(*flattenJSON)
	if !ok {
		t.Error("should be flattenJSON")
	}
	_, err = New(WithFlattener(nil))
	if err == nil {
		t.Error("accepted nil flattener")
	}
	_, err = New(WithPatternStorage(nil))
	if err == nil {
		t.Error("accepted WIthPatternStorage")
	}
	q, err = New(WithPatternDeletion(true))
	if err != nil {
		t.Error("didn't take PatternDeletion(true")
	}
	_, ok = q.matcher.(*prunerMatcher)
	if !ok {
		t.Error("should be pruner")
	}
	q, err = New(WithPatternDeletion(false))
	if err != nil {
		t.Error("didn't take PatternDeletion(false")
	}
	_, ok = q.matcher.(*coreMatcher)
	if !ok {
		t.Error("should be core")
	}

	_, err = New(WithPatternDeletion(true), WithPatternDeletion(true))
	if err == nil {
		t.Error("allowed 2 patternDel" + err.Error())
	}
	_, err = New(WithFlattener(newJSONFlattener()), WithFlattener(newJSONFlattener()))
	if err == nil {
		t.Error("allowed 2 flatteners" + err.Error())
	}
	_, err = New(WithMediaType("application/json"), WithMediaType("application/json"))
	if err == nil {
		t.Error("allowed 2 mediatypes" + err.Error())
	}
	_, err = New(WithMediaType("application/json"), WithFlattener(newJSONFlattener()))
	if err == nil {
		t.Error("allowed flattener and media type" + err.Error())
	}
	q, err = New(WithPatternDeletion(true))
	if err != nil {
		t.Error("WithPatternDeletion failed: " + err.Error())
	}
	_, ok = q.matcher.(*prunerMatcher)
	if !ok {
		t.Error("not a pruner matcher")
	}
	_, ok = q.flattener.(*flattenJSON)
	if !ok {
		t.Error("flattener not for JSON")
	}
}

// reduced to allow unit tests in slow GitHub actions to pass
// const thresholdPerformance = 120000.0
const thresholdPerformance = 1.0

// TestCityLots is the benchmark that was used in most of Quamina's performance tuning.  It's fairly pessimal in
// that it uses geometry/co-ordintes, which will force the fj flattener to process the big arrays of numbers in
// each line.  A high proportion of typical Quamina workloads should run faster.
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
	q, err := New()
	if err != nil {
		t.Error("New(): " + err.Error())
	}
	for i := range names {
		err = q.AddPattern(names[i], patterns[i])
		if err != nil {
			t.Error("Addpattern: " + err.Error())
		}
	}
	results := make(map[X]int)
	fmt.Println(matcherStats(q.matcher.(*coreMatcher)))

	lines := getCityLotsLines(t)
	before := time.Now()
	for _, line := range lines {
		matches, err := q.MatchesForEvent(line)
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
		t.Error(message1 + message2)
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
