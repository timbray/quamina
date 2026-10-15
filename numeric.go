package quamina

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// A numeric pattern matches number-valued event fields that fall within a range:
//
//	{"risk": [{"numeric": [">=", 3]}]}
//	{"price": [{"numeric": [">", 0, "<=", 9.5]}]}
//	{"count": [{"numeric": ["=", 100]}]}
//
// The operators are "<", "<=", ">", ">=", and "="; there may be one lower bound and one upper bound, or a
// single "=". Only JSON numbers in events are matched, never strings that look like numbers.
//
// Ranges are compiled over numbits, the order-preserving form of float64 that underlies Q numbers, written
// as numericDigits fixed-width base-128 digits. A range then becomes a deterministic automaton of at most
// two states per digit, one tracking each bound. Numeric ranges have their own automaton per field, which
// only number-valued event fields traverse, because Q number bytes overlap the bytes of quoted strings and
// literals.

const numericDigits = MaxBytesInEncoding

var (
	minNumericDigits = numericDigitsOf(0)
	maxNumericDigits = numericDigitsOf(^numbits(0))

	// JSON numbers are finite, so ranges are clamped to the finite float64s
	minFiniteNumbits = numbitsFromFloat64(-math.MaxFloat64)
	maxFiniteNumbits = numbitsFromFloat64(math.MaxFloat64)
)

type numericRange struct{ lo, hi numbits } // inclusive

func numericDigitsOf(nb numbits) (d [numericDigits]byte) {
	for i := numericDigits - 1; i >= 0; i-- {
		d[i] = byte(nb & 0x7f)
		nb >>= 7
	}
	return
}

func readNumericSpecial(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, err error) {
	pathVals = valsIn
	t, err := pb.jd.Token()
	if err != nil {
		return
	}
	if d, ok := t.(json.Delim); !ok || d != '[' {
		err = errors.New("value for 'numeric' must be an array")
		return
	}

	r := numericRange{lo: 0, hi: ^numbits(0)}
	var hasLower, hasUpper, hasEquals bool
	for {
		t, err = pb.jd.Token()
		if err != nil {
			return
		}
		if _, ok := t.(json.Delim); ok {
			// has to be ]
			break
		}
		op, ok := t.(string)
		if !ok {
			err = fmt.Errorf("'numeric' expected an operator, got %v", t)
			return
		}
		if t, err = pb.jd.Token(); err != nil {
			return
		}
		num, ok := t.(json.Number)
		if !ok {
			err = fmt.Errorf("'numeric' operator %q must be followed by a number", op)
			return
		}
		var f float64
		if f, err = strconv.ParseFloat(num.String(), 64); err != nil || math.IsInf(f, 0) {
			err = fmt.Errorf("'numeric' value %s is out of range", num)
			return
		}
		nb := numbitsFromFloat64(f)
		if hasEquals {
			err = errors.New("'numeric' \"=\" cannot be combined with other operators")
			return
		}
		switch op {
		case ">", ">=":
			if hasLower {
				err = errors.New("'numeric' allows only one lower bound")
				return
			}
			hasLower = true
			r.lo = nb
			if op == ">" {
				if nb == ^numbits(0) {
					err = errors.New("'numeric' range matches no numbers")
					return
				}
				r.lo++
			}
		case "<", "<=":
			if hasUpper {
				err = errors.New("'numeric' allows only one upper bound")
				return
			}
			hasUpper = true
			r.hi = nb
			if op == "<" {
				if nb == 0 {
					err = errors.New("'numeric' range matches no numbers")
					return
				}
				r.hi--
			}
		case "=":
			if hasLower || hasUpper {
				err = errors.New("'numeric' \"=\" cannot be combined with other operators")
				return
			}
			hasEquals = true
			r.lo, r.hi = nb, nb
		default:
			err = fmt.Errorf("'numeric' operator %q is not one of <, <=, >, >=, =", op)
			return
		}
	}
	if !hasLower && !hasUpper && !hasEquals {
		err = errors.New("'numeric' requires at least one operator")
		return
	}
	r.lo = max(r.lo, minFiniteNumbits)
	r.hi = min(r.hi, maxFiniteNumbits)
	if r.lo > r.hi {
		err = errors.New("'numeric' range matches no numbers")
		return
	}
	pathVals = append(pathVals, typedVal{vType: numericRangeType, val: r.canonical()})

	// has to be } or tokenizer will throw error
	_, err = pb.jd.Token()
	return
}

