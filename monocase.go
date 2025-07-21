package quamina

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

func readMonocaseSpecial(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, err error) {
	t, err := pb.jd.Token()
	if err != nil {
		return
	}
	pathVals = valsIn

	monocaseString, ok := t.(string)
	if !ok {
		err = errors.New("value for 'prefix' must be a string")
		return
	}
	val := typedVal{
		vType: monocaseType,
		val:   `"` + monocaseString + `"`,
	}
	pathVals = append(pathVals, val)

	// has to be } or tokenizer will throw error
	_, err = pb.jd.Token()
	return
}

// makeMonocaseFA builds a FA to match "ignore-case" patterns. The Unicode Standard specifies algorithm 3.13,
// relying on the file CaseFolding.txt in the Unicode Character Database. This function uses the "Simple" flavor
// of casefolding, i.e. the lines in CaseFolding.txt that are marked with "C". The discussion in the Unicode
// standard doesn't mention this, but the algorithm essentially replaces upper-case characters with lower-case
// equivalents.
// We need to exercise caution to keep from creating states wastefully. For "CAT", after matching '"',
// you transition on either 'c' or 'C' but in this particular case you want to transition to the same
// next state. Note that there are many characters in Unicode where the upper and lower case forms are
// multi-byte and in fact not even the same number of bytes. So in that case you need two paths forward that step
// through the bytes of each form and then rejoin to arrive at a state. Also note
// that in many cases the upper/lower case versions of a rune have leading bytes in common
func makeMonocaseFA(val []byte, pp printer) (*smallTable, *fieldMatcher) {
	fm := newFieldMatcher()
	index := 0
	table := newSmallTable() // start state
	startTable := table
	var nextStep *faState
	for index < len(val) {
		var orig, alt []byte
		r, width := utf8.DecodeRune(val[index:])
		orig = val[index : index+width]
		altRune, ok := caseFoldingPairs[r]
		if ok {
			alt = make([]byte, utf8.RuneLen(altRune))
			utf8.EncodeRune(alt, altRune)
		}
		nextStep = &faState{table: newSmallTable()}
		pp.labelTable(nextStep.table, fmt.Sprintf("On %d, alt=%v", val[index], alt))
		if alt == nil {
			// easy case, no casefolding issues.  We should maybe try to coalesce these
			// no-casefolding sections and only call makeFAFragment once for all of them
			origFA := makeFAFragment(orig, nextStep, pp)
			table.addByteStep(orig[0], origFA)
		} else {
			// two paths to next state
			// but they might have a common prefix
			var commonPrefix int
			for commonPrefix = 0; orig[commonPrefix] == alt[commonPrefix]; commonPrefix++ {
				prefixStep := &faState{table: newSmallTable()}
				table.addByteStep(orig[commonPrefix], prefixStep)
				table = prefixStep.table
				pp.labelTable(table, fmt.Sprintf("common prologue on %x", orig[commonPrefix]))
			}
			// now build automata for the orig and alt versions of the char
			origFA := makeFAFragment(orig[commonPrefix:], nextStep, pp)
			altFA := makeFAFragment(alt[commonPrefix:], nextStep, pp)
			table.addByteStep(orig[commonPrefix], origFA)
			table.addByteStep(alt[commonPrefix], altFA)
		}
		table = nextStep.table
		index += width
	}
	lastState := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{fm}}
	nextStep.table.addByteStep(valueTerminator, lastState)
	return startTable, fm
}
