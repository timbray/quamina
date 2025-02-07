package quamina

import (
	"fmt"
	"unicode/utf8"
)

// regexpParse represents the state of a regexp read, validate, and parse project
type regexpParse struct {
	bytes     []byte
	index     int
	lastIndex int
	nesting   []regexpRoot
	features  *regexpFeatureChecker
	tree      regexpRoot
}

func (p *regexpParse) nest() {
	p.nesting = append(p.nesting, p.tree)
	p.tree = regexpRoot{}
}

// unNest is only called after isNested. We've been building up a subtree in p.tree, so we need to
// save that subtree, pop whatever was on the nesting stack back into p.tree, and then return the
// sub tree so it can be built into a quantifiedAtom
func (p *regexpParse) unNest() regexpRoot {
	subtree := p.tree
	p.tree = p.nesting[len(p.nesting)-1]
	p.nesting = p.nesting[0 : len(p.nesting)-1]
	return subtree
}

func (p *regexpParse) isNested() bool {
	return len(p.nesting) > 0
}

func newRxParseState(t []byte) *regexpParse {
	return &regexpParse{
		bytes:    t,
		features: defaultRegexpFeatureChecker(),
		tree:     regexpRoot{},
	}
}

func (p *regexpParse) nextRune() (rune, error) {
	if p.index >= len(p.bytes) {
		return 0, errRegexpEOF
	}
	p.lastIndex = p.index
	c, length := utf8.DecodeRune(p.bytes[p.index:])
	if c == utf8.RuneError {
		return 0, fmt.Errorf("UTF-8 encoding error at offset %d", p.lastOffset())
	}
	p.index += length
	return c, nil
}

// require checks to see if the first rune matches the supplied argument. If it fails, it doesn't back up or
// recover or anything, on the assumption that you're giving up.
func (p *regexpParse) require(wanted rune) error {
	got, err := p.nextRune()
	if err != nil {
		return err
	}
	if got != wanted {
		return fmt.Errorf("incorrect character at %d; got %c wanted %c", p.lastOffset(), got, wanted)
	}
	return nil
}

func (p *regexpParse) bypassOptional(c rune) (bool, error) {
	next, err := p.nextRune()
	if err != nil {
		return false, err
	}
	if next != c {
		p.backup1(next)
	}
	return next == c, nil
}

func (p *regexpParse) backup1(oneRune rune) {
	p.index -= utf8.RuneLen(oneRune)
}

func (p *regexpParse) offset() int {
	return p.index
}
func (p *regexpParse) lastOffset() int {
	return p.lastIndex
}

func (p *regexpParse) isEmpty() bool {
	return p.index >= len(p.bytes)
}
