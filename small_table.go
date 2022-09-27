package quamina

// dfaStep and nfaStep are used by the valueMatcher automaton - every step through the
// automaton requires a smallTable and for some of them, taking the step means you've matched a value and can
// transition to a new fieldMatcher, in which case the fieldTransitions slice will be non-nil
type dfaStep struct {
	table            *smallTable[*dfaStep]
	fieldTransitions []*fieldMatcher
}

type nfaStep struct {
	table            *smallTable[*nfaStepList]
	fieldTransitions []*fieldMatcher
}

// struct wrapper to make this comparable to help with pack/unpack
type nfaStepList struct {
	steps []*nfaStep
}

// TODO: declare dfaTable { smallTable[*dfaStep } and nfaTable { smallTable[*nfaStepList] }
//  and make a bunch of code more concise and readable.

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
//	steps:---|nil|-|&ss1|--|nil|-|&ss2|---------|nil|
//	invariant: The last element of ceilings is always byteCeiling
//
// The motivation is that we want to build a state machine on byte values to implement things like prefixes and
// ranges of bytes.  This could be done simply with an array of size byteCeiling for each state in the machine,
// or a map[byte]S, but both would be size-inefficient, particularly in the case where you're implementing
// ranges.  Now, the step function is O(N) in the number of entries, but empirically, the number of entries is
// small even in large automata, so skipping throgh the ceilings list is measurably about the same speed as a map
// or array construct. One could imagine making step() smarter and do a binary search in the case where there are
// more than some number of entries. But I'm dubious, the ceilings field is []byte and running through a single-digit
// number of those has a good chance of minimizing memory fetches
type smallTable[S comparable] struct {
	ceilings []byte
	steps    []S
}

// newSmallTable mostly exists to enforce the constraint that every smallTable has a byteCeiling entry at
// the end, which smallTable.step totally depends on.
func newSmallTable[S comparable]() *smallTable[S] {
	var sNil S // declared but not assigned, thus serves as nil
	return &smallTable[S]{
		ceilings: []byte{byte(byteCeiling)},
		steps:    []S{sNil},
	}
}

// step finds the member of steps in the smallTable that corresponds to the utf8Byte argument. It may return nil.
func (t *smallTable[S]) step(utf8Byte byte) S {
	for index, ceiling := range t.ceilings {
		if utf8Byte < ceiling {
			return t.steps[index]
		}
	}
	panic("Malformed smallTable")
}

// mergeDfas and mergeNfas compute the union of two valueMatch automata.  If you look up the textbook theory about this,
// they say to compute the set product for automata A and B and build A0B0, A0B1 … A1BN, A1B0 … but if you look
// at that you realize that many of the product states aren't reachable. So you compute A0B0 and then keep
// recursing on the transitions coming out, I'm pretty sure you get a correct result. I don't know if it's
// minimal or even avoids being wasteful.
// INVARIANT: neither argument is nil
// INVARIANT: To be thread-safe, no existing table can be updated except when we're building it
func mergeDfas(existing, newStep *smallTable[*dfaStep]) *smallTable[*dfaStep] {
	step1 := &dfaStep{table: existing}
	step2 := &dfaStep{table: newStep}
	return mergeOneDfaStep(step1, step2, make(map[dfaStepKey]*dfaStep)).table
}

// dfaStepKey exists to serve as the key for the memoize map that's needed to control recursion in mergeAutomata
type dfaStepKey struct {
	step1 *dfaStep
	step2 *dfaStep
}

func mergeOneDfaStep(step1, step2 *dfaStep, memoize map[dfaStepKey]*dfaStep) *dfaStep {
	var combined *dfaStep

	// to support automata that loop back to themselves (typically on *) we have to stop recursing (and also
	//  trampolined recursion)
	mKey := dfaStepKey{step1: step1, step2: step2}
	combined, ok := memoize[mKey]
	if ok {
		return combined
	}

	// TODO: this works, all the tests pass, but I'm not satisfied with it. My intuition is that you ought
	// to be able to come out of this with just one *fieldMatcher
	newTable := newSmallTable[*dfaStep]()
	switch {
	case step1.fieldTransitions == nil && step2.fieldTransitions == nil:
		combined = &dfaStep{table: newTable}
	case step1.fieldTransitions != nil && step2.fieldTransitions != nil:
		transitions := append(step1.fieldTransitions, step2.fieldTransitions...)
		combined = &dfaStep{table: newTable, fieldTransitions: transitions}
	case step1.fieldTransitions != nil && step2.fieldTransitions == nil:
		combined = &dfaStep{table: newTable, fieldTransitions: step1.fieldTransitions}
	case step1.fieldTransitions == nil && step2.fieldTransitions != nil:
		combined = &dfaStep{table: newTable, fieldTransitions: step2.fieldTransitions}
	}
	memoize[mKey] = combined

	uExisting := unpackTable(step1.table)
	uNew := unpackTable(step2.table)
	var uComb unpackedTable[*dfaStep]
	for i, stepExisting := range uExisting {
		stepNew := uNew[i]
		switch {
		case stepExisting == nil && stepNew == nil:
			uComb[i] = nil
		case stepExisting != nil && stepNew == nil:
			uComb[i] = stepExisting
		case stepExisting == nil && stepNew != nil:
			uComb[i] = stepNew
		case stepExisting != nil && stepNew != nil:
			// there are considerable runs of the same value
			if i > 0 && stepExisting == uExisting[i-1] && stepNew == uNew[i-1] {
				uComb[i] = uComb[i-1]
			} else {
				uComb[i] = mergeOneDfaStep(stepExisting, stepNew, memoize)
			}
		}
	}
	combined.table.pack(&uComb)
	return combined
}