// canonical returns a string that is equal for two ranges exactly when they match the same numbers; the
// range can be rebuilt from it with numericRangeFromVal.
func (r numericRange) canonical() string {
	return fmt.Sprintf("%016x %016x", uint64(r.lo), uint64(r.hi))
}

func numericRangeFromVal(val string) numericRange {
	lo, hi, _ := strings.Cut(val, " ")
	l, errL := strconv.ParseUint(lo, 16, 64)
	h, errH := strconv.ParseUint(hi, 16, 64)
	if errL != nil || errH != nil {
		panic("malformed numeric range " + val)
	}
	return numericRange{lo: numbits(l), hi: numbits(h)}
}

// makeNumericRangeFA builds a deterministic automaton over numericDigits digits that transitions to next
// for exactly the numbits in r.
func makeNumericRangeFA(r numericRange, next *fieldMatcher) *faState {
	accept := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{next}}
	lo, hi := numericDigitsOf(r.lo), numericDigitsOf(r.hi)
	atMin := func(i int) bool { return bytes.Equal(lo[i:], minNumericDigits[i:]) }
	atMax := func(i int) bool { return bytes.Equal(hi[i:], maxNumericDigits[i:]) }
	state := func(u *unpackedTable) *faState {
		s := &faState{table: newSmallTable()}
		s.table.pack(u)
		return s
	}

	// above follows the lower bound's digits; everything greater than them is in range
	var above, below, between func(i int) *faState
	above = func(i int) *faState {
		if i == numericDigits || atMin(i) {
			return accept
		}
		var u unpackedTable
		for b := int(lo[i]) + 1; b < 0x80; b++ {
			u[b] = accept
		}
		u[lo[i]] = above(i + 1)
		return state(&u)
	}
	// below follows the upper bound's digits; everything less than them is in range
	below = func(i int) *faState {
		if i == numericDigits || atMax(i) {
			return accept
		}
		var u unpackedTable
		for b := 0; b < int(hi[i]); b++ {
			u[b] = accept
		}
		u[hi[i]] = below(i + 1)
		return state(&u)
	}
	between = func(i int) *faState {
		if i == numericDigits || (atMin(i) && atMax(i)) {
			return accept
		}
		var u unpackedTable
		if lo[i] == hi[i] {
			u[lo[i]] = between(i + 1)
			return state(&u)
		}
		u[lo[i]] = above(i + 1)
		for b := int(lo[i]) + 1; b < int(hi[i]); b++ {
			u[b] = accept
		}
		u[hi[i]] = below(i + 1)
		return state(&u)
	}

	start := between(0)
	if start == accept {
		// traversal collects fieldTransitions only from states it steps to, so the start can't accept
		var u unpackedTable
		for b := 0; b < 0x80; b++ {
			u[b] = accept
		}
		start = state(&u)
	}
	return start
}

// numericTransitions appends the transitions of the numeric-range automaton matching a number-valued field.
func numericTransitions(start *faState, field *Field, transitions []*fieldMatcher, bufs *nfaBuffers) []*fieldMatcher {
	f, err := strconv.ParseFloat(string(field.Val), 64)
	if err != nil || math.IsInf(f, 0) {
		return transitions
	}
	bufs.numericBuf = numericDigitsOf(numbitsFromFloat64(f))
	return traverseDFA(start, bufs.numericBuf[:], transitions)
}
