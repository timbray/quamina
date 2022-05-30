package core

import (
	"github.com/timbray/quamina/constants"
	"math/rand"
	"testing"
)

func TestMakeSmallTable(t *testing.T) {
	tMST(t, []byte{1, 2, 33})
	tMST(t, []byte{0, 1, 2, 33, byte(constants.ByteCeiling - 1)})
	tMST(t, []byte{2, 33, byte(constants.ByteCeiling - 1)})
	tMST(t, []byte{0, 1, 2, 33})
}

func tMST(t *testing.T, b []byte) {
	comp := &dfaStep{table: newSmallTable[*dfaStep]()}
	sdef := &dfaStep{table: newSmallTable[*dfaStep]()}
	comp.table.addRangeSteps(0, constants.ByteCeiling, sdef)
	var steps []*dfaStep
	for _, pos := range b {
		onestep := &dfaStep{table: newSmallTable[*dfaStep]()}
		steps = append(steps, onestep)
		comp.table.addByteStep(pos, onestep)
	}
	table := makeSmallDfaTable(sdef, b, steps)
	uComp := unpackTable(comp.table)
	uT := unpackTable(table)
	for i := range uComp {
		if uComp[i] != uT[i] {
			t.Errorf("wrong at %d", i)
		}
	}
}

func newDfaTransition(f *fieldMatcher) *dfaStep {
	return &dfaStep{table: newSmallTable[*dfaStep](), fieldTransitions: []*fieldMatcher{f}}
}
func TestCombiner(t *testing.T) {

	// "jab"
	A0 := &dfaStep{table: newSmallTable[*dfaStep]()}
	A1 := &dfaStep{table: newSmallTable[*dfaStep]()}
	A2 := &dfaStep{table: newSmallTable[*dfaStep]()}
	A3 := &dfaStep{table: newSmallTable[*dfaStep]()}
	A0.table.addByteStep('j', A1)
	A1.table.addByteStep('a', A2)
	A2.table.addByteStep('b', A3)
	AFM := newFieldMatcher()
	AFM.fields().transitions["AFM"] = newValueMatcher()
	st := newDfaTransition(AFM)
	A3.table.addByteStep(constants.ValueTerminator, st)

	// *ay*
	B0 := &dfaStep{table: newSmallTable[*dfaStep]()}
	B1 := &dfaStep{table: newSmallTable[*dfaStep]()}
	B2 := &dfaStep{table: newSmallTable[*dfaStep]()}
	B0.table.addRangeSteps(0, constants.ByteCeiling, B0)
	B0.table.addByteStep('a', B1)
	B1.table.addRangeSteps(0, constants.ByteCeiling, B0)
	B1.table.addByteStep('y', B2)
	BFM := newFieldMatcher()
	BFM.fields().transitions["BFM"] = newValueMatcher()
	st = newDfaTransition(BFM)
	B2.table.addRangeSteps(0, constants.ByteCeiling, st)

	combo := mergeOneDfaStep(A0, B0, make(map[dfaStepKey]*dfaStep))

	state := &vmFields{startDfa: combo.table}
	vm := newValueMatcher()
	vm.update(state)
	matches := vm.transitionOn([]byte("jab"))
	if len(matches) != 1 || matches[0].fields().transitions["AFM"] == nil {
		t.Error("wanted AFM")
	}
	matches = vm.transitionOn([]byte("jayhawk"))
	if len(matches) != 1 || matches[0].fields().transitions["BFM"] == nil {
		t.Error("wanted BFM")
	}

	// "*yy"
	C0 := &dfaStep{table: newSmallTable[*dfaStep]()}
	C1 := &dfaStep{table: newSmallTable[*dfaStep]()}
	C2 := &dfaStep{table: newSmallTable[*dfaStep]()}
	C0.table.addRangeSteps(0, constants.ByteCeiling, C0)
	C0.table.addByteStep('y', C1)
	C1.table.addRangeSteps(0, constants.ByteCeiling, C0)
	C1.table.addByteStep('y', C2)
	C2.table.addRangeSteps(0, constants.ByteCeiling, C0)
	CFM := newFieldMatcher()
	CFM.fields().transitions["CFM"] = newValueMatcher()
	st = newDfaTransition(CFM)
	C2.table.addByteStep(constants.ValueTerminator, st)

	combo = mergeOneDfaStep(&dfaStep{table: vm.getFields().startDfa}, C0, make(map[dfaStepKey]*dfaStep))
	vm.update(&vmFields{startDfa: combo.table})
	matches = vm.transitionOn([]byte("jab"))
	if len(matches) != 1 || matches[0].fields().transitions["AFM"] == nil {
		t.Error("wanted AFM")
	}
	matches = vm.transitionOn([]byte("jayhawk"))
	if len(matches) != 1 || matches[0].fields().transitions["BFM"] == nil {
		t.Error("wanted BFM")
	}
	matches = vm.transitionOn([]byte("xayjjyy"))
	if len(matches) != 2 {
		t.Error("less than two matches")
	}
	if !(contains(matches, BFM) && contains(matches, CFM)) {
		t.Error("should have BFM and CFM")
	}
}

func TestUnpack(t *testing.T) {

	st1 := &dfaStep{table: newSmallTable[*dfaStep]()}

	st := smallTable[*dfaStep]{
		ceilings: []uint8{2, 3, byte(constants.ByteCeiling)},
		steps:    []*dfaStep{nil, st1, nil},
	}
	u := unpackTable(&st)
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
	var used [constants.ByteCeiling]bool
	var unpacked unpackedTable[*dfaStep]

	// we're going to full up an unPackedTable with byte[*dfaStep] mappings, 30 clusters of between one and
	//  five adjacent bytes mapped to the same *dfaStep.  Then we'll pack it and verify that the indexing works,
	//  then unpack it again and make sure it's the same
	for i := 0; i < 30; i++ {
		var clusterLength, clusterBase int32
		for {
			clusterLength = rand.Int31n(4) + 1
			clusterBase = rand.Int31n(int32(constants.ByteCeiling - 6))
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
	for i := 0; i < constants.ByteCeiling; i++ {
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
