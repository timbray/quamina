package quamina

import "fmt"

// TODO: add stats for average and max smallTable fanout
type stats struct {
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
	stVisited  map[any]bool
	siCount    int
}

// matcherStats gathers statistics about the size of a coreMatcher, including the average and max fanout sizes of
//
//	the transition tables, returning this information in string form
func matcherStats(m *coreMatcher) string {
	s := stats{
		fmVisited: make(map[*fieldMatcher]bool),
		vmVisited: make(map[*valueMatcher]bool),
		stVisited: make(map[any]bool),
	}
	fmStats(m.start().state, &s)
	avgFmSize := fmt.Sprintf("%.3f", float64(s.fmEntries)/float64(s.fmTblCount))
	avgStSize := "n/a"
	if s.stTblCount > 0 {
		avgStSize = fmt.Sprintf("%.3f", float64(s.stEntries)/float64(s.stTblCount))
	}
	fmPart := fmt.Sprintf("Field matchers: %d (avg size %s, max %d), ", s.fmCount, avgFmSize, s.fmMax)
	vmPart := fmt.Sprintf("Value matchers: %d, ", s.vmCount)
	stPart := fmt.Sprintf("SmallTables %d (avg size %s, max %d), singletons %d", s.stCount, avgStSize, s.stMax, s.siCount)
	return fmPart + vmPart + stPart
}

func fmStats(m *fieldMatcher, s *stats) {
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

func vmStats(m *valueMatcher, s *stats) {
	if s.vmVisited[m] {
		return
	}
	s.vmVisited[m] = true
	s.vmCount++
	state := m.getFields()
	if state.singletonMatch != nil {
		s.siCount++
		fmStats(state.singletonTransition, s)
	}
	if state.startDfa != nil {
		dfaStats(state.startDfa, s)
	}
}

func dfaStats(t *smallTable[*dfaStep], s *stats) {
	if s.stVisited[t] {
		return
	}
	s.stVisited[t] = true
	s.stCount++
	tSize := len(t.ceilings)
	if tSize > 1 {
		if tSize > s.stMax {
			s.stMax = tSize
		}
		s.stTblCount++
		s.stEntries += len(t.ceilings)
	}
	for _, step := range t.steps {
		if step != nil {
			if step.fieldTransitions != nil {
				for _, m := range step.fieldTransitions {
					fmStats(m, s)
				}
			}
			dfaStats(step.table, s)
		}
	}
}
