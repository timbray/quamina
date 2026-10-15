package quamina

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// Benchmarks using real alert areas: the utility service territories, threat buffers, fire districts, and
// high-fire-threat zones in weathernode-camera-system/layers/geodata/geojson. Subscriptions follow the
// legacy fancy-alerts model: a polygon covered by every precision-5 geohash it intersects, with complete
// parents compressed (polygon_to_geohashes(polygon, 5, False) and georaptor.compress(..., 1, 5)); weekly
// day/hour sets with inclusive end hours, in the area's time zone; and an optional minimum risk. Each
// subscription either takes one area, so that many share it, or the union of three; see realSelection.
//
// Set QUAMINA_GEOJSON_DIR to use a different directory. Run with e.g.
//   go test -run '^$' -bench 'RealAlerts' -benchtime 1x

const realCellDegrees = 360.0 / 8192 // precision-5 geohash cells are 2^13 x 2^12 over 360 x 180 degrees

type realArea struct {
	name      string
	tz        string
	loc       *time.Location
	cells     []string   // compressed, sorted
	raw       [][2]int32 // uncompressed precision-5 cells, for placing events
	rawHashes []string   // the geohashes of raw
}

type realWindow struct {
	days       []int // 0 = Monday
	start, end int   // hours, end inclusive, as in the legacy datesTimes
}

type realSub struct {
	id      string
	area    *realArea
	windows []realWindow // nil means at all times
	minRisk int          // -1 means no risk filter
}

type realEvent struct {
	json []byte
	gh   string
	ts   int64
	risk int
}

var (
	realAreasOnce sync.Once
	realAreas     []*realArea
	realAreasErr  error
)

func loadRealAreas(tb testing.TB) []*realArea {
	realAreasOnce.Do(func() {
		dir := os.Getenv("QUAMINA_GEOJSON_DIR")
		if dir == "" {
			dir = "../weathernode-camera-system/layers/geodata/geojson"
		}
		realAreas, realAreasErr = readRealAreas(dir)
	})
	if realAreasErr != nil {
		tb.Skip("real alert areas unavailable: ", realAreasErr)
	}
	return realAreas
}

func readRealAreas(dir string) ([]*realArea, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.geojson"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no geojson in %s", dir)
	}
	slices.Sort(files)
	var areas []*realArea
	for _, file := range files {
		base := strings.TrimSuffix(filepath.Base(file), ".geojson")
		if strings.HasSuffix(base, "-simplified") && slices.Contains(files, strings.Replace(file, "-simplified", "", 1)) {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var fc struct {
			Features []struct {
				Geometry struct {
					Type        string          `json:"type"`
					Coordinates json.RawMessage `json:"coordinates"`
				} `json:"geometry"`
			} `json:"features"`
		}
		if err := json.Unmarshal(data, &fc); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		whole := map[[2]int32]bool{}
		var parts []map[[2]int32]bool
		for _, f := range fc.Features {
			var polygons [][][][]float64
			switch f.Geometry.Type {
			case "Polygon":
				var p [][][]float64
				if err := json.Unmarshal(f.Geometry.Coordinates, &p); err != nil {
					return nil, err
				}
				polygons = [][][][]float64{p}
			case "MultiPolygon":
				if err := json.Unmarshal(f.Geometry.Coordinates, &polygons); err != nil {
					return nil, err
				}
			default:
				continue
			}
			cells := map[[2]int32]bool{}
			for _, p := range polygons {
				coverPolygon(p, cells)
			}
			for c := range cells {
				whole[c] = true
			}
			parts = append(parts, cells)
		}
		if len(whole) == 0 {
			continue
		}
		areas = append(areas, newRealArea(base, realTimeZone(base, whole), whole))
		if len(parts) > 1 {
			for i, cells := range parts {
				if i >= 100 {
					break
				}
				if len(cells) > 0 {
					areas = append(areas, newRealArea(fmt.Sprintf("%s#%d", base, i), realTimeZone(base, cells), cells))
				}
			}
		}
	}
	return areas, nil
}

