package quamina

import (
	"fmt"
	"sync/atomic"
	"testing"
)

func TestMemIterateFerr(t *testing.T) {
	s := newMemState()
	f := func(x X, pattern string, buildMode MatcherBuildMode) error {
		return fmt.Errorf("broken")
	}
	if err := s.Add(1, "{}", BuiltForComfort); err != nil {
		t.Fatal(err)
	}
	if err := s.Iterate(f); err == nil {
		t.Fatal("expected error")
	}
}

// This test was used to exercise and then fix the data race from bug #554
func TestLivePatternConcurrency(t *testing.T) {
	deleter, err := New(WithPatternDeletion(true))
	if err != nil {
		t.Error(err)
	}
	matcher := deleter.Copy()
	patterns := []string{
		`{"x": [{"wildcard": "t*ortilla"}]}`,
		`{"x": [{"wildcard": "tortilla*"}]}`,
		`{"x": [{"wildcard": "*tortilla"}]}`,
		`{"x": [{"wildcard": "tortil*la"}]}`,
	}
	reps := 5000
	var stopper atomic.Bool
	stopper.Store(false)
	go doPrunerUpdates(t, deleter, reps, patterns, &stopper)
	doPrunerMatches(t, matcher, &stopper)
}

func doPrunerMatches(t *testing.T, matcher *Quamina, stopper *atomic.Bool) {
	t.Helper()
	for i := 0; true; i++ {
		_, err := matcher.MatchesForEvent([]byte(`{"x": "tortilla"}`))
		if err != nil {
			t.Error(err)
		}
		if (i%100 == 0) && stopper.Load() {
			break
		}
	}
}

func doPrunerUpdates(t *testing.T, deleter *Quamina, reps int, patterns []string, stopper *atomic.Bool) {
	t.Helper()
	calls := 0
	var err error
	for i, pattern := range patterns {
		err = deleter.AddPattern(i, pattern)
		if err != nil {
			t.Error(err)
		}
	}
	for i := 0; i < reps; i++ {
		targetInd := calls % len(patterns)
		err = deleter.DeletePatterns(targetInd)
		if err != nil {
			t.Error(err)
		}
		err = deleter.AddPattern(targetInd, patterns[targetInd])
		if err != nil {
			t.Error(err)
		}
	}
	stopper.Store(true)
}

func TestStateDelete(t *testing.T) {
	s := newMemState()

	if err := s.Add(1, `{"likes":"queso"}`, BuiltForComfort); err != nil {
		t.Fatal(err)
	}

	if err := s.Add(1, `{"likes":"tacos"}`, BuiltForComfort); err != nil {
		t.Fatal(err)
	}

	if n, err := s.Delete(1); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}
}
