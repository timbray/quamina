package quamina

import (
	"encoding/json"
	"errors"
	"fmt"
)

func readShellStyleSpecial(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, err error) {
	t, err := pb.jd.Token()
	if err != nil {
		return
	}
	pathVals = valsIn
	shellString, ok := t.(string)
	if !ok {
		err = errors.New("value for `shellstyle` must be a string")
		return
	}

	// no adjacent wildcards
	valBytes := []byte(shellString)
	for i := 1; i < len(valBytes); i++ {
		if valBytes[i] == '*' && valBytes[i-1] == '*' {
			err = errors.New("adjacent '*' characters not allowed in shellstyle pattern")
			return
		}
	}

	pathVals = append(pathVals, typedVal{vType: shellStyleType, val: `"` + shellString + `"`})

	t, err = pb.jd.Token()
	if err != nil {
		return
	}
	switch tt := t.(type) {
	case json.Delim:
		if tt != '}' {
			err = errors.New(fmt.Sprintf("invalid character %v in 'shellstyle' pattern", tt))
		}
	default:
		err = errors.New("trailing garbage in shellstyle pattern")
	}

	return
}

// makeShellStyleAutomaton - recognize a "-delimited string containing one or more '*' globs.
// TODO: Add “?”
func makeShellStyleAutomaton(val []byte, useThisTansition *fieldMatcher) (start *smallTable, nextField *fieldMatcher) {
	table := newSmallTable()
	start = table
	if useThisTansition != nil {
		nextField = useThisTansition
	} else {
		nextField = newFieldMatcher()
	}

	// since this is provided as a string, the last byte will be '"'. In the special case where the pattern ends
	//  with '*' (and thus the string ends with '*"', we will insert a successful transition as soon as we hit
	//  that last '*', so that the reaching the transition doesn't require going through the trailing characters to
	//  reach the '"'
	if val[len(val) - 2] == '*' {
		for i := 0; i < len(val) - 2; i++ {
			ch := val[i]
			if ch == '*' {
				table.addRangeSteps(0, ByteCeiling, table)
			} else {
				next := newSmallTable()
				table.addByteStep(ch, next)
				table = next
			}
		}
		table.addRangeSteps(0, ByteCeiling, newSmallTransition(nextField))
		return
	}

	// loop through all but last byte
	for i := 0; i < len(val)-1; i++ {
		ch := val[i]
		if ch == '*' {
			// just loop back
			table.addRangeSteps(0, ByteCeiling, table)
		} else {
			next := newSmallTable()
			table.addByteStep(ch, next)
			table = next
		}
	}

	table.addByteStep(val[len(val)-1], newSmallTransition(nextField))
	return
}
