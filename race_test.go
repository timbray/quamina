package quamina

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
)

func TestConcurrencyCore(t *testing.T) {
	testConcurrency(t, newCoreMatcher())
}

func testConcurrency(t *testing.T, m matcher) {
	var (
		goroutines = 4
		n          = 500
		tasks      = 6
	)

	log.Printf("TestConcurrency %T goroutines: %d, tasks: %d",
		m, goroutines, tasks)

	populate := func() {
		for i := 0; i < n; i++ {
			p := fmt.Sprintf(`{"like":["tacos","queso"],"want":[%d]}`, i)
			if err := m.addPattern(i, p); err != nil {
				t.Fatal(err)
			}
		}
	}

	// depopulate := func() {
	//      for i := 0; i < n; i += 2 {
	//              if err := m.deletePattern(i); err != nil {
	//                      t.Fatal(err)
	//              }
	//      }
	// }

	query := func(verify bool) {
		f := newJSONFlattener()

		for i := 0; i < n; i++ {
			e := fmt.Sprintf(`{"like":"tacos","want":%d}`, i)
			fs, err := f.Flatten([]byte(e), m.(*coreMatcher))
			if err != nil {
				t.Fatal(err)
			}
			if got, err := m.matchesForFields(fs); err != nil {
				t.Fatal(err)
			} else if verify && len(got) != 1 {
				t.Fatal(got)
			}
		}
	}

	wg := sync.WaitGroup{}
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			// We defer to get Done called after a t.Fatal().
			defer wg.Done()
			for j, k := range rand.Perm(tasks) {
				switch k {
				case 0, 1:
					populate()
				// case 1:
				//      depopulate()
				default:
					query(false)
				}
				log.Printf("task %d,%d (%d) complete", i, j, k)
			}
		}(i)
	}
	wg.Wait()
}
