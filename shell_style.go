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
func makeShellStyleFA(val []byte, printer printer) (start *smallTable, nextField *fieldMatcher) {
	table := newSmallTable()
	start = table
	nextField = newFieldMatcher()

	// for each byte in the pattern
	valIndex := 0
	for valIndex < len(val) {
		ch := val[valIndex]
		if ch == '*' {
			// special-case handling for string ending in '*"' - transition to field match on any character.
			// we know the trailing '"' will be there because of JSON syntax.  We could use an epsilon state
			// but then the matcher will process through all the rest of the bytes, when it doesn't need to
			if valIndex == len(val)-2 {
				step := &faState{
					table:            newSmallTable(),
					fieldTransitions: []*fieldMatcher{nextField},
				}
				table.epsilon = []*faState{step}
				printer.labelTable(table, fmt.Sprintf("prefix escape at %d", valIndex))
				return
			}
			globStep := &faState{table: table}
			printer.labelTable(table, fmt.Sprintf("gS at %d", valIndex))
			table.epsilon = []*faState{globStep}

			valIndex++
			globNext := &faState{table: newSmallTable()}
			printer.labelTable(globNext.table, fmt.Sprintf("gX on %c at %d", val[valIndex], valIndex))
			table.addByteStep(val[valIndex], &faNext{states: []*faState{globNext}})
			table = globNext.table
		} else {
			nextStep := &faState{table: newSmallTable()}
			printer.labelTable(nextStep.table, fmt.Sprintf("on %c at %d", val[valIndex], valIndex))
			table.addByteStep(ch, &faNext{states: []*faState{nextStep}})
			table = nextStep.table
		}
		valIndex++
	}
	lastStep := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{nextField}}
	printer.labelTable(lastStep.table, fmt.Sprintf("last step at %d", valIndex))
	table.addByteStep(valueTerminator, &faNext{states: []*faState{lastStep}})
	return
}
