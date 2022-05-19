package quamina

type Matcher interface {
	AddPattern(x X, pat string) error
	MatchesForJSONEvent(event []byte) ([]X, error)
	MatchesForFields(fields []Field) []X
	DeletePattern(x X) error
}
