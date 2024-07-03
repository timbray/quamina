package quamina

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
)

// You can't easily build automata to compare numbers based on either the decimal notation found
// in text data or the internal floating-point bits. Therefore, for a restricted subset of numbers,
// we define a 7-byte (14 hex digit) representation that facilitates building automata to support
// equality and ordering comparison.
//
// The representation supports 10**15 numbers. The first three are:
// decimal: -5_000_000_000, -4_999_999_999.99999, -4_999_999_999.99998, ...
// 14-byte: 00000000000000,       00000000000009,       00000000000014
// and the last three are
// decimal: .., 4_999_999_999.99998, 4_999_999_999.99999,  5_000_000_000
// 14-byte:          2386F26FC0FFEC,      2386F26FC0FFF6, 2386F26FC10000
//
// In English: all numbers that are between negative and positive 5 billion inclusive, with up to five
// digits after the decimal point.
// These numbers have fifteen decimal digits of precision, which is what double floats can offer.
// They include most numbers that are used in practice, including prices, occurrence counts, size
// measurements, and so on.
// Examples of numbers that do NOT meet these criteria include AWS account numbers, some telephone
// numbers, and cryptographic keys/signatures. For these, treatment as strings seems to produce
// satisfactory results for equality testing.

const (
	TenE6               = 1e6
	FiveBillion         = 5e9
	Hexes               = "0123456789ABCDEF"
	MaxFractionalDigits = 5
	ASCII0              = 48
	ASCII9              = 57
)

type canonicalNumber []byte

func canonicalFromBytes(bytes []byte) (canonicalNumber, error) {
	// compute number of fractional digits
	seenDot := false
	expStart := 0
	var fractionalDigits int64 = 0
ForEachByte:
	for i, utf8Byte := range bytes {
		switch {
		case seenDot && utf8Byte >= ASCII0 && utf8Byte <= ASCII9:
			fractionalDigits++
		case utf8Byte == '.':
			seenDot = true
		case utf8Byte == 'e' || utf8Byte == 'E':
			// TODO: Test this bit
			expStart = i + 1
			break ForEachByte
		}
	}
	if expStart != 0 {
		exp, err := strconv.ParseInt(string(bytes[expStart:]), 10, 32)
		if err == nil {
			fractionalDigits -= exp
		}
	}
	if fractionalDigits > MaxFractionalDigits {
		return nil, fmt.Errorf("more than %d fractional digits", MaxFractionalDigits)
	}

	numeric, err := strconv.ParseFloat(string(bytes), 64)
	if err != nil {
		return nil, errors.New("not a float")
	}
	return canonicalFromFloat(numeric)
}
func canonicalFromFloat(f float64) (canonicalNumber, error) {
	if f < -FiveBillion || f > FiveBillion {
		return nil, fmt.Errorf("value must be between %.0f and %.0f inclusive", -FiveBillion, FiveBillion)
	}
	value := uint64(TenE6 * (FiveBillion + f))
	return toHexStringSkippingFirstByte(value), nil
}

func toHexStringSkippingFirstByte(value uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], value)
	var outputChars [14]byte
	for i, utf8Byte := range buf {
		if i == 0 {
			continue
		}
		pos := (i - 1) * 2
		outputChars[pos] = Hexes[utf8Byte>>4]
		outputChars[pos+1] = Hexes[buf[i]&0xf]
	}
	return outputChars[:]
}
