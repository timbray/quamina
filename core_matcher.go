package quamina

// coreMatcher represents an automaton that allows matching sequences of
// name/value field pairs against
//  patterns, which are combinations of field names and lists of allowed valued field values.
// The field names are called "Paths" because they encode, in a jsonpath-ish
// style, the pathSegments from the
//  root of an incoming object to the leaf field.
// Since the order of fields is generally not significant in encoded data
// objects, the fields are sorted
//  by name before constructing the automaton, and so are the incoming field lists to be matched, allowing
//  the automaton to work.

import (
	"errors"
	"sort"
	"sync"
	"sync/atomic"
)

// coreMatcher uses a finite automaton to implement the matchesForJSONEvent and MatchesForFields functions.
// The updateable fields are grouped into the coreStart member so they can be updated atomically using atomic.Load()
// and atomic.Store(). This is necessary for coreMatcher to be thread-safe.
type coreMatcher struct {
	updateable atomic.Value // always holds a *coreStart
	lock       sync.Mutex
}

// coreStart groups the updateable fields in coreMatcher.
// state is the start of the automaton.
// namesUsed is a map of field names that are used in any of the patterns that this automaton encodes. Typically,
// patterns only consider a subset of the fields in an incoming data object, and there is no reason to consider
// fields that do not appear in patterns when using the automaton for matching.
// fakeField is used when the flattener for an event returns no fields, because it could still match if
// there were patterns with "exists":false. So in this case we run one fake field through the matcher
// which will cause it to notice that any "exists":false patterns should match.
type coreStart struct {
	state     *fieldMatcher
	namesUsed map[string]bool
	fakeField []Field
}

func newCoreMatcher() *coreMatcher {

	// because of the way the matcher works, to serve its purpose of ensuring that "exists":false maches
	// will be detected, the Path has to be lexically greater than any field path that appears in
	// "exists":false. The value with byteCeiling works because that byte can't actually appear in any
	// user-supplied path-name because it's not valid in UTF-8
	fake := Field{
		Path:       []byte{byte(byteCeiling)},
		Val:        []byte(""),
		ArrayTrail: []ArrayPos{{0, 0}},
	}
	m := coreMatcher{}
	m.updateable.Store(&coreStart{
		state:     newFieldMatcher(),
		namesUsed: make(map[string]bool),
		fakeField: []Field{fake},
	})
	return &m
}

func (m *coreMatcher) start() *coreStart {
	return m.updateable.Load().(*coreStart)
}

// addPattern - the patternBytes is a JSON object. The X is what the matcher returns to indicate that the
// provided pattern has been matched. In many applications it might be a string which is the pattern's name.
func (m *coreMatcher) addPattern(x X, patternJSON string) error {
	patternFields, patternNamesUsed, err := patternFromJSON([]byte(patternJSON))
	if err != nil {
		return err
	}

	sort.Slice(patternFields, func(i, j int) bool { return patternFields[i].path < patternFields[j].path })

	// only one thread can be updating at a time
	// NOTE: threads can be calling MatchesFor* functions at any time as we update the automaton. The goal is to
	//  maintain consistency during updates, in the sense that a pattern that has been matching events will not
	//  stop working during an update.
	m.lock.Lock()
	defer m.lock.Unlock()

	// we build up the new coreMatcher state in freshStart so we can atomically switch it in once complete
	freshStart := &coreStart{}
	freshStart.namesUsed = make(map[string]bool)
	current := m.start()
	freshStart.state = current.state
	freshStart.fakeField = current.fakeField

	for k := range current.namesUsed {
		freshStart.namesUsed[k] = true
	}
	for used := range patternNamesUsed {
		freshStart.namesUsed[used] = true
	}

	// now we add each of the name/value pairs in fields slice to the automaton, starting with the start state -
	//  the addTransition for a field returns a list of the fieldMatchers transitioned to for that name/val
	//  combo.
	states := []*fieldMatcher{current.state}
	for _, field := range patternFields {
		var nextStates []*fieldMatcher
		for _, state := range states {
			var ns []*fieldMatcher
			if field.vals[0].vType == existsFalseType {
				ns = state.addExistsFalseTransition(field)
			} else {
				ns = state.addTransition(field)
			}

			nextStates = append(nextStates, ns...)
		}
		states = nextStates
	}

	// we've processed all the name/val combos in fields, "states" now holds the set of terminal states arrived at
	//  by matching each field in the pattern so update the matches value to indicate this (skipping those that
	//  are only there to serve exists:false processing)
	for _, endState := range states {
		endState.addMatch(x)
	}
	m.updateable.Store(freshStart)

	return err
}

