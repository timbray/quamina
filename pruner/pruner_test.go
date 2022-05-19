package pruner

import (
	"fmt"
	"log"
	quamina "quamina/lib"
	"testing"
	"time"
)

var verbose = false

func logf(format string, args ...interface{}) {
	if !verbose {
		return
	}
	log.Printf(format, args...)
}

func (m *Matcher) printStats() {
	logf("%#v", m.Stats())
}

func TestBasic(t *testing.T) {

	var (
		pat   = `{"likes":["tacos"]}`
		id    = 1
		event = []byte(`{"likes":"tacos"}`)

		m = NewMatcher(nil)
	)

	if err := m.AddPattern(id, pat); err != nil {
		t.Fatal(err)
	}

	// It's okay to update a pattern.
	if err := m.AddPattern(id, pat); err != nil {
		t.Fatal(err)
	}

	got, err := m.MatchesForJSONEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatal(got)
	}

	m.printStats()

	if have, err := m.DeletePattern(id); err != nil {
		t.Fatal(err)
	} else if !have {
		t.Fatal(have)
	}
	if have, err := m.DeletePattern(id); err != nil {
		t.Fatal(err)
	} else if have {
		t.Fatal(have)
	}

	m.printStats()

	got, err = m.MatchesForJSONEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if 0 != len(got) {
		t.Fatal(got)
	}

	m.printStats()

	if err = m.Rebuild(true); err != nil {
		t.Fatal(err)
	}

	m.printStats()

	got, err = m.MatchesForJSONEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if 0 != len(got) {
		t.Fatal(got)
	}

	m.printStats()
}

func TestRebuildSome(t *testing.T) {
	var (
		m = NewMatcher(nil)
		n = int(defaultRebuildTrigger.MinAction + 100)
	)

	populate := func() {
		for i := 0; i < n; i++ {
			p := fmt.Sprintf(`{"like":["tacos","queso"],"want":[%d]}`, i)
			if err := m.AddPattern(i, p); err != nil {
				t.Fatal(err)
			}
		}
	}

	depopulate := func() {
		for i := 0; i < n; i += 2 {
			if had, err := m.DeletePattern(i); err != nil {
				t.Fatal(err)
			} else if !had {
				t.Fatal(i)
			}
		}
		// Maybe check a lot more often.
		if err := m.checkStats(); err != nil {
			t.Fatal(err)
		}
	}

	query := func(verify bool) {
		for i := 0; i < n; i++ {
			e := fmt.Sprintf(`{"like":"tacos","want":%d}`, i)
			if got, err := m.MatchesForJSONEvent([]byte(e)); err != nil {
				t.Fatal(err)
			} else if verify && len(got) != 1 {
				t.Fatal(got)
			}
		}
	}

	queryFast := func(verify bool) {
		f := m.NewFJ()
		for i := 0; i < n; i++ {
			e := fmt.Sprintf(`{"like":"tacos","want":%d}`, i)
			fs, err := f.Flatten([]byte(e))
			if err != nil {
				t.Fatal(err)
			}
			if got, err := m.MatchesForFields(fs); err != nil {
				t.Fatal(err)
			} else if verify && len(got) != 1 {
				t.Fatal(got)
			}
		}
	}

	t.Run("rebuild", func(t *testing.T) {
		// See a rebuild.
		populate()
		query(true)
		m.printStats()
		depopulate()
		query(false)
		if s := m.Stats(); 0 == s.RebuildDuration {
			t.Fatal(s)
		}
	})

	t.Run("norebuild", func(t *testing.T) {
		// Prevent a rebuild.
		m = NewMatcher(nil)
		m.DisableRebuild()
		populate()
		query(true)
		depopulate()
		query(false)
		if s := m.Stats(); 0 != s.RebuildDuration {
			t.Fatal(s)
		}
	})

	t.Run("rebuildafterfj", func(t *testing.T) {
		// Prevent a rebuild.
		m = NewMatcher(nil)
		m.DisableRebuild()
		populate()
		queryFast(false)
		depopulate()
		queryFast(false)
		if s := m.Stats(); 0 != s.RebuildDuration {
			t.Fatal(s)
		}
	})

}

func TestTriggerTooManyFilteredDenom(t *testing.T) {
	// Verify that a zero denominator doesn't cause problems.
	m := NewMatcher(nil)
	trigger := m.rebuildTrigger.(*tooMuchFiltering)
	trigger.MinAction = 0

	if err := m.AddPattern(1, `{"likes":["tacos"]}`); err != nil {
		t.Fatal(err)
	}
	if _, err := m.DeletePattern(1); err != nil {
		t.Fatal(err)
	}
	_, err := m.MatchesForJSONEvent([]byte(`{"likes":"tacos"}`))
	if err != nil {
		t.Fatal(err)
	}

}

