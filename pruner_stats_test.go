package quamina

import (
	"fmt"
	"testing"
	"time"
)

func TestDurationAccumulator(t *testing.T) {
	val1, _ := time.ParseDuration("1s")
	val2, _ := time.ParseDuration("2s")
	da := newDurationAccumulator()
	done := make(chan bool)
	go aoSetDurations(t, da, val1, done)
	go aoSetDurations(t, da, val2, done)

	<-done
	<-done

	got := da.get()
	switch got {
	case val1:
		fmt.Println("Yay 1s")
	case val2:
		fmt.Println("Yay 2s")
	default:
		t.Errorf("got wrong duration")
	}
}

func aoSetDurations(t *testing.T, ti durationAccumulator, val time.Duration, done chan bool) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		ti.set(val)
	}
	done <- true
}

func TestTimeAccumulator(t *testing.T) {
	// set initial value
	now := time.Now()
	val1 := now.Add(time.Second)
	val2 := now.Add(-1 * time.Second)
	ta := newTimeAccumulator()
	done := make(chan bool)

	// thread sets initial - 1sec 1000 times
	go aoSetTimes(t, ta, val1, done)

	// thread sets initial + 1sec 1000 times
	go aoSetTimes(t, ta, val2, done)

	<-done
	<-done

	// val better be either +1 or -1
	got := ta.get()
	switch {
	case got.Equal(val1):
		fmt.Println("Yay +")
	case got.Equal(val2):
		fmt.Println("Yay -")
	default:
		t.Errorf("got wrong time")
	}
}

func aoSetTimes(t *testing.T, ti timeAccumulator, val time.Time, done chan bool) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		ti.set(val)
	}
	done <- true
}

func TestIntAccumulator(t *testing.T) {
	// set initial value
	ia := newIntAccumulator()
	ia.set(20)
	done := make(chan bool)

	// thread sends 1000 +2's
	go tiaAdd1000p2(t, ia, done)

	// thread sends 1000 -1's
	go tiaSub1000(t, ia, done)

	// wait for threads
	<-done
	<-done

	// val better be 1020
	if ia.get() != 20+2000-1000 {
		t.Errorf("got %d wanted 520", ia.get())
	}
}

func tiaAdd1000p2(t *testing.T, a intAccumulator, done chan bool) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		a.bump(2)
	}
	done <- true
}

func tiaSub1000(t *testing.T, a intAccumulator, done chan bool) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		a.bump(-1)
	}
	done <- true
}
