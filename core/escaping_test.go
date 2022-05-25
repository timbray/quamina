package core

import (
	"testing"
)

func TestReadMemberName(t *testing.T) {
	j := `{"😀💋😺": 1, "x\u0078\ud83d\udc8by": "2"}`
	m := fakeMatcher("😀💋😺", `xx💋y`)
	f := NewFJ(m)
	fields, err := f.Flatten([]byte(j))
	if err != nil {
		t.Error("TRMN: " + err.Error())
	}
	if len(fields) != 2 {
		t.Errorf("wanted 2 fields got %d", len(fields))
	}
	if string(fields[0].Path) != "😀💋😺" || string(fields[0].Val) != "1" {
		t.Error("botched field 0")
	}
	if string(fields[1].Path) != "xx💋y" || string(fields[1].Val) != `"2"` {
		t.Error("botched field 0")
	}
}

func TestStringValuesWithEscapes(t *testing.T) {
	j := `{"a": "x\u0078\ud83d\udc8by", "b": "\ud83d\ude00\ud83d\udc8b\ud83d\ude3a"}`
	f := NewFJ(fakeMatcher("a", "b"))
	fields, err := f.Flatten([]byte(j))
	if err != nil {
		t.Error("TSVWE: " + err.Error())
	}
	if len(fields) != 2 {
		t.Errorf("wanted 2 fields got %d", len(fields))
	}
	wanted := `"xx💋y"`
	if string(fields[0].Path) != "a" || string(fields[0].Val) != wanted {
		t.Errorf("wanted %s got %s", wanted, "["+string(fields[0].Val)+"]")
	}
	if string(fields[1].Path) != "b" || string(fields[1].Val) != `"😀💋😺"` {
		t.Errorf("1 wanted %s got %s", `"😀💋😺"`, string(fields[1].Val))
	}
}

func TestOneEscape(t *testing.T) {
	tests := map[string]string{
		`\"z`:                       `"`,
		`\\z`:                       `\`,
		`\/z`:                       "/",
		`\bz`:                       string([]byte{8}),
		`\fz`:                       string([]byte{0xc}),
		`\nz`:                       "\n",
		`\rz`:                       "\r",
		`\tz`:                       "\t",
		`\u0416\ud83d\udc8b\u4e2dz`: `Ж💋中`,
	}
	for escape, wanted := range tests {
		f := &FJ{event: []byte(escape), fields: make([]Field, 0, 32)}
		unescaped, from, err := f.readTextWithEscapes(0)
		if err != nil {
			t.Errorf("for %s: %s", escape, err.Error())
		}
		if from != len(escape)-2 {
			t.Errorf("for %s from %d wanted %d", escape, from, len(escape)-2)
		}
		if string(unescaped) != wanted {
			t.Errorf("got %s wanted %s", string(unescaped), wanted)
		}

	}
}

func TestUTF16Escaping(t *testing.T) {
	str := `?*\u0066\u006f\u006f<>`
	b := []byte(str)
	f := &FJ{fields: make([]Field, 0, 32)}
	f.event = b
	f.eventIndex = 0
	chars, from, err := f.readHexUTF16(3)
	if err != nil {
		t.Error("TUE: " + err.Error())
	}
	if string(f.event[from:]) != "f<>" {
		t.Errorf("tail=%s should be f<>", string(f.event[from:]))
	}
	if string(chars) != "foo" {
		t.Errorf("Chars = '%s' wanted foo", string(chars))
	}
	str = `?*\u0066\u006f\u006f\t<>`
	b = []byte(str)
	f = &FJ{fields: make([]Field, 0, 32)}
	f.event = b
	f.eventIndex = 0
	chars, from, err = f.readHexUTF16(3)
	if err != nil {
		t.Error("TUE: " + err.Error())
	}
	if string(f.event[from:]) != "\\t<>" {
		t.Errorf("tail=%s should be \\t<>", string(f.event[from:]))
	}
	if string(chars) != "foo" {
		t.Errorf("Chars = '%s' wanted foo", string(chars))
	}

	shouldBeBad := []string{
		`!!!\uaabx27`,
		`cde\u03`,
	}
	for _, bad := range shouldBeBad {
		b = []byte(bad)
		f = &FJ{fields: make([]Field, 0, 32)}
		f.event = b
		_, _, err = f.readHexUTF16(4)
		if err == nil {
			t.Error("Missed error on " + bad)
		}
	}

	// emoji: U+1F600 d83d de00 😀 U+1F48B d83d dc8b 💋 U+1F408 d83d de3a 😺 U+4E2D 4e2d 中 U+0416 0416 Ж
	// trying to mix up various combinations of utf-16 one-codepoint and two-codepoint encodings
	emojis := []string{
		`😀💋😺`,
		`中Жy`,
		`x中Ж`,
		`x中y`,
		`x💋y`,
		`😺Ж💋`,
		`Ж💋中`,
	}
	utf16 := []string{
		`<\ud83d\ude00\ud83d\udc8b\ud83d\ude3a>`,
		`<\u4e2d\u0416\u0079>`,
		`<\u0078\u4e2d\u0416>`,
		`<\u0078\u4e2d\u0079>`,
		`<\u0078\ud83d\udc8b\u0079>`,
		`<\ud83d\ude3a\u0416\ud83d\udc8b>`,
		`<\u0416\ud83d\udc8b\u4e2d>`,
	}

	for i, emoji := range emojis {
		b = []byte(utf16[i])
		f = &FJ{fields: make([]Field, 0, 32)}
		f.event = b
		chars, from, err = f.readHexUTF16(2)
		if err != nil {
			t.Error("Ouch: '" + emoji + "': " + err.Error())
		}
		if from != len(b)-2 {
			t.Errorf("for %s wanted from %d got %d", emoji, len(b)-2, from)
		}
		if string(chars) != emoji {
			t.Errorf("wanted '%s' got '%s'", emoji, string(chars))
		}
	}
}
