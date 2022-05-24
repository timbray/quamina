package quamina

// fieldMatcher represents a state in the matching automaton, which matches field names and dispatches to
//  valueMatcher to complete matching of name/value combinations.  fieldMatcher has a map which is keyed by the
//  field pathSegments values that can start transitions from this matcher; for each such field, there is a
//  valueMatcher which, given the field's value, determines whether the automaton progresses to another fieldMatcher
// matches contains the X values that arrival at this state implies have matched
// existsFalseFailures reports the condition that traversal has occurred by matching a field which is named in an
//  exists:false pattern, and the named X's should be subtracted from the matches list being built up by a match project
type fieldMatcher struct {
	transitions         map[string]*valueMatcher
	matches             []X
	existsFalseFailures *matchSet
}

func newFieldMatcher() *fieldMatcher {
	return &fieldMatcher{transitions: make(map[string]*valueMatcher), existsFalseFailures: newMatchSet()}
}

func (m *fieldMatcher) addTransition(field *patternField) []*fieldMatcher {

	// transition from a fieldMatchstate might already be present; create a new empty one if not
	vm, ok := m.transitions[field.path]
	if !ok {
		vm = newValueMatcher()

		// Add valueMatcher for new path in a thread-safe way.
		//  this is klunky and slow but I'm optimizing the read-path performance and I don't want locks in the path
		newTrans := make(map[string]*valueMatcher)
		for k, v := range m.transitions {
			newTrans[k] = v
		}
		newTrans[field.path] = vm
		m.transitions = newTrans
	}

	// suppose I'm adding the first pattern to a matcher and it has "x": [1, 2]. In principle the branches on
	//  "x": 1 and "x": 2 could go to tne same next state. But we have to make a unique next state for each of them
	//  because some future other pattern might have "x": [2, 3] and thus we need a separate branch to potentially
	//  match two patterns on "x": 2 but not "x": 1. If you were optimizing the automaton for size you might detect
	//  cases where this doesn't happen and reduce the number of fieldMatchStates
	var nextFieldMatchers []*fieldMatcher
	for _, val := range field.vals {
		nextFieldMatchers = append(nextFieldMatchers, vm.addTransition(val))

		// if the val is a number, let's add a transition on the canonicalized number
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
	}
	return nextFieldMatchers
}

// transitionOn returns one or more fieldMatchStates you can transition to on a field's name/value combination,
//  or nil if no transitions are possible.  An example of name/value that could produce multiple next states
//  would be if you had the pattern { "a": [ "foo" ] } and another pattern that matched any value with
//  a prefix of "f".
func (m *fieldMatcher) transitionOn(field *Field) []*fieldMatcher {

	// are there transitions on this field name?
	valMatcher, ok := m.transitions[string(field.Path)]
	if !ok {
		return nil
	}

	return valMatcher.transitionOn(field.Val)
}
