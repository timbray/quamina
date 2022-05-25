package core

// dfaStep and nfaStep are used by the valueMatcher automaton - every step through the
//  automaton requires a smallTable and for some of them, taking the step means you've matched a value and can
//  transition to a new fieldMatcher, in which case the fieldTransitions slice will be non-nil
type dfaStep struct {
	table            *smallTable[DS]
	fieldTransitions []*fieldMatcher
}
type DS *dfaStep

type nfaStep struct {
	table            *smallTable[NSL]
	fieldTransitions []*fieldMatcher
}

// struct wrapper to make this comparable to help with pack/unpack
type nfaStepList struct {
	steps []*nfaStep
}
type NSL *nfaStepList

// TODO: declare dfaTable { smallTable[*dfaStep } and nfaTable { smallTable[*nfaStepList] }
//  and make a bunch of code more concise and readable.

// ValueTerminator - whenever we're trying to match a value with a pattern that extends to the end of that
//  value, we virtually add one of these as the last character, both to the automaton and the value at run-time.
//  This simplifies things because you don't have to treat absolute-string-match (only works at last char in
//  value) and prefix match differently.
const ValueTerminator byte = 0xf5

// smallTable serves as a lookup table that encodes mappings between ranges of byte values and the SmallStep
//  transition on any byte in the range.
//  The way it works is exposed in the step() function just below.  Logically, it's a slice of {byte, S}
//  but I imagine organizing it this way is a bit more memory-efficient.  Suppose we want to model a table where
//  byte values 3 and 4 map to ss1 and byte 0x34 maps to ss2.  Then the smallTable would look like:
//  ceilings: 3,   5,    0x34, 0x35, ByteCeiling
//     steps: nil, &ss1, nil,  &ss2, nil
//  invariant: The last element of ceilings is always ByteCeiling
// The motivation is that we want to build a state machine on byte values to implement things like prefixes and
//  ranges of bytes.  This could be done simply with a byte array of size ByteCeiling for each state in the machine,
//  or a map[byte]S, but both would be size-inefficient, particularly in the case where you're implementing
//  ranges.  Now, the step function is O(N) in the number of entries, but empirically, the number of entries is
//  small even in large automata, so skipping throgh the ceilings list is measurably about the same speed as a map
//  or array construct. One could imagine making step() smarter and do a binary search in the case where there are
//  more than some number of entries. But I'm dubious, the ceilings field is []byte and running through a single-digit
//  number of those has a good chance of staying in the L1 cache
type smallTable[S comparable] struct {
	slices stSlices[S]
}

// stSlices exists so that we can construct the ceilings and steps arrays and then atomically update both at
//  the same time, in place, while other threads are using the table
type stSlices[S comparable] struct {
	ceilings []byte
	steps    []S
}

// ByteCeiling - the automaton runs on UTF-8 bytes, which map nicely to Go's byte, which is uint8. The values
//  0xF5-0xFF can't appear in UTF-8 strings. We use 0xF5 as a value terminator, so characters F6 and higher
//  can't appear.
const ByteCeiling int = 0xf6

// newSmallTable mostly exists to enforce the constraint that every smallTable has a ByteCeiling entry at
//  the end, which smallTable.step totally depends on.
func newSmallTable[S comparable]() *smallTable[S] {
	var s S
	return &smallTable[S]{
		slices: stSlices[S]{
			ceilings: []byte{byte(ByteCeiling)},
			steps:    []S{s},
		},
	}
}

// step finds the step in the smallTable that corresponds to the utf8Byte argument. It may return nil.
func (t *smallTable[S]) step(utf8Byte byte) S {
	for index, ceiling := range t.slices.ceilings {
		if utf8Byte < ceiling {
			return t.slices.steps[index]
		}
	}
	panic("Malformed smallTable")
}

// mergeDfas and mergeNfas compute the union of two valueMatch automata.  If you look up the textbook theory about this,
//  they say to compute the set product for automata A and B and build A0B0, A0B1 … A1BN, A1B0 … but if you look
//  at that you realize that many of the product states aren't reachable. So you compute A0B0 and then keep
//  recursing on the transitions coming out there, I'm pretty sure you get a correct result. I don't know if it's
//  minimal or even avoids being wasteful.
//  INVARIANT: neither argument is nil
//  INVARIANT: To be thread-safe, no existing table can be updated except when we're building it
func mergeDfas(existing, newStep *smallTable[DS]) *smallTable[DS] {
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
	//  to be able to come out of this with just one *fieldMatcher
	newTable := newSmallTable[DS]()
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
	var uComb unpackedTable[DS]
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
			uComb[i] = mergeOneDfaStep(stepExisting, stepNew, memoize)
		}
	}
	combined.table.pack(&uComb)
	return combined
}

func dfa2Nfa(table *smallTable[DS]) *smallTable[NSL] {
	lister := newListMaker()
	return dfaStep2NfaStep(&dfaStep{table: table}, lister).table
}

func dfaStep2NfaStep(dStep *dfaStep, lister *listMaker) *nfaStep {
	nStep := &nfaStep{table: newSmallTable[NSL](), fieldTransitions: dStep.fieldTransitions}
	dUnpacked := unpackTable(dStep.table)
	var nUnpacked unpackedTable[NSL]
	for i, nextDStep := range dUnpacked {
		if nextDStep != nil {
			nUnpacked[i] = lister.getList(dfaStep2NfaStep(nextDStep, lister))
		}
	}
	nStep.table.pack(&nUnpacked)
	return nStep
}

