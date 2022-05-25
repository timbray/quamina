package core

type Matcher interface {
	AddPattern(x X, pat string) error
	MatchesForJSONEvent(event []byte) ([]X, error)
	MatchesForFields(fields []Field) ([]X, error)
	DeletePattern(x X) error
}
