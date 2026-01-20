package quamina

import (
	"fmt"
)

// In the regular expressions represented by the I-Regexp syntax, the | connector has the lowest
// precedence, so at the top level, it's a slice of what the ABNF calls branches - generate an NFA
// for each branch and then take their union.
// Inside a branch, the structure is obviously recursive because of the ()-group, which itself can
// carry a slice of branches etc.  Aside from that, the branch contains a sequence of atom/quantifier
// pairs.  All the "atom" syntax options describe ranges of characters and are well-represented by
// the RuneRange type. This includes [] and \[pP]{whatever}.
// All the forms of quantifiers can be described by pairs of numbers. ? is [0,1]. + is [1,♾️]. * is [0,♾️].
// {m,n} ranges also, obviously.

type regexpBranch []*quantifiedAtom
type regexpRoot []regexpBranch

// makeRegexpNFA traverses the parsed regexp tree and generates a finite automaton
// that matches it. If forField is true, then the FA will have states that match " at the beginning
// and end.
func makeRegexpNFA(root regexpRoot, forField bool, pp printer) (*smallTable, *fieldMatcher) {
	nextField := newFieldMatcher()
	nextStep := makeNFATrailer(nextField)
	pp.labelTable(nextStep.table, "Trailer")
	if forField {
		table := makeSmallTable(nil, []byte{'"'}, []*faState{nextStep})
		pp.labelTable(table, "</Field>")
		nextStep = &faState{table: table}
	}
	return makeNFAFromBranches(root, nextStep, forField, pp), nextField
}
func makeNFAFromBranches(root regexpRoot, nextStep *faState, forField bool, pp printer) *smallTable {
	// completely empty regexp
	if len(root) == 0 {
		return makeSmallTable(nil, []byte{'"'}, []*faState{nextStep})
	}
	var fa *smallTable
	for _, branch := range root {
		var nextBranch *smallTable
		if len(branch) == 0 {
			nextBranch = makeSmallTable(nil, []byte{'"'}, []*faState{nextStep})
			pp.labelTable(nextBranch, "next on len 0")
		} else {
			nextBranch = faFromBranch(branch, nextStep, forField, pp)
		}
		if fa != nil {
			fa = mergeFAs(fa, nextBranch, pp)
		} else {
			fa = nextBranch
		}
	}
	return fa
}

func faFromBranch(branch regexpBranch, nextStep *faState, forField bool, pp printer) *smallTable {
	state := faFromQuantifiedAtom(branch, 0, nextStep, pp)
	table := state.table
	if forField {
		firstState := &faState{table: table}
		table = makeSmallTable(nil, []byte{'"'}, []*faState{firstState})
		pp.labelTable(table, "<Field>")
	}
	return table
}

// faFromQuantifiedAtom builds regular expression NFAs per the Thompson process
func faFromQuantifiedAtom(branch regexpBranch, index int, finalStep *faState, pp printer) *faState {
	atom := branch[index]
	var nextState *faState
	if index == len(branch)-1 {
		nextState = finalStep
	} else {
		nextState = faFromQuantifiedAtom(branch, index+1, finalStep, pp)
	}
	var state *faState

	switch {
	case atom.isPlus():
		// the + construction requires a loopback state in front of the state table
		plusLoopback := &faState{table: newSmallTable()}
		pp.labelTable(plusLoopback.table, "PlusLoopback")
		state = &faState{table: atom.makeFA(plusLoopback, pp)}

		// for the + case, need to loop back to the newly created state
		plusLoopback.table.epsilons = []*faState{nextState, state}

	case atom.isStar():
		// the * construction requires that the generated FA points back to itself
		state = &faState{}
		state.table = atom.makeFA(state, pp)

		// the * construct requires an epsilon transition forward, possibly
		// passing over a multi-step sequence
		state.table.epsilons = append(state.table.epsilons, nextState)

	case atom.hasMinMax():
		shellTable := atom.makeFA(PlaceholderState, pp)
		nextMinMaxStep := nextState

		for counter := atom.quantMax; counter > 0; counter-- {
			stepTable := faFromShell(shellTable, PlaceholderState, nextMinMaxStep)
			pp.labelTable(stepTable, fmt.Sprintf("minmax at %d", counter))

			// if it's between quantMin & max, we're in optional territory
			// so it needs an epsilon to allow jumping out
			if counter > atom.quantMin {
				stepTable.epsilons = append(stepTable.epsilons, nextState)
			}
			state = &faState{table: stepTable}
			nextMinMaxStep = state
		}

	case atom.isQM():
		// for the ? case, forward epsilon
		state = &faState{table: atom.makeFA(nextState, pp)}
		state.table.epsilons = append(state.table.epsilons, nextState)

	case atom.isNoOp():
		// when we see a{0}, which the grammar allows
		state = &faState{table: atom.makeFA(nextState, pp)}
		state.table.epsilons = []*faState{nextState}

	case atom.isMinimumOnly():
		shellTable := atom.makeFA(PlaceholderState, pp)
		nextMinMaxStep := nextState

		var lastState *faState
		for counter := atom.quantMin; counter > 0; counter-- {
			stepTable := faFromShell(shellTable, PlaceholderState, nextMinMaxStep)
			pp.labelTable(stepTable, fmt.Sprintf("minmax at %d", counter))
			state = &faState{table: stepTable}

			// there's a chain of the minimum-count steps, but the last one has to
			// loop back to the first one, so we have to remember the last one
			if counter == atom.quantMin {
				lastState = state
			}

			nextMinMaxStep = state
		}
		lastState.table.epsilons = append(lastState.table.epsilons, state)

	default:
		state = &faState{table: atom.makeFA(nextState, pp)}
	}

	return state
}

