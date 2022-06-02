package quamina

import (
	"fmt"
	"testing"
)

func TestLiveRatioTrigger(t *testing.T) {
	r := newLiveRatioTrigger(0.5, 2)

	s := &prunerStats{}

	if r.rebuild(false, s) {
		t.Fatal("shouldn't have fired")
	}

	s.Live = 5
	s.Deleted = 3

	if r.rebuild(true, s) {
		t.Fatal("shouldn't have fired")
	}

	if !r.rebuild(false, s) {
		t.Fatal("should have fired")
	}

	s.Live = 1
	if r.rebuild(false, s) {
		t.Fatal("shouldn't have fired")
	}
}

func TestNeverTrigger(t *testing.T) {
	r := newNeverTrigger()
	s := &prunerStats{
		Live:    42,
		Deleted: 17,
	}
	if r.rebuild(false, s) {
		t.Fatal("you only had one job")
	}
}

// sane verifies that certain prunerStats are not negative.
//
// The types in question aren't uint(64) but maybe they should be.
func (s prunerStats) sane() error {
	if s.Live < 0 {
		return fmt.Errorf("prunerStats.Live is negative")
	}

	if s.Added < 0 {
		return fmt.Errorf("prunerStats.Added is negative")
	}

	if s.Deleted < 0 {
		return fmt.Errorf("prunerStats.Deleted is negative")
	}

	if s.Filtered < 0 {
		return fmt.Errorf("prunerStats.Filtered is negative")
	}

	return nil
}

func (m *prunerMatcher) checkStats() error {
	return m.getStats().sane()
}
