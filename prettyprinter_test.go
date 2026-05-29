package quamina

import (
	"testing"
)

func TestPP(t *testing.T) {
	pp := newPrettyPrinter(1)
	table, _ := makeShellStyleFA([]byte(`"x*9"`), pp)
	pp.labelTable(table, "START HERE")
	wanted := ` 884[START HERE] '22/"' → (914[on " at 0][s/t 240/312] 
 914[on " at 0] '78/x' → (384[*-Spinner][s/t 240/312] 
 384[*-Spinner] '39/9' → (322[spinEscape on 9 at 3] / ★ → 384[*-Spinner][s/t 240/312] 
 322[spinEscape on 9 at 3] ε → 384[*-Spinner] / '22/"' → (769[on " at 4][s/t 248/320] 
 769[on " at 4] 'f5/ℵ' → (301[last step at 5][s/t 240/312] 
 301[last step at 5]  [1 transition(s)][s/t 105/185] 
`
	s := pp.printNFA(table)
	if s != wanted {
		t.Errorf("LONG: wanted\n<%s>\ngot\n<%s>\n", wanted, s)
	}
	if pp.shortPrintNFA(table) != "884[START HERE]" {
		t.Errorf("SHORT: wanted <%s> got <%s>\n", "758[START HERE]", pp.shortPrintNFA(table))
	}
}

func TestNullPP(t *testing.T) {
	np := &nullPrinter{}
	table := newSmallTable()
	table.addByteStep(3, &faState{})
	np.labelTable(table, "foo")
	if np.printNFA(table) != noPP || np.shortPrintNFA(table) != noPP {
		t.Error("didn't get noPP")
	}
}
