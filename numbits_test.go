package quamina

import (
	"math"
	"testing"
)

var (
	// special case, compiler does not create it when writing -0.0
	f64NegZero = math.Float64frombits(0b1_00000000000_0000_00000000_00000000_00000000_00000000_00000000_00000000)

	// boundaries of floating point value ranges
	f64Zero      = math.Float64frombits(0b0_00000000000_0000_00000000_00000000_00000000_00000000_00000000_00000000)
	f64SubnormLo = math.Float64frombits(0b0_00000000000_0000_00000000_00000000_00000000_00000000_00000000_00000001)
	f64SubnormHi = math.Float64frombits(0b0_00000000000_1111_11111111_11111111_11111111_11111111_11111111_11111111)
	f64NormLoLo  = math.Float64frombits(0b0_00000000001_0000_00000000_00000000_00000000_00000000_00000000_00000000)
	f64NormLoHi  = math.Float64frombits(0b0_00000000001_1111_11111111_11111111_11111111_11111111_11111111_11111111)
	f64NormHiLo  = math.Float64frombits(0b0_11111111110_0000_00000000_00000000_00000000_00000000_00000000_00000000)
	f64NormHiHi  = math.Float64frombits(0b0_11111111110_1111_11111111_11111111_11111111_11111111_11111111_11111111)
	f64Inf       = math.Float64frombits(0b0_11111111111_0000_00000000_00000000_00000000_00000000_00000000_00000000)
	f64NaNLo     = math.Float64frombits(0b0_11111111111_0000_00000000_00000000_00000000_00000000_00000000_00000001)
	f64NaNHi     = math.Float64frombits(0b0_11111111111_1111_11111111_11111111_11111111_11111111_11111111_11111111)

	// named values including boundaries
	values = func(positive map[string]float64) map[string]float64 {
		// this function mirrors the values to negative
		const sign uint64 = 1 << 63
		m2 := make(map[string]float64, len(positive)*2)
		for n, v := range positive {
			m2[n] = v
			m2["negative "+n] = math.Float64frombits(math.Float64bits(v) | sign)
		}
		return m2
	}(map[string]float64{
		"zero":                  f64Zero,
		"subnormal;lo":          f64SubnormLo,
		"subnormal;hi":          f64SubnormHi,
		"normal;lo-exp,lo-mant": f64NormLoLo,
		"normal;lo-exp,hi-mant": f64NormLoHi,
		"normal;hi-exp,lo-mant": f64NormHiLo,
		"normal;hi-exp,hi-mant": f64NormHiHi,
		"infinity":              f64Inf,
		"NaN;lo":                f64NaNLo,
		"NaN;hi":                f64NaNHi,
		"0.1":                   0.1,
		"1.0":                   1.0,
		"pi":                    math.Pi,
	})
)

func TestNumbits(t *testing.T) {
	// roundtrips for various values - creation and conversion
	normalNaN := NumbitsFromFloat64(math.NaN()).Normalize()
	for n, f := range values {
		t.Run(n, func(t *testing.T) {
			got := NumbitsFromFloat64(f)
			if f2 := got.Float64(); compare(f2, f) != 0 {
				t.Errorf("NumbitsFromFloat64().Float64() = %v, want %v", f2, f)
			}
			if bin := got.Bytes(); NumbitsFromBytes(bin) != got {
				t.Errorf("NumbitsFromBytes().Bytes() = %x, want %x", got, bin)
			}
			if str := got.BinaryString(); NumbitsFromBinaryString(str) != got {
				t.Errorf("NumbitsFromBytes().BinaryString() = %x, want %x", got, str)
			}
			if math.IsNaN(f) && got.Normalize() != normalNaN {
				t.Errorf("Normalize for NaN failed")
			}
			if got.IsFinite() == (math.IsNaN(f) || math.IsInf(f, 0)) {
				t.Errorf("IsFinite failed for %v", f)
			}
		})
	}
	t.Run("neg-zero_to_zero", func(t *testing.T) {
		negZero := NumbitsFromFloat64(f64NegZero)
		normZero := negZero.Normalize()
		if normZero == negZero {
			t.Errorf("Normalize for -0.0 failed")
		}
		if negZero.Float64() != 0 || normZero.Float64() != 0 {
			t.Errorf("0.0 representation error")
		}
	})
}

func TestNumbits_Compare(t *testing.T) {
	for n1, f1 := range values {
		v1 := NumbitsFromFloat64(f1)
		nanf1 := math.IsNaN(f1)
		for n2, f2 := range values {
			// redefine in scope so v1 can be changed without changing the outer one
			v1 := v1
			v2 := NumbitsFromFloat64(f2)
			nanf2 := math.IsNaN(f2)
			order := compare(f1, f2)
			if o := compare(v1.Float64(), v2.Float64()); order != o {
				t.Errorf("%v->%v: comparison after Float64() failed: want %v, got %v", n1, n2, order, o)
			}
			if nanf1 || (f1 == 0 && f2 == 0) {
				v1 = v1.Normalize()
			}
			if nanf2 || (f1 == 0 && f2 == 0) {
				v2 = v2.Normalize()
			}
			b1, b2 := v1.BinaryString(), v2.BinaryString()
			if o := compare(v1, v2); order != o {
				t.Errorf("%v->%v: direct comparison of Numbits failed: want %v, got %v for %x -> %x", n1, n2, order, o, b1, b2)
			}
			if o := compare(v1.BinaryString(), v2.BinaryString()); order != o {
				t.Errorf("%v->%v: comparison after BinaryString() failed: want %v, got %v for %x -> %x", n1, n2, order, o, b1, b2)
			}
		}
	}
}
