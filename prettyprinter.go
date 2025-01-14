package quamina

import (
	"fmt"
	"math/rand"
	"strings"
)

// printer is an interface used to generate representations of Quamina data structures to facilitate
// debugging and optimization. It's an interface rather than a type so that a null implementation can
// be provided for production that should incur very little performance cost.
type printer interface {
	labelTable(table *smallTable, label string)
	printNFA(table *smallTable) string
	shortPrintNFA(table *smallTable) string
}

// nullPrinter is what the name says, a do-nothing implementation of the printer interface which ideally
// should consume close to zero CPU cycles.
type nullPrinter struct{}

const noPP = "prettyprinting not enabled"

func (*nullPrinter) labelTable(_ *smallTable, _ string) {
}
func (*nullPrinter) printNFA(_ *smallTable) string {
	return noPP
}
func (*nullPrinter) shortPrintNFA(_ *smallTable) string {
	return noPP
}

var sharedNullPrinter = &nullPrinter{}

// prettyPrinter makes a human-readable representation of a NFA; each smallTable may be
// given a label and as a side effect will get a random 3-digit serial number. For an example
// of the output, see the functions TestPP and TestNullPP in prettyprinter_test.go
type prettyPrinter struct {
	randInts     rand.Source
	tableLabels  map[*smallTable]string
	tableSerials map[*smallTable]uint
}

func newPrettyPrinter(seed int) *prettyPrinter {
	return &prettyPrinter{
		randInts:     rand.NewSource(int64(seed)),
		tableLabels:  make(map[*smallTable]string),
		tableSerials: make(map[*smallTable]uint),
	}
}

func (pp *prettyPrinter) tableSerial(t *smallTable) uint {
	return pp.tableSerials[t]
}
func (pp *prettyPrinter) tableLabel(t *smallTable) string {
	return pp.tableLabels[t]
}

func (pp *prettyPrinter) labelTable(table *smallTable, label string) {
	pp.tableLabels[table] = label
	newSerial := pp.randInts.Int63()%500 + 500
	//nolint:gosec
	pp.tableSerials[table] = uint(newSerial)
}

func (pp *prettyPrinter) printNFA(t *smallTable) string {
	return pp.printNFAStep(&faState{table: t}, 0, make(map[*smallTable]bool))
}

func (pp *prettyPrinter) printNFAStep(fas *faState, indent int, already map[*smallTable]bool) string {
	t := fas.table
	trailer := "\n"
	if len(fas.fieldTransitions) != 0 {
		trailer = fmt.Sprintf(" [%d transition(s)]\n", len(fas.fieldTransitions))
	}
	s := " " + pp.printTable(t) + trailer
	for _, step := range t.steps {
		if step != nil {
			for _, state := range step.states {
				_, ok := already[state.table]
				if !ok {
					already[state.table] = true
					s += pp.printNFAStep(state, indent+1, already)
				}
			}
		}
	}
	return s
}

func (pp *prettyPrinter) printTable(t *smallTable) string {
	// going to build a string rep of a smallTable based on the unpacked form
	// each line is going to be a range like
	// 'c' .. 'e' => %X
	// lines where the *faNext is nil are omitted
	// TODO: Post-nfa-rationalization, I don't think the whole defTrans thing is necessary any more?
	var rows []string
	unpacked := unpackTable(t)

	var rangeStart int
	var b int

	defTrans := unpacked[0]

	// TODO: Try to generate an NFA with a state with multiple epsilons
	if len(t.epsilon) != 0 {
		fas := ""
		for i, eps := range t.epsilon {
			ep := &faNext{states: []*faState{eps}}
			if i != 0 {
				fas += ", "
			}
			fas += pp.nextString(ep)
		}
		rows = append(rows, "ε → "+fas)
	}
	for {
		for b < len(unpacked) && unpacked[b] == nil {
			b++
		}
		if b == len(unpacked) {
			break
		}
		rangeStart = b
		lastN := unpacked[b]
		for b < len(unpacked) && unpacked[b] == lastN {
			b++
		}
		if lastN != defTrans {
			row := ""
			if b == rangeStart+1 {
				row += fmt.Sprintf("'%s'", branchChar(byte(rangeStart)))
			} else {
				row += fmt.Sprintf("'%s'…'%s'", branchChar(byte(rangeStart)), branchChar(byte(b-1)))
			}
			row += " → " + pp.nextString(lastN)
			rows = append(rows, row)
		}
	}
	serial := pp.tableSerial(t)
	label := pp.tableLabel(t)
	if defTrans != nil {
		dtString := "★ → " + pp.nextString(defTrans)
		return fmt.Sprintf("%d[%s] ", serial, label) + strings.Join(rows, " / ") + " / " + dtString
	} else {
		return fmt.Sprintf("%d[%s] ", serial, label) + strings.Join(rows, " / ")
	}
}

func (pp *prettyPrinter) nextString(n *faNext) string {
	var snames []string
	for _, step := range n.states {
		snames = append(snames, fmt.Sprintf("%d[%s]",
			pp.tableSerial(step.table), pp.tableLabel(step.table)))
	}
	return strings.Join(snames, " · ")
}

func branchChar(b byte) string {
	replaceStr := []string{
		"nul", "soh", "stx", "etx", "eot", "enq", "ack", "bel", "bs", "ht", "nl", "vt", "np", "cr", "so", "si", "dle",
		"dc1", "dc2", "dc3", "dc4", "nak", "syn", "etb", "can", "em", "sub", "esc", "fs", "gs", "rs", "us", "sp",
		"!", "\"", "#", "$", "%", "&", "'", "(", ")", "*", "+", ",", "-", ".", "/",
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		":", ";", "<", "=", ">", "?", "@",
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R",
		"S", "T", "U", "V", "W", "X", "Y", "Z",
		"[", "\\", "]", "^", "_", "`",
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r",
		"s", "t", "u", "v", "w", "x", "y", "z",
		"{", "|", "}", "~", "del"}
	switch b {
	// TODO: Figure out how to test commented-out cases
	case valueTerminator:
		return fmt.Sprintf("%x/ℵ", valueTerminator)
	default:
		if b < 128 {
			return fmt.Sprintf("%x/%s", b, replaceStr[b])
		} else {
			return fmt.Sprintf("%x/", b)
		}
	}
}

func (pp *prettyPrinter) shortPrintNFA(table *smallTable) string {
	return fmt.Sprintf("%d[%s]", pp.tableSerials[table], pp.tableLabels[table])
}
