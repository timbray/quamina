package quamina

// X is used in the AddPattern and MatchesForEvent APIs to identify the patterns that are added to
// a Quamina instance and are reported by that instance as matching an event. Commonly, X is a string
// used to name the event.
type X any

type matcher interface {
	addPattern(x X, pat string) error
	matchesForFields(fields []Field) ([]X, error)
	deletePatterns(x X) error
	IsNameUsed(label []byte) bool
}
