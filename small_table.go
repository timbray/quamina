package quamina

// byteCeiling - the automaton runs on UTF-8 bytes, which map nicely to Go's byte, which is uint8. The values
// 0xF5-0xFF can't appear in UTF-8 strings. We use 0xF5 as a value terminator, so characters F6 and higher
// can't appear.
const byteCeiling int = 0xf6

// valueTerminator - whenever we're trying to match a value with a pattern that extends to the end of that
// value, we virtually add one of these as the last character, both to the automaton and the value at run-time.
// This simplifies things because you don't have to treat absolute-string-match (only works at last char in
// value) and prefix match differently.
const valueTerminator byte = 0xf5

// nolint:gofmt,goimports
// smallTable serves as a lookup table that encodes mappings between ranges of byte values and the
// transition on any byte in the range.
//
// The way it works is exposed in the step() function just below.  Logically, it's a slice of {byte, S}
// but I imagine organizing it this way is a bit more memory-efficient.  Suppose we want to model a table where
// byte values 3 and 4 map to ss1 and byte 0x34 maps to ss2.  Then the smallTable would look like:
//
//	ceilings:-|  3|-|   5|-|0x34|-| x35|-|byteCeiling|
//	states:---|nil|-|&ss1|-| nil|-|&ss2|-|        nil|
//	invariant: The last element of ceilings is always byteCeiling
//
// The motivation is that we want to build a state machine on byte values to implement things like prefixes and
// ranges of bytes.  This could be done simply with an array of size byteCeiling for each state in the machine,
// or a map[byte]S, but both would be size-inefficient, particularly in the case where you're implementing
// ranges.  Now, the step function is O(N) in the number of entries, but empirically, the average number of entries is
// small even in large automata, so skipping through the ceilings list is measurably about the same speed as a map
// or array construct. One could imagine making step() smarter and do a binary search in the case where there are
// more than some number of entries. But I'm dubious, the ceilings field is []byte and running through a single-digit
// number of those has a good chance of minimizing memory fetches.
// Since this is used to support nondeterministic finite automata (NFAs), it is possible for a state
// to have epsilon transitions, i.e. a transition that is always taken whatever the next input symbol is.
// NFAs in theory can branch to two or more other states on a single input symbol, but that can always be
// handled with epsilons. For example, if the symbol 'b' should branch to both s1 and s2, that can be handled
// by branching on 'b' to a state that has no byte transitions but two epsilons, one each for s1 and s2.

type smallTable struct {
	ceilings       []byte
	steps          []*faState
	epsilons       []*faState
	lastVisitedGen uint64   // generation counter for epsilon closure traversal
	// closureRepGen records which closureRepGeneration this table's
	// representative was set in. If it equals the current global
	// closureRepGeneration, then closureRep is valid; otherwise, the
	// table has not yet been seen in this dedup pass.
	closureRepGen uint64
	// closureRep is the representative faState for this table in the
	// current closure dedup pass. When multiple states share the same
	// smallTable and have identical fieldTransitions, only this
	// representative is kept in the closure.
	closureRep *faState
}

// newSmallTable mostly exists to enforce the constraint that every smallTable has a byteCeiling entry at
// the end, which smallTable.step totally depends on.
func newSmallTable() *smallTable {
	return &smallTable{
		ceilings: []byte{byte(byteCeiling)},
		steps:    []*faState{nil},
	}
}

func (t *smallTable) isEpsilonOnly() bool {
	return len(t.epsilons) > 0 && len(t.ceilings) == 1
}

type stepOut struct {
	step     *faState
	epsilons []*faState
}

var forbiddenBytes = map[byte]bool{
	0xC0: true, 0xC1: true,
	0xF5: true, 0xF6: true, 0xF7: true, 0xF8: true, 0xF9: true, 0xFA: true,
	0xFB: true, 0xFC: true, 0xFD: true, 0xFE: true, 0xFF: true,
}

func (t *smallTable) isJustEpsilons() bool {
	// TODO I think the second of the three conditions is unnecessary
	return len(t.steps) == 1 && t.steps[0] == nil && len(t.epsilons) != 0
}

// step finds the list of states that result from a transition on the utf8Byte argument. The states can come
// as a result of looking in the table structure, and also the "epsilon" transitions that occur on every
// input byte.  Since this is the white-hot center of Quamina's runtime CPU, we don't want to be merging
// the two lists. So to avoid any memory allocation, the caller passes in a structure with the two lists
// and step fills them in.
func (t *smallTable) step(utf8Byte byte, out *stepOut) {
	out.epsilons = t.epsilons
	for index, ceiling := range t.ceilings {
		if utf8Byte < ceiling {
			out.step = t.steps[index]
			return
		}
	}
	_, forbidden := forbiddenBytes[utf8Byte]
	if forbidden {
		return
	}
	panic("Malformed smallTable")
}