// nfa2Dfa does what the name says. As of now it does not consider epsilon
// transitions in the NFA because, as of the time of writing, none of the
// pattern-matching required those transitions.  It is based on the algorithm
// taught in the TU München course “Automata and Formal Languages”, lecturer
// Prof. Dr. Ernst W. Mayr in 2014-15, in particular the examples appearing in
// http://wwwmayr.informatik.tu-muenchen.de/lehre/2014WS/afs/2014-10-14.pdf
// especially the slide in Example 11.
func nfa2Dfa(table *smallTable[*nfaStepList]) *smallTable[*dfaStep] {
	firstStep := &nfaStepList{steps: []*nfaStep{{table: table}}}
	return nfaStep2DfaStep(firstStep, newDfaMemory()).table
}

func nfaStep2DfaStep(stepList *nfaStepList, memoize *dfaMemory) *dfaStep {
	var dStep *dfaStep
	dStep, ok := memoize.dfaForNfas(stepList.steps...)
	if ok {
		return dStep
	}
	dStep = &dfaStep{
		table: &smallTable[*dfaStep]{},
	}
	memoize.rememberDfaForList(dStep, stepList.steps...)
	if len(stepList.steps) == 1 {
		// there's only stepList.steps[0]
		nStep := stepList.steps[0]
		dStep.fieldTransitions = nStep.fieldTransitions
		dStep.table.ceilings = make([]byte, len(nStep.table.ceilings))
		dStep.table.steps = make([]*dfaStep, len(nStep.table.ceilings)) // defaults will be nil, which is OK
		for i, nfaList := range nStep.table.steps {
			dStep.table.ceilings[i] = nStep.table.ceilings[i]
			if nfaList != nil {
				dStep.table.steps[i] = nfaStep2DfaStep(nfaList, memoize)
			}
		}
	} else {
		// coalesce - first, unpack each of the steps
		unpackedNfaSteps := make([]*unpackedTable[*nfaStepList], len(stepList.steps))
		var unpackedDfa unpackedTable[*dfaStep]
		for i, list := range stepList.steps {
			unpackedNfaSteps[i] = unpackTable(list.table)
			dStep.fieldTransitions = append(dStep.fieldTransitions, list.fieldTransitions...)
		}
		for utf8Byte := 0; utf8Byte < byteCeiling; utf8Byte++ {
			steps := make(map[*nfaStep]bool)
			for _, table := range unpackedNfaSteps {
				if table[utf8Byte] != nil {
					for _, step := range table[utf8Byte].steps {
						steps[step] = true
					}
				}
			}
			var synthStep nfaStepList
			for step := range steps {
				synthStep.steps = append(synthStep.steps, step)
			}
			unpackedDfa[utf8Byte] = nfaStep2DfaStep(&synthStep, memoize)
		}
		dStep.table.pack(&unpackedDfa)
	}

	return dStep
}

// makeSmallDfaTable creates a pre-loaded small table, with all bytes not otherwise specified having the defaultStep
// value, and then a few other values with their indexes and values specified in the other two arguments. The
// goal is to reduce memory churn
// constraint: positions must be provided in order
func makeSmallDfaTable(defaultStep *dfaStep, indices []byte, steps []*dfaStep) *smallTable[*dfaStep] {
	t := smallTable[*dfaStep]{
		ceilings: make([]byte, 0, len(indices)+2),
		steps:    make([]*dfaStep, 0, len(indices)+2),
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

// unpackedTable replicates the data in the smallTable ceilings and steps arrays.  It's quite hard to
// update the list structure in a smallDfaTable, but trivial in an unpackedTable.  The idea is that to update
// a smallDfaTable you unpack it, update, then re-pack it.  Not gonna be the most efficient thing so at some future point…
// TODO: Figure out how to update a smallDfaTable in place
type unpackedTable[S comparable] [byteCeiling]S

func unpackTable[S comparable](t *smallTable[S]) *unpackedTable[S] {
	var u unpackedTable[S]
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

func (t *smallTable[S]) pack(u *unpackedTable[S]) {
	ceilings := make([]byte, 0, 16)
	steps := make([]S, 0, 16)
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

func (t *smallTable[S]) addByteStep(utf8Byte byte, step S) {
	unpacked := unpackTable(t)
	unpacked[utf8Byte] = step
	t.pack(unpacked)
}

func (t *smallTable[S]) addRangeSteps(floor int, ceiling int, s S) {
	unpacked := unpackTable(t)
	for i := floor; i < ceiling; i++ {
		unpacked[i] = s
	}
	t.pack(unpacked)
}
