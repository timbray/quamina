package quamina

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// Benchmarks for AddPatterns, using a workload shaped like geofenced alert subscriptions: each subscription
// covers a circle of 10-60km somewhere in California with precision-5 geohash cells (compressed where all 32
// children of a cell are present), plus weekly hour windows, either as a list of hours of the week or as
// time-window values in the subscriber's time zone, and a minimum risk level.
//
// Run with e.g.
//   go test -run '^$' -bench 'CompactBuild|CompactMatch|CompactIncremental|CompactMixed' -benchtime 1x

const geohashBase32 = "0123456789bcdefghjkmnpqrstuvwxyz"

func geohashEncode(lat, lon float64, precision int) string {
	latMin, latMax, lonMin, lonMax := -90.0, 90.0, -180.0, 180.0
	var sb strings.Builder
	bit, idx, even := 0, 0, true
	for sb.Len() < precision {
		if even {
			mid := (lonMin + lonMax) / 2
			if lon >= mid {
				idx = idx*2 + 1
				lonMin = mid
			} else {
				idx *= 2
				lonMax = mid
			}
		} else {
			mid := (latMin + latMax) / 2
			if lat >= mid {
				idx = idx*2 + 1
				latMin = mid
			} else {
				idx *= 2
				latMax = mid
			}
		}
		even = !even
		bit++
		if bit == 5 {
			sb.WriteByte(geohashBase32[idx])
			bit, idx = 0, 0
		}
	}
	return sb.String()
}

type subscription struct {
	cells   []string
	hours   []int
	risks   []int
	tz      string
	windows []dayWindow
}

type dayWindow struct{ day, startHour, endHour int }

var dayNames = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

func randomCALocation(r *rand.Rand) (float64, float64) {
	return 32.5 + r.Float64()*9.5, -124.4 + r.Float64()*10.3
}

func generateSubscriptions(n int, seed uint64) []subscription {
	r := rand.New(rand.NewPCG(seed, seed))
	tzRand := rand.New(rand.NewPCG(seed, seed^0x7a))
	subs := make([]subscription, n)
	for i := range subs {
		subs[i].tz = []string{"America/Los_Angeles", "America/Los_Angeles", "America/Los_Angeles", "America/Denver", "America/Phoenix", "UTC"}[tzRand.IntN(6)]
		lat, lon := randomCALocation(r)
		radiusKm := 10 + r.Float64()*50
		dLat := radiusKm / 111.0
		dLon := radiusKm / (111.0 * math.Cos(lat*math.Pi/180))
		cellSet := make(map[string]bool)
		// precision-5 cells are ~4.9km x 4.9km; sample at ~1.2km so every overlapped cell is found
		for y := lat - dLat; y <= lat+dLat; y += 0.011 {
			for x := lon - dLon; x <= lon+dLon; x += 0.011 {
				ny, nx := (y-lat)/dLat, (x-lon)/dLon
				if ny*ny+nx*nx <= 1 {
					cellSet[geohashEncode(y, x, 5)] = true
				}
			}
		}
		subs[i].cells = compressGeohashes(cellSet)

		hourSet := make(map[int]bool)
		for _, day := range r.Perm(7)[:5] {
			start := 5 + r.IntN(8)
			h := start
			for ; h < start+6+r.IntN(5); h++ {
				hourSet[day*24+h] = true
			}
			subs[i].windows = append(subs[i].windows, dayWindow{day: day, startHour: start, endHour: h})
		}
		for h := range hourSet {
			subs[i].hours = append(subs[i].hours, h)
		}
		slices.Sort(subs[i].hours)

		for risk := 1 + r.IntN(4); risk <= 6; risk++ {
			subs[i].risks = append(subs[i].risks, risk)
		}
	}
	return subs
}

func compressGeohashes(cells map[string]bool) []string {
	for precision := 5; precision > 1; precision-- {
		children := make(map[string]int)
		for c := range cells {
			if len(c) == precision {
				children[c[:precision-1]]++
			}
		}
		for parent, count := range children {
			if count == 32 {
				for _, ch := range geohashBase32 {
					delete(cells, parent+string(ch))
				}
				cells[parent] = true
			}
		}
	}
	out := make([]string, 0, len(cells))
	for c := range cells {
		out = append(out, c)
	}
	slices.Sort(out)
	return out
}

type patternShape string

const (
	shapeGeohash patternShape = "geohash"           // {"geohash": [cells]}
	shapeCross   patternShape = "geohash×hour×risk" // the shape used by the existing notifications rule evaluator
	shapeTime    patternShape = "geohash×time-window×risk"
	shapeNumeric patternShape = "geohash×time-window×numeric" // risk as {"numeric": [">=", minimum]}
)

