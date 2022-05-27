package core

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
			err = fmt.Errorf("invalid character %v in 'shellstyle' pattern", tt)
		}
	default:
		err = errors.New("trailing garbage in shellstyle pattern")
	}

	return
}

// makeShellStyleAutomaton - recognize a "-delimited string containing one '*' glob.
// TODO: Make this recursive like makeStringAutomaton
func makeShellStyleAutomaton(val []byte, useThisTransition *fieldMatcher) (start *smallTable[*nfaStepList], nextField *fieldMatcher) {
	table := newSmallTable[*nfaStepList]()
	start = table
	if useThisTransition != nil {
		nextField = useThisTransition
	} else {
		nextField = newFieldMatcher()
	}
	lister := newListMaker()

	// for each byte in the pattern
	var globStep *nfaStep = nil
	var globExitStep *nfaStep = nil
	var globExitByte byte
	i := 0
	for i < len(val) {
		ch := val[i]
		if ch == '*' {
			// special-case handling for string ending in '*"' - transition to field match on any character.
			//  we know the trailing '"' will be there because of JSON syntax.
			// TODO: This doesn't even need to be an NFA
			if i == len(val)-2 {
				step := &nfaStep{table: newSmallTable[*nfaStepList](), fieldTransitions: []*fieldMatcher{nextField}}
				list := lister.getList(step)
				table.addRangeSteps(0, byteCeiling, list)
				return
			}

			// loop back on everything
			globStep = &nfaStep{table: table}
			table.addRangeSteps(0, byteCeiling, lister.getList(globStep))

			// escape the glob on the next char from the pattern - remember the byte and the state escaped to
			i++
			globExitByte = val[i]
			globExitStep = &nfaStep{table: newSmallTable[*nfaStepList]()}
			// escape the glob
			table.addByteStep(globExitByte, lister.getList(globExitStep))
			table = globExitStep.table
		} else {
			nextStep := &nfaStep{table: newSmallTable[*nfaStepList]()}

			// we're going to move forward on 'ch'.  On anything else, we leave it at nil or - if we've passed
			//  a glob, loop back to the glob stae.  if 'ch' is also the glob exit byte, also put in a transfer
			//  back to the glob exist state
			if globExitStep != nil {
				table.addRangeSteps(0, byteCeiling, lister.getList(globStep))
				if ch == globExitByte {
					table.addByteStep(ch, lister.getList(globExitStep, nextStep))
				} else {
					table.addByteStep(globExitByte, lister.getList(globExitStep))
					table.addByteStep(ch, lister.getList(nextStep))
				}
			} else {
				table.addByteStep(ch, lister.getList(nextStep))
			}
			table = nextStep.table
		}
		i++
	}

	lastStep := &nfaStep{table: newSmallTable[*nfaStepList](), fieldTransitions: []*fieldMatcher{nextField}}
	if globExitStep != nil {
		table.addRangeSteps(0, byteCeiling, lister.getList(globStep))
		table.addByteStep(globExitByte, lister.getList(globExitStep))
		table.addByteStep(valueTerminator, lister.getList(lastStep))
	} else {
		table.addByteStep(valueTerminator, lister.getList(lastStep))
	}
	return
}
