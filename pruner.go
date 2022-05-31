package quamina

import (
	"sync"
	"time"
)

// Stats reports basic counts to aid in deciding when to Rebuild.
type Stats struct {
	// Some of these values are ints instead of uints because Go
	// likes to print uints in hex, and I'd like to see some
	// temporary logging output in decimal.  ToDo: back to uints?

	// Live the count of total live patterns.
	Live int

	// Added is the count of total patterns added.
	Added int

	// Deleted is the count of the patterns removed.
	Deleted int

	// Emitted is the count to total patterns found since the last
	// rebuild.
	Emitted int64

	// Filtered is the count of pattners that have been removed
	// from MatchFor results (since the last rebuild) because
	// their patterns had been removed.
	Filtered int64

	// LastRebuilt is the time the last rebuild started.
	LastRebuilt time.Time

	// RebuildDuration is the duration of the last rebuild.
	RebuildDuration time.Duration

	// RebuildPurged is the count of patterns removed during
	// rebuild.
	RebuildPurged int
}

// PrunerMatcher provides DeletePattern on top of quamina.PrunerMatcher.
//
// PrunerMatcher maintains the set of live patterns, and it will rebuild the
// underlying matcher synchronously periodically during standard
// operations (AddPattern, DeletePattern, MatchesForFields).
//
// Roughly speaking, the current rebuild policy automatically rebuilds
// the index when the ratio of filtered patterns to emitted patterns
// exceeds 0.2 (and if there's been some traffic).
//
// An application can call Rebuild to force a rebuild at any time.
// See Stats() to obtain some useful statistics about the matcher.
//
// Eventually automatically-invoked rebuild policies might be
// pluggable.
type PrunerMatcher struct {
	// Matcher is the underlying matcher that does the hard work.
	Matcher *CoreMatcher

	// Maybe PrunerMatcher should maybe not be embedded or public.

	// live is live set of patterns.
	live LivePatternsState

	stats Stats

	// rebuildTrigger, if not nil, determines when a mutation
	// triggers a rebuild.
	//
	// If nil, no automatic Rebuild is ever triggered.
	rebuildTrigger rebuildTrigger

	// lock protectes the pointer the underlying Matcher as well as stats.
	//
	// The Matcher pointer is updated after a successful Rebuild.
	// Stats are updated by Add, Delete, and Rebuild.
	lock sync.RWMutex
}

func (m *PrunerMatcher) IsNameUsed(label []byte) bool {
	return m.Matcher.IsNameUsed(label)
}

var defaultRebuildTrigger = newTooMuchFiltering(0.2, 1000)

// tooMuchFiltering is the standard rebuildTrigger, which will fire
// when:
//
//   MinAction is less than the sum of counts of found and filtered
//   patterns and
//
//   FilteredToEmitted is greater than the ratio of counts of filtered
//   and found patterns.
//
// defaultRebuildTrigger provides the default trigger policy used by
// NewPrunerMatcher.
type tooMuchFiltering struct {
	FilteredToEmitted float64
	MinAction         int64
}

func newTooMuchFiltering(ratio float64, min int64) *tooMuchFiltering {
	return &tooMuchFiltering{
		FilteredToEmitted: ratio,
		MinAction:         min,
	}
}

func (t *tooMuchFiltering) Rebuild(added bool, s *Stats) bool {

	if added {
		// No need to think when we're adding a pattern since
		// that operation cannot result in an increase of
		// filtered patterns.
		return false
	}

	// If we haven't seen enough patterns emitted by the core
	// PrunerMatcher, don't rebuild.
	if s.Emitted+s.Filtered < t.MinAction {
		return false
	}

	// We won't rebuild if nothing's been emitted yet.
	//
	// In isolation, this heuristic is arguable, but for this
	// policy we need it.  Otherwise we'll divide by zero, and
	// nobody wants that.
	if s.Emitted == 0 {
		return false
	}

	var (
		numerator   = float64(s.Filtered)
		denominator = float64(s.Emitted)
		ratio       = numerator / denominator
	)

	return t.FilteredToEmitted < ratio
}

// DisableRebuild will prevent any automatic rebuilds.
func (m *PrunerMatcher) DisableRebuild() {
	m.lock.Lock()
	m.rebuildTrigger = nil
	m.lock.Unlock()
}

// rebuildTrigger provides a way to control when rebuilds are
// automatically triggered during standard operations.
//
// Currently an AddPattern, DeletePattern, or MatchesForFields can
// trigger a rebuild.  When a rebuild is triggered, it's executed
// synchronously: the the Add/Delete/Match method doesn't return until
// the rebuild is complete.
type rebuildTrigger interface {
	// Rebuild should return true to trigger a rebuild.
	//
	// This method is called by AddPatter,DeletePattern, and
	// MatchesForFields.  added is true when called by AddPattern;
	// false otherwise. These methods currently do not return
	// until the rebuild is complete, so beware.
	Rebuild(added bool, s *Stats) bool
}

