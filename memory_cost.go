package quamina

import (
	"unsafe"
)

var mcPointer = int64(unsafe.Sizeof(&faState{}))
var mcSmallTableBase = int64(unsafe.Sizeof(smallTable{})) // should include 3* slice descriptor
var mcFaStateBase = int64(unsafe.Sizeof(faState{}))

func (m *coreMatcher) getStats() *matcherStats {
	stats := &matcherStats{
		seenStates: make(map[*faState]bool),
	}
	cmFieldMatcherStats(m.fields().state, stats, nil)
	return stats
}

func cmFieldMatcherStats(fm *fieldMatcher, stats *matcherStats, pp printer) {
	fmTrans := fm.fields().transitions
	for _, vm := range fmTrans {
		singleton := vm.fields().singletonMatch
		if singleton != nil {
			stats.bytes += int64(cap(singleton))
		}
		start := vm.fields().startState
		if start == nil {
			continue
		}
		cmStateStats(start, stats, pp)
	}
}

func cmStateStats(state *faState, stats *matcherStats, pp printer) {
	if stats.seenStates[state] {
		return
	}
	stats.seenStates[state] = true
	stats.states++
	stats.bytes += mcFaState(state)
	le := int64(len(state.epsilonClosure))
	if le > stats.maxFanout {
		stats.maxFanout = le
	}
	stats.fanouts += le
	for _, step := range state.table.steps {
		if step != nil {
			cmStateStats(step, stats, pp)
		}
	}
	for _, eps := range state.table.epsilons {
		if eps != nil {
			cmStateStats(eps, stats, pp)
		}
	}
	for _, trans := range state.fieldTransitions {
		cmFieldMatcherStats(trans, stats, pp)
	}
}

func mcSmallTable(st *smallTable) int64 {
	cost := mcSmallTableBase
	cost += int64(cap(st.ceilings))          // this is a []byte, so *1
	cost += mcPointer * int64(cap(st.steps)) // one pointer for each step, doesn't matter whether it's nil or not
	cost += mcPointer * int64(cap(st.epsilons))
	return cost
}

func mcFaState(state *faState) int64 {
	cost := mcFaStateBase
	cost += int64(cap(state.table.ceilings))
	cost += mcPointer * int64(cap(state.table.steps))
	cost += mcPointer * int64(cap(state.table.epsilons))
	cost += mcPointer * int64(cap(state.fieldTransitions))
	cost += mcPointer * int64(cap(state.epsilonClosure))
	return cost
}
