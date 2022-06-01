package quamina

import (
	"errors"
	"fmt"
)

// Quamina instances provide the public APIs of this pattern-matching library.
// flattener is responsible for turning the bytes of incoming events into a list of name/value pairs. Each
//  Quamina instance has its own flattener, because flatteners are stateful and not designed for concurrent use.
// matcher is the root of the two-level automaton structure containing fieldMatcher and valueMatcher nodes.  Multiple
//  Quamina instances may have the same matcher value, since it is designed for concurrent operation.
type Quamina struct {
	flattener          Flattener
	matcher            matcher
	flattenerSpecified bool
	mediaTypeSpecified bool
	deletionSpecified  bool
}

type Option func(q *Quamina) error

// WithMediaType provides a media-type to support the selection of an appropriate Flattener.
//  This option call may not be provided more than once, nor can it be combined on the same
//  invocation of quamina.New() with the WithFlattener() option.
func WithMediaType(mediaType string) Option {
	return func(q *Quamina) error {
		if q.flattenerSpecified {
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
//  This option call may not be provided more than once, nor can it be combined on the same
//  invocation of quamina.New() with the WithMediaType() option.
func WithFlattener(f Flattener) Option {
	return func(q *Quamina) error {
		if q.mediaTypeSpecified {
			return errors.New("media-type already specified")
		}
		if q.flattenerSpecified {
			return errors.New("flattener specified more than once")
		}
		if f == nil {
			return errors.New("nil Flattener")
		}
		q.flattener = f
		q.flattenerSpecified = true
		return nil
	}
}

// WithPatternDeletion arranges, if the argument is true, that this Quamina instance will support
//  the DeletePatterns() method. This option call may not be provided more than once.
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

// WithPatternDeletion supplies the Quamina instance with a LivePatternState instance to be used to store
//  the active patterns, i.e. those that have been added with AddPattern but not deleted with
//  DeletePattern. This option call may not be provided more than once.
func WithPatternStorage(ps LivePatternsState) Option {
	return func(q *Quamina) error {
		if ps == nil {
			return errors.New("null PatternStorage")
		}
		return errors.New(" Pattern storage option not implemented yet")
	}
}

func New(opts ...Option) (*Quamina, error) {
	var q Quamina
	for _, option := range opts {
		if err := option(&q); err != nil {
			return nil, err
		}
	}
	if !(q.mediaTypeSpecified || q.flattenerSpecified) {
		q.flattener = newJSONFlattener()
	}
	if !q.deletionSpecified {
		q.matcher = newCoreMatcher()
	}
	return &q, nil
}

// Copy produces a new Quamina instance which share the matcher of the current, but starts with
//  a new flattener.
func (q *Quamina) Copy() *Quamina {
	return &Quamina{matcher: q.matcher, flattener: q.flattener.Copy()}
}

func (q *Quamina) AddPattern(x X, patternJSON string) error {
	return q.matcher.addPattern(x, patternJSON)
}
func (q *Quamina) DeletePatterns(x X) error {
	return q.matcher.deletePatterns(x)
}
func (q *Quamina) MatchesForEvent(event []byte) ([]X, error) {
	fields, err := q.flattener.Flatten(event, q.matcher)
	if err != nil {
		return nil, err
	}
	matches, err := q.matcher.matchesForFields(fields)
	return matches, err
}
