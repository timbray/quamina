package quamina

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

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

// TestCityLots is the benchmark that was used in most of Quamina's performance tuning.  It's fairly pessimal in
//  that it uses geometry/co-ordintes, which will force the fj flattener to process the big arrays of numbers in
//  each line.  A high proportion of typical Quamina workloads should run faster.
func TestCityLots(t *testing.T) {
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
	fj := NewFJ(m)
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
		matches, err := fj.FlattenAndMatch(line)
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

// - restore when we've got multi-glob working
/*
func TestMySoftwareHatesMe(t *testing.T) {
	line := `{ "type": "Feature", "properties": { "STREET": "BELVEDERE" }  }`
	m := NewMatcher()
	Bpat := `{"properties": {"STREET":[ {"shellstyle": "B*"} ] } }`
	EEEpat := `{"properties": {"STREET":[ {"shellstyle": "*E*E*E*"} ] } }`

	if m.AddPattern("EEE", EEEpat) != nil {
		t.Error("Huh add?")
	}
	matches, err := m.MatchesForJSONEvent([]byte(line))
	if len(matches) != 1 || matches[0] != "EEE" {
		t.Error("Failed to match EEE")
	}

	m = NewMatcher()
	_ = m.AddPattern("B", Bpat)
	_ = m.AddPattern("EEE", EEEpat)

	matches, err = m.MatchesForJSONEvent([]byte(line))
	if err != nil {
		t.Error("Huh? " + err.Error())
	}
	if !containsX(matches, "B") {
		t.Error("no match for B")
	}
	if !containsX(matches, "EEE") {
		t.Error("no match for EEE")
	}
}
*/

// exercise shellstyle matching a little, is much faster than TestCityLots because it's only working wth one field
func TestBigShellStyle(t *testing.T) {
	file, err := os.Open("../test_data/citylots.jlines")
	if err != nil {
		t.Error("Can't open file: " + err.Error())
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)
	buf := make([]byte, oneMeg)
	scanner.Buffer(buf, oneMeg)

	m := NewMatcher()

	wanted := map[X]int{
		"A": 5883, "B": 12765, "C": 14824, "D": 6124, "E": 3402, "F": 7999, "G": 8555,
		"H": 7829, "I": 1330, "J": 3853, "K": 2595, "L": 8168, "M": 14368, "N": 3710,
		"O": 3413, "P": 11250, "Q": 719, "R": 4354, "S": 13255, "T": 4209, "U": 4636,
		"V": 4322, "W": 4162, "X": 0, "Y": 721, "Z": 25,
	}

	/* - restore when we've got multi-glob working
	funky := map[X]int{
		`{"properties": {"STREET":[ {"shellstyle": "N*P*"} ] } }`:    927,
		`{"properties": {"STREET":[ {"shellstyle": "*E*E*E*"} ] } }`: 1212,
	}
	*/

	for letter := range wanted {
		pat := fmt.Sprintf(`{"properties": {"STREET":[ {"shellstyle": "%s*"} ] } }`, letter)
		err := m.AddPattern(letter, pat)
		if err != nil {
			t.Errorf("err on %c: %s", letter, err.Error())
		}
	}

	/*
		for funk := range funky {
			err := m.AddPattern(funk, funk.(string))
			if err != nil {
				t.Errorf("err on %s: %s", funk, err.Error())
			}
		}
	*/
	fmt.Println(matcherStats(m))

	lineCount := 0
	var lines [][]byte
	for scanner.Scan() {
		lineCount++
		lines = append(lines, []byte(scanner.Text()))
	}
	lCounts := make(map[X]int)
	before := time.Now()
	fj := NewFJ(m)
	for _, line := range lines {
		matches, err := fj.FlattenAndMatch(line)
		if err != nil {
			t.Error("Matches4JSON: " + err.Error())
		}

		for _, match := range matches {
			lc, ok := lCounts[match]
			if ok {
				lCounts[match] = lc + 1
			} else {
				lCounts[match] = 1
			}
		}
	}
	elapsed := float64(time.Now().Sub(before).Milliseconds())
	perSecond := float64(lineCount) / (elapsed / 1000.0)
	fmt.Printf("%.2f matches/second with letter patterns\n\n", perSecond)

	for k, wc := range wanted {
		if lCounts[k] != wc {
			t.Errorf("for %s wanted %d got %d", k, wc, lCounts[k])
		}
	}
	/*
		for k, wc := range funky {
			if lCounts[k] != wc {
				t.Errorf("for %s wanted %d got %d", k, wc, lCounts[k])
			}
		}

	*/
}

// TestPatternAddition adds a whole lot of string-only rules as fast as possible  The profiler says that the
//  performance is totally doinated by the garbage-collector thrashing, in particular it has to allocate
//  ~220K smallTables.  Tried https://blog.twitch.tv/en/2019/04/10/go-memory-ballast-how-i-learnt-to-stop-worrying-and-love-the-heap/
//  but it doesn't seem to help.
//  TODO: Find a way to allocate less tables.
//  TODO: Add shellstyle patterns
func TestPatternAddition(t *testing.T) {
	w := worder{0, readWWords(t)}

	var msBefore, msAfter runtime.MemStats

	// now we're going to add 200 fields, 200 values, so 40K name/value pairs. There might be some duplication?
	m := NewMatcher()
	before := time.Now()
	fieldCount := 0
	runtime.ReadMemStats(&msBefore)
	debug.SetGCPercent(500)
	for x1 := 0; x1 < 10; x1++ {
		for x2 := 0; x2 < 20; x2++ {
			pat := fmt.Sprintf(`{"%s": { "%s": [ "%s"`, w.next(), w.next(), w.next())
			for x3 := 0; x3 < 199; x3++ {
				pat = pat + fmt.Sprintf(`, "%s"`, w.next())
			}
			fieldCount += 200
			pat = pat + `] } }`
			pName := string(w.next()) + string(w.next())
			err := m.AddPattern(pName, pat)
			if err != nil {
				t.Error("addPattern " + err.Error())
			}
		}
	}
	runtime.ReadMemStats(&msAfter)
	delta := 1.0 / 1000000.0 * float64(msAfter.Alloc-msBefore.Alloc)
	fmt.Printf("before %d, after %d, delta %f\n", msBefore.Alloc, msAfter.Alloc, delta)
	fmt.Println("stats:" + matcherStats(m))
	elapsed := float64(time.Now().Sub(before).Milliseconds())
	perSecond := float64(fieldCount) / (elapsed / 1000.0)
	fmt.Printf("%.2f fields/second\n\n", perSecond)
}

type worder struct {
	index int
	lines [][]byte
}

func (w *worder) next() []byte {
	w.index += 761 // relatively prime with the number of lines
	w.index = w.index % len(w.lines)
	return w.lines[w.index]
}

func readWWords(t *testing.T) [][]byte {
	// that's a list from the Wordle source code with a few erased to get a prime number
	file, err := os.Open("../test_data/wwords.txt")
	if err != nil {
		t.Error("Can't open file: " + err.Error())
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, oneMeg)
	scanner.Buffer(buf, oneMeg)

	lineCount := 0
	var lines [][]byte
	for scanner.Scan() {
		lineCount++
		lines = append(lines, []byte(scanner.Text()))
	}
	return lines
}
