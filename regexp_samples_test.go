package quamina

import (
	"testing"
)

// This file produced by processing a set of XSD regexp syntax samples by Michael Kay
// from the repo https://github.com/qt4cg/xslt40-test - thanks to Michael!
// The code may be found in codegen/qtest-main.go. It is fairly horrible and my assumption
// is that it will never be run again; there is plenty of room for more regexp-related testing
// but I think as much benefit has been extracted from this sample set as is reasonable to expect.

type regexpSample struct {
	regex     string
	matches   []string
	nomatches []string
	valid     bool
}

func TestRegexpSamplesExist(t *testing.T) {
	if len(regexpSamples) == 0 {
		t.Error("no samples")
	}
}

// test-case numbers are off by one, i.e. the first one below is actually for regex-syntax-0001
var regexpSamples = []regexpSample{
	//    <test-case name="regex-syntax-0002">
	{
		regex:     "",
		matches:   []string{"", ""},
		nomatches: []string{"a", " ", "\r", "	", "\n"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0003">
	{
		regex:     "a",
		matches:   []string{"a"},
		nomatches: []string{"aa", "b", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0004">
	{
		regex:     "a|a",
		matches:   []string{"a"},
		nomatches: []string{"aa", "b", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0005">
	{
		regex:     "a|b",
		matches:   []string{"a", "b"},
		nomatches: []string{"aa", "bb", "ab", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0006">
	{
		regex:     "ab",
		matches:   []string{"ab"},
		nomatches: []string{"a", "b", "aa", "bb", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0007">
	{
		regex:     "a|b|a|c|b|d|a",
		matches:   []string{"a", "b", "c", "d"},
		nomatches: []string{"aa", "ac", "e"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0008">
	{
		regex:     "       a|b      ",
		matches:   []string{"       a"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0009">
	{
		regex:     "ab?c",
		matches:   []string{"ac", "abc"},
		nomatches: []string{"a", "ab", "bc", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0010">
	{
		regex:     "abc?",
		matches:   []string{"ab", "abc"},
		nomatches: []string{"a", "bc", "abcc", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0011">
	{
		regex:     "ab+c",
		matches:   []string{"abc", "abbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbc"},
		nomatches: []string{"ac", "bbbc", "abbb", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0012">
	{
		regex:     "abc+",
		matches:   []string{"abc", "abccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		nomatches: []string{"a", "ab", "abcd"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0013">
	{
		regex:     "ab*c",
		matches:   []string{"abc", "abbbbbbbc", "ac"},
		nomatches: []string{"a", "ab", "bc", "c", "abcb", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0014">
	{
		regex:     "abc*",
		matches:   []string{"abc", "ab", "abccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		nomatches: []string{"a", "abcd", "abbc", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0015">
	{
		regex:     "a?b+c*",
		matches:   []string{"b", "ab", "bcccccc", "abc", "abbbc"},
		nomatches: []string{"aabc", "a", "c", "ac", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0016">
	{
		regex:     "(ab+c)a?~?~??",
		matches:   []string{"abc?", "abbbc??", "abca??", "abbbbca?"},
		nomatches: []string{"ac??", "bc??", "abc", "abc???"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0017">
	{
		regex: "?a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0018">
	{
		regex: "+a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0019">
	{
		regex: "*a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0020">
	{
		regex: "{1}a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0021">
	{
		regex:     "a{0}",
		matches:   []string{"", ""},
		nomatches: []string{"a"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0022">
	{
		regex: "a{2,1}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0023">
	{
		regex: "a{1,0}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0024">
	{
		regex:     "((ab){2})?",
		matches:   []string{"abab", ""},
		nomatches: []string{"a", "ab", "ababa", "abababab"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0025">
	{
		regex:     "(a{2})+",
		matches:   []string{"aa", "aaaa", "aaaaaaaaaaaaaaaaaaaa"},
		nomatches: []string{"", "a", "a2", "aaa"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0026">
	{
		regex:     "(a{2})*",
		matches:   []string{"", "aa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		nomatches: []string{"a", "aaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0027">
	{
		regex:     "ab{2}c",
		matches:   []string{"abbc"},
		nomatches: []string{"ac", "abc", "abbbc", "a", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0028">
	{
		regex:     "abc{2}",
		matches:   []string{"abcc"},
		nomatches: []string{"abc", "abccc", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0029">
	{
		regex:     "a*b{2,4}c{0}",
		matches:   []string{"aaabbb", "bb", "bbb", "bbbb"},
		nomatches: []string{"ab", "abbc", "bbc", "abbbbb", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0030">
	{
		regex:     "((ab)(ac){0,2})?",
		matches:   []string{"ab", "abac", "abacac"},
		nomatches: []string{"ac", "abacacac", "abaca", "abab", "abacabac"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0031">
	{
		regex: "(a~sb){0,2}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0032">
	{
		regex:     "(ab){2,}",
		matches:   []string{"abab", "ababab", "ababababababababababababababababababababababababababababababababab"},
		nomatches: []string{"ab", "ababa", "ababaa", "ababababa", "abab abab", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0033">
	{
		regex: "a{,2}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0034">
	{
		regex: "(ab){2,0}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0035">
	{
		regex:     "(ab){0,0}",
		matches:   []string{""},
		nomatches: []string{"a", "ab"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0036">
	{
		regex:     "a{0,1}b{1,2}c{2,3}",
		matches:   []string{"abcc", "abccc", "abbcc", "abbccc", "bbcc", "bbccc"},
		nomatches: []string{"aabcc", "bbbcc", "acc", "aabcc", "abbc", "abbcccc"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0037">
	{
		regex:     "(((((boy)|(girl))[0-1][x-z]{2})?)|(man|woman)[0-1]?[y|n])*",
		matches:   []string{"", "boy0xx", "woman1y", "girl1xymany", "boy0xxwoman1ygirl1xymany", "boy0xxwoman1ygirl1xymanyboy0xxwoman1ygirl1xymany"},
		nomatches: []string{"boy0xxwoman1ygirl1xyman", "boyxx"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0038">
	{
		regex: "((a)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0039">
	{
		regex: "(a))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0040">
	{
		regex: "ab|(d))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0041">
	{
		regex: "((a*(b*)((a))*(a))))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0042">
	{
		regex: "~",
		valid: false,
	},
	//    <test-case name="regex-syntax-0043">
	{
		regex: "?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0044">
	{
		regex: "*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0045">
	{
		regex: "+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0046">
	{
		regex: "(",
		valid: false,
	},
	//    <test-case name="regex-syntax-0047">
	{
		regex: ")",
		valid: false,
	},
	//    <test-case name="regex-syntax-0048">
	{
		regex:     "|",
		matches:   []string{""},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0049">
	{
		regex: "[",
		valid: false,
	},
	//    <test-case name="regex-syntax-0050">
	{
		regex:     "~.~~~?~*~+~{~}~[~]~(~)~|",
		matches:   []string{".~?*+{}[]()|"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0051">
	{
		regex:     "(([~.~~~?~*~+~{~}~[~]~(~)~|]?)*)+",
		matches:   []string{".~?*+{}[]()|.~?*+{}[]()|.~?*+{}[]()|"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0052">
	{
		regex:     "[^2-9a-x]{2}",
		matches:   []string{"1z"},
		nomatches: []string{"1x"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0053">
	{
		regex: "[^~s]{3}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0054">
	{
		regex:     "[^@]{0,2}",
		matches:   []string{"", "a", "ab", " a"},
		nomatches: []string{"@"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0055">
	{
		regex:     "[^-z]+",
		matches:   []string{""},
		nomatches: []string{"aaz", "a-z"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0056">
	{
		regex: "[a-d-[b-c]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0056a">
	{
		regex: "[^a-d-b-c]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0057">
	{
		regex: "[^a-d-b-c]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0058">
	{
		regex:     "[a-~}]+",
		matches:   []string{"abcxyz}"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0059">
	{
		regex: "[a-b-[0-9]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0060">
	{
		regex: "[a-c-[^a-c]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0061">
	{
		regex: "[a-z-[^a]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0062">
	{
		regex: "[^~p{IsBasicLatin}]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0063">
	{
		regex: "[^~p{IsBasicLatin}]*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0064">
	{
		regex: "[^~P{IsBasicLatin}]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0065">
	{
		regex:     "[^~?]",
		matches:   []string{""},
		nomatches: []string{"?"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0066">
	{
		regex:     "([^~?])*",
		matches:   []string{"a+*abc"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0067">
	{
		regex: "~c[^~d]~c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0068">
	{
		regex: "~c[^~s]~c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0069">
	{
		regex:     "[^~^a]",
		matches:   []string{""},
		nomatches: []string{"^", "a"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0070">
	{
		regex:     "[a-abc]{3}",
		matches:   []string{"abc"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0071">
	{
		regex:     "[a-~}-]+",
		matches:   []string{"}-"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0072">
	{
		regex: "[a--b]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0073">
	{
		regex: "[^[a-b]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0074">
	{
		regex:     "[a]",
		matches:   []string{""},
		nomatches: []string{"b", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0075">
	{
		regex:     "[1-3]{1,4}",
		matches:   []string{"123"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0076">
	{
		regex:     "[a-a]",
		matches:   []string{"a"},
		nomatches: []string{"b"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0077">
	{
		regex:     "[0-z]*",
		matches:   []string{"1234567890:;<=>?@Azaz"},
		nomatches: []string{"{", "/"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0078">
	{
		regex:     "[~n]",
		matches:   []string{""},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0079">
	{
		regex:     "[~t]",
		matches:   []string{""},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0080">
	{
		regex:     "[~~~|~.~?~*~+~(~)~{~}~-~[~]~^]*",
		matches:   []string{"~|.?*+(){}-[]^"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0081">
	{
		regex:     "[^a-z^]",
		matches:   []string{""},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0082">
	{
		regex:     "[\\-~{^]",
		matches:   []string{""},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0083">
	{
		regex: "[~C~?a-c~?]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0084">
	{
		regex: "[~c~?a-c~?]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0085">
	{
		regex: "[~D~?a-c~?]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0086">
	{
		regex: "[~S~?a-c~?]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0086a">
	{
		regex: "[a-c-1-4x-z-7-9]*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0087">
	{
		regex: "[a-c-1-4x-z-7-9]*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0088">
	{
		regex: "[a-\\]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0089">
	{
		regex: "[a-~[]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0090">
	{
		regex:     "[~*a]*",
		matches:   []string{"a*a****aaaaa*"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0091">
	{
		regex: "[a-;]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0092">
	{
		regex:     "[1-~]]+",
		matches:   []string{"1]"},
		nomatches: []string{"0", "^"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0093">
	{
		regex:     "[=->]",
		matches:   []string{"=", ">"},
		nomatches: []string{"~?"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0094">
	{
		regex: "[>-=]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0095">
	{
		regex:     "[@]",
		matches:   []string{"@"},
		nomatches: []string{"a"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0096">
	{
		regex:     "[࿿]",
		matches:   []string{"࿿"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0097">
	{
		regex:     "[𐀀]",
		matches:   []string{"𐀀"},
		nomatches: []string{"𐀁"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0098">
	{
		regex: "[~]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0099">
	{
		regex:     "[~~~[~]]{0,3}",
		matches:   []string{"~", "[", "]", "~[", "~[]", "[]", "[~~", "~]~", "[]["},
		nomatches: []string{"~[][", "~]~]", "[][]"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0100">
	{
		regex:     "[-]",
		matches:   []string{"-"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0101">
	{
		regex:     "[-a]+",
		matches:   []string{"a--aa---"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0102">
	{
		regex:     "[a-]*",
		matches:   []string{"a--aa---"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0102a">
	{
		regex: "[a-a-x-x]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0103">
	{
		regex: "[a-a-x-x]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0104">
	{
		regex:     "[~n~r~t~~~|~.~-~^~?~*~+~{~}~[~]~(~)]*",
		matches:   []string{"~|.-^?*+[]{}()*[[]{}}))\n\r		\n\n\r*()"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0105">
	{
		regex:     "[a~*]*",
		matches:   []string{"a**", "aa*", "a"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0106">
	{
		regex:     "[(a~?)?]+",
		matches:   []string{"a?", "a?a?a?", "a", "a??", "aa?"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0107">
	{
		regex:     "~~t",
		matches:   []string{"~t"},
		nomatches: []string{"t", "~~t", "	"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0108">
	{
		regex:     "~~n",
		matches:   []string{"~n"},
		nomatches: []string{"n", "~~n", "\n"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0109">
	{
		regex:     "~~r",
		matches:   []string{"~r"},
		nomatches: []string{"r", "~~r", "\r"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0110">
	{
		regex:     "~n",
		matches:   []string{""},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0111">
	{
		regex:     "~t",
		matches:   []string{""},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0112">
	{
		regex:     "~~",
		matches:   []string{"~"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0113">
	{
		regex:     "~|",
		matches:   []string{"|"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0114">
	{
		regex:     "~.",
		matches:   []string{"."},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0115">
	{
		regex:     "~-",
		matches:   []string{"-"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0116">
	{
		regex:     "~^",
		matches:   []string{"^"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0117">
	{
		regex:     "~?",
		matches:   []string{"?"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0118">
	{
		regex:     "~*",
		matches:   []string{"*"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0119">
	{
		regex:     "~+",
		matches:   []string{"+"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0120">
	{
		regex:     "~{",
		matches:   []string{"{"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0121">
	{
		regex:     "~}",
		matches:   []string{"}"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0122">
	{
		regex:     "~(",
		matches:   []string{"("},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0123">
	{
		regex:     "~)",
		matches:   []string{")"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0124">
	{
		regex:     "~[",
		matches:   []string{"["},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0125">
	{
		regex:     "~]",
		matches:   []string{"]"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0126">
	{
		regex:     "~n~~~r~|~t~.~-~^~?~*~+~{~}~(~)~[~]",
		matches:   []string{""},
		nomatches: []string{"\n~\r|	.-^?*+{}()[", "~\r|	.-^?*+{}()[]", "\n~\r|	-^?*+{}()[]"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0127">
	{
		regex:     "~n~na~n~nb~n~n",
		matches:   []string{""},
		nomatches: []string{"\n\na\n\nb\n", "\na\n\nb\n\n", "\n\na\n\n\n\n", "\n\na\n\r\nb\n\n"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0128">
	{
		regex:     "~r~ra~r~rb~r~r",
		matches:   []string{"\r\ra\r\rb\r\r"},
		nomatches: []string{"\r\ra\r\rb\r", "\ra\r\rb\r\r", "\r\ra\r\r\r\r", "\r\ra\r\n\rb\r\r"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0129">
	{
		regex:     "~t~ta~t~tb~t~t",
		matches:   []string{""},
		nomatches: []string{"		a		b	", "	a		b		", "		a				", "		a			b		"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0130">
	{
		regex:     "a~r~nb",
		matches:   []string{"a\r\nb"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0131">
	{
		regex:     "~n~ra~n~rb",
		matches:   []string{"\n\ra\n\rb"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0132">
	{
		regex:     "~ta~tb~tc~t",
		matches:   []string{"	a	b	c	"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0133">
	{
		regex:     "~na~nb~nc~n",
		matches:   []string{"\na\nb\nc\n"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0134">
	{
		regex: "(~t|~s)a(~r~n|~r|~n|~s)+(~s|~t)b(~s|~r~n|~r|~n)*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0135">
	{
		regex:     "~~c",
		matches:   []string{"~c"},
		nomatches: []string{"~p{_xmlC}", "~~c", "~~"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0136">
	{
		regex:     "~~.,~~s,~~S,~~i,~~I,~~c,~~C,~~d,~~D,~~w,~~W",
		matches:   []string{"~.,~s,~S,~i,~I,~c,~C,~d,~D,~w,~W"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0137">
	{
		regex:     "~~.*,~~s*,~~S*,~~i*,~~I?,~~c+,~~C+,~~d{0,3},~~D{1,1000},~~w*,~~W+",
		matches:   []string{"~.abcd,~sssss,~SSSSSS,~iiiiiii,~,~c,~CCCCCC,~ddd,~D,~wwwwwww,~WWW"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0138">
	{
		regex:     "[~p{L}*]{0,2}",
		matches:   []string{"aX"},
		nomatches: []string{"aBC"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0139">
	{
		regex:     "(~p{Ll}~p{Cc}~p{Nd})*",
		matches:   []string{""},
		nomatches: []string{" "},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0140">
	{
		regex:     "~p{L}*",
		matches:   []string{""},
		nomatches: []string{"⃝"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0141">
	{
		regex:     "~p{Lu}*",
		matches:   []string{"A𝞨"},
		nomatches: []string{"a"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0142">
	{
		regex:     "~p{Ll}*",
		matches:   []string{"a𝟉"},
		nomatches: []string{"ǅ"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0143">
	{
		regex:     "~p{Lt}*",
		matches:   []string{"ǅῼ"},
		nomatches: []string{"ʰ"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0144">
	{
		regex:     "~p{Lm}*",
		matches:   []string{"ʰﾟ"},
		nomatches: []string{"א"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0145">
	{
		regex:     "~p{Lo}*",
		matches:   []string{"א𪘀"},
		nomatches: []string{"ً"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0146">
	{
		regex:     "~p{M}*",
		matches:   []string{"ً𝆭ः𝅲ः𝅲⃝⃝⃠"},
		nomatches: []string{"ǅ"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0147">
	{
		regex:     "~p{Mn}*",
		matches:   []string{"ً𝆭"},
		nomatches: []string{"ः"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0148">
	{
		regex:     "~p{Mc}*",
		matches:   []string{"ः𝅲"},
		nomatches: []string{"⃝"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0149">
	{
		regex:     "~p{Me}*",
		matches:   []string{"⃝⃠"},
		nomatches: []string{"０"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0150">
	{
		regex:     "~p{N}*",
		matches:   []string{"０𝟿𐍊𐍊〥²²𐌣"},
		nomatches: []string{"ः"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0151">
	{
		regex:     "~p{Nd}*",
		matches:   []string{"０𝟿"},
		nomatches: []string{"𐍊"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0152">
	{
		regex:     "~p{Nl}*",
		matches:   []string{"𐍊〥"},
		nomatches: []string{"²"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0153">
	{
		regex:     "~p{No}*",
		matches:   []string{"²𐌣"},
		nomatches: []string{"‿"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0154">
	{
		regex:     "~p{P}*",
		matches:   []string{"‿･〜〜－〝〝｢〞〞｣««‹»»›¿¿､"},
		nomatches: []string{"²"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0155">
	{
		regex:     "~p{Pc}*",
		matches:   []string{""},
		nomatches: []string{"〜"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0156">
	{
		regex:     "~p{Pd}*",
		matches:   []string{"〜－"},
		nomatches: []string{"〝"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0157">
	{
		regex:     "~p{Ps}*",
		matches:   []string{"〝｢"},
		nomatches: []string{"〞"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0158">
	{
		regex:     "~p{Pe}*",
		matches:   []string{"〞｣"},
		nomatches: []string{"«"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0159">
	{
		regex:     "~p{Pi}*",
		matches:   []string{"«‹"},
		nomatches: []string{"»"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0160">
	{
		regex:     "~p{Pf}*",
		matches:   []string{"»›"},
		nomatches: []string{"¿"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0161">
	{
		regex:     "~p{Po}*",
		matches:   []string{"¿､"},
		nomatches: []string{" "},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0162">
	{
		regex:     "~p{Z}*",
		matches:   []string{" 　    "},
		nomatches: []string{"¿"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0163">
	{
		regex:     "~p{Zs}*",
		matches:   []string{" 　"},
		nomatches: []string{" "},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0164">
	{
		regex:     "~p{Zl}*",
		matches:   []string{" "},
		nomatches: []string{" "},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0165">
	{
		regex:     "~p{Zp}*",
		matches:   []string{" "},
		nomatches: []string{"⁄"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0166">
	{
		regex:     "~p{S}*",
		matches:   []string{"⁄￢₠₠￦゛゛￣㆐㆐𝇝"},
		nomatches: []string{" "},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0167">
	{
		regex:     "~p{Sm}*",
		matches:   []string{"⁄￢"},
		nomatches: []string{"₠"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0168">
	{
		regex:     "~p{Sc}*",
		matches:   []string{"₠￦"},
		nomatches: []string{"゛"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0169">
	{
		regex:     "~p{Sk}*",
		matches:   []string{"゛￣"},
		nomatches: []string{"㆐"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0170">
	{
		regex:     "~p{So}*",
		matches:   []string{"㆐𝇝"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0171">
	{
		regex:     "~p{C}*",
		matches:   []string{""},
		nomatches: []string{"₠"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0172">
	{
		regex:     "~p{Cc}*",
		matches:   []string{""},
		nomatches: []string{"܏"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0173">
	{
		regex:     "~p{Cf}*",
		matches:   []string{"܏󠁸"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0174">
	{
		regex:     "(~p{Co})*",
		matches:   []string{"􀀀󰀀󿿽􏿽"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0175">
	{
		regex:     "~p{Co}*",
		matches:   []string{""},
		nomatches: []string{"⁄"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0176">
	{
		regex:     "~p{Cn}*",
		matches:   []string{""},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0177">
	{
		regex:     "~P{L}*",
		matches:   []string{"_", "⃝"},
		nomatches: []string{"aAbB", "A𝞨aa𝟉ǅǅῼʰʰﾟאא𪘀"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0178">
	{
		regex:     "[~P{L}*]{0,2}",
		matches:   []string{"", "#$"},
		nomatches: []string{"!$#", "A"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0179">
	{
		regex:     "~P{Lu}*",
		matches:   []string{"a"},
		nomatches: []string{"A𝞨"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0180">
	{
		regex:     "~P{Ll}*",
		matches:   []string{"ǅ"},
		nomatches: []string{"a𝟉"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0181">
	{
		regex:     "~P{Lt}*",
		matches:   []string{"ʰ"},
		nomatches: []string{"ǅῼ"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0182">
	{
		regex:     "~P{Lm}*",
		matches:   []string{"א"},
		nomatches: []string{"ʰﾟ"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0183">
	{
		regex:     "~P{Lo}*",
		matches:   []string{"ً"},
		nomatches: []string{"א𪘀"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0184">
	{
		regex:     "~P{M}*",
		matches:   []string{"ǅ"},
		nomatches: []string{"ً𝆭ः𝅲ः𝅲⃝⃝⃠"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0185">
	{
		regex:     "~P{Mn}*",
		matches:   []string{"ः𝅲"},
		nomatches: []string{"ً𝆭"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0186">
	{
		regex:     "~P{Mc}*",
		matches:   []string{"⃝"},
		nomatches: []string{"ः𝅲"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0187">
	{
		regex:     "~P{Me}*",
		matches:   []string{"０"},
		nomatches: []string{"⃝⃠"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0188">
	{
		regex:     "~P{N}*",
		matches:   []string{"ः"},
		nomatches: []string{"０𝟿𐍊𐍊〥²²𐌣"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0189">
	{
		regex:     "~P{Nd}*",
		matches:   []string{"𐍊"},
		nomatches: []string{"０𝟿"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0190">
	{
		regex:     "~P{Nl}*",
		matches:   []string{"²"},
		nomatches: []string{"𐍊〥"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0191">
	{
		regex:     "~P{No}*",
		matches:   []string{"‿"},
		nomatches: []string{"²𐌣"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0192">
	{
		regex:     "~P{P}*",
		matches:   []string{"²"},
		nomatches: []string{"‿･〜〜－〝〝｢〞〞｣««‹»»›¿¿､"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0193">
	{
		regex:     "~P{Pc}*",
		matches:   []string{"〜"},
		nomatches: []string{"‿･"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0194">
	{
		regex:     "~P{Pd}*",
		matches:   []string{"〝"},
		nomatches: []string{"〜－"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0195">
	{
		regex:     "~P{Ps}*",
		matches:   []string{"〞"},
		nomatches: []string{"〝｢"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0196">
	{
		regex:     "~P{Pe}*",
		matches:   []string{"«"},
		nomatches: []string{"〞｣"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0197">
	{
		regex:     "~P{Pi}*",
		matches:   []string{"»"},
		nomatches: []string{"«‹"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0198">
	{
		regex:     "~P{Pf}*",
		matches:   []string{"¿"},
		nomatches: []string{"»›"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0199">
	{
		regex:     "~P{Po}*",
		matches:   []string{" "},
		nomatches: []string{"¿､"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0200">
	{
		regex:     "~P{Z}*",
		matches:   []string{"¿"},
		nomatches: []string{" 　    "},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0201">
	{
		regex:     "~P{Zs}*",
		matches:   []string{" "},
		nomatches: []string{" 　"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0202">
	{
		regex:     "~P{Zl}*",
		matches:   []string{" "},
		nomatches: []string{" "},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0203">
	{
		regex:     "~P{Zp}*",
		matches:   []string{"⁄"},
		nomatches: []string{" "},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0204">
	{
		regex:     "~P{S}*",
		matches:   []string{" "},
		nomatches: []string{"⁄￢₠₠￦゛゛￣㆐㆐𝇝"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0205">
	{
		regex:     "~P{Sm}*",
		matches:   []string{"₠"},
		nomatches: []string{"⁄￢"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0206">
	{
		regex:     "~P{Sc}*",
		matches:   []string{"゛"},
		nomatches: []string{"₠￦"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0207">
	{
		regex:     "~P{Sk}*",
		matches:   []string{"㆐"},
		nomatches: []string{"゛￣"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0208">
	{
		regex:     "~P{So}*",
		matches:   []string{""},
		nomatches: []string{"㆐𝇝"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0209">
	{
		regex:     "~P{C}*",
		matches:   []string{"₠"},
		nomatches: []string{"	܏܏󠁸􀀀󰀀󿿽􏿽"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0210">
	{
		regex:     "~P{Cc}*",
		matches:   []string{"܏"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0211">
	{
		regex:     "~P{Cf}*",
		matches:   []string{""},
		nomatches: []string{"܏󠁸"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0212">
	{
		regex:     "~P{Co}*",
		matches:   []string{"⁄"},
		nomatches: []string{"􀀀󰀀󿿽􏿽"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0213">
	{
		regex: "~p{~~L}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0214">
	{
		regex:     "~~~p{L}*",
		matches:   []string{"~a"},
		nomatches: []string{"a"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0215">
	{
		regex: "~p{Is}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0216">
	{
		regex: "~P{Is}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0217">
	{
		regex: "~p{IsaA0-a9}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0218">
	{
		regex: "~p{IsBasicLatin}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0219">
	{
		regex: "~p{IsLatin-1Supplement}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0220">
	{
		regex: "~p{IsLatinExtended-A}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0221">
	{
		regex: "~p{IsLatinExtended-B}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0222">
	{
		regex: "~p{IsIPAExtensions}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0223">
	{
		regex: "~p{IsSpacingModifierLetters}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0224">
	{
		regex: "~p{IsArmenian}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0225">
	{
		regex: "~p{IsHebrew}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0226">
	{
		regex: "~p{IsArabic}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0227">
	{
		regex: "~p{IsSyriac}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0228">
	{
		regex: "~p{IsThaana}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0229">
	{
		regex: "~p{IsDevanagari}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0230">
	{
		regex: "~p{IsBengali}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0231">
	{
		regex: "~p{IsGurmukhi}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0232">
	{
		regex: "~p{IsGujarati}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0233">
	{
		regex: "~p{IsOriya}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0234">
	{
		regex: "~p{IsTamil}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0235">
	{
		regex: "~p{IsTelugu}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0236">
	{
		regex: "~p{IsKannada}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0237">
	{
		regex: "~p{IsMalayalam}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0238">
	{
		regex: "~p{IsSinhala}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0239">
	{
		regex: "~p{IsThai}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0240">
	{
		regex: "~p{IsLao}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0241">
	{
		regex: "~p{IsTibetan}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0242">
	{
		regex: "~p{IsMyanmar}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0243">
	{
		regex: "~p{IsGeorgian}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0244">
	{
		regex: "~p{IsHangulJamo}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0245">
	{
		regex: "~p{IsEthiopic}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0246">
	{
		regex: "~p{IsCherokee}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0247">
	{
		regex: "~p{IsUnifiedCanadianAboriginalSyllabics}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0248">
	{
		regex: "~p{IsOgham}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0249">
	{
		regex: "~p{IsRunic}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0250">
	{
		regex: "~p{IsKhmer}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0251">
	{
		regex: "~p{IsMongolian}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0252">
	{
		regex: "~p{IsLatinExtendedAdditional}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0253">
	{
		regex: "~p{IsGreekExtended}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0254">
	{
		regex: "~p{IsGeneralPunctuation}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0255">
	{
		regex: "~p{IsSuperscriptsandSubscripts}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0256">
	{
		regex: "~p{IsCurrencySymbols}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0257">
	{
		regex: "~p{IsCombiningDiacriticalMarksforSymbols}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0258">
	{
		regex: "~p{IsLetterlikeSymbols}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0259">
	{
		regex: "~p{IsNumberForms}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0260">
	{
		regex: "~p{IsArrows}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0261">
	{
		regex: "~p{IsMathematicalOperators}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0262">
	{
		regex: "~p{IsMiscellaneousTechnical}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0263">
	{
		regex: "~p{IsControlPictures}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0264">
	{
		regex: "~p{IsOpticalCharacterRecognition}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0265">
	{
		regex: "~p{IsEnclosedAlphanumerics}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0266">
	{
		regex: "~p{IsBoxDrawing}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0267">
	{
		regex: "~p{IsBlockElements}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0268">
	{
		regex: "~p{IsGeometricShapes}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0269">
	{
		regex: "~p{IsMiscellaneousSymbols}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0270">
	{
		regex: "~p{IsDingbats}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0271">
	{
		regex: "~p{IsBraillePatterns}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0272">
	{
		regex: "~p{IsCJKRadicalsSupplement}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0273">
	{
		regex: "~p{IsKangxiRadicals}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0274">
	{
		regex: "~p{IsIdeographicDescriptionCharacters}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0275">
	{
		regex: "~p{IsCJKSymbolsandPunctuation}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0276">
	{
		regex: "~p{IsHiragana}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0277">
	{
		regex: "~p{IsKatakana}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0278">
	{
		regex: "~p{IsBopomofo}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0279">
	{
		regex: "~p{IsHangulCompatibilityJamo}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0280">
	{
		regex: "~p{IsKanbun}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0281">
	{
		regex: "~p{IsBopomofoExtended}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0282">
	{
		regex: "~p{IsEnclosedCJKLettersandMonths}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0283">
	{
		regex: "~p{IsCJKCompatibility}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0284">
	{
		regex: "~p{IsCJKUnifiedIdeographsExtensionA}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0285">
	{
		regex: "~p{IsCJKUnifiedIdeographs}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0286">
	{
		regex: "~p{IsYiSyllables}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0287">
	{
		regex: "~p{IsYiRadicals}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0288">
	{
		regex: "~p{IsHangulSyllables}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0288a">
	{
		regex: "~p{IsPrivateUseArea}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0288b">
	{
		regex: "~p{IsSupplementaryPrivateUseArea-A}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0289">
	{
		regex: "~p{IsSupplementaryPrivateUseArea-B}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0290">
	{
		regex: "~p{IsCJKCompatibilityIdeographs}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0291">
	{
		regex: "~p{IsAlphabeticPresentationForms}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0292">
	{
		regex: "~p{IsArabicPresentationForms-A}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0293">
	{
		regex: "~p{IsCombiningHalfMarks}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0294">
	{
		regex: "~p{IsCJKCompatibilityForms}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0295">
	{
		regex: "~p{IsSmallFormVariants}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0296">
	{
		regex: "~p{IsArabicPresentationForms-B}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0297">
	{
		regex: "~p{IsHalfwidthandFullwidthForms}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0298">
	{
		regex: "~p{IsSpecials}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0299">
	{
		regex: "~p{IsBasicLatin}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0300">
	{
		regex: "~p{IsLatin-1Supplement}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0301">
	{
		regex: "~p{IsLatinExtended-A}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0302">
	{
		regex: "~p{IsLatinExtended-B}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0303">
	{
		regex: "~p{IsIPAExtensions}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0304">
	{
		regex: "~p{IsSpacingModifierLetters}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0305">
	{
		regex: "~p{IsCyrillic}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0306">
	{
		regex: "~p{IsArmenian}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0307">
	{
		regex: "~p{IsHebrew}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0308">
	{
		regex: "~p{IsArabic}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0309">
	{
		regex: "~p{IsSyriac}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0310">
	{
		regex: "~p{IsThaana}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0311">
	{
		regex: "~p{IsDevanagari}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0312">
	{
		regex: "~p{IsBengali}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0313">
	{
		regex: "~p{IsGurmukhi}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0314">
	{
		regex: "~p{IsGujarati}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0315">
	{
		regex: "~p{IsOriya}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0316">
	{
		regex: "~p{IsTamil}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0317">
	{
		regex: "~p{IsTelugu}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0318">
	{
		regex: "~p{IsKannada}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0319">
	{
		regex: "~p{IsMalayalam}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0320">
	{
		regex: "~p{IsSinhala}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0321">
	{
		regex: "~p{IsThai}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0322">
	{
		regex: "~p{IsLao}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0323">
	{
		regex: "~p{IsTibetan}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0324">
	{
		regex: "~p{IsMyanmar}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0325">
	{
		regex: "~p{IsGeorgian}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0326">
	{
		regex: "~p{IsHangulJamo}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0327">
	{
		regex: "~p{IsEthiopic}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0328">
	{
		regex: "~p{IsCherokee}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0329">
	{
		regex: "~p{IsUnifiedCanadianAboriginalSyllabics}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0330">
	{
		regex: "~p{IsOgham}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0331">
	{
		regex: "~p{IsRunic}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0332">
	{
		regex: "~p{IsKhmer}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0333">
	{
		regex: "~p{IsMongolian}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0334">
	{
		regex: "~p{IsLatinExtendedAdditional}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0335">
	{
		regex: "~p{IsGreekExtended}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0336">
	{
		regex: "~p{IsGeneralPunctuation}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0337">
	{
		regex: "~p{IsSuperscriptsandSubscripts}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0338">
	{
		regex: "~p{IsCurrencySymbols}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0339">
	{
		regex: "~p{IsCombiningDiacriticalMarksforSymbols}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0340">
	{
		regex: "~p{IsLetterlikeSymbols}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0341">
	{
		regex: "~p{IsNumberForms}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0342">
	{
		regex: "~p{IsArrows}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0343">
	{
		regex: "~p{IsMathematicalOperators}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0344">
	{
		regex: "~p{IsMiscellaneousTechnical}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0345">
	{
		regex: "~p{IsControlPictures}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0346">
	{
		regex: "~p{IsOpticalCharacterRecognition}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0347">
	{
		regex: "~p{IsEnclosedAlphanumerics}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0348">
	{
		regex: "~p{IsBoxDrawing}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0349">
	{
		regex: "~p{IsBlockElements}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0350">
	{
		regex: "~p{IsGeometricShapes}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0351">
	{
		regex: "~p{IsMiscellaneousSymbols}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0352">
	{
		regex: "~p{IsDingbats}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0353">
	{
		regex: "~p{IsBraillePatterns}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0354">
	{
		regex: "~p{IsCJKRadicalsSupplement}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0355">
	{
		regex: "~p{IsKangxiRadicals}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0356">
	{
		regex: "~p{IsIdeographicDescriptionCharacters}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0357">
	{
		regex: "~p{IsCJKSymbolsandPunctuation}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0358">
	{
		regex: "~p{IsHiragana}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0359">
	{
		regex: "~p{IsKatakana}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0360">
	{
		regex: "~p{IsBopomofo}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0361">
	{
		regex: "~p{IsHangulCompatibilityJamo}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0362">
	{
		regex: "~p{IsKanbun}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0363">
	{
		regex: "~p{IsBopomofoExtended}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0364">
	{
		regex: "~p{IsEnclosedCJKLettersandMonths}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0365">
	{
		regex: "~p{IsCJKCompatibility}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0366">
	{
		regex: "~p{IsCJKUnifiedIdeographsExtensionA}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0367">
	{
		regex: "~p{IsCJKUnifiedIdeographs}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0368">
	{
		regex: "~p{IsYiSyllables}?",
		valid: false,
	},
	//    <!--<test-case name="regex-syntax-0369">
	{
		regex: "~p{IsYiRadicals}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0370">
	{
		regex: "~p{IsLowSurrogates}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0370a">
	{
		regex: "~p{IsPrivateUseArea}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0371">
	{
		regex: "~p{IsSupplementaryPrivateUseArea-B}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0372">
	{
		regex: "~p{IsCJKCompatibilityIdeographs}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0373">
	{
		regex: "~p{IsAlphabeticPresentationForms}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0374">
	{
		regex: "~p{IsArabicPresentationForms-A}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0375">
	{
		regex: "~p{IsCombiningHalfMarks}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0376">
	{
		regex: "~p{IsCJKCompatibilityForms}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0377">
	{
		regex: "~p{IsSmallFormVariants}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0378">
	{
		regex: "~p{IsSpecials}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0379">
	{
		regex: "~p{IsHalfwidthandFullwidthForms}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0380">
	{
		regex: "~p{IsOldItalic}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0381">
	{
		regex: "~p{IsGothic}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0382">
	{
		regex: "~p{IsDeseret}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0383">
	{
		regex: "~p{IsByzantineMusicalSymbols}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0384">
	{
		regex: "~p{IsMusicalSymbols}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0385">
	{
		regex: "~p{IsMathematicalAlphanumericSymbols}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0386">
	{
		regex: "~p{IsCJKUnifiedIdeographsExtensionB}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0387">
	{
		regex: "~p{IsCJKCompatibilityIdeographsSupplement}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0388">
	{
		regex: "~p{IsTags}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0389">
	{
		regex: "~p{IsBasicLatin}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0390">
	{
		regex: "~p{IsLatin-1Supplement}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0391">
	{
		regex: "~p{IsLatinExtended-A}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0392">
	{
		regex: "~p{IsLatinExtended-B}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0393">
	{
		regex: "~p{IsIPAExtensions}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0394">
	{
		regex: "~p{IsSpacingModifierLetters}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0395">
	{
		regex: "~p{IsGreekandCoptic}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0396">
	{
		regex: "~p{IsCyrillic}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0397">
	{
		regex: "~p{IsArmenian}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0398">
	{
		regex: "~p{IsHebrew}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0399">
	{
		regex: "~p{IsArabic}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0400">
	{
		regex: "~p{IsSyriac}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0401">
	{
		regex: "~p{IsThaana}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0402">
	{
		regex: "~p{IsDevanagari}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0403">
	{
		regex: "~p{IsBengali}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0404">
	{
		regex: "~p{IsGurmukhi}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0405">
	{
		regex: "~p{IsGujarati}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0406">
	{
		regex: "~p{IsOriya}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0407">
	{
		regex: "~p{IsTamil}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0408">
	{
		regex: "~p{IsTelugu}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0409">
	{
		regex: "~p{IsKannada}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0410">
	{
		regex: "~p{IsMalayalam}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0411">
	{
		regex: "~p{IsSinhala}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0412">
	{
		regex: "~p{IsThai}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0413">
	{
		regex: "~p{IsLao}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0414">
	{
		regex: "~p{IsTibetan}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0415">
	{
		regex: "~p{IsMyanmar}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0416">
	{
		regex: "~p{IsGeorgian}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0417">
	{
		regex: "~p{IsHangulJamo}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0418">
	{
		regex: "~p{IsEthiopic}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0419">
	{
		regex: "~p{IsCherokee}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0420">
	{
		regex: "~p{IsUnifiedCanadianAboriginalSyllabics}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0421">
	{
		regex: "~p{IsOgham}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0422">
	{
		regex: "~p{IsRunic}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0423">
	{
		regex: "~p{IsKhmer}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0424">
	{
		regex: "~p{IsMongolian}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0425">
	{
		regex: "~p{IsLatinExtendedAdditional}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0426">
	{
		regex: "~p{IsGreekExtended}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0427">
	{
		regex: "~p{IsGeneralPunctuation}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0428">
	{
		regex: "~p{IsSuperscriptsandSubscripts}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0429">
	{
		regex: "~p{IsCurrencySymbols}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0430">
	{
		regex: "~p{IsCombiningDiacriticalMarksforSymbols}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0431">
	{
		regex: "~p{IsLetterlikeSymbols}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0432">
	{
		regex: "~p{IsNumberForms}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0433">
	{
		regex: "~p{IsArrows}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0434">
	{
		regex: "~p{IsMathematicalOperators}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0435">
	{
		regex: "~p{IsMiscellaneousTechnical}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0436">
	{
		regex: "~p{IsControlPictures}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0437">
	{
		regex: "~p{IsOpticalCharacterRecognition}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0438">
	{
		regex: "~p{IsEnclosedAlphanumerics}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0439">
	{
		regex: "~p{IsBoxDrawing}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0440">
	{
		regex: "~p{IsBlockElements}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0441">
	{
		regex: "~p{IsGeometricShapes}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0442">
	{
		regex: "~p{IsMiscellaneousSymbols}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0443">
	{
		regex: "~p{IsDingbats}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0444">
	{
		regex: "~p{IsBraillePatterns}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0445">
	{
		regex: "~p{IsCJKRadicalsSupplement}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0446">
	{
		regex: "~p{IsKangxiRadicals}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0447">
	{
		regex: "~p{IsIdeographicDescriptionCharacters}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0448">
	{
		regex: "~p{IsCJKSymbolsandPunctuation}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0449">
	{
		regex: "~p{IsHiragana}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0450">
	{
		regex: "~p{IsKatakana}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0451">
	{
		regex: "~p{IsBopomofo}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0452">
	{
		regex: "~p{IsHangulCompatibilityJamo}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0453">
	{
		regex: "~p{IsKanbun}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0454">
	{
		regex: "~p{IsBopomofoExtended}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0455">
	{
		regex: "~p{IsEnclosedCJKLettersandMonths}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0456">
	{
		regex: "~p{IsCJKCompatibility}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0457">
	{
		regex: "~p{IsCJKUnifiedIdeographsExtensionA}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0458">
	{
		regex: "~p{IsCJKUnifiedIdeographs}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0459">
	{
		regex: "~p{IsYiSyllables}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0460">
	{
		regex: "~p{IsYiRadicals}",
		valid: false,
	},
	//    <!--<test-case name="regex-syntax-0461">
	{
		regex: "~p{IsHangulSyllables}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0462">
	{
		regex: "~p{IsHighSurrogates}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0463">
	{
		regex: "~p{IsCJKCompatibilityIdeographs}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0464">
	{
		regex: "~p{IsAlphabeticPresentationForms}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0465">
	{
		regex: "~p{IsArabicPresentationForms-A}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0466">
	{
		regex: "~p{IsCombiningHalfMarks}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0467">
	{
		regex: "~p{IsCJKCompatibilityForms}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0468">
	{
		regex: "~p{IsSmallFormVariants}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0469">
	{
		regex: "~p{IsArabicPresentationForms-B}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0470">
	{
		regex: "~p{IsSpecials}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0471">
	{
		regex: "~p{IsHalfwidthandFullwidthForms}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0472">
	{
		regex: "~p{IsOldItalic}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0473">
	{
		regex: "~p{IsGothic}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0474">
	{
		regex: "~p{IsDeseret}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0475">
	{
		regex: "~p{IsByzantineMusicalSymbols}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0476">
	{
		regex: "~p{IsMusicalSymbols}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0477">
	{
		regex: "~p{IsMathematicalAlphanumericSymbols}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0478">
	{
		regex: "~p{IsCJKUnifiedIdeographsExtensionB}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0479">
	{
		regex: "~p{IsCJKCompatibilityIdeographsSupplement}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0480">
	{
		regex: "~p{IsTags}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0481">
	{
		regex: "~p{IsSupplementaryPrivateUseArea-A}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0482">
	{
		regex:     ".",
		matches:   []string{"a", " "},
		nomatches: []string{"aa", ""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0483">
	{
		regex: "~s",
		valid: false,
	},
	//    <test-case name="regex-syntax-0484">
	{
		regex: "~s*~c~s?~c~s+~c~s*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0485">
	{
		regex: "a~s{0,3}a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0486">
	{
		regex: "a~sb",
		valid: false,
	},
	//    <test-case name="regex-syntax-0487">
	{
		regex: "~S",
		valid: false,
	},
	//    <test-case name="regex-syntax-0488">
	{
		regex: "~S+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0489">
	{
		regex: "~S*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0490">
	{
		regex: "~S?~s?~S?~s+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0491">
	{
		regex: "~i",
		valid: false,
	},
	//    <test-case name="regex-syntax-0492">
	{
		regex: "~i*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0493">
	{
		regex: "~i+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0494">
	{
		regex: "~c~i*a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0495">
	{
		regex: "[~s~i]*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0496">
	{
		regex: "~I",
		valid: false,
	},
	//    <test-case name="regex-syntax-0497">
	{
		regex: "~I*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0498">
	{
		regex: "a~I+~c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0499">
	{
		regex: "~c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0500">
	{
		regex: "~c?~?~d~s~c+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0501">
	{
		regex: "~c?~c+~c*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0502">
	{
		regex: "~C",
		valid: false,
	},
	//    <test-case name="regex-syntax-0503">
	{
		regex: "~c~C?~c~C+~c~C*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0504">
	{
		regex: "~d",
		valid: false,
	},
	//    <test-case name="regex-syntax-0505">
	{
		regex: "~D",
		valid: false,
	},
	//    <test-case name="regex-syntax-0506">
	{
		regex: "~w",
		valid: false,
	},
	//    <test-case name="regex-syntax-0507">
	{
		regex: "~W",
		valid: false,
	},
	//    <test-case name="regex-syntax-0508">
	{
		regex:     "true",
		matches:   []string{"true"},
		nomatches: []string{"false"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0509">
	{
		regex:     "false",
		matches:   []string{"false"},
		nomatches: []string{"true"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0510">
	{
		regex:     "(true|false)",
		matches:   []string{"true", "false"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0511">
	{
		regex:     "(1|true)",
		matches:   []string{"1"},
		nomatches: []string{"0"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0512">
	{
		regex:     "(1|true|false|0|0)",
		matches:   []string{"0"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0513">
	{
		regex:     "([0-1]{4}|(0|1){8})",
		matches:   []string{"1111", "11001010"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0514">
	{
		regex:     "AF01D1",
		matches:   []string{"AF01D1"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0515">
	{
		regex: "~d*~.~d+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0516">
	{
		regex: "http://~c*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0517">
	{
		regex: "[~i~c]+:[~i~c]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0518">
	{
		regex:     "P~p{Nd}{4}Y~p{Nd}{2}M",
		matches:   []string{"P1111Y12M"},
		nomatches: []string{"P111Y12M", "P1111Y1M", "P11111Y12M", "P1111Y", "P12M", "P11111Y00M", "P11111Y13M"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0519">
	{
		regex: "~p{Nd}{4}-~d~d-~d~dT~d~d:~d~d:~d~d",
		valid: false,
	},
	//    <test-case name="regex-syntax-0520">
	{
		regex: "~p{Nd}{2}:~d~d:~d~d(~-~d~d:~d~d)?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0521">
	{
		regex:     "~p{Nd}{4}-~p{Nd}{2}-~p{Nd}{2}",
		matches:   []string{"1999-12-12"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0522">
	{
		regex: "~p{Nd}{4}-~[{Nd}{2}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0523">
	{
		regex:     "~p{Nd}{4}",
		matches:   []string{"1999"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0524">
	{
		regex:     "~p{Nd}{2}",
		matches:   []string{""},
		nomatches: []string{"1999"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0525">
	{
		regex:     "--0[123]~-(12|14)",
		matches:   []string{"--03-14"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0526">
	{
		regex:     "---([123]0)|([12]?[1-9])|(31)",
		matches:   []string{"---30"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0527">
	{
		regex:     "--((0[1-9])|(1(1|2)))--",
		matches:   []string{"--12--"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0528">
	{
		regex: "~c+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0529">
	{
		regex: "~c{2,4}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0530">
	{
		regex: "[~i~c]*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0531">
	{
		regex: "~c[~c~d]*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0532">
	{
		regex:     "~p{Nd}+",
		matches:   []string{"10000101", "10000201"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0533">
	{
		regex: "~-~d~d",
		valid: false,
	},
	//    <test-case name="regex-syntax-0534">
	{
		regex: "~-?~d",
		valid: false,
	},
	//    <test-case name="regex-syntax-0535">
	{
		regex: "~d+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0536">
	{
		regex:     "~-?[0-3]{3}",
		matches:   []string{"-300"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0537">
	{
		regex:     "((~-|~+)?[1-127])|(~-?128)",
		matches:   []string{"-128"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0538">
	{
		regex: "~p{Nd}~d+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0539">
	{
		regex: "~d+~d+~d+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0540">
	{
		regex: "~d+~d+~p{Nd}~d+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0541">
	{
		regex: "~+?~d",
		valid: false,
	},
	//    <test-case name="regex-syntax-0542">
	{
		regex: "++",
		valid: false,
	},
	//    <test-case name="regex-syntax-0543">
	{
		regex:     "[0-9]*",
		matches:   []string{"9", "0"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0544">
	{
		regex:     "~-[0-9]*",
		matches:   []string{"-11111", "-9"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0545">
	{
		regex:     "[13]",
		matches:   []string{"1", "3"},
		nomatches: []string{"2"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0546">
	{
		regex:     "[123]+|[abc]+",
		matches:   []string{"112233123", "abcaabbccabc"},
		nomatches: []string{"1a", "a1"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0547">
	{
		regex:     "([abc]+)|([123]+)",
		matches:   []string{"112233123", "abcaabbccabc", "abab"},
		nomatches: []string{"1a", "1a", "x"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0548">
	{
		regex:     "[abxyz]+",
		matches:   []string{"abab"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0549">
	{
		regex: "(~p{Lu}~w*)~s(~p{Lu}~w*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0550">
	{
		regex: "(~p{Lu}~p{Ll}*)~s(~p{Lu}~p{Ll}*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0551">
	{
		regex: "(~P{Ll}~p{Ll}*)~s(~P{Ll}~p{Ll}*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0552">
	{
		regex: "(~P{Lu}+~p{Lu})~s(~P{Lu}+~p{Lu})",
		valid: false,
	},
	//    <test-case name="regex-syntax-0553">
	{
		regex: "(~p{Lt}~w*)~s(~p{Lt}*~w*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0554">
	{
		regex: "(~P{Lt}~w*)~s(~P{Lt}*~w*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0555">
	{
		regex:     "[@-D]+",
		matches:   []string{""},
		nomatches: []string{"eE?@ABCDabcdeE"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0556">
	{
		regex:     "[>-D]+",
		matches:   []string{""},
		nomatches: []string{"eE=>?@ABCDabcdeE"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0557">
	{
		regex: "[~u0554-~u0557]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0558">
	{
		regex:     "[X-~]]+",
		matches:   []string{""},
		nomatches: []string{"wWXYZxyz[~]^"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0559">
	{
		regex: "[X-~u0533]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0560">
	{
		regex:     "[X-a]+",
		matches:   []string{""},
		nomatches: []string{"wWAXYZaxyz"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0561">
	{
		regex:     "[X-c]+",
		matches:   []string{""},
		nomatches: []string{"wWABCXYZabcxyz"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0562">
	{
		regex: "[X-~u00C0]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0563">
	{
		regex: "[~u0100~u0102~u0104]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0564">
	{
		regex: "[B-D~u0130]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0565">
	{
		regex: "[~u013B~u013D~u013F]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0566">
	{
		regex:     "(Foo) (Bar)",
		matches:   []string{"Foo Bar", "Foo Bar"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0567">
	{
		regex: "~p{klsak",
		valid: false,
	},
	//    <test-case name="regex-syntax-0568">
	{
		regex: "{5",
		valid: false,
	},
	//    <test-case name="regex-syntax-0569">
	{
		regex: "{5,",
		valid: false,
	},
	//    <test-case name="regex-syntax-0570">
	{
		regex: "{5,6",
		valid: false,
	},
	//    <test-case name="regex-syntax-0571">
	{
		regex: "(?r:foo)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0572">
	{
		regex: "(?c:foo)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0573">
	{
		regex: "(?n:(foo)(~s+)(bar))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0574">
	{
		regex: "(?e:foo)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0575">
	{
		regex: "(?+i:foo)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0576">
	{
		regex: "foo([~d]*)bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0577">
	{
		regex: "([~D]*)bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0578">
	{
		regex: "foo([~s]*)bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0579">
	{
		regex: "foo([~S]*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0580">
	{
		regex: "foo([~w]*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0581">
	{
		regex: "foo([~W]*)bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0582">
	{
		regex: "([~p{Lu}]~w*)~s([~p{Lu}]~w*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0583">
	{
		regex: "([~P{Ll}][~p{Ll}]*)~s([~P{Ll}][~p{Ll}]*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0584">
	{
		regex: "foo([a-~d]*)bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0585">
	{
		regex: "([5-~D]*)bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0586">
	{
		regex: "foo([6-~s]*)bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0587">
	{
		regex: "foo([c-~S]*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0588">
	{
		regex: "foo([7-~w]*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0589">
	{
		regex: "foo([a-~W]*)bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0590">
	{
		regex: "([f-~p{Lu}]~w*)~s([~p{Lu}]~w*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0591">
	{
		regex: "([1-~P{Ll}][~p{Ll}]*)~s([~P{Ll}][~p{Ll}]*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0592">
	{
		regex: "[~p]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0593">
	{
		regex: "[~P]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0594">
	{
		regex: "([~pfoo])",
		valid: false,
	},
	//    <test-case name="regex-syntax-0595">
	{
		regex: "([~Pfoo])",
		valid: false,
	},
	//    <test-case name="regex-syntax-0596">
	{
		regex: "(~p{",
		valid: false,
	},
	//    <test-case name="regex-syntax-0597">
	{
		regex: "(~p{Ll",
		valid: false,
	},
	//    <test-case name="regex-syntax-0598">
	{
		regex: "(foo)([~x41]*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0599">
	{
		regex: "(foo)([~u0041]*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0600">
	{
		regex:     "(foo)([~r]*)(bar)",
		matches:   []string{""},
		nomatches: []string{"foo   bar"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0601">
	{
		regex: "(foo)([~o]*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0602">
	{
		regex: "(foo)~d*bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0603">
	{
		regex: "~D*(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0604">
	{
		regex: "(foo)~s*(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0605">
	{
		regex: "(foo)~S*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0606">
	{
		regex: "(foo)~w*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0607">
	{
		regex: "(foo)~W*(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0608">
	{
		regex: "~p{Lu}(~w*)~s~p{Lu}(~w*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0609">
	{
		regex: "~P{Ll}~p{Ll}*~s~P{Ll}~p{Ll}*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0610">
	{
		regex: "foo(?(?#COMMENT)foo)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0611">
	{
		regex: "foo(?(?afdfoo)bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0612">
	{
		regex: "(foo) #foo        ~s+ #followed by 1 or more whitespace        (bar)  #followed by bar        ",
		valid: false,
	},
	//    <test-case name="regex-syntax-0613">
	{
		regex: "(foo) #foo        ~s+ #followed by 1 or more whitespace        (bar)  #followed by bar",
		valid: false,
	},
	//    <test-case name="regex-syntax-0614">
	{
		regex: "(foo) (?#foo) ~s+ (?#followed by 1 or more whitespace) (bar)  (?#followed by bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0615">
	{
		regex: "(foo) (?#foo) ~s+ (?#followed by 1 or more whitespace",
		valid: false,
	},
	//    <test-case name="regex-syntax-0616">
	{
		regex: "(foo)(~077)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0617">
	{
		regex: "(foo)(~77)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0618">
	{
		regex: "(foo)(~176)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0619">
	{
		regex: "(foo)(~300)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0620">
	{
		regex: "(foo)(~477)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0621">
	{
		regex: "(foo)(~777)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0622">
	{
		regex: "(foo)(~7770)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0623">
	{
		regex: "(foo)(~7)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0624">
	{
		regex: "(foo)(~40)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0625">
	{
		regex: "(foo)(~040)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0626">
	{
		regex: "(foo)(~377)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0627">
	{
		regex: "(foo)(~400)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0628">
	{
		regex: "(foo)(~x2a*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0629">
	{
		regex: "(foo)(~x2b*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0630">
	{
		regex: "(foo)(~x2c*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0631">
	{
		regex: "(foo)(~x2d*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0632">
	{
		regex: "(foo)(~x2e*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0633">
	{
		regex: "(foo)(~x2f*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0634">
	{
		regex: "(foo)(~x2A*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0635">
	{
		regex: "(foo)(~x2B*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0636">
	{
		regex: "(foo)(~x2C*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0637">
	{
		regex: "(foo)(~x2D*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0638">
	{
		regex: "(foo)(~x2E*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0639">
	{
		regex: "(foo)(~x2F*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0640">
	{
		regex: "(foo)(~c*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0641">
	{
		regex: "(foo)~c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0642">
	{
		regex: "(foo)(~c *)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0643">
	{
		regex: "(foo)(~c?*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0644">
	{
		regex: "(foo)(~c`*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0645">
	{
		regex: "(foo)(~c~|*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0646">
	{
		regex: "(foo)(~c~[*)(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0647">
	{
		regex: "~A(foo)~s+(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0648">
	{
		regex: "(foo)~s+(bar)~Z",
		valid: false,
	},
	//    <test-case name="regex-syntax-0649">
	{
		regex: "(foo)~s+(bar)~z",
		valid: false,
	},
	//    <test-case name="regex-syntax-0650">
	{
		regex: "~b@foo",
		valid: false,
	},
	//    <test-case name="regex-syntax-0651">
	{
		regex: "~b,foo",
		valid: false,
	},
	//    <test-case name="regex-syntax-0652">
	{
		regex: "~b~[foo",
		valid: false,
	},
	//    <test-case name="regex-syntax-0653">
	{
		regex: "~B@foo",
		valid: false,
	},
	//    <test-case name="regex-syntax-0654">
	{
		regex: "~B,foo",
		valid: false,
	},
	//    <test-case name="regex-syntax-0655">
	{
		regex: "~B~[foo",
		valid: false,
	},
	//    <test-case name="regex-syntax-0656">
	{
		regex: "(~w+)~s+(~w+)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0657">
	{
		regex: "(foo~w+)~s+(bar~w+)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0658">
	{
		regex:     "([^{}]|~n)+",
		matches:   []string{""},
		nomatches: []string{"{{{{Hello  World  }END"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0659">
	{
		regex:     "(([0-9])|([a-z])|([A-Z]))*",
		matches:   []string{""},
		nomatches: []string{"{hello 1234567890 world}", "{HELLO 1234567890 world}", "{1234567890 hello  world}"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0660">
	{
		regex:     "(([0-9])|([a-z])|([A-Z]))+",
		matches:   []string{""},
		nomatches: []string{"{hello 1234567890 world}", "{HELLO 1234567890 world}", "{1234567890 hello world}"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0661">
	{
		regex:     "(([a-d]*)|([a-z]*))",
		matches:   []string{"aaabbbcccdddeeefff"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0662">
	{
		regex:     "(([d-f]*)|([c-e]*))",
		matches:   []string{"dddeeeccceee"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0663">
	{
		regex:     "(([c-e]*)|([d-f]*))",
		matches:   []string{"dddeeeccceee"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0664">
	{
		regex:     "(([a-d]*)|(.*))",
		matches:   []string{"aaabbbcccdddeeefff"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0665">
	{
		regex:     "(([d-f]*)|(.*))",
		matches:   []string{"dddeeeccceee"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0666">
	{
		regex:     "(([c-e]*)|(.*))",
		matches:   []string{"dddeeeccceee"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0667">
	{
		regex:     "CH",
		matches:   []string{""},
		nomatches: []string{"Ch", "Ch"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0668">
	{
		regex:     "cH",
		matches:   []string{""},
		nomatches: []string{"Ch", "Ch"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0669">
	{
		regex:     "AA",
		matches:   []string{""},
		nomatches: []string{"Aa", "Aa"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0670">
	{
		regex:     "aA",
		matches:   []string{""},
		nomatches: []string{"Aa", "Aa"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0671">
	{
		regex:     "ı",
		matches:   []string{""},
		nomatches: []string{"I", "I", "I", "i", "I", "i"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0672">
	{
		regex:     "İ",
		matches:   []string{""},
		nomatches: []string{"i", "i", "I", "i", "I", "i"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0673">
	{
		regex: "([0-9]+?)([~w]+?)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0674">
	{
		regex: "([0-9]+?)([a-z]+?)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0675">
	{
		regex: "^[abcd]{0,16}*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0676">
	{
		regex: "^[abcd]{1,}*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0677">
	{
		regex: "^[abcd]{1}*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0678">
	{
		regex: "^[abcd]{0,16}?*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0679">
	{
		regex: "^[abcd]{1,}?*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0680">
	{
		regex: "^[abcd]{1}?*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0681">
	{
		regex: "^[abcd]*+$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0682">
	{
		regex: "^[abcd]+*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0683">
	{
		regex: "^[abcd]?*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0684">
	{
		regex: "^[abcd]*?+$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0685">
	{
		regex: "^[abcd]+?*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0686">
	{
		regex: "^[abcd]??*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0687">
	{
		regex: "^[abcd]*{0,5}$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0688">
	{
		regex: "^[abcd]+{0,5}$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0689">
	{
		regex: "^[abcd]?{0,5}$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0690">
	{
		regex: "http://([a-zA-z0-9~-]*~.?)*?(:[0-9]*)??/",
		valid: false,
	},
	//    <test-case name="regex-syntax-0691">
	{
		regex: "http://([a-zA-Z0-9~-]*~.?)*?/",
		valid: false,
	},
	//    <test-case name="regex-syntax-0692">
	{
		regex: "([a-z]*?)([~w])",
		valid: false,
	},
	//    <test-case name="regex-syntax-0693">
	{
		regex: "([a-z]*)([~w])",
		valid: false,
	},
	//    <test-case name="regex-syntax-0694">
	{
		regex: "[abcd-[d]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0695">
	{
		regex: "[~d-[357]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0696">
	{
		regex: "[~w-[b-y]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0697">
	{
		regex: "[~w-[~d]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0698">
	{
		regex: "[~w-[~p{Ll}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0699">
	{
		regex: "[~d-[13579]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0700">
	{
		regex: "[~p{Ll}-[ae-z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0701">
	{
		regex: "[~p{Nd}-[2468]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0702">
	{
		regex: "[~P{Lu}-[ae-z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0703">
	{
		regex: "[abcd-[def]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0704">
	{
		regex: "[~d-[357a-z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0705">
	{
		regex: "[~d-[de357fgA-Z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0706">
	{
		regex: "[~d-[357~p{Ll}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0707">
	{
		regex: "[~w-[b-y~s]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0708">
	{
		regex: "[~w-[~d~p{Po}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0709">
	{
		regex: "[~w-[~p{Ll}~s]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0710">
	{
		regex: "[~d-[13579a-zA-Z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0711">
	{
		regex: "[~d-[13579abcd]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0712">
	{
		regex: "[~d-[13579~s]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0713">
	{
		regex: "[~w-[b-y~p{Po}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0714">
	{
		regex: "[~w-[b-y!.,]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0715">
	{
		regex: "[~p{Ll}-[ae-z0-9]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0716">
	{
		regex: "[~p{Nd}-[2468az]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0717">
	{
		regex: "[~P{Lu}-[ae-zA-Z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0718">
	{
		regex: "[abc-[defg]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0719">
	{
		regex: "[~d-[abc]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0720">
	{
		regex: "[~d-[a-zA-Z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0721">
	{
		regex: "[~d-[~p{Ll}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0722">
	{
		regex: "[~w-[~p{Po}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0723">
	{
		regex: "[~d-[~D]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0724">
	{
		regex: "[a-zA-Z0-9-[~s]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0725">
	{
		regex: "[~p{Ll}-[A-Z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0726">
	{
		regex: "[~p{Nd}-[a-z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0727">
	{
		regex: "[~P{Lu}-[~p{Lu}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0728">
	{
		regex: "[~P{Lu}-[A-Z]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0729">
	{
		regex: "[~P{Nd}-[~p{Nd}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0730">
	{
		regex: "[~P{Nd}-[2-8]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0731">
	{
		regex: "([ ]|[~w-[0-9]])+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0732">
	{
		regex: "([0-9-[02468]]|[0-9-[13579]])+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0733">
	{
		regex: "([^0-9-[a-zAE-Z]]|[~w-[a-zAF-Z]])+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0734">
	{
		regex: "([~p{Ll}-[aeiou]]|[^~w-[~s]])+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0735">
	{
		regex: "98[~d-[9]][~d-[8]][~d-[0]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0736">
	{
		regex: "m[~w-[^aeiou]][~w-[^aeiou]]t",
		valid: false,
	},
	//    <test-case name="regex-syntax-0737">
	{
		regex: "[abcdef-[^bce]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0738">
	{
		regex: "[^cde-[ag]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0739">
	{
		regex: "[~p{IsGreekandCoptic}-[~P{Lu}]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0740">
	{
		regex: "[a-zA-Z-[aeiouAEIOU]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0741">
	{
		regex: "[abcd~-d-[bc]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0742">
	{
		regex: "[^a-f-[~x00-~x60~u007B-~uFFFF]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0743">
	{
		regex: "[a-f-[]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0744">
	{
		regex: "[~[~]a-f-[[]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0745">
	{
		regex: "[~[~]a-f-[]]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0746">
	{
		regex: "[ab~-~[cd-[-[]]]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0747">
	{
		regex: "[ab~-~[cd-[[]]]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0748">
	{
		regex: "[a-[a-f]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0749">
	{
		regex: "[a-[c-e]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0750">
	{
		regex: "[a-d~--[bc]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0751">
	{
		regex: "[[abcd]-[bc]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0752">
	{
		regex: "[-[e-g]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0753">
	{
		regex:     "[-e-g]+",
		matches:   []string{""},
		nomatches: []string{"ddd---eeefffggghhh", "ddd---eeefffggghhh"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0754">
	{
		regex:     "[a-e - m-p]+",
		matches:   []string{""},
		nomatches: []string{"---a b c d e m n o p---"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0755">
	{
		regex: "[^-[bc]]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0756">
	{
		regex: "[A-[]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0757">
	{
		regex: "[a~-[bc]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0758">
	{
		regex: "[a~-[~-~-bc]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0759">
	{
		regex:     "[a~-~[~-~[~-bc]+",
		matches:   []string{""},
		nomatches: []string{"```bbbaaa---[[[cccddd"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0760">
	{
		regex: "[abc~--[b]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0761">
	{
		regex: "[abc~-z-[b]]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0762">
	{
		regex: "[a-d~-[b]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0763">
	{
		regex: "[abcd~-d~-[bc]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0764">
	{
		regex: "[a - c - [ b ] ]+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0765">
	{
		regex: "[a - c - [ b ] +",
		valid: false,
	},
	//    <test-case name="regex-syntax-0766">
	{
		regex: "(?<first_name>~~S+)~~s(?<last_name>~~S+)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0767">
	{
		regex: "(a+)(?:b*)(ccc)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0768">
	{
		regex: "abc(?=XXX)~w+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0769">
	{
		regex: "abc(?!XXX)~w+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0770">
	{
		regex: "[^0-9]+(?>[0-9]+)3",
		valid: false,
	},
	//    <test-case name="regex-syntax-0771">
	{
		regex:     "^aa$",
		matches:   []string{""},
		nomatches: []string{"aA"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0772">
	{
		regex:     "^Aa$",
		matches:   []string{""},
		nomatches: []string{"aA"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0773">
	{
		regex: "~s+~d+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0774">
	{
		regex: "foo~d+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0775">
	{
		regex: "foo~s+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0776">
	{
		regex: "(hello)foo~s+bar(world)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0777">
	{
		regex: "(hello)~s+(world)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0778">
	{
		regex: "(foo)~s+(bar)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0779">
	{
		regex: "(d)(o)(g)(~s)(c)(a)(t)(~s)(h)(a)(s)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0780">
	{
		regex:     "^([a-z0-9]+)@([a-z]+)~.([a-z]+)$",
		matches:   []string{""},
		nomatches: []string{"bar@bar.foo.com"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0781">
	{
		regex:     "^http://www.([a-zA-Z0-9]+)~.([a-z]+)$",
		matches:   []string{""},
		nomatches: []string{"http://www.foo.bar.com"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0782">
	{
		regex:     "(.*)",
		matches:   []string{"abc~nsfc"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0783">
	{
		regex:     "            ((.)+)      ",
		matches:   []string{""},
		nomatches: []string{"abc"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0784">
	{
		regex:     " ([^/]+)       ",
		matches:   []string{" abc       "},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0785">
	{
		regex: ".*~B(SUCCESS)~B.*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0786">
	{
		regex: "~060(~061)?~061",
		valid: false,
	},
	//    <test-case name="regex-syntax-0787">
	{
		regex: "(~x30~x31~x32)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0788">
	{
		regex: "(~u0034)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0789">
	{
		regex:     "(a+)(b*)(c?)",
		matches:   []string{""},
		nomatches: []string{"aaabbbccc"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0790">
	{
		regex: "(d+?)(e*?)(f??)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0791">
	{
		regex:     "(111|aaa)",
		matches:   []string{"aaa"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0792">
	{
		regex: "(abbc)(?(1)111|222)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0793">
	{
		regex: ".*~b(~w+)~b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0794">
	{
		regex:     "a+~.?b*~.+c{2}",
		matches:   []string{"ab.cc"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0795">
	{
		regex:     "(abra(cad)?)+",
		matches:   []string{""},
		nomatches: []string{"abracadabra1abracadabra2abracadabra3"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0796">
	{
		regex:     "^(cat|chat)",
		matches:   []string{""},
		nomatches: []string{"cats are bad"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0797">
	{
		regex:     "([0-9]+(~.[0-9]+){3})",
		matches:   []string{"209.25.0.111"},
		nomatches: []string{""},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0798">
	{
		regex:     "qqq(123)*",
		matches:   []string{""},
		nomatches: []string{"Startqqq123123End"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0799">
	{
		regex: "(~s)?(-)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0800">
	{
		regex:     "a(.)c(.)e",
		matches:   []string{""},
		nomatches: []string{"123abcde456aBCDe789"},
		valid:     true,
	},
	//    <test-case name="regex-syntax-0801">
	{
		regex: "(~S+):~W(~d+)~s(~D+)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0802">
	{
		regex: "a[b-a]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0803">
	{
		regex: "a[]b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0804">
	{
		regex: "a[",
		valid: false,
	},
	//    <test-case name="regex-syntax-0805">
	{
		regex: "a]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0806">
	{
		regex: "a[]]b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0807">
	{
		regex: "a[^]b]c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0808">
	{
		regex: "~ba~b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0809">
	{
		regex: "~by~b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0810">
	{
		regex: "~Ba~B",
		valid: false,
	},
	//    <test-case name="regex-syntax-0811">
	{
		regex: "~By~b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0812">
	{
		regex: "~by~B",
		valid: false,
	},
	//    <test-case name="regex-syntax-0813">
	{
		regex: "~By~B",
		valid: false,
	},
	//    <test-case name="regex-syntax-0814">
	{
		regex: "(*)b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0815">
	{
		regex: "a~",
		valid: false,
	},
	//    <test-case name="regex-syntax-0816">
	{
		regex: "abc)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0817">
	{
		regex: "(abc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0818">
	{
		regex: "a**",
		valid: false,
	},
	//    <test-case name="regex-syntax-0819">
	{
		regex: "a.+?c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0820">
	{
		regex: "))((",
		valid: false,
	},
	//    <test-case name="regex-syntax-0821">
	{
		regex: "~10((((((((((a))))))))))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0822">
	{
		regex: "~1(abc)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0823">
	{
		regex: "~1([a-c]*)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0824">
	{
		regex: "~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0825">
	{
		regex: "~2",
		valid: false,
	},
	//    <test-case name="regex-syntax-0826">
	{
		regex: "(a)|~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0827">
	{
		regex: "(a)|~6",
		valid: false,
	},
	//    <test-case name="regex-syntax-0828">
	{
		regex: "(~2b*?([a-c]))*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0829">
	{
		regex: "(~2b*?([a-c])){3}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0830">
	{
		regex: "(x(a)~3(~2|b))+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0831">
	{
		regex: "((a)~3(~2|b)){2,}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0832">
	{
		regex: "ab*?bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0833">
	{
		regex: "ab{0,}?bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0834">
	{
		regex: "ab+?bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0835">
	{
		regex: "ab{1,}?bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0836">
	{
		regex: "ab{1,3}?bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0837">
	{
		regex: "ab{3,4}?bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0838">
	{
		regex: "ab{4,5}?bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0839">
	{
		regex: "ab??bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0840">
	{
		regex: "ab{0,1}?bc",
		valid: false,
	},
	//    <test-case name="regex-syntax-0841">
	{
		regex: "ab??c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0842">
	{
		regex: "ab{0,1}?c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0843">
	{
		regex: "a.*?c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0844">
	{
		regex: "a.{0,5}?c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0845">
	{
		regex: "(a+|b){0,1}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0846">
	{
		regex: "(?:(?:(?:(?:(?:(?:(?:(?:(?:(a))))))))))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0847">
	{
		regex: "(?:(?:(?:(?:(?:(?:(?:(?:(?:(a|b|c))))))))))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0848">
	{
		regex: "(.)(?:b|c|d)a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0849">
	{
		regex: "(.)(?:b|c|d)*a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0850">
	{
		regex: "(.)(?:b|c|d)+?a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0851">
	{
		regex: "(.)(?:b|c|d)+a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0852">
	{
		regex: "(.)(?:b|c|d){2}a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0853">
	{
		regex: "(.)(?:b|c|d){4,5}a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0854">
	{
		regex: "(.)(?:b|c|d){4,5}?a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0855">
	{
		regex: ":(?:",
		valid: false,
	},
	//    <test-case name="regex-syntax-0856">
	{
		regex: "(.)(?:b|c|d){6,7}a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0857">
	{
		regex: "(.)(?:b|c|d){6,7}?a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0858">
	{
		regex: "(.)(?:b|c|d){5,6}a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0859">
	{
		regex: "(.)(?:b|c|d){5,6}?a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0860">
	{
		regex: "(.)(?:b|c|d){5,7}a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0861">
	{
		regex: "(.)(?:b|c|d){5,7}?a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0862">
	{
		regex: "(.)(?:b|(c|e){1,2}?|d)+?a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0863">
	{
		regex: "^(a~1?){4}$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0864">
	{
		regex: "^(a(?(1)~1)){4}$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0865">
	{
		regex: "(?:(f)(o)(o)|(b)(a)(r))*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0866">
	{
		regex: "(?:..)*a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0867">
	{
		regex: "(?:..)*?a",
		valid: false,
	},
	//    <test-case name="regex-syntax-0868">
	{
		regex: "(?:(?i)a)b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0869">
	{
		regex: "((?i)a)b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0870">
	{
		regex: "(?i:a)b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0871">
	{
		regex: "((?i:a))b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0872">
	{
		regex: "(?:(?-i)a)b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0873">
	{
		regex: "((?-i)a)b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0874">
	{
		regex: "(?-i:a)b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0875">
	{
		regex: "((?-i:a))b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0876">
	{
		regex: "((?-i:a.))b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0877">
	{
		regex: "((?s-i:a.))b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0878">
	{
		regex: "(?:c|d)(?:)(?:a(?:)(?:b)(?:b(?:))(?:b(?:)(?:b)))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0879">
	{
		regex: "(?:c|d)(?:)(?:aaaaaaaa(?:)(?:bbbbbbbb)(?:bbbbbbbb(?:))(?:bbbbbbbb(?:)(?:bbbbbbbb)))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0880">
	{
		regex: "~1~d(ab)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0881">
	{
		regex: "x(~~)*(?:(?:F)?)?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0882">
	{
		regex: "^a(?#xxx){3}c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0883">
	{
		regex: "^a (?#xxx) (?#yyy) {3}c",
		valid: false,
	},
	//    <test-case name="regex-syntax-0884">
	{
		regex: "^(?:a?b?)*$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0885">
	{
		regex: "((?s)^a(.))((?m)^b$)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0886">
	{
		regex: "((?m)^b$)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0887">
	{
		regex: "(?m)^b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0888">
	{
		regex: "(?m)^(b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0889">
	{
		regex: "((?m)^b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0890">
	{
		regex: "~n((?m)^b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0891">
	{
		regex: "((?s).)c(?!.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0892">
	{
		regex: "((?s)b.)c(?!.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0893">
	{
		regex: "((c*)(?(1)a|b))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0894">
	{
		regex: "((q*)(?(1)b|a))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0895">
	{
		regex: "(?(1)a|b)(x)?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0896">
	{
		regex: "(?(1)b|a)(x)?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0897">
	{
		regex: "(?(1)b|a)()?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0898">
	{
		regex: "(?(1)b|a)()",
		valid: false,
	},
	//    <test-case name="regex-syntax-0899">
	{
		regex: "(?(1)a|b)()?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0900">
	{
		regex: "^(?(2)(~())blah(~))?$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0901">
	{
		regex: "^(?(2)(~())blah(~)+)?$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0902">
	{
		regex: "(?(1?)a|b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0903">
	{
		regex: "(?(1)a|b|c)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0904">
	{
		regex: "(ba~2)(?=(a+?))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0905">
	{
		regex: "ba~1(?=(a+?))$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0906">
	{
		regex: "(?>a+)b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0907">
	{
		regex: "([[:]+)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0908">
	{
		regex: "([[=]+)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0909">
	{
		regex: "([[.]+)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0910">
	{
		regex: "[a[:xyz:",
		valid: false,
	},
	//    <test-case name="regex-syntax-0911">
	{
		regex: "[a[:xyz:]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0912">
	{
		regex: "([a[:xyz:]b]+)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0913">
	{
		regex: "((?>a+)b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0914">
	{
		regex: "(?>(a+))b",
		valid: false,
	},
	//    <test-case name="regex-syntax-0915">
	{
		regex: "((?>[^()]+)|~([^()]*~))+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0916">
	{
		regex: "a{37,17}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0917">
	{
		regex: "a~Z",
		valid: false,
	},
	//    <test-case name="regex-syntax-0918">
	{
		regex: "b~Z",
		valid: false,
	},
	//    <test-case name="regex-syntax-0919">
	{
		regex: "b~z",
		valid: false,
	},
	//    <test-case name="regex-syntax-0920">
	{
		regex: "round~(((?>[^()]+))~)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0921">
	{
		regex: "(a~1|(?(1)~1)){2}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0922">
	{
		regex: "(a~1|(?(1)~1)){1,2}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0923">
	{
		regex: "(a~1|(?(1)~1)){0,2}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0924">
	{
		regex: "(a~1|(?(1)~1)){2,}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0925">
	{
		regex: "(a~1|(?(1)~1)){1,2}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0926">
	{
		regex: "(a~1|(?(1)~1)){0,2}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0927">
	{
		regex: "(a~1|(?(1)~1)){2,}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0928">
	{
		regex: "~1a(~d*){0,2}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0929">
	{
		regex: "~1a(~d*){2,}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0930">
	{
		regex: "~1a(~d*){0,2}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0931">
	{
		regex: "~1a(~d*){2,}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0932">
	{
		regex: "z~1a(~d*){2,}?",
		valid: false,
	},
	//    <test-case name="regex-syntax-0933">
	{
		regex: "((((((((((a))))))))))~10",
		valid: false,
	},
	//    <test-case name="regex-syntax-0934">
	{
		regex: "(abc)~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0935">
	{
		regex: "([a-c]*)~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0936">
	{
		regex: "(([a-c])b*?~2)*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0937">
	{
		regex: "(([a-c])b*?~2){3}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0938">
	{
		regex: "((~3|b)~2(a)x)+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0939">
	{
		regex: "((~3|b)~2(a)){2,}",
		valid: false,
	},
	//    <test-case name="regex-syntax-0940">
	{
		regex: "a(?!b).",
		valid: false,
	},
	//    <test-case name="regex-syntax-0941">
	{
		regex: "a(?=d).",
		valid: false,
	},
	//    <test-case name="regex-syntax-0942">
	{
		regex: "a(?=c|d).",
		valid: false,
	},
	//    <test-case name="regex-syntax-0943">
	{
		regex: "a(?:b|c|d)(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0944">
	{
		regex: "a(?:b|c|d)*(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0945">
	{
		regex: "a(?:b|c|d)+?(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0946">
	{
		regex: "a(?:b|c|d)+(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0947">
	{
		regex: "a(?:b|c|d){2}(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0948">
	{
		regex: "a(?:b|c|d){4,5}(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0949">
	{
		regex: "a(?:b|c|d){4,5}?(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0950">
	{
		regex: "a(?:b|c|d){6,7}(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0951">
	{
		regex: "a(?:b|c|d){6,7}?(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0952">
	{
		regex: "a(?:b|c|d){5,6}(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0953">
	{
		regex: "a(?:b|c|d){5,6}?(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0954">
	{
		regex: "a(?:b|c|d){5,7}(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0955">
	{
		regex: "a(?:b|c|d){5,7}?(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0956">
	{
		regex: "a(?:b|(c|e){1,2}?|d)+?(.)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0957">
	{
		regex: "^(?:b|a(?=(.)))*~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0958">
	{
		regex: "(ab)~d~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0959">
	{
		regex: "((q*)(?(1)a|b))",
		valid: false,
	},
	//    <test-case name="regex-syntax-0960">
	{
		regex: "(x)?(?(1)a|b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0961">
	{
		regex: "(x)?(?(1)b|a)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0962">
	{
		regex: "()?(?(1)b|a)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0963">
	{
		regex: "()(?(1)b|a)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0964">
	{
		regex: "()?(?(1)a|b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0965">
	{
		regex: "^(~()?blah(?(1)(~)))$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0966">
	{
		regex: "^(~(+)?blah(?(1)(~)))$",
		valid: false,
	},
	//    <test-case name="regex-syntax-0967">
	{
		regex: "(?(?!a)a|b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0968">
	{
		regex: "(?(?!a)b|a)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0969">
	{
		regex: "(?(?=a)b|a)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0970">
	{
		regex: "(?(?=a)a|b)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0971">
	{
		regex: "(?=(a+?))(~1ab)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0972">
	{
		regex: "^(?=(a+?))~1ab",
		valid: false,
	},
	//    <test-case name="regex-syntax-0973">
	{
		regex: "(~d*){0,2}a~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0974">
	{
		regex: "(~d*){2,}a~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0975">
	{
		regex: "(~d*){0,2}?a~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0976">
	{
		regex: "(~d*){2,}?a~1",
		valid: false,
	},
	//    <test-case name="regex-syntax-0977">
	{
		regex: "(~d*){2,}?a~1z",
		valid: false,
	},
	//    <test-case name="regex-syntax-0978">
	{
		regex: "(?>~d+)3",
		valid: false,
	},
	//    <test-case name="regex-syntax-0979">
	{
		regex: "(~w(?=aa)aa)",
		valid: false,
	},
	//    <test-case name="regex-syntax-0980">
	{
		regex: "~p{IsCombiningDiacriticalMarks}+",
		valid: false,
	},
	//    <!--<test-case name="regex-syntax-0981">
	{
		regex: "~p{IsCyrillic}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0982">
	{
		regex: "~p{IsHighSurrogates}+",
		valid: false,
	},
	//    <test-case name="regex-syntax-0983">
	{
		regex: "^([0-9a-zA-Z]([-.~w]*[0-9a-zA-Z])*@(([0-9a-zA-Z])+([-~w]*[0-9a-zA-Z])*~.)+[a-zA-Z]{2,9})",
		valid: false,
	},
	//    <test-case name="regex-syntax-0984">
	{
		regex: "[~w~-~.]+@.*",
		valid: false,
	},
	//    <test-case name="regex-syntax-0985">
	{
		regex: "[~w]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0986">
	{
		regex: "[~d]",
		valid: false,
	},
	//    <test-case name="regex-syntax-0987">
	{
		regex: "[~i]",
		valid: false,
	},
}
