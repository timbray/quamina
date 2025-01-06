package quamina

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

var unicodeAssignables = RuneRange{
	{0x9, 0x9}, {0xA, 0xA}, {0xD, 0xD},
	{0x20, 0x7E},
	{0xA0, 0xD7FF},
	{0xE000, 0xFDCF},
	{0xFDF0, 0xFFFD},
	{0x10000, 0x1FFFD}, {0x20000, 0x2FFFD},
	{0x30000, 0x3FFFD}, {0x40000, 0x4FFFD},
	{0x50000, 0x5FFFD}, {0x60000, 0x6FFFD},
	{0x70000, 0x7FFFD}, {0x80000, 0x8FFFD},
	{0x90000, 0x9FFFD}, {0xA0000, 0xAFFFD},
	{0xB0000, 0xBFFFD}, {0xC0000, 0xCFFFD},
	{0xD0000, 0xDFFFD}, {0xE0000, 0xEFFFD},
	{0xF0000, 0xFFFFD}, {0x100000, 0x10FFFD},
}

var unicodeScalars = RuneRange{
	{0x0, 0xD800}, {0xE000, 0x10FFFF},
}

func TestExploreUTF8Form(t *testing.T) {
	bads := [][]byte{
		{0xc0, 0x80},             //0
		{0xc0, 0x8f},             //1
		{0xc1, 0x80},             //2
		{0xc1, 0x8f},             //3
		{0xe0, 0x9f, 0x80},       //4
		{0xe0, 0xc0, 0x80},       //5
		{0xe0, 0x9f, 0x80},       //6
		{0xed, 0xa0, 0x80},       //7
		{0xed, 0xb0, 0x80},       //8
		{0xed, 0xbf, 0x80},       //9
		{0xf0, 0x80, 0x80, 0x80}, //10
		{0xf0, 0x8f, 0x80, 0x80}, //11
		{0xf4, 0xa0, 0x80, 0x80}, //12
		{0xf4, 0xb0, 0x80, 0x80}, //13
		{0xf4, 0xbf, 0x80, 0x80}, //14
		{0x80},                   //15
		{0xfe},                   //16,
	}

	wantFM := &fieldMatcher{}
	targetState := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{wantFM}}
	table := makeDotFA(&faNext{states: []*faState{targetState}})
	var matchers []*fieldMatcher
	var got []*fieldMatcher
	for i, bad := range bads {
		got = traverseDFA(table, bad, matchers)
		if len(got) != 0 {
			t.Errorf("accepted index %d", i)
		}
	}
}

func TestDotSemantics(t *testing.T) {
	wantFM := &fieldMatcher{}
	targetState := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{wantFM}}
	table := makeDotFA(&faNext{states: []*faState{targetState}})
	var matchers []*fieldMatcher
	var got []*fieldMatcher
	var r rune
	for r = 0; r < unicode.MaxRune; r++ {
		// These actually would work because the string cast below would convert the char to �
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		got = traverseDFA(table, []byte(string([]rune{r})), matchers)
		if len(got) != 1 || got[0] != wantFM {
			t.Errorf("failed on %x", r)
		}
		matchers = matchers[:0]
	}

	// goodUTF are the UTF-8 sequences for 0, U+D7FF, U+E000, and U+10F0000, which should all pass.
	goodUTF8 := [][]byte{
		{0}, {0xED, 0x9F, 0xBF}, {0xE8, 0x80, 0x80}, {0xF4, 0x8F, 0x80, 0x80},
	}
	// badUTF are the UTF-8 sequences for surrogates U+D800, U+DAAA, and U+DFFF, which should not pass.
	// They are provided as literals because Go refuses to provide the UTF-8 for surrogates
	badUTF8 := [][]byte{
		{0xED, 0xA0, 0x80}, {0xED, 0xAA, 0xAA}, {0xED, 0xBF, 0xBF},
	}

	for _, good := range goodUTF8 {
		got = traverseDFA(table, good, matchers)
		if len(got) != 1 || got[0] != wantFM {
			t.Errorf("failed on non-surrogate %04x", r)
		}
		matchers = matchers[:0]
	}
	for _, bad := range badUTF8 {
		got = traverseDFA(table, bad, matchers)
		if len(got) != 0 {
			t.Errorf("accepted surrogate %04x", r)
		}
		matchers = matchers[:0]
	}
}

func containsFM(t *testing.T, fms []*fieldMatcher, wanted *fieldMatcher) bool {
	t.Helper()
	for _, fm := range fms {
		if fm == wanted {
			return true
		}
	}
	return false
}

