package quamina

import "fmt"

// TODO: This is messy. Clean it up, dotRunes and subtree!=nil are a code smell, and
// so is putting magic values in the quantMin/Max.
type quantifiedAtom struct {
	dotRunes bool
	runes    RuneRange
	quantMin int
	quantMax int
	subtree  regexpRoot // if non-nil, ()-enclosed subtree here
}

func (qa *quantifiedAtom) getSubtree() regexpRoot {
	return qa.subtree
}

func (qa *quantifiedAtom) isSingleton() bool {
	return qa.quantMin == 1 && qa.quantMax == 1
}

func (qa *quantifiedAtom) isDot() bool {
	return qa.dotRunes
}

func (qa *quantifiedAtom) isQM() bool {
	return qa.quantMin == 0 && qa.quantMax == 1
}

func (qa *quantifiedAtom) isPlus() bool {
	return qa.quantMin == 1 && qa.quantMax == regexpQuantifierMax
}

func (qa *quantifiedAtom) isStar() bool {
	return qa.quantMin == 0 && qa.quantMax == regexpQuantifierMax
}

func (qa *quantifiedAtom) makeFA(nextStep *faState, pp printer) *smallTable {
	var table *smallTable
	switch {
	case qa.isDot():
		table = makeDotFA(nextStep)
		pp.labelTable(table, "Dot")
	case qa.getSubtree() != nil:
		table = makeNFAFromBranches(qa.getSubtree(), nextStep, false, pp)
	default:
		// if it's not a subtree, it has to boil down to a rune range
		// we're not doing ranges yet, so for now we'll only directly index one character at a time
		table = makeRuneRangeNFA(qa.runes, nextStep, sharedNullPrinter)
		pp.labelTable(table, fmt.Sprintf("RR %x/%x, %d-%d", qa.runes[0].Lo, qa.runes[0].Hi, qa.quantMin, qa.quantMax))
	}
	return table
}
