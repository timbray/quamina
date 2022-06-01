package quamina

import "fmt"

// TODO: add stats for average and max smallTable fanout
type stats struct {
	fmCount    int
	fmTblCount int
	fmEntries  int
	fmVisited  map[*fieldMatcher]bool
	vmCount    int
	vmVisited  map[*valueMatcher]bool
	stCount    int
	stEntries  int
	stVisited  map[any]bool
	siCount    int
}

func matcherStats(m *coreMatcher) string {
	s := stats{
		fmVisited: make(map[*fieldMatcher]bool),
		vmVisited: make(map[*valueMatcher]bool),
		stVisited: make(map[any]bool),
	}
	fmStats(m.start().state, &s)
	avgFmSize := fmt.Sprintf("%.3f", float64(s.fmEntries)/float64(s.fmTblCount))
	avgStSize := "n/a"
	if s.stCount > 0 {
		avgStSize = fmt.Sprintf("%.3f", float64(s.stEntries)/float64(s.stCount))
	}
	return fmt.Sprintf("Field matchers: %d (avg size %s), Value matchers: %d, SmallTables %d (avg size %s), singletons %d", s.fmCount, avgFmSize, s.vmCount, s.stCount, avgStSize, s.siCount)
}

func fmStats(m *fieldMatcher, s *stats) {
	if s.fmVisited[m] {
		return
	}
	s.fmVisited[m] = true
	s.fmCount++
	tSize := len(m.fields().transitions)
	if tSize > 0 {
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
	if state.startNfa != nil {
		nfaStats(state.startNfa, s)
	} else if state.startDfa != nil {
		dfaStats(state.startDfa, s)
	}
}

func dfaStats(t *smallTable[*dfaStep], s *stats) {
	if s.stVisited[t] {
		return
	}
	s.stVisited[t] = true
	s.stCount++
	s.stEntries += len(t.ceilings)
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
func nfaStats(t *smallTable[*nfaStepList], s *stats) {
	if s.stVisited[t] {
		return
	}
	s.stVisited[t] = true
	s.stCount++
	s.stEntries += len(t.ceilings)
	for _, stepList := range t.steps {
		if stepList == nil {
			continue
		}
		for _, step := range stepList.steps {
			if step != nil {
				if step.fieldTransitions != nil {
					for _, m := range step.fieldTransitions {
						fmStats(m, s)
					}
				}
				nfaStats(step.table, s)
			}
		}
	}
}
