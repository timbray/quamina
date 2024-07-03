package quamina

import (
	"bytes"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"testing"
)

func TestWildlyVaryingNumbersAreComparable(t *testing.T) {
	data := []float64{
		-FiveBillion, -4_999_999_999.99999, -4_999_999_999.99998, -4_999_999_999.99997,
		-999999999.99999, -999999999.99, -10000, -122.413496, -0.000002,
		0, 0.000001, 3.8, 3.9, 11, 12, 122.415028, 2.5e4, 999999999.999998, 999999999.999999,
		4_999_999_999.99997, 4_999_999_999.99998, 4_999_999_999.99999, FiveBillion,
	}
	for i := 1; i < len(data); i++ {
		s0, err := canonicalFromFloat(data[i-1])
		if err != nil {
			t.Error("s0")
		}
		s1, err := canonicalFromFloat(data[i])
		if err != nil {
			t.Error("s1")
		}
		if bytes.Compare(s0, s1) >= 0 {
			t.Errorf("FOO %d / %f - %f", i, data[i-1], data[i])
			fmt.Printf("lo %s0\nhi %s\n", s0, s1)
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
		c, err := canonicalFromBytes([]byte(low))
		if err != nil {
			t.Errorf("Problem with %s: %s", low, err.Error())
		}
		fmt.Printf("%s %s\n", low, c)
	}
	for _, high := range highs {
		c, err := canonicalFromBytes([]byte(high))
		if err != nil {
			t.Errorf("Problem with %s: %s", high, err.Error())
		}
		fmt.Printf("%s %s\n", high, c)
	}
}

func TestBadNumbers(t *testing.T) {
	var err error
	_, err = canonicalFromFloat(9999999999999999999)
	if err == nil {
		t.Error("took 20 9's")
	}
	_, err = canonicalFromFloat(9000000000000)
	if err == nil {
		t.Error("took huge number")
	}
	bads := []string{
		"5_000_000_001", "-5_000_000_001",
		"5_000_000_000.001", "-5_000_000_000.001",
		"3.1234567", "-5.1234567890",
		"0.0000001", "-0.0000001", "124x",
	}
	for _, bad := range bads {
		_, err = canonicalFromBytes([]byte(bad))
		if err == nil {
			t.Error("Accepted: " + bad)
		}
	}
}

func TestExponentialDigits(t *testing.T) {
	goods := []string{
		"3.1234567e3", "-.123456789012345e10",
		"0.0000001e3", "-0.0000001e2",
	}
	for _, good := range goods {
		_, err := canonicalFromBytes([]byte(good))
		if err != nil {
			t.Error("Rejected: " + good + ": " + err.Error())
		}
	}
}

func TestVariants(t *testing.T) {
	f := []float64{350, 350.0, 350.0000000000, 3.5e2}
	var o []canonicalNumber
	for _, s := range f {
		c, err := canonicalFromFloat(s)
		if err != nil {
			t.Errorf("canon err on %f: %s", s, err.Error())
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
	sort.Float64s(in)
	var out []string
	for _, f := range in {
		c, err := canonicalFromFloat(f)
		if err != nil {
			t.Errorf("failed on %f", f)
		}
		out = append(out, string(c))
	}
	if !sort.StringsAreSorted(out) {
		t.Errorf("Not sorted")
	}
	for i, c := range out {
		if len(c) != 14 {
			t.Errorf("%s: %d at %d", c, len(c), i)
		}
	}
}
