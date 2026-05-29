package quamina

type matcher interface {
	addPattern(x X, pat string) error
	matchesForFields(fields []Field, bufs *nfaBuffers) ([]X, error)
	deletePatterns(x X) error
	getSegmentsTreeTracker() SegmentsTreeTracker
	getStats() *matcherStats
}

type matcherStats struct {
	states     int64
	bytes      int64
	fanouts    int64
	maxFanout  int64
	seenStates map[*faState]bool
}
