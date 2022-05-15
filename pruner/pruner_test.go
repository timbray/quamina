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

	if err := m.AddPattern(id, pat); err == nil {
		t.Fatal("expected protest")
	}

	got, err := m.MatchesForJSONEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatal(got)
	}

	m.printStats()

	if have, err := m.DelPattern(id); err != nil {
		t.Fatal(err)
	} else if !have {
		t.Fatal(have)
	}
	if have, err := m.DelPattern(id); err != nil {
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
			if had, err := m.DelPattern(i); err != nil {
				t.Fatal(err)
			} else if !had {
				t.Fatal(i)
			}
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

	t.Run("rebuuild", func(t *testing.T) {
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

	t.Run("norebuuild", func(t *testing.T) {
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

}

func TestTriggerTooManyFilteredDenom(t *testing.T) {
	// Verify that a zero denominator doesn't cause problems.
	m := NewMatcher(nil)
	trigger := m.rebuildTrigger.(*tooMuchFiltering)
	trigger.MinAction = 0

	if err := m.AddPattern(1, `{"likes":["tacos"]}`); err != nil {
		t.Fatal(err)
	}
	if _, err := m.DelPattern(1); err != nil {
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
			if _, err := m.DelPattern(i); err != nil {
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

func (s *badState) Get(x quamina.X) (string, error) {
	return "", s.err
}

func (s *badState) Del(x quamina.X) (bool, error) {
	return false, s.err
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
	if _, err := m.DelPattern(1); err == nil {
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
