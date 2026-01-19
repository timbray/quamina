package quamina

import (
	"cmp"
	"slices"
	"unsafe"
)

// The idea is that in we are going to be computing the epsilon closures of NFA states, which
// will be slices of states. There will be duplicate slices and we want to deduplicate. There's
// probably a more idiomatic and efficient way to do this.
type stateLists struct {
	lists     map[string][]*faState
	dfaStates map[string]*faState
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
	// dedupe the collection
	uniquemap := make(map[*faState]bool)
	for _, state := range list {
		uniquemap[state] = true
	}
	uniques := make([]*faState, 0, len(uniquemap))
	for unique := range uniquemap {
		uniques = append(uniques, unique)
	}

	// compute a key representing the set. Disclosure: My first use of an AI to help
	// code. I had done this by Sprintf("%p")-ing the addresses and sorting/concatenating
	// the strings. Which works fine but grabbing the raw bytes and pretending they're
	// a string is going to produce keys that are exactly half the size
	keyBytes := make([]byte, 0, len(uniques)*8)
	slices.SortFunc(uniques, func(a, b *faState) int {
		return cmp.Compare(uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b)))
	})

	for _, state := range uniques {
		addr := uintptr(unsafe.Pointer(state))
		for i := 0; i < 8; i++ {
			keyBytes = append(keyBytes, byte(addr>>(i*8)))
		}
	}
	key := string(keyBytes)

	// either we have already seen this or not
	list, exists := sl.lists[key]
	if exists {
		return list, sl.dfaStates[key], true
	}
	dfaState := &faState{table: newSmallTable()}
	sl.lists[key] = uniques
	sl.dfaStates[key] = dfaState
	return uniques, dfaState, false
}
