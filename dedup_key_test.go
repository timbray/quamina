package quamina

import (
	"testing"
)

func TestTableShareKey_SharedBackings(t *testing.T) {
	// Construct one smallTable, value-copy it (simulating post-embed share).
	src := smallTable{
		ceilings: []byte{'a', 'b', byte(byteCeiling)},
		steps:    []*faState{nil, nil, nil},
	}
	copy1 := src
	copy2 := src
	if newTableShareKey(&copy1) != newTableShareKey(&copy2) {
		t.Errorf("value-copied tables should share key; got %v vs %v",
			newTableShareKey(&copy1), newTableShareKey(&copy2))
	}
}

func TestTableShareKey_DistinctBackings(t *testing.T) {
	t1 := smallTable{
		ceilings: []byte{'a', byte(byteCeiling)},
		steps:    []*faState{nil, nil},
	}
	t2 := smallTable{
		ceilings: []byte{'a', byte(byteCeiling)},
		steps:    []*faState{nil, nil},
	}
	if newTableShareKey(&t1) == newTableShareKey(&t2) {
		t.Errorf("independently-built tables should not share key")
	}
}

// TestTableShareKey_AppendBreaksShare verifies that when a value-copy
// is mutated via append in a way that reallocates the backing array,
// the keys diverge. We force reallocation by starting at cap=1 and
// appending many entries.
func TestTableShareKey_AppendBreaksShare(t *testing.T) {
	src := smallTable{
		ceilings: make([]byte, 0, 1),
		steps:    make([]*faState, 0, 1),
	}
	src.ceilings = append(src.ceilings, byte(byteCeiling))
	src.steps = append(src.steps, nil)
	copy1 := src
	// Appending 8 entries to a slice with cap=1 guarantees at least one
	// realloc of the steps backing.
	for i := 0; i < 8; i++ {
		copy1.steps = append(copy1.steps, nil)
		copy1.ceilings = append(copy1.ceilings, byte(i))
	}
	if newTableShareKey(&src) == newTableShareKey(&copy1) {
		t.Errorf("expected keys to diverge after append-with-realloc; got equal: %v",
			newTableShareKey(&src))
	}
}
