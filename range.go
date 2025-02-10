package quamina

import (
	"bytes"
	"fmt"
)

// Range represents a continuous block of numeric values with defined boundaries.
// It supports both conventional numeric ranges and CIDR/IP matching.
type Range struct {
	bottom     qNumber // Lower boundary
	top        qNumber // Upper boundary
	openBottom bool    // If true, range does not include bottom value
	openTop    bool    // If true, range does not include top value
	isCIDR     bool    // If true, interpret values as hex (for IP addresses)
}

// NewRange creates a new Range with the specified boundaries and options.
// The boundaries should be provided as string representations of numbers.
func NewRange(bottom string, openBottom bool, top string, openTop bool, isCIDR bool) (*Range, error) {
	var bottomNum, topNum qNumber
	var err error

	if bottom != "" {
		if isCIDR {
			// TODO: Implement CIDR-specific parsing
			return nil, fmt.Errorf("CIDR ranges not yet implemented")
		} else {
			bottomNum, err = qNumFromBytes([]byte(bottom))
			if err != nil {
				return nil, fmt.Errorf("invalid bottom boundary: %v", err)
			}
		}
	}

	if top != "" {
		if isCIDR {
			// TODO: Implement CIDR-specific parsing
			return nil, fmt.Errorf("CIDR ranges not yet implemented")
		} else {
			topNum, err = qNumFromBytes([]byte(top))
			if err != nil {
				return nil, fmt.Errorf("invalid top boundary: %v", err)
			}
		}
	}

	r := &Range{
		bottom:     bottomNum,
		top:        topNum,
		openBottom: openBottom,
		openTop:    openTop,
		isCIDR:     isCIDR,
	}

	if err := r.validate(); err != nil {
		return nil, err
	}

	return r, nil
}

// LessThan creates a Range that matches all values less than the given value
func LessThan(val string, isCIDR bool) (*Range, error) {
	return NewRange("", true, val, true, isCIDR)
}

// LessThanOrEqualTo creates a Range that matches values less than or equal to the given value
func LessThanOrEqualTo(val string, isCIDR bool) (*Range, error) {
	return NewRange("", true, val, false, isCIDR)
}

// GreaterThan creates a Range that matches values greater than the given value
func GreaterThan(val string, isCIDR bool) (*Range, error) {
	return NewRange(val, true, "", true, isCIDR)
}

// GreaterThanOrEqualTo creates a Range that matches values greater than or equal to the given value
func GreaterThanOrEqualTo(val string, isCIDR bool) (*Range, error) {
	return NewRange(val, false, "", true, isCIDR)
}

// Equals creates a Range that matches exactly the given value
func Equals(val string, isCIDR bool) (*Range, error) {
	return NewRange(val, false, val, false, isCIDR)
}

// Between creates a Range with explicitly defined boundaries
func Between(bottom string, openBottom bool, top string, openTop bool, isCIDR bool) (*Range, error) {
	return NewRange(bottom, openBottom, top, openTop, isCIDR)
}

// validate ensures the range boundaries are valid
func (r *Range) validate() error {
	// If both bounds are empty, the range is invalid
	if len(r.bottom) == 0 && len(r.top) == 0 {
		return fmt.Errorf("invalid range: at least one boundary must be specified")
	}

	// If both bounds are present, ensure bottom is less than top
	if len(r.bottom) > 0 && len(r.top) > 0 {
		if bytes.Compare(r.bottom, r.top) > 0 {
			return fmt.Errorf("invalid range: bottom boundary must be less than top boundary")
		}
	}

	return nil
}

// Contains checks if a value is within the range
func (r *Range) Contains(val qNumber) bool {
	if len(r.bottom) > 0 {
		cmp := bytes.Compare(val, r.bottom)
		if cmp < 0 || (r.openBottom && cmp == 0) {
			return false
		}
	}

	if len(r.top) > 0 {
		cmp := bytes.Compare(val, r.top)
		if cmp > 0 || (r.openTop && cmp == 0) {
			return false
		}
	}

	return true
}

// String returns a string representation of the range for debugging
func (r *Range) String() string {
	var bounds []string
	if len(r.bottom) > 0 {
		bounds = append(bounds, fmt.Sprintf("%s%s", map[bool]string{true: "(", false: "["}[r.openBottom], bytesToqNum(r.bottom)))
	} else {
		bounds = append(bounds, "(-∞")
	}

	if len(r.top) > 0 {
		bounds = append(bounds, fmt.Sprintf("%s%s", bytesToqNum(r.top), map[bool]string{true: ")", false: "]"}[r.openTop]))
	} else {
		bounds = append(bounds, "+∞)")
	}

	return fmt.Sprintf("%s, %s", bounds[0], bounds[1])
}

// bytesToqNum converts a byte slice to a float64 string
func bytesToqNum(b []byte) string {
	if len(b) == 0 {
		return "0"
	}
	val := qNumberToFloat64(b)
	return fmt.Sprintf("%.1f", val)
}