func TestTriggerRebuild(t *testing.T) {

	// This test just checks that a rebuild was actually triggered.

	var (
		then    = time.Now()
		m       = NewMatcher(nil)
		trigger = m.rebuildTrigger.(*tooMuchFiltering)
		n       = 10
		doomed  = func(id int) bool {
			return id%2 == 0
		}
		printState = func() {
			logf("state:")
			for id, p := range m.live.(*MemState).m {
				logf("  %v -> %s", id, p)
			}
		}
	)

	trigger.MinAction = 5
	trigger.FilteredToEmitted = 0.5

	for i := 0; i < n; i++ {
		pat := fmt.Sprintf(`{"n":[%d]}`, i)
		if err := m.AddPattern(i, pat); err != nil {
			t.Fatal(err)
		}

		if doomed(i) {
			if _, err := m.DeletePattern(i); err != nil {
				t.Fatal(err)
			}
		}
	}

	printState()
	m.printStats()

	for i := 0; i < n; i++ {
		event := fmt.Sprintf(`{"n":%d}`, i)
		got, err := m.MatchesForJSONEvent([]byte(event))
		if err != nil {
			t.Fatal(err)
		}
		if doomed(i) {
			if 0 != len(got) {
				t.Fatal(got)
			}
		} else {
			if 1 != len(got) {
				t.Fatal(got)
			}
		}
	}

	printState()
	m.printStats()

	s := m.Stats()
	if n <= s.Live {
		t.Fatal(s.Live)
	}

	if !s.LastRebuilt.After(then) {
		t.Fatal(s.LastRebuilt)
	}

	if 0 == s.RebuildPurged {
		t.Fatal(s.RebuildPurged)
	}

	if 0 == s.RebuildDuration {
		t.Fatal(s.RebuildDuration)
	}

}

type badState struct {
	err error
}

var badStateErr = fmt.Errorf("BadState can't do anything right")

func (s *badState) Add(x quamina.X, pattern string) error {
	return s.err
}

func (s *badState) Contains(x quamina.X) (bool, error) {
	return false, s.err
}

func (s *badState) Delete(x quamina.X) (int, error) {
	return 0, s.err
}

func (s *badState) Iterate(f func(x quamina.X, pattern string) error) error {
	return s.err
}

func TestBadState(t *testing.T) {
	bad := &badState{
		err: badStateErr,
	}
	m := NewMatcher(bad)

	if err := m.AddPattern(1, `{"likes":["queso"]}`); err == nil {
		t.Fatal("expected error")
	}
	if _, err := m.DeletePattern(1); err == nil {
		t.Fatal("expected error")
	}
	if err := m.Rebuild(false); err == nil {
		t.Fatal("expected error")
	}

	bad.err = nil
	if err := m.AddPattern(1, `{"likes":["queso"]}`); err != nil {
		t.Fatal(err)
	}
	bad.err = badStateErr
	event := `{"likes":"queso"}`
	if _, err := m.MatchesForJSONEvent([]byte(event)); err == nil {
		t.Fatal("expected error")
	}
}

func TestBadPattern(t *testing.T) {
	m := NewMatcher(&badState{})

	if err := m.AddPattern(1, `Dream baby dream`); err == nil {
		t.Fatal("expected error")
	}
}

func TestBadEvent(t *testing.T) {
	m := NewMatcher(&badState{})

	event := `My heart's not in it`
	if _, err := m.MatchesForJSONEvent([]byte(event)); err == nil {
		t.Fatal("expected error")
	}
}

func TestUnsetRebuildTrigger(t *testing.T) {
	m := NewMatcher(&badState{})
	m.rebuildTrigger = nil
	if err := m.maybeRebuild(false); err != nil {
		t.Fatal(err)
	}
}

func TestFlattener(t *testing.T) {
	var (
		m = NewMatcher(nil)
		f = NewFJ(m) // Variation for test coverage.
	)

	if err := m.AddPattern(1, `{"wants":["queso"]}`); err != nil {
		t.Fatal(err)
	}

	fs, err := f.Flatten([]byte(`{"wants":"queso"}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.MatchesForFields(fs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatal(got)
	}
	if got[0] != 1 {
		t.Fatal(got)
	}

}

func TestMultiplePatternsWithSameId(t *testing.T) {
	var (
		m              = NewMatcher(nil)
		id interface{} = 1
	)

	if err := m.AddPattern(id, `{"enjoys":["queso"]}`); err != nil {
		t.Fatal(err)
	}

	if err := m.AddPattern(id, `{"needs":["chips"]}`); err != nil {
		t.Fatal(err)
	}

	if err := m.Rebuild(false); err != nil {
		t.Fatal(err)
	}

	// If we weren't able to remember that both patterns are still
	// live, then one of the two checks below will fail.  In that
	// case, we can't tell which one in advance (because Go map
	// iteration order is not specified).

	xs, err := m.MatchesForJSONEvent([]byte(`{"enjoys":"queso"}`))

	check := func() {
		if err != nil {
			t.Fatal(err)
		}
		if len(xs) != id {
			t.Fatal(xs)
		}
		if xs[0] != id {
			t.Fatal(xs)
		}
	}

	check()

	xs, err = m.MatchesForJSONEvent([]byte(`{"needs":"chips"}`))

	check()

	s := m.Stats()

	if s.Live != 2 {
		t.Fatal(s.Live)
	}

	if had, err := m.DeletePattern(id); err != nil {
		t.Fatal(err)
	} else if !had {
		t.Fatal(err)
	}

	s = m.Stats()

	if s.Live != 0 {
		t.Fatal(s.Live)
	}

	if s.Deleted != 2 {
		t.Fatal(s.Deleted)
	}

}
