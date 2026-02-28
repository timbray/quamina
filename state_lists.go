package quamina

import (
	"cmp"
	"encoding/binary"
	"slices"
	"unsafe"
)

// The idea is that in we are going to be computing the epsilon closures of NFA states, which
// will be slices of states. There will be duplicate slices and we want to deduplicate. There's
// probably a more idiomatic and efficient way to do this.
type stateLists struct {
	lists     map[string][]*faState
	dfaStates map[string]*faState
	// Scratch space reused across intern() calls
	sortBuf []*faState // reusable sorted buffer
	keyBuf  []byte     // reusable key bytes buffer
}

func newStateLists() *stateLists {
	return &stateLists{
		lists:     make(map[string][]*faState),
		dfaStates: make(map[string]*faState),
	}
}

// intern turns a collection of states that may have dupes and, when deduped and
// considered as a set of states, may be identical to a previously-seen set of states.
// It returns a canonicalized set representation of the collection, a DFA state
// which either has already been computed for the set or is created and empty, and
// a boolean indicating whether the DFA state has already been computed or not.
func (sl *stateLists) intern(list []*faState) ([]*faState, *faState, bool) {
	// dedupe using generation counter instead of a map
	closureGeneration++
	gen := closureGeneration
	sl.sortBuf = sl.sortBuf[:0]
	for _, state := range list {
		if state.closureSetGen != gen {
			state.closureSetGen = gen
			sl.sortBuf = append(sl.sortBuf, state)
		}
	}

	// compute a key representing the set
	slices.SortFunc(sl.sortBuf, func(a, b *faState) int {
		return cmp.Compare(uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b)))
	})

	needed := len(sl.sortBuf) * 8
	if cap(sl.keyBuf) < needed {
		sl.keyBuf = make([]byte, needed)
	} else {
		sl.keyBuf = sl.keyBuf[:needed]
	}
	for i, state := range sl.sortBuf {
		binary.LittleEndian.PutUint64(sl.keyBuf[i*8:], uint64(uintptr(unsafe.Pointer(state))))
	}

	// string(sl.keyBuf) in a map lookup is optimized by the compiler to avoid allocation
	if list, exists := sl.lists[string(sl.keyBuf)]; exists {
		return list, sl.dfaStates[string(sl.keyBuf)], true
	}

	// cache miss: allocate owned copies for the map
	key := string(sl.keyBuf)
	stored := make([]*faState, len(sl.sortBuf))
	copy(stored, sl.sortBuf)

	dfaState := &faState{table: newSmallTable()}
	sl.lists[key] = stored
	sl.dfaStates[key] = dfaState
	return stored, dfaState, false
}
