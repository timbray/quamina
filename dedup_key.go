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
// A zero key (nil pointer, len 0) means "no share group" — used for tables
// with no byte transitions. Callers that want to dedup such tables should
// skip the zero key.
type tableShareKey struct {
	stepsData unsafe.Pointer
	stepsLen  int
}

func newTableShareKey(t *smallTable) tableShareKey {
	return tableShareKey{
		stepsData: unsafe.Pointer(unsafe.SliceData(t.steps)),
		stepsLen:  len(t.steps),
	}
}
