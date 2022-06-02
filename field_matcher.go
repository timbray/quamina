package quamina

import "sync/atomic"

// fieldMatcher represents a state in the matching automaton, which matches field names and dispatches to
//  valueMatcher to complete matching of field values.  fieldMatcher has a map which is keyed by the
//  field pathSegments values that can start transitions from this matcher; for each such field, there is a
//  valueMatcher which, given the field's value, determines whether the automaton progresses to another fieldMatcher
// matches contains the X values that arrival at this state implies have matched
// existsFalseFailures reports the condition that traversal has occurred by matching a field which is named in an
//  exists:false pattern, and the named X's should be subtracted from the matches list being built up by a match project
// the fields that hold state are segregated in updateable so they can be replaced atomically and make the matcher
//  thread-safe.
type fieldMatcher struct {
	updateable atomic.Value // always holds an *fmFields
}
type fmFields struct {
	transitions         map[string]*valueMatcher
	matches             []X
	existsFalseFailures *matchSet
}

// fields / update / addExistsFalseFailure / addMatch exist to insuleate callers from dealing with
//  the atomic Load/Store business
func (m *fieldMatcher) fields() *fmFields {
	return m.updateable.Load().(*fmFields)
}

func (m *fieldMatcher) update(fields *fmFields) {
	m.updateable.Store(fields)
}

func (m *fieldMatcher) addExistsFalseFailure(x X) {
	current := m.fields()
	newFields := &fmFields{
		transitions:         current.transitions,
		matches:             current.matches,
		existsFalseFailures: current.existsFalseFailures.addX(x),
	}
	m.update(newFields)
}

func (m *fieldMatcher) addMatch(x X) {
	current := m.fields()
	newFields := &fmFields{
		transitions:         current.transitions,
		existsFalseFailures: current.existsFalseFailures,
	}
	newFields.matches = append(newFields.matches, current.matches...)
	newFields.matches = append(newFields.matches, x)
	m.update(newFields)
}

func newFieldMatcher() *fieldMatcher {
	fields := &fmFields{transitions: make(map[string]*valueMatcher), existsFalseFailures: newMatchSet()}
	fm := &fieldMatcher{}
	fm.updateable.Store(fields)
	return fm
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

	freshStart.matches = append(freshStart.matches, current.matches...)
	freshStart.existsFalseFailures = current.existsFalseFailures

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
//  or nil if no transitions are possible.  An example of name/value that could produce multiple next states
//  would be if you had the pattern { "a": [ "foo" ] } and another pattern that matched any value with
//  a prefix of "f".
func (m *fieldMatcher) transitionOn(field *Field) []*fieldMatcher {
	// are there transitions on this field name?
	valMatcher, ok := m.fields().transitions[string(field.Path)]
	if !ok {
		return nil
	}

	return valMatcher.transitionOn(field.Val)
}
