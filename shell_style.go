package quamina

import (
	"encoding/json"
	"errors"
	"fmt"
)

// readShellStyleSpecial parses a shellStyle object in a Pattern
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
func makeShellStyleAutomaton(val []byte, printer printer) (start *smallTable, nextField *fieldMatcher) {
	table := newSmallTable()
	start = table
	nextField = newFieldMatcher()

	// for each byte in the pattern
	var globStep *faState = nil
	var globExitStep *faState = nil
	var globExitByte byte
	i := 0
	for i < len(val) {
		ch := val[i]
		if ch == '*' {
			// special-case handling for string ending in '*"' - transition to field match on any character.
			//  we know the trailing '"' will be there because of JSON syntax.
			if i == len(val)-2 {
				step := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{nextField}}
				table.setDefault(&faNext{steps: []*faState{step}})
				printer.labelTable(table, fmt.Sprintf("prefix escape at %d", i))
				return
			}

			// loop back on everything
			globStep = &faState{table: table}
			printer.labelTable(table, fmt.Sprintf("gS at %d", i))
			table.setDefault(&faNext{steps: []*faState{globStep}})

			// escape the glob on the next char from the pattern - remember the byte and the state escaped to
			i++
			globExitByte = val[i]
			globExitStep = &faState{table: newSmallTable()}
			printer.labelTable(globExitStep.table, fmt.Sprintf("gX on %c at %d", val[i], i))
			// escape the glob
			table.addByteStep(globExitByte, &faNext{steps: []*faState{globExitStep}})
			table = globExitStep.table
		} else {
			nextStep := &faState{table: newSmallTable()}
			printer.labelTable(nextStep.table, fmt.Sprintf("on %c at %d", val[i], i))

			// we're going to move forward on 'ch'.  On anything else, we leave it at nil or - if we've passed
			//  a glob, loop back to the glob stae.  if 'ch' is also the glob exit byte, also put in a transfer
			//  back to the glob exist state
			if globExitStep != nil {
				table.setDefault(&faNext{steps: []*faState{globStep}})
				if ch == globExitByte {
					table.addByteStep(ch, &faNext{steps: []*faState{globExitStep, nextStep}})
				} else {
					table.addByteStep(globExitByte, &faNext{steps: []*faState{globExitStep}})
					table.addByteStep(ch, &faNext{steps: []*faState{nextStep}})
				}
			} else {
				table.addByteStep(ch, &faNext{steps: []*faState{nextStep}})
			}
			table = nextStep.table
		}
		i++
	}

	lastStep := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{nextField}}
	printer.labelTable(lastStep.table, fmt.Sprintf("last step at %d", i))
	if globExitStep != nil {
		table.setDefault(&faNext{steps: []*faState{globStep}})
		table.addByteStep(globExitByte, &faNext{steps: []*faState{globExitStep}})
		table.addByteStep(valueTerminator, &faNext{steps: []*faState{lastStep}})
	} else {
		table.addByteStep(valueTerminator, &faNext{steps: []*faState{lastStep}})
	}
	// fmt.Printf("new for [%s]: %s\n", string(val), printer.printNFA(start))
	return
}
