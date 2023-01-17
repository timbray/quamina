package quamina

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

var (
	cl2Lock      sync.Mutex
	cl2Lines     [][]byte
	cl2LineCount int
)

/* This test adopted, with thanks, from aws/event-ruler */

func getCL2Lines(t *testing.T) [][]byte {
	t.Helper()

	cl2Lock.Lock()
	defer cl2Lock.Unlock()
	if cl2Lines != nil {
		return cl2Lines
	}
	file, err := os.Open("testdata/citylots2.json.gz")
	if err != nil {
		t.Fatalf("Can't open citylots2.json.gz: %v", err.Error())
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	zr, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Can't open zip reader: %v", err.Error())
	}

	scanner := bufio.NewScanner(zr)
	buf := make([]byte, oneMeg)
	scanner.Buffer(buf, oneMeg)
	for scanner.Scan() {
		cl2LineCount++
		cl2Lines = append(cl2Lines, []byte(scanner.Text()))
	}
	return cl2Lines
}

func TestCl2(t *testing.T) {
	exactRules := []string{
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ \"1430022\" ]\n" +
			"  }" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ \"2607117\" ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ \"2607218\" ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ \"3745012\" ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ \"VACSTWIL\" ]\n" +
			"  }\n" +
			"}",
	}
	exactMatches := []int{1, 101, 35, 655, 1}

	anythingButRules := []string{
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"anything-but\": [ \"FULTON\" ] } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"anything-but\": [ \"MASON\" ] } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"ST_TYPE\": [ { \"anything-but\": [ \"ST\" ] } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"geometry\": {\n" +
			"    \"type\": [ {\"anything-but\": [ \"Polygon\" ] } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"FROM_ST\": [ { \"anything-but\": [ \"441\" ] } ]\n" +
			"  }\n" +
			"}",
	}
	anythingButMatches := []int{211158, 210411, 96682, 120, 210615}

	/* will add when we have numeric
	complexArraysRules := []string{
		"{\n" +
			"  \"geometry\": {\n" +
			"    \"type\": [ \"Polygon\" ],\n" +
			"    \"coordinates\": {\n" +
			"      \"x\": [ { \"numeric\": [ \"=\", -122.42916360922355 ] } ]\n" +
			"    }\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"geometry\": {\n" +
			"    \"type\": [ \"MultiPolygon\" ],\n" +
			"    \"coordinates\": {\n" +
			"      \"y\": [ { \"numeric\": [ \"=\", 37.729900216217324 ] } ]\n" +
			"    }\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"geometry\": {\n" +
			"    \"coordinates\": {\n" +
			"      \"x\": [ { \"numeric\": [ \"<\", -122.41600944012424 ] } ]\n" +
			"    }\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"geometry\": {\n" +
			"    \"coordinates\": {\n" +
			"      \"x\": [ { \"numeric\": [ \">\", -122.41600944012424 ] } ]\n" +
			"    }\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"geometry\": {\n" +
			"    \"coordinates\": {\n" +
			"      \"x\": [ { \"numeric\": [ \">\",  -122.46471267081272, \"<\", -122.4063085128395 ] } ]\n" +
			"    }\n" +
			"  }\n" +
			"}",
	bm = newBenchmarker()
	bm.addRules(complexArraysRules, complexArraysMatches)
	fmt.Printf("COMPLEX_ARRAYS events/sec: %.1f\n", bm.run(lines, t))
	}
	complexArraysMatches := []int{227, 2, 149444, 64368, 127485}
	*/

	lines := getCL2Lines(t)
	fmt.Printf("lines: %d\n", len(lines))

	// initial run to stabilize memory
	bm := newBenchmarker()
	bm.addRules(exactRules, exactMatches)

	bm.run(t, lines)

	bm = newBenchmarker()
	bm.addRules(exactRules, exactMatches)
	fmt.Printf("EXACT events/sec: %.1f\n", bm.run(t, lines))

	bm = newBenchmarker()
	bm.addRules(anythingButRules, anythingButMatches)
	fmt.Printf("ANYTHING-BUT events/sec: %.1f\n", bm.run(t, lines))
}

type benchmarker struct {
	wanted map[X]int
	q      *Quamina
}

func newBenchmarker() *benchmarker {
	q, _ := New()
	return &benchmarker{q: q, wanted: make(map[X]int)}
}

func (bm *benchmarker) addRules(rules []string, wanted []int) {
	for i, rule := range rules {
		rname := fmt.Sprintf("r%d", i)
		_ = bm.q.AddPattern(rname, rule)
		bm.wanted[rname] = wanted[i]
	}
	fmt.Println(matcherStats(bm.q.matcher.(*coreMatcher)))
}

func (bm *benchmarker) run(t *testing.T, events [][]byte) float64 {
	t.Helper()
	gotMatches := make(map[X]int)
	before := time.Now()
	for _, event := range events {
		matches, err := bm.q.MatchesForEvent(event)
		if err != nil {
			t.Error("m4e: " + err.Error())
		}
		for _, match := range matches {
			got, ok := gotMatches[match]
			if !ok {
				got = 1
			} else {
				got++
			}
			gotMatches[match] = got
		}
	}
	elapsed := float64(time.Since(before).Milliseconds())
	eps := float64(cl2LineCount) / (elapsed / 1000.0)

	for match := range gotMatches {
		if bm.wanted[match] != gotMatches[match] {
			t.Errorf("for %s wanted %d got %d", match, bm.wanted[match], gotMatches[match])
		}
	}
	for match := range bm.wanted {
		got, ok := gotMatches[match]
		if !ok {
			got = 0
		}
		if bm.wanted[match] != got {
			t.Errorf("for %s wanted %d got %d", match, bm.wanted[match], got)
		}
	}
	return eps
}
