package quamina

import (
	"testing"
)

func TestPP(t *testing.T) {
	pp := newPrettyPrinter(1)
	table, _ := makeShellStyleAutomaton([]byte(`"x*9"`), pp)
	pp.labelTable(table, "START HERE")
	wanted := ` 758 [START HERE] '"' → [910 on " at 0]
 910 [on " at 0] 'x' → [821 gS at 2]
 821 [gS at 2] '9' → [551 gX on 9 at 3] / ★ → [821 gS at 2]
 551 [gX on 9 at 3] '"' → [937 on " at 4] / '9' → [551 gX on 9 at 3] / ★ → [821 gS at 2]
 937 [on " at 4] '9' → [551 gX on 9 at 3] / 'ℵ' → [820 last step at 5] / ★ → [821 gS at 2]
 820 [last step at 5]  [1 transition(s)]
`
	s := pp.printNFA(table)
	if s != wanted {
		t.Errorf("LONG: wanted\n<%s>\ngot\n<%s>\n", wanted, s)
	}
	if pp.shortPrintNFA(table) != "758-START HERE" {
		t.Errorf("SHORT: wanted <%s> got <%s>\n", "758-START HERE", pp.shortPrintNFA(table))
	}
}

func TestNullPP(t *testing.T) {
	np := &nullPrinter{}
	table := newSmallTable()
	table.addByteStep(3, &faNext{})
	np.labelTable(table, "foo")
	if np.printNFA(table) != noPP || np.shortPrintNFA(table) != noPP {
		t.Error("didn't get noPP")
	}
}
