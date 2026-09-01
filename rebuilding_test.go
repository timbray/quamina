package quamina

import (
	"testing"
)

func TestLiveRatioTrigger(t *testing.T) {
	r := newLiveRatioTrigger(0.5, 2)

	s := newAsyncPrunerStats()

	if r.rebuild(false, s) {
		t.Fatal("shouldn't have fired")
	}

	s.Live.set(5)
	s.Deleted.set(3)

	if r.rebuild(true, s) {
		t.Fatal("shouldn't have fired")
	}

	if !r.rebuild(false, s) {
		t.Fatal("should have fired")
	}

	s.Live.set(1)
	if r.rebuild(false, s) {
		t.Fatal("shouldn't have fired")
	}
}

func TestNeverTrigger(t *testing.T) {
	r := newNeverTrigger()
	s := newAsyncPrunerStats()
	s.Live.set(42)
	s.Deleted.set(17)
	if r.rebuild(false, s) {
		t.Fatal("you only had one job")
	}
}

func (pm *prunerMatcher) checkStats() error {
	return pm.getPrunerStats().sane()
}
