package quamina

type matcher interface {
	addPattern(x X, pat string) error
	matchesForFields(fields []Field, bufs *nfaBuffers) ([]X, error)
	deletePatterns(x X) error
	getSegmentsTreeTracker() SegmentsTreeTracker
}
