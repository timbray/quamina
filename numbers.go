package quamina

import (
	"encoding/binary"
	"errors"
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
// In Quamina these are called "Q numbers".
// How It's Done
// There is considerable effort to track, at the NFA level, which NFAs are built to match field values
// that are Q numbers; see vmFields.hasQNumbers. Similarly, the JSONFlattener, since it has to
// look at all the digits in a number in order to parse it, can keep track of whether it can be made
// a Q number. The key benefit of this is in valueMatcher.transitionOn, which incurs the cost of
// making a Q number only if it is known that the valueMatcher's NFA can benefit from it and
// that the number in the incoming event can in fact be made a Q number.

const (
	TenE6               = 1e6
	FiveBillion         = 5e9
	Hexes               = "0123456789ABCDEF"
	MaxFractionalDigits = 5
)

type qNumber []byte

// qNumFromBytes works out whether a string representing a number falls within the
// limits imposed for Q numbers. It is heavily optimized and relies on  the form
// of the number already having been validated, e.g. by flattenJSON().
func qNumFromBytes(bytes []byte) (qNumber, error) {
	// shortcut: The shorest number with more than 5 fractional digits is like 0.123456
	if len(bytes) < 8 {
		numeric, err := strconv.ParseFloat(string(bytes), 64)
		if err != nil {
			return nil, errors.New("not a float") // should never happen, json parser upstream
		}
		return qNumFromFloat(numeric)
	}
	// compute number of fractional digits. The loop below relies on the fact that anything between '.' and either
	// 'e' or the end of the string must be a digit, as must anything between 'e' and the end of the string.
	//. NOTE: This will be fooled by "35.000000"
	fracStart := 0
	expStart := 0
	index := 0
	var utf8Byte byte
	fractionalDigits := 0
ForEachByte:
	for index, utf8Byte = range bytes {
		switch utf8Byte {
		case '.':
			fracStart = index + 1
		case 'e', 'E':
			expStart = index + 1
			break ForEachByte
		}
	}
	if fracStart != 0 {
		fractionalDigits = index - fracStart
	}
	// if too many fractional digits, perhaps the exponent will push the '.' to the right
	if fractionalDigits > MaxFractionalDigits {
		if expStart != 0 {
			exp, err := strconv.ParseInt(string(bytes[expStart:]), 10, 32)
			if err == nil {
				fractionalDigits -= int(exp)
			}
		}
	}
	if fractionalDigits > MaxFractionalDigits {
		return nil, errors.New("more than 5 fractional digits")
	}

	numeric, err := strconv.ParseFloat(string(bytes), 64)
	if err != nil {
		return nil, errors.New("not a float") // shouldn't happen, upstream parser should prvent
	}
	return qNumFromFloat(numeric)
}

func qNumFromFloat(f float64) (qNumber, error) {
	if f < -FiveBillion || f > FiveBillion {
		return nil, errors.New("value must be between -5e9 and +5e9 inclusive")
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
