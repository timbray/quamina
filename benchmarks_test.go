package quamina

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

const oneMeg = 1024 * 1024

var cityLotsLock sync.Mutex
var cityLotsLines [][]byte
var cityLotsLineCount int

func getCityLotsLines(t *testing.T) [][]byte {
	cityLotsLock.Lock()
	defer cityLotsLock.Unlock()
	if cityLotsLines != nil {
		return cityLotsLines
	}
	file, err := os.Open("testdata/citylots.jlines.gz")
	if err != nil {
		t.Error("Can't open citlots.jlines.gz: " + err.Error())
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	zr, err := gzip.NewReader(file)
	if err != nil {
		t.Error("Can't open zip reader: " + err.Error())
	}

	scanner := bufio.NewScanner(zr)
	buf := make([]byte, oneMeg)
	scanner.Buffer(buf, oneMeg)
	for scanner.Scan() {
		cityLotsLineCount++
		cityLotsLines = append(cityLotsLines, []byte(scanner.Text()))
	}
	return cityLotsLines
}

func TestCRANLEIGH(t *testing.T) {

	jCranleigh := `{ "type": "Feature", "properties": { "MAPBLKLOT": "7222001", "BLKLOT": "7222001", "BLOCK_NUM": "7222", "LOT_NUM": "001", "FROM_ST": "1", "TO_ST": "1", "STREET": "CRANLEIGH", "ST_TYPE": "DR", "ODD_EVEN": "O" }, "geometry": { "type": "Polygon", "coordinates": [ [ [ -122.472773074480756, 37.73439178240811, 0.0 ], [ -122.47278111723567, 37.73451247621523, 0.0 ], [ -122.47242608711845, 37.73452184591072, 0.0 ], [ -122.472418368113281, 37.734401143064396, 0.0 ], [ -122.472773074480756, 37.73439178240811, 0.0 ] ] ] } }`
	j108492 := `{ "type": "Feature", "properties": { "MAPBLKLOT": "0011008", "BLKLOT": "0011008", "BLOCK_NUM": "0011", "LOT_NUM": "008", "FROM_ST": "500", "TO_ST": "550", "STREET": "BEACH", "ST_TYPE": "ST", "ODD_EVEN": "E" }, "geometry": { "type": "Polygon", "coordinates": [ [ [ -122.418114728237924, 37.807058866808987, 0.0 ], [ -122.418261722815416, 37.807807921694092, 0.0 ], [ -122.417544151208375, 37.807900142836701, 0.0 ], [ -122.417397010603693, 37.807150305505004, 0.0 ], [ -122.418114728237924, 37.807058866808987, 0.0 ] ] ] } }`
	m := newCoreMatcher()
	pCranleigh := `{ "properties": { "STREET": [ "CRANLEIGH" ] } }`
	p108492 := `{ "properties": { "MAPBLKLOT": ["0011008"], "BLKLOT": ["0011008"]},  "geometry": { "coordinates": [ 37.807807921694092 ] } } `

	err := m.addPattern("CRANLEIGH", pCranleigh)
	if err != nil {
		t.Error("!? " + err.Error())
	}
	err = m.addPattern("108492", p108492)
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

// - restore when we've got multi-glob working
/*
func TestMySoftwareHatesMe(t *testing.T) {
	line := `{ "type": "Feature", "properties": { "STREET": "BELVEDERE" }  }`
	m := newCoreMatcher()
	Bpat := `{"properties": {"STREET":[ {"shellstyle": "B*"} ] } }`
	EEEpat := `{"properties": {"STREET":[ {"shellstyle": "*E*E*E*"} ] } }`

	if m.addPattern("EEE", EEEpat) != nil {
		t.Error("Huh add?")
	}
	matches, err := m.MatchesForEvent([]byte(line))
	if len(matches) != 1 || matches[0] != "EEE" {
		t.Error("Failed to match EEE")
	}

	m = newCoreMatcher()
	_ = m.addPattern("B", Bpat)
	_ = m.addPattern("EEE", EEEpat)

	matches, err = m.MatchesForEvent([]byte(line))
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
	lines := getCityLotsLines(t)
	m := newCoreMatcher()

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
		err := m.addPattern(letter, pat)
		if err != nil {
			t.Errorf("err on %c: %s", letter, err.Error())
		}
	}

	/*
		for funk := range funky {
			err := m.addPattern(funk, funk.(string))
			if err != nil {
				t.Errorf("err on %s: %s", funk, err.Error())
			}
		}
	*/
	fmt.Println(matcherStats(m))

	lCounts := make(map[X]int)
	before := time.Now()
	fj := newJSONFlattener()
	for _, line := range lines {
		fields, err := fj.Flatten(line, m)
		if err != nil {
			t.Error("Flatten: " + err.Error())
		}
		matches, err := m.matchesForFields(fields)
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
	elapsed := float64(time.Since(before).Milliseconds())
	perSecond := float64(cityLotsLineCount) / (elapsed / 1000.0)
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
	m := newCoreMatcher()
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
			err := m.addPattern(pName, pat)
			if err != nil {
				t.Error("addPattern " + err.Error())
			}
		}
	}
	runtime.ReadMemStats(&msAfter)
	delta := 1.0 / 1000000.0 * float64(msAfter.Alloc-msBefore.Alloc)
	fmt.Printf("before %d, after %d, delta %f\n", msBefore.Alloc, msAfter.Alloc, delta)
	fmt.Println("stats:" + matcherStats(m))
	elapsed := float64(time.Since(before).Milliseconds())
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
	file, err := os.Open("testdata/wwords.txt")
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
