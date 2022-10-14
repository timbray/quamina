package quamina

import (
	"sync/atomic"
)

// The only thing that's complicated here is exists:false matching. Here's an explanation by example.
// If we have the pattern {"x": [ { "exists": false } ] }
// Then as we process the fields in addPattern, in the FieldMatcher for the field before "x" we put a
// notice that an exists:false on "x" is pending and if it matches, here's another fieldMatcher to transition to.
// Then in the match code, when we hit that state we remember the pending exists:false matches in a list that
// gets passed along from field to field.
// Whenever we arrive at a new field, if its path is "x", the exists:false failed, and we can remove it from the
// pending list. If its path is something like "z", i.e. lexically greater than "x", then because the fields are
// sorted we can tell that we won't be seeing an "x" path, i.e. the exists:false matched, so we can go ahead and
// transition to the one in the pending list, and once again remove the "x" from the pending list.  If the path
// is something like "k", lexically less than "x", we haven't learned anything and we keep entry for "x"
// in the pending list. (Exercise for the reader: How can that happen?)
// There are more details, and more commentary in the relevant code.

// fieldMatcher represents a state in the matching automaton, which matches field names and dispatches to
// valueMatcher to complete matching of field values.
// the fields that hold state are segregated in updateable so they can be replaced atomically and make the matcher
// thread-safe.
type fieldMatcher struct {
	updateable atomic.Value // always holds an *fmFields
}

// fmFields groups the updateable fields in fieldMatcher.
// transitions is a map keyed by the field paths that can start transitions from this state; for each such field,
// there is a valueMatcher which, given the field's value, determines whether the automaton progresses to another
// fieldMatcher.
// matches contains the X values that arrival at this state implies have matched.
// pendingExistsFalses is keyed by field paths which have appeared in an exists:false pattern where those paths
// are lexically greater than the one whose name/val matched and transitioned to this one.  The values are
// states to transition to in the case that the exists:false matches.
// So when traversing the states, we have to start looking for occurrences of the paths appearing as heys here;
// if we arrive at a state with one of those paths, the exists:false failed, but if we arrive at a state whose
// path is lexically greater than one of these, the exists:false will have matched.
type fmFields struct {
	transitions         map[string]*valueMatcher
	matches             []X
	pendingExistsFalses map[string]*fieldMatcher
}

// fields / update / addExistsFalseFailure / addMatch exist to insulate callers from dealing with
// the atomic Load/Store business
func (m *fieldMatcher) fields() *fmFields {
	return m.updateable.Load().(*fmFields)
}

func (m *fieldMatcher) update(fields *fmFields) {
	m.updateable.Store(fields)
}

func (m *fieldMatcher) addMatch(x X) {
	current := m.fields()
	newFields := &fmFields{
		transitions:         current.transitions,
		pendingExistsFalses: current.pendingExistsFalses,
	}
	newFields.matches = append(newFields.matches, current.matches...)
	newFields.matches = append(newFields.matches, x)
	m.update(newFields)
}

func newFieldMatcher() *fieldMatcher {
	fields := &fmFields{
		transitions:         make(map[string]*valueMatcher),
		pendingExistsFalses: make(map[string]*fieldMatcher),
	}
	fm := &fieldMatcher{}
	fm.updateable.Store(fields)
	return fm
}

// addExistsFalseTransition is really different from adding a normal transition. We're just going to work on
// the pendingExistsFalses field. We do *not* make a new valueMatcher or update the transitions field.
func (m *fieldMatcher) addExistsFalseTransition(field *patternField) []*fieldMatcher {
	current := m.fields()
	freshStart := &fmFields{
		transitions: current.transitions,
		matches:     current.matches,
	}
	freshStart.pendingExistsFalses = make(map[string]*fieldMatcher)
	for path, trans := range current.pendingExistsFalses {
		freshStart.pendingExistsFalses[path] = trans
	}

	pending, ok := freshStart.pendingExistsFalses[field.path]
	if !ok {
		pending = newFieldMatcher()
		freshStart.pendingExistsFalses[field.path] = pending
	}
	m.update(freshStart)
	return []*fieldMatcher{pending}
}

func (m *fieldMatcher) addTransition(field *patternField) []*fieldMatcher {
	// we build the new updateable state in freshStart so we can blsat it in atomically once computed
	current := m.fields()
	freshStart := &fmFields{}
	freshStart.transitions = make(map[string]*valueMatcher)
	for k, v := range current.transitions {
		freshStart.transitions[k] = v
	}
	vm, ok := freshStart.transitions[field.path]
	if !ok {
		vm = newValueMatcher()
	}
	freshStart.transitions[field.path] = vm
	freshStart.pendingExistsFalses = current.pendingExistsFalses

	// TODO: pretty sure we can delete the following line, how could current have picked up any new matches?
	freshStart.matches = append(freshStart.matches, current.matches...)

	// suppose I'm adding the first pattern to a matcher and it has "x": [1, 2]. In principle the branches on
	//  "x": 1 and "x": 2 could go to tne same next state. But we have to make a unique next state for each of them
	//  because some future other pattern might have "x": [2, 3] and thus we need a separate branch to potentially
	//  match two patterns on "x": 2 but not "x": 1. If you were optimizing the automaton for size you might detect
	//  cases where this doesn't happen and reduce the number of fieldMatchStates
	var nextFieldMatchers []*fieldMatcher
	for _, val := range field.vals {
		nextFieldMatchers = append(nextFieldMatchers, vm.addTransition(val))
		// if the val is a number, let's add a transition on the canonicalized number
		// TODO: Only do this if asked
		/*
			if val.vType == numberType {
				c, err := canonicalize([]byte(val.val))
				if err == nil {
					number := typedVal{
						vType: literalType,
						val:   c,
					}
					nextFieldMatchers = append(nextFieldMatchers, vm.addTransition(number))
				}
			}
		*/
	}
	m.update(freshStart)
	return nextFieldMatchers
}

// transitionOn returns one or more fieldMatchStates you can transition to on a field's name/value combination,
// or nil if no transitions are possible.  An example of name/value that could produce multiple next states
// would be if you had the pattern { "a": [ "foo" ] } and another pattern that matched any value with
// a prefix of "f".
func (m *fieldMatcher) transitionOn(field *Field) []*fieldMatcher {
	// are there transitions on this field name?
	valMatcher, ok := m.fields().transitions[string(field.Path)]
	if !ok {
		return nil
	}

	return valMatcher.transitionOn(field.Val)
}
