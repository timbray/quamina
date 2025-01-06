package quamina

import (
	"fmt"
	"testing"
	"time"
)

func TestFAMergePerf(t *testing.T) {
	words := readWWords(t)
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
	nextStep := faNext{states: []*faState{&nextState}}

	st := smallTable{
		ceilings: []uint8{2, 3, byte(byteCeiling)},
		steps:    []*faNext{nil, &nextStep, nil},
	}
	u := unpackTable(&st)
	for i := range u {
		if i == 2 {
			if u[i] != &nextStep {
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
	st := makeSmallTable(nil, []byte{'a'}, []*faNext{{states: []*faState{{}}}})
	so := &stepOut{}
	st.step(0xFE, so)
	st.dStep(0xFE)
}
