package quamina

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func readAnythingButSpecial(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, err error) {
	t, err := pb.jd.Token()
	if err != nil {
		return
	}
	pathVals = valsIn
	fieldCount := 0
	delim, ok := t.(json.Delim)
	if (!ok) || delim != '[' {
		err = errors.New("value for anything-but must be an array")
		return
	}
	done := false
	val := typedVal{vType: anythingButType}
	for !done {
		t, err = pb.jd.Token()
		if errors.Is(err, io.EOF) {
			err = errors.New("anything-but list truncated")
			return
		} else if err != nil {
			return
		}
		switch tt := t.(type) {
		case json.Delim:
			if tt == ']' {
				done = true
			} else {
				err = fmt.Errorf("spurious %c in anything-but list", tt)
			}
		case string:
			fieldCount++
			val.list = append(val.list, []byte(`"`+tt+`"`))
		default:
			err = errors.New("malformed anything-but list")
			done = true
		}
	}
	if err != nil {
		return
	}
	if fieldCount == 0 {
		err = errors.New("empty list in 'anything-but' pattern")
		return
	}
	pathVals = append(pathVals, val)

	// this has to be a '}' or you're going to get an err from the tokenizer, so no point looking at the value
	_, err = pb.jd.Token()
	return
}

// makeMultiAnythingButDFA exists to handle constructs such as
//
// {"x": [ {"anything-but": [ "a", "b" ] } ] }
//
// A finite automaton that matches anything but one byte sequence is like this:
// For each byte in val with value Z, we produce a table that leads to a nextField match on all non-Z values,
// and to another such table for Z. After all the bytes have matched, a match on valueTerminator leads to
// an empty table with no field Transitions, all others to a nexField match
//
// Making a succession of anything-but automata for each of "a" and "b" and then merging them turns out not
// to work because what the caller means is really an AND - everything that matches neither "a" nor "b". So
// in principle we could intersect automata.
func makeMultiAnythingButFA(vals [][]byte) (*smallTable, *fieldMatcher) {
	nextField := newFieldMatcher()
	success := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{nextField}}

	ret, _ := makeOneMultiAnythingButStep(vals, 0, success), nextField
	return ret, nextField
}

// makeOneMultiAnythingButStep - spookeh. The idea is that there will be N smallTables in this FA, where N is
// the longest among the vals. So for each value from 0 through N, we make a smallTable whose default is
// success but transfers to the next step on whatever the current byte in each of the vals that have not
// yet been exhausted. We notice when we get to the end of each val and put in a valueTerminator transition
// to a step with no nextField entry, i.e. failure because we've exactly matched one of the anything-but
// strings.
func makeOneMultiAnythingButStep(vals [][]byte, index int, success *faState) *smallTable {
	// this will be the default transition in all the anything-but tables.
	var u unpackedTable
	for i := range u {
		u[i] = success
	}

	// for the char at position 'index' in each val. valsWithBytesRemaining is keyed by that char (assuming that 'index' isn't
	// off the edge of that val. valsEndingHere[index] being true for some val means that val ends here.
	valsWithBytesRemaining := make(map[byte][][]byte)
	valsEndingHere := make(map[byte]bool)
	for _, val := range vals {
		lastIndex := len(val) - 1
		switch {
		case index < lastIndex:
			// gather vals that still have characters past 'index'
			utf8Byte := val[index]
			step := valsWithBytesRemaining[utf8Byte]
			valsWithBytesRemaining[utf8Byte] = append(step, val)
		case index == lastIndex:
			// remember if this particular val ends here
			valsEndingHere[val[index]] = true
		case index > lastIndex:
			// no-op
		}
	}

	// for each val that still has bytes to process, recurse to process the next one
	for utf8Byte, val := range valsWithBytesRemaining {
		nextTable := makeOneMultiAnythingButStep(val, index+1, success)
		nextStep := &faState{table: nextTable}
		u[utf8Byte] = nextStep
	}

	// for each val that ends at 'index', put a failure-transition for this anything-but
	// if you hit the valueTerminator, success for everything else
	for utf8Byte := range valsEndingHere {
		failState := &faState{table: newSmallTable()} // note no transitions
		lastTable := makeSmallTable(success, []byte{valueTerminator}, []*faState{failState})
		u[utf8Byte] = &faState{table: lastTable}
	}

	table := newSmallTable()
	table.pack(&u)
	return table
}
