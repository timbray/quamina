package quamina

// valueMatchState maps field values to the next automaton fieldMatchStates.
// this is a primitive first-cut approach. It doesn't know that "x": 35, "x": 3.5e1, and "x": 35.0 are
//  the same. It has no facilities for matching on prefixes or ranges, nor for negative matching.
// valueTransitions maps fields, as strings, to the next field-matching state from some pattern
// existsTransitions contains transitions on field existence (thus any value)
type valueMatchState struct {
	valueTransitions  map[string]*fieldMatchState
	existsTransitions []*fieldMatchState
}

func newValueMatchState() *valueMatchState {
	return &valueMatchState{valueTransitions: make(map[string]*fieldMatchState)}
}

func (m *valueMatchState) addTransition(val typedVal) *fieldMatchState {
	var next *fieldMatchState
	if val.vType == existsTrueType || val.vType == existsFalseType {
		next = newFieldMatchState()
		m.existsTransitions = append(m.existsTransitions, next)
		return next
	}

	var ok bool
	next, ok = m.valueTransitions[val.val]
	if !ok {
		next = newFieldMatchState()
		m.valueTransitions[val.val] = next
	}
	return next
}

// transitionOn transitions to a new fieldMatchState either based on the value (as a string) or
//  based on the field's existence
func (m *valueMatchState) transitionOn(val []byte) []*fieldMatchState {

	var transitions []*fieldMatchState
	for _, existsTransition := range m.existsTransitions {
		transitions = append(transitions, existsTransition)
	}
	next, ok := m.valueTransitions[string(val)]
	if ok {
		transitions = append(transitions, next)
	}
	return transitions
}

// for debugging
/*
func (m *valueMatchState) String() string {
	var keys = []string{"VM"}
	for k := range m.transitions {
		keys = append(keys, k)
	}
	return strings.Join(keys, " / ")
}
*/
