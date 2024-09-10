package quamina

import (
	"fmt"
	"testing"
)

func TestWildcardSyntax(t *testing.T) {
	cm := newCoreMatcher()
	busted1 := `{"x": [{"wildcard": . }]}`
	err := cm.addPattern("x", busted1)
	if err == nil {
		t.Error("accepted " + busted1)
	}
	busted2 := `{"x": [{"wildcard": 3}]}`
	err = cm.addPattern("x", busted2)
	if err == nil {
		t.Error("accepted " + busted2)
	}
	busted3 := `{"x": [{"wildcard": "x" ]}`
	err = cm.addPattern("x", busted3)
	if err == nil {
		t.Error("accepted " + busted3)
	}
}

// these tests copied with thanks from aws/event-ruler - didn't grab them all, just too many for my poor fingers
func TestWildcardMatching(t *testing.T) {
	exercisePattern(t, "*", []string{"", "*", "h", "hello"}, []string{})
	exercisePattern(t, "*hello", []string{"hello", "hhello", "xxxhello", "*hello", "23Őzhello"}, []string{"", "ello", "hellx", "xhellx", "hell5rHGGHo"})
	exercisePattern(t, "h*llo", []string{"hllo", "hello", "hxxxllo", "hel23Őzlllo"}, []string{"", "hlo", "hll", "hellol", "hel5rHGGHlo"})
	exercisePattern(t, "hel*o", []string{"helo", "hello", "helxxxo", "hel23Őzlllo"}, []string{"", "hell", "helox", "hellox", "hel5rHGGHe"})
	exercisePattern(t, "hello*", []string{"hello", "hellox", "hellooo", "hello*", "hello23Őzlllo"}, []string{"", "hell", "hellx", "hellxo", "hol5rHGGHo"})
	exercisePattern(t, "h*l*o", []string{"hlo", "helo", "hllo", "hloo", "hello", "hxxxlxxxo", "h*l*o", "hel*o", "h*llo", "hel23Őzlllo"}, []string{"", "ho", "heeo", "helx", "llo", "hex5rHGGHo"})
	exercisePattern(t, "he*l*", []string{"hel", "hexl", "helx", "helxx", "helxl", "helxlx", "helxxl", "helxxlxx", "helxxlxxl"}, []string{"", "he", "hex", "hexxx"})
	exercisePattern(t, "*l*", []string{"l", "xl", "lx", "xlx", "xxl", "lxx", "xxlxx", "xlxlxlxlxl", "lxlxlxlxlx"}, []string{"", "x", "xx", "xtx"})
	exercisePattern(t, `hel\\*o`, []string{"hel*o"}, []string{"helo", "hello"})
	exercisePattern(t, `he\\**o`, []string{"he*o", "he*llo", "he*hello"}, []string{"heo", "helo", "hello", "he*l"})
	exercisePattern(t, `he\\\\llo`, []string{"he\\\\llo"}, []string{"hello", "he\\llo"})
	exercisePattern(t, `he\\\\\\*llo`, []string{`he\\*llo`}, []string{`hello`, `he\\\\llo`, `he\\llo`, `he\\xxllo`})
	exercisePattern(t, `he\\\\*llo`, []string{`he\\llo`, `he\\*llo`, `he\\\\llo`, `he\\\\\\llo`, `he\\xxllo`}, []string{`hello`, `he\\ll`})
	exerciseMultiPatterns(t, nil, []pwanted{
		{`{"x":[{"wildcard": "*"}]}`, []string{"", "*", "h", "ho", "hello"}},
		{`{"x":[{"wildcard": "h*o"}]}`, []string{"ho", "hello"}},
		{`{"x":["hello"]}`, []string{"hello"}}})
	exerciseMultiPatterns(t, []string{"", "hellox", "blahabc"}, []pwanted{
		{`{"x":[{"wildcard": "*hello"}]}`, []string{"hello", "xhello", "hehello"}},
		{`{"x":["abc"]}`, []string{"abc"}}})
	exerciseMultiPatterns(t, []string{"", "h", "ello", "hel", "hlo", "hell"}, []pwanted{
		{`{"x":[{"wildcard": "*hello"}]}`, []string{"hello", "xhello", "hehello"}},
		{`{"x":[{"wildcard": "h*llo"}]}`, []string{"hllo", "hello", "hehello"}}})
	exerciseMultiPatterns(t, []string{"", "h", "ello", "hel", "heo", "hell"}, []pwanted{
		{`{"x":[{"wildcard": "*hello"}]}`, []string{"hello", "xhello", "hehello"}},
		{`{"x":[{"wildcard": "he*lo"}]}`, []string{"helo", "hello", "hehello"}}})
	exerciseMultiPatterns(t, []string{"", "e", "l", "lo", "hel"}, []pwanted{
		{`{"x":[{"wildcard": "*elo"}]}`, []string{"elo", "helo", "xhelo"}},
		{`{"x":[{"wildcard": "e*l*"}]}`, []string{"el", "elo", "exl", "elx", "exlx", "exxl", "elxx", "exxlxx"}}})
	exerciseMultiPatterns(t, []string{"", "he", "hexxo", "ello"}, []pwanted{
		{`{"x":[{"wildcard": "*hello"}]}`, []string{"hello", "xhello", "xxhello"}},
		{`{"x":[{"wildcard": "he*l*"}]}`, []string{"hel", "hello", "helo", "hexl", "hexlx", "hexxl", "helxx", "hexxlxx"}}})
	exerciseMultiPatterns(t, []string{"", "hlo", "heo", "hllol", "helol"}, []pwanted{
		{`{"x":[{"wildcard": "h*llo"}]}`, []string{"hllo", "hello", "hxxxllo", "hexxxllo"}},
		{`{"x":[{"wildcard": "he*lo"}]}`, []string{"helo", "hello", "hexxxlo", "hexxxllo"}}})
	exerciseMultiPatterns(t, []string{"", "hlox", "hllo", "helo", "heox", "helx", "hellx", "helloxx", "heloxx"}, []pwanted{
		{`{"x":[{"wildcard": "h*llox"}]}`, []string{"hllox", "hellox", "hxxxllox", "helhllox", "hheloxllox"}},
		{`{"x":[{"wildcard": "hel*ox"}]}`, []string{"helox", "hellox", "helxxxox", "helhllox", "helhlloxox"}}})
	exerciseMultiPatterns(t, []string{"", "h", "he", "hl", "el", "hlo", "llo", "hllol", "hxll", "hexxx"}, []pwanted{
		{`{"x":[{"wildcard": "h*llo"}]}`, []string{"hllo", "hello", "hxxxllo", "hexxxllo", "hexxxlllo"}},
		{`{"x":[{"wildcard": "he*l*"}]}`, []string{"hel", "helo", "hexl", "hello", "helol", "hexxxlo", "hexxxllo", "hexxxlllo"}}})
	exerciseMultiPatterns(t, []string{"", "h", "hex", "hl", "exl", "hxlo", "xllo", "hxllol", "hxxll", "hexxx"}, []pwanted{
		{`{"x":[{"wildcard": "h*xllo"}]}`, []string{"hxllo", "hexllo", "hxxxllo", "hexxxllo"}},
		{`{"x":[{"wildcard": "hex*l*"}]}`, []string{"hexl", "hexlo", "hexxl", "hexllo", "hexlol", "hexxxlo", "hexxxllo", "hexxxlllo"}}})
	exerciseMultiPatterns(t, []string{"", "hel", "heo", "hlo", "hellxox"}, []pwanted{
		{`{"x":[{"wildcard": "he*lo"}]}`, []string{"helo", "hello", "hexxxlo", "helxxxlo"}},
		{`{"x":[{"wildcard": "hel*o"}]}`, []string{"helo", "hello", "hellxo", "helxxxo", "helxxxlo"}}})
	exerciseMultiPatterns(t, []string{"", "hlo", "hll", "hel", "helox"}, []pwanted{
		{`{"x":[{"wildcard": "h*llo"}]}`, []string{"hllo", "hello", "hxxxllo", "helllo"}},
		{`{"x":[{"wildcard": "hel*o"}]}`, []string{"helo", "hello", "helxo", "helllo"}}})
	exerciseMultiPatterns(t, []string{"", "he", "hel", "helox", "helx", "hxlo"}, []pwanted{
		{`{"x":[{"wildcard": "he*lo"}]}`, []string{"helo", "hello", "helllo", "helxlo"}},
		{`{"x":[{"wildcard": "hell*"}]}`, []string{"hell", "hello", "helllo", "hellx", "hellxxx"}}})
	exerciseMultiPatterns(t, []string{"", "hel", "helox", "helxox", "hexo"}, []pwanted{
		{`{"x":[{"wildcard": "hel*o"}]}`, []string{"helo", "hello", "helllo", "hellloo", "helloo", "heloo"}},
		{`{"x":[{"wildcard": "hell*"}]}`, []string{"hell", "hello", "helllo", "hellloo", "helloo", "hellox"}}})
	exerciseMultiPatterns(t, []string{"", "he", "hex", "hexlo"}, []pwanted{
		{`{"x":[{"wildcard": "hel*"}]}`, []string{"hel", "helx", "hello", "hellox"}},
		{`{"x":[{"wildcard": "hello*"}]}`, []string{"hello", "hellox"}}})
	exerciseMultiPatterns(t, []string{"", "he", "hex", "hexlo"}, []pwanted{
		{`{"x":[{"wildcard": "*hello"}]}`, []string{"hello", "hhello", "hhhello"}},
		{`{"x":["hello"]}`, []string{"hello"}}})
	exerciseMultiPatterns(t, []string{"", "he", "hel", "heo", "heloz", "hellox", "heloxo"}, []pwanted{
		{`{"x":[{"wildcard": "he*lo"}]}`, []string{"helo", "hello", "helllo"}},
		{`{"x":["helox"]}`, []string{"helox"}}})
	exerciseMultiPatterns(t, []string{"", "he", "helx", "helo", "hexlx", "hellox", "heloxx"}, []pwanted{
		{`{"x":[{"wildcard": "he*l"}]}`, []string{"hel", "hexl", "hexxxl"}},
		{`{"x":["helox"]}`, []string{"helox"}}})
	exerciseMultiPatterns(t, []string{"", "h", "hxlox", "hxelox"}, []pwanted{
		{`{"x":[{"wildcard": "he*"}]}`, []string{"he", "helo", "helox", "heloxx"}},
		{`{"x":["helox"]}`, []string{"helox"}}})
	exerciseMultiPatterns(t, []string{"", "h", "he", "hel", "hexxo", "hexxohexxo"}, []pwanted{
		{`{"x":[{"wildcard": "h*l*o"}]}`, []string{"hlo", "helo", "hllo", "hello", "hexloo", "hellohello", "hellohellxo"}},
		{`{"x":["hellohello"]}`, []string{"hellohello"}}})
	exerciseMultiPatterns(t, []string{"", "h", "he", "hlo", "hexxo", "hexxohexxo"}, []pwanted{
		{`{"x":[{"wildcard": "he*l*"}]}`, []string{"hel", "helo", "hexl", "hello", "hexloo", "hellohellx", "hellohello"}},
		{`{"x":["hellohello"]}`, []string{"hellohello"}}})
}

