package quamina

import (
	"errors"
	"fmt"
	"slices"
	"sort"
)

// It turns out that makeRuneRangeNFA is an expensive call that burns memory.  So for the big
// RuneRange FAs, we compute and cache shells. A shell is an FA computed with the "next" value
// being PlaceholderState.  When you need a rune range FA, you take the shell and build a copy,
// replacing transitions to PlaceholderState by whatever the "next" value is.
// Note that FAs are only built during AddPattern calls and this is single-threaded, se we
// can safely build and update the cachedRRFaShells

var PlaceholderState *faState = &faState{table: newSmallTable()}
var cachedFaShells = make(map[string]*smallTable)

func faFromShell(shell *smallTable, oldNext *faState, newNext *faState) *smallTable {
	return copyShellNode(&faState{table: shell}, oldNext, newNext).table
}
func copyShellNode(shell *faState, oldNext *faState, newNext *faState) *faState {
	table := &smallTable{
		ceilings: slices.Clone(shell.table.ceilings),
		steps:    make([]*faState, len(shell.table.steps)),
	}
	for i, step := range shell.table.steps {
		switch step {
		case nil:
			// no-op
		case oldNext:
			table.steps[i] = newNext
		default:
			table.steps[i] = copyShellNode(step, oldNext, newNext)
		}
	}
	return &faState{table: table}
}

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

const runeMax = 0x10ffff

func newRuneRangeIterator(rr RuneRange) (*runeRangeIterator, error) {
	if len(rr) == 0 {
		return nil, errors.New("empty range")
	}
	sort.Slice(rr, func(i, j int) bool { return rr[i].Lo < rr[j].Lo })
	return &runeRangeIterator{pairs: rr, whichPair: 0, inPair: rr[0].Lo}, nil
}

// here's the problem: A construct like [~p{L}~p[Nd}~p{Zs}] is going to be brutally expensive, because
// it'll have to build the FA to match the combination of all those huge rune-ranges.

func makeAndCacheRuneRangeFA(rr RuneRange, next *faState, name string, pp printer) *smallTable {
	if name != "" {
		fa, ok := cachedFaShells[name]
		if !ok {
			fa = makeAndCacheRuneRangeFA(rr, PlaceholderState, "", pp)
			cachedFaShells[name] = fa
		}
		return faFromShell(fa, PlaceholderState, next)
	}

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

func makeRuneRangeNFA(rr RuneRange, next *faState, pp printer) *smallTable {
	return makeAndCacheRuneRangeFA(rr, next, "", pp)
}

func InvertRuneRange(rr RuneRange) RuneRange {
	sort.Slice(rr, func(i, j int) bool {
		return rr[i].Lo < rr[j].Lo
	})
	var inverted RuneRange
	var point rune = 0
	for _, pair := range rr {
		if pair.Lo > point {
			inverted = append(inverted, RunePair{point, pair.Lo - 1})
		}
		point = pair.Hi + 1
	}
	if point < runeMax {
		inverted = append(inverted, RunePair{point, runeMax})
	}
	return inverted
}

type runeTreeEntry struct {
	next  *faState
	child runeTreeNode
}
type runeTreeNode []*runeTreeEntry

// This burns memory like crazy, we build a 246-entry x 64-bit table for
// each smallTable-to-be, which makes it slow in dealing with things like
// ~p{L}. TODO: Here are ideas:
//  1. For things like ~{L}, build the FA with a distinguished *faState "dest"
//     value, then recursively copy all the faStates and smallTablss but replace
//     the distinguished pointer with the real "next" value.
//  2. Don't use a dumbass make([]*runeTreeEntry, byteCeiling) slice, but
//     rather a list of byte/pointer pairs. Way less memory.
//  3. Use a slightly less dumbass [byteCeiling]*faState and ideally in such a way
//     that it comes off the stack.
//     Hmm, that tree could get pretty huge, every new level brings in another power
//     of 246.
// As of 2026/01, #1 above has been implemented with a cache, see cachedFaShells.
// The "skinny" stuff below is an attempt at #2, but runs much slower than the memory burner. TODO: Investigate further

// only "next" or "node" is provided
type skinnyRuneTreeEntry struct {
	next *faState
	node *skinnyRuneTreeNode
}
type skinnyRuneTreeNode struct {
	byteVals []byte
	entries  []*skinnyRuneTreeEntry
}

func addSkinnyRuneTreeEntry(root *skinnyRuneTreeNode, r rune, dest *faState) {
	node := root
	runeBytes, err := runeToUTF8(r)
	// Invalid bytes should be caught at another level, but if they show up here, silently ignore
	if err != nil {
		return
	}
	// find or make entry
	for runeByteIndex, runeByte := range runeBytes {
		var nextEntry *skinnyRuneTreeEntry
		for byteValsIndex, byteVal := range node.byteVals {
			if runeByte == byteVal {
				nextEntry = node.entries[byteValsIndex]
				break
			}
		}
		if nextEntry == nil {
			// have to make a new entry
			nextEntry = &skinnyRuneTreeEntry{}
			if runeByteIndex == len(runeBytes)-1 {
				nextEntry.next = dest
			} else {
				nextEntry.node = &skinnyRuneTreeNode{}
			}
			node.byteVals = append(node.byteVals, runeByte)
			node.entries = append(node.entries, nextEntry)
		}
		node = nextEntry.node
	}
}
func nfaFromSkinnyRuneTree(root *skinnyRuneTreeNode, pp printer) *smallTable {
	return tableFromSkinnyRuneTreeNode(root, pp)
}
func tableFromSkinnyRuneTreeNode(node *skinnyRuneTreeNode, pp printer) *smallTable {
	var unpacked unpackedTable
	for index, byteVal := range node.byteVals {
		entry := node.entries[index]
		if entry.next != nil {
			unpacked[byteVal] = entry.next
		} else {
			table := tableFromSkinnyRuneTreeNode(entry.node, pp)
			pp.labelTable(table, fmt.Sprintf("on %x", byteVal))
			unpacked[byteVal] = &faState{table: table}
		}
	}
	st := newSmallTable()
	st.pack(&unpacked)
	return st
}

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
