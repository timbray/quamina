package quamina

import (
	"errors"
	"fmt"
	"sort"
)

// RunePair and related types exported to facilitate building Unicode tables in code_gen
type RunePair struct {
	Lo, Hi rune
}
type RuneRange []RunePair

type runeRangeIterator struct {
	pairs     RuneRange
	whichPair int
	inPair    rune
}

const runeMax = 0x10ffff

func newRuneRangeIterator(rr RuneRange) (*runeRangeIterator, error) {
	if len(rr) == 0 {
		return nil, errors.New("empty range")
	}
	sort.Slice(rr, func(i, j int) bool { return rr[i].Lo < rr[j].Lo })
	return &runeRangeIterator{pairs: rr, whichPair: 0, inPair: rr[0].Lo}, nil
}

// In the regular expressions represented by the I-Regexp syntax, the | connector has the lowest
// precedence, so at the top level, it's a slice of what the ABNF calls branches - generate an NFA
// for each branch and then take their union.
// Inside a branch, the structure is obviously recursive because of the ()-group, which itself can
// carry a slice of branches etc.  Aside from that, the branch contains a sequence of atom/quantifier
// pairs.  All the "atom" syntax options describe ranges of characters and are well-represented by
// the RuneRange type. This includes [] and \[pP]{whatever}.
// All the forms of quantifiers can be described by pairs of numbers. ? is [0,1]. + is [1,♾️]. * is [0,♾️].
// {m,n} ranges also, obviously.

type regexpBranch []*quantifiedAtom
type regexpRoot []regexpBranch

// makeRegexpNFA traverses the parsed regexp tree and generates a finite automaton
// that matches it. If forField is true, then the FA will have states that match " at the beginning
// and end.
func makeRegexpNFA(root regexpRoot, forField bool, pp printer) (*smallTable, *fieldMatcher) {
	nextField := newFieldMatcher()
	nextStep := makeNFATrailer(nextField)
	pp.labelTable(nextStep.table, "Trailer")
	if forField {
		table := makeSmallTable(nil, []byte{'"'}, []*faState{nextStep})
		pp.labelTable(table, "</Field>")
		nextStep = &faState{table: table}
	}
	return makeNFAFromBranches(root, nextStep, forField, pp), nextField
}
func makeNFAFromBranches(root regexpRoot, nextStep *faState, forField bool, pp printer) *smallTable {
	// completely empty regexp
	if len(root) == 0 {
		return makeSmallTable(nil, []byte{'"'}, []*faState{nextStep})
	}
	var fa *smallTable
	for _, branch := range root {
		var nextBranch *smallTable
		if len(branch) == 0 {
			nextBranch = makeSmallTable(nil, []byte{'"'}, []*faState{nextStep})
			pp.labelTable(nextBranch, "next on len 0")
		} else {
			nextBranch = makeOneRegexpBranchFA(branch, nextStep, forField, pp)
		}
		if fa != nil {
			fa = mergeFAs(fa, nextBranch, pp)
		} else {
			fa = nextBranch
		}
	}
	return fa
}

// makeOneRegexpBranchFA - We know what the last step looks like, so we proceed back to
// front through the members of the branch, which are quantified atoms. Each can be a runeRange (which
// can be a single character or a dot or a subtree, in each case followed by a quantifier.
// We know the last step, which points at the nextField argument.
// There's a problem here. Quamina's match* methods feed string values including the enclosing "" to
// the automaton. This is useful for a variety of reasons. But that means if the regexp is a|b, because
// the | has the lowest precedence, it'd build an automaton that would match ("a)|(b"). So we need to
// just build the automaton on a|b and then manually fasten "-transitions in front of and behind it.
func makeOneRegexpBranchFA(branch regexpBranch, nextStep *faState, forField bool, pp printer) *smallTable {
	var step *faState
	var table *smallTable

	// When you're computing the FA for one piece of a regexp, it's useful to know which faState is the
	// next step. You can do this by recursing to compute the next step and yield the target, but this
	// could lead to very deep recursion on long regexps. So instead, the code works through the atoms
	// in a branch back to front. After you've built the FA for the Nth step, you know the target for
	// the N-1th step.  Since the nextStep argument to the function gives you the target of the last
	// step, this should work well.  As I write this comment, Quamina's FA-constructions use a
	// variety of approaches but I think I'd like to converge on this one.
	for index := len(branch) - 1; index >= 0; index-- {
		// The following implements the Thompson construction as described in
		// https://swtch.com/%7Ersc/regexp/regexp1.html
		qa := branch[index]
		plusLoopback := &faState{} // unnecessary but shuts up annoying JetBrains warning
		var starNext *faState
		step = &faState{}

		// setting the stage for Thompson construction before making this step's FA
		if qa.isPlus() {
			// the + construction requires a loopback state in front of the state table
			plusLoopback = &faState{table: newSmallTable()}
			plusLoopback.table.epsilons = []*faState{nextStep}
			pp.labelTable(plusLoopback.table, "PlusLoopback")
			nextStep = plusLoopback
		} else if qa.isStar() {
			// the * construction requires that the generated FA points back to itself
			starNext = nextStep
			nextStep = step
		}

		// now we make the step FA
		table = qa.makeFA(nextStep, pp)
		if table == nil {
			panic("makeFA returned nil")
		}
		step.table = table

		// now more epsilon shuffling to complete quantification construction
		switch {
		case qa.isSingleton():
			// we're good
		case qa.isPlus():
			// for the + case, need to loop back to the newly created state
			plusLoopback.table.epsilons = append(plusLoopback.table.epsilons, step)
		case qa.isQM():
			// for the ? case, forward epsilon
			table.epsilons = append(table.epsilons, nextStep)
		case qa.isStar():
			// the * construct requires an epsilon transition forward
			table.epsilons = append(step.table.epsilons, starNext)
		case qa.isPlus():
			// +, no-op, but avoids default case
		default:
			panic("unhandled quantification case")
		}

		nextStep = step
	}
	if forField {
		firstState := &faState{table: table}
		table = makeSmallTable(nil, []byte{'"'}, []*faState{firstState})
		pp.labelTable(table, "<Field>")
	}
	return table
}

