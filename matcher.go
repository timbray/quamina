package quamina

// X for anything, should eventually be a generic? TODO
type X any

type matcher interface {
	addPattern(x X, pat string) error
	matchesForFields(fields []Field) ([]X, error)
	deletePatterns(x X) error
	IsNameUsed(label []byte) bool
}
