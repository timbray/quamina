package quamina

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

func updateTree(m *Matcher, use37 bool, t *testing.T, ch chan (string)) {
	var pattern string
	var val string
	if use37 {
		val = fmt.Sprintf("%f", 37.0+rand.Float64())
		pattern = fmt.Sprintf(`{ "geometry": { "coordinates": [ %s ] } }`, val)
	} else {
		val = fmt.Sprintf(`"%d"`, rand.Int())
		pattern = fmt.Sprintf(`{ "properties": { "STREET": [ %s ] } }`, val)
	}
	err := m.AddPattern(val, pattern)
	if err != nil {
		t.Error("Concurrent: " + err.Error())
	}
	ch <- val
}

func TestConcurrency(t *testing.T) {
	const UpdateLines = 250

	// this is a cut/paste of TestCityLots, except for every few lines we add another rule to the matcher,
	//  focusing on the fields that are being used by the patterns. The idea is to exercise concurrent
	//  update and use of the automaton
	// I was initially surprised that adding 860 or so changes to the automaton while it's running doesn't seem to
	//  cause any decrease in performance. But I guess it splits out very cleanly onto another core and really
	//  doesn't steal any resources from the thread doing the Match calls
	file, err := os.Open("../test_data/citylots.jlines")
	if err != nil {
		t.Error("Can't open file: " + err.Error())
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

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

	scanner := bufio.NewScanner(file)
	buf := make([]byte, oneMeg)
	scanner.Buffer(buf, oneMeg)

	m := NewMatcher()
	for i := range names {
		err = m.AddPattern(names[i], patterns[i])
		if err != nil {
			t.Error("Addpattern: " + err.Error())
		}
	}
	results := make(map[X]int)

	lineCount := 0
	var lines [][]byte
	for scanner.Scan() {
		lineCount++
		lines = append(lines, []byte(scanner.Text()))
	}
	use37 := true
	lineCount = 0
	before := time.Now()
	ch := make(chan (string), 1000)
	for _, line := range lines {
		matches, err := m.MatchesForJSONEvent(line)
		if err != nil {
			t.Error("Matches4JSON: " + err.Error())
		}
		lineCount++
		if lineCount%UpdateLines == 0 {
			use37 = !use37
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
	fmt.Println()

	elapsed := float64(time.Now().Sub(before).Milliseconds())
	perSecond := float64(lineCount) / (elapsed / 1000.0)
	fmt.Printf("%.2f matches/second with updates\n\n", perSecond)

	err = scanner.Err()
	if err != nil {
		t.Error("Scanner error: " + err.Error())
	}

	if len(results) != len(wanted) {
		t.Errorf("got %d results, wanted %d", len(results), len(wanted))
	}
	for match, count := range results {
		if count != wanted[match] {
			t.Errorf("For %s, wanted=%d, result=%d", match, wanted[match], count)
		}
	}

	// now we go back and make sure that all those AddPattern calls actually made it into the Matcher
	close(ch)
	for val := range ch {
		var event string
		if val[0] == '"' {
			event = fmt.Sprintf(`{"properties": { "STREET": %s} }`, val)
		} else {
			event = fmt.Sprintf(`{"geometry": { "coordinates": [ %s ] } }`, val)
		}
		var matches []X
		matches, err = m.MatchesForJSONEvent([]byte(event))
		if err != nil {
			t.Error("after concur: " + err.Error())
		}
		if len(matches) != 1 || matches[0] != val {
			t.Error("problem with: " + val)
		}
	}
}
