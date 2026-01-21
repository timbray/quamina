package quamina

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

/*
// Too slow to run with every unit test. Shows that with the cachedFaShells cache, Quamina can build the
// ~p{L}+ machine 4K/second, as opposed to 135/second without the cache, a speedup factor of 30 or so
// regular RR: 4338.39/second with cache, 136.69 without, speedup 31.7
// skinny  RR: 3853.56/second with cache, 60.31 without, speedup 63.9
//
func TestRRCacheEffectiveness(t *testing.T) {
	words := readWWords(t)[:2000]
	re := "~p{L}+"
	pp := sharedNullPrinter
	var transitions []*fieldMatcher
	bufs := newNfaBuffers()

	before := time.Now()
	for _, w := range words {
		fa := faFromRegexp(t, re, pp)
		qm := []byte(`"` + string(w) + `"`)
		matches := traverseNFA(fa, qm, transitions, bufs, pp)
		if len(matches) != 1 {
			t.Errorf("missed <%s>", string(w))
		}
	}
	mid := time.Now()
	for _, w := range words {
		fa := faFromRegexp(t, re, pp)
		qm := []byte(`"` + string(w) + `"`)
		matches := traverseNFA(fa, qm, transitions, bufs, pp)
		if len(matches) != 1 {
			t.Errorf("missed <%s>", string(w))
		}
		delete(cachedFaShells, "L")
	}
	after := time.Now()
	elapsed1 := mid.Sub(before).Milliseconds()
	elapsed2 := after.Sub(mid).Milliseconds()
	perSecond1 := float64(len(words)) / (float64(elapsed1) / 1000.0)
	perSecond2 := float64(len(words)) / (float64(elapsed2) / 1000.0)
	fmt.Printf("\n%.2f/second with cache, %.2f without, speedup %.1f\n",
		perSecond1, perSecond2, perSecond1/perSecond2)
}
*/

func TestRegexpWorkbench(t *testing.T) {
	// previously on the workbench:
	// ~p{L}~p{Zs}~p{Nd}
	// ((ab){2})?
	// ([0-9]+(~.[0-9]+){3})
	pp := newPrettyPrinter(2355)
	matches := applyAndRunRegexp(t, "(ab){2,}", "ababab", pp)
	if matches != 1 {
		t.Error("Workbench")
	}
}
func applyAndRunRegexp(t *testing.T, regexp string, match string, pp printer) int {
	t.Helper()
	qm := []byte(`"` + match + `"`)
	fa := faFromRegexp(t, regexp, pp)
	// fmt.Println("FA:\n" + pp.printNFA(fa))
	var transitions []*fieldMatcher
	bufs := newNfaBuffers()
	matches := traverseNFA(fa, qm, transitions, bufs, pp)
	return len(matches)
}

func faFromRegexp(t *testing.T, r string, pp printer) *smallTable {
	t.Helper()
	parse, err := readRegexp(r)
	if err != nil {
		t.Error("bad regexp " + r)
	}
	if parse == nil {
		t.Error("nil parse")
		return nil
	}
	fa, _ := makeRegexpNFA(parse.tree, true, pp)
	return fa
}

