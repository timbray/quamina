package quamina

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"unicode/utf8"
)

// Reads a subset of regular expressions as defined in I-Regexp, RFC 9485
// At the current time, represents a subset of a subset. I-Regexp support will be
// built incrementally, adding features until full compatibility is achieved. The code
// will not allow the use of patterns containing regexps that rely on features that are
// not yet implemented.

// Note that since the regexp is composed of runes, i.e. Unicode code points, and since we use Go's built-in
// utf8.Decode()/Encode() to roundtrip between UTF-8 []bytes and code points, surrogate code points can neither
// be used in a regular expression nor will they be matched if they show up in an Event.

type regexpFeature string

const (
	rxfDot          regexpFeature = "'.' single-character matcher"
	rxfStar         regexpFeature = "'*' zero-or-more matcher"
	rxfPlus         regexpFeature = "'+' one-or-more matcher"
	rxfQM           regexpFeature = "'?' optional matcher"
	rxfRange        regexpFeature = "'{}' range matcher"
	rxfParenGroup   regexpFeature = "() parenthetized group"
	rxfProperty     regexpFeature = "~[Pp]-prefixed {}-enclosed Unicode property matcher"
	rxfClass        regexpFeature = "[]-enclosed character-class matcher"
	rxfNegatedClass regexpFeature = "[^]-enclosed negative character-class matcher"
	rxfOrBar        regexpFeature = "|-separated logical alternatives"
)

type regexpFeatureChecker struct {
	implemented map[regexpFeature]bool
	found       map[regexpFeature]bool
}

var implementedRegexpFeatures = map[regexpFeature]bool{
	rxfDot:          true,
	rxfClass:        true,
	rxfOrBar:        true,
	rxfParenGroup:   true,
	rxfQM:           true,
	rxfPlus:         true,
	rxfStar:         true,
	rxfNegatedClass: true,
}

const regexpQuantifierMax = 100 // TODO: make this into an option

const Escape rune = '~'

func runeToUTF8(r rune) ([]byte, error) {
	rl := utf8.RuneLen(r)
	if rl == -1 {
		return nil, errors.New("ill-formed UTF-8")
	}
	buf := make([]byte, rl)
	_ = utf8.EncodeRune(buf, r)
	return buf, nil
}

func readRegexpSpecial(pb *patternBuild, valsIn []typedVal) (pathVals []typedVal, err error) {
	pathVals = valsIn
	t, err := pb.jd.Token()
	if err != nil {
		return
	}

	regexpString, ok := t.(string)
	if !ok {
		err = errors.New("value for 'regexp' must be a string")
		return
	}
	val := typedVal{
		vType: regexpType,
	}
	var parse *regexpParse
	parse, err = readRegexp(regexpString)
	if err != nil {
		return
	}
	unimplemented := parse.features.foundUnimplemented()
	if len(unimplemented) != 0 {
		problem := "found unimplemented features:"
		for _, ui := range unimplemented {
			problem += " " + string(ui)
		}
		return nil, errors.New(problem)
	}

	val.parsedRegexp = parse.tree
	pathVals = append(pathVals, val)
	// has to be } or tokenizer will throw error
	_, err = pb.jd.Token()

	return
}

func defaultRegexpFeatureChecker() *regexpFeatureChecker {
	return &regexpFeatureChecker{implemented: implementedRegexpFeatures, found: make(map[regexpFeature]bool)}
}

func (fc *regexpFeatureChecker) recordFeature(feature regexpFeature) {
	fc.found[feature] = true
}

func (fc *regexpFeatureChecker) foundUnimplemented() []regexpFeature {
	var unimplemented []regexpFeature
	for feature := range fc.found {
		_, ok := fc.implemented[feature]
		if !ok {
			unimplemented = append(unimplemented, feature)
		}
	}
	return unimplemented
}

var errRegexpEOF = errors.New("end of string")
var errRegexpStuck = errors.New("unable to move forward")

