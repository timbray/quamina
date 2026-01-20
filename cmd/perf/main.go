package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"quamina.net/go/quamina"
)

var (
	cl2Lines     [][]byte
	cl2LineCount int
)

// Data and Rules
var (
	anythingButRules = []string{
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
	anythingButMatches = []int{211158, 210411, 96682, 120, 210615}
	exactRules         = []string{
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
	exactMatches = []int{1, 101, 35, 655, 1}
	prefixRules  = []string{
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"prefix\": \"AC\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"prefix\": \"BL\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"prefix\": \"DR\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"prefix\": \"FU\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"prefix\": \"RH\" } ]\n" +
			"  }\n" +
			"}",
	}
	prefixMatches   = []int{24, 442, 38, 2387, 328}
	shellstyleRules = []string{
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ { \"shellstyle\": \"143*\" } ]\n" +
			"  }" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ { \"shellstyle\": \"2*0*1*7\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ { \"shellstyle\": \"*218\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ { \"shellstyle\": \"3*5*2\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"MAPBLKLOT\": [ { \"shellstyle\": \"VA*IL\" } ]\n" +
			"  }\n" +
			"}",
	}
	shellstyleMatches     = []int{490, 713, 43, 2540, 1}
	equalsIgnoreCaseRules = []string{
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"equals-ignore-case\": \"jefferson\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"equals-ignore-case\": \"bEaCh\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"equals-ignore-case\": \"HyDe\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"equals-ignore-case\": \"CHESTNUT\" } ]\n" +
			"  }\n" +
			"}",
		"{\n" +
			"  \"properties\": {\n" +
			"    \"ST_TYPE\": [ { \"equals-ignore-case\": \"st\" } ]\n" +
			"  }\n" +
			"}",
	}

	equalsIgnoreCaseMatches = []int{131, 211, 1758, 825, 116386}
	regexpRules             = []string{
		"{\n" +
			"  \"properties\": {\n" +
			"    \"STREET\": [ { \"regexp\": \"B..CH\" } ]\n" +
			"  }\n" +
			"}",
	}
	regexpMatches = []int{220}
)

func main() {
	// 1. Load Data
	fmt.Println("Loading data...")
	getCL2Lines()
	fmt.Printf("Loaded %d lines\n", len(cl2Lines))

	// 2. Start Profiling
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	// 3. Run Benchmarks
	bm := newBenchmarker()
	// initial run to stabilize memory
	bm.addRules(exactRules, exactMatches)
	bm.run(cl2Lines)

	bm = newBenchmarker()
	bm.addRules(exactRules, exactMatches)
	fmt.Printf("EXACT events/sec: %.1f\n", bm.run(cl2Lines))

	bm = newBenchmarker()
	bm.addRules(prefixRules, prefixMatches)
	fmt.Printf("PREFIX events/sec: %.1f\n", bm.run(cl2Lines))

	bm = newBenchmarker()
	bm.addRules(anythingButRules, anythingButMatches)
	fmt.Printf("ANYTHING-BUT events/sec: %.1f\n", bm.run(cl2Lines))

	bm = newBenchmarker()
	bm.addRules(shellstyleRules, shellstyleMatches)
	fmt.Printf("SHELLSTYLE events/sec: %.1f\n", bm.run(cl2Lines))

	bm = newBenchmarker()
	bm.addRules(equalsIgnoreCaseRules, equalsIgnoreCaseMatches)
	fmt.Printf("EQUALS_IGNORE-CASE events/sec: %.1f\n", bm.run(cl2Lines))

	bm = newBenchmarker()
	bm.addRules(regexpRules, regexpMatches)
	fmt.Printf("REGEXP events/sec: %.1f\n", bm.run(cl2Lines))

	// Memory Profile
	fMem, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	defer fMem.Close()
	if err := pprof.WriteHeapProfile(fMem); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
}

func getCL2Lines() {
	file, err := os.Open("../../testdata/citylots2.json.gz")
	if err != nil {
		log.Fatalf("Can't open citylots2.json.gz: %v", err.Error())
	}
	defer file.Close()
	zr, err := gzip.NewReader(file)
	if err != nil {
		log.Fatalf("Can't open zip reader: %v", err.Error())
	}

	scanner := bufio.NewScanner(zr)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		cl2LineCount++
		cl2Lines = append(cl2Lines, []byte(scanner.Text()))
	}
}

type benchmarker struct {
	wanted map[quamina.X]int
	q      *quamina.Quamina
}

func newBenchmarker() *benchmarker {
	q, _ := quamina.New()
	return &benchmarker{q: q, wanted: make(map[quamina.X]int)}
}

func (bm *benchmarker) addRules(rules []string, wanted []int) {
	for i, rule := range rules {
		rname := fmt.Sprintf("r%d", i)
		_ = bm.q.AddPattern(rname, rule)
		bm.wanted[rname] = wanted[i]
	}
}

func (bm *benchmarker) run(events [][]byte) float64 {
	gotMatches := make(map[quamina.X]int)
	before := time.Now()
	for _, event := range events {
		matches, err := bm.q.MatchesForEvent(event)
		if err != nil {
			log.Fatal("m4e: " + err.Error())
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
			log.Printf("for %s wanted %d got %d", match, bm.wanted[match], gotMatches[match])
		}
	}
	for match := range bm.wanted {
		got, ok := gotMatches[match]
		if !ok {
			got = 0
		}
		if bm.wanted[match] != got {
			log.Printf("for %s wanted %d got %d", match, bm.wanted[match], got)
		}
	}
	return eps
}
