package quamina

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestMemoryBudgetBasic(t *testing.T) {
	cm := newCoreMatcher()
	var err error
	i1, i2 := cm.getMemoryBudget()
	if i1 != 0 || i2 != 0 {
		t.Error("Ouch 1")
	}
	var budget uint64 = 1000
	i2, err = cm.setMemoryBudget(budget)
	if err != nil {
		t.Error(err)
	}
	if i2 != 0 {
		t.Errorf("i1/2 %d/%d\n", i1, i2)
	}
	err = cm.addPattern("x", `{"x":["abc"]}`)
	if err != nil {
		t.Error(err)
	}
	i1, _ = cm.getMemoryBudget()
	if i1 != budget {
		t.Errorf("i1: %d\n", i1)
	}

	_, err = cm.setMemoryBudget(0x10)
	if err == nil {
		t.Error("allowed invalid memory budget reduction")
	}
	i200 := iString(t, 200)
	cm = newCoreMatcher()
	_, err = cm.setMemoryBudget(0x5000)
	if err != nil {
		t.Error(err)
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, "xyz"))
	if err != nil {
		t.Error(err)
	}
	_, _ = cm.getMemoryBudget()

	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, "abc"))
	if err != nil {
		t.Error(err)
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, i200))
	if err == nil {
		t.Error("Accepted overly long string")
	} else {
		fmt.Println("E: " + err.Error())
	}
}

func TestMemoryStress(t *testing.T) {
	xx := 1e6
	fmt.Println(xx)
	words := readWWords(t, 20)
	q, _ := New()
	source := rand.NewSource(293591)
	type patternMemory struct {
		pattern string
		mem     uint64
	}
	pms := []patternMemory{}
	for _, word := range words {
		//nolint:gosec
		starAt := source.Int63() % 6
		starWord := string(word[:starAt]) + "*" + string(word[starAt:])
		pattern := fmt.Sprintf(`{"x": ["%s"]}`, starWord)
		err := q.AddPattern("x", pattern)
		if err != nil {
			t.Error(err)
		}
		_, mem := q.GetMemoryBudget()
		pms = append(pms, patternMemory{pattern, mem})
	}
	q, _ = New()
	for i, pm := range pms {
		lowBudget := pm.mem / 2
		_, mem := q.GetMemoryBudget()
		if lowBudget < mem {
			lowBudget = mem + 1
		}
		_, err := q.SetMemoryBudget(lowBudget)
		if err != nil {
			t.Error(err)
		}
		err = q.AddPattern("x", pm.pattern)
		if err == nil {
			t.Errorf("allowed at %d, mem %d", i, pm.mem)
		} else {
			_, err = q.SetMemoryBudget(pm.mem * 2)
			if err != nil {
				t.Error(err)
			}
			err = q.AddPattern("x", pm.pattern)
			if err != nil {
				t.Errorf("Err on add at %d: %s", i, err.Error())
			}
		}
	}
}

func iString(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	for i := range b {
		b[i] = 'i'
	}
	return string(b)
}

func TestStringFA(t *testing.T) {
	cm := newCoreMatcher()
	var err error
	_, err = cm.setMemoryBudget(10000)
	if err != nil {
		t.Error("SMB")
	}
	// force it to build an FA
	err = cm.addPattern("x", `{"x": ["x"]}`)
	if err != nil {
		t.Error("x?")
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, iString(t, 100)))
	if err == nil {
		t.Error("should not succeed")
	}
	_, err = cm.setMemoryBudget(10000000)
	if err != nil {
		t.Error(err)
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, iString(t, 100)))
	if err != nil {
		t.Error("should succeed")
	}
	_, current := cm.getMemoryBudget()
	cm = newCoreMatcher()
	_, _ = cm.setMemoryBudget(current)
	err = cm.addPattern("x", `{"x": ["x"]}`)
	if err != nil {
		t.Error("x?")
	}
	i100 := iString(t, 100)
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, i100))
	if err != nil {
		t.Error("should succeed")
	}

	_, current = cm.getMemoryBudget()
	cm = newCoreMatcher()
	// Leave a margin larger than typical TotalAlloc variance across Go
	// versions and the race detector's bookkeeping overhead. A 1-byte
	// margin was flaky under Go 1.23 + -race; the i100 pattern's FA
	// comfortably exceeds any reasonable margin below its own cost.
	_, _ = cm.setMemoryBudget(current - uint64(len(i100)))
	err = cm.addPattern("x", `{"x": ["x"]}`)
	if err != nil {
		t.Error("x?")
	}
	err = cm.addPattern("x", fmt.Sprintf(`{"x": ["%s"]}`, i100))
	if err == nil {
		t.Error("should fail")
	}
}