// regexps are anchored by definition, i.e. behave as if they began with ^ and ended with $
// here is the grammar from the RFC 9485, I-Regexp, in IETF ABNF syntax
/*
i-regexp = branch *( "|" branch )
branch = *piece
piece = atom [ quantifier ]
quantifier = ( "*" / "+" / "?" ) / range-quantifier
range-quantifier = "{" QuantExact [ "," [ QuantExact ] ] "}"
QuantExact = 1*%x30-39 ; '0'-'9'

atom = NormalChar / charClass / ( "(" i-regexp ")" )
NormalChar = ( %x00-27 / "," / "-" / %x2F-3E ; '/'-'>'
 / %x40-5A ; '@'-'Z'
 / %x5E-7A ; '^'-'z'
 / %x7E-D7FF ; skip surrogate code points
 / %xE000-10FFFF )
charClass = "." / SingleCharEsc / charClassEsc / charClassExpr
SingleCharEsc = "\" ( %x28-2B ; '('-'+'
 / "-" / "." / "?" / %x5B-5E ; '['-'^'
 / %s"n" / %s"r" / %s"t" / %x7B-7D ; '{'-'}'
 )
charClassEsc = catEsc / complEsc
charClassExpr = "[" [ "^" ] ( "-" / CCE1 ) *CCE1 [ "-" ] "]"
CCE1 = ( CCchar [ "-" CCchar ] ) / charClassEsc
CCchar = ( %x00-2C / %x2E-5A ; '.'-'Z'
 / %x5E-D7FF ; skip surrogate code points
 / %xE000-10FFFF ) / SingleCharEsc
catEsc = %s"\p{" charProp "}"
complEsc = %s"\P{" charProp "}"
charProp = IsCategory
IsCategory = Letters / Marks / Numbers / Punctuation / Separators /
    Symbols / Others
Letters = %s"L" [ ( %s"l" / %s"m" / %s"o" / %s"t" / %s"u" ) ]
Marks = %s"M" [ ( %s"c" / %s"e" / %s"n" ) ]
Numbers = %s"N" [ ( %s"d" / %s"l" / %s"o" ) ]
Punctuation = %s"P" [ ( %x63-66 ; 'c'-'f'
 / %s"i" / %s"o" / %s"s" ) ]
Separators = %s"Z" [ ( %s"l" / %s"p" / %s"s" ) ]
Symbols = %s"S" [ ( %s"c" / %s"k" / %s"m" / %s"o" ) ]
Others = %s"C" [ ( %s"c" / %s"f" / %s"n" / %s"o" ) ]
*/

// recursive-descent starts here
func readRegexp(re string) (*regexpParse, error) {
	return readRegexpWithParse(newRxParseState([]byte(re)))
}

func readRegexpWithParse(parse *regexpParse) (*regexpParse, error) {
	return parse, readBranches(parse)
}

// branch = *piece
func readBranches(parse *regexpParse) error {
	for !parse.isEmpty() {
		branch, err := readBranch(parse)
		if (err != nil) && !errors.Is(err, errRegexpStuck) {
			return err
		}
		parse.tree = append(parse.tree, branch)
		if errors.Is(err, errRegexpEOF) {
			return nil
		}
		var b rune
		b, _ = parse.nextRune() // we already know we're not at EOF
		if b == '|' {
			parse.features.recordFeature(rxfOrBar)
			continue
		} else if b == ')' {
			parse.backup1(b)
			return nil
		}
		// no else, can't happen
	}
	return nil
}

func readBranch(parse *regexpParse) (regexpBranch, error) {
	branch := regexpBranch{}
	var err error
	for err == nil {
		var piece *quantifiedAtom
		piece, err = readPiece(parse)
		if err == nil {
			branch = append(branch, piece)
		}
	}
	if errors.Is(err, errRegexpEOF) {
		return branch, nil
	}
	return branch, err
}

// piece = atom [ quantifier ]
func readPiece(parse *regexpParse) (*quantifiedAtom, error) {
	var err error
	var nextQA *quantifiedAtom
	nextQA, err = readAtom(parse)
	if err != nil {
		return nil, err
	}
	if nextQA == nil {
		return nil, errRegexpStuck
	}

	err = readQuantifier(parse, nextQA)
	if (err == nil) || errors.Is(err, errRegexpStuck) {
		return nextQA, nil
	}
	return nil, err
}

