package quamina

import (
	"fmt"
	"testing"
)

func TestLiveRatioTrigger(t *testing.T) {
	r := newLiveRatioTrigger(0.5, 2)

	s := &Stats{}

	if r.Rebuild(false, s) {
		t.Fatal("shouldn't have fired")
	}

	s.Live = 5
	s.Deleted = 3

	if r.Rebuild(true, s) {
		t.Fatal("shouldn't have fired")
	}

	if !r.Rebuild(false, s) {
		t.Fatal("should have fired")
	}

	s.Live = 1
	if r.Rebuild(false, s) {
		t.Fatal("shouldn't have fired")
	}

}

func TestNeverTrigger(t *testing.T) {
	r := newNeverTrigger()
	s := &Stats{
		Live:    42,
		Deleted: 17,
	}
	if r.Rebuild(false, s) {
		t.Fatal("you only had one job")
	}
}

// sane verifies that certain Stats are not negative.
//
// The types in question aren't uint(64) but maybe they should be.
func (s Stats) sane() error {

	if s.Live < 0 {
		return fmt.Errorf("Stats.Live is negative")
	}

	if s.Added < 0 {
		return fmt.Errorf("Stats.Added is negative")
	}

	if s.Deleted < 0 {
		return fmt.Errorf("Stats.Deleted is negative")
	}

	if s.Filtered < 0 {
		return fmt.Errorf("Stats.Filtered is negative")
	}

	return nil

}

func (m *PrunerMatcher) checkStats() error {
	return m.Stats().sane()
}
