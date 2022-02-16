package quamina

import (
	"math/rand"
	"testing"
)

func TestUnpack(t *testing.T) {

	st1 := newSmallTable()

	st := smallTable{
		slices: &stSlices{
			ceilings: []uint8{2, 3, byte(Utf8ByteCeiling)},
			steps:    []smallStep{nil, st1, nil},
		},
	}
	u := unpack(&st)
	for i := range u {
		if i == 2 {
			if u[i] != st1 {
				t.Error("Not in pos 2")
			}
		} else {
			if u[i] != nil {
				t.Errorf("Non-nil at %d", i)
			}
		}
	}
}

func TestFuzzPack(t *testing.T) {
	seeds := []int64{9, 81, 1729, 8, 64, 512, 7, 49, 343, 6, 36, 216, 5, 25, 125}
	for _, seed := range seeds {
		fuzzPack(t, seed)
	}
}

func fuzzPack(t *testing.T, seed int64) {
	rand.Seed(seed)
	var used [Utf8ByteCeiling]bool
	var unpacked unpackedTable

	// we're going to full up an unPackedTable with byte[smallStep] mappings, 30 clusters of between one and
	//  five adjacent bytes mapped to the same smallStep.  Then we'll pack it and verify that the indexing works,
	//  then unpack it again and make sure it's the same
	for i := 0; i < 30; i++ {
		var clusterLength, clusterBase int32
		for {
			clusterLength = rand.Int31n(4) + 1
			clusterBase = rand.Int31n(int32(Utf8ByteCeiling - 6))
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

		xx := newSmallTable()
		var u int32
		for u = 0; u < clusterLength; u++ {
			unpacked[clusterBase+u] = xx
		}
	}
	packed := &smallTable{}
	packed.pack(&unpacked)
	for i := 0; i < Utf8ByteCeiling; i++ {
		if unpacked[i] != packed.step(byte(i)) {
			t.Errorf("T1 seed %d at %d", seed, i)
		}
	}
	reUnpacked := unpack(packed)
	for i := range reUnpacked {
		if unpacked[i] != reUnpacked[i] {
			t.Errorf("T2 seed %d unpacked/reUnpacked differ position %d", seed, i)
		}
	}
	rePacked := &smallTable{}
	rePacked.pack(reUnpacked)
	for i, c := range rePacked.slices.ceilings {
		if c != packed.slices.ceilings[i] {
			t.Errorf("seed %d ceilings differ at %d wanted %d got %d", seed, i, c, packed.slices.ceilings[i])
		}
		if packed.slices.steps[i] != rePacked.slices.steps[i] {
			t.Errorf("seed %d ssteps differ at %d", seed, i)
		}
	}
}
