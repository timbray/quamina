package quamina

import (
	"fmt"
	"time"
)

// Erlang-flavored accumulators for integers, times, and durations. This was written in error, when a misinterpreted
// unit test seemed to be showing that mutex.Lock()/Unlock() was creating a severe bottleneck.  When the smoke cleared,
// I liked this too much to discard it.

const PrunerStatsBuffer = 100

type asyncPrunerStats struct {
	Live            intAccumulator
	Added           intAccumulator
	Deleted         intAccumulator
	Emitted         intAccumulator
	Filtered        intAccumulator
	LastRebuilt     timeAccumulator
	RebuildDuration durationAccumulator
	RebuildPurged   intAccumulator
}

func (s *asyncPrunerStats) String() string {
	return fmt.Sprintf("live %d added %d deleted %d emitted %d filtered %d purged %d",
		s.Live.get(), s.Added.get(), s.Deleted.get(), s.Emitted.get(), s.Filtered.get(), s.RebuildPurged.get())
}

func newAsyncPrunerStats() *asyncPrunerStats {
	nps := &asyncPrunerStats{
		Live:            newIntAccumulator(),
		Added:           newIntAccumulator(),
		Deleted:         newIntAccumulator(),
		Emitted:         newIntAccumulator(),
		Filtered:        newIntAccumulator(),
		LastRebuilt:     newTimeAccumulator(),
		RebuildDuration: newDurationAccumulator(),
		RebuildPurged:   newIntAccumulator(),
	}
	nps.RebuildDuration.set(time.Duration(0))
	return nps
}

type accumulatorOpcode byte

const (
	aoSet accumulatorOpcode = iota
	aoGet
	aoBump
)

type intAoMessage struct {
	val int64
	op  accumulatorOpcode
}

type intAccumulator struct {
	input  chan intAoMessage
	output chan int64
	val    int64
}

func newIntAccumulator() intAccumulator {
	accum := intAccumulator{
		input:  make(chan intAoMessage, PrunerStatsBuffer),
		output: make(chan int64, PrunerStatsBuffer),
		val:    0,
	}
	go accum.loop()
	return accum
}

func (a intAccumulator) loop() {
	for message := range a.input {
		switch message.op {
		case aoSet:
			a.val = message.val
		case aoGet:
			a.output <- a.val
		case aoBump:
			a.val += message.val
		}
	}
}

func (a intAccumulator) get() int64 {
	a.input <- intAoMessage{op: aoGet}
	return <-a.output
}

func (a intAccumulator) set(val int64) {
	a.input <- intAoMessage{op: aoSet, val: val}
}

func (a intAccumulator) bump(delta int64) {
	a.input <- intAoMessage{op: aoBump, val: delta}
}

type durationAoMessage struct {
	val time.Duration
	op  accumulatorOpcode
}

type durationAccumulator struct {
	input  chan durationAoMessage
	output chan time.Duration
	val    time.Duration
}

func (a durationAccumulator) loop() {
	for message := range a.input {
		switch message.op {
		case aoSet:
			a.val = message.val
		case aoGet:
			a.output <- a.val
		default:
			panic("Bad duration accumulator op")
		}
	}
}

func newDurationAccumulator() durationAccumulator {
	zero, _ := time.ParseDuration("0s")
	accum := durationAccumulator{
		input:  make(chan durationAoMessage, PrunerStatsBuffer),
		output: make(chan time.Duration, PrunerStatsBuffer),
		val:    zero,
	}
	go accum.loop()
	return accum
}

func (a durationAccumulator) get() time.Duration {
	a.input <- durationAoMessage{op: aoGet}
	return <-a.output
}

func (a durationAccumulator) set(val time.Duration) {
	a.input <- durationAoMessage{op: aoSet, val: val}
}

type timeAoMessage struct {
	val time.Time
	op  accumulatorOpcode
}

type timeAccumulator struct {
	input  chan timeAoMessage
	output chan time.Time
	val    time.Time
}

func (a timeAccumulator) loop() {
	for message := range a.input {
		switch message.op {
		case aoSet:
			a.val = message.val
		case aoGet:
			a.output <- a.val
		default:
			panic("Bad time accumulator op")
		}
	}
}

func newTimeAccumulator() timeAccumulator {
	accum := timeAccumulator{
		input:  make(chan timeAoMessage, PrunerStatsBuffer),
		output: make(chan time.Time, PrunerStatsBuffer),
		val:    time.Now(),
	}
	go accum.loop()
	return accum
}

func (a timeAccumulator) get() time.Time {
	a.input <- timeAoMessage{op: aoGet}
	return <-a.output
}

func (a timeAccumulator) set(val time.Time) {
	a.input <- timeAoMessage{op: aoSet, val: val}
}

// sane verifies that certain prunerStats are not negative.
func (s *asyncPrunerStats) sane() error {
	if s.Live.get() < 0 {
		return fmt.Errorf("prunerStats.Live is negative")
	}

	if s.Added.get() < 0 {
		return fmt.Errorf("prunerStats.Added is negative")
	}

	if s.Deleted.get() < 0 {
		return fmt.Errorf("prunerStats.Deleted is negative")
	}

	if s.Filtered.get() < 0 {
		return fmt.Errorf("prunerStats.Filtered is negative")
	}

	return nil
}
