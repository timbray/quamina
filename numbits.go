package quamina

import (
	"math"
)

// numbits is an alternative binary representation of float64 numbers.
// float64 are stored as (sign | exponent | mantissa)
// with 1 bit sign, 11 bits exponent, 52 bits mantissa
// They can be represented as [8]byte or as string and can be created from
// these representations.
// All possible float64 values are representable as numbits.
// numbits were implemented by Arne Hormann for Quamina; he later discovered
// that an equivalent representation was used long ago in the disk format of DB2.
//
// Arne's implementation carefully handled NaN, -0, and infinities, but
// Quamina ignores those issues because a combination of JSON rules and
// Quamina's parsers prevent those values from occurring.
type numbits uint64

// numbitsFromFloat64 converts a float64 value to its numbits representation.
func numbitsFromFloat64(f float64) numbits {
	u := math.Float64bits(f)
	// transform without branching:
	// if high bit is 0, xor with sign bit 1 << 63, else negate (xor with ^0)
	mask := uint64(int64(u)>>63) | (1 << 63)
	return numbits(u ^ mask)
}

const MaxBytesInEncoding = 10

// toQNumber turns a numbits into a minimal variable-width encoding that preservers equality and ordering.
// Storing 8 bytes of data in base-128 would in principle require 10 bytes, but it turns out that since
// the byte-string encoding is big-endian, trailing zeroes don't count, so the encoding can be as short as
// one byte.
// Idea and some code by Axel Wagner
func (nb numbits) toQNumber() qNumber {
	// Iterate through the numbits 7 bits at a time, right to left, first bypassing bits that generate
	// trailing zeroes in the encoded form. Note that index could go to 0 if the numbits value was uint(0)
	// but that value represents NaN and can't appear in JSON
	trailingZeroes := 0
	var index int
	for index = MaxBytesInEncoding - 1; index >= 0; index-- {
		if nb&0x7f != 0 {
			break
		}
		trailingZeroes++
		nb >>= 7
	}

	// now we fill in the byte encoding for the digits up to the last non-zero
	b := make([]byte, MaxBytesInEncoding-trailingZeroes)
	for ; index >= 0; index-- {
		b[index] = byte(nb & 0x7f)
		nb >>= 7
	}
	return b
}
