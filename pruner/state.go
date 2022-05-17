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
	//
	// Note that multiple patterns can be associated with the same
	// X.
	Add(x quamina.X, pattern string) error

	Delete(x quamina.X) (bool, error)

	// Iterate calls the given function for every stored pattern.
	Iterate(func(x quamina.X, pattern string) error) error

	// Contains returns true if x is in the live set; false
	// otherwise.
	//
	// Since a pattern can't be the empty string, that zero value
	// indicates no corresponding pattern.
	Contains(x quamina.X) (bool, error)
}

type (
	stringSet map[string]nothing
	nothing   struct{}
)

var na = nothing{}

// MemState is a State that is just a map (with a RWMutex).
//
// Since the State implementation can be provided to the
// application, we're keeping things simple here initially.
type MemState struct {
	lock sync.RWMutex
	m    map[quamina.X]stringSet
}

func NewMemState() *MemState {
	// Accept initial size as a parameter?
	return &MemState{
		m: make(map[quamina.X]stringSet),
	}
}

var ErrExists = fmt.Errorf("pattern already exists for that X")

func (s *MemState) Add(x quamina.X, pattern string) error {
	s.lock.Lock()
	ps, have := s.m[x]
	if !have {
		ps = make(stringSet)
		s.m[x] = ps
	}
	ps[pattern] = na
	s.lock.Unlock()
	return nil
}

func (s *MemState) Contains(x quamina.X) (bool, error) {
	s.lock.RLock()
	_, have := s.m[x]
	s.lock.RUnlock()
	return have, nil
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
	for x, ps := range s.m {
		for p := range ps {
			if err = f(x, p); err != nil {
				break
			}
		}
	}
	s.lock.RUnlock()
	return err
}
