package quamina

import (
	"errors"
	"fmt"
)

// Quamina instances provide the public APIs of this pattern-matching library.  A single Quamina instance is
// not thread-safe in that it cannot safely be used simultaneously in multiple goroutines. To re-use a
// Quamina instance concurrently in multiple goroutines, create copies using the Copy API.
type Quamina struct {
	flattener          Flattener
	bufs               *nfaBuffers
	matcher            matcher
	mediaTypeSpecified bool
	deletionSpecified  bool
	buildMode          MatcherBuildMode
}

// Option is an interface type used in Quamina's New API to pass in options. By convention, Option names
// have a prefix of "With".
type Option func(q *Quamina) error

// WithMediaType provides a media-type to support the selection of an appropriate Flattener.
// This option call may not be provided more than once, nor can it be combined on the same
// invocation of quamina.New() with the WithFlattener() option.
func WithMediaType(mediaType string) Option {
	return func(q *Quamina) error {
		if q.flattener != nil {
			return errors.New("flattener already specified")
		}
		if q.mediaTypeSpecified {
			return errors.New("media-type specified more than once")
		}
		switch mediaType {
		case "application/json":
			q.flattener = newJSONFlattener()
		default:
			return fmt.Errorf(`media type "%s" is not supported by Quamina`, mediaType)
		}
		q.mediaTypeSpecified = true
		return nil
	}
}

// WithFlattener allows the specification of a caller-provided Flattener instance to use on incoming Events.
// This option call may not be provided more than once, nor can it be combined on the same
// invocation of quamina.New() with the WithMediaType() option.
func WithFlattener(f Flattener) Option {
	return func(q *Quamina) error {
		if q.mediaTypeSpecified {
			return errors.New("media-type already specified")
		}
		if q.flattener != nil {
			return errors.New("flattener specified more than once")
		}
		if f == nil {
			return errors.New("nil Flattener")
		}
		q.flattener = f
		return nil
	}
}

// WithPatternDeletion arranges, if the argument is true, that this Quamina instance will support
// the DeletePatterns() method. This option call may not be provided more than once.
func WithPatternDeletion(b bool) Option {
	return func(q *Quamina) error {
		if q.deletionSpecified {
			return errors.New("pattern deletion already specified")
		}
		if b {
			q.matcher = newPrunerMatcher(nil)
		} else {
			q.matcher = newCoreMatcher()
		}
		q.deletionSpecified = true
		return nil
	}
}

// WithPatternStorage supplies the Quamina instance with a LivePatternState
// instance to be used to store the active patterns, i.e. those that have been
// added with AddPattern but not deleted with DeletePattern. This option call
// may not be provided more than once.
func WithPatternStorage(ps LivePatternsState) Option {
	return func(q *Quamina) error {
		if ps == nil {
			return errors.New("null PatternStorage")
		}
		return errors.New(" Pattern storage option not implemented yet")
	}
}

// New returns a new Quamina instance. Consult the APIs beginning with “With” for the options
// that may be used to configure the new instance.
func New(opts ...Option) (*Quamina, error) {
	var q Quamina
	for _, option := range opts {
		if err := option(&q); err != nil {
			return nil, err
		}
	}
	if (!q.mediaTypeSpecified) && (q.flattener == nil) {
		q.flattener = newJSONFlattener()
	}
	if !q.deletionSpecified {
		q.matcher = newCoreMatcher()
	}
	q.bufs = newNfaBuffers()
	q.buildMode = BuiltForComfort
	return &q, nil
}

// Copy produces a new Quamina instance designed to be used safely in parallel with existing instances on different
// goroutines.  Copy'ed instances share the same underlying data structures, so a pattern added to any instance
// with AddPattern will be visible in all of them.
func (q *Quamina) Copy() *Quamina {
	return &Quamina{matcher: q.matcher, flattener: q.flattener.Copy(), bufs: newNfaBuffers()}
}

// X is used in the AddPattern and MatchesForEvent APIs to identify the patterns that are added to
// a Quamina instance and are reported by that instance as matching an event. Commonly, X is a string
// used to name the pattern.
type X any

