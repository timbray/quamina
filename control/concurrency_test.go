package control

import (
	"fmt"
	"github.com/timbray/quamina/core"
	"github.com/timbray/quamina/flattener"
	"math/rand"
	"testing"
	"time"
)

func updateTree(m *core.CoreMatcher, use37 bool, t *testing.T, ch chan string) {
	var pattern string
	var val string
	if use37 {
		val = fmt.Sprintf("%f", 37.0+rand.Float64())
		pattern = fmt.Sprintf(`{ "geometry": { "coordinates": [ %s ] } }`, val)
	} else {
		val = fmt.Sprintf(`"%d"`, rand.Int())
		pattern = fmt.Sprintf(`{ "properties": { "STREET": [ %s ] } }`, val)
		/* TODO: alternate literal and shellstyle addition
		val = fmt.Sprintf(`"*%d"`, rand.Int())
		pattern = fmt.Sprintf(`{ "properties": { "STREET": [ {"shellstyle": %s } ] } }`, val)
		*/
	}
	err := m.AddPattern(val, pattern)
	if err != nil {
		t.Error("Concurrent: " + err.Error())
	}
	ch <- val
}

func TestConcurrency(t *testing.T) {
	const UpdateLines = 250

	// this is a cut/paste of TestCityLots, except for every few lines we add another pattern to the matcher,
	//  focusing on the fields that are being used by the patterns. The idea is to exercise concurrent
	//  update and use of the automaton
	// I was initially surprised that adding 860 or so changes to the automaton while it's running doesn't seem to
	//  cause any decrease in performance. But I guess it splits out very cleanly onto another core and really
	//  doesn't steal any resources from the thread doing the Match calls
	lines := getCityLotsLines(t)

	patterns := []string{
		`{ "properties": { "STREET": [ "CRANLEIGH" ] } }`,
		`{ "properties": { "STREET": [ { "shellstyle": "B*K"} ] } }`,
		`{ "properties": { "STREET": [ "17TH" ], "ODD_EVEN": [ "E"] } }`,
		`{ "geometry": { "coordinates": [ 37.807807921694092 ] } }`,
		`{ "properties": { "MAPBLKLOT": ["0011008"], "BLKLOT": ["0011008"]},  "geometry": { "coordinates": [ 37.807807921694092 ] } } `,
	}
	names := []string{
		"CRANLEIGH",
		"shellstyle",
		"17TH Even",
		"Geometry",
		"0011008",
	}
	wanted := map[core.X]int{
		"CRANLEIGH":  7,
		"shellstyle": 746,
		"17TH Even":  836,
		"Geometry":   2,
		"0011008":    1,
	}

	var err error
	m := core.NewCoreMatcher()
	for i := range names {
		err = m.AddPattern(names[i], patterns[i])
		if err != nil {
			t.Error("Addpattern: " + err.Error())
		}
	}
	results := make(map[core.X]int)

	use37 := true
	lineCount := 0
	before := time.Now()
	ch := make(chan string, 1000)
	sent := 0
	fj := flattener.NewFJ()
	for _, line := range lines {
		fields, err := fj.Flatten(line, m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err := m.MatchesForFields(fields)
		if err != nil {
			t.Error("Matches4JSON: " + err.Error())
		}
		lineCount++
		if lineCount%UpdateLines == 0 {
			use37 = !use37
			sent++
			go updateTree(m, use37, t, ch)
		}
		for _, match := range matches {
			count, ok := results[match]
			if !ok {
				count = 0
			}
			results[match] = count + 1
		}
	}

	elapsed := float64(time.Since(before).Milliseconds())
	perSecond := float64(lineCount) / (elapsed / 1000.0)
	fmt.Printf("\n%.2f matches/second with updates\n\n", perSecond)

	if len(results) != len(wanted) {
		t.Errorf("got %d results, wanted %d", len(results), len(wanted))
	}
	for match, count := range results {
		if count != wanted[match] {
			t.Errorf("For %s, wanted=%d, result=%d", match, wanted[match], count)
		}
	}

	// now we go back and make sure that all those AddPattern calls actually made it into the Matcher
	fj = flattener.NewFJ()
	for i := 0; i < sent; i++ {
		val := <-ch
		var event string
		if val[0] == '"' {
			event = fmt.Sprintf(`{"properties": { "STREET": %s} }`, val)
		} else {
			event = fmt.Sprintf(`{"geometry": { "coordinates": [ %s ] } }`, val)
		}
		var matches []core.X
		fields, err := fj.Flatten([]byte(event), m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err = m.MatchesForFields(fields)
		if err != nil {
			t.Error("after concur: " + err.Error())
		}
		if len(matches) != 1 || matches[0] != val {
			t.Error("problem with: " + val)
		}
	}
	close(ch)
}
