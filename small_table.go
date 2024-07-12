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
//	ceilings:--|3|----|5|-|0x34|--|x35|-|byteCeiling|
//	states:---|nil|-|&ss1|--|nil|-|&ss2|---------|nil|
//	invariant: The last element of ceilings is always byteCeiling
//
// The motivation is that we want to build a state machine on byte values to implement things like prefixes and
// ranges of bytes.  This could be done simply with an array of size byteCeiling for each state in the machine,
// or a map[byte]S, but both would be size-inefficient, particularly in the case where you're implementing
// ranges.  Now, the step function is O(N) in the number of entries, but empirically, the number of entries is
// small even in large automata, so skipping throgh the ceilings list is measurably about the same speed as a map
// or array construct. One could imagine making step() smarter and do a binary search in the case where there are
// more than some number of entries. But I'm dubious, the ceilings field is []byte and running through a single-digit
// number of those has a good chance of minimizing memory fetches.
// Since this is used to support nondeterministic finite automata (NFAs), it is possible for a state
// to have epsilon transitions, i.e. a transition that is always taken whatever the next input symbol is.
type smallTable struct {
	ceilings []byte
	steps    []*faNext
	epsilon  []*faState
}

// newSmallTable mostly exists to enforce the constraint that every smallTable has a byteCeiling entry at
// the end, which smallTable.step totally depends on.
func newSmallTable() *smallTable {
	return &smallTable{
		ceilings: []byte{byte(byteCeiling)},
		steps:    []*faNext{nil},
	}
}

type stepOut struct {
	steps   []*faState
	epsilon []*faState
}

// step finds the list of states that result from a transition on the utf8Byte argument. The states can come
// as a result of looking in the table structure, and also the "epsilon" transitions that occur on every
// input byte.  Since this is the white-hot center of Quamina's runtime CPU, we don't want to be merging
// the two lists. So to avoid any memory allocation, the caller passes in a structure with the two lists
// and step fills them in.
func (t *smallTable) step(utf8Byte byte, out *stepOut) {
	out.epsilon = t.epsilon
	for index, ceiling := range t.ceilings {
		if utf8Byte < ceiling {
			if t.steps[index] == nil {
				out.steps = nil
			} else {
				out.steps = t.steps[index].states
			}
			return
		}
	}
	panic("Malformed smallTable")
}

// dStep takes a step through an NFA in the case where it is known that the NFA in question
// is deterministic, i.e. each combination of an faState and a byte value transitions to at
// most one other byte value.
func (t *smallTable) dStep(utf8Byte byte) *faState {
	for index, ceiling := range t.ceilings {
		if utf8Byte < ceiling {
			if t.steps[index] == nil {
				return nil
			} else {
				return t.steps[index].states[0]
			}
		}
	}
	panic("Malformed smallTable")
}

// makeSmallTable creates a pre-loaded small table, with all bytes not otherwise specified having the defaultStep
// value, and then a few other values with their indexes and values specified in the other two arguments. The
// goal is to reduce memory churn
// constraint: positions must be provided in order
func makeSmallTable(defaultStep *faNext, indices []byte, steps []*faNext) *smallTable {
	t := smallTable{
		ceilings: make([]byte, 0, len(indices)+2),
		steps:    make([]*faNext, 0, len(indices)+2),
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

func (t *smallTable) gatherMetadata(meta *nfaMetadata) {
	eps := len(t.epsilon)
	for _, step := range t.steps {
		if step != nil {
			if (eps + len(step.states)) > meta.maxOutDegree {
				meta.maxOutDegree = eps + len(step.states)
			}
			for _, state := range step.states {
				state.table.gatherMetadata(meta)
			}
		}
	}
}

// unpackedTable replicates the data in the smallTable ceilings and states arrays.  It's quite hard to
// update the list structure in a smallTable, but trivial in an unpackedTable.  The idea is that to update
// a smallTable you unpack it, update, then re-pack it.  Not gonna be the most efficient thing so at some future point…
// TODO: Figure out how to update a smallTable in place
type unpackedTable [byteCeiling]*faNext

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
	steps := make([]*faNext, 0, 16)
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

func (t *smallTable) addByteStep(utf8Byte byte, step *faNext) {
	unpacked := unpackTable(t)
	unpacked[utf8Byte] = step
	t.pack(unpacked)
}
