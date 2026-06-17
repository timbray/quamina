package quamina

import "unsafe"

// tableShareKey returns a stable identifier for a smallTable's "share group".
// Two states whose smallTables hold slice-headers pointing at the same `steps`
// backing array (which is what happens when one smallTable struct value is
// copied into multiple faStates during construction) will produce equal
// keys. This replaces *smallTable-pointer identity as the dedup key in
// epsilon-closure computation after smallTable is embedded into faState
// by value.
//
// The key is just the steps backing-array pointer: share groups are only ever
// born by copying a whole steps slice-header (see the spinner merges in
// nfa.go), so two tables that share the data pointer always share the length
// too — nothing in the package reslices steps. Pointer identity is therefore
// sufficient to identify a share group; carrying the length as well would
// never break a tie the pointer didn't already break.
//
// A zero key (nil pointer) means "no share group" — used for tables with no
// byte transitions. Callers that want to dedup such tables should skip the
// zero key.
type tableShareKey struct {
	stepsData unsafe.Pointer
}

func newTableShareKey(t *smallTable) tableShareKey {
	return tableShareKey{
		stepsData: unsafe.Pointer(unsafe.SliceData(t.steps)),
	}
}
