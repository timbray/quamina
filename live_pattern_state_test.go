package quamina

import (
	"fmt"
	"testing"
)

func TestMemIterateFerr(t *testing.T) {
	s := newMemState()
	f := func(x X, pattern string) error {
		return fmt.Errorf("broken")
	}
	if err := s.Add(1, "{}"); err != nil {
		t.Fatal(err)
	}
	if err := s.Iterate(f); err == nil {
		t.Fatal("expected error")
	}
}

func TestStateDelete(t *testing.T) {
	s := newMemState()

	if err := s.Add(1, `{"likes":"queso"}`); err != nil {
		t.Fatal(err)
	}

	if err := s.Add(1, `{"likes":"tacos"}`); err != nil {
		t.Fatal(err)
	}

	if n, err := s.Delete(1); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}
}
