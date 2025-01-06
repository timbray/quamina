package quamina

import (
	"errors"
	"testing"
	"unicode/utf8"
)

func TestBasicRunelist(t *testing.T) {
	bytes := []byte("foo")
	r := newRxParseState(bytes)
	for i, b := range bytes {
		next, err := r.nextRune()
		if err != nil {
			t.Errorf("err at %d", i)
		}
		if next != rune(b) {
			t.Errorf("mismatch at %d", i)
		}
	}
	_, err := r.nextRune()
	if !errors.Is(err, errRegexpEOF) {
		t.Error("missed EOF")
	}
	if !r.isEmpty() {
		t.Error("missed empty")
	}
}

func TestBadUTF8(t *testing.T) {
	bad := []byte{0xF8}
	ps := newRxParseState(bad)
	_, err := ps.nextRune()
	if err == nil {
		t.Error("bad UTF8")
	}
}

func TestVariablePlaneRunelist(t *testing.T) {
	runes := []rune{'&', 0x416, 0x4E2D, 0x10346}
	lengths := []int{1, 2, 3, 4}
	list := newRxParseState([]byte(string(runes)))
	read := 0
	for i := range runes {
		r, err := list.nextRune()
		read += utf8.RuneLen(r)
		if err != nil {
			t.Errorf("err at %d", i)
		}
		if r != runes[i] {
			t.Errorf("mismatch at %d", i)
		}
		if utf8.RuneLen(r) != lengths[i] {
			t.Errorf("length mismatch at %d", i)
		}
		if read != list.offset() {
			t.Errorf("wrong length at %d", i)
		}
	}
	if !list.isEmpty() {
		t.Error("Missed empty")
	}
	for i := 3; i >= 0; i-- {
		list.backup1(runes[i])
		read -= utf8.RuneLen(runes[i])
		if list.offset() != read {
			t.Errorf("wrong offset at %d", i)
		}
	}
	if list.offset() != 0 {
		t.Error("offset not 0")
	}
}

func TestRuneListRequire(t *testing.T) {
	r := newRxParseState([]byte("foo"))
	err := r.require('f')
	if err != nil {
		t.Error("require mode 1")
	}
	r = newRxParseState([]byte("foo"))
	err = r.require('É')
	if err == nil {
		t.Error("require mode 2")
	}
	r = newRxParseState([]byte("Éé"))
	err = r.require('É')
	if err != nil {
		t.Error("require mode 3")
	}
	r = newRxParseState([]byte("Éé"))
	err = r.require('é')
	if err == nil {
		t.Error("require mode 4")
	}
}

func TestRuneListBypass(t *testing.T) {
	r := newRxParseState([]byte("Éé"))
	_, err := r.bypassOptional('é')
	if err != nil {
		t.Error("bypass mode 1")
	}
	next, err := r.nextRune()
	if err != nil || next != 'É' {
		t.Error("bypass mode 2")
	}
	r = newRxParseState([]byte("Éé"))
	_, err = r.bypassOptional('x')
	if err != nil {
		t.Error("bypass mode 3")
	}
	next, err = r.nextRune()
	if err != nil || next != 'É' {
		t.Error("bypass mode 4")
	}
}
