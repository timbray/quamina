package quamina

// matchSet is what it says on the tin; implements a set semantic on matches, which are of type X. These could all
//  be implemented as match[X]bool but this makes the calling code more readable.
type matchSet struct {
	set map[X]bool
}

func newMatchSet() *matchSet {
	return &matchSet{set: make(map[X]bool)}
}

// this is klunky and slow but I don't want to put a lock in the access path
func (m *matchSet) addX(x X) {
	newSet := make(map[X]bool)
	for k := range m.set {
		newSet[k] = true
	}
	newSet[x] = true
	m.set = newSet
}

func (m *matchSet) contains(x X) bool {
	_, ok := m.set[x]
	return ok
}

func (m *matchSet) matches() []X {
	matches := make([]X, 0, len(m.set))
	for x := range m.set {
		matches = append(matches, x)
	}
	return matches
}
