package quamina

import (
	"fmt"
	"strings"
	"testing"
)

// TestMeasureNextStates instruments traverseNFA to measure the actual
// nextStates dynamics: how big does the list get, how many are unique,
// and what's the dedup ratio per byte step?
func TestMeasureNextStates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping heavy pattern test in short mode")
	}
	q, _ := New()
	patCount := 0
	letters := "abcdefghijklmnopqrstuvwxyz"

	for i := 0; i < len(letters); i++ {
		for j := i + 1; j < len(letters); j++ {
			ss := fmt.Sprintf("*%c*%c*", letters[i], letters[j])
			pat := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, ss)
			if err := q.AddPattern(fmt.Sprintf("p%d", patCount), pat); err != nil {
				t.Fatal(err)
			}
			patCount++
		}
	}
	for i := 0; i < len(letters); i++ {
		for j := i + 1; j < len(letters); j++ {
			for k := j + 1; k < len(letters); k++ {
				ss := fmt.Sprintf("*%c*%c*%c*", letters[i], letters[j], letters[k])
				pat := fmt.Sprintf(`{"val": [{"shellstyle": "%s"}]}`, ss)
				if err := q.AddPattern(fmt.Sprintf("p%d", patCount), pat); err != nil {
					t.Fatal(err)
				}
				patCount++
			}
		}
	}

	t.Logf("Added %d patterns", patCount)
	m := q.matcher.(*coreMatcher)
	t.Log(matcherStats(m))

	// Use a short input so we can see per-byte dynamics
	// Get the value matcher's start table to call traverseNFA directly
	root := m.fields().state
	vmf := root.fields()

	// Find the "val" value matcher
	vm, ok := vmf.transitions["val"]
	if !ok {
		t.Fatal("no 'val' transition")
	}
	fields := vm.fields()
	if fields.startTable == nil {
		t.Fatal("no start table")
	}

	// Values include surrounding quotes, matching how the JSON flattener passes them
	val := []byte(`"` + strings.Repeat("abcdef", 3) + `"`)

	// Manually walk the NFA to measure state dynamics
	bufs := newNfaBuffers()
	currentStates := bufs.getBuf1()
	currentStates = append(currentStates, bufs.getStartState(fields.startTable))
	nextStates := bufs.getBuf2()

	stepResult := &stepOut{}
	for index := 0; len(currentStates) != 0 && index <= len(val); index++ {
		var utf8Byte byte
		if index < len(val) {
			utf8Byte = val[index]
		} else {
			utf8Byte = valueTerminator
		}

		totalClosureStates := 0
		for _, state := range currentStates {
			totalClosureStates += len(state.epsilonClosure)
			for _, ecState := range state.epsilonClosure {
				ecState.table.step(utf8Byte, stepResult)
				if stepResult.step != nil {
					nextStates = append(nextStates, stepResult.step)
				}
			}
		}

		// Count unique
		unique := make(map[*faState]bool, len(nextStates))
		for _, s := range nextStates {
			unique[s] = true
		}

		char := string(utf8Byte)
		if utf8Byte == valueTerminator {
			char = "VT"
		}
		t.Logf("byte %2d '%s': current=%d, closure_expand=%d, next_raw=%d, next_unique=%d, dup_ratio=%.1f%%",
			index, char, len(currentStates), totalClosureStates,
			len(nextStates), len(unique),
			100.0*float64(len(nextStates)-len(unique))/max(float64(len(nextStates)), 1))

		swapStates := currentStates
		currentStates = nextStates
		nextStates = swapStates[:0]
	}
}