// coverPolygon adds every precision-5 cell that the polygon, given as GeoJSON rings, intersects: the cells
// along its edges and the cells whose centers are inside it.
func coverPolygon(rings [][][]float64, cells map[[2]int32]bool) {
	cellOf := func(lon, lat float64) [2]int32 {
		return [2]int32{int32(math.Floor((lon + 180) / realCellDegrees)), int32(math.Floor((lat + 90) / realCellDegrees))}
	}
	for _, ring := range rings {
		minX, maxX := math.Inf(1), math.Inf(-1)
		for _, p := range ring {
			minX, maxX = min(minX, p[0]), max(maxX, p[0])
		}
		if maxX-minX > 180 {
			// crosses the antimeridian, which planar scanning can't handle
			return
		}
	}
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, ring := range rings {
		for i := 0; i+1 < len(ring); i++ {
			x1, y1, x2, y2 := ring[i][0], ring[i][1], ring[i+1][0], ring[i+1][1]
			minY, maxY = min(minY, y1, y2), max(maxY, y1, y2)
			steps := int(math.Ceil(max(math.Abs(x2-x1), math.Abs(y2-y1))/realCellDegrees*4)) + 1
			for s := 0; s <= steps; s++ {
				t := float64(s) / float64(steps)
				cells[cellOf(x1+t*(x2-x1), y1+t*(y2-y1))] = true
			}
		}
	}
	if math.IsInf(minY, 0) {
		return
	}
	var xs []float64
	for row := int32(math.Floor((minY + 90) / realCellDegrees)); row <= int32(math.Floor((maxY+90)/realCellDegrees)); row++ {
		yc := (float64(row)+0.5)*realCellDegrees - 90
		xs = xs[:0]
		for _, ring := range rings {
			for i := 0; i+1 < len(ring); i++ {
				x1, y1, x2, y2 := ring[i][0], ring[i][1], ring[i+1][0], ring[i+1][1]
				if (y1 > yc) != (y2 > yc) {
					xs = append(xs, x1+(yc-y1)/(y2-y1)*(x2-x1))
				}
			}
		}
		slices.Sort(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			first := int32(math.Ceil((xs[i]+180)/realCellDegrees - 0.5))
			last := int32(math.Floor((xs[i+1]+180)/realCellDegrees - 0.5))
			for col := first; col <= last; col++ {
				cells[[2]int32{col, row}] = true
			}
		}
	}
}

func realCellCenter(c [2]int32) (lat, lon float64) {
	return (float64(c[1])+0.5)*realCellDegrees - 90, (float64(c[0])+0.5)*realCellDegrees - 180
}

func newRealArea(name, tz string, cells map[[2]int32]bool) *realArea {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		panic(err)
	}
	area := &realArea{name: name, tz: tz, loc: loc}
	for c := range cells {
		area.raw = append(area.raw, c)
	}
	slices.SortFunc(area.raw, func(a, b [2]int32) int {
		if a[0] != b[0] {
			return int(a[0] - b[0])
		}
		return int(a[1] - b[1])
	})
	hashes := make(map[string]bool, len(cells))
	for _, c := range area.raw {
		h := geohashEncodeP5(realCellCenter(c))
		area.rawHashes = append(area.rawHashes, h)
		hashes[h] = true
	}
	area.cells = compressGeohashes(hashes)
	return area
}

// unionArea returns the area covering all of the given areas, in the first one's time zone.
func unionArea(name string, parts []*realArea) *realArea {
	hashes := make(map[string]bool)
	for _, p := range parts {
		for _, h := range p.rawHashes {
			hashes[h] = true
		}
	}
	return &realArea{name: name, tz: parts[0].tz, loc: parts[0].loc, cells: compressGeohashes(hashes)}
}

func (a *realArea) contains(gh string) bool {
	for i := 1; i <= 5; i++ {
		if _, found := slices.BinarySearch(a.cells, gh[:i]); found {
			return true
		}
	}
	return false
}

func geohashEncodeP5(lat, lon float64) string { return geohashEncode(lat, lon, 5) }

