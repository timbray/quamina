package quamina

import (
	"cmp"
	"encoding/binary"
	"slices"
	"unsafe"
)

// internEntry bundles the list and DFA state into one map value so that
// cache hits require a single map lookup instead of two.
type internEntry struct {
	states   []*faState
	dfaState *faState
}

// The idea is that in we are going to be computing the epsilon closures of NFA states, which
// will be slices of states. There will be duplicate slices and we want to deduplicate. There's
// probably a more idiomatic and efficient way to do this.
type stateLists struct {
	entries map[string]internEntry
	// Scratch space reused across intern() calls
	sortBuf []*faState // reusable sorted buffer
	keyBuf  []byte     // reusable key bytes buffer
}

func newStateLists() *stateLists {
	return &stateLists{
		entries: make(map[string]internEntry),
	}
}

// intern turns a collection of states that may have dupes and, when deduped and
// considered as a set of states, may be identical to a previously-seen set of states.
// It returns a canonicalized set representation of the collection, a DFA state
// which either has already been computed for the set or is created and empty, and
// a boolean indicating whether the DFA state has already been computed or not.
func (sl *stateLists) intern(list []*faState) ([]*faState, *faState, bool) {
	// Dedup by sorting then compacting adjacent duplicates. The set key is
	// built from sorted pointers anyway, so sorting is not extra work; once
	// sorted, duplicates are adjacent and Compact removes them in one linear
	// pass. This avoids both a per-call dedup map and a per-faState
	// generation field (the latter was removed to shrink steady-state memory).
	sl.sortBuf = append(sl.sortBuf[:0], list...)
	slices.SortFunc(sl.sortBuf, func(a, b *faState) int {
		return cmp.Compare(uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b)))
	})
	sl.sortBuf = slices.Compact(sl.sortBuf)

	// Pre-size the key buffer and write pointers with PutUint64 instead of
	// appending byte-by-byte, avoiding 8 append calls and bounds checks per state.
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
	if entry, exists := sl.entries[string(sl.keyBuf)]; exists {
		return entry.states, entry.dfaState, true
	}

	// cache miss: allocate owned copies for the map
	key := string(sl.keyBuf)
	stored := make([]*faState, len(sl.sortBuf))
	copy(stored, sl.sortBuf)

	dfaState := &faState{table: newSmallTable()}
	sl.entries[key] = internEntry{states: stored, dfaState: dfaState}
	return stored, dfaState, false
}
