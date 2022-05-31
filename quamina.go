package quamina

import (
	"errors"
	"fmt"
)

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

func (q *Quamina) Copy() *Quamina {
	return &Quamina{matcher: q.matcher, flattener: q.flattener.Copy()}
}

func (q *Quamina) AddPattern(x X, patternJSON string) error {
	return q.matcher.addPattern(x, patternJSON)
}
func (q *Quamina) MatchesForEvent(event []byte) ([]X, error) {
	fields, err := q.flattener.Flatten(event, q.matcher)
	if err != nil {
		return nil, err
	}
	matches, err := q.matcher.matchesForFields(fields)
	return matches, err
}
