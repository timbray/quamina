package quamina

import (
	"sync"
	"time"
)

// prunerStats reports basic counts to aid in deciding when to rebuildWhileLocked.
type prunerStats struct {
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
	// rebuildWhileLocked.
	Emitted int64

	// Filtered is the count of pattners that have been removed
	// from MatchFor results (since the last rebuildWhileLocked) because
	// their patterns had been removed.
	Filtered int64

	// LastRebuilt is the time the last rebuildWhileLocked started.
	LastRebuilt time.Time

	// RebuildDuration is the duration of the last rebuildWhileLocked.
	RebuildDuration time.Duration

	// RebuildPurged is the count of patterns removed during
	// rebuildWhileLocked.
	RebuildPurged int
}

// prunerMatcher provides DeletePattern on top of quamina.matcher.
//
// prunerMatcher maintains the set of live patterns, and it will rebuildWhileLocked the
// underlying matcher synchronously periodically during standard
// operations (addPattern, DeletePattern, MatchesForFields).
//
// Roughly speaking, the current rebuildWhileLocked policy automatically rebuilds
// the index when the ratio of filtered patterns to emitted patterns
// exceeds 0.2 (and if there's been some traffic).
//
// An application can call rebuild to force a rebuildWhileLocked at any time.
// See prunerStats() to obtain some useful statistics about the matcher.
//
// Eventually automatically-invoked rebuildWhileLocked policies might be
// pluggable.
type prunerMatcher struct {
	// Matcher is the underlying matcher that does the hard work.
	Matcher *coreMatcher

	// Maybe prunerMatcher should maybe not be embedded or public.

	// live is the live set of patterns.
	live LivePatternsState

	stats prunerStats

	// rebuildTrigger, if not nil, determines when a mutation
	// triggers a rebuildWhileLocked.
	//
	// If nil, no automatic rebuild is ever triggered.
	rebuildTrigger rebuildTrigger

	// lock protects the pointer the underlying Matcher as well as stats.
	//
	// The Matcher pointer is updated after a successful rebuild.
	// Stats are updated by Add, Delete, and rebuild.
	lock sync.RWMutex
}

var defaultRebuildTrigger = newTooMuchFiltering(0.2, 1000)

// nolint:gofmt,goimports
// tooMuchFiltering is the standard rebuildTrigger, which will fire
// when:
//
//	MinAction is less than the sum of counts of found and filtered
//	patterns and
//
//	FilteredToEmitted is greater than the ratio of counts of filtered
//	and found patterns.
//
// defaultRebuildTrigger provides the default trigger policy used by
// newPrunerMatcher.
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

