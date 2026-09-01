package quamina

import (
	"sync"
	"sync/atomic"
	"time"
)

type prunerFields struct {
	// Matcher is the underlying matcher that does the hard work.
	Matcher *coreMatcher

	// Maybe prunerMatcher should maybe not be embedded or public.

	// live is the live set of patterns.
	live LivePatternsState

	// rebuildTrigger, if not nil, determines when a mutation
	// triggers a rebuildWhileLocked.
	//
	// If nil, no automatic rebuild is ever triggered.
	rebuildTrigger rebuildTrigger
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
	updateable atomic.Pointer[prunerFields]
	stats      *asyncPrunerStats

	lock sync.RWMutex
}

func (pm *prunerMatcher) loadFieldsForUpdate() *prunerFields {
	currentFields := pm.updateable.Load()
	newFields := *currentFields
	return &newFields
}
func (pm *prunerMatcher) loadFields() *prunerFields {
	currentFields := pm.updateable.Load()
	return currentFields
}
func (pm *prunerMatcher) storeFields(fields *prunerFields) {
	pm.updateable.Store(fields)
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

func newTooMuchFiltering(ratio float64, minimum int64) *tooMuchFiltering {
	return &tooMuchFiltering{
		FilteredToEmitted: ratio,
		MinAction:         minimum,
	}
}

// TODO: Figure out how to expose this through the Quamina type
func (t *tooMuchFiltering) rebuild(added bool, s *asyncPrunerStats) bool {
	if added {
		// No need to think when we're adding a pattern since
		// that operation cannot result in an increase of
		// filtered patterns.
		return false
	}

	// If we haven't seen enough patterns emitted by the core
	// prunerMatcher, don't rebuildWhileLocked.

	if s.Emitted.get()+s.Filtered.get() < t.MinAction {
		return false
	}

	// We won't rebuildWhileLocked if nothing's been emitted yet.
	//
	// In isolation, this heuristic is arguable, but for this
	// policy we need it. Otherwise, we'll divide by zero, and
	// nobody wants that.
	if s.Emitted.get() == 0 {
		return false
	}

	var (
		numerator   = float64(s.Filtered.get())
		denominator = float64(s.Emitted.get())
		ratio       = numerator / denominator
	)
	return t.FilteredToEmitted < ratio
}

// disableRebuild will prevent any automatic rebuilds.
func (pm *prunerMatcher) disableRebuild() {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	freshStart := *pm.loadFieldsForUpdate()
	freshStart.rebuildTrigger = nil
	pm.storeFields(&freshStart)
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
	// This method is called by AddPattern, deletePatterns, and
	// matchesForFields.  added is true when called by addPattern;
	// false otherwise. These methods currently do not return
	// until the rebuild is complete, so beware.
	rebuild(added bool, s *asyncPrunerStats) bool
}

// newPrunerMatcher does what you'd expect.
//
// The LivePatternsState defaults to memState.
func newPrunerMatcher(s LivePatternsState) *prunerMatcher {
	if s == nil {
		s = newMemState()
	}
	trigger := *defaultRebuildTrigger // Copy
	matcher := &prunerMatcher{stats: newAsyncPrunerStats()}
	fields := &prunerFields{
		Matcher:        newCoreMatcher(),
		live:           s,
		rebuildTrigger: &trigger,
	}
	matcher.storeFields(fields)
	return matcher
}

// maybeRebuild calls rebuildTrigger and calls rebuildWhileLocked() if that
// trigger said to do that.  If rebuildTrigger is nil, no rebuildWhileLocked is
// executed.
//
// This method assumes the caller has a write lock.
func (pm *prunerMatcher) maybeRebuild(added bool) error {
	pmFields := pm.loadFields()
	if pmFields.rebuildTrigger == nil {
		return nil
	}
	if pmFields.rebuildTrigger.rebuild(added, pm.stats) {
		return pm.rebuildWhileLocked(added)
	}
	return nil
}

// addPattern calls the underlying quamina.coreMatcher.addPattern
// method and then maybe rebuilds the index (if the addPattern
// succeeded).
func (pm *prunerMatcher) addPattern(x X, pat string, buildMode MatcherBuildMode) error {
	var err error

	// Do we m.live.Add first or do we m.prunerMatcher.addPattern first?
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pmFields := pm.loadFields()
	if err = pmFields.Matcher.addPattern(x, pat, buildMode); err == nil {
		pm.stats.Added.bump(1)
		pm.stats.Live.bump(1)
		_ = pm.maybeRebuild(true)
		err = pmFields.live.Add(x, pat, buildMode)
		// ToDo: Contemplate what do to about an error here
		// (or if we got an error from addPattern after we did
		// live.Add.
	}

	return err
}

// MatchesForJSONEvent calls MatchesForFields with a new Flattener.
func (pm *prunerMatcher) MatchesForJSONEvent(event []byte) ([]X, error) {
	pmFields := pm.loadFields()
	fs, err := newJSONFlattener().Flatten(event, pmFields.Matcher.fields().segmentsTree)
	if err != nil {
		return nil, err
	}
	return pm.matchesForFields(fs, newNfaBuffers())
}

func (pm *prunerMatcher) getStats() *matcherStats {
	pmFields := pm.loadFields()
	return pmFields.Matcher.getStats()
}

// MatchesForFields calls the underlying
// quamina.coreMatcher.matchesForFields and then maybe rebuilds the
// index.
func (pm *prunerMatcher) matchesForFields(fields []Field, bufs *nfaBuffers) ([]X, error) {
	pmFields := pm.loadFields()
	xs, err := pmFields.Matcher.matchesForFields(fields, bufs)
	if err != nil {
		return nil, err
	}

	// Remove any X that isn't in the live set.

	acc := make([]X, 0, len(xs))

	var emitted, filtered int64
	for _, x := range xs {
		have, err := pmFields.live.Contains(x)
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

	pm.stats.Filtered.bump(filtered)
	pm.stats.Emitted.bump(emitted)
	_ = pm.maybeRebuild(false)

	return acc, nil
}

// DeletePattern removes the pattern from the index and maybe rebuilds
// the index.
func (pm *prunerMatcher) deletePatterns(x X) error {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pmFields := pm.loadFieldsForUpdate()
	n, err := pmFields.live.Delete(x)
	if err == nil {
		if 0 < n {
			pm.stats.Deleted.bump(int64(n))
			pm.stats.Live.bump(int64(-n))
			_ = pm.maybeRebuild(false)
			pm.storeFields(pmFields)
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
func (pm *prunerMatcher) rebuild(fearlessly bool) error {
	pm.lock.Lock()
	err := pm.rebuildWhileLocked(fearlessly)
	pm.lock.Unlock()
	return err
}

// rebuildWhileLocked is rebuild but assumes having the lock.
func (pm *prunerMatcher) rebuildWhileLocked(fearlessly bool) error {
	// We assume we have the lock.

	// Nothing fancy here now.

	var (
		then = time.Now()
		m1   = newCoreMatcher()
	)

	pmFields := pm.loadFieldsForUpdate()
	if fearlessly {
		// Let the GC reduce heap requirements?
		pmFields.Matcher = nil
	}

	count := 0
	err := pmFields.live.Iterate(func(x X, p string, buildMode MatcherBuildMode) error {
		err := m1.addPattern(x, p, buildMode)
		if err == nil {
			count++
		}
		return err
	})

	if err == nil {
		pmFields.Matcher = m1
		pm.stats.RebuildPurged.set(pm.stats.Deleted.get())
		pm.stats.Live.set(int64(count))
		pm.stats.Added.set(0)
		pm.stats.Deleted.set(0)
		pm.stats.Filtered.set(0)
		pm.stats.LastRebuilt.set(then)
		pm.stats.RebuildDuration.set(time.Since(then))
	}
	pm.storeFields(pmFields)

	return err
}

// prunerStats returns some statistics that might be helpful to rebuildWhileLocked
// policies.
func (pm *prunerMatcher) getPrunerStats() *asyncPrunerStats {
	return pm.stats
}

func (pm *prunerMatcher) getSegmentsTreeTracker() SegmentsTreeTracker {
	pmFields := pm.loadFields()
	return pmFields.Matcher.getSegmentsTreeTracker()
}
