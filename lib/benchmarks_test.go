package quamina

import (
	"bufio"
	"fmt"
	"os"
	"testing"
	"time"
)

/*
properties.17TH:1579
properties.BALANCE: 1
properties.CRANLEIGH: 7
properties.ODD_EVEN": "O"': 97561
properties.BLOCK_NUM  0256T: 810
properties.geometry.coordinates: 37.80805250792249: 2
*/

const oneMeg = 1024 * 1024

func TestCRANLEIGH(t *testing.T) {

	jCranleigh := `{ "type": "Feature", "properties": { "MAPBLKLOT": "7222001", "BLKLOT": "7222001", "BLOCK_NUM": "7222", "LOT_NUM": "001", "FROM_ST": "1", "TO_ST": "1", "STREET": "CRANLEIGH", "ST_TYPE": "DR", "ODD_EVEN": "O" }, "geometry": { "type": "Polygon", "coordinates": [ [ [ -122.472773074480756, 37.73439178240811, 0.0 ], [ -122.47278111723567, 37.73451247621523, 0.0 ], [ -122.47242608711845, 37.73452184591072, 0.0 ], [ -122.472418368113281, 37.734401143064396, 0.0 ], [ -122.472773074480756, 37.73439178240811, 0.0 ] ] ] } }`
	j108492 := `{ "type": "Feature", "properties": { "MAPBLKLOT": "0011008", "BLKLOT": "0011008", "BLOCK_NUM": "0011", "LOT_NUM": "008", "FROM_ST": "500", "TO_ST": "550", "STREET": "BEACH", "ST_TYPE": "ST", "ODD_EVEN": "E" }, "geometry": { "type": "Polygon", "coordinates": [ [ [ -122.418114728237924, 37.807058866808987, 0.0 ], [ -122.418261722815416, 37.807807921694092, 0.0 ], [ -122.417544151208375, 37.807900142836701, 0.0 ], [ -122.417397010603693, 37.807150305505004, 0.0 ], [ -122.418114728237924, 37.807058866808987, 0.0 ] ] ] } }`
	m := NewMatcher()
	pCranleigh := `{ "properties": { "STREET": [ "CRANLEIGH" ] } }`
	p108492 := `{ "properties": { "MAPBLKLOT": ["0011008"], "BLKLOT": ["0011008"]},  "geometry": { "coordinates": [ 37.807807921694092 ] } } `

	err := m.AddPattern("CRANLEIGH", pCranleigh)
	if err != nil {
		t.Error("!? " + err.Error())
	}
	err = m.AddPattern("108492", p108492)
	if err != nil {
		t.Error("!? " + err.Error())
	}
	var matches []X
	lines := [][]byte{[]byte(jCranleigh), []byte(j108492)}
	// lines := [][]byte{[]byte(j108492)}
	for _, line := range lines {
		mm, err := m.MatchesForJSONEvent(line)
		if err != nil {
			t.Error("OOPS " + err.Error())
		}
		matches = append(matches, mm...)
	}
	wanteds := []string{"CRANLEIGH", "108492"}
	for i, wanted := range wanteds {
		g := matches[i].(string)
		if wanted != g {
			t.Errorf("wanted %s got %s", wanted, g)
		}
	}
}

const thresholdPerformance = 120000.0

func TestCityLots(t *testing.T) {
	file, err := os.Open("../test_data/citylots.jlines")
	if err != nil {
		t.Error("Can't open file: " + err.Error())
	}
	defer file.Close()

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
	lineCount = 0
	before := time.Now()
	for _, line := range lines {
		matches, err := m.MatchesForJSONEvent(line)
		if err != nil {
			t.Error("Matches4JSON: " + err.Error())
		}
		lineCount++
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

}