// realTimeZone picks the area's time zone from its location, as a user in that area would.
func realTimeZone(name string, cells map[[2]int32]bool) string {
	var sumLat, sumLon float64
	for c := range cells {
		lat, lon := realCellCenter(c)
		sumLat += lat
		sumLon += lon
	}
	lat, lon := sumLat/float64(len(cells)), sumLon/float64(len(cells))
	switch {
	case strings.HasPrefix(name, "idp-"):
		return "America/Boise"
	case lon < -140:
		return "Pacific/Honolulu"
	case lon > -90:
		return "America/New_York"
	case lat > 49:
		return "America/Edmonton"
	case lon > -114:
		return "America/Denver"
	default:
		return "America/Los_Angeles"
	}
}

var realSchedules = []struct {
	weight  int
	windows []realWindow
}{
	{50, nil}, // at all times
	{25, []realWindow{{[]int{0, 1, 2, 3, 4}, 8, 17}}},
	{10, []realWindow{{[]int{0, 1, 2, 3, 4, 5, 6}, 6, 21}}},
	{10, []realWindow{{[]int{0, 1, 2, 3, 4}, 0, 7}, {[]int{0, 1, 2, 3, 4}, 18, 23}, {[]int{5, 6}, 0, 23}}},
}

// realSelection is how subscriptions choose their areas.
type realSelection string

const (
	// selectShared gives each subscription one area, weighting org-wide areas 10x, so many share each area
	selectShared realSelection = "one-shared"
	// selectThree gives each subscription the union of 3 distinct areas chosen uniformly, so nearly every
	// subscription's geometry is unique
	selectThree realSelection = "union-of-3"
)

func generateRealSubs(areas []*realArea, n int, seed uint64, selection realSelection) []realSub {
	r := rand.New(rand.NewPCG(seed, seed))
	areaRand := rand.New(rand.NewPCG(seed, seed^0x3a))
	weights := make([]int, len(areas))
	total := 0
	for i, a := range areas {
		weights[i] = 1
		if !strings.Contains(a.name, "#") {
			weights[i] = 10
		}
		total += weights[i]
	}
	subs := make([]realSub, n)
	for i := range subs {
		pick := r.IntN(total)
		var area *realArea
		for j, w := range weights {
			if pick < w {
				area = areas[j]
				break
			}
			pick -= w
		}
		if selection == selectThree {
			picks := map[int]bool{}
			var parts []*realArea
			for len(parts) < 3 {
				if j := areaRand.IntN(len(areas)); !picks[j] {
					picks[j] = true
					parts = append(parts, areas[j])
				}
			}
			area = unionArea(fmt.Sprintf("union-%d", i), parts)
		}
		sub := realSub{id: fmt.Sprintf("sub-%d", i), area: area, minRisk: -1}
		switch s := r.IntN(100); {
		case s < 95:
			for _, sched := range realSchedules {
				if s < sched.weight {
					sub.windows = sched.windows
					break
				}
				s -= sched.weight
			}
		default:
			for k := 1 + r.IntN(3); k > 0; k-- {
				var days []int
				for d := 0; d < 7; d++ {
					if r.IntN(2) == 0 {
						days = append(days, d)
					}
				}
				if len(days) == 0 {
					days = []int{r.IntN(7)}
				}
				start := r.IntN(24)
				sub.windows = append(sub.windows, realWindow{days, start, start + r.IntN(24-start)})
			}
		}
		switch x := r.IntN(100); {
		case x < 35:
			sub.minRisk = 1
		case x < 50:
			sub.minRisk = 4
		}
		subs[i] = sub
	}
	return subs
}

func (s realSub) pattern() Pattern {
	incident := map[string]any{"geohash": s.area.cells}
	if s.windows != nil {
		var windows []any
		for _, w := range s.windows {
			var days []string
			for _, d := range w.days {
				days = append(days, dayNames[d])
			}
			windows = append(windows, map[string]any{"time-window": map[string]any{
				"tz":    s.area.tz,
				"days":  days,
				"start": fmt.Sprintf("%02d:00", w.start),
				"end":   fmt.Sprintf("%02d:00", w.end+1),
			}})
		}
		incident["timestamp"] = windows
	}
	if s.minRisk >= 0 {
		incident["risk"] = []any{map[string]any{"numeric": []any{">=", s.minRisk}}}
	}
	j, _ := json.Marshal(map[string]any{"incident": incident})
	return Pattern{X: s.id, JSON: string(j)}
}

