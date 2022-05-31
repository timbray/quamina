package quamina

type Quamina struct {
	flattener Flattener
	matcher   Matcher
}

func New() *Quamina {
	return &Quamina{
		flattener: NewFJ(),
		matcher:   NewCoreMatcher(),
	}
}

func (q *Quamina) AddPattern(x X, patternJSON string) error {
	return q.matcher.AddPattern(x, patternJSON)
}
func (q *Quamina) MatchesForJSONEvent(event []byte) ([]X, error) {
	fields, err := q.flattener.Flatten(event, q.matcher)
	if err != nil {
		return nil, err
	}
	matches, err := q.matcher.MatchesForFields(fields)
	return matches, err
}

// TODO: Make a parameterized version so you can request a custom flattener or a matcher that supports DeletePattern()
