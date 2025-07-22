package quamina

import "fmt"

/*
func nfaBufStats(bufs *nfaBuffers) (string, *faState) {
	b1 := fmt.Sprintf("buf1 %d", cap(bufs.buf1))
	b2 := fmt.Sprintf("buf2 %d", cap(bufs.buf2))
	ecLen := len(bufs.eClosure.closures)
	tot := 0
	longest := 0
	var lState *faState
	for s, ec := range bufs.eClosure.closures {
		l := len(ec)
		if l > longest {
			longest = l
			lState = s
		}
		tot += l
	}
	avg := float64(tot) / float64(ecLen)
	ec := fmt.Sprintf("ec l=%d, avg=%0.1f, max=%d", ecLen, avg, longest)
	return "b1=" + b1 + ", b2=" + b2 + ", " + ec, lState
}
*/

type statsAccum struct {
	fmCount    int
	fmTblCount int
	fmEntries  int
	fmMax      int
	fmVisited  map[*fieldMatcher]bool
	vmCount    int
	vmVisited  map[*valueMatcher]bool
	stCount    int
	stTblCount int
	stEntries  int
	stMax      int
	stDepth    int
	stEpsilon  int
	stepMax    int
	stVisited  map[*smallTable]bool
	siCount    int
	splices    int
}

func (s *statsAccum) stStats() string {
	avgStSize := "n/a"
	if s.stTblCount > 0 {
		avgStSize = fmt.Sprintf("%.3f", float64(s.stEntries)/float64(s.stTblCount))
	}
	return fmt.Sprintf("SmallTables %d (avg size %s, max %d), singletons %d", s.stCount, avgStSize, s.stMax, s.siCount)
}

// matcherStats gathers statistics about the size of a coreMatcher, including the average and max fanout sizes of
// the transition tables, returning this information in string form
func matcherStats(m *coreMatcher) string {
	s := statsAccum{
		fmVisited: make(map[*fieldMatcher]bool),
		vmVisited: make(map[*valueMatcher]bool),
		stVisited: make(map[*smallTable]bool),
	}
	fmStats(m.fields().state, &s)
	avgFmSize := fmt.Sprintf("%.3f", float64(s.fmEntries)/float64(s.fmTblCount))
	avgStSize := "n/a"
	avgEpSize := "n/a"
	if s.stTblCount > 0 {
		avgStSize = fmt.Sprintf("%.3f", float64(s.stEntries)/float64(s.stTblCount))
	}
	if s.stEpsilon > 0 {
		avgEpSize = fmt.Sprintf("%.3f", float64(s.stEpsilon)/float64(s.stTblCount))
	}
	fmPart := fmt.Sprintf("Field matchers: %d (avg size %s, max %d)", s.fmCount, avgFmSize, s.fmMax)
	vmPart := fmt.Sprintf("Value matchers: %d", s.vmCount)
	stPart := fmt.Sprintf("SmallTables %d (unique %d, splices %d ,avg %s, max %d, epsilons avg %s, max %d) singletons %d",
		s.stCount, len(s.stVisited), s.splices, avgStSize, s.stMax, avgEpSize, s.stepMax, s.siCount)

	return fmPart + "\n" + vmPart + "\n" + stPart
}

func fmStats(m *fieldMatcher, s *statsAccum) {
	if s.fmVisited[m] {
		return
	}
	s.fmVisited[m] = true
	s.fmCount++
	tSize := len(m.fields().transitions)
	if tSize > 0 {
		if tSize > s.fmMax {
			s.fmMax = tSize
		}
		s.fmTblCount++
		s.fmEntries += tSize
	}

	for _, val := range m.fields().transitions {
		vmStats(val, s)
	}
}

func vmStats(m *valueMatcher, s *statsAccum) {
	if s.vmVisited[m] {
		return
	}
	s.vmVisited[m] = true
	s.vmCount++
	state := m.fields()
	if state.singletonMatch != nil {
		s.siCount++
		fmStats(state.singletonTransition, s)
	}
	if state.startTable != nil {
		faStats(state.startTable, s)
	}
}

func faStats(t *smallTable, s *statsAccum) {
	s.stCount++
	if s.stVisited[t] {
		return
	}
	s.stVisited[t] = true
	tSize := len(t.ceilings) + len(t.epsilons)
	if t.isJustEpsilons() {
		s.splices++
	}
	if tSize > 1 {
		if tSize > s.stMax {
			s.stMax = tSize
		}
		s.stTblCount++
		s.stEntries += len(t.ceilings)
		s.stEpsilon += len(t.epsilons)
		if len(t.epsilons) > s.stepMax {
			s.stepMax = len(t.epsilons)
		}
	}
	for _, next := range t.steps {
		if next != nil {
			faStats(next.table, s)
		}
	}
	for _, epsilon := range t.epsilons {
		faStats(epsilon.table, s)
	}
}