// deletePattern not implemented by coreMatcher
func (m *coreMatcher) deletePatterns(_ X) error {
	return errors.New("operation not supported")
}

// matchesForJSONEvent calls the flattener to pull the fields out of the event and
// hands over to MatchesForFields
// This is a leftover from previous times, is only used by tests, but it's used by a *lot*
// so removing it would require a lot of tedious work
func (m *coreMatcher) matchesForJSONEvent(event []byte) ([]X, error) {
	fields, err := newJSONFlattener().Flatten(event, m)
	if err != nil {
		return nil, err
	}

	// see the commentary on coreMatcher for an explanation of this.
	// tl;dr: If the flattener returns no fields because there's nothing in the event that's mentioned in
	// any patterns, the event could still match if there are only "exists":false patterns.
	if len(fields) == 0 {
		fields = m.start().fakeField
	}

	return m.matchesForFields(fields)
}

// matchesForFields takes a list of Field structures, sorts them by pathname, and launches the field-matching
// process. The fields in a pattern to match are similarly sorted; thus running an automaton over them works
func (m *coreMatcher) matchesForFields(fields []Field) ([]X, error) {
	sort.Slice(fields, func(i, j int) bool { return string(fields[i].Path) < string(fields[j].Path) })
	matches := newMatchSet()

	// for each of the fields, we'll try to match the automaton start state to that field - the tryToMatch
	// routine will, in the case that there's a match, call itself to see if subsequent fields after the
	// first matched will transition through the machine and eventually achieve a match
	for i := range fields {
		tryToMatch(fields, i, m.start().state, matches, make(map[string]*fieldMatcher))
	}
	return matches.matches(), nil
}

// tryToMatch tries to match the field at fields[index] to the provided state. If it does match and generate
// 1 or more transitions to other states, it calls itself recursively to see if any of the remaining fields
// can continue the process by matching that state.
func tryToMatch(fields []Field, index int, state *fieldMatcher, matches *matchSet, incomingEFMs map[string]*fieldMatcher) {

	// finished?
	if index == len(fields) {
		return
	}

	fieldPath := fields[index].Path

	// in the following discussion, "efm" and "EFM" stand for "exists":false matches.
	// first, we construct the efmSignal going forward, which is the merger of any incoming ones from
	// previous states and any in the signal for this state
	// all this looks expensive but in most cases both incoming and state efm signals will be empty, thus a no-op
	efmsFromState := state.fields().pendingExistsFalses

	// newEFMs = incomingEFMs + efmsFromState
	newEFMs := make(map[string]*fieldMatcher, len(efmsFromState)+len(incomingEFMs))
	for path, trans := range incomingEFMs {
		newEFMs[path] = trans
	}
	for path, trans := range efmsFromState {
		newEFMs[path] = trans
	}

	// the list in which we'll store any states we'll be processing transitions to as a result of this field
	var nextStates []*fieldMatcher

	// now we'll look at the pending EFMs to see if any have succeeded or failed
	for efmPath, trans := range newEFMs {
		switch {
		case string(fieldPath) < efmPath:
			// no-op, we haven't got to a path lexically >= the one this signal is looking for
		case string(fieldPath) == efmPath:
			// the path that appeared in an exists:false exists, so we will delete it from the pending list
			delete(newEFMs, efmPath)
		case string(fieldPath) > efmPath:
			// we have lexically passed the path that appeared in an exists:false and it's still pending,
			// so this exists:false must have matched, save the associated transition for later processing
			nextStates = []*fieldMatcher{trans}
		}
	}

	// try to transition through the machine
	nextStates = append(nextStates, state.transitionOn(&fields[index])...)

	// for each state in the possibly-empty list of transitions from this state on fields[index]
	for _, nextState := range nextStates {

		// if arriving at this state means we've matched one or more patterns, record that fact
		matches = matches.addXSingleThreaded(nextState.fields().matches...)

		// for each state we've transitioned to, give each subsequent field a chance to
		//  transition on it, assuming it's not in an object that's in a different element
		//  of the same array
		for nextIndex := index + 1; nextIndex < len(fields); nextIndex++ {
			if noArrayTrailConflict(fields[index].ArrayTrail, fields[nextIndex].ArrayTrail) {
				tryToMatch(fields, nextIndex, nextState, matches, newEFMs)
			}
		}
	}
}

func noArrayTrailConflict(from []ArrayPos, to []ArrayPos) bool {
	for _, fromAPos := range from {
		for _, toAPos := range to {
			if fromAPos.Array == toAPos.Array && fromAPos.Pos != toAPos.Pos {
				return false
			}
		}
	}
	return true
}

func (m *coreMatcher) IsNameUsed(label []byte) bool {
	_, ok := m.start().namesUsed[string(label)]
	return ok
}
