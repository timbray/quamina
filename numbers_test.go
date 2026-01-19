package quamina

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func BenchmarkNumberMatching(b *testing.B) {
	// we’re going to have a pattern that matches one of ten random floats, then we're going to throw
	// 10K random events at it, 10% of which will match the pattern
	rand.New(rand.NewSource(2325))
	pattern := `{"x": [`
	var targets []string
	for i := 0; i < 10; i++ {
		numString := fmt.Sprintf("%.6f", rand.Float64())
		targets = append(targets, numString)
		if i != 0 {
			pattern += ", "
		}
		pattern += numString
	}
	pattern += `]}`
	cm := newCoreMatcher()
	flattener := newJSONFlattener()
	err := cm.addPattern("P", pattern)
	if err != nil {
		b.Error("addP")
	}
	b.ResetTimer()
	b.ReportAllocs()
	targetInd := 0
	calls := 0
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			val := targets[targetInd]
			event := `{"x":` + val + "}"
			matches, err := cm.matchesForJSONWithFlattener([]byte(event), flattener)
			calls++
			if err != nil {
				b.Error("match target")
			}
			if len(matches) == 0 {
				b.Error("Missed target")
			}
			targetInd = (targetInd + 1) % len(targets)
		} else {
			event := `{"x":` + fmt.Sprintf("%.6f", rand.Float64()) + "}"
			_, err := cm.matchesForJSONEvent([]byte(event))
			if err != nil {
				b.Error("match non-target")
			}
		}
	}
}

func TestWildlyVaryingNumbersAreComparable(t *testing.T) {
	data := []float64{
		-5_000_000_000, -4_999_999_999.99999, -4_999_999_999.99998, -4_999_999_999.99997,
		-999999999.99999, -999999999.99, -10000, -122.413496, -0.000002,
		0, 0.000001, 3.8, 3.9, 11, 12, 122.415028, 2.5e4, 999999999.999998, 999999999.999999,
		4_999_999_999.99997, 4_999_999_999.99998, 4_999_999_999.99999, 5_000_000_000,
	}
	for i := 1; i < len(data); i++ {
		s0 := qNumFromFloat(data[i-1])
		s1 := qNumFromFloat(data[i])
		if bytes.Compare(s0, s1) >= 0 {
			t.Errorf("FOO %d / %f - %f", i, data[i-1], data[i])
			fmt.Printf("lo %s %f\nhi %s %f\n", s0, data[i-1], s1, data[i])
		}
	}
}

/* needs extension to ruler-style anything-but
func TestNumericAnythingBut(t *testing.T) {
	pat := `{"x": [ { "anything-but": [3.50, 4.5e1]}`
	m := newCoreMatcher()
	err := m.addPattern("p", pat)
	if err != nil {
		t.Error("Add Pattern: " + err.Error())
	}
	event := `{"x": 3.5}`
	matches, _ := m.matchesForJSONEvent([]byte(event))
	if len(matches) != 0 {
		t.Error("NumAB")
	}
}
*/

func TestShowBigSmall(t *testing.T) {
	lows := []string{"-5_000_000_000.00000", "-4_999_999_999.99999", "-4_999_999_999.99998"}
	highs := []string{"4_999_999_999.99998", "4_999_999_999.99999", "5_000_000_000.00000"}
	for _, low := range lows {
		_, err := qNumFromBytes([]byte(low))
		if err != nil {
			t.Errorf("Problem with %s: %s", low, err.Error())
		}
	}
	for _, high := range highs {
		_, err := qNumFromBytes([]byte(high))
		if err != nil {
			t.Errorf("Problem with %s: %s", high, err.Error())
		}
	}
}

func TestBadNumbers(t *testing.T) {
	var err error
	bads := []string{
		"xy", "- 53", "124x", "1.5ee7",
	}
	for _, bad := range bads {
		_, err = qNumFromBytes([]byte(bad))
		if err == nil {
			t.Error("Accepted: " + bad)
		}
	}
}

func TestFloatVariants(t *testing.T) {
	f := []float64{350, 350.0, 350.0000000000, 3.5e2}
	var o []qNumber
	for _, s := range f {
		c := qNumFromFloat(s)
		o = append(o, c)
	}
	for i := 1; i < len(o); i++ {
		if !bytes.Equal(o[i], o[i-1]) {
			t.Errorf("%s and %s differ", o[i-1], o[i])
		}
	}
}
func TestByteVariants(t *testing.T) {
	f := []string{"350", "350.0", "350.0000", "3.5e2"}
	var o []qNumber
	for _, s := range f {
		c, err := qNumFromBytes([]byte(s))
		if err != nil {
			t.Errorf("qnum err on %s: %s", s, err.Error())
		}
		o = append(o, c)
	}
	for i := 1; i < len(o); i++ {
		if !bytes.Equal(o[i], o[i-1]) {
			t.Errorf("%s and %s differ", o[i-1], o[i])
		}
	}
}

func TestOrdering(t *testing.T) {
	var in []float64
	for i := 0; i < 10000; i++ {
		// nolint:gosec
		f := rand.Float64() * math.Pow(10, 9) * 2
		f -= 1000000000.0
		in = append(in, f)
	}
	slices.Sort(in)
	var out []string
	for _, f := range in {
		c := qNumFromFloat(f)
		out = append(out, string(c))
	}
	if !slices.IsSorted(out) {
		t.Errorf("Not sorted")
	}
}

func TestMatcherNumerics(t *testing.T) {
	p := `{"x": [35.0]}`
	shoulds := []string{
		"35", "3.5e1", "35.000", "0.000035e6",
	}
	for _, should := range shoulds {
		f, err := strconv.ParseFloat(should, 64)
		if err != nil {
			t.Error("Parse? " + err.Error())
		}
		_, err = qNumFromBytes([]byte(should))
		if err != nil {
			t.Error("QF: " + err.Error())
		}
		if f != 35.0 {
			t.Error("Not 35!")
		}
	}

	template := `{"x": NUM}`
	m := newCoreMatcher()
	err := m.addPattern("35", p)
	if err != nil {
		t.Error("Oops " + err.Error())
	}
	for _, should := range shoulds {
		event := strings.Replace(template, "NUM", should, 5)
		matches, err := m.matchesForJSONEvent([]byte(event))
		if err != nil {
			t.Error("Match: " + err.Error())
		}
		if len(matches) != 1 {
			t.Error("Didn't match " + should)
		}
	}
}