func TestWildcardInvalidEscape(t *testing.T) {
	cm := newCoreMatcher()
	goods := []string{
		`he*\\**`,
	}
	bads := []string{
		`he\\llo`, `foo**bar`, `**f`, `x**`, `x\\`,
	}
	for _, good := range goods {
		pattern := fmt.Sprintf(`{"x": [{"wildcard": "%s"}]}`, good)
		err := cm.addPattern("x", pattern)
		if err != nil {
			t.Error("rejected \\:", good)
		}
	}
	for _, bad := range bads {
		pattern := fmt.Sprintf(`{"x": [{"wildcard": "%s"}]}`, bad)
		err := cm.addPattern("x", pattern)
		if err == nil {
			t.Error("Allowed bad \\:", bad)
		}
	}
}

type pwanted struct {
	pattern string
	wanted  []string
}

func exerciseMultiPatterns(t *testing.T, nos []string, pws []pwanted) {
	t.Helper()
	cm := newCoreMatcher()
	for _, pw := range pws {
		err := cm.addPattern(pw.pattern, pw.pattern)
		if err != nil {
			t.Errorf("Addpattern %s: %s", pw.pattern, err.Error())
		}
	}
	for _, pw := range pws {
		for _, want := range pw.wanted {
			event := fmt.Sprintf(`{"x":"%s"}`, want)
			matches, _ := cm.matchesForJSONEvent([]byte(event))
			var i int
			for i = 0; i < len(matches); i++ {
				if matches[i] == pw.pattern {
					break
				}
			}
			if i == len(matches) {
				t.Errorf("event [%s] didn't match pattern[%s]", event, pw.pattern)
			}
		}
	}
	for _, n := range nos {
		event := fmt.Sprintf(`{"x": "%s"}`, n)
		matches, _ := cm.matchesForJSONEvent([]byte(event))
		if len(matches) != 0 {
			t.Errorf("%s did match", n)
		}
	}
}

func exercisePattern(t *testing.T, pattern string, yes []string, no []string) {
	t.Helper()
	cm := newCoreMatcher()
	err := cm.addPattern(pattern, fmt.Sprintf(`{"x": [ {"wildcard": "%s"}]}`, pattern))
	if err != nil {
		t.Errorf("Addpattern %s: %s", pattern, err.Error())
	}
	for _, y := range yes {
		event := fmt.Sprintf(`{"x": "%s"}`, y)
		matches, _ := cm.matchesForJSONEvent([]byte(event))
		if len(matches) != 1 || matches[0] != pattern {
			t.Errorf("[%s] doesn't match %s", y, pattern)
		}
	}
	for _, n := range no {
		event := fmt.Sprintf(`{"x": "%s"}`, n)
		matches, _ := cm.matchesForJSONEvent([]byte(event))
		if len(matches) != 0 {
			t.Errorf("%s did match %s", n, pattern)
		}
	}
}
