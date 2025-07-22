package quamina

import (
	"sort"
	"strings"
	"testing"
)

// NormalChar = ( %x00-27 / "," / "-" / %x2F-3E ; '/'-'>'
// / %x40-5A ; '@'-'Z'
// / %x5E-7A ; '^'-'z'
// / %x7E-D7FF ; skip surrogate code points
// / %xE000-10FFFF )
func TestIsNormalChar(t *testing.T) {
	normals := []rune{
		0, 1, 0x26, 0x27,
		0x40, 0x41, 0x59, 0x5a, 0x5c,
		0x5e, 0x5f, 0x79, 0x7a,
		0x7f, 0xd7fe, 0xd7ff,
		0xe000, 0xe001, 0x10fffe, 0x10ffff,
	}
	for _, normal := range normals {
		if !isNormalChar(normal) {
			t.Errorf("%x abnormal", normal)
		}
	}
	abormals := []rune{
		0x28, 0x2e, 0x3f,
		0x3f, 0x5b,
		0x5d, 0x7b,
		0x7d, 0x7e, 0xd800,
		0xdfff,
	}
	for _, abnormal := range abormals {
		if isNormalChar(abnormal) {
			t.Errorf("%x normal", abnormal)
		}
	}
}

func TestSingleCharEscape(t *testing.T) {
	// SingleCharEsc = "\" ( %x28-2B ; '('-'+'
	// / "-" / "." / "?" / %x5B-5E ; '['-'^'
	// / %s"n" / %s"r" / %s"t" / %x7B-7D ; '{'-'}'
	//)
	sces := []rune{
		0x28, 0x29, 0x2a, 0x2b,
		'-', '.', '?', 0x5B, 0x5C, 0x5D, 0x5E,
		'n', 'r', 't', 0x7B, 0x7C, 0x7D,
		'~',
	}
	for _, sce := range sces {
		_, ok := checkSingleCharEscape(sce)
		if !ok {
			t.Errorf("%x not sce", sce)
		}
	}
	notSces := []rune{
		0x27, 0x2C, 0x5A, 0x5F, 'j', 0x7A, 0x7F,
	}
	for _, notSce := range notSces {
		_, ok := checkSingleCharEscape(notSce)
		if ok {
			t.Errorf("%x is sce", notSce)
		}
	}
}

func TestReadCCE1(t *testing.T) {
	goods := []string{
		"a", `~n-~r`, "ab", "a-b",
	}
	bads := []string{
		"a-~P{Lu}", "~P{Lu}-x",
	}
	for _, good := range goods {
		_, err := readRegexp("[" + good + "]")
		if err != nil {
			t.Errorf("Missed good /[%s]/: %s", good, err.Error())
		}
	}
	for _, bad := range bads {
		_, err := readRegexp("[" + bad + "]")
		if err == nil {
			t.Errorf("Missed bad %s", bad)
		}
	}
}

func TestRuneRangesFromCCE1(t *testing.T) {
	cce1s := []string{
		"[ax]", "[a]", "[abc]",
		"[c-g]", "[ah-mq]",
		"[~n-~r]",
		"[-bdg-h]",
	}
	wanted := []RuneRange{
		{{'a', 'a'}, {'x', 'x'}}, {{'a', 'a'}}, {{'a', 'c'}},
		{{'c', 'g'}}, {{'a', 'a'}, {'h', 'm'}, {'q', 'q'}},
		{{10, 13}},
		{{'-', '-'}, {'b', 'b'}, {'d', 'd'}, {'g', 'h'}},
	}
	for i, cce1 := range cce1s {
		parse := newRxParseState([]byte(cce1[1:]))
		rr, err := readCCE1s(parse)
		if err != nil {
			t.Error("RC: " + err.Error())
		}
		if !runeRangeEqual(t, wanted[i], rr) {
			t.Errorf("Failed on %s", cce1)
		}
	}
}

func TestSimplifyRR(t *testing.T) {
	in := []RuneRange{
		{{'a', 'b'}, {'e', 'j'}, {'l', 'n'}},
		{{'a', 'e'}, {'b', 'e'}, {'d', 'm'}},
		{{'a', 'c'}, {'d', 'r'}, {'s', 'x'}},
	}
	wanteds := []RuneRange{
		{{'a', 'b'}, {'e', 'j'}, {'l', 'n'}},
		{{'a', 'm'}},
		{{'a', 'x'}},
	}
	for i, rrin := range in {
		wanted := wanteds[i]
		out := simplifyRuneRange(rrin)
		if !runeRangeEqual(t, out, wanted) {
			t.Errorf("botch at %d", i)
		}
	}
}

func runeRangeEqual(t *testing.T, wanted RuneRange, got RuneRange) bool {
	t.Helper()
	if len(wanted) != len(got) {
		return false
	}
	sort.Slice(wanted, func(i, j int) bool { return wanted[i].Lo < wanted[j].Lo })
	sort.Slice(got, func(i, j int) bool { return got[i].Lo < got[j].Lo })
	for i, w := range wanted {
		g := got[i]
		if w.Lo != g.Lo || w.Hi != g.Hi {
			return false
		}
	}
	return true
}

