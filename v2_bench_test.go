//go:build go1.24

package quamina

import (
	"fmt"
	"testing"
)

// Benchmarks designed to work with Go's 1.24 testing.B.Loop().  Note: When doing this kind of benchmarking, always
// call quamina.MatchesForEvent, as opposed to working directly with the coreMatcher, because the top-level function
// is clever about re-using the nfaBuffers structure.

func Benchmark8259Example(b *testing.B) {
	j := `{
        "Image": {
            "Width":  800,
            "Height": 600,
            "Title":  "View from 15th Floor",
            "Thumbnail": {
                "Url":    "https://www.example.com/image/481989943",
                "Height": 125,
                "Width":  100
            },
            "Animated" : false,
            "IDs": [116, 943, 234, 38793]
          }
      }`
	patternsFromReadme := []string{
		`{"Image": {"Width": [800]}}`,
		`{"Image": { "Animated": [ false], "Thumbnail": { "Height": [ 125 ] } } }}, "IDs": [943]}`,
		`{"Image": { "Title": [ { "exists": true } ] } }`,
		`{"Image": { "Width": [800], "Title": [ { "exists": true } ], "Animated": [ false ] } }`,
		`{"Image": { "Width": [800], "IDs": [ { "exists": true } ] } }`,
		`{"Foo": [ { "exists": false } ] }"`,
		`{"Image": { "Thumbnail": { "Url": [ { "wildcard": "*9943" } ] } } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "wildcard": "https://www.example.*/*9943" } ] } } }`,
		`{"Image": { "Title": [ {"anything-but":  ["Pikachu", "Eevee"] } ]  } }`,
		`{"Image": { "Thumbnail": { "Url": [ "a", { "prefix": "https:" } ] } } }`,
		`{"Image": { "Title": [ { "equals-ignore-case": "VIEW FROM 15th FLOOR" } ] } }`,
		`{"Image": { "Title": [ { "regexp": "View .... [0-9][0-9][rtn][dh] Floor" } ]  } }`,
		`{"Image": { "Title": [ { "regexp": "(View)?( down)? from 15th (Floor|Storey)" } ]  } }`,
		`{"Image": { "Thumbnail": { "Url": [ { "regexp": "https://www.example.com/[^0-9/]+/[1-9]+" } ] } } }`,
		`{"Image": { "Title": [ { "regexp": "[~p{L}~p{Zs}~p{Nd}]*"} ] } }"`,
	}

	var err error
	q, _ := New()
	m := q.matcher.(*coreMatcher)
	for _, should := range patternsFromReadme {
		err = q.AddPattern(should, should)
		if err != nil {
			b.Error("add one of many: " + err.Error())
		}
	}
	fmt.Printf("FA: %s\n", matcherStats(m))
	bytes := []byte(j)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		matches, _ := q.MatchesForEvent(bytes)
		if len(matches) == 0 {
			b.Errorf("No matches")
		}
	}
	elapsed := float64(b.Elapsed().Seconds())
	count := float64(b.N)
	fmt.Printf("%.0f/sec\n", count/elapsed)
}
