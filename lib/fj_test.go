package quamina

import (
	"io/ioutil"
	"testing"
)

func bequal(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func TestFJBasic(t *testing.T) {
	j := `{ "a": 1, "b": "two", "c": true, "d": null, "e": { "e1": 2, "e2": 3.02e-5}, "f": [33, "x"]}`
	allYes := fakeMatcher("a", "b", "c", "d", "e", "e1", "e2", "f")

	f := NewFJ(allYes)
	list, err := f.Flatten([]byte(j))

	if err != nil {
		t.Error("E: " + err.Error())
	}
	wantedPaths := []string{"a", "b", "c", "d", "e\ne1", "e\ne2", "f", "f"}
	wantedVals := []string{"1", "\"two\"", "true", "null", "2", "3.02e-5", "33", "\"x\""}
	if len(list) != len(wantedVals) {
		t.Errorf("list len %d wanted %d", len(list), len(wantedVals))
	}
	for i, field := range list {
		if !bequal([]byte(wantedPaths[i]), field.Path) {
			t.Errorf("pos %d wanted %s got %s", i, wantedPaths[i], field.Path)
		}
		if !bequal([]byte(wantedVals[i]), field.Val) {
			t.Errorf("pos %d wanted %s got %s", i, wantedVals[i], field.Val)
		}
	}

	justAF := fakeMatcher("a", "f")
	f = NewFJ(justAF)
	list, err = f.Flatten([]byte(j))
	wantedPaths = []string{"a", "f", "f"}
	wantedVals = []string{"1", "33", "\"x\""}
	for i, field := range list {
		if !bequal([]byte(wantedPaths[i]), field.Path) {
			t.Errorf("pos %d wanted %s got %s", i, wantedPaths[i], field.Path)
		}
		if !bequal([]byte(wantedVals[i]), field.Val) {
			t.Errorf("pos %d wanted %s got %s", i, wantedVals[i], field.Val)
		}
	}
}

func TestFJ10Lines(t *testing.T) {

	geo := fakeMatcher("type", "geometry")
	testTrackerSelection(NewFJ(geo), "L0", "../test_data/cl-sample-0",
		[]string{"type", "geometry\ntype"},
		[]string{`"Feature"`, `"Polygon"`},
		t)

	coordVals := []string{
		"-122.45409388918634",
		"37.777883689479076",
		"0",
		"-122.45413030345098",
		"37.778062628581004",
		"0",
		"-122.45395950559532",
		"37.77808448801483",
		"0",
		"-122.45392309059642",
		"37.77790554887966",
		"0",
		"-122.45409388918634",
		"37.777883689479076",
		"0",
	}
	coordNames := []string{
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
		"geometry\ncoordinates",
	}

	coords := fakeMatcher("coordinates", "geometry")
	testTrackerSelection(NewFJ(coords), "L1", "../test_data/cl-sample-1",
		coordNames, coordVals, t)

	l2names := []string{"properties\nFROM_ST", "properties\nODD_EVEN"}
	l2vals := []string{`"1917"`, `"O"`}
	proFoOd := fakeMatcher("properties", "FROM_ST", "ODD_EVEN")
	testTrackerSelection(NewFJ(proFoOd), "L2", "../test_data/cl-sample-2",
		l2names, l2vals, t)
}

// left here as a memorial
func TestMinimal(t *testing.T) {
	a := `{"a": 1}`
	nt := fakeMatcher("a")
	f := NewFJ(nt)
	fields, err := f.Flatten([]byte(a))
	if err != nil {
		t.Error("Huh? " + err.Error())
	}
	if len(fields) != 1 || !bequal(fields[0].Path, []byte("a")) || len(fields[0].Val) != 1 || fields[0].Val[0] != '1' {
		t.Error("Name/Val wrong")
	}
}
func testTrackerSelection(fj Flattener, label string, filename string, wantedPaths []string, wantedVals []string, t *testing.T) {
	event, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Error(filename + ": " + err.Error())
	}

	list, err := fj.Flatten(event)
	if err != nil {
		t.Error(label + ": " + err.Error())
	}
	for i, field := range list {
		if !bequal([]byte(wantedPaths[i]), field.Path) {
			t.Errorf("pos %d wanted Path %s got %s", i, wantedPaths[i], field.Path)
		}
		if wantedVals[i] != string(field.Val) {
			t.Errorf("pos %d wanted Val %s got %s", i, wantedVals[i], field.Val)
		}
	}
}

func fakeMatcher(segs ...string) *CoreMatcher {
	m := NewCoreMatcher()
	for _, seg := range segs {
		m.start().namesUsed[seg] = true
	}
	return m
}
