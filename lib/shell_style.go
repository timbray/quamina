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
	globs := 0
	for _, ch := range valBytes {
		if ch == '*' {
			globs++
		}
	}
	if globs > 1 {
		err = errors.New("only one '*' character allowed in a shellstyle pattern")
		return
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

// makeShellStyleAutomaton - recognize a "-delimited string containing one '*' glob.
// TODO: Make this recursive like makeStringAutomaton
func makeShellStyleAutomaton(val []byte, useThisTransition *fieldMatcher) (start *smallTable, nextField *fieldMatcher) {
	table := newSmallTable()
	start = table
	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}

	// loop through all but last byte
	var globStep smallStep = nil
	i := 0
	for i < len(val)-1 {
		ch := val[i]
		if ch == '*' {
			// special-case handling for string ending in '*"'
			if i == len(val)-2 {
				lastStep := newSmallTransition(nextField)
				table.addRangeSteps(0, ByteCeiling, lastStep)
				return
			}
			table.addRangeSteps(0, ByteCeiling, table)
			globStep = table
		} else {
			next := newSmallTable()
			if globStep != nil {
				table.addRangeSteps(0, ByteCeiling, globStep)
			}
			table.addByteStep(ch, next)
			table = next
		}
		i++
	}
	table.addByteStep(val[len(val)-1], newSmallTransition(nextField))

	return
}