func subscriptionPatterns(subs []subscription, shape patternShape) []Pattern {
	patterns := make([]Pattern, len(subs))
	for i, s := range subs {
		pattern := map[string]any{"geohash": s.cells}
		switch shape {
		case shapeCross:
			pattern["hour"] = s.hours
			pattern["risk"] = s.risks
		case shapeTime, shapeNumeric:
			var windows []any
			for _, w := range s.windows {
				windows = append(windows, map[string]any{"time-window": map[string]any{
					"tz":    s.tz,
					"days":  []string{dayNames[w.day]},
					"start": fmt.Sprintf("%02d:00", w.startHour),
					"end":   fmt.Sprintf("%02d:00", w.endHour),
				}})
			}
			pattern["timestamp"] = windows
			pattern["risk"] = s.risks
			if shape == shapeNumeric {
				pattern["risk"] = []any{map[string]any{"numeric": []any{">=", s.risks[0]}}}
			}
		}
		j, _ := json.Marshal(pattern)
		patterns[i] = Pattern{X: fmt.Sprintf("sub-%d", i), JSON: string(j)}
	}
	return patterns
}

func generateEvents(n int, seed uint64) [][]byte {
	r := rand.New(rand.NewPCG(seed, seed+1))
	tsRand := rand.New(rand.NewPCG(seed, seed+2))
	weekStart := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	events := make([][]byte, n)
	for i := range events {
		lat, lon := randomCALocation(r)
		gh := geohashEncode(lat, lon, 8)
		prefixes := make([]string, len(gh))
		for p := range prefixes {
			prefixes[p] = gh[:p+1]
		}
		ts := weekStart.Add(time.Duration(tsRand.Int64N(int64(14 * 24 * time.Hour)))).UnixMilli()
		j, _ := json.Marshal(map[string]any{"geohash": prefixes, "hour": r.IntN(168), "risk": 1 + r.IntN(6), "timestamp": ts})
		events[i] = j
	}
	return events
}

func buildSequential(tb testing.TB, patterns []Pattern) *Quamina {
	q, _ := New()
	for _, p := range patterns {
		if err := q.AddPattern(p.X, p.JSON); err != nil {
			tb.Fatal(err)
		}
	}
	return q
}

func buildCompact(tb testing.TB, patterns []Pattern) *Quamina {
	q, _ := New()
	if err := q.AddPatterns(patterns); err != nil {
		tb.Fatal(err)
	}
	return q
}

func sortedMatches(tb testing.TB, q *Quamina, event []byte) []string {
	matches, err := q.MatchesForEvent(event)
	if err != nil {
		tb.Fatal(err)
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, fmt.Sprint(m))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// requireSameMatches fails unless both matchers give the same results for every event, and returns how many
// events matched at least one Pattern so that a vacuous comparison is visible.
func requireSameMatches(tb testing.TB, want, got *Quamina, events [][]byte) int {
	nonEmpty := 0
	for _, e := range events {
		w, g := sortedMatches(tb, want, e), sortedMatches(tb, got, e)
		if !slices.Equal(w, g) {
			tb.Fatalf("matches differ for %s:\nAddPattern:  %v\nAddPatterns: %v", e, w, g)
		}
		if len(w) > 0 {
			nonEmpty++
		}
	}
	return nonEmpty
}

func heapInUse() uint64 {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapInuse
}

func reportBuild(b *testing.B, build func(testing.TB, []Pattern) *Quamina, patterns []Pattern) *Quamina {
	b.Helper()
	var q *Quamina
	var heap uint64
	b.StopTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q = nil
		before := heapInUse()
		b.StartTimer()
		q = build(b, patterns)
		b.StopTimer()
		heap = heapInUse() - before
	}
	stats := q.GetMatcherStats()
	b.ReportMetric(stats["states"], "faStates")
	b.ReportMetric(float64(heap)/(1<<20), "heap-MB")
	b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/build")
	return q
}

// sequentialLimit caps the sizes at which AddPattern is benchmarked: for the cross-product shape it allocates
// roughly cells*hours*risks fieldMatchers per subscription and runs out of memory quickly.
var sequentialLimit = map[patternShape]int{shapeGeohash: 5000, shapeCross: 100, shapeTime: 1000, shapeNumeric: 1000}

func BenchmarkCompactBuild(b *testing.B) {
	events := generateEvents(2000, 99)
	for _, shape := range []patternShape{shapeGeohash, shapeCross, shapeTime, shapeNumeric} {
		for _, n := range []int{10, 100, 1000, 5000} {
			patterns := subscriptionPatterns(generateSubscriptions(n, 7), shape)
			var reference *Quamina
			if n <= sequentialLimit[shape] {
				b.Run(fmt.Sprintf("%s/n=%d/AddPattern", shape, n), func(b *testing.B) {
					reference = reportBuild(b, buildSequential, patterns)
				})
			}
			b.Run(fmt.Sprintf("%s/n=%d/AddPatterns", shape, n), func(b *testing.B) {
				q := reportBuild(b, buildCompact, patterns)
				if reference != nil {
					b.ReportMetric(float64(requireSameMatches(b, reference, q, events)), "events-matched")
				}
			})
		}
	}
}

