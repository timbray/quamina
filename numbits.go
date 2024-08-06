package quamina

import (
	"encoding/binary"
	"math"
)

// float64 are stored as (sign | exponent | mantissa)
// with 1 bit sign, 11 bits exponent, 52 bits mantissa
const (
	maskSign     uint64 = 1 << 63
	maskExponent uint64 = 0b11111111111 << 52
	maskMantissa uint64 = ^uint64(0) >> 12
)

// Numbits representation of some boundary values.
const (
	numbitsZero          = Numbits(maskSign)
	numbitsNegZero       = numbitsZero - 1
	numbitsNegInf        = Numbits(maskMantissa)
	numbitsPosInf        = Numbits(maskSign | maskExponent)
	numbitsNormalizedNaN = numbitsNegInf - 1
)

// Numbits is an alternative binary representation of float64 numbers.
// They can be represented as [8]byte or as string and can be created from
// these representations.
// All possible float64 values are representable as Numbits.
//
// The comparability differs from cmp.Compare for float64, though:
//   - 0.0 and -0.0 are not equal.
//   - NaNs are equal if their representation as bits is equal.
//   - NaNs can be either larger than Infinity
//     or smaller than -Infinity (depending on the sign bit).
//   - use Normalize() to align the comparability.
type Numbits uint64

// NumbitsFromFloat64 converts a float64 value to its Numbits representation.
func NumbitsFromFloat64(f float64) Numbits {
	u := math.Float64bits(f)
	// transform without branching (inverse of Numbits.Float64):
	// if high bit is 0, xor with sign bit 1 << 63, else negate (xor with ^0)
	mask := (u>>63)*^uint64(0) | (1 << 63)
	return Numbits(u ^ mask)
}

// NumbitsFromBytes converts a [8]byte value to its Numbits representation.
func NumbitsFromBytes(b [8]byte) Numbits {
	return Numbits(binary.BigEndian.Uint64(b[:]))
}

// NumbitsFromBinaryString converts a string value created by BinaryString to its Numbits representation.
// It uses the first 8 bytes from the string and panics if it is shorter.
func NumbitsFromBinaryString(s string) Numbits {
	// This code could use slice to array conversion, but at implementation time,
	// quamina still supported Go 1.19. The feature was introduced in 1.20.
	return Numbits(binary.BigEndian.Uint64([]byte(s[:8])))
}

// Float64 converts Numbits back to its float64 representation
func (n Numbits) Float64() float64 {
	u := uint64(n)
	// transform without branching (inverse of NumbitsFromFloat64):
	// if high bit is 1, xor with sign bit 1 << 63, else negate (xor with ^0)
	mask := (1-(u>>63))*^uint64(0) | (1 << 63)
	return math.Float64frombits(u ^ mask)
}

// Normalize the value to align the comparability to cmp.Compare.
//
// Normalization only affects -0.0 (converted to 0.0) and NaN (all converted to the same representation).
func (n Numbits) Normalize() Numbits {
	if n == numbitsNegZero {
		return numbitsZero
	}
	if n < numbitsNegInf || numbitsPosInf < n {
		return numbitsNormalizedNaN
	}
	return n
}

// IsFinite returns true iff n is not infinite or NaN.
func (n Numbits) IsFinite() bool {
	return numbitsNegInf < n && n < numbitsPosInf
}

// Bytes retrieves a representation as [8]byte.
// The returned bytes are in big-endian order.
func (n Numbits) Bytes() [8]byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(n))
	return b
}

// BinaryString retrieves a lexically ordered string representation.
func (n Numbits) BinaryString() string {
	b := n.Bytes()
	return string(b[:])
}
