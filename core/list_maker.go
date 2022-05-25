package core

// this needs to exist so that all all the lists containing a single step to X, or the triple step to X,Y,Z are the
//  same list, so that pack/unpack work properly

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