/* handy for debugging
func atomType(qa *quantifiedAtom) string {
	if qa == nil {
		return "NIL"
	}
	if qa.dotRunes {
		return "DOT"
	}
	if qa.subtree != nil {
		return "SUB"
	}
	return "RR"
}
func dumpTree(tree regexpRoot, depth int) string {
	out := ""
	for i := 0; i < depth; i++ {
		out = out + " "
	}
	for _, branch := range tree {
		for _, qa := range branch {
			if qa.dotRunes {
				out += "."
			} else if qa.subtree != nil {
				out += dumpTree(qa.subtree, depth+1)
			} else {
				out += string([]rune{qa.runes[0].Lo}) + ".." + string([]rune{qa.runes[0].Hi})
			}
			out += " "
		}
		out += " | "
	}
	return out + "\n"
}
*/

// atom = NormalChar / charClass / ( "(" i-regexp ")" )
func readAtom(parse *regexpParse) (*quantifiedAtom, error) {
	var qa quantifiedAtom
	b, err := parse.nextRune()
	if err != nil {
		return nil, err
	}
	switch {
	case isNormalChar(b):
		qa.runes = RuneRange{RunePair{b, b}}
		qa.quantMin, qa.quantMax = 1, 1
		return &qa, nil
	case b == '.':
		// charClass = "." / SingleCharEsc / charClassEsc / charClassExpr
		parse.features.recordFeature(rxfDot)
		qa.dotRunes = true
		qa.quantMin, qa.quantMax = 1, 1
		return &qa, nil
	case b == '(':
		parse.nest()
		parse.features.recordFeature(rxfParenGroup)
		err = readBranches(parse)
		if (err != nil) && !errors.Is(err, errRegexpEOF) {
			return nil, err
		}
		err = parse.require(')')
		if err != nil {
			return nil, fmt.Errorf("missing ')' at %d", parse.lastOffset())
		}
		qa.subtree = parse.unNest()
		return &qa, nil
	case b == ')':
		if parse.isNested() {
			parse.backup1(b)
			return nil, errRegexpStuck
		} else {
			return nil, fmt.Errorf("unbalanced ')' at %d", parse.lastOffset())
		}
	case b == '[':
		parse.features.recordFeature(rxfClass)
		qa.runes, err = readCharClassExpr(parse)
		if err != nil {
			return nil, err
		}
		qa.quantMin, qa.quantMax = 1, 1
		return &qa, nil
	case b == ']':
		return nil, fmt.Errorf("invalid ']' at %d", parse.lastOffset())
	case b == Escape:
		c, err := parse.nextRune()
		if errors.Is(err, errRegexpEOF) {
			return nil, errors.New("'~' at end of regular expression")
		}
		if err != nil {
			return nil, err
		}
		escaped, ok := checkSingleCharEscape(c)
		if ok {
			qa.runes = RuneRange{RunePair{escaped, escaped}}
			return &qa, nil
		}
		if c == 'p' || c == 'P' {
			// QA not implemented yet
			parse.features.recordFeature(rxfProperty)
			return &quantifiedAtom{}, readCategory(parse)
		}
		if bytes.ContainsRune([]byte("sSiIcCdDwW"), c) {
			return nil, fmt.Errorf("multiple-character escape ~%c at %d", c, parse.lastOffset())
		}
		return nil, fmt.Errorf("invalid character '%c' after '~' at %d", c, parse.lastOffset())

	case bytes.ContainsRune([]byte("?+*{"), b):
		return nil, fmt.Errorf("invalid character '%c' at %d", b, parse.lastOffset())

	default:
		parse.backup1(b)
		return nil, errRegexpStuck
	}
}

// charClassExpr = "[" [ "^" ] ( "-" / CCE1 ) *CCE1 [ "-" ] "]"

func readCharClassExpr(parse *regexpParse) (RuneRange, error) {
	// starting after the "["
	var err error
	isNegated, err := parse.bypassOptional('^')
	if errors.Is(err, errRegexpEOF) {
		err = errors.New("empty character class []")
	}
	if err != nil {
		return nil, err
	}
	rr, err := readCCE1s(parse)
	if err != nil {
		return nil, err
	}
	trailingHyphen, _ := parse.bypassOptional('-') // already probed
	if trailingHyphen {
		rr = append(rr, RunePair{Lo: '-', Hi: '-'})
	}
	if err = parse.require(']'); err != nil {
		return nil, err
	}
	if isNegated {
		parse.features.recordFeature(rxfNegatedClass)
		rr = invertRuneRange(rr)
	}
	return rr, nil
}

