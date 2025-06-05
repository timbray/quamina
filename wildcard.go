package quamina

import (
	"encoding/json"
	"errors"
	"fmt"
)

type wcState int

const (
	wcChilling wcState = iota
	wcAfterBS
	wcAfterGlob
)

func readWildcardSpecial(pb *patternBuild, valsIn []typedVal) ([]typedVal, error) {
	t, err := pb.jd.Token()
	if err != nil {
		return nil, err
	}
	pathVals := valsIn
	wcInput, ok := t.(string)
	if !ok {
		return nil, errors.New("value for `wildcard` must be a string")
	}
	inBytes := []byte(wcInput)
	state := wcChilling
	for i, b := range inBytes {
		switch state {
		case wcChilling:
			switch b {
			case '\\':
				if i == len(inBytes)-1 {
					return nil, errors.New("'\\' at end of string not allowed")
				}
				state = wcAfterBS
			case '*':
				state = wcAfterGlob
			}
		case wcAfterBS:
			switch b {
			case '\\', '*':
				state = wcChilling
			default:
				return nil, errors.New("`\\` can only be followed by '\\' or '*'")
			}
		case wcAfterGlob:
			switch b {
			case '*':
				return nil, fmt.Errorf("adjacent '*' characters not allowed")
			case '\\':
				state = wcAfterBS
			default:
				state = wcChilling
			}
		}
	}
	pathVals = append(pathVals, typedVal{vType: wildcardType, val: `"` + wcInput + `"`})

	t, err = pb.jd.Token()
	if err != nil {
		return nil, err
	}
	switch t.(type) {
	case json.Delim:
		// } is all that will be returned
	default:
		return nil, errors.New("trailing garbage in wildcard pattern")
	}

	return pathVals, nil
}

// makeWildcardFA is a replacement for shellstyle patterns, the only difference being that escaping is
// provided for * and \.
func makeWildcardFA(val []byte, printer printer) (start *smallTable, nextField *fieldMatcher) {
	table := newSmallTable()
	start = table
	nextField = newFieldMatcher()

	// for each byte in the pattern. \-escape processing is simplified because illegal constructs such as \a and \
	// at the end of the value have been rejected by readWildcardSpecial.
	valIndex := 0
	for valIndex < len(val) {
		ch := val[valIndex]
		escaped := ch == '\\'
		if escaped {
			valIndex++
			ch = val[valIndex]
		}
		if ch == '*' && !escaped {
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
			// ** is forbidden, if we're seeing *\* then the second * is non-magic, if we're seeing *\\, it
			// just means \, so either way, all we need to do is hop over this \
			if val[valIndex] == '\\' {
				valIndex++
			}
			globNext := &faState{table: newSmallTable()}
			printer.labelTable(globNext.table, fmt.Sprintf("gX on %c at %d", val[valIndex], valIndex))
			table.addByteStep(val[valIndex], globNext)
			table = globNext.table
		} else {
			nextStep := &faState{table: newSmallTable()}
			printer.labelTable(nextStep.table, fmt.Sprintf("on %c at %d", val[valIndex], valIndex))
			table.addByteStep(ch, nextStep)
			table = nextStep.table
		}
		valIndex++
	}
	lastStep := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{nextField}}
	printer.labelTable(lastStep.table, fmt.Sprintf("last step at %d", valIndex))
	table.addByteStep(valueTerminator, lastStep)
	return
}