// TODO: Figure out how to expose this through the Quamina type
func (t *tooMuchFiltering) rebuild(added bool, s *prunerStats) bool {
	if added {
		// No need to think when we're adding a pattern since
		// that operation cannot result in an increase of
		// filtered patterns.
		return false
	}

	// If we haven't seen enough patterns emitted by the core
	// prunerMatcher, don't rebuildWhileLocked.
	if s.Emitted+s.Filtered < t.MinAction {
		return false
	}

	// We won't rebuildWhileLocked if nothing's been emitted yet.
	//
	// In isolation, this heuristic is arguable, but for this
	// policy we need it. Otherwise, we'll divide by zero, and
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

// disableRebuild will prevent any automatic rebuilds.
func (m *prunerMatcher) disableRebuild() {
	m.lock.Lock()
	m.rebuildTrigger = nil
	m.lock.Unlock()
}

// rebuildTrigger provides a way to control when rebuilds are
// automatically triggered during standard operations.
//
// Currently, an addPattern, deletePatterns, or matchesForFields can
// trigger a rebuild.  When a rebuild is triggered, it's executed
// synchronously: the Add/Delete/Match method doesn't return until
// the rebuild is complete.
type rebuildTrigger interface {
	// rebuild should return true to trigger a rebuild.
	//
	// This method is called by AddPatter,deletePatterns, and
	// matchesForFields.  added is true when called by addPattern;
	// false otherwise. These methods currently do not return
	// until the rebuild is complete, so beware.
	rebuild(added bool, s *prunerStats) bool
}

// newPrunerMatcher does what you'd expect.
//
// The LivePatternsState defaults to memState.
func newPrunerMatcher(s LivePatternsState) *prunerMatcher {
	if s == nil {
		s = newMemState()
	}
	trigger := *defaultRebuildTrigger // Copy
	return &prunerMatcher{
		Matcher:        newCoreMatcher(),
		live:           s,
		rebuildTrigger: &trigger,
	}
}

// maybeRebuild calls rebuildTrigger and calls rebuildWhileLocked() if that
// trigger said to do that.  If rebuildTrigger is nil, no rebuildWhileLocked is
// executed.
//
// This method assumes the caller has a write lock.
func (m *prunerMatcher) maybeRebuild(added bool) error {
	if m.rebuildTrigger == nil {
		return nil
	}
	if m.rebuildTrigger.rebuild(added, &m.stats) {
		return m.rebuildWhileLocked(added)
	}

	return nil
}

// addPattern calls the underlying quamina.coreMatcher.addPattern
// method and then maybe rebuilds the index (if the addPattern
// succeeded).
func (m *prunerMatcher) addPattern(x X, pat string) error {
	var err error

	// Do we m.live.Add first or do we m.prunerMatcher.addPattern first?
	if err = m.Matcher.addPattern(x, pat); err == nil {
		m.lock.Lock()
		m.stats.Added++
		m.stats.Live++
		_ = m.maybeRebuild(true)
		m.lock.Unlock()
		err = m.live.Add(x, pat)
		// ToDo: Contemplate what do to about an error here
		// (or if we got an error from addPattern after we did
		// live.Add.
	}

	return err
}

// MatchesForJSONEvent calls MatchesForFields with a new Flattener.
func (m *prunerMatcher) MatchesForJSONEvent(event []byte) ([]X, error) {
	fs, err := newJSONFlattener().Flatten(event, m.Matcher.fields().segmentsTree)
	if err != nil {
		return nil, err
	}
	return m.matchesForFields(fs)
}

// MatchesForFields calls the underlying
// quamina.coreMatcher.matchesForFields and then maybe rebuilds the
// index.
func (m *prunerMatcher) matchesForFields(fields []Field) ([]X, error) {
	xs, err := m.Matcher.matchesForFields(fields)
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
func (m *prunerMatcher) deletePatterns(x X) error {
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

// rebuild rebuilds the matcher state based on only live patterns.
//
// If calling fearlessly, then the old matcher is released before
// building the new one.
//
// This method resets the prunerStats.
func (m *prunerMatcher) rebuild(fearlessly bool) error {
	m.lock.Lock()
	err := m.rebuildWhileLocked(fearlessly)
	m.lock.Unlock()
	return err
}

// rebuildWhileLocked is rebuild but assumes having the lock.
func (m *prunerMatcher) rebuildWhileLocked(fearlessly bool) error {
	// We assume we have the lock.

	// Nothing fancy here now.

	var (
		then = time.Now()
		m1   = newCoreMatcher()
	)

	if fearlessly {
		// Let the GC reduce heap requirements?
		m.Matcher = nil
	}

	count := 0
	err := m.live.Iterate(func(x X, p string) error {
		err := m1.addPattern(x, p)
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

// prunerStats returns some statistics that might be helpful to rebuildWhileLocked
// policies.
func (m *prunerMatcher) getStats() prunerStats {
	m.lock.RLock()
	s := m.stats // Copies
	m.lock.RUnlock()
	return s
}

func (m *prunerMatcher) getSegmentsTreeTracker() SegmentsTreeTracker {
	return m.Matcher.getSegmentsTreeTracker()
}
