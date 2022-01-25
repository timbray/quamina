package quamina

type matchSet struct {
	set map[X]bool
}

func newMatchSet() *matchSet {
	return &matchSet{set: make(map[X]bool)}
}

func (m *matchSet) addX(x X) {
	m.set[x] = true
}
func (m *matchSet) removeX(x X) {
	delete(m.set, x)
}
func (m *matchSet) contains(x X) bool {
	_, ok := m.set[x]
	return ok
}

func (m *matchSet) subtractXs(xs []X) {
	for _, x := range xs {
		delete(m.set, x)
	}
}

func (m *matchSet) addSet(addend *matchSet) {
	for x := range addend.set {
		m.set[x] = true
	}
}

func (m *matchSet) matches() []X {
	matches := make([]X, 0, len(m.set))
	for x := range m.set {
		matches = append(matches, x)
	}
	return matches
}
