package quamina

import (
	"testing"
)

func TestPatternFromJSON(t *testing.T) {
	bads := []string{
		`x`,
		`{"foo": ]`,
		`{"foo": 11 }`,
		`{"foo": "x" }`,
		`{"foo": true}`,
		`{"foo": null}`,
		`{"oof": [ ]`,
		`[33,22]`,
		`{"xxx": { }`,
		`{"xxx": [ [ 22 ] }`,
		`{"xxx": [ {"x": 1} ]`,
		`{"xxx": [ { [`,
		`{"xxx": [ { "exists": 23 } ] }`,
		`{"xxx": [ { "exists": true, "a": 3 }] }`,
		`{"xxx": [ { "exists": false, "x": ["a", 3 ] }] }`,
	}
	for _, b := range bads {
		_, _, err := patternFromJSON([]byte(b))
		if err == nil {
			t.Error("accepted bad pattern: " + b)
		}
	}

	goods := []string{
		`{"x": [ 2 ]}`,
		`{"x": [ null, true, false, "hopp", 3.072e-11] }`,
		`{"x": { "a": [27, 28], "b": { "m": [ "a", "b" ] } } }`,
		`{"x": [ {"exists": true} ] }`,
		`{"x": { "y": [ {"exists": false} ] } }`,
	}
	w1 := []*patternField{&patternField{path: "x", vals: []typedVal{typedVal{numberType, "2"}}}}
	w2 := []*patternField{&patternField{path: "x", vals: []typedVal{
		{literalType, "null"},
		{literalType, "true"},
		{literalType, "false"},
		{stringType, `"hopp"`},
		{numberType, "3.072e-11"},
	}}}
	w3 := []*patternField{
		&patternField{path: "x\na", vals: []typedVal{
			{numberType, "27"},
			{numberType, "28"},
		}},
		&patternField{path: "x\nb\nm", vals: []typedVal{
			{stringType, `"a"`},
			{stringType, `"b"`},
		}},
	}
	w4 := []*patternField{
		&patternField{path: "x", vals: []typedVal{
			{vType: existsTrueType, val: ""},
		},
		}}
	w5 := []*patternField{
		&patternField{path: "x\ny", vals: []typedVal{
			{vType: existsFalseType, val: ""},
		},
		}}
	wanted := [][]*patternField{w1, w2, w3, w4, w5}

	for i, good := range goods {
		fields, _, err := patternFromJSON([]byte(good))
		if err != nil {
			t.Error("pattern:" + good + ": " + err.Error())
		}
		w := wanted[i]
		if len(w) != len(fields) {
			t.Errorf("at %d len(w)=%d, len(fields)=%d", i, len(w), len(fields))
		}
		for j, ww := range w {
			if ww.path != fields[j].path {
				t.Error("pathSegments mismatch: " + ww.path + "/" + fields[j].path)
			}
			for k, www := range ww.vals {
				if www.val != fields[j].vals[k].val {
					t.Errorf("At [%d][%d], val mismatch %s/%s", j, k, www.val, fields[j].vals[k].val)
				}
			}
		}
	}
}