// matches is the reference implementation of the legacy semantics.
func (s realSub) matches(e realEvent) bool {
	if !s.area.contains(e.gh) || (s.minRisk >= 0 && e.risk < s.minRisk) {
		return false
	}
	if s.windows == nil {
		return true
	}
	local := time.UnixMilli(e.ts).In(s.area.loc)
	day, hour := (int(local.Weekday())+6)%7, local.Hour()
	for _, w := range s.windows {
		if slices.Contains(w.days, day) && hour >= w.start && hour <= w.end {
			return true
		}
	}
	return false
}

func generateRealEvents(areas []*realArea, n int, seed uint64) []realEvent {
	r := rand.New(rand.NewPCG(seed, seed+1))
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC).UnixMilli() // spans the November DST change
	span := int64(14 * 24 * time.Hour / time.Millisecond)
	riskWeights := []int{30, 25, 15, 12, 10, 8}
	events := make([]realEvent, n)
	for i := range events {
		var lat, lon float64
		if r.IntN(100) < 85 {
			area := areas[r.IntN(len(areas))]
			lat, lon = realCellCenter(area.raw[r.IntN(len(area.raw))])
			lat += (r.Float64() - 0.5) * realCellDegrees
			lon += (r.Float64() - 0.5) * realCellDegrees
		} else {
			lat, lon = 31+r.Float64()*18, -125+r.Float64()*23
		}
		risk, pick := 0, r.IntN(100)
		for risk = range riskWeights {
			if pick < riskWeights[risk] {
				break
			}
			pick -= riskWeights[risk]
		}
		gh := geohashEncode(lat, lon, 8)
		prefixes := make([]string, len(gh))
		for p := range prefixes {
			prefixes[p] = gh[:p+1]
		}
		e := realEvent{gh: gh, ts: start + r.Int64N(span), risk: risk}
		e.json, _ = json.Marshal(map[string]any{
			"type":   "incident.detected",
			"source": "cameras-rt-stream-handler",
			"incident": map[string]any{
				"id":         fmt.Sprintf("inc-%08d", i),
				"camera_id":  fmt.Sprintf("cam-%04d", r.IntN(3000)),
				"org_id":     fmt.Sprintf("org-%03d", r.IntN(200)),
				"latitude":   lat,
				"longitude":  lon,
				"geohash":    prefixes,
				"timestamp":  e.ts,
				"risk":       risk,
				"confidence": math.Round(r.Float64()*1000) / 1000,
				"status":     "new",
				"image_url":  fmt.Sprintf("https://images.example.com/%08d.jpg", i),
			},
		})
		events[i] = e
	}
	return events
}

func realPatterns(subs []realSub) []Pattern {
	patterns := make([]Pattern, len(subs))
	for i, s := range subs {
		patterns[i] = s.pattern()
	}
	return patterns
}

func realReference(subs []realSub, e realEvent) []string {
	out := []string{}
	for _, s := range subs {
		if s.matches(e) {
			out = append(out, s.id)
		}
	}
	slices.Sort(out)
	return out
}

// requireReferenceMatches fails unless q matches each event exactly as the reference implementation does,
// and returns the mean number of matches per event.
func requireReferenceMatches(tb testing.TB, q *Quamina, subs []realSub, events []realEvent) float64 {
	total := 0
	for _, e := range events {
		want, got := realReference(subs, e), sortedMatches(tb, q, e.json)
		if !slices.Equal(want, got) {
			tb.Fatalf("matches differ for %s:\nreference: %v\nquamina:   %v", e.json, want, got)
		}
		total += len(want)
	}
	return float64(total) / float64(len(events))
}

