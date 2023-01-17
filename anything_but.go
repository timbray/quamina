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

// makeMultiAnythingButAutomaton exists to handle constructs such as
//
// {"x": [ {"anything-but": [ "a", "b" ] } ] }
//
// A DFA that matches anything but one byte sequence is like this:
// For each byte in val with value Z, we produce a table that leads to a nextField match on all non-Z values,
// and to another such table for Z. After all the bytes have matched, a match on valueTerminator leads to
// an empty table with no field Transitions, all others to a nexField match
//
// Making a succession of anything-but automata for each of "a" and "b" and then merging them turns out not
// to work because what the caller means is really an AND - everything that matches neither "a" nor "b". So
// in principle we could intersect automata.
func makeMultiAnythingButAutomaton(vals [][]byte, useThisTransition *fieldMatcher) (*smallTable[*dfaStep], *fieldMatcher) {
	var nextField *fieldMatcher
	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}
	ret, _ := oneMultiAnythingButStep(vals, 0, nextField), nextField
	return ret, nextField
}

// oneMultiAnythingButStep - spookeh
func oneMultiAnythingButStep(vals [][]byte, index int, nextField *fieldMatcher) *smallTable[*dfaStep] {
	success := &dfaStep{table: newSmallTable[*dfaStep](), fieldTransitions: []*fieldMatcher{nextField}}
	var u unpackedTable[*dfaStep]
	for i := range u {
		u[i] = success
	}
	// for the char at position 'index' in each val
	nextSteps := make(map[byte][][]byte)
	lastSteps := make(map[byte]bool)
	for _, val := range vals {
		lastIndex := len(val) - 1
		switch {
		case index < lastIndex:
			utf8Byte := val[index]
			step := nextSteps[utf8Byte]
			nextSteps[utf8Byte] = append(step, val)
		case index == lastIndex:
			lastSteps[val[index]] = true
		case index > lastIndex:
			// no-op
		}
	}

	for utf8Byte, valList := range nextSteps {
		u[utf8Byte] = &dfaStep{table: oneMultiAnythingButStep(valList, index+1, nextField)}
	}
	for utf8Byte := range lastSteps {
		lastStep := &dfaStep{table: newSmallTable[*dfaStep]()} // note no transition
		u[utf8Byte] = &dfaStep{table: makeSmallDfaTable(success, []byte{valueTerminator}, []*dfaStep{lastStep})}
	}
	table := newSmallTable[*dfaStep]()
	table.pack(&u)
	return table
}