type nfaStepKey struct {
	step1 *nfaStep
	step2 *nfaStep
}

func mergeNfas(nfa1, nfa2 *smallTable[NSL]) *smallTable[NSL] {
	step1 := &nfaStep{table: nfa1}
	step2 := &nfaStep{table: nfa2}
	return mergeOneNfaStep(step1, step2, make(map[nfaStepKey]*nfaStep), newListMaker(), 0).table
}

func mergeOneNfaStep(step1, step2 *nfaStep, memoize map[nfaStepKey]*nfaStep, lister *listMaker, depth int) *nfaStep {
	var combined *nfaStep
	mKey := nfaStepKey{step1: step1, step2: step2}
	combined, ok := memoize[mKey]
	if ok {
		return combined
	}

	newTable := newSmallTable[NSL]()
	switch {
	case step1.fieldTransitions == nil && step2.fieldTransitions == nil:
		combined = &nfaStep{table: newTable}
	case step1.fieldTransitions != nil && step2.fieldTransitions != nil:
		transitions := append(step1.fieldTransitions, step2.fieldTransitions...)
		combined = &nfaStep{table: newTable, fieldTransitions: transitions}
	case step1.fieldTransitions != nil && step2.fieldTransitions == nil:
		combined = &nfaStep{table: newTable, fieldTransitions: step1.fieldTransitions}
	case step1.fieldTransitions == nil && step2.fieldTransitions != nil:
		combined = &nfaStep{table: newTable, fieldTransitions: step2.fieldTransitions}
	}
	memoize[mKey] = combined

	u1 := unpackTable(step1.table)
	u2 := unpackTable(step2.table)
	var uComb unpackedTable[NSL]
	for i, list1 := range u1 {
		list2 := u2[i]
		switch {
		case list1 == nil && list2 == nil:
			uComb[i] = nil
		case list1 != nil && list2 == nil:
			uComb[i] = u1[i]
		case list1 == nil && list2 != nil:
			uComb[i] = u2[i]
		case list1 != nil && list2 != nil:
			var comboList []*nfaStep
			for _, nextStep1 := range list1.steps {
				for _, nextStep2 := range list2.steps {
					merged := mergeOneNfaStep(nextStep1, nextStep2, memoize, lister, depth+1)
					comboList = append(comboList, merged)
				}
			}
			uComb[i] = lister.getList(comboList...)
		}
	}
	combined.table.pack(&uComb)
	return combined
}

// TODO: Clean up from here on down - too many funcs doing about the same thing, and also it seems that
//  we never want to have more than one "range", which is the whole table.

// makeSmallDfaTable creates a pre-loaded small table, with all bytes not otherwise specified having the defaultStep
//  value, and then a few other values with their indexes and values specified in the other two arguments. The
//  goal is to reduce memory churn
// constraint: positions must be provided in order
func makeSmallDfaTable(defaultStep DS, indices []byte, steps []DS) *smallTable[DS] {
	t := smallTable[DS]{
		slices: stSlices[DS]{
			ceilings: make([]byte, 0, len(indices)+2),
			steps:    make([]DS, 0, len(indices)+2),
		}}
	slices := &t.slices
	var lastIndex byte = 0
	for i, index := range indices {
		if index > lastIndex {
			slices.ceilings = append(slices.ceilings, index)
			slices.steps = append(slices.steps, defaultStep)
		}
		slices.ceilings = append(slices.ceilings, index+1)
		slices.steps = append(slices.steps, steps[i])
		lastIndex = index + 1
	}
	if indices[len(indices)-1] < byte(ByteCeiling) {
		slices.ceilings = append(slices.ceilings, byte(ByteCeiling))
		slices.steps = append(slices.steps, defaultStep)
	}
	return &t
}

// unpackedTable replicates the data in the smallDfaTable ceilings and steps arrays.  It's quite hard to
//  update the list structure in a smallDfaTable, but trivial in an unpackedTable.  The idea is that to update
//  a smallDfaTable you unpack it, update, then re-pack it.  Not gonna be the most efficient thing so at some future point…
// TODO: Figure out how to update a smallDfaTable in place
type unpackedTable[S comparable] [ByteCeiling]S

func unpackTable[S comparable](t *smallTable[S]) *unpackedTable[S] {
	var u unpackedTable[S]
	unpackedIndex := 0
	for packedIndex, c := range t.slices.ceilings {
		ceiling := int(c)
		for unpackedIndex < ceiling {
			u[unpackedIndex] = t.slices.steps[packedIndex]
			unpackedIndex++
		}
	}
	return &u
}

func (t *smallTable[S]) pack(u *unpackedTable[S]) {
	var slices stSlices[S]
	lastStep := u[0]
	for unpackedIndex, ss := range u {
		if ss != lastStep {
			slices.ceilings = append(slices.ceilings, byte(unpackedIndex))
			slices.steps = append(slices.steps, lastStep)
		}
		lastStep = ss
	}
	slices.ceilings = append(slices.ceilings, byte(ByteCeiling))
	slices.steps = append(slices.steps, lastStep)
	t.slices = slices // atomic update
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
