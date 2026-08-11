package quamina_test

import (
	"fmt"
	"testing"
	"unicode"

	"quamina.net/go/quamina/v2"
)

type fakeFlattener struct {
	r []quamina.Field
}

func (f *fakeFlattener) Flatten(_ []byte, _ quamina.SegmentsTreeTracker) ([]quamina.Field, error) {
	return f.r, nil
}

func (f *fakeFlattener) Copy() quamina.Flattener {
	return &fakeFlattener{r: f.r}
}

// TestNew proves we can actually call New() using With options
func TestNew(t *testing.T) {
	_, err := quamina.New(quamina.WithFlattener(&fakeFlattener{}))
	if err != nil {
		t.Error("qNew: " + err.Error())
	}
}

func TestDifferentFlattener(t *testing.T) {
	pos := quamina.ArrayPos{Array: 1, Pos: 1}
	f := quamina.Field{
		Path:       []byte{97},
		Val:        []byte{49},
		ArrayTrail: []quamina.ArrayPos{pos},
	}
	flattener := &fakeFlattener{r: []quamina.Field{f}}
	q, err := quamina.New(quamina.WithFlattener(flattener))
	if err != nil {
		t.Error("q.new: " + err.Error())
	}
	err = q.AddPattern("xyz", `{"a": [1]}`)
	if err != nil {
		t.Error("addP: " + err.Error())
	}
	matches, err := q.MatchesForEvent([]byte(`{"a": 1}`))
	if err != nil {
		t.Error("m4: " + err.Error())
	}
	if len(matches) != 1 || matches[0] != "xyz" {
		t.Error("missed!")
	}
}

func TestRegexpCharacterClassAboveU10FFF(t *testing.T) {
	const mathematicalBoldDigitZero = '\U0001D7CE'
	if !unicode.Is(unicode.Nd, mathematicalBoldDigitZero) {
		t.Fatalf("U+%04X is not a decimal digit", mathematicalBoldDigitZero)
	}

	q, err := quamina.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	patterns := []struct {
		name    string
		pattern string
	}{
		{name: "literal character class", pattern: fmt.Sprintf(`{"value": [{"regexp": "[%c]"}]}`, mathematicalBoldDigitZero)},
		{name: "decimal digit property", pattern: `{"value": [{"regexp": "~p{Nd}"}]}`},
	}
	wantMatches := make(map[string]bool, len(patterns))
	for _, pattern := range patterns {
		if err := q.AddPattern(pattern.name, pattern.pattern); err != nil {
			t.Fatalf("AddPattern(%q): %v", pattern.name, err)
		}
		wantMatches[pattern.name] = true
	}

	event := fmt.Appendf(nil, `{"value": "%c"}`, mathematicalBoldDigitZero)
	matches, err := q.MatchesForEvent(event)
	if err != nil {
		t.Fatalf("MatchesForEvent: %v", err)
	}
	for _, match := range matches {
		name, ok := match.(string)
		if !ok || !wantMatches[name] {
			t.Fatalf("got unexpected match %v", match)
		}
		delete(wantMatches, name)
	}
	if len(wantMatches) != 0 {
		t.Fatalf("missing regexp pattern matches: %v", wantMatches)
	}
}