func TestRegexpPlus(t *testing.T) {
	res := []string{
		"[123]",
		"[123]+",
		"[abc]+",
		"[123]+|[abc]+",
	}
	pp := newPrettyPrinter(4623)
	var fa *smallTable
	for _, re := range res {
		fa = faFromRegexp(t, re, pp)
	}
	goods := []string{
		`"123"`,
		`"abc"`,
	}
	bads := []string{
		"1a",
		"a1",
	}
	trans := []*fieldMatcher{}
	bufs := newNfaBuffers()
	for _, good := range goods {
		res := traverseNFA(fa, []byte(good), trans, bufs, pp)
		if len(res) != 1 {
			t.Errorf("missed good %s", good)
		}
	}
	for _, bad := range bads {
		res := traverseNFA(fa, []byte(bad), trans, bufs, pp)
		if len(res) != 0 {
			t.Error("matched bad " + bad)
		}
	}
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
	table := makeDotFA(targetState)
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
	table := makeDotFA(targetState)
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
func containsState(t *testing.T, states []*faState, wanted *faState) bool {
	t.Helper()
	for _, state := range states {
		if state == wanted {
			return true
		}
	}
	return false
}

func TestMakeByteDotFA(t *testing.T) {
	dest := &faState{}
	st := makeByteDotFA(dest, sharedNullPrinter)
	for i := 0; i < 256; i++ {
		b := byte(i)
		got := st.dStep(b)
		if forbiddenBytes[b] {
			if got != nil {
				t.Errorf("accepted %x", b)
			}
		} else {
			if got == nil {
				t.Errorf("rejected %x", b)
			}
		}
	}
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
		st, wanted := makeRegexpNFA(parsed.tree, false, sharedNullPrinter)
		bufs := newNfaBuffers()
		for _, r := range runes {
			// func traverseNFA(table *smallTable, val []byte, transitions []*fieldMatcher, bufs *bufpair) []*fieldMatcher {
			toMatch := strings.Replace(match, "X", string([]rune{r}), 1)
			found := traverseNFA(st, []byte(toMatch), nil, bufs, sharedNullPrinter)
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
		st, _ := makeRegexpNFA(parsed.tree, false, sharedNullPrinter)
		bufs := newNfaBuffers()
		for _, nonMatch := range nonMatches {
			found := traverseNFA(st, []byte(nonMatch), nil, bufs, sharedNullPrinter)
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
	bufs := newNfaBuffers()
	for _, pat := range daodechingpatterns {
		parsed, err := readRegexp(pat)
		if err != nil {
			t.Error("Parse failure: " + pat)
		}
		st, wanted := makeRegexpNFA(parsed.tree, false, sharedNullPrinter)
		found := traverseNFA(st, []byte(daodechingorig), nil, bufs, sharedNullPrinter)
		if len(found) != 1 {
			t.Errorf("Failed to match ")
		}
		if !containsFM(t, found, wanted) {
			t.Errorf("missed FM in matching %s to /%s", daodechingorig, pat)
		}
	}
}

func TestAddRuneTreeEntry(t *testing.T) {
	var root runeTreeNode = make([]*runeTreeEntry, byteCeiling)
	bbs := [][]rune{
		{'a', 'b', 'c'},
	}
	dest := &faState{}
	for _, runes := range bbs {
		for _, r := range runes {
			addRuneTreeEntry(root, r, dest)
		}
		fmt.Printf("RL: %d\n", len(root))
	}
}

func TestMultiLengthRR(t *testing.T) {
	range1 := RuneRange{
		{'a', 'd'},
		{0xf800, 0x10005},
	}
	ranges := []RuneRange{range1}

	for _, rr := range ranges {
		var multiLengthTest = rr

		//pp := newPrettyPrinter(2335)
		wantFM := &fieldMatcher{}

		dest := &faState{table: newSmallTable(), fieldTransitions: []*fieldMatcher{wantFM}}
		st := makeRuneRangeNFA(rr, dest, sharedNullPrinter)

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
		s.stEpsilon += len(st.epsilons)
		if len(st.epsilons) > s.stepMax {
			s.stepMax = len(st.epsilons)
		}
	}
	for _, step := range st.steps {
		if step != nil {
			nfaSizeStep(t, step.table, s, depth+1)
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

func TestZeroBasedRuneRange(t *testing.T) {
	tests := []regexpSample{
		{
			regex:     "xa?b?c?",
			matches:   []string{"xa", "xab", "xabc", "xb", "xbc", "xc"},
			nomatches: []string{"b", "Á"},
		},
		{regex: "ab?c", matches: []string{"ac", "abc"}, nomatches: []string{"bc", "Ác"}},
		{regex: "a?", matches: []string{"a", ""}, nomatches: []string{"b", "Á"}},
	}
	testRegexpMatches(t, tests)
}

func TestSimpleRegexpMerging(t *testing.T) {
	// I peeked into the machine for the RE below and it was horribly wrong
	re := "(a|b)c"
	parse, err := readRegexp(re)
	if err != nil {
		t.Error(err.Error())
	}
	fa, fm := makeRegexpNFA(parse.tree, false, sharedNullPrinter)
	tr := []*fieldMatcher{}
	out := traverseDFA(fa, []byte("ac"), tr)
	if len(out) != 1 || out[0] != fm {
		t.Error("MISS1")
	}
	tr = tr[:0]
	out = traverseDFA(fa, []byte("bc"), tr)
	if len(out) != 1 || out[0] != fm {
		t.Error("MISS2")
	}
	tr = tr[:0]
	out = traverseDFA(fa, []byte("a"), tr)
	if len(out) != 0 {
		t.Error("MISS3")
	}
}

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