// NewPrunerMatcher does what you'd expect.
//
// The LivePatternsState defaults to MemState.
func NewPrunerMatcher(s LivePatternsState) *PrunerMatcher {
	if s == nil {
		s = NewMemState()
	}
	trigger := *defaultRebuildTrigger // Copy
	return &PrunerMatcher{
		Matcher:        NewCoreMatcher(),
		live:           s,
		rebuildTrigger: &trigger,
	}
}

// maybeRebuild calls rebuildTrigger and calls rebuild() if that
// trigger said to do that.  If rebuildTrigger is nil, no rebuild is
// executed.
//
// This method assumes the caller has a write lock.
func (m *PrunerMatcher) maybeRebuild(added bool) error {
	if m.rebuildTrigger == nil {
		return nil
	}
	if m.rebuildTrigger.Rebuild(added, &m.stats) {
		return m.rebuild(added)
	}

	return nil
}

// AddPattern calls the underlying quamina.CoreMatcher.AddPattern
// method and then maybe rebuilds the index (if the AddPattern
// succeeded).
func (m *PrunerMatcher) AddPattern(x X, pat string) error {
	var err error

	// Do we m.live.Add first or do we m.PrunerMatcher.AddPattern first?
	if err = m.Matcher.AddPattern(x, pat); err == nil {
		m.lock.Lock()
		m.stats.Added++
		m.stats.Live++
		_ = m.maybeRebuild(true)
		m.lock.Unlock()
		err = m.live.Add(x, pat)
		// ToDo: Contemplate what do to about an error here
		// (or if we got an error from AddPattern after we did
		// live.Add.
	}

	return err
}

// MatchesForJSONEvent calls MatchesForFields with a new Flattener.
func (m *PrunerMatcher) MatchesForJSONEvent(event []byte) ([]X, error) {
	fs, err := NewFJ().Flatten(event, m)
	if err != nil {
		return nil, err
	}
	return m.MatchesForFields(fs)
}

// MatchesForFields calls the underlying
// quamina.CoreMatcher.MatchesForFields and then maybe rebuilds the
// index.
func (m *PrunerMatcher) MatchesForFields(fields []Field) ([]X, error) {

	xs, err := m.Matcher.MatchesForFields(fields)
	if err != nil {
		return nil, err
	}

	// Remove any X that isn't in the live set.

	acc := make([]X, 0, len(xs))

	var emitted, filtered int64
	for _, x := range xs {
		have, err := m.live.Contains(x)
		if err != nil {
			return nil, err
		}
		if !have {
			filtered++
			continue
		}
		acc = append(acc, x)
		emitted++
	}

	m.lock.Lock()
	m.stats.Filtered += filtered
	m.stats.Emitted += emitted
	_ = m.maybeRebuild(false)
	m.lock.Unlock()

	return acc, nil
}

// DeletePattern removes the pattern from the index and maybe rebuilds
// the index.
func (m *PrunerMatcher) DeletePattern(x X) error {
	n, err := m.live.Delete(x)
	if err == nil {
		if 0 < n {
			m.lock.Lock()
			m.stats.Deleted += n
			m.stats.Live -= n
			_ = m.maybeRebuild(false)
			m.lock.Unlock()
		}
	}

	return err
}

// Rebuild rebuilds the matcher state based on only live patterns.
//
// If calling fearlessly, then the old matcher is released before
// building the new one.
//
// This method resets the Stats.
func (m *PrunerMatcher) Rebuild(fearlessly bool) error {
	m.lock.Lock()
	err := m.rebuild(fearlessly)
	m.lock.Unlock()
	return err
}

// rebuild is Rebuild but assumes having the lock.
func (m *PrunerMatcher) rebuild(fearlessly bool) error {
	// We assume we have the lock.

	// Nothing fancy here now.

	var (
		then = time.Now()
		m1   = NewCoreMatcher()
	)

	if fearlessly {
		// Let the GC reduce heap requirements?
		m.Matcher = nil
	}

	count := 0
	err := m.live.Iterate(func(x X, p string) error {
		err := m1.AddPattern(x, p)
		if err == nil {
			count++
		}
		return err
	})

	if err == nil {
		m.Matcher = m1
		m.stats.RebuildPurged = m.stats.Deleted
		m.stats.Live = count
		m.stats.Added = 0
		m.stats.Deleted = 0
		m.stats.Filtered = 0
		m.stats.LastRebuilt = then
		m.stats.RebuildDuration = time.Since(then)
	}

	return err
}

// Stats returns some statistics that might be helpful to rebuild
// policies.
func (m *PrunerMatcher) Stats() Stats {
	m.lock.RLock()
	s := m.stats // Copies
	m.lock.RUnlock()
	return s
}
