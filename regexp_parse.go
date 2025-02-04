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

func (r *regexpParse) nest() {
	r.nesting = append(r.nesting, r.tree)
	r.tree = regexpRoot{}
}

// unNest is only called after isNested
func (r *regexpParse) unNest() {
	newTree := append(r.nesting[len(r.nesting)-1], r.tree...)
	r.nesting = r.nesting[0 : len(r.nesting)-1]
	r.tree = newTree
}

func (r *regexpParse) isNested() bool {
	return len(r.nesting) > 0
}

func newRxParseState(t []byte) *regexpParse {
	return &regexpParse{
		bytes:    t,
		features: defaultRegexpFeatureChecker(),
		tree:     regexpRoot{},
	}
}

func (r *regexpParse) nextRune() (rune, error) {
	if r.index >= len(r.bytes) {
		return 0, errRegexpEOF
	}
	r.lastIndex = r.index
	c, length := utf8.DecodeRune(r.bytes[r.index:])
	if c == utf8.RuneError {
		return 0, fmt.Errorf("UTF-8 encoding error at offset %d", r.lastOffset())
	}
	r.index += length
	return c, nil
}

// require checks to see if the first rune matches the supplied argument. If it fails, it doesn't back up or
// recover or anything, on the assumption that you're giving up.
func (r *regexpParse) require(wanted rune) error {
	got, err := r.nextRune()
	if err != nil {
		return err
	}
	if got != wanted {
		return fmt.Errorf("incorrect character at %d; got %c wanted %c", r.lastOffset(), got, wanted)
	}
	return nil
}

func (r *regexpParse) bypassOptional(c rune) (bool, error) {
	next, err := r.nextRune()
	if err != nil {
		return false, err
	}
	if next != c {
		r.backup1(next)
	}
	return next == c, nil
}

func (r *regexpParse) backup1(oneRune rune) {
	r.index -= utf8.RuneLen(oneRune)
}

func (r *regexpParse) offset() int {
	return r.index
}
func (r *regexpParse) lastOffset() int {
	return r.lastIndex
}

func (r *regexpParse) isEmpty() bool {
	return r.index >= len(r.bytes)
}
