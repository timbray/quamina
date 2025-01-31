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

type regexpQuantifiedAtom struct {
	isDot    bool
	runes    RuneRange
	quantMin int
	quantMax int
	subtree  regexpRoot // if non-nil, ()-enclosed subtree here
}
type regexpBranch []*regexpQuantifiedAtom
type regexpRoot []regexpBranch

// makeRegexpNFA traverses the parsed regexp tree and generates a finite automaton
// that matches it. If forField is true, then the FA will have states that match " at the beginning
// and end.
func makeRegexpNFA(root regexpRoot, forField bool) (*smallTable, *fieldMatcher) {
	nextField := newFieldMatcher()
	nextStep := makeNFATrailer(nextField)
	if forField {
		table := makeSmallTable(nil, []byte{'"'}, []*faNext{nextStep})
		state := &faState{table: table}
		nextStep = &faNext{states: []*faState{state}}
	}
	// completely empty regexp
	if len(root) == 0 {
		return makeSmallTable(nil, []byte{'"'}, []*faNext{nextStep}), nextField
	}
	fa := newSmallTable()
	for _, branch := range root {
		var nextBranch *smallTable
		if len(branch) == 0 {
			nextBranch = makeSmallTable(nil, []byte{'"'}, []*faNext{nextStep})
		} else {
			nextBranch = makeOneRegexpBranchFA(branch, nextStep, forField)
		}
		fa = mergeFAs(fa, nextBranch, sharedNullPrinter)
	}
	return fa, nextField
}

// makeOneRegexpBranchFA - We know what the last step looks like, so we proceed back to
// front through the members of the branch, which are quantified atoms. Each can be a runeRange (which
// can be a single character or a dot or a subtree, in each case followed by a quantifier.
// We know the last step, which points at the nextField argument.
// There's a problem here. Quamina's match* methods feed string values including the enclosing "" to
// the automaton. This is useful for a variety of reasons. But that means if the regexp is a|b, because
// the | has the lowest precedence, it'd build an automaton that would match ("a)|(b"). So we need to
// just build the automaton on a|b and then manually fasten "-transitions in front of and behind it.
func makeOneRegexpBranchFA(branch regexpBranch, nextStep *faNext, forField bool) *smallTable {
	var step *faNext
	var table *smallTable

	// TODO: Assuming this works, rewrite a bunch of other make*NFA calls in this style, without recursion
	for index := len(branch) - 1; index >= 0; index-- {
		qa := branch[index]
		if qa.isDot {
			table = makeDotFA(nextStep)
			step = &faNext{states: []*faState{{table: table}}}
		} else if qa.subtree != nil {
			panic("Not supported " + rxfParenGroup)
		} else {
			// it's a rune range
			if qa.quantMin != 1 || qa.quantMax != 1 {
				panic("Not supported: quantifiers")
			}

			// just match a rune
			table = makeRuneRangeNFA(qa.runes, nextStep, sharedNullPrinter)
			step = &faNext{states: []*faState{{table: table}}}
		}
		nextStep = step
	}
	if forField {
		firstState := &faState{table: table}
		firstStep := &faNext{states: []*faState{firstState}}
		table = makeSmallTable(nil, []byte{'"'}, []*faNext{firstStep})
	}
	return table
}

// makeNFATrailer generates the last two steps in every NFA, because all field values end with the
// valueTerminator marker, so you need the field-matched state and you need another state that branches
// to it based on valueTerminator
// TODO: Prove that this is useful in other make*NFA scenarios
func makeNFATrailer(nextField *fieldMatcher) *faNext {
	matchState := &faState{
		table:            newSmallTable(),
		fieldTransitions: []*fieldMatcher{nextField},
	}
	matchStep := &faNext{[]*faState{matchState}}
	table := makeSmallTable(nil, []byte{valueTerminator}, []*faNext{matchStep})
	return &faNext{states: []*faState{{table: table}}}
}

// plan B

type runeTreeEntry struct {
	next  *faNext
	child runeTreeNode
}
type runeTreeNode []*runeTreeEntry

func addRuneTreeEntry(root runeTreeNode, r rune, dest *faNext) {
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
			state := &faState{table: table}
			unpacked[b] = &faNext{states: []*faState{state}}
		}
	}
	st := newSmallTable()
	st.pack(&unpacked)
	return st
}

func makeRuneRangeNFA(rr RuneRange, next *faNext, pp printer) *smallTable {
	pp.labelTable(next.states[0].table, "Next")

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

func makeDotFA(dest *faNext) *smallTable {
	sLast := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faNext{nil, dest, nil},
	}
	targetLast := &faNext{states: []*faState{{table: sLast}}}
	sLastInter := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faNext{nil, targetLast, nil},
	}
	targetLastInter := &faNext{states: []*faState{{table: sLastInter}}}
	sFirstInter := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faNext{nil, targetLastInter, nil},
	}
	targetFirstInter := &faNext{states: []*faState{{table: sFirstInter}}}

	sE0 := &smallTable{
		ceilings: []byte{0xa0, 0xc0, byte(byteCeiling)},
		steps:    []*faNext{nil, targetLast, nil},
	}
	targetE0 := &faNext{states: []*faState{{table: sE0}}}

	sED := &smallTable{
		ceilings: []byte{0x80, 0xA0, byte(byteCeiling)},
		steps:    []*faNext{nil, targetLast, nil},
	}
	targetED := &faNext{states: []*faState{{table: sED}}}

	sF0 := &smallTable{
		ceilings: []byte{0x90, 0xC0, byte(byteCeiling)},
		steps:    []*faNext{nil, targetLastInter, nil},
	}
	targetF0 := &faNext{states: []*faState{{table: sF0}}}

	sF4 := &smallTable{
		ceilings: []byte{0x80, 0x90, byte(byteCeiling)},
		steps:    []*faNext{nil, targetLastInter, nil},
	}
	targetF4 := &faNext{states: []*faState{{table: sF4}}}

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
		steps: []*faNext{
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
