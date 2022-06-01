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
	flattener Flattener
	matcher   matcher
}

type Option func(q *Quamina) error

func WithMediaType(mediaType string) Option {
	return func(q *Quamina) error {
		switch mediaType {
		case "application/json":
			q.flattener = newJSONFlattener()
		default:
			return fmt.Errorf(`media type "%s" is not supported by Quamina`, mediaType)
		}
		return nil
	}
}
func WithFlattener(f Flattener) Option {
	return func(q *Quamina) error {
		if f == nil {
			return errors.New("nil Flattener")
		}
		q.flattener = f
		return nil
	}
}
func WithPatternDeletion(b bool) Option {
	return func(q *Quamina) error {
		if b {
			q.matcher = newPrunerMatcher(nil)
		} else {
			q.matcher = newCoreMatcher()
		}
		return nil
	}
}
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
	defaultOps := []Option{WithPatternDeletion(false), WithFlattener(newJSONFlattener())}
	for _, option := range defaultOps {
		if err := option(&q); err != nil {
			return nil, err
		}
	}
	for _, option := range opts {
		if err := option(&q); err != nil {
			return nil, err
		}
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
