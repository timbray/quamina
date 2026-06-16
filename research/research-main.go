// This runs benchmarks against Quamina and, using the GetMatcherStats() API, retrieves data about the size
// and complexity of the matcher, and writes it out in a CSV file suitable for producing graphs.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"quamina.net/go/quamina/v2"
)

const oneMeg = 1024 * 1024

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write a CPU profile to this file")
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			die(err.Error())
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			die(err.Error())
		}
		defer pprof.StopCPUProfile()
	}
	writeNfaMetrics()
}

func writeNfaMetrics() {
	words := readWWords(10000)

	fmt.Printf("WC %d\n", len(words))
	starWords := make([]string, 0, len(words))
	patterns := make([]string, 0, len(words))
	source := rand.NewSource(293591)

	for _, word := range words {
		//nolint:gosec
		starAt := source.Int63() % 6
		starWord := string(word[:starAt]) + "*" + string(word[starAt:])
		starWords = append(starWords, starWord)
		pattern := fmt.Sprintf(`{"x": [ {"shellstyle": "%s" } ] }`, starWord)
		patterns = append(patterns, pattern)
	}

	q, _ := quamina.New()
	before := time.Now()
	b10 := before
	var diffs []float64
	var stateses []float64
	var byteses []float64
	var maxFanouts []float64
	var fanoutsAvg []float64
	var matchesPerSecond []float64

	for i := range words {
		err := q.AddPattern(starWords[i], patterns[i])
		if err != nil {
			die("AddP: " + err.Error())
		}

		if i%100 == 0 {
			bDiff := float64(time.Since(b10).Milliseconds())
			stats := q.GetMatcherStats()
			states := stats["states"]
			bytes := stats["bytes"]
			fanout := stats["fanouts"]
			maxfanout := stats["maxFanout"]
			avgFanout := fanout / states

			// save addPattern stats
			diffs = append(diffs, bDiff)
			byteses = append(byteses, bytes)
			stateses = append(stateses, states)
			fanoutsAvg = append(fanoutsAvg, avgFanout)
			maxFanouts = append(maxFanouts, maxfanout)

			// compute and save average MatchesForEvent time
			beforeMatches := time.Now()
			var perSecond float64
			if i < 100 {
				perSecond = 0.0
			} else {
				for j := i - 100; j < i; j++ {
					record := fmt.Sprintf(`{"x": "%s"}`, words[j])
					matches, err := q.MatchesForEvent([]byte(record))
					if err != nil {
						die(err.Error())
					}
					if len(matches) == 0 {
						die(fmt.Sprintf("0 matches for %s", record))
					}
				}
				perSecond = 100.0 / time.Since(beforeMatches).Seconds()
			}
			matchesPerSecond = append(matchesPerSecond, perSecond)
			b10 = time.Now()
		}
	}

	now := time.Now()
	fn := fmt.Sprintf("research/%d-%02d-%02d.csv", now.Year(), now.Month(), now.Day())
	csv, err := os.Create(fn)
	if err != nil {
		die(err.Error())
	}
	_, _ = fmt.Fprintln(csv, "ms/100 AddP calls,state count,byte count,average fanout,max fanout,matches/sec")
	for i := range diffs {
		_, _ = fmt.Fprintf(csv,
			"%d,%d,%d,%.1f,%d,%.1f\n",
			int(diffs[i]), int(stateses[i]), int(byteses[i]), fanoutsAvg[i], int(maxFanouts[i]), matchesPerSecond[i])
	}
	_ = csv.Close()

	fmt.Println("Done adding patterns")
	elapsed := time.Since(before).Seconds()
	eps := float64(len(words)) / elapsed
	fmt.Printf("Patterns/sec: %.1f\n", eps)
}

func die(why string) {
	fmt.Println(why)
	os.Exit(1)
}

func readWWords(maxWords int) [][]byte {
	// that's a list from the Wordle source code with a few erased to get a prime number
	file, err := os.Open("testdata/wwords.txt")
	if err != nil {
		die("Can't open file: " + err.Error())
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, oneMeg)
	scanner.Buffer(buf, oneMeg)

	var lines [][]byte
	for scanner.Scan() {
		lines = append(lines, []byte(scanner.Text()))
		if maxWords > 0 && len(lines) >= maxWords {
			break
		}
	}
	return lines
}
