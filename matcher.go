package quamina

type matcher interface {
	addPattern(x X, pat string) error
	MatchesForJSONEvent(event []byte) ([]X, error)
	matchesForFields(fields []Field) ([]X, error)
	deletePattern(x X) error
	IsNameUsed(label []byte) bool
}
