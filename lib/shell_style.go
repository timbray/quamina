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

// mssA2 - recognize a "-delimited string containing one or more '*' globs.
//  This isn't quite as simple as you'd think.  Consider matching "*abc". When you see an 'a' you move to a state
//  where you're looking for 'b'. So if it's not a 'b' you go back to the '*' state. But suppose you see "xaabc";
//  when you're in that looking-for-'b' state and you see that second 'a', you don't go back to the '*' state, you
//  have to stay in the looking-for-'b' state because you have seen the 'a'.  Similarly, when you see 'xabac', when
//  you're looking for 'c' and you see the 'a', once again, you have to go to the looking-for-'b' state.  Let's
//  call the 'a the bounceBackByte and the looking-for-b state the bounceBackStep
func makeShellStyleAutomaton(val []byte, useThisTransition *fieldMatcher) (start *smallTable, nextField *fieldMatcher) {
	table := newSmallTable()
	start = table
	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}

	var bounceBackByte byte
	var bounceBackStep smallStep = nil
	var globStep smallStep = nil

	// loop through all but last bytea
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
			globStep = table
			i++
			bounceBackStep = newSmallTable()
			bounceBackByte = val[i]
			table.load(table, []byte{val[i]}, []smallStep{bounceBackStep})
			table = bounceBackStep.SmallTable()
		} else {
			next := newSmallTable()
			if globStep != nil {
				table.load(globStep, []byte{bounceBackByte}, []smallStep{bounceBackStep})
			}
			table.addByteStep(ch, next)
			table = next
		}
		i++
	}
	table.addByteStep(val[len(val)-1], newSmallTransition(nextField))

	return
}
