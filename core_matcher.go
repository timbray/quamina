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

// coreMatcher uses an automaton to implement addPattern and matchesForFields.
// There are two levels of concurrency here. First, the lock field in this struct must be held by any goroutine
// that is executing addPattern(), i.e. only one thread may be updating the state machine at one time.
// However, any number of goroutines may in parallel be executing matchesForFields while the addPattern
// update is in progress. The updateable atomic.Value allows the addPattern thread to change the maps and
// slices in the structure atomically with atomic.Load() while matchesForFields threads are reading them.
type coreMatcher struct {
	updateable atomic.Value // always holds a *coreFields
	lock       sync.Mutex
}

// coreFields groups the updateable fields in coreMatcher.
// state is the start of the automaton.
// namesUsed is a map of field names that are used in any of the patterns that this automaton encodes. Typically,
// patterns only consider a subset of the fields in an incoming data object, and there is no reason to consider
// fields that do not appear in patterns when using the automaton for matching.
// fakeField is used when the flattener for an event returns no fields, because it could still match if
// there were patterns with "exists":false. So in this case we run one fake field through the matcher
// which will cause it to notice that any "exists":false patterns should match.
type coreFields struct {
	state     *fieldMatcher
	namesUsed map[string]bool
}

func newCoreMatcher() *coreMatcher {
	// because of the way the matcher works, to serve its purpose of ensuring that "exists":false maches
	// will be detected, the Path has to be lexically greater than any field path that appears in
	// "exists":false. The value with byteCeiling works because that byte can't actually appear in any
	// user-supplied path-name because it's not valid in UTF-8
	m := coreMatcher{}
	m.updateable.Store(&coreFields{
		state:     newFieldMatcher(),
		namesUsed: make(map[string]bool),
	})
	return &m
}

func (m *coreMatcher) start() *coreFields {
	return m.updateable.Load().(*coreFields)
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
	m.lock.Lock()
	defer m.lock.Unlock()

	// we build up the new coreMatcher state in freshStart so we can atomically switch it in once complete
	freshStart := &coreFields{}
	freshStart.namesUsed = make(map[string]bool)
	current := m.start()
	freshStart.state = current.state

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

		// separate handling for field exists:true/false and regular field name/val matches. Since the exists
		// true/false are only allowed one value, we can test vals[0] to figure out which type
		for _, state := range states {
			var ns []*fieldMatcher
			switch field.vals[0].vType {
			case existsTrueType:
				ns = state.addExists(true, field)
			case existsFalseType:
				ns = state.addExists(false, field)
			default:
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
		fields = []Field{
			{
				Path:       []byte{byte(byteCeiling)},
				Val:        []byte(""),
				ArrayTrail: []ArrayPos{{0, 0}},
			},
		}
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
	s := m.start()
	for i := 0; i < len(fields); i++ {
		tryToMatch(fields, i, s.state, matches)
	}
	return matches.matches(), nil
}

// tryToMatch tries to match the field at fields[index] to the provided state. If it does match and generate
// 1 or more transitions to other states, it calls itself recursively to see if any of the remaining fields
// can continue the process by matching that state.
func tryToMatch(fields []Field, index int, state *fieldMatcher, matches *matchSet) {
	stateFields := state.fields()

	// transition on exists:true?
	existsTrans, ok := stateFields.existsTrue[string(fields[index].Path)]
	if ok {
		matches = matches.addXSingleThreaded(existsTrans.fields().matches...)
		for nextIndex := index + 1; nextIndex < len(fields); nextIndex++ {
			if noArrayTrailConflict(fields[index].ArrayTrail, fields[nextIndex].ArrayTrail) {
				tryToMatch(fields, nextIndex, existsTrans, matches)
			}
		}
	}

	// an exists:false transition is possible if there is no matching field in the event
	// func checkExistsFalse(stateFields *fmFields, fields []Field, index int, matches *matchSet) {
	checkExistsFalse(stateFields, fields, index, matches)

	// try to transition through the machine
	nextStates := state.transitionOn(&fields[index])

	// for each state in the possibly-empty list of transitions from this state on fields[index]
	for _, nextState := range nextStates {
		nextStateFields := nextState.fields()
		matches = matches.addXSingleThreaded(nextStateFields.matches...)

		// for each state we've transitioned to, give each subsequent field a chance to
		//  transition on it, assuming it's not in an object that's in a different element
		//  of the same array
		for nextIndex := index + 1; nextIndex < len(fields); nextIndex++ {
			if noArrayTrailConflict(fields[index].ArrayTrail, fields[nextIndex].ArrayTrail) {
				tryToMatch(fields, nextIndex, nextState, matches)
			}
		}
		// now we've run out of fields to match this nextState against. But suppose it has an exists:false
		// transition, and it so happens that the exists:false pattern field is lexically larger than the other
		// fields and that in fact such a field does not exist. That state would be left hanging. So…
		checkExistsFalse(nextStateFields, fields, index, matches)
	}
}

func checkExistsFalse(stateFields *fmFields, fields []Field, index int, matches *matchSet) {
	for existsFalsePath, existsFalseTrans := range stateFields.existsFalse {
		// it seems like there ought to be a more state-machine-idiomatic way to do this but
		// I thought of a few and none of them worked.  Quite likely someone will figure it out eventually.
		// Could get slow for big events with hundreds or more fields (not that I've ever seen that) - might
		// be worthwhile switching to binary search at some field count.
		var i int
		var thisFieldIsAnExistsFalse bool
		for i = 0; i < len(fields); i++ {
			if string(fields[i].Path) == existsFalsePath {
				if i == index {
					thisFieldIsAnExistsFalse = true
				}
				break
			}
		}
		if i == len(fields) {
			matches = matches.addXSingleThreaded(existsFalseTrans.fields().matches...)
			if thisFieldIsAnExistsFalse {
				tryToMatch(fields, index+1, existsFalseTrans, matches)
			} else {
				tryToMatch(fields, index, existsFalseTrans, matches)
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
