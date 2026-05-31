package quamina

import (
	"reflect"
	"sort"
	"testing"
)

// Patterns chosen to maximize merge interleaving on a single field: shared
// prefixes/suffixes, multiple stars, and an exact value.
var incrClosurePatterns = []struct{ x, p string }{
	{"p1", `{"x":[{"shellstyle":"*foo*"}]}`},
	{"p2", `{"x":[{"shellstyle":"*foobar*"}]}`},
	{"p3", `{"x":[{"shellstyle":"foo*"}]}`},
	{"p4", `{"x":[{"shellstyle":"*bar"}]}`},
	{"p5", `{"x":[{"shellstyle":"a*b*c"}]}`},
	{"p6", `{"x":[{"shellstyle":"*x*y*"}]}`},
	{"p7", `{"x":["foobar"]}`},
}

var incrClosureEvents = []string{
	`{"x":"foobar"}`, `{"x":"afoob"}`, `{"x":"foo"}`, `{"x":"xxbar"}`,
	`{"x":"abc"}`, `{"x":"axbyc"}`, `{"x":"nomatch"}`, `{"x":"foobarbaz"}`,
	`{"x":"axxbyyc"}`, `{"x":"bar"}`,
}

func buildAndMatch(t *testing.T, order []int) map[string][]string {
	t.Helper()
	q, _ := New()
	for _, i := range order {
		if err := q.AddPattern(incrClosurePatterns[i].x, incrClosurePatterns[i].p); err != nil {
			t.Fatalf("AddPattern %s: %v", incrClosurePatterns[i].x, err)
		}
	}
	out := make(map[string][]string, len(incrClosureEvents))
	for _, ev := range incrClosureEvents {
		matches, err := q.MatchesForEvent([]byte(ev))
		if err != nil {
			t.Fatalf("MatchesForEvent %s: %v", ev, err)
		}
		ss := make([]string, 0, len(matches))
		for _, x := range matches {
			ss = append(ss, x.(string))
		}
		sort.Strings(ss)
		out[ev] = ss
	}
	return out
}

// TestClosureWalkPrunesClosedSubtree fully closes an NFA, then re-runs the
// closure walk (as epsilonClosure does) with an inspectable buffer. A
// fully-closed NFA must be walked in zero state-visits: every state should be
// pruned because it was closed by the prior epsilonClosure call.
func TestClosureWalkPrunesClosedSubtree(t *testing.T) {
	pp := newPrettyPrinter(99)
	nfa, _ := makeShellStyleFA([]byte(`"*foo*bar*"`), pp)
	epsilonClosure(nfa) // fully close every reachable state

	bufs := newClosureBuffers()
	bufs.gen++
	bufs.walkGen = bufs.gen
	closureForNfa(nfa, bufs)

	if bufs.nfaWalkCount != 0 {
		t.Errorf("re-walking a fully-closed NFA should process 0 states, processed %d", bufs.nfaWalkCount)
	}
}

func TestIncrementalClosureOrderIndependence(t *testing.T) {
	forward := []int{0, 1, 2, 3, 4, 5, 6}
	reverse := []int{6, 5, 4, 3, 2, 1, 0}
	shuffled := []int{3, 0, 6, 1, 5, 2, 4}

	base := buildAndMatch(t, forward)
	if got := buildAndMatch(t, reverse); !reflect.DeepEqual(base, got) {
		t.Errorf("reverse-order matches differ from forward:\nforward=%v\nreverse=%v", base, got)
	}
	if got := buildAndMatch(t, shuffled); !reflect.DeepEqual(base, got) {
		t.Errorf("shuffled-order matches differ from forward:\nforward=%v\nshuffled=%v", base, got)
	}

	// Sanity: the exact-value event must at least match p7 and the wildcards.
	if len(base[`{"x":"foobar"}`]) == 0 {
		t.Errorf(`expected matches for {"x":"foobar"}, got none`)
	}
}
