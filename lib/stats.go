package quamina

import "fmt"

type stats struct {
	fmCount int
	fmVisited map[*fieldMatcher]bool
	vmCount int
	vmVisited map[*valueMatcher]bool
	stCount int
	stVisited map[*smallTable]bool
	siCount int
}

func matcherStats(m *Matcher) string {
	s := stats{
		fmVisited: make(map[*fieldMatcher]bool),
		vmVisited: make(map[*valueMatcher]bool),
		stVisited: make(map[*smallTable]bool),
	}
	fmStats(m.startState, &s)
	return fmt.Sprintf("Field matchers: %d, Value matchers: %d, SmallTables %d, singletons %d",
		s.fmCount, s.vmCount, s.stCount, s.siCount)
}

func fmStats(m *fieldMatcher, s *stats) {
	if s.fmVisited[m] {
		return
	}
	s.fmVisited[m] = true
	s.fmCount++
	for _, val := range m.transitions {
		vmStats(val, s)
	}
}

func vmStats(m *valueMatcher, s *stats) {
	if s.vmVisited[m] {
		return
	}
	s.vmVisited[m] = true
	s.vmCount++
	if m.singletonMatch != nil {
		s.siCount++
		fmStats(m.singletonTransition, s)
	}
	if m.startTable != nil {
		smallStats(m.startTable, s)
	}
}

func smallStats(t *smallTable, s *stats) {
	if s.stVisited[t] {
		return
	}
	s.stVisited[t] = true
	s.stCount++
	for _, step := range t.slices.steps {
		if step != nil {
			if step.HasTransition() {
				for _, m := range step.SmallTransition().fieldMatchers {
					fmStats(m, s)
				}
			}
			smallStats(step.SmallTable(), s)
		}
	}
}