// dStep takes a step through an NFA in the case where it is known that the NFA in question
// is deterministic, i.e. each combination of an faState and a byte value transitions to at
// most one other byte value.
func (t *smallTable) dStep(utf8Byte byte) *faState {
	for index, ceiling := range t.ceilings {
		if utf8Byte < ceiling {
			return t.steps[index]
		}
	}
	_, forbidden := forbiddenBytes[utf8Byte]
	if forbidden {
		return nil
	}
	panic("Malformed smallTable")
}

// makeSmallTable creates a pre-loaded small table, with all bytes not otherwise specified having the defaultStep
// value, and then a few other values with their indexes and values specified in the other two arguments. The
// goal is to reduce memory churn
// constraint: positions must be provided in order
func makeSmallTable(defaultStep *faState, indices []byte, steps []*faState) *smallTable {
	t := smallTable{
		ceilings: make([]byte, 0, len(indices)+2),
		steps:    make([]*faState, 0, len(indices)+2),
	}

	var lastIndex byte = 0
	for i, index := range indices {
		if index > lastIndex {
			t.ceilings = append(t.ceilings, index)
			t.steps = append(t.steps, defaultStep)
		}
		t.ceilings = append(t.ceilings, index+1)
		t.steps = append(t.steps, steps[i])
		lastIndex = index + 1
	}
	if indices[len(indices)-1] < byte(byteCeiling) {
		t.ceilings = append(t.ceilings, byte(byteCeiling))
		t.steps = append(t.steps, defaultStep)
	}
	return &t
}

// For manipulating larger-scale machines, the performance starts to be dominated by
// the unpack/pack overhead required for addByteStep, specifically by creating lots of
// garbage-collection work.  stIterator provides a cheap way to cycle through all the
// legal byte values without unpacking the smallTable.
type stIterator struct {
	table        *smallTable
	ceilingIndex int
	byteIndex    byte
}

func newSTIterator(t *smallTable, iter *stIterator) stIterator {
	// make new iterator
	if iter == nil {
		return stIterator{table: t, byteIndex: 0, ceilingIndex: 0}
	}

	// reuse existing iterator
	iter.table = t
	iter.byteIndex = 0
	iter.ceilingIndex = 0
	return *iter
}
func (si *stIterator) hasNext() bool {
	return si.byteIndex < byte(byteCeiling)
}
func (si *stIterator) next() (byte, *faState) {
	utf8byte := byte(si.byteIndex)
	si.byteIndex++
	if utf8byte == si.table.ceilings[si.ceilingIndex] {
		si.ceilingIndex++
	}
	return utf8byte, si.table.steps[si.ceilingIndex]
}
func (si *stIterator) nextState() *faState {
	utf8byte := byte(si.byteIndex)
	si.byteIndex++
	if utf8byte == si.table.ceilings[si.ceilingIndex] {
		si.ceilingIndex++
	}
	return si.table.steps[si.ceilingIndex]
}

// unpackedTable replicates the data in the smallTable ceilings and states arrays.  It's quite hard to
// update the list structure in a smallTable, but trivial in an unpackedTable.  The idea is that to update
// a smallTable you unpack it, update, then re-pack it.  Not gonna be the most efficient thing so at some future point…
// TODO: Figure out how to update a smallTable in place
type unpackedTable [byteCeiling]*faState

func unpackTable(t *smallTable) *unpackedTable {
	var u unpackedTable
	unpackedIndex := 0
	for packedIndex, c := range t.ceilings {
		ceiling := int(c)
		for unpackedIndex < ceiling {
			u[unpackedIndex] = t.steps[packedIndex]
			unpackedIndex++
		}
	}
	return &u
}

func (t *smallTable) pack(u *unpackedTable) {
	ceilings := make([]byte, 0, 16)
	steps := make([]*faState, 0, 16)
	lastStep := u[0]
	for unpackedIndex, ss := range u {
		if ss != lastStep {
			ceilings = append(ceilings, byte(unpackedIndex))
			steps = append(steps, lastStep)
		}
		lastStep = ss
	}
	ceilings = append(ceilings, byte(byteCeiling))
	steps = append(steps, lastStep)
	t.ceilings = ceilings
	t.steps = steps
}

func (t *smallTable) addByteStep(utf8Byte byte, step *faState) {
	unpacked := unpackTable(t)
	unpacked[utf8Byte] = step
	t.pack(unpacked)
}

// not all regexp FAs are nondeterministic. This could have been detected at
// FA-building time, but doing so and then sending the status over to the valueMatcher
// turned out to be complex, as opposed to the following, which is not only simple but fast.
func (t *smallTable) isNondeterministic() bool {
	if len(t.epsilons) > 0 {
		return true
	}
	for _, step := range t.steps {
		if step != nil && step.table.isNondeterministic() {
			return true
		}
	}
	return false
}
