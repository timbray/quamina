package pruner

import (
	"sync"
	"time"

	quamina "github.com/timbray/quamina/core"
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

// Matcher provides DeletePattern on top of quamina.Matcher.
//
// Matcher maintains the set of live patterns, and it will rebuild the
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
type Matcher struct {
	// Matcher is the underlying matcher that does the hard work.
	Matcher *quamina.CoreMatcher

	// Maybe Matcher should maybe not be embedded or public.

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
// NewMatcher.
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
	// Matcher, don't rebuild.
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
func (m *Matcher) DisableRebuild() {
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

// NewMatcher does what you'd expect.
//
// The LivePatternsState defaults to MemState.
func NewMatcher(s LivePatternsState) *Matcher {
	if s == nil {
		s = NewMemState()
	}
	trigger := *defaultRebuildTrigger // Copy
	return &Matcher{
		Matcher:        quamina.NewCoreMatcher(),
		live:           s,
		rebuildTrigger: &trigger,
	}
}

// maybeRebuild calls rebuildTrigger and calls rebuild() if that
// trigger said to do that.  If rebuildTrigger is nil, no rebuild is
// executed.
//
// This method assumes the caller has a write lock.
func (m *Matcher) maybeRebuild(added bool) error {
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
func (m *Matcher) AddPattern(x quamina.X, pat string) error {
	var err error

	// Do we m.live.Add first or do we m.Matcher.AddPattern first?
	if err = m.Matcher.AddPattern(x, pat); err == nil {
		m.lock.Lock()
		m.stats.Added++
		m.stats.Live++
		m.maybeRebuild(true)
		m.lock.Unlock()
		err = m.live.Add(x, pat)
		// ToDo: Contemplate what do to about an error here
		// (or if we got an error from AddPattern after we did
		// live.Add.
	}

	return err
}

// NewFJ just calls quamina.FJ.
//
// Here for convenience only.
func NewFJ(m *Matcher) quamina.Flattener {
	return quamina.NewFJ(m.Matcher)
}

// NewFJ calls quamina.NewFJ with this Matcher's core quamina.Matcher
//
// Here for convenience only.
func (m *Matcher) NewFJ() quamina.Flattener {
	return quamina.NewFJ(m.Matcher)
}

// MatchesForJSONEvent calls MatchesForFields with a new Flattener.
func (m *Matcher) MatchesForJSONEvent(event []byte) ([]quamina.X, error) {
	fs, err := m.NewFJ().Flatten(event)
	if err != nil {
		return nil, err
	}
	return m.MatchesForFields(fs)
}

// MatchesForFields calls the underlying
// quamina.CoreMatcher.MatchesForFields and then maybe rebuilds the
// index.
func (m *Matcher) MatchesForFields(fields []quamina.Field) ([]quamina.X, error) {

	xs, err := m.Matcher.MatchesForFields(fields)
	if err != nil {
		return nil, err
	}

	// Remove any X that isn't in the live set.

	acc := make([]quamina.X, 0, len(xs))

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
	m.maybeRebuild(false)
	m.lock.Unlock()

	return acc, nil
}

// DeletePattern removes the pattern from the index and maybe rebuilds
// the index.
func (m *Matcher) DeletePattern(x quamina.X) error {
	n, err := m.live.Delete(x)
	if err == nil {
		if 0 < n {
			m.lock.Lock()
			m.stats.Deleted += n
			m.stats.Live -= n
			m.maybeRebuild(false)
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
func (m *Matcher) Rebuild(fearlessly bool) error {
	m.lock.Lock()
	err := m.rebuild(fearlessly)
	m.lock.Unlock()
	return err
}

// rebuild is Rebuild but assumes having the lock.
func (m *Matcher) rebuild(fearlessly bool) error {
	// We assume we have the lock.

	// Nothing fancy here now.

	var (
		then = time.Now()
		m1   = quamina.NewCoreMatcher()
	)

	if fearlessly {
		// Let the GC reduce heap requirements?
		m.Matcher = nil
	}

	count := 0
	err := m.live.Iterate(func(x quamina.X, p string) error {
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
		m.stats.RebuildDuration = time.Now().Sub(then)
	}

	return err
}

// Stats returns some statistics that might be helpful to rebuild
// policies.
func (m *Matcher) Stats() Stats {
	m.lock.RLock()
	s := m.stats // Copies
	m.lock.RUnlock()
	return s
}
