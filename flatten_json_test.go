package quamina

import (
	"bytes"
	"os"
	"testing"
)

func TestFJObjects(t *testing.T) {
	event := `{
        "Image": {
            "Width":  800,
            "Height": 600,
            "Title":  "View from 15th Floor",
            "Thumbnail": {
                "Url":    "https://www.example.com/image/481989943",
                "Height": 125,
                "Width":  100
            },
            "Animated" : false,
            "IDs": [116, 943, 234, 38793]
          }
      }`

	flattener := newJSONFlattener()

	// Verify the case on object pointers, this can happen if we get a pattern of "exists" on object
	// currently we don't support this case, and we don't pluck it.
	matcher := fakeMatcher("Image\nThumbnail")

	list, err := flattener.Flatten([]byte(event), matcher.getSegmentsTreeTracker())
	if err != nil {
		t.Errorf("Failed to flatten: %s", err)
	}

	// Make sure we got empty matches.
	expectToHavePaths(t,
		list,
		[]string{},
		[]string{},
	)

	matcher = fakeMatcher("Image\nThumbnail", "Image\nThumbnail\nUrl")

	list, err = flattener.Flatten([]byte(event), matcher.getSegmentsTreeTracker())
	if err != nil {
		t.Errorf("Failed to flatten: %s", err)
	}

	expectToHavePaths(t,
		list,
		[]string{"Image\nThumbnail\nUrl"},
		[]string{`"https://www.example.com/image/481989943"`},
	)
}

func TestFJBasic(t *testing.T) {
	j := `{ "a": 1, "b": "two", "c": true, "d": null, "e": { "e1": 2, "e2": 3.02e-5}, "f": [33e2, "x", true, false, null], "g": false, "h": [], "i": {}}`
	allYes := fakeMatcher("a", "b", "c", "d", "e\ne1", "e\ne2", "f", "g", "h")

	f := newJSONFlattener()
	list, err := f.Flatten([]byte(j), allYes.getSegmentsTreeTracker())
	if err != nil {
		t.Error("E: " + err.Error())
	}
	expectToHavePaths(t,
		list,
		[]string{"a", "b", "c", "d", "e\ne1", "e\ne2", "f", "f", "f", "f", "f", "g", "h", "i"},
		[]string{"1", "\"two\"", "true", "null", "2", "3.02e-5", "33e2", "\"x\"", "true", "false", "null", "false"},
	)

	justAF := fakeMatcher("a", "f")
	f = newJSONFlattener()
	list, _ = f.Flatten([]byte(j), justAF.getSegmentsTreeTracker())

	expectToHavePaths(t,
		list,
		[]string{"a", "f", "f", "f", "f", "f"},
		[]string{"1", "33e2", "\"x\"", "true", "false", "null"},
	)
}

func TestFJStrings(t *testing.T) {
	j := `{
		"skipped_escaped_string": "\"hello\"",
		"skipped_escaped_string_in_middle": "\"hello\" world",
		"two_escaping": "\"hello\" world \\",
		"skipped_normal_string": "abc",
		"normal_string": "abc",
		"escaped_string": "\"hello\"",
		"unicode_string": "\uD83D\ude04"
	}`
	matcher := fakeMatcher("normal_string", "escaped_string", "unicode_string")

	f := newJSONFlattener()
	list, err := f.Flatten([]byte(j), matcher.getSegmentsTreeTracker())
	if err != nil {
		t.Error("E: " + err.Error())
	}

	expectToHavePaths(t,
		list,
		[]string{"normal_string", "escaped_string", "unicode_string"},
		[]string{`"abc"`, `""hello""`, `"😄"`},
	)
}

func TestFJSkippingErrors(t *testing.T) {
	events := []string{
		// Block with strings that never ends.
		`{ "a": { "v": "hello`,
		`{ "a": ["hello`,
		// String that never ends.
		`{ "k": "`,
		// Truncated block
		`{ "k": { "a":`,
		`{ "k": {`,
		`{ "k": [1, `,
		`{ "k": [`,
	}

	matcher := fakeMatcher("non_existing_value")
	f := newJSONFlattener()

	for _, event := range events {
		fields, err := f.Flatten([]byte(event), matcher.getSegmentsTreeTracker())
		if err == nil {
			t.Errorf("Expected to fail [%s], but got %d fields", event, len(fields))
		}
	}
}

func TestFJSkippingBlocks(t *testing.T) {
	j := `{
		"skipped_objects_with_objects": {
			"num": 1,
			"str": "hello world",
			"arr": [1, "yo", { "k": "val", "arr": [1, 2, "name"] }],
			"obj": {
				"another_obj": {
					"name": "yo",
					"patterns": [{ "a": 1 }, { "b": [1, 2, 3] }, "d"]
				}
			}
		},
		"skipped_array_of_primitives": [1, 324, 534, "string"],
		"skipped_array_of_arrays": [[0, 1], ["lat", "lng"], [{ "name": "quamina" }, { "description": "patterns matching" }]],
		"requested_object": {
			"another_num": 1,
			"another_str": "hello world",
			"another_arr": [1, "yo", { "k": "val", "arr": [1, 2, "name"] }],
			"another_obj": {
				"key": "value"
			}
		},
	}`
	matcher := fakeMatcher("requested_object\nanother_obj\nkey")

	f := newJSONFlattener()
	list, err := f.Flatten([]byte(j), matcher.getSegmentsTreeTracker())
	if err != nil {
		t.Error("E: " + err.Error())
	}

	expectToHavePaths(t,
		list,
		[]string{"requested_object\nanother_obj\nkey"},
		[]string{`"value"`},
	)
}