// makeNFATrailer generates the last two steps in every NFA, because all field values end with the
// valueTerminator marker, so you need the field-matched state and you need another state that branches
// to it based on valueTerminator
// TODO: Prove that this is useful in other make*NFA scenarios
func makeNFATrailer(nextField *fieldMatcher) *faState {
	matchState := &faState{
		table:            newSmallTable(),
		fieldTransitions: []*fieldMatcher{nextField},
	}
	table := makeSmallTable(nil, []byte{valueTerminator}, []*faState{matchState})
	return &faState{table: table}
}

// plan B

type runeTreeEntry struct {
	next  *faState
	child runeTreeNode
}
type runeTreeNode []*runeTreeEntry

func addRuneTreeEntry(root runeTreeNode, r rune, dest *faState) {
	// this works because no UTF-8 representation of a code point can be a prefix of any other
	node := root
	bytes, err := runeToUTF8(r)
	// Invalid bytes should be caught at another level, but if they show up here, silently ignore
	if err != nil {
		return
	}

	// find or make entry
	for i, b := range bytes {
		if node[b] != nil {
			node = node[b].child
			continue
		}
		// need to make a new node
		entry := &runeTreeEntry{}
		node[b] = entry
		if i == len(bytes)-1 {
			entry.next = dest
		} else {
			entry.child = make([]*runeTreeEntry, byteCeiling)
		}
		node = entry.child
	}
}

func nfaFromRuneTree(root runeTreeNode, pp printer) *smallTable {
	return tableFromRuneTreeNode(root, pp)
}

func tableFromRuneTreeNode(node runeTreeNode, pp printer) *smallTable {
	var unpacked unpackedTable
	for b, entry := range node {
		if entry == nil {
			continue
		}
		if entry.next != nil {
			unpacked[b] = entry.next
		} else {
			table := tableFromRuneTreeNode(entry.child, pp)
			pp.labelTable(table, fmt.Sprintf("on %x", b))
			unpacked[b] = &faState{table: table}
		}
	}
	st := newSmallTable()
	st.pack(&unpacked)
	return st
}

func makeRuneRangeNFA(rr RuneRange, next *faState, pp printer) *smallTable {
	pp.labelTable(next.table, "Next")

	// turn the slice of hi/lo inclusive endpoints into a slice of utf8 encodings
	ri, err := newRuneRangeIterator(rr)

	// can't happen I think
	if err != nil {
		panic("Invalid rune range")
	}

	var root runeTreeNode = make([]*runeTreeEntry, byteCeiling)

	// for each rune
	for r := ri.next(); r != -1; r = ri.next() {
		addRuneTreeEntry(root, r, next)
	}
	return nfaFromRuneTree(root, pp)
}

func makeByteDotFA(dest *faState, pp printer) *smallTable {
	ceilings := []byte{0xC0, 0xC2, 0xF5, 0xF6}
	steps := []*faState{dest, nil, dest, nil}
	t := &smallTable{ceilings: ceilings, steps: steps}
	pp.labelTable(t, " · ")
	return t
}

func makeDotFA(dest *faState) *smallTable {
	sLast := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faState{nil, dest, nil},
	}
	targetLast := &faState{table: sLast}
	sLastInter := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLast, nil},
	}
	targetLastInter := &faState{table: sLastInter}
	sFirstInter := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLastInter, nil},
	}
	targetFirstInter := &faState{table: sFirstInter}

	sE0 := &smallTable{
		ceilings: []byte{0xa0, 0xc0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLast, nil},
	}
	targetE0 := &faState{table: sE0}

	sED := &smallTable{
		ceilings: []byte{0x80, 0xA0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLast, nil},
	}
	targetED := &faState{table: sED}

	sF0 := &smallTable{
		ceilings: []byte{0x90, 0xC0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLastInter, nil},
	}
	targetF0 := &faState{table: sF0}

	sF4 := &smallTable{
		ceilings: []byte{0x80, 0x90, byte(byteCeiling)},
		steps:    []*faState{nil, targetLastInter, nil},
	}
	targetF4 := &faState{table: sF4}

	// for reference, see https://www.tbray.org/ongoing/When/202x/2024/12/29/Matching-Dot-Redux
	return &smallTable{
		ceilings: []byte{
			0x80,              // 0
			0xC2,              // 1
			0xE0,              // 2
			0xE1,              // 3
			0xED,              // 4
			0xEE,              // 5
			0xF0,              // 6
			0xF1,              // 7
			0xF4,              // 8
			0xF5,              // 9
			byte(byteCeiling), // 10
		},
		steps: []*faState{
			dest,             // 0
			nil,              // 1
			targetLast,       // 2
			targetE0,         // 3
			targetLastInter,  // 4
			targetED,         // 5
			targetLastInter,  // 6
			targetF0,         // 7
			targetFirstInter, // 8
			targetF4,         // 9
			nil,              // 10
		},
	}
}

func (i *runeRangeIterator) next() rune {
	if i.inPair <= i.pairs[i.whichPair].Hi {
		r := i.inPair
		i.inPair++
		return r
	}
	// will blow up on empty pair, could put a check in, or just don't generate them
	// while parsing regexp
	i.whichPair++
	if i.whichPair == len(i.pairs) {
		return -1
	}
	r := i.pairs[i.whichPair].Lo
	i.inPair = r + 1
	return r
}
