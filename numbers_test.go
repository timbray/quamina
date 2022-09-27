package quamina

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
)

func TestBadNumbers(t *testing.T) {
	var err error
	_, err = canonicalize([]byte("9999999999999999999"))
	if err == nil {
		t.Error("took 20 9's")
	}
	_, err = canonicalize([]byte("2z3z"))
	if err == nil {
		t.Error("took 2z3z")
	}
	_, err = canonicalize([]byte("9000000000000"))
	if err == nil {
		t.Error("took huge number")
	}
}

func TestVariants(t *testing.T) {
	f := []string{
		"350",
		"350.0",
		"350.0000000000",
		"3.5e2",
	}
	var o []string
	for _, s := range f {
		c, err := canonicalize([]byte(s))
		if err != nil {
			t.Errorf("canon err on %s: %s", s, err.Error())
		}
		o = append(o, c)
	}
	for i := 1; i < len(o); i++ {
		if o[i] != o[i-1] {
			t.Errorf("%s and %s differ", o[i-1], o[i])
		}
	}
}

func TestOrdering(t *testing.T) {
	var in []float64
	for i := 0; i < 10000; i++ {
		// nolint:gosec
		f := rand.Float64() * math.Pow(10, 9) * 2
		f -= nineDigits
		in = append(in, f)
	}
	sort.Float64s(in)
	var out []string
	for _, f := range in {
		s := fmt.Sprintf("%f", f)
		c, err := canonicalize([]byte(s))
		if err != nil {
			t.Errorf("failed on %s", s)
		}
		out = append(out, c)
	}
	if !sort.StringsAreSorted(out) {
		t.Errorf("Not sorted")
	}
	for i, c := range out {
		if len(c) != 19 {
			t.Errorf("%s: %d at %d", c, len(c), i)
		}
	}
}