func invertRuneRange(rr RuneRange) RuneRange {
	sort.Slice(rr, func(i, j int) bool {
		return rr[i].Lo < rr[j].Lo
	})
	var inverted RuneRange
	var point rune = 0
	for _, pair := range rr {
		if pair.Lo > point {
			inverted = append(inverted, RunePair{point, pair.Lo - 1})
		}
		point = pair.Hi + 1
	}
	if point < runeMax {
		inverted = append(inverted, RunePair{point, runeMax})
	}
	return inverted
}

// readCCE1s proceeds forward until the next chunk is not a CCE1
func readCCE1s(parse *regexpParse) (RuneRange, error) {
	var rr RuneRange
	first := true
	for {
		cce1, err := readCCE1(parse, first)
		if err != nil {
			return nil, err
		}
		rr = append(rr, cce1...)
		first = false
		r, _ := parse.nextRune() // already probed
		parse.backup1(r)
		if r == '-' || r == ']' {
			return simplifyRuneRange(rr), nil
		}
	}
}

func simplifyRuneRange(rranges RuneRange) RuneRange {
	if len(rranges) == 0 {
		return rranges
	}
	sort.Slice(rranges, func(i, j int) bool { return rranges[i].Lo < rranges[j].Lo })
	var out RuneRange
	currentPair := rranges[0]
	for i := 1; i < len(rranges); i++ {
		nextPair := rranges[i]
		if nextPair.Lo > currentPair.Hi+1 {
			out = append(out, currentPair)
			currentPair = nextPair
			continue
		}
		if nextPair.Hi <= currentPair.Hi {
			continue
		}
		currentPair.Hi = nextPair.Hi
	}
	out = append(out, currentPair)
	return out
}

// CCE1 = ( CCchar [ "-" CCchar ] ) / charClassEsc
// CCchar = ( %x00-2C / %x2E-5A ; '.'-'Z'
// / %x5E-D7FF ; skip surrogate code points
// / %xE000-10FFFF ) / SingleCharEsc

func isCCchar(r rune) bool {
	if r <= 0x2c || (r >= 0x2e && r <= 0x5A) {
		return true
	}
	if r >= 0x5e && r <= 0xd7ff {
		return true
	}
	if r >= 0xe000 && r <= 0x10fff {
		return true
	}
	if r == '\\' {
		// weird but true
		return true
	}
	return false
}

// CCE1 = ( CCchar [ "-" CCchar ] ) / charClassEsc
// CCchar = ( %x00-2C / %x2E-5A ; '.'-'Z'
// / %x5E-D7FF ; skip surrogate code points
// / %xE000-10FFFF ) / SingleCharEsc