func TestFJ10Lines(t *testing.T) {
	geo := fakeMatcher("type", "geometry\ntype")
	testTrackerSelection(t, newJSONFlattener(), geo.getSegmentsTreeTracker(), "L0", "testdata/cl-sample-0", []string{"type", "geometry\ntype"}, []string{`"Feature"`, `"Polygon"`})

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

	coords := fakeMatcher("geometry\ncoordinates")
	testTrackerSelection(t, newJSONFlattener(), coords.getSegmentsTreeTracker(), "L1", "testdata/cl-sample-1", coordNames, coordVals)

	l2names := []string{"properties\nFROM_ST", "properties\nODD_EVEN"}
	l2vals := []string{`"1917"`, `"O"`}
	proFoOd := fakeMatcher("properties\nFROM_ST", "properties\nODD_EVEN")
	testTrackerSelection(t, newJSONFlattener(), proFoOd.getSegmentsTreeTracker(), "L2", "testdata/cl-sample-2", l2names, l2vals)
}

// left here as a memorial
func TestFJMinimal(t *testing.T) {
	a := `{"a": 1}`
	nt := fakeMatcher("a")
	f := newJSONFlattener()
	fields, err := f.Flatten([]byte(a), nt.getSegmentsTreeTracker())
	if err != nil {
		t.Error("Huh? " + err.Error())
	}
	if len(fields) != 1 || !bytes.Equal(fields[0].Path, []byte("a")) || len(fields[0].Val) != 1 || fields[0].Val[0] != '1' {
		t.Error("Name/Val wrong")
	}
}

func testTrackerSelection(t *testing.T, fj Flattener, tracker SegmentsTreeTracker, label string, filename string, wantedPaths, wantedVals []string) {
	t.Helper()

	event, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("%s: failed to read file %s", filename, err.Error())
	}

	list, err := fj.Flatten(event, tracker)
	if err != nil {
		t.Fatalf("%s: failed to flatten: %s", label, err.Error())
	}
	expectToHavePaths(t, list, wantedPaths, wantedVals)
}

func TestFJErrorCases(t *testing.T) {
	// adding "a\nx" so we will go into "a" object, otherwise we will skip it
	matcher := fakeMatcher("a", "b", "c", "d", "e", "f", "a\nx")
	fj := newJSONFlattener().(*flattenJSON)

	e := ` { "a" : [1]}`
	fields, err := fj.Flatten([]byte(e), matcher.getSegmentsTreeTracker())
	if err != nil {
		t.Error("reset test: " + err.Error())
	}
	if len(fields) != 1 {
		t.Error("")
	}
	fj.reset()
	if fj.eventIndex != 0 || len(fj.fields) != 0 || fj.skipping != 0 || len(fj.arrayTrail) != 0 {
		t.Error("reset didn't work")
	}
	badUtf := "a" + string([]byte{0, 1, 2}) + "z"
	shouldFails := []string{
		`{"a`,
		`{"a"` + badUtf + `": 3}`,
		`{"a": "a\zb"}`,
		`{"a\zb": 2}`,
		`{"a": 23z}`,
		"",
		`"xx"`,
		`{"a": xx}`,
		`{"a": 1} x`,
		`{`,
		`{ "a` + string([]byte{0, 1, 2}) + `": 1}`,
		`{ r "a": 1}`,
		`{ "a" r: 1}`,
		`{ "a" :`,
		`{ "a" : `,
		`{"a" : [ foo ]}`,
		`{"a": { x }}`,
		`{"a": 2`,
		`{"a": 4 4}"`,
		`{"a": [`,
		`{"a": [  `,
		`{"a" : [ {"a": xx ]}`,
		`{"a" : [ z ]}`,
		`{"a" : [ 34r ]}`,
		`{"a" : [ 34 r ]}`,
		`{"a" : 3.3z}`,
		`{"a" : 3.3e3z}`,
		`{"a" : tru}`,
		`{"a" : tru`,
		`{"a" : truse}`,
		`{"a" : "`,
		`{"a" : "` + badUtf + `"}"`,
		`{"a" : "t`,
		`{"a": "\n` + badUtf + `"}"`,
		`{"a": "\nab`,
		`{"`,
		`{"a`,
		`{"` + badUtf + `": 1}`,
		`{"a": "\`,
		`{"a": -z}`,
		`{"a": 23ez}`,
	}
	for i, shouldFail := range shouldFails {
		_, err := fj.Flatten([]byte(shouldFail), matcher.getSegmentsTreeTracker())
		if err == nil {
			t.Errorf("Accepted bad JSON at %d: %s", i, shouldFail)
		}
	}
}

func fakeMatcher(paths ...string) *coreMatcher {
	m := newCoreMatcher()
	for _, path := range paths {
		m.fields().segmentsTree.add(path)
	}
	return m
}

func expectToHavePaths(t *testing.T, list []Field, wantedPaths, wantedVals []string) {
	t.Helper()

	if len(list) != len(wantedVals) {
		t.Errorf("got %d fields, wanted %d", len(list), len(wantedVals))
	}

	for i, field := range list {
		if !bytes.Equal([]byte(wantedPaths[i]), field.Path) {
			t.Errorf("pos %d wanted %s got %s", i, wantedPaths[i], field.Path)
		}

		if !bytes.Equal([]byte(wantedVals[i]), field.Val) {
			t.Errorf("pos %d wanted %s got %s", i, wantedVals[i], field.Val)
		}
	}
}
