package pruner

import (
	"fmt"
	"sync"

	quamina "quamina/lib"
)

// State represents the required capabilities for maintaining the set
// of live patterns.
type State interface {
	// Add adds a new pattern or updates an old pattern.
	Add(x quamina.X, pattern string) error

	Delete(x quamina.X) (bool, error)

	// Iterate calls the given function for every stored pattern.
	Iterate(func(x quamina.X, pattern string) error) error

	// Get returns the pattern for the given X.
	//
	// Since a pattern can't be the empty string, that zero value
	// indicates no corresponding pattern.
	Get(x quamina.X) (string, error)
}

// MemState is a State that is just a map (with a RWMutex).
//
// Since the State implementation can be provided to the
// application, we're keeping things simple here initially.
type MemState struct {
	lock sync.RWMutex
	m    map[quamina.X]string
}

func NewMemState() *MemState {
	// Accept initial size as a parameter?
	return &MemState{
		m: make(map[quamina.X]string),
	}
}

var ErrExists = fmt.Errorf("pattern already exists for that X")

func (s *MemState) Add(x quamina.X, pattern string) error {
	s.lock.Lock()
	// We don't care if the X is already there.
	s.m[x] = pattern
	s.lock.Unlock()
	return nil
}

func (s *MemState) Get(x quamina.X) (string, error) {
	s.lock.RLock()
	p := s.m[x]
	s.lock.RUnlock()
	return p, nil
}

func (s *MemState) Delete(x quamina.X) (bool, error) {
	s.lock.Lock()
	_, had := s.m[x]
	if had {
		delete(s.m, x)
	}
	s.lock.Unlock()

	return had, nil
}

func (s *MemState) Iterate(f func(x quamina.X, pattern string) error) error {
	s.lock.RLock()
	var err error
	for x, p := range s.m {
		if err = f(x, p); err != nil {
			break
		}
	}
	s.lock.RUnlock()
	return err
}