// makeNFATrailer generates the last two steps in every NFA, because all field values end with the
// valueTerminator marker, so you need the field-matched state and you need another state that branches
// to it based on valueTerminator
func makeNFATrailer(nextField *fieldMatcher) *faState {
	matchState := &faState{
		table:            newSmallTable(),
		fieldTransitions: []*fieldMatcher{nextField},
	}
	table := makeSmallTable(nil, []byte{valueTerminator}, []*faState{matchState})
	return &faState{table: table}
}

func makeByteDotFA(dest *faState, pp printer) *smallTable {
	ceilings := []byte{0xC0, 0xC2, 0xF5, 0xF6}
	steps := []*faState{dest, nil, dest, nil}
	t := &smallTable{ceilings: ceilings, steps: steps}
	pp.labelTable(t, " · ")
	return t
}

func makeDotFA(dest *faState) *smallTable {
	sLast := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faState{nil, dest, nil},
	}
	targetLast := &faState{table: sLast}
	sLastInter := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLast, nil},
	}
	targetLastInter := &faState{table: sLastInter}
	sFirstInter := &smallTable{
		ceilings: []byte{0x80, 0xc0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLastInter, nil},
	}
	targetFirstInter := &faState{table: sFirstInter}

	sE0 := &smallTable{
		ceilings: []byte{0xa0, 0xc0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLast, nil},
	}
	targetE0 := &faState{table: sE0}

	sED := &smallTable{
		ceilings: []byte{0x80, 0xA0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLast, nil},
	}
	targetED := &faState{table: sED}

	sF0 := &smallTable{
		ceilings: []byte{0x90, 0xC0, byte(byteCeiling)},
		steps:    []*faState{nil, targetLastInter, nil},
	}
	targetF0 := &faState{table: sF0}

	sF4 := &smallTable{
		ceilings: []byte{0x80, 0x90, byte(byteCeiling)},
		steps:    []*faState{nil, targetLastInter, nil},
	}
	targetF4 := &faState{table: sF4}

	// for reference, see https://www.tbray.org/ongoing/When/202x/2024/12/29/Matching-Dot-Redux
	return &smallTable{
		ceilings: []byte{
			0x80,              // 0
			0xC2,              // 1
			0xE0,              // 2
			0xE1,              // 3
			0xED,              // 4
			0xEE,              // 5
			0xF0,              // 6
			0xF1,              // 7
			0xF4,              // 8
			0xF5,              // 9
			byte(byteCeiling), // 10
		},
		steps: []*faState{
			dest,             // 0
			nil,              // 1
			targetLast,       // 2
			targetE0,         // 3
			targetLastInter,  // 4
			targetED,         // 5
			targetLastInter,  // 6
			targetF0,         // 7
			targetFirstInter, // 8
			targetF4,         // 9
			nil,              // 10
		},
	}
}
