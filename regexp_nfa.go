package quamina

import (
	"errors"
	"fmt"
	"sort"
	"unicode/utf8"
)

// these types exported to facilitate building Unicode tables in code_gen
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

func makeRegexpNFA(root regexpRoot) (*smallTable, *fieldMatcher) {
	nextField := newFieldMatcher()
	fa := newSmallTable()
	for _, branch := range root {
		nextBranch := makeOneRegexpBranchFA(branch, nextField)
		fa = mergeFAs(fa, nextBranch, sharedNullPrinter)
	}
	return fa, nextField
}

// makeOneRegexpBranchFA - exploring… we know what the last step looks like, so we proceed back to
// front through the members of the branch, which are quantified atoms. Each can be a runeRange (which
// can be a single character or a dot or a subtree, in each case followed by a quantifier.
// We know the last step, which points at the nextField argument.
func makeOneRegexpBranchFA(branch regexpBranch, nextField *fieldMatcher) *smallTable {
	nextStep := makeNFATrailer(nextField)
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
			if len(qa.runes) != 1 || qa.quantMin != 1 || qa.quantMax != 1 {
				panic("Not supported: quantifiers")
			}

			// just match a rune
			u, _ := runeToUTF8(qa.runes[0].Lo)
			trailer := makeFAFragment(u, nextStep, sharedNullPrinter)
			table = makeSmallTable(nil, []byte{u[0]}, []*faNext{trailer})
			step = &faNext{states: []*faState{{table: table}}}
		}
		nextStep = step
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

func makeRuneRangeNFA(rr RuneRange, next *faState, pp *prettyPrinter) (*smallTable, error) {
	// these have to be in increasing order to work
	sort.Slice(rr, func(i, j int) bool { return rr[i].Lo < rr[j].Lo })

	// turn the slice of hi/lo inclusive endpoints into a slice of utf8 encodings
	var utf8Range [][]byte
	ri, err := newRuneRangeIterator(rr)
	if err != nil {
		return nil, err
	}
	// for each rune
	for r := ri.next(); r != -1; r = ri.next() {
		buf, err := runeToUTF8(r)
		if err != nil {
			continue
		}
		utf8Range = append(utf8Range, buf)
	}
	pp.labelTable(next.table, "DESTINATION")
	step := &faNext{[]*faState{next}}
	table := newSmallTable()
	pp.labelTable(table, "ROOT")
	makeRuneRangeNFALevel(utf8Range, 0, table, step, pp)
	return table, nil
}

type runeSubrange struct {
	lo int
	hi int
}

// makeRuneRangeNFALevel fills in the transitions in the 'table' argument based on the bytes in all the utf8Range
// byte slices
func makeRuneRangeNFALevel(utf8Range [][]byte, level int, targetTable *smallTable, next *faNext, pp *prettyPrinter) {
	unpacked := unpackTable(targetTable)
	nextLevelSubrange := make(map[byte]*runeSubrange)
	nextLevelSteps := make(map[byte]*faNext)

	var lastByte byte = 0xff
	var subrange *runeSubrange
	for index, u := range utf8Range {
		if level >= len(u) {
			continue
		}
		b := u[level]
		if b != lastByte {
			subrange = &runeSubrange{lo: index, hi: index}
			nextLevelSubrange[b] = subrange
			lastByte = b
		} else {
			subrange.hi = index
		}

		if len(u) > (level + 1) {
			nextStep, ok := nextLevelSteps[b]
			if !ok {
				table := newSmallTable()
				nextState := &faState{table: table}
				asRune, _ := utf8.DecodeRune(u)
				pp.labelTable(table, fmt.Sprintf("For level %d in %x", level+1, asRune))
				nextStep = &faNext{[]*faState{nextState}}
				nextLevelSteps[b] = nextStep
			}
			unpacked[b] = nextStep
		} else {
			unpacked[b] = next
		}
	}
	for b, nextStep := range nextLevelSteps {
		subrange = nextLevelSubrange[b]
		makeRuneRangeNFALevel(utf8Range[subrange.lo:subrange.hi+1], level+1, nextStep.states[0].table, next, pp)
	}

	targetTable.pack(unpacked)
}

func makeDotFA(dest *faNext) *smallTable {
	if dest == nil {
		dest = &faNext{}
	}
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
