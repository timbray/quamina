package quamina

// this needs to exist so that all all the lists containing a single step to X, or the triple step to X,Y,Z are the
// same list, so that pack/unpack work properly. In a large majority of cases, there's only one step in the list, so
// those are handled straightforwardly with a map. Otherwise, we laboriously look through all the lists for a match.
// In Java I'd implement a hashCode() method and everything would be a hash, but I haven't learned yet what the Go
// equivalent is.
type dfaMemory struct {
	singletons map[*nfaStep]*dfaStep
	plurals    []perList
}
type perList struct {
	list []*nfaStep
	dfa  *dfaStep
}

func newDfaMemory() *dfaMemory {
	return &dfaMemory{singletons: make(map[*nfaStep]*dfaStep)}
}

func (m *dfaMemory) rememberDfaForList(dfa *dfaStep, steps ...*nfaStep) {
	if len(steps) == 1 {
		m.singletons[steps[0]] = dfa
	} else {
		m.plurals = append(m.plurals, perList{list: steps, dfa: dfa})
	}
}

func (m *dfaMemory) dfaForNfas(steps ...*nfaStep) (*dfaStep, bool) {
	if len(steps) == 1 {
		d, ok := m.singletons[steps[0]]
		return d, ok
	}
	for _, p := range m.plurals {
		if nfaListsEqual(p.list, steps) {
			return p.dfa, true
		}
	}
	return nil, false
}

func nfaListsEqual(l1, l2 []*nfaStep) bool {
	if len(l1) != len(l2) {
		return false
	}
	for _, e1 := range l1 {
		if !nfaListContains(l2, e1) {
			return false
		}
	}
	return true
}

func nfaListContains(list []*nfaStep, step *nfaStep) bool {
	for _, e := range list {
		if e == step {
			return true
		}
	}
	return false
}

type listMaker struct {
	singletons map[*nfaStep]*nfaStepList
	plurals    []*nfaStepList
}

func newListMaker() *listMaker {
	return &listMaker{singletons: make(map[*nfaStep]*nfaStepList)}
}

func (l *listMaker) getSingleton(step *nfaStep) *nfaStepList {
	already, ok := l.singletons[step]
	if ok {
		return already
	}
	list := &nfaStepList{steps: []*nfaStep{step}}
	l.singletons[step] = list
	return list
}

func (l *listMaker) getList(steps ...*nfaStep) *nfaStepList {
	if len(steps) == 1 {
		return l.getSingleton(steps[0])
	}

	for _, already := range l.plurals {
		if listsAreEqual(already.steps, steps) {
			return already
		}
	}
	list := &nfaStepList{steps: steps}
	l.plurals = append(l.plurals, list)
	return list
}

func listsAreEqual(l1, l2 []*nfaStep) bool {
	if len(l1) != len(l2) {
		return false
	}
	for _, step := range l1 {
		if !listMakerContains(l2, step) {
			return false
		}
	}
	return true
}

func listMakerContains(list []*nfaStep, step *nfaStep) bool {
	for _, fromList := range list {
		if step == fromList {
			return true
		}
	}
	return false
}
