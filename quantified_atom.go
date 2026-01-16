package quamina

import "fmt"

// represents the "atom [ quantifier ]" piece of the regexp grammar. Is kind of messy but
// faithful to the kind-of-messy regexp semantics. I blame Kleene.
type quantifiedAtom struct {
	runes           RuneRange
	dotRunes        bool       // true if atom is "."
	bigRuneRangeKey string     // for the huge character_properties RuneRanges
	quantMin        int        // 0 means ? or *
	quantMax        int        // the value regexpQuantifierMax means + or *, no max
	subtree         regexpRoot // if non-nil, ()-enclosed subtree here
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

func (qa *quantifiedAtom) runeRangeCache() string {
	return qa.bigRuneRangeKey
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
	case qa.runeRangeCache() != "":
		table = makeAndCacheRuneRangeFA(qa.runes, nextStep, qa.runeRangeCache(), pp)
	default:
		// if it's none of these other things, it has to boil down to a rune range
		table = makeRuneRangeNFA(qa.runes, nextStep, pp)
		pp.labelTable(table, fmt.Sprintf("RR %x/%x, %d-%d", qa.runes[0].Lo, qa.runes[0].Hi, qa.quantMin, qa.quantMax))
	}
	return table
}
