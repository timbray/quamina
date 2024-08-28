package quamina

// coreMatcher represents an automaton that allows matching sequences of name/value field pairs against patterns,
// which are combinations of field names and lists of allowed field values. The field names are called
// "Paths" because they encode, in a jsonpath-ish style, the pathSegments from the root of an incoming object to
// the leaf field. Since the order of fields is generally not significant in encoded data objects, the fields are
// sorted by name before constructing the automaton, and so are the incoming field lists to be matched, allowing
// the automaton to work.

import (
	"bytes"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
)

// coreMatcher uses an automaton to implement addPattern and matchesForFields.
// There are two levels of concurrency here. First, the lock field in this struct must be held by any goroutine
// that is executing addPattern(), i.e. only one thread may be updating the state machine at one time.
// However, any number of goroutines may in parallel be executing matchesForFields while the addPattern
// update is in progress. The updateable atomic.Pointer allows the addPattern thread to change the maps and
// slices in the structure atomically with atomic.Load() while matchesForFields threads are reading them.
type coreMatcher struct {
	updateable atomic.Pointer[coreFields]
	lock       sync.Mutex
}

// coreFields groups the updateable fields in coreMatcher.
// state is the start of the automaton.
// segmentsTree is a structure that encodes which fields appear in the Patterns that are added to the coreMatcher.
// It is built during calls to addPattern. It implements SegmentsTreeTracker, which is used by the event flattener
// to optimize the flattening process by skipping the processing of fields which are not used in any pattern.
type coreFields struct {
	state        *fieldMatcher
	segmentsTree *segmentsTree
	nfaMeta      *nfaMetadata
}

func newCoreMatcher() *coreMatcher {
	m := coreMatcher{}
	m.updateable.Store(&coreFields{
		state:        newFieldMatcher(),
		segmentsTree: newSegmentsIndex(),
		nfaMeta:      &nfaMetadata{},
	})
	return &m
}

func (m *coreMatcher) fields() *coreFields {
	return m.updateable.Load()
}

// analyze traverses all the different per-field NFAs and gathers metadata that can be
// used to optimize traversal. At the moment, all that it gathers is the maximum outdegree
// from any smallTable, where outdegree is the epsilon count plus the largest number of
// targets jumped to from a single byte transition. Can be called any time but normally
// you'd do this after you've added a bunch of patterns and are ready to start matching
func (m *coreMatcher) analyze() {
	// only one thread can be updating at a time
	m.lock.Lock()
	defer m.lock.Unlock()

	fields := m.fields()
	fields.state.gatherMetadata(fields.nfaMeta)
	m.updateable.Store(fields)
}

// addPattern - the patternBytes is a JSON text which must be an object. The X is what the matcher returns to indicate
// that the provided pattern has been matched. In many applications it might be a string which is the pattern's name.
func (m *coreMatcher) addPattern(x X, patternJSON string) error {
	return m.addPatternWithPrinter(x, patternJSON, sharedNullPrinter)
}

// addPatternWithPrinter can be called from debugging and under-development code to allow viewing pretty-printed
// NFAs
func (m *coreMatcher) addPatternWithPrinter(x X, patternJSON string, printer printer) error {
	patternFields, err := patternFromJSON([]byte(patternJSON))
	if err != nil {
		return err
	}

	// sort the pattern fields lexically
	sort.Slice(patternFields, func(i, j int) bool { return patternFields[i].path < patternFields[j].path })

	// only one thread can be updating at a time
	m.lock.Lock()
	defer m.lock.Unlock()

	// we build up the new coreMatcher state in freshStart so that we can atomically switch it in once complete
	freshStart := &coreFields{}
	currentFields := m.fields()
	freshStart.segmentsTree = currentFields.segmentsTree.copy()
	freshStart.state = currentFields.state
	freshStart.nfaMeta = currentFields.nfaMeta

	// Add paths to the segments tree index.
	for _, field := range patternFields {
		freshStart.segmentsTree.add(field.path)
	}

	// now we add each of the name/value pairs in fields slice to the automaton, starting with the start state -
	// the addTransition for a field returns a list of the fieldMatchers transitioned to for that name/val
	// combo.
	states := []*fieldMatcher{currentFields.state}
	for _, field := range patternFields {
		// if the field has no values, this is a no-op
		if len(field.vals) == 0 {
			continue
		}

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
				ns = state.addTransition(field, printer)
			}

			nextStates = append(nextStates, ns...)
		}
		states = nextStates
	}

	// we've processed all the name/val combos in fields, "states" now holds the set of terminal states arrived at
	//  by matching each field in the pattern, so update the matches value to indicate this.
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
// and it's a convenient API for testing.
func (m *coreMatcher) matchesForJSONEvent(event []byte) ([]X, error) {
	return m.matchesForJSONWithFlattener(event, newJSONFlattener())
}

