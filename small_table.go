package quamina

// faState is used by the valueMatcher automaton - every step through the
// automaton requires a smallTable and for some of them, taking the step means you've matched a value and can
// transition to a new fieldMatcher, in which case the fieldTransitions slice will be non-nil
type faState struct {
	table            *smallTable
	fieldTransitions []*fieldMatcher
}

// struct wrapper to make this comparable to help with pack/unpack
type faNext struct {
	// serial int // very useful in debugging table construction
	steps []*faState
}

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
type smallTable struct {
	//DEBUG label string
	//DEBUG serial   uint64
	ceilings []byte
	steps    []*faNext
}

// newSmallTable mostly exists to enforce the constraint that every smallTable has a byteCeiling entry at
// the end, which smallTable.step totally depends on.
func newSmallTable() *smallTable {
	return &smallTable{
		//DEBUG serial:   rand.Uint64() % 1000,
		ceilings: []byte{byte(byteCeiling)},
		steps:    []*faNext{nil},
	}
}

// step finds the member of steps in the smallTable that corresponds to the utf8Byte argument. It may return nil.
func (t *smallTable) step(utf8Byte byte) *faNext {
	for index, ceiling := range t.ceilings {
		if utf8Byte < ceiling {
			return t.steps[index]
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

// unpackedTable replicates the data in the smallTable ceilings and steps arrays.  It's quite hard to
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

// setDefault sets all the values of the table to the provided faNext pointer
// TODO: Do we need this at all? Maybe just a variant of newSmallTable?
func (t *smallTable) setDefault(s *faNext) {
	t.steps = []*faNext{s}
	t.ceilings = []byte{byte(byteCeiling)}
}

// Debugging from here down
/*
// addRangeSteps not currently used but think it will be useful in future regex-y work
func (t *smallTable) addRangeSteps(floor int, ceiling int, s *faNext) {
	unpacked := unpackTable(t)
	for i := floor; i < ceiling; i++ {
		unpacked[i] = s
	}
	t.pack(unpacked)
}

func st2(t *smallTable) string {
	// going to build a string rep of a smallTable based on the unpacked form
	// each line is going to be a range like
	// 'c' .. 'e' => %X
	// lines where the *faNext is nil are omitted
	var rows []string
	unpacked := unpackTable(t)

	var rangeStart int
	var b int

	defTrans := unpacked[0]

	for {
		for b < len(unpacked) && unpacked[b] == nil {
			b++
		}
		if b == len(unpacked) {
			break
		}
		rangeStart = b
		lastN := unpacked[b]
		for b < len(unpacked) && unpacked[b] == lastN {
			b++
		}
		if lastN != defTrans {
			row := ""
			if b == rangeStart+1 {
				row += fmt.Sprintf("'%s'", branchChar((byte(rangeStart))))
			} else {
				row += fmt.Sprintf("'%s'…'%s'", branchChar(byte(rangeStart)), branchChar(byte(b-1)))
			}
			row += " → " + lastN.String()
			rows = append(rows, row)
		}
	}
	if defTrans != nil {
		dtString := "★ → " + defTrans.String()
		return fmt.Sprintf("%d [%s] ", t.serial, t.label) + strings.Join(rows, " / ") + " / " + dtString
	} else {
		return fmt.Sprintf("%d [%s] ", t.serial%1000, t.label) + strings.Join(rows, " / ")
	}
}

func branchChar(b byte) string {
	switch b {
	case 0:
		return "∅"
	case valueTerminator:
		return "ℵ"
	case byte(byteCeiling):
		return "♾️"
	default:
		return fmt.Sprintf("%c", b)
	}
}
*/
