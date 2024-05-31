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
	nextStep := faNext{steps: []*faState{&nextState}}

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

/* TODO: Restore (sigh, going to be a lot of tedious work)
func TestFuzzPack(t *testing.T) {
	seeds := []int64{9, 81, 1729, 8, 64, 512, 7, 49, 343, 6, 36, 216, 5, 25, 125}
	for _, seed := range seeds {
		fuzzPack(t, seed)
	}
}

func fuzzPack(t *testing.T, seed int64) {
	t.Helper()

	rand.Seed(seed)
	var used [byteCeiling]bool
	var unpacked unpackedTable[*dfaStep]

	// we're going to full up an unPackedTable with byte[*dfaStep] mappings, 30 clusters of between one and
	//  five adjacent bytes mapped to the same *dfaStep.  Then we'll pack it and verify that the indexing works,
	//  then unpack it again and make sure it's the same
	for i := 0; i < 30; i++ {
		var clusterLength, clusterBase int32
		for {
			//nolint:gosec
			clusterLength = rand.Int31n(4) + 1
			//nolint:gosec
			clusterBase = rand.Int31n(int32(byteCeiling - 6))
			var u int32
			for u = 0; u < clusterLength; u++ {
				if used[clusterBase+u] {
					break
				}
			}
			if u == clusterLength {
				for u = 0; u < clusterLength; u++ {
					used[clusterBase+u] = true
				}
				break
			}
		}

		xx := &dfaStep{table: newSmallTable[*dfaStep]()}
		var u int32
		for u = 0; u < clusterLength; u++ {
			unpacked[clusterBase+u] = xx
		}
	}
	packed := &smallTable[*dfaStep]{}
	packed.pack(&unpacked)
	for i := 0; i < byteCeiling; i++ {
		if unpacked[i] != packed.step(byte(i)) {
			t.Errorf("T1 seed %d at %d", seed, i)
		}
	}
	reUnpacked := unpackTable(packed)
	for i := range reUnpacked {
		if unpacked[i] != reUnpacked[i] {
			t.Errorf("T2 seed %d unpacked/reUnpacked differ position %d", seed, i)
		}
	}
	rePacked := &smallTable[*dfaStep]{}
	rePacked.pack(reUnpacked)
	for i, c := range rePacked.ceilings {
		if c != packed.ceilings[i] {
			t.Errorf("seed %d ceilings differ at %d wanted %d got %d", seed, i, c, packed.ceilings[i])
		}
		if packed.steps[i] != rePacked.steps[i] {
			t.Errorf("seed %d ssteps differ at %d", seed, i)
		}
	}
}
*/
