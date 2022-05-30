package control

import (
	"github.com/timbray/quamina/core"
	"github.com/timbray/quamina/flattener"
)

type Quamina struct {
	flattener flattener.Flattener
	matcher   core.Matcher
}

func NewQuamina() *Quamina {
	return &Quamina{
		flattener: flattener.NewFJ(),
		matcher:   core.NewCoreMatcher(),
	}
}

// MatchesForJSONEvent calls the flattener to pull the fields out of the event and
//  hands over to MatchesForFields
func (q *Quamina) MatchesForJSONEvent(event []byte) ([]core.X, error) {
	fields, err := flattener.NewFJ().Flatten(event, q.matcher)
	if err != nil {
		return nil, err
	}
	matches, err := q.matcher.MatchesForFields(fields)
	return matches, err
}

func (q *Quamina) AddPattern(x core.X, pattern string) error {
	return q.matcher.AddPattern(x, pattern)
}
