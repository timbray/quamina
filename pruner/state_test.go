package pruner

import (
	"fmt"
	quamina "quamina/lib"
	"testing"
)

func TestMemIterateFerr(t *testing.T) {
	s := NewMemState()
	f := func(x quamina.X, pattern string) error {
		return fmt.Errorf("broken")
	}
	if err := s.Add(1, "{}"); err != nil {
		t.Fatal(err)
	}
	if err := s.Iterate(f); err == nil {
		t.Fatal("expected error")
	}
}