func BenchmarkCompactMatch(b *testing.B) {
	events := generateEvents(10000, 123)
	for _, shape := range []patternShape{shapeGeohash, shapeCross, shapeTime, shapeNumeric} {
		n := 1000
		if shape == shapeCross {
			n = 100
		}
		patterns := subscriptionPatterns(generateSubscriptions(n, 7), shape)
		sequential := buildSequential(b, patterns)
		compact := buildCompact(b, patterns)
		requireSameMatches(b, sequential, compact, events)
		for _, c := range []struct {
			name string
			q    *Quamina
		}{{"AddPattern", sequential}, {"AddPatterns", compact}} {
			b.Run(fmt.Sprintf("%s/n=%d/%s", shape, n, c.name), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := c.q.MatchesForEvent(events[i%len(events)]); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkCompactIncremental measures adding single Patterns to an instance built with AddPatterns, which
// goes through the compiler, and checks the result against an instance built by AddPattern alone.
func BenchmarkCompactIncremental(b *testing.B) {
	events := generateEvents(2000, 5)
	for _, n := range []int{1000, 5000} {
		patterns := subscriptionPatterns(generateSubscriptions(n+100, 11), shapeGeohash)
		b.Run(fmt.Sprintf("%s/n=%d", shapeGeohash, n), func(b *testing.B) {
			var q *Quamina
			var elapsed time.Duration
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				q = buildCompact(b, patterns[:n])
				start := time.Now()
				for _, p := range patterns[n:] {
					if err := q.AddPattern(p.X, p.JSON); err != nil {
						b.Fatal(err)
					}
				}
				elapsed += time.Since(start)
			}
			b.ReportMetric(float64(elapsed.Microseconds())/float64(b.N*100)/1000, "ms/AddPattern")
			requireSameMatches(b, buildSequential(b, patterns), q, events)
		})
	}
}

// mixedWorkload generates Patterns using every value type on several fields, some nested, and events that
// exercise them.
func mixedWorkload(nPatterns, nEvents int) ([]Pattern, [][]byte) {
	r := rand.New(rand.NewPCG(3, 4))
	words := []string{"alpha", "beta", "gamma", "delta", "Epsilon", "zeta", "eta", "theta", "IOTA", "kappa"}
	word := func() string { return words[r.IntN(len(words))] }
	valueFor := func() string {
		switch r.IntN(13) {
		case 12:
			lo := r.IntN(20) - 2
			switch r.IntN(3) {
			case 0:
				return fmt.Sprintf(`{"numeric": [">=", %d]}`, lo)
			case 1:
				return fmt.Sprintf(`{"numeric": ["<", %d.5]}`, lo)
			default:
				return fmt.Sprintf(`{"numeric": [">", %d, "<=", %d]}`, lo, lo+1+r.IntN(5))
			}
		case 11:
			tz := []string{"America/Los_Angeles", "America/Denver", "UTC"}[r.IntN(3)]
			return fmt.Sprintf(`{"time-window": {"tz": "%s", "days": ["%s"], "start": "%02d:%02d", "end": "%02d:00"}}`,
				tz, []string{"mon", "wed", "sat", "sun"}[r.IntN(4)], r.IntN(24), 15*(1+r.IntN(3)), r.IntN(25))
		case 0:
			return fmt.Sprintf(`"%s"`, word())
		case 1:
			return fmt.Sprintf(`%d`, r.IntN(20))
		case 2:
			return fmt.Sprintf(`{"prefix": "%s"}`, word()[:1+r.IntN(3)])
		case 3:
			return fmt.Sprintf(`{"anything-but": ["%s", "%s"]}`, word(), word())
		case 4:
			return fmt.Sprintf(`{"wildcard": "*%s*"}`, word()[1:3])
		case 5:
			return fmt.Sprintf(`{"shellstyle": "%s*"}`, word()[:2])
		case 6:
			return fmt.Sprintf(`{"equals-ignore-case": "%s"}`, strings.ToUpper(word()))
		case 7:
			return fmt.Sprintf(`{"regexp": "%s[a-z]*"}`, word()[:2])
		case 8:
			return `true`
		case 9:
			return `null`
		default:
			return fmt.Sprintf(`%d.5`, r.IntN(5))
		}
	}
	fieldNames := []string{"a", "b", "c.d", "c.e"}
	var patterns []Pattern
	for i := 0; i < nPatterns; i++ {
		obj := map[string]map[string]string{}
		var sb strings.Builder
		sb.WriteString("{")
		first := true
		for _, f := range fieldNames {
			if r.IntN(2) == 0 {
				continue
			}
			var vals string
			switch r.IntN(8) {
			case 0:
				vals = `[{"exists": true}]`
			case 1:
				vals = `[{"exists": false}]`
			default:
				parts := make([]string, 1+r.IntN(3))
				for p := range parts {
					parts[p] = valueFor()
					// anything-but and regexp must be the only value for their field
					if strings.Contains(parts[p], "anything-but") || strings.Contains(parts[p], "regexp") {
						parts = parts[p : p+1]
						break
					}
				}
				vals = "[" + strings.Join(parts, ",") + "]"
			}
			outer, inner, nested := strings.Cut(f, ".")
			if nested {
				if obj[outer] == nil {
					obj[outer] = map[string]string{}
				}
				obj[outer][inner] = vals
				continue
			}
			if !first {
				sb.WriteString(",")
			}
			first = false
			fmt.Fprintf(&sb, `"%s": %s`, outer, vals)
		}
		for _, outer := range sortedKeys(obj) {
			if !first {
				sb.WriteString(",")
			}
			first = false
			fmt.Fprintf(&sb, `"%s": {`, outer)
			innerFirst := true
			for _, inner := range sortedKeys(obj[outer]) {
				if !innerFirst {
					sb.WriteString(",")
				}
				innerFirst = false
				fmt.Fprintf(&sb, `"%s": %s`, inner, obj[outer][inner])
			}
			sb.WriteString("}")
		}
		sb.WriteString("}")
		if first {
			continue
		}
		patterns = append(patterns, Pattern{X: fmt.Sprintf("p%d", i), JSON: sb.String()})
	}

	weekStart := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC) // a Monday, just before US DST ends
	eventValue := func() string {
		switch r.IntN(6) {
		case 5:
			ts := weekStart.Add(time.Duration(r.Int64N(int64(14 * 24 * time.Hour))))
			if r.IntN(2) == 0 {
				return fmt.Sprintf(`%d`, ts.UnixMilli())
			}
			return fmt.Sprintf(`"%s"`, ts.Format(time.RFC3339))
		case 0:
			return fmt.Sprintf(`%d`, r.IntN(20))
		case 1:
			return fmt.Sprintf(`%d.5`, r.IntN(5))
		case 2:
			return []string{"true", "false", "null"}[r.IntN(3)]
		default:
			w := word()
			if r.IntN(2) == 0 {
				w = strings.ToLower(w)
			}
			return fmt.Sprintf(`"%s%s"`, w, []string{"", "x", "abc"}[r.IntN(3)])
		}
	}
	events := make([][]byte, nEvents)
	for i := range events {
		var parts []string
		for _, f := range []string{"a", "b"} {
			if r.IntN(4) > 0 {
				parts = append(parts, fmt.Sprintf(`"%s": %s`, f, eventValue()))
			}
		}
		var inner []string
		for _, f := range []string{"d", "e"} {
			if r.IntN(4) > 0 {
				inner = append(inner, fmt.Sprintf(`"%s": %s`, f, eventValue()))
			}
		}
		if len(inner) > 0 {
			parts = append(parts, `"c": {`+strings.Join(inner, ",")+`}`)
		}
		events[i] = []byte("{" + strings.Join(parts, ",") + "}")
	}
	return patterns, events
}

// BenchmarkCompactMixed builds the mixed workload, checking the result against AddPattern, in both build modes.
func BenchmarkCompactMixed(b *testing.B) {
	patterns, events := mixedWorkload(2000, 5000)
	for _, mode := range []MatcherBuildMode{BuiltForComfort, BuiltForSpeed} {
		name := map[MatcherBuildMode]string{BuiltForComfort: "BuiltForComfort", BuiltForSpeed: "BuiltForSpeed"}[mode]
		build := func(add func(*Quamina) error) func(testing.TB, []Pattern) *Quamina {
			return func(tb testing.TB, _ []Pattern) *Quamina {
				q, _ := New()
				_ = q.SetMatcherBuildMode(mode)
				if err := add(q); err != nil {
					tb.Fatal(err)
				}
				return q
			}
		}
		sequential := build(func(q *Quamina) error {
			for _, p := range patterns {
				if err := q.AddPattern(p.X, p.JSON); err != nil {
					return err
				}
			}
			return nil
		})
		compact := build(func(q *Quamina) error { return q.AddPatterns(patterns) })

		var reference *Quamina
		b.Run(name+"/AddPattern", func(b *testing.B) { reference = reportBuild(b, sequential, patterns) })
		b.Run(name+"/AddPatterns", func(b *testing.B) {
			q := reportBuild(b, compact, patterns)
			b.ReportMetric(float64(requireSameMatches(b, reference, q, events)), "events-matched")
		})
	}
}
