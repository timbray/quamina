package core

import "fmt"

// TODO: add stats for average and max smallTable fanout
type stats struct {
	fmCount   int
	fmVisited map[*fieldMatcher]bool
	vmCount   int
	vmVisited map[*valueMatcher]bool
	stCount   int
	stVisited map[*smallTable[*dfaStep]]bool
	siCount   int
}

func matcherStats(m *CoreMatcher) string {
	s := stats{
		fmVisited: make(map[*fieldMatcher]bool),
		vmVisited: make(map[*valueMatcher]bool),
		stVisited: make(map[*smallTable[*dfaStep]]bool),
	}
	fmStats(m.start().state, &s)
	return fmt.Sprintf("Field matchers: %d, Value matchers: %d, SmallTables %d, singletons %d",
		s.fmCount, s.vmCount, s.stCount, s.siCount)
}

func fmStats(m *fieldMatcher, s *stats) {
	if s.fmVisited[m] {
		return
	}
	s.fmVisited[m] = true
	s.fmCount++
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
		smallStats(state.startDfa, s)
	}
}

func smallStats(t *smallTable[*dfaStep], s *stats) {
	if s.stVisited[t] {
		return
	}
	s.stVisited[t] = true
	s.stCount++
	for _, step := range t.steps {
		if step != nil {
			if step.fieldTransitions != nil {
				for _, m := range step.fieldTransitions {
					fmStats(m, s)
				}
			}
			smallStats(step.table, s)
		}
	}
}