// AddPattern adds a pattern, identified by the x argument, to a Quamina instance.
// patternJSON is a JSON object. error is returned in the case that the PatternJSON is invalid JSON or
// has a leaf which is not provided as an array. AddPattern is single-threaded; if it is invoked concurrently
// from multiple goroutines (in instances created using the Copy method) calls will block until any other
// AddPattern call in progress succeeds.
func (q *Quamina) AddPattern(x X, patternJSON string) error {
	return q.matcher.addPattern(x, patternJSON, q.buildMode)
}

// DeletePatterns removes patterns identified by the x argument from the Quamina instance; the effect
// is that return values from future calls to MatchesForEvent will not include this x value.
func (q *Quamina) DeletePatterns(x X) error {
	return q.matcher.deletePatterns(x)
}

// MatchesForEvent returns a slice of X values which identify patterns that have previously been added to this
// Quamina instance and which “match” the event in the sense described in README. The matches slice may be empty
// if no patterns match. error can be returned in case that the event is not a valid JSON object or contains
// invalid UTF-8 byte sequences.
func (q *Quamina) MatchesForEvent(event []byte) ([]X, error) {
	fields, err := q.flattener.Flatten(event, q.matcher.getSegmentsTreeTracker())
	if err != nil {
		return nil, err
	}
	return q.matcher.matchesForFields(fields, q.bufs)
}

// GetMatcherStats retrieves resource consumption data from a Quamina instance; its results depend only
// on the AddPattern() calls that have been made previously. It runs in read-only mode without mutex
// locking, so it should not be run in parallel with AddPattern() calls.
// It returns a map to allow for the addition of consumption metrics in future. At the moment, the
// only useful key is "bytes" and the corresponding value is the number of bytes consumed by the Quamina
// matcher's data structures. The growth in this value correlates reasonably well with the slowdown
// in AddPattern() and MatchesForEvent() performance in the case when the Patterns being added are
// of the "wildcard" or "regexp" flavors.
func (q *Quamina) GetMatcherStats() map[string]float64 {
	stats := q.matcher.getStats()
	return map[string]float64{
		"states":    float64(stats.states),
		"bytes":     float64(stats.bytes),
		"fanouts":   float64(stats.fanouts),
		"maxFanout": float64(stats.maxFanout),
	}
}

// MatcherBuildMode enumerates the modes a Quamina instance can be in. The default is BuiltForComfort.
// When a Quamina instance is in BuiltForComfort mode, adding Patterns which include wildcards and regexps
// result in NFA-based matchers. These are more compact and faster to build, but result in MatchesForEvent
// performance slowing down as a fairly linear function of the number of such patterns added.
// When an instance is in BuiltForSpeed mode, the NFAs which implement wildcard and regexp Patterns are converted
// to DFAs. As a result, the performance of MatchesForEvent is only very weakly related to the number of
// Patterns added and in practice is much faster.  However, certain combinations of such patterns can result
// in explosive growth of the size of the Matcher and the AddPattern latency.  This can be as bad as O(2**N)
// in the number of Patterns. The use of the GetMatcherStats API is advised to investigate the effects of the
// combination of this setting with an app's typical usage of AddPattern in production.
type MatcherBuildMode int

const (
	BuiltForComfort MatcherBuildMode = iota
	BuiltForSpeed
)

// SetMatcherBuildMode puts a Quamina instance into the provided mode
func (q *Quamina) SetMatcherBuildMode(mode MatcherBuildMode) error {
	q.buildMode = mode
	return nil
}
func (q *Quamina) GetMatcherBuildMode() MatcherBuildMode {
	return q.buildMode
}

// deprecated from here down

// SetMemoryBudget used to set an approximate limit on the number of bytes allocated in building matchers for complex
// patterns. However, it proved difficult to find an implementation that was both deterministic and had
// acceptable cost. Thus, this method is deprecated.
func (q *Quamina) SetMemoryBudget(budget uint64) (uint64, error) {
	return 0, errors.New("the MemoryBudget API is deprecated")
}

// GetMemoryBudget is deprecated; see the comment on SetMemoryBudget
func (q *Quamina) GetMemoryBudget() (uint64, uint64) {
	return 0, 0
}