func TestBasicRegexpFeatureRead(t *testing.T) {
	type fw struct {
		rx     string
		wanted []regexpFeature
	}

	var tfw = []fw{
		{rx: "a.b", wanted: []regexpFeature{rxfDot}},
		{rx: "ab*", wanted: []regexpFeature{rxfStar}},
		{rx: "a+b", wanted: []regexpFeature{rxfPlus}},
		{rx: "(ab)+", wanted: []regexpFeature{rxfParenGroup, rxfPlus}},
		{rx: "zz?zz", wanted: []regexpFeature{rxfQM}},
		{rx: "zzzz{3}", wanted: []regexpFeature{rxfRange}},
		{rx: "zzzz{0,3}", wanted: []regexpFeature{rxfRange}},
		{rx: "zzzz{3,}", wanted: []regexpFeature{rxfRange}},
		{rx: "a~p{Lt}", wanted: []regexpFeature{rxfProperty}},
		{rx: "a~P{Me}", wanted: []regexpFeature{rxfProperty}},
		{rx: "a[fox37é]z", wanted: []regexpFeature{rxfClass}},
		{rx: "a[-fox37é-]z", wanted: []regexpFeature{rxfClass}},
		{rx: "a[fox33-87é]z", wanted: []regexpFeature{rxfClass}},
		{rx: "a[^fox37é]z", wanted: []regexpFeature{rxfClass, rxfNegatedClass}},
		{rx: "(abc)|(def)", wanted: []regexpFeature{rxfOrBar, rxfParenGroup}},
	}

	var parse *regexpParse
	var err error
	for _, w := range tfw {
		parse, err = readRegexp(w.rx)
		if err != nil {
			t.Errorf("botch on %s: %s", w.rx, err.Error())
		}
		if len(w.wanted) != len(parse.features.found) {
			t.Errorf("for %s got %d wanted %d", w.rx, len(parse.features.found), len(w.wanted))
		} else {
			for _, f := range w.wanted {
				_, ok := parse.features.found[f]
				if !ok {
					t.Errorf("for %s missed feature %s", w.rx, f)
				}
			}
		}
	}
	parse, _ = readRegexp("a*b")
	unimpl := parse.features.foundUnimplemented()
	foundStar := false
	for _, u := range unimpl {
		if u == rxfStar {
			foundStar = true
		}
	}
	if !foundStar {
		t.Error("Didn't find Star")
	}
}

func TestRegexpErrors(t *testing.T) {
	bads := []string{
		"~P{L",
		"~P{L*}",
		string([]byte{'~', 0xfe, 0xff}),
		string([]byte{'[', 'a', 'b', 0xfe, 0xff, ']'}),
		string([]byte{'[', 'a', '-', 0xff, ']'}),
		string([]byte{'[', 'a', '-', '~', 0xff, ']'}),
		string([]byte{'a', 0xff}),
		string([]byte{'a', '{', 0xff, '}'}),
		string([]byte{'a', '{', '2', 0xff, '}'}),
		"a{9999999999998,9999999999999}",
		"a{2x-3}",
		"a{2,",
		string([]byte{'a', '{', '2', 0xff}),
		"a{2,r}",
		string([]byte{'a', '{', '2', ',', 0xff}),
		"a{2,4",
		string([]byte{'a', '{', '2', ',', '4', 0xff}),
		"a{2,4x",
		"a{2,9999999999999}",
		"abc)",
	}
	for _, bad := range bads {
		_, err := readRegexp(bad)
		if err == nil {
			t.Error("Took " + bad)
		}
	}
}

func TestAddRegexpTransition(t *testing.T) {
	// TODO: Keep adding/subtracting from this as we add features
	goods := []string{
		"a.",
	}
	bads := []string{
		"a*", "a+",
		"a{1,3}", "~p{Lu}", "[^abc]",
	}
	template := `{"a":[{"regexp": "FOO"}]}`
	cm := newCoreMatcher()
	for _, good := range goods {
		pat := strings.Replace(template, "FOO", good, 10)
		err := cm.addPattern("foo", pat)
		if err != nil {
			t.Errorf("thinks it found unimplemented feature in /%s/", good)
		}
	}
	for _, bad := range bads {
		pat := strings.Replace(template, "FOO", bad, 10)
		err := cm.addPattern("foo", pat)
		if err == nil {
			t.Errorf("missed unimplemented feature in /%s/", bad)
		}
	}
}

func TestRegexpReader(t *testing.T) {
	pat := `{"a":[{"regexp": "a.b"}]}`
	cm := newCoreMatcher()
	err := cm.addPattern("x", pat)
	if err != nil {
		t.Error("ap: " + err.Error())
	}
}
