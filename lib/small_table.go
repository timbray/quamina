package quamina

import "fmt"

// smallTable serves as a lookup table that encodes mappings between ranges of byte values and the SmallStep
//  transition on any byte in the range.
//  The way it works is exposed in the step() function just below.  Logically, it's a slice of {byte, *smallStep}
//  but I imagine organizing it this way is a bit more memory-efficient.  Suppose we want to model a table where
//  byte values 3 and 4 (0-based) map to ss1 and byte 0x34 maps to ss2.  Then the smallTable would look like:
//  ceilings: 3,   5,    0x34, 0x35, ByteCeiling
//     steps: nil, &ss1, nil,  &ss2, nil
//  invariant: The last element of ceilings is always ByteCeiling
// The motivation is that we want to build a state machine on byte values to implement things like prefixes and
//  ranges of bytes.  This could be done simply with a byte array of size ByteCeiling for each state in the machine,
//  or a map[byte]smallStep, but both would be size-inefficient, particularly in the case where you're implementing
//  ranges.  Now, the step function is O(N) in the number of entries, but empirically, the number of entries is
//  small even in large automata, so skipping throgh the ceilings list is measurably about the same speed as a map
//  or array construct
type smallTable struct {
	slices *stSlices
}

// stSlices exists so that we can construct the ceilings and steps arrays and then atomically update both at
//  the same time, in place, while other threads are using the table
type stSlices struct {
	ceilings []byte
	steps    []smallStep
}

// ByteCeiling - the automaton runs on UTF-8 bytes, which map nicely to Go byte, which is uint8. The values
//  0xF5-0xFF can't appear in UTF-8 strings. We use 0xF5 as a value terminator, so characters F6 and higher
//  can't appear.
const ByteCeiling int = 0xf6

func newSmallTable() *smallTable {
	return &smallTable{
		slices: &stSlices{
			ceilings: []byte{byte(ByteCeiling)},
			steps:    []smallStep{nil},
		},
	}
}

// SmallTable and SmallTransition implement smallStep interface
func (t *smallTable) SmallTable() *smallTable {
	return t
}
func (t *smallTable) SmallTransition() *smallTransition {
	return nil
}
func (t *smallTable) HasTransition() bool {
	return false
}

func (t *smallTable) step(utf8Byte byte) smallStep {
	for index, ceiling := range t.slices.ceilings {
		if utf8Byte < ceiling {
			return t.slices.steps[index]
		}
	}
	panic("Malformed SmallTable")
}

// mergeAutomata computes the union of two valueMatch automata
//  INVARIANT: neither argument is nil
//  INVARIANT: To be thread-safe, no existing table can be updated
func mergeAutomata(existing, newStep smallStep) *smallTable {
	return mergeOne(existing, newStep, make(map[string]smallStep)).SmallTable()
}

func mergeOne(existing, newStep smallStep, memoize map[string]smallStep) smallStep {
	var combined smallStep

	// to support automata that loop back to themselves (typically on *) we have to stop recursing (and also
	//  trampolined recursion)
	// I'm not sure this is the best way to key a map with two interface{} variables. Among other things, it
	// relies on the ordering, but a glance at this func seems to show that's OK in this particular situation
	mKey := fmt.Sprintf("%v%v", existing, newStep)
	combined, ok := memoize[mKey]
	if ok {
		return combined
	}

	// we always take the transition from the existing step, even if there's another in the merged-in step
	switch {
	case !(existing.HasTransition() || newStep.HasTransition()):
		combined = newSmallTable()
	case existing.HasTransition() && newStep.HasTransition():
		transitions := append(existing.SmallTransition().fieldMatchers, newStep.SmallTransition().fieldMatchers...)
		combined = newSmallMultiTransition(transitions)
	case existing.HasTransition() && (!newStep.HasTransition()):
		combined = newSmallMultiTransition(existing.SmallTransition().fieldMatchers)
	case (!existing.HasTransition()) && newStep.HasTransition():
		combined = newSmallMultiTransition(newStep.SmallTransition().fieldMatchers)
	}
	memoize[mKey] = combined

	uExisting := unpack(existing.SmallTable())
	uNew := unpack(newStep.SmallTable())
	var uComb unpackedTable
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
			uComb[i] = mergeOne(stepExisting, stepNew, memoize)
		}
	}
	combined.SmallTable().pack(&uComb)
	return combined
}

// unpackedTable replicates the data in the smallTable ceilings and steps arrays.  It's quite hard to
//  update the list structure in a smallTable, but trivial in an unpackedTable.  The idea is that to update
//  a smallTable you unpack it, update, then re-pack it.  Not gonna be the most efficient thing so at some future point…
// TODO: Figure out how to update a smallTable in place
type unpackedTable [ByteCeiling]smallStep

func unpack(t *smallTable) *unpackedTable {
	var u unpackedTable
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

func (t *smallTable) pack(u *unpackedTable) {
	var slices stSlices
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
	t.slices = &slices // atomic update
}

func (t *smallTable) addByteStep(utf8Byte byte, step smallStep) {
	unpacked := unpack(t)
	unpacked[utf8Byte] = step
	t.pack(unpacked)
}

func (t *smallTable) addRangeSteps(floor int, ceiling int, step smallStep) {
	unpacked := unpack(t)
	for i := floor; i < ceiling; i++ {
		unpacked[i] = step
	}
	t.pack(unpacked)
}