func TestMakeDotRegexpNFA(t *testing.T) {
	runes := []rune{0x26, 0x416, 0x4e2d, 0x10346} // 1, 2, 3, & 4 bytes in UTF-8
	resAndMatches := map[string]string{
		"a.b": "aXb",
		".ab": "Xab",
		"ab.": "abX",
	}
	for re, match := range resAndMatches {
		parsed, err := readRegexp(re)
		if err != nil {
			t.Error("Parse " + err.Error())
		}
		st, wanted := makeRegexpNFA(parsed.tree)
		bufs := &bufpair{}
		for _, r := range runes {
			// func traverseNFA(table *smallTable, val []byte, transitions []*fieldMatcher, bufs *bufpair) []*fieldMatcher {
			toMatch := strings.Replace(match, "X", string([]rune{r}), 1)
			found := traverseNFA(st, []byte(toMatch), nil, bufs)
			if len(found) == 0 {
				t.Errorf("struck out matching %s to /%s/", match, re)
			}
			if !containsFM(t, found, wanted) {
				t.Errorf("Wrong FM returned matching %s to /%s/", match, re)
			}
		}
	}
	resAndNonMatches := map[string][]string{
		"a.b": {"ab", "axyb"},
		".ab": {"ab", "zzab"},
		"ab.": {"ab", "abab"},
	}
	for re, nonMatches := range resAndNonMatches {
		parsed, err := readRegexp(re)
		if err != nil {
			t.Error("Parse " + err.Error())
		}
		st, _ := makeRegexpNFA(parsed.tree)
		bufs := &bufpair{}
		for _, nonMatch := range nonMatches {
			found := traverseNFA(st, []byte(nonMatch), nil, bufs)
			if len(found) != 0 {
				t.Errorf("false match to %s to /%s/", nonMatch, re)
			}
		}
	}

	daodechingorig := "道可道，非常道。名可名"
	daodechingpatterns := []string{
		"道可道.非常道.名可名",
		"道..，非..。名.名",
		".可道，非常道。名..",
		"....非常道。名可名",
		"道可道，非常...可名",
	}
	bufs := &bufpair{}
	for _, pat := range daodechingpatterns {
		parsed, err := readRegexp(pat)
		if err != nil {
			t.Error("Parse failure: " + pat)
		}
		st, wanted := makeRegexpNFA(parsed.tree)
		found := traverseNFA(st, []byte(daodechingorig), nil, bufs)
		if len(found) != 1 {
			t.Errorf("Failed to match ")
		}
		if !containsFM(t, found, wanted) {
			t.Errorf("missed FM in matching %s to /%s", daodechingorig, pat)
		}
	}
}

func TestMultiLengthRR(t *testing.T) {
	ranges := []RuneRange{unicodeScalars, unicodeAssignables}
	rangeNames := []string{"Scalars", "assignables"}

	// get UTF-8 versions of all the code points
	for index, rr := range ranges {
		fmt.Println(" " + rangeNames[index])
		var multiLengthTest = rr

		pp := newPrettyPrinter(2335)
		wantFM := &fieldMatcher{}
		next := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{wantFM}}
		st, err := makeRuneRangeNFA(rr, next, pp)
		if err != nil {
			t.Error(err.Error())
		}

		matchers := []*fieldMatcher{}
		var got []*fieldMatcher
		for _, rp := range multiLengthTest {
			got = traverseDFA(st, []byte(string([]rune{rp.Lo})), matchers)
			if len(got) != 1 || got[0] != wantFM {
				t.Errorf("failed on %x", rp.Lo)
			}
		}
		nfaSize(t, st)
	}
}

func nfaSize(t *testing.T, st *smallTable) {
	t.Helper()
	s := &statsAccum{}
	nfaSizeStep(t, st, s, 0)
	fmt.Printf("Tables: %d\n", s.stCount)
	fmt.Printf("Avg size: %d\n", int(float64(s.stEntries)/float64(s.stCount)))
	fmt.Printf("Max size: %d\n", s.stMax)
	fmt.Printf("Max depth %d\n", s.stDepth)
}
func nfaSizeStep(t *testing.T, st *smallTable, s *statsAccum, depth int) {
	t.Helper()
	if depth > s.stDepth {
		s.stDepth = depth
	}
	s.stCount++
	tSize := len(st.ceilings)
	if tSize > 1 {
		if tSize > s.stMax {
			s.stMax = tSize
		}
		s.stTblCount++
		s.stEntries += len(st.ceilings)
		s.stEpsilon += len(st.epsilon)
		if len(st.epsilon) > s.stEpMax {
			s.stEpMax = len(st.epsilon)
		}
	}
	for _, next := range st.steps {
		if next != nil {
			for _, step := range next.states {
				nfaSizeStep(t, step.table, s, depth+1)
			}
		}
	}
}

/* useful for debugging
func showUTF8(t *testing.T, lo rune, hi rune) {
	t.Helper()
	for r := lo; r < hi; r++ {
		rl := utf8.RuneLen(r)
		if rl == -1 {
			fmt.Printf("invalid rune value %x", r)
			continue
		}
		buf := make([]byte, rl)
		_ = utf8.EncodeRune(buf, r)
		fmt.Printf("%x/%c: %d:", r, r, len(buf))
		for _, b := range buf {
			fmt.Printf(" %x,", b)
		}
		fmt.Println()
	}
}
*/

func TestRRiterator(t *testing.T) {
	rr := RuneRange{
		{'a', 'c'},
		{'f', 'f'},
		{'g', 'i'},
	}

	wanteds := []rune{'a', 'b', 'c', 'f', 'g', 'h', 'i'}
	i, err := newRuneRangeIterator(rr)
	if err != nil {
		t.Error(err.Error())
	}
	for index, wanted := range wanteds {
		r := i.next()
		if r != wanted {
			t.Errorf("mismatch at %d, %c != %c", index, r, wanted)
		}
	}
}

func TestBasicRRNFABuilding(t *testing.T) {
	rr := RuneRange{{'a', 'c'}}
	pp := newPrettyPrinter(2335)
	next := &faState{table: newSmallTable()}
	st, err := makeRuneRangeNFA(rr, next, pp)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println("ST: " + pp.printNFA(st))
}
