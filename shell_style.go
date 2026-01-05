package quamina

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	if strings.Contains(shellString, "**") {
		err = fmt.Errorf("adjacent '*' characters not allowed")
		return
	}

	pathVals = append(pathVals, typedVal{vType: shellStyleType, val: `"` + shellString + `"`})

	t, err = pb.jd.Token()
	if err != nil {
		return
	}
	switch t.(type) {
	case json.Delim:
		// } is all that will be returned
	default:
		err = errors.New("trailing garbage in shellstyle pattern")
	}

	return
}

// makeShellStyleFA does what it says.  It is precisely equivalent to a regex with the only operator
// being a single ".*". Once we've implemented regular expressions we can use that to more or less eliminate this
func makeShellStyleFA(val []byte, pp printer) (start *smallTable, nextField *fieldMatcher) {
	state := &faState{table: newSmallTable()}
	start = state.table
	pp.labelTable(start, "SHELLSTYLE")
	nextField = newFieldMatcher()

	// for each byte in the pattern
	valIndex := 0
	for valIndex < len(val) {
		ch := val[valIndex]
		if ch == '*' {
			spinner := state
			spinner.isSpinner = true

			valIndex++
			spinEscape := &faState{table: newSmallTable()}
			spinEscape.table.epsilons = []*faState{spinner}
			spinner.table = makeByteDotFA(spinner, pp)
			spinner.table.addByteStep(val[valIndex], spinEscape)
			pp.labelTable(spinner.table, "*-Spinner")
			pp.labelTable(spinEscape.table, fmt.Sprintf("spinEscape on %c at %d", val[valIndex], valIndex))
			state = spinEscape
		} else {
			nextStep := &faState{table: newSmallTable()}
			pp.labelTable(nextStep.table, fmt.Sprintf("on %c at %d", val[valIndex], valIndex))
			state.table.addByteStep(ch, nextStep)
			state = nextStep
		}
		valIndex++
	}
	lastStep := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{nextField}}
	pp.labelTable(lastStep.table, fmt.Sprintf("last step at %d", valIndex))
	state.table.addByteStep(valueTerminator, lastStep)
	return
}
