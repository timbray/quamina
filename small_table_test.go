package quamina

import (
	"fmt"
	"testing"
	"time"
)

func TestFAMergePerf(t *testing.T) {
	words := readWWords(t, 0)
	patterns := make([]string, 0, len(words))
	for _, word := range words {
		pattern := fmt.Sprintf(`{"x": [ "%s" ] }`, string(word))
		patterns = append(patterns, pattern)
	}
	before := time.Now()
	q, _ := New()
	for _, pattern := range patterns {
		err := q.AddPattern(pattern, pattern)
		if err != nil {
			t.Error("ap: " + err.Error())
		}
	}
	elapsed := float64(time.Since(before).Milliseconds())

	for _, word := range words {
		event := fmt.Sprintf(`{"x": "%s"}`, string(word))
		matches, err := q.MatchesForEvent([]byte(event))
		if err != nil {
			t.Error("M4: " + err.Error())
		}
		if len(matches) != 1 {
			t.Errorf("wanted 1 got %d", len(matches))
		}
	}
	perSecond := float64(len(patterns)) / (elapsed / 1000.0)
	fmt.Printf("%.2f addPatterns/second with letter patterns\n\n", perSecond)
}

func TestUnpack(t *testing.T) {
	st1 := newSmallTable()
	nextState := faState{
		table:            st1,
		fieldTransitions: nil,
	}

	st := smallTable{
		ceilings: []uint8{2, 3, byte(byteCeiling)},
		steps:    []*faState{nil, &nextState, nil},
	}
	u := unpackTable(&st)
	for i := range u {
		if i == 2 {
			if u[i] != &nextState {
				t.Error("Not in pos 2")
			}
		} else {
			if u[i] != nil {
				t.Errorf("Non-nil at %d", i)
			}
		}
	}
}

func TestDodgeBadUTF8(t *testing.T) {
	st := makeSmallTable(nil, []byte{'a'}, []*faState{{}})
	so := &stepOut{}
	st.step(0xFE, so)
	st.dStep(0xFE)
}

func TestSmallTableIterator(t *testing.T) {
	bytevals := []byte{11, 22, 'a', 'b', 'c', 'z', 0xf3}
	var s1, s2, s3, s4, s5, s6, s7 faState
	var steps = []*faState{&s1, &s2, &s3, &s4, &s5, &s6, &s7}
	st := makeSmallTable(nil, bytevals, steps)
	wanted := make([]*faState, byteCeiling)
	for i, byteval := range bytevals {
		wanted[byteval] = steps[i]
	}
	iter := newSTIterator(st, nil)
	for iter.hasNext() {
		utf8byte, step := iter.next()
		if wanted[utf8byte] != step {
			t.Errorf("at u=%x wanted %p got %p", utf8byte, wanted[utf8byte], step)
		}
	}
	iter.byteIndex = 0
	iter.ceilingIndex = 0
	for i := 0; i < byteCeiling; i++ {
		state := iter.nextState()
		if wanted[i] != state {
			t.Errorf("at u=%x wanted %p got %p", i, wanted[i], state)
		}
	}
	unpacked := unpackTable(st)
	iter = newSTIterator(st, &iter)
	for iter.hasNext() {
		utf8byte, step := iter.next()
		if unpacked[utf8byte] != step {
			t.Errorf("Wrong unpacked at %x", utf8byte)
		}
	}

	//	ceilings:-|  3|-|   5|-|0x34|-| x35|-|byteCeiling|
	//	states:---|nil|-|&ss1|-| nil|-|&ss2|-|        nil|
	bytevals = []byte{3, 5, 0x34, 0x35}
	var ss1, ss2 faState
	steps = []*faState{nil, &ss1, nil, &ss2}
	st = makeSmallTable(nil, bytevals, steps)
	wanted = make([]*faState, byteCeiling)
	for i, byteval := range bytevals {
		wanted[byteval] = steps[i]
	}
	iter = newSTIterator(st, &iter)
	for iter.hasNext() {
		utf8byte, step := iter.next()
		if wanted[utf8byte] != step {
			t.Errorf("at u=%x wanted %p got %p", utf8byte, wanted[utf8byte], step)
		}
	}
	unpacked = unpackTable(st)
	iter = newSTIterator(st, &iter)
	for iter.hasNext() {
		utf8byte, step := iter.next()
		if unpacked[utf8byte] != step {
			t.Errorf("Wrong unpacked at %x", utf8byte)
		}
	}
}