// if your test is a benchmark, call newJSONFlattener and pass it to this routine, matchesForJSONWithFlattener
// because newJSONFlattener() is fairly heavyweight and you want it out of the benchmark loop
func (m *coreMatcher) matchesForJSONWithFlattener(event []byte, f Flattener) ([]X, error) {
	fields, _ := f.Flatten(event, m.getSegmentsTreeTracker())
	return m.matchesForFields(fields)
}

// emptyFields returns a fake []Field list containing a single field whose name is lexically greater than any that
// can occur in real data
// see the commentary on coreMatcher for an explanation.
// tl;dr: If the flattener returns no fields because there's nothing in the event that's mentioned in
// any patterns, the event could still match if there are only "exists":false patterns.
func emptyFields() []Field {
	return []Field{
		{
			Path:       []byte{byte(byteCeiling)},
			Val:        []byte(""),
			ArrayTrail: []ArrayPos{{0, 0}},
		},
	}
}

// fieldsList exists to support the sort.Sort call in matchesForFields()
type fieldsList []Field

func (a fieldsList) Len() int {
	return len(a)
}
func (a fieldsList) Less(i, j int) bool {
	return bytes.Compare(a[i].Path, a[j].Path) < 0
}
func (a fieldsList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// matchesForFields takes a list of Field structures, sorts them by pathname, and launches the field-matching
// process. The fields in a pattern to match are similarly sorted; thus running an automaton over them works.
// No error can be returned but the matcher interface requires one, and it is used by the pruner implementation
func (m *coreMatcher) matchesForFields(fields []Field) ([]X, error) {
	if len(fields) == 0 {
		fields = emptyFields()
	} else {
		sort.Sort(fieldsList(fields))
	}
	matches := newMatchSet()
	cmFields := m.fields()

	// nondeterministic states in this matcher's automata have a list of current states and
	// transition to a list of next states. This requires memory shuffling, which we want to
	// minimize at matching/traversal time. Whatever we do, we want to keep one pair of
	// buffers around for an entire matchesForFields call, bufs is that.
	// In theory, there should be significant savings to be had by pre-allocating those buffers,
	// or managing a pool of them with sync.Pool, or some such. However, adding any straightforward
	// pre-allocation causes massive slowdown on the mainstream cases such as EXACT_MATCH in
	// TestRulerCl2(). My hypothesis is that the DFA-like processing there is so efficient that
	// anything that does actual allocation is death.
	// Thus was created the analyze() call, which traverses the whole coreMatcher tree and returns
	// the maximum state outdegree in the nfaMeta data structure, then pre-allocates a quality
	// estimate of what's going to be used. This did in fact produce an increase in performnance,
	// but that improvement was a small single-digit percentage and things that made one of EXACT,
	// ANYTHING_BUT, and SHELLSTYLE matches go faster made one of the others go slower.
	// Complicating factor: even if there is some modest amount of garbage collection, the Go
	// runtime seems to be very good at shuffling it off into another thread so that the actual
	// pattern-matching throughput doesn't suffer much. That's true at least on my massively
	// over-equipped M2 MBPro, but probably not on some miserable cloud event-handling worker.
	// Conclusion: I dunno. I left the analyze() func in but for now, don't use its results in
	// production.
	var bufs = &bufpair{}
	/*
		if cmFields.nfaMeta.maxOutDegree < 2 {
			bufs = &bufpair{}
		} else {
			bufferSize := cmFields.nfaMeta.maxOutDegree * 2
			bufs = &bufpair{
				buf1: make([]*faState, 0, bufferSize),
				buf2: make([]*faState, 0, bufferSize),
			}
		}
	*/

	// for each of the fields, we'll try to match the automaton start state to that field - the tryToMatch
	// routine will, in the case that there's a match, call itself to see if subsequent fields after the
	// first matched will transition through the machine and eventually achieve a match
	for i := 0; i < len(fields); i++ {
		tryToMatch(fields, i, cmFields.state, matches, bufs)
	}
	return matches.matches(), nil
}

// tryToMatch tries to match the field at fields[index] to the provided state. If it does match and generate
// 1 or more transitions to other states, it calls itself recursively to see if any of the remaining fields
// can continue the process by matching that state.
func tryToMatch(fields []Field, index int, state *fieldMatcher, matches *matchSet, bufs *bufpair) {
	stateFields := state.fields()

	// transition on exists:true?
	existsTrans, ok := stateFields.existsTrue[string(fields[index].Path)]
	if ok {
		matches = matches.addXSingleThreaded(existsTrans.fields().matches...)
		for nextIndex := index + 1; nextIndex < len(fields); nextIndex++ {
			if noArrayTrailConflict(fields[index].ArrayTrail, fields[nextIndex].ArrayTrail) {
				tryToMatch(fields, nextIndex, existsTrans, matches, bufs)
			}
		}
	}

	// an exists:false transition is possible if there is no matching field in the event
	checkExistsFalse(stateFields, fields, index, matches, bufs)

	// try to transition through the machine
	nextStates := state.transitionOn(&fields[index], bufs)

	// for each state in the possibly-empty list of transitions from this state on fields[index]
	for _, nextState := range nextStates {
		nextStateFields := nextState.fields()
		matches = matches.addXSingleThreaded(nextStateFields.matches...)

		// for each state we've transitioned to, give each subsequent field a chance to
		//  transition on it, assuming it's not in an object that's in a different element
		//  of the same array
		for nextIndex := index + 1; nextIndex < len(fields); nextIndex++ {
			if noArrayTrailConflict(fields[index].ArrayTrail, fields[nextIndex].ArrayTrail) {
				tryToMatch(fields, nextIndex, nextState, matches, bufs)
			}
		}
		// now we've run out of fields to match this state against. But suppose it has an exists:false
		// transition, and it so happens that the exists:false pattern field is lexically larger than the other
		// fields and that in fact such a field does not exist. That state would be left hanging. So…
		checkExistsFalse(nextStateFields, fields, index, matches, bufs)
	}
}

func checkExistsFalse(stateFields *fmFields, fields []Field, index int, matches *matchSet, bufs *bufpair) {
	for existsFalsePath, existsFalseTrans := range stateFields.existsFalse {
		// it seems like there ought to be a more state-machine-idiomatic way to do this, but
		// I thought of a few and none of them worked.  Quite likely someone will figure it out eventually.
		// Could get slow for big events with hundreds or more fields (not that I've ever seen that) - might
		// be worthwhile switching to binary search at some field count or building a map[]boolean in addPattern
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
				tryToMatch(fields, index+1, existsFalseTrans, matches, bufs)
			} else {
				tryToMatch(fields, index, existsFalseTrans, matches, bufs)
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

func (m *coreMatcher) getSegmentsTreeTracker() SegmentsTreeTracker {
	return m.fields().segmentsTree
}