// readCCE1 reads one instance of CCE1 token
func readCCE1(parse *regexpParse, first bool) (RuneRange, error) {
	// starts after [
	var rr RuneRange
	var err error
	r, _ := parse.nextRune() // have already probed, can't fail

	var lo rune
	if first && r == '-' {
		return RuneRange{RunePair{'-', '-'}}, nil
	} else if r == Escape {
		r, _ = parse.nextRune() // have already probed
		if r == 'p' || r == 'P' {
			// maybe a good category, in which case we can't participate in range, so we're done
			// or a malformed category
			parse.features.recordFeature(rxfProperty)
			return rr, readCategory(parse)
		}
		escaped, ok := checkSingleCharEscape(r)
		if !ok {
			return nil, fmt.Errorf("invalid character '%c' after ~ at %d", r, parse.lastOffset())
		}
		lo = escaped
		// we've seen a single-character escape
	} else {
		if !isCCchar(r) {
			return nil, fmt.Errorf("invalid character '%c' after [ at %d", r, parse.lastOffset())
		}
		lo = r
	}
	// either a regular character or a single-char escape, either we're done or we're looking for '-'
	r, err = parse.nextRune()
	if err != nil {
		return nil, fmt.Errorf("error in range at %d", parse.lastOffset())
	}
	if r != '-' {
		// not a range, so probably looking for the next cce1
		parse.backup1(r)
		return RuneRange{RunePair{lo, lo}}, nil
	}
	// looking at a range?
	r, err = parse.nextRune()
	if err != nil {
		return nil, err
	}
	// might be end of range -] which is legal. Otherwise, has to be either a CChar or single-char escape
	if r == ']' {
		parse.backup1(r)
		return RuneRange{RunePair{lo, lo}, {'-', '-'}}, nil
	}
	if r == Escape {
		r, err = parse.nextRune()
		if err != nil {
			return nil, err
		}
		escaped, ok := checkSingleCharEscape(r)
		if !ok {
			return nil, fmt.Errorf("invalid char '%c' after - at %d", r, parse.lastOffset())
		}
		if lo > escaped {
			return nil, fmt.Errorf("invalid range %c-%c", lo, r)
		}
		return RuneRange{RunePair{lo, escaped}}, nil
	}
	if !isCCchar(r) {
		return nil, fmt.Errorf("invalid char '%c' after - at %d", r, parse.lastOffset())
	}
	if lo > r {
		return nil, fmt.Errorf("invalid range %c-%c", lo, r)
	}
	return RuneRange{RunePair{lo, r}}, nil
}

// catEsc = %s"\p{" charProp "}"
// complEsc = %s"\P{" charProp "}"
// charProp = IsCategory
// IsCategory = Letters / Marks / Numbers / Punctuation / Separators /
// Symbols / Others
// Letters = %s"L" [ ( %s"l" / %s"m" / %s"o" / %s"t" / %s"u" ) ]
// Marks = %s"M" [ ( %s"c" / %s"e" / %s"n" ) ]
// Numbers = %s"N" [ ( %s"d" / %s"l" / %s"o" ) ]
// Punctuation = %s"P" [ ( %x63-66 ; 'c'-'f'
// / %s"i" / %s"o" / %s"s" ) ]
// Separators = %s"Z" [ ( %s"l" / %s"p" / %s"s" ) ]
// Symbols = %s"S" [ ( %s"c" / %s"k" / %s"m" / %s"o" ) ]
// Others = %s"C" [ ( %s"c" / %s"f" / %s"n" / %s"o" ) ]

var regexpCatDetails = map[rune]string{
	'L': "ultmo",
	'M': "nce",
	'N': "dlo",
	'P': "cdseifo",
	'Z': "slp",
	'S': "mcko",
	'C': "cfon",
}

func readCategory(parse *regexpParse) error {
	var err error
	if err = parse.require('{'); err != nil {
		return err
	}
	categoryInitial, err := parse.nextRune()
	if err != nil {
		return err
	}
	categoryDetail, ok := regexpCatDetails[categoryInitial]
	if !ok {
		return fmt.Errorf("unknown category %c at %d", categoryInitial, parse.lastOffset())
	}
	catDetailLetter, err := parse.nextRune()
	if err != nil {
		return fmt.Errorf("error in category after {%c at %d", categoryInitial, parse.lastOffset())
	}
	if catDetailLetter == '}' {
		return nil
	}
	if !bytes.ContainsRune([]byte(categoryDetail), catDetailLetter) {
		return fmt.Errorf("unknown category ~P{%c%c} at %d", categoryInitial, catDetailLetter, parse.lastOffset())
	}
	if err = parse.require('}'); err != nil {
		return err
	}
	return nil
}

func readQuantifier(parse *regexpParse, qa *quantifiedAtom) error {
	// quantifier = ( "*" / "+" / "?" ) / range-quantifier
	// range-quantifier = "{" QuantExact [ "," [ QuantExact ] ] "}"
	// QuantExact = 1*%x30-39 ; '0'-'9'
	b, err := parse.nextRune()
	if errors.Is(err, errRegexpEOF) {
		qa.quantMin, qa.quantMax = 1, 1
		return nil
	}
	if err != nil {
		return err
	}
	switch b {
	case '*':
		parse.features.recordFeature(rxfStar)
		qa.quantMin, qa.quantMax = 0, regexpQuantifierMax
		return nil
	case '+':
		parse.features.recordFeature(rxfPlus)
		qa.quantMin, qa.quantMax = 1, regexpQuantifierMax
		return nil
	case '?':
		parse.features.recordFeature(rxfQM)
		qa.quantMin, qa.quantMax = 0, 1
		return nil
	case '{':
		parse.features.recordFeature(rxfRange)
		return readRangeQuantifier(parse, qa)
	}
	qa.quantMin, qa.quantMax = 1, 1
	parse.backup1(b)
	return errRegexpStuck
}

