package quamina

import (
	"math"
)

// float64 are stored as (sign | exponent | mantissa)
// with 1 bit sign, 11 bits exponent, 52 bits mantissa
const ()

// numbits is an alternative binary representation of float64 numbers.
// They can be represented as [8]byte or as string and can be created from
// these representations.
// All possible float64 values are representable as numbits.
// numbits were implemented by Arne Hormann for Quamina; he later discovered
// that an equivalent representation was used long ago in the disk format of DB2.
//
// Arne's implementation carefully handled NaN, +/-0, and infinities, but
// Quamina ignores those issues because a combination of JSON rules and
// Quamina's parsers prevent those values from occurring.
type numbits uint64

// numbitsFromFloat64 converts a float64 value to its numbits representation.
func numbitsFromFloat64(f float64) numbits {
	u := math.Float64bits(f)
	// transform without branching (inverse of numbits.Float64):
	// if high bit is 0, xor with sign bit 1 << 63, else negate (xor with ^0)
	mask := (u>>63)*^uint64(0) | (1 << 63)
	return numbits(u ^ mask)
}

// toUTF8 turns a numbits into 10 bytes of UTF-8 encoded via Base-256
// code copied with thanks from a sample by Axel Wagner
func (nb numbits) toUTF8() []byte {
	var b [10]byte
	for i := len(b) - 1; i >= 0; i-- {
		b[i] = byte(nb & 0x7f)
		nb >>= 7
	}
	return b[:]
}
