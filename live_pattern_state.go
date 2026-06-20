package quamina

import (
	"sync"
)

// LivePatternsState represents the required capabilities for maintaining the
// set of live patterns.
type LivePatternsState interface {
	// Add adds a new pattern or updates an old pattern.
	//
	// Note that multiple patterns can be associated with the same X.
	Add(x X, pattern string) error

	// Delete removes all patterns associated with the given X and returns the
	// number of removed patterns.
	Delete(x X) (int, error)

	// Iterate calls the given function for every stored pattern.
	Iterate(func(x X, pattern string, buildMode MatcherBuildMode) error) error

	// Contains returns true if x is in the live set; false otherwise.
	Contains(x X) (bool, error)

	// SetMatcherBuildMode - See the method of the same name in quamina.go
	SetMatcherBuildMode(mode MatcherBuildMode)
}

// memState is a LivePatternsState that is just a slice of buildMode/pattern pairs
//
// Since the LivePatternsState implementation can be provided to the
// application, we're keeping things simple here initially.
type memStateEntry struct {
	x           X
	pattern     string
	builderMode MatcherBuildMode
}
type memState struct {
	lock        sync.RWMutex
	builderMode MatcherBuildMode
	entries     []memStateEntry
}

func newMemState() *memState {
	return &memState{builderMode: BuiltForComfort}
}
func (s *memState) SetMatcherBuildMode(mode MatcherBuildMode) {
	s.builderMode = mode
}
func (s *memState) Add(x X, pattern string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.entries = append(s.entries, memStateEntry{x, pattern, s.builderMode})
	return nil
}
func (s *memState) Delete(x X) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	howMany := 0
	var newEntries []memStateEntry
	for _, entry := range s.entries {
		if entry.x == x {
			howMany++
		} else {
			newEntries = append(newEntries, entry)
		}
	}
	s.entries = newEntries
	return howMany, nil
}

func (s *memState) Contains(x X) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, entry := range s.entries {
		if entry.x == x {
			return true, nil
		}
	}
	return false, nil
}
func (s *memState) Iterate(f func(x X, pattern string, buildMode MatcherBuildMode) error) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	var err error
	for _, entry := range s.entries {
		err = f(entry.x, entry.pattern, s.builderMode)
		if err != nil {
			break
		}
	}
	return err
}
