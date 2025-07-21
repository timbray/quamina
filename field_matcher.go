package quamina

import (
	"sync/atomic"
)

// fieldMatcher represents a state in the matching automaton, which matches field names and dispatches to
// valueMatcher to complete matching of field values.
// the fields that hold state are segregated in updateable, so they can be replaced atomically and make the coreMatcher
// thread-safe.
type fieldMatcher struct {
	updateable atomic.Pointer[fmFields]
}

// fmFields contains the updateable fields in fieldMatcher.
// transitions is a map keyed by the field paths that can start transitions from this state; for each such field,
// there is a valueMatcher which, given the field's value, determines whether the automaton progresses to another
// fieldMatcher.
// matches contains the X values that arrival at this state implies have matched.
// existsTrue and existsFalse record those types of patterns; traversal doesn't require looking at a valueMatcher
type fmFields struct {
	transitions map[string]*valueMatcher
	matches     []X
	existsTrue  map[string]*fieldMatcher
	existsFalse map[string]*fieldMatcher
}

// fields / update / addExistsFalseFailure / addMatch exist to insulate callers from dealing with
// the atomic Load/Store business
func (m *fieldMatcher) fields() *fmFields {
	return m.updateable.Load()
}

func (m *fieldMatcher) update(fields *fmFields) {
	m.updateable.Store(fields)
}

func (m *fieldMatcher) addMatch(x X) {
	current := m.fields()
	newFields := &fmFields{
		transitions: current.transitions,
		existsTrue:  current.existsTrue,
		existsFalse: current.existsFalse,
	}

	newFields.matches = append(newFields.matches, current.matches...)
	newFields.matches = append(newFields.matches, x)
	m.update(newFields)
}

func newFieldMatcher() *fieldMatcher {
	fields := &fmFields{
		transitions: make(map[string]*valueMatcher),
		existsTrue:  make(map[string]*fieldMatcher),
		existsFalse: make(map[string]*fieldMatcher),
	}
	fm := &fieldMatcher{}
	fm.updateable.Store(fields)
	return fm
}

func (m *fieldMatcher) addExists(exists bool, field *patternField) []*fieldMatcher {
	var trans *fieldMatcher
	current := m.fields()
	freshStart := &fmFields{
		transitions: current.transitions,
		matches:     current.matches,
		existsTrue:  make(map[string]*fieldMatcher),
		existsFalse: make(map[string]*fieldMatcher),
	}
	var path string
	for path, trans = range current.existsTrue {
		freshStart.existsTrue[path] = trans
	}
	for path, trans = range current.existsFalse {
		freshStart.existsFalse[path] = trans
	}
	var ok bool
	if exists {
		trans, ok = freshStart.existsTrue[field.path]
		if !ok {
			trans = newFieldMatcher()
			freshStart.existsTrue[field.path] = trans
		}
	} else {
		trans, ok = freshStart.existsFalse[field.path]
		if !ok {
			trans = newFieldMatcher()
			freshStart.existsFalse[field.path] = trans
		}
	}
	m.update(freshStart)
	return []*fieldMatcher{trans}
}

func (m *fieldMatcher) addTransition(field *patternField, printer printer) []*fieldMatcher {
	// we build the new updateable state in freshStart so that we can blast it in atomically once computed
	current := m.fields()
	freshStart := &fmFields{
		matches:     current.matches,
		existsTrue:  current.existsTrue,
		existsFalse: current.existsFalse,
	}

	freshStart.transitions = make(map[string]*valueMatcher)
	for k, v := range current.transitions {
		freshStart.transitions[k] = v
	}
	vm, ok := freshStart.transitions[field.path]
	if !ok {
		vm = newValueMatcher()
	}
	freshStart.transitions[field.path] = vm

	// suppose I'm adding the first pattern to a matcher, and it has "x": [1, 2]. In principle the branches on
	//  "x": 1 and "x": 2 could go to tne same next state. But we have to make a unique next state for each of them
	//  because some future other pattern might have "x": [2, 3] and thus we need a separate branch to potentially
	//  match two patterns on "x": 2 but not "x": 1. If you were optimizing the automaton for size you might detect
	//  cases where this doesn't happen and reduce the number of fieldMatchStates
	var nextFieldMatchers []*fieldMatcher
	for _, val := range field.vals {
		nextFieldMatchers = append(nextFieldMatchers, vm.addTransition(val, printer))
	}
	m.update(freshStart)
	return nextFieldMatchers
}

// transitionOn returns one or more fieldMatchStates you can transition to on a field's name/value combination,
// or nil if no transitions are possible.  An example of name/value that could produce multiple next states
// would be if you had the pattern { "a": [ "foo" ] } and another pattern that matched any value with
// a prefix of "f".
func (m *fieldMatcher) transitionOn(field *Field, bufs *nfaBuffers) []*fieldMatcher {
	// are there transitions on this field name?
	valMatcher, ok := m.fields().transitions[string(field.Path)]
	if !ok {
		return nil
	}
	return valMatcher.transitionOn(field, bufs)
}