func readRangeQuantifier(parse *regexpParse, qa *quantifiedAtom) error {
	// after {
	var loDigits []rune
	b, err := parse.nextRune()
	if err != nil {
		return err
	}
	for b >= '0' && b <= '9' {
		loDigits = append(loDigits, b)
		b, err = parse.nextRune()
		if err != nil {
			return err
		}
	}
	if len(loDigits) == 0 {
		return fmt.Errorf("invalid range quantifier, expecting digits at %d", parse.lastOffset())
	}
	// have read some digits
	lo, err := strconv.ParseInt(string(loDigits), 10, 32)
	if err != nil {
		return err
	}
	qa.quantMin = int(lo)
	qa.quantMax = regexpQuantifierMax
	switch b {
	case '}':
		return nil
	case ',':
	// no-op, good
	default:
		return fmt.Errorf("unexpected character %c at %d", b, parse.lastOffset())
	}
	// have seen digits and a comma
	var hiDigits []rune
	b, err = parse.nextRune()
	if errors.Is(err, errRegexpEOF) {
		return fmt.Errorf("incomplete range quantifier at %d", parse.lastOffset())
	}
	if err != nil {
		return err
	}
	if b == '}' {
		return nil
	}
	if b < '0' || b > '9' {
		return fmt.Errorf("invalid character '%c' in quantifier range at %d, wanted a digit", b, parse.lastOffset())
	}
	for b >= '0' && b <= '9' {
		hiDigits = append(hiDigits, b)
		b, err = parse.nextRune()
		if errors.Is(err, errRegexpEOF) {
			return fmt.Errorf("incomplete range quantifier at %d", parse.lastOffset())
		}
		if err != nil {
			return err
		}
	}
	// have scanned digits, have to close with '}'
	if b != '}' {
		return fmt.Errorf("invalid character %c at %d, expected '}'", b, parse.lastOffset())
	}
	hi, err := strconv.ParseInt(string(hiDigits), 10, 32)
	if err != nil {
		return err
	}
	if hi < lo {
		return fmt.Errorf("invalid range quantifier, top must be greater than bottom")
	}
	qa.quantMax = int(hi)
	return nil
}

// isNormalChar - not optimized, implemented line-by-line from the production for clarity
func isNormalChar(c rune) bool {
	if c <= 0x27 || c == ',' || c == '-' || (c >= 0x2F && c <= 0x3E) {
		return true
	}
	if c >= 0x40 && c <= 0x5A {
		return true
	}
	// allow \
	if c == 0x5c {
		return true
	}
	if c >= 0x5E && c <= 0x7A {
		return true
	}
	// exclude ~
	if c >= 0x7F && c <= 0xD7FF {
		return true
	}
	if c >= 0xE000 && c <= 0x10FFFF {
		return true
	}
	return false
}

// checkSingleCharEscape - things that need escaping
// SingleCharEsc = "\" ( %x28-2B ; '('-'+'
// / "-" / "." / "?" / %x5B-5E ; '['-'^'
// / %s"n" / %s"r" / %s"t" / %x7B-7D ; '{'-'}'
// )
func checkSingleCharEscape(c rune) (rune, bool) {
	if c >= 0x28 && c <= 0x2B {
		return c, true
	}
	if c == '-' || c == '.' || c == '?' || (c >= 0x5B && c <= 0x5E) {
		return c, true
	}
	if c == 'n' {
		return '\n', true
	}
	if c == 'r' {
		return '\r', true
	}
	if c == 't' {
		return '\t', true
	}
	if c >= 0x7B && c <= 0x7D {
		return c, true
	}
	if c == Escape {
		return Escape, true
	}
	return 0, false
}
