package quamina

import (
	"errors"
	"fmt"
	"strconv"
)

// You can't easily build automata to compare numbers based on either the decimal notation found
// in text data or the internal floating-point bits. Therefore, we map floating-point numbers
// (which is what JSON numbers basically are) to comparable slices of 7-bit bytes which preserve the
// numbers' ordering. Versions of Quamina up to 1.3 used a home-grown format which used 14 hex digits
// to represent a subset of numbers. This has now been replaced by Arne Hormann's "numbits"
// construct, see numbits.go. It uses up to 10 base128 bytes to represent the entire range of float64 numbers.
// Both this file and numbits.go are very short, but I'm keeping them separated because someone might
// figure out a still-better serialization of numbers and then this part wouldn't have to change.
// In Quamina these are called "Q numbers".

// There is considerable effort to track, at the NFA level, which NFAs are built to match field values
// that are Q numbers; see vmFields.hasNumbers. Similarly, the JSONFlattener, since it has to
// look at all the digits in a number in order to parse it, can keep track of whether it can be made
// a Q number. The key benefit of this is in valueMatcher.transitionOn, which incurs the cost of
// making a Q number only if it is known that the valueMatcher's NFA can benefit from it and
// that the number in the incoming event can in fact be made a Q number.

type qNumber []byte

// qNumFromBytes works out whether a string representing a number falls within the
// limits imposed for Q numbers. It is heavily optimized and relies on  the form
// of the number already having been validated, e.g. by flattenJSON().
func qNumFromBytes(bytes []byte) (qNumber, error) {
	numeric, err := strconv.ParseFloat(string(bytes), 64)
	if err != nil {
		return nil, errors.New("not a float") // should never happen, json parser upstream
	}
	return qNumFromFloat(numeric), nil
}

// qNumFromFLoat is here mostly to support testing
func qNumFromFloat(f float64) qNumber {
	return numbitsFromFloat64(f).toQNumber()
}

// for debugging
func (q qNumber) String() string {
	ret := ""
	for i, b := range q {
		if i != 0 {
			ret += "-"
		}
		ret += fmt.Sprintf("%02x", b)
	}
	return ret
}
