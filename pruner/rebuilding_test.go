package pruner

import "testing"

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
