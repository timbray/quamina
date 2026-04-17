package quamina

import (
	"errors"
	"math"
	"runtime"
)

// Tracking the memory cost of building NFAs is hard. One can get a pretty exact count of memory usage by accounting
// for faStates as they are created, and then also inserting tracking in the mergeFAs() code, but this is
// expensive and proved unacceptably so for large FAs being built by successive merges.
// So, we'll instead use the runtime.ReadMemStats() call to query the system for how much memory has been allocated
// and use that instead.
// For efficiency, we won't call ReadMemStats() for each new FA, but sample at a rate defined by the
// MeṁWatcherSamplingInterval value.

// memoryMonitor is an interface so that we can have nullMemoryMonitor for code that sometimes needs to
// monitor the budget but other times doesn't
type memoryMonitor interface {
	sample() error
	check() error
}

type nullMemoryMonitor struct{}

func (mm *nullMemoryMonitor) sample() error {
	return nil
}
func (mm *nullMemoryMonitor) check() error {
	return nil
}

var sharedNullMonitor = &nullMemoryMonitor{}

const MemWatcherSamplingInterval = 100

// a samplingMemoryMonitor only exists for one addPattern invocation. The baseAlloc
// is the amount of memory allocated at the start of addPattern. The headroom is
// the amount of memory that may be allocated in this addPattern, given the user-supplied
// memory Budget and the amount already allocated in any previous addPattern calls.
type samplingMemoryMonitor struct {
	baseAlloc uint64 // the memory allocated at the start of addPattern
	headroom  uint64 // the remaining amount of memory that may be allocated in this addPattern
	samples   int    // call ReadMemStats once each MemWatcherSamplingInterval times
}

func newSamplingMemoryMonitor(fields *coreFields) *samplingMemoryMonitor {
	var headroom uint64
	if fields.memoryBudget == 0 {
		headroom = math.MaxInt64
	} else {
		headroom = fields.memoryBudget - fields.currentMemory
	}
	return &samplingMemoryMonitor{
		baseAlloc: bytesAllocated(),
		headroom:  headroom,
	}
}

func (mm *samplingMemoryMonitor) sample() error {
	if mm.samples < MemWatcherSamplingInterval {
		mm.samples++
		return nil
	}

	mm.samples = 0
	return mm.check()
}

func (mm *samplingMemoryMonitor) check() error {
	// HeapInuse can decrease mid-build when GC reclaims spans, so
	// bytesAllocated() may fall below baseAlloc. Clamp the delta to 0
	// rather than letting the uint64 subtraction underflow into a huge
	// value that would spuriously trip the budget.
	current := bytesAllocated()
	if current > mm.baseAlloc && current-mm.baseAlloc > mm.headroom {
		return errors.New("MemoryBudget")
	}
	return nil
}

// Return HeapInuse — bytes held in spans that have at least one live
// object. Unlike TotalAlloc, this tracks retained memory (not cumulative
// allocation), which matches the semantic a user is asking about with
// SetMemoryBudget. The reading is slightly lagged relative to the GC but
// that fuzziness is acceptable for a budget heuristic. In particular, it
// credits the old+new matcher overlap during atomic swap (both are held),
// while not penalizing transient build-time scratch buffers that GC will
// reclaim.
func bytesAllocated() uint64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats.HeapInuse
}

/*
// for the memories
// computes the memory size of Quamina's finite automata. We're ignoring everything except smallTable and faState.

var mcPointer = int64(unsafe.Sizeof(&faState{}))
var mcSmallTableBase = int64(unsafe.Sizeof(smallTable{})) // should include 3* slice descriptor
var mcFaStateBase = int64(unsafe.Sizeof(faState{}))
var mcSlice = int64(unsafe.Sizeof([]*int{}))

func mcSmallTable(st *smallTable) int64 {
	cost := mcSmallTableBase
	cost += int64(cap(st.ceilings))          // this is a []byte, so *1
	cost += mcPointer * int64(cap(st.steps)) // one pointer for each step, doesn't matter whether it's nil or not
	cost += mcPointer * int64(cap(st.epsilons))
	return cost
}

func mcFaState(s *faState) int64 {
	cost := mcFaStateBase
	if s.table != nil {
		cost += mcSmallTable(s.table)
	}
	cost += mcPointer * int64(cap(s.fieldTransitions))
	cost += mcPointer * int64(cap(s.epsilonClosure))
	return cost
}

*/
