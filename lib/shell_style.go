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
		if valBytes[i] == '*' && valBytes[i - 1] == '*' {
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

// makeShellStyleAutomaton - recognize a "-delimited string containing one or more '*' globs. It's useful that
//  the string ends with a '"' because we don't have to deal with the special case of '*' at end.  Arguably, if
//  we ignored the '"' markers, we could be a little more efficient matching "foo*" but it'd add complexity
func makeShellStyleAutomaton(val []byte, useThisTansition *fieldMatcher) (start smallStep, nextField *fieldMatcher) {
	table := newSmallTable()
	start = table

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

	// last byte, can't be '*'
	if useThisTansition != nil {
		nextField = useThisTansition
	} else {
		nextField = newFieldMatcher()
	}
	table.addByteStep(val[len(val)-1], newSmallTransition(nextField))
	return
}
