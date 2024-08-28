package quamina

import (
	"bytes"
	"math"
	"math/rand"
	"sort"
	"testing"
	"unicode/utf8"
)

func TestToQNumber(t *testing.T) {
	rand.Seed(230948)
	var nbs []numbits
	var utf8s [][]byte
	for i := 0; i < 10000; i++ {
		nb := numbits(rand.Uint64())
		nbs = append(nbs, nb)
		nbu := nb.toQNumber()
		if !utf8.Valid(nbu) {
			t.Error("Invalid UTF8!")
		}
		utf8s = append(utf8s, nbu)
	}
	for i := 1; i < len(nbs); i++ {
		uCompare := bytes.Compare(utf8s[i], utf8s[i-1])
		if nbs[i] > nbs[i-1] {
			if uCompare <= 0 {
				t.Error("Out of order 1")
			}
		} else if nbs[i] < nbs[i-1] {
			if uCompare >= 0 {
				t.Error("Out of order 2")
			}
		} else if nbs[i] == nbs[i-1] {
			if uCompare != 0 {
				t.Error("Out of order 3")
			}
		}
	}
}

var (
	// boundaries of floating point value ranges
	f64Zero      = math.Float64frombits(0b0_00000000000_0000_00000000_00000000_00000000_00000000_00000000_00000000)
	f64SubnormLo = math.Float64frombits(0b0_00000000000_0000_00000000_00000000_00000000_00000000_00000000_00000001)
	f64SubnormHi = math.Float64frombits(0b0_00000000000_1111_11111111_11111111_11111111_11111111_11111111_11111111)
	f64NormLoLo  = math.Float64frombits(0b0_00000000001_0000_00000000_00000000_00000000_00000000_00000000_00000000)
	f64NormLoHi  = math.Float64frombits(0b0_00000000001_1111_11111111_11111111_11111111_11111111_11111111_11111111)
	f64NormHiLo  = math.Float64frombits(0b0_11111111110_0000_00000000_00000000_00000000_00000000_00000000_00000000)
	f64NormHiHi  = math.Float64frombits(0b0_11111111110_1111_11111111_11111111_11111111_11111111_11111111_11111111)
	specials     = []float64{f64Zero, f64SubnormHi, f64SubnormLo, f64NormLoLo, f64NormLoHi, f64NormHiLo, f64NormHiHi}
)

func TestNumbits_Compare(t *testing.T) {
	rand.Seed(203785)
	floats := append([]float64{}, specials...)

	for i := 0; i < 1000; i++ {
		floats = append(floats, rand.Float64())
	}
	sort.Float64s(floats)
	last := numbitsFromFloat64(floats[0])
	for i := 1; i < len(floats); i++ {
		this := numbitsFromFloat64(floats[i])
		if last >= this {
			t.Error("out of order")
		}
		last = this
	}
}
