package quamina_test

import (
	"testing"

	"quamina.net/go/quamina"
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
