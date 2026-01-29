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
func makeWildCardFA(val []byte, pp printer) (start *smallTable, nextField *fieldMatcher) {
	state := &faState{table: newSmallTable()}
	start = state.table
	pp.labelTable(start, "WILDCARD")
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
	precomputeEpsilonClosures(start)
	return
}