func BenchmarkRealAlertsBuild(b *testing.B) {
	areas := loadRealAreas(b)
	events := generateRealEvents(areas, 2000, 77)
	for _, selection := range []realSelection{selectShared, selectThree} {
		for _, n := range []int{100, 1000, 5000, 10000, 25000} {
			subs := generateRealSubs(areas, n, 42, selection)
			patterns := realPatterns(subs)
			cells := 0
			for _, s := range subs {
				cells += len(s.area.cells)
			}
			if n <= 100 {
				b.Run(fmt.Sprintf("%s/n=%d/AddPattern", selection, n), func(b *testing.B) {
					q := reportBuild(b, buildSequential, patterns)
					b.ReportMetric(requireReferenceMatches(b, q, subs, events), "matches/event")
				})
			}
			b.Run(fmt.Sprintf("%s/n=%d/AddPatterns", selection, n), func(b *testing.B) {
				q := reportBuild(b, buildCompact, patterns)
				b.ReportMetric(float64(cells)/float64(n), "geohashes/sub")
				b.ReportMetric(requireReferenceMatches(b, q, subs, events), "matches/event")
			})
		}
	}
}

func BenchmarkRealAlertsMatch(b *testing.B) {
	areas := loadRealAreas(b)
	events := generateRealEvents(areas, 10000, 123)
	for _, sn := range []struct {
		selection realSelection
		n         int
	}{
		{selectShared, 1000}, {selectShared, 5000}, {selectShared, 10000}, {selectShared, 25000},
		{selectThree, 1000}, {selectThree, 5000}, {selectThree, 10000}, {selectThree, 25000},
	} {
		n := sn.n
		subs := generateRealSubs(areas, n, 42, sn.selection)
		q := buildCompact(b, realPatterns(subs))
		perEvent := requireReferenceMatches(b, q, subs, events[:2000])
		prefix := string(sn.selection) + "/"
		b.Run(fmt.Sprintf("%sn=%d/quamina", prefix, n), func(b *testing.B) {
			b.ReportMetric(perEvent, "matches/event")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := q.MatchesForEvent(events[i%len(events)].json); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/s")
		})
		b.Run(fmt.Sprintf("%sn=%d/quamina-parallel", prefix, n), func(b *testing.B) {
			b.ReportMetric(float64(runtime.GOMAXPROCS(0)), "goroutines")
			b.ResetTimer()
			var next sync.Mutex
			i := 0
			b.RunParallel(func(pb *testing.PB) {
				local := q.Copy()
				next.Lock()
				j := i * 7919
				i++
				next.Unlock()
				for pb.Next() {
					if _, err := local.MatchesForEvent(events[j%len(events)].json); err != nil {
						b.Error(err)
						return
					}
					j++
				}
			})
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/s")
		})
		b.Run(fmt.Sprintf("%sn=%d/naive-loop", prefix, n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				e := events[i%len(events)]
				var parsed struct {
					Incident struct {
						Geohash   []string `json:"geohash"`
						Timestamp int64    `json:"timestamp"`
						Risk      int      `json:"risk"`
					} `json:"incident"`
				}
				if err := json.Unmarshal(e.json, &parsed); err != nil {
					b.Fatal(err)
				}
				ev := realEvent{gh: parsed.Incident.Geohash[len(parsed.Incident.Geohash)-1], ts: parsed.Incident.Timestamp, risk: parsed.Incident.Risk}
				for _, s := range subs {
					s.matches(ev)
				}
			}
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/s")
		})
	}
}

// BenchmarkRealAlertsAreas reports the area catalog the other benchmarks use.
func BenchmarkRealAlertsAreas(b *testing.B) {
	areas := loadRealAreas(b)
	whole, parts, cells := 0, 0, 0
	tzs := map[string]int{}
	for _, a := range areas {
		if strings.Contains(a.name, "#") {
			parts++
		} else {
			whole++
			b.Logf("%-40s %-20s %6d cells (%6d before compression)", a.name, a.tz, len(a.cells), len(a.raw))
		}
		cells += len(a.cells)
		tzs[a.tz]++
	}
	b.Logf("%d org-wide areas, %d sub-areas, %d geohashes in total, time zones %v", whole, parts, cells, tzs)
	b.ReportMetric(float64(len(areas)), "areas")
}
