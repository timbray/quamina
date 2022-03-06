package quamina

import (
	"math/rand"
	"testing"
)

func TestMakeSmallTable(t *testing.T) {
	tMST(t, []byte{1, 2, 33})
	tMST(t, []byte{0, 1, 2, 33, byte(ByteCeiling - 1)})
	tMST(t, []byte{2, 33, byte(ByteCeiling - 1)})
	tMST(t, []byte{0, 1, 2, 33})
}

func tMST(t *testing.T, b []byte) {
	comp := newSmallTable()
	sdef := newSmallTable()
	comp.addRangeSteps(0, ByteCeiling, sdef)
	var steps []smallStep
	for _, pos := range b {
		onestep := newSmallTable()
		steps = append(steps, onestep)
		comp.addByteStep(pos, onestep)
	}
	table := makeSmallTable(sdef, b, steps)
	uComp := unpack(comp)
	uT := unpack(table)
	for i := range uComp {
		if uComp[i] != uT[i] {
			t.Errorf("wrong at %d", i)
		}
	}
}

func TestCombiner(t *testing.T) {

	// "jab"
	A0 := newSmallTable()
	A1 := newSmallTable()
	A2 := newSmallTable()
	A3 := newSmallTable()
	A0.addByteStep('j', A1)
	A1.addByteStep('a', A2)
	A2.addByteStep('b', A3)
	AFM := newFieldMatcher()
	AFM.transitions["AFM"] = newValueMatcher()
	st := newSmallTransition(AFM)
	A3.addByteStep(ValueTerminator, st)

	// *ay*
	B0 := newSmallTable()
	B1 := newSmallTable()
	B2 := newSmallTable()
	B0.addRangeSteps(0, ByteCeiling, B0)
	B0.addByteStep('a', B1)
	B1.addRangeSteps(0, ByteCeiling, B0)
	B1.addByteStep('y', B2)
	BFM := newFieldMatcher()
	BFM.transitions["BFM"] = newValueMatcher()
	st = newSmallTransition(BFM)
	B2.addRangeSteps(0, ByteCeiling, st)

	combo := mergeOne(A0, B0, make(map[stepKey]smallStep))

	vm := &valueMatcher{
		startTable: combo.SmallTable(),
	}
	matches := vm.transitionOn([]byte("jab"))
	if len(matches) != 1 || matches[0].transitions["AFM"] == nil {
		t.Error("wanted AFM")
	}
	matches = vm.transitionOn([]byte("jayhawk"))
	if len(matches) != 1 || matches[0].transitions["BFM"] == nil {
		t.Error("wanted BFM")
	}

	// "*yy"
	C0 := newSmallTable()
	C1 := newSmallTable()
	C2 := newSmallTable()
	C0.addRangeSteps(0, ByteCeiling, C0)
	C0.addByteStep('y', C1)
	C1.addRangeSteps(0, ByteCeiling, C0)
	C1.addByteStep('y', C2)
	C2.addRangeSteps(0, ByteCeiling, C0)
	CFM := newFieldMatcher()
	CFM.transitions["CFM"] = newValueMatcher()
	st = newSmallTransition(CFM)
	C2.addByteStep(ValueTerminator, st)

	combo = mergeOne(vm.startTable, C0, make(map[stepKey]smallStep))
	vm.startTable = combo.SmallTable()
	matches = vm.transitionOn([]byte("jab"))
	if len(matches) != 1 || matches[0].transitions["AFM"] == nil {
		t.Error("wanted AFM")
	}
	matches = vm.transitionOn([]byte("jayhawk"))
	if len(matches) != 1 || matches[0].transitions["BFM"] == nil {
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

	st1 := newSmallTable()

	st := smallTable{
		slices: stSlices{
			ceilings: []uint8{2, 3, byte(ByteCeiling)},
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
	var used [ByteCeiling]bool
	var unpacked unpackedTable

	// we're going to full up an unPackedTable with byte[smallStep] mappings, 30 clusters of between one and
	//  five adjacent bytes mapped to the same smallStep.  Then we'll pack it and verify that the indexing works,
	//  then unpack it again and make sure it's the same
	for i := 0; i < 30; i++ {
		var clusterLength, clusterBase int32
		for {
			clusterLength = rand.Int31n(4) + 1
			clusterBase = rand.Int31n(int32(ByteCeiling - 6))
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
	for i := 0; i < ByteCeiling; i++ {
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
