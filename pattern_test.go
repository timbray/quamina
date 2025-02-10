package quamina

import (
	"bytes"
	"testing"
)

func TestPatternErrorHandling(t *testing.T) {
	_, err := patternFromJSON([]byte{})
	if err == nil {
		t.Error("accepted empty pattern")
	}
	_, err = patternFromJSON([]byte("33"))
	if err == nil {
		t.Error("accepted non-object JSON text")
	}
	_, err = patternFromJSON([]byte("{"))
	if err == nil {
		t.Error("accepted stub JSON object")
	}
	_, err = patternFromJSON([]byte("{ ="))
	if err == nil {
		t.Error("accepted malformed JSON object")
	}
	_, err = patternFromJSON([]byte(`{ "foo": `))
	if err == nil {
		t.Error("accepted stub JSON object")
	}
	_, err = patternFromJSON([]byte(`{ "foo": [`))
	if err == nil {
		t.Error("accepted stub JSON array")
	}

	_, err = patternFromJSON([]byte(`{ "foo": [ { "exists" == ] }`))
	if err == nil {
		t.Error("accepted stub JSON array")
	}

	_, err = patternFromJSON([]byte(`{ "foo": [ { "exists": false . ] }`))
	if err == nil {
		t.Error("accepted stub JSON array")
	}
}

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
		`{"xxx": [ { "exists": true }, 15 ] }`,
		`{"xxx": [ { "exists": true, "a": 3 }] }`,
		`{"xxx": [ { "exists": false, "x": ["a", 3 ] }] }`,
		`{"abc": [ {"shellstyle":15} ] }`,
		`{"abc": [ {"shellstyle":"15"] ] }`,
		`{"abc": [ {"shellstyle":"15", "x", 1} ] }`,
		`{"abc": [ {"shellstyle":"a**b"}, "foo" ] }`,
		`{"abc": [ {"prefix":23}, "foo" ] }`,
		`{"abc": [ {"prefix":["a", "b"]}, "foo" ] }`,
		`{"abc": [ {"prefix": - }, "foo" ] }`,
		`{"abc": [ {"prefix":  - "a" }, "foo" ] }`,
		`{"abc": [ {"prefix":  "a" {, "foo" ] }`,
		`{"abc": [ {"equals-ignore-case":23}, "foo" ] }`,
		`{"abc": [ {"wildcard":"15", "x", 1} ] }`,
		`{"abc": [ {"wildcard":"a**b"}, "foo" ] }`,
		`{"abc": [ {"wildcard":"a\\b"}, "foo" ] }`,                                             // after JSON parsing, code sees `a/b`
		`{"abc": [ {"wildcard":"a\\"}, "foo" ] }`,                                              // after JSON parsing, code sees `a\`
		"{\"a\": [ { \"anything-but\": { \"equals-ignore-case\": [\"1\", \"2\" \"3\"] } } ] }", // missing ,
		"{\"a\": [ { \"anything-but\": { \"equals-ignore-case\": [1, 2, 3] } } ] }",            // no numbers
		"{\"a\": [ { \"anything-but\": { \"equals-ignore-case\": [\"1\", \"2\" } } ] }",        // missing ]
		"{\"a\": [ { \"anything-but\": { \"equals-ignore-case\": [\"1\", \"2\" ] } ] }",        // missing }
		"{\"a\": [ { \"equals-ignore-case\": 5 } ] }",
		"{\"a\": [ { \"equals-ignore-case\": [ \"abc\" ] } ] }",
	}
	for _, b := range bads {
		_, err := patternFromJSON([]byte(b))
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
		`{"abc": [ 3, {"shellstyle":"a*b"} ] }`,
		`{"abc": [ {"shellstyle":"a*b"}, "foo" ] }`,
		`{"abc": [ {"shellstyle":"a*b*c"} ] }`,
		`{"x": [ {"equals-ignore-case":"a*b*c"} ] }`,
		`{"abc": [ 3, {"wildcard":"a*b"} ] }`,
		`{"abc": [ {"wildcard":"a*b"}, "foo" ] }`,
		`{"abc": [ {"wildcard":"a*b*c"} ] }`,
		`{"abc": [ {"wildcard":"a*b\\*c"} ] }`,
	}
	w1 := []*patternField{{path: "x", vals: []typedVal{{vType: numberType, val: "2"}}}}
	w2 := []*patternField{{path: "x", vals: []typedVal{
		{vType: literalType, val: "null", list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
		{vType: literalType, val: "true", list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
		{vType: literalType, val: "false", list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
		{vType: stringType, val: `"hopp"`, list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
		{vType: numberType, val: "3.072e-11", list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
	}}}
	w3 := []*patternField{
		{path: "x\na", vals: []typedVal{
			{vType: numberType, val: "27", list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
			{vType: numberType, val: "28", list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
		}},
		{path: "x\nb\nm", vals: []typedVal{
			{vType: stringType, val: `"a"`, list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
			{vType: stringType, val: `"b"`, list: nil, parsedRegexp: regexpRoot{}, numericRange: nil},
		}},
	}
	w4 := []*patternField{
		{
			path: "x", vals: []typedVal{
				{vType: existsTrueType, val: ""},
			},
		},
	}
	w5 := []*patternField{
		{
			path: "x\ny", vals: []typedVal{
				{vType: existsFalseType, val: ""},
			},
		},
	}
	w6 := []*patternField{
		{
			path: "abc", vals: []typedVal{
				{vType: stringType, val: "3"},
				{vType: shellStyleType, val: `"a*b"`},
			},
		},
	}
	w7 := []*patternField{
		{
			path: "abc", vals: []typedVal{
				{vType: shellStyleType, val: `"a*b"`},
				{vType: stringType, val: `"foo"`},
			},
		},
	}
	w8 := []*patternField{
		{
			path: "abc", vals: []typedVal{
				{vType: shellStyleType, val: `"a*b*c"`},
			},
		},
	}
	w9 := []*patternField{
		{
			path: "x", vals: []typedVal{
				{vType: monocaseType, val: `"a*b*c"`},
			},
		},
	}
	w10 := []*patternField{
		{
			path: "abc", vals: []typedVal{
				{vType: stringType, val: "3"},
				{vType: wildcardType, val: `"a*b"`},
			},
		},
	}
	w11 := []*patternField{
		{
			path: "abc", vals: []typedVal{
				{vType: wildcardType, val: `"a*b"`},
				{vType: stringType, val: `"foo"`},
			},
		},
	}
	w12 := []*patternField{
		{
			path: "abc", vals: []typedVal{
				{vType: wildcardType, val: `"a*b*c"`},
			},
		},
	}
	w13 := []*patternField{
		{
			path: "abc", vals: []typedVal{
				{vType: wildcardType, val: `"a*b\*c"`},
			},
		},
	}
	wanted := [][]*patternField{w1, w2, w3, w4, w5, w6, w7, w8, w9, w10, w11, w12, w13}

	for i, good := range goods {
		fields, err := patternFromJSON([]byte(good))
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

func TestNumericRangePatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    *patternField
		wantErr bool
	}{
		{
			name:    "equals",
			pattern: `{"price": [ {"numeric": ["=", 100]} ]}`,
			want: &patternField{
				path: "price",
				vals: []typedVal{{
					vType:        numericRangeType,
					val:          "",
					list:         nil,
					parsedRegexp: regexpRoot{},
					numericRange: &Range{
						bottom:     qNumFromFloat(100),
						top:        qNumFromFloat(100),
						openBottom: false,
						openTop:    false,
					},
				}},
			},
		},
		{
			name:    "less than",
			pattern: `{"price": [ {"numeric": ["<", 100]} ]}`,
			want: &patternField{
				path: "price",
				vals: []typedVal{{
					vType:        numericRangeType,
					val:          "",
					list:         nil,
					parsedRegexp: regexpRoot{},
					numericRange: &Range{openBottom: true, openTop: true, top: qNumFromFloat(100)},
				}},
			},
		},
		{
			name:    "greater than or equal",
			pattern: `{"quantity": [ {"numeric": [">=", 10]} ]}`,
			want: &patternField{
				path: "quantity",
				vals: []typedVal{{
					vType:        numericRangeType,
					val:          "",
					list:         nil,
					parsedRegexp: regexpRoot{},
					numericRange: &Range{bottom: qNumFromFloat(10), openTop: true},
				}},
			},
		},
		{
			name:    "greater than negative",
			pattern: `{"score": [ {"numeric": [">", -5.5]} ]}`,
			want: &patternField{
				path: "score",
				vals: []typedVal{{
					vType:        numericRangeType,
					val:          "",
					list:         nil,
					parsedRegexp: regexpRoot{},
					numericRange: &Range{bottom: qNumFromFloat(-5.5), openBottom: true, openTop: true},
				}},
			},
		},
		{
			name:    "less than or equal",
			pattern: `{"rating": [ {"numeric": ["<=", 5.0]} ]}`,
			want: &patternField{
				path: "rating",
				vals: []typedVal{{
					vType:        numericRangeType,
					val:          "",
					list:         nil,
					parsedRegexp: regexpRoot{},
					numericRange: &Range{top: qNumFromFloat(5.0), openBottom: true},
				}},
			},
		},
		{
			name:    "invalid operator",
			pattern: `{"x": [ {"numeric": ["!=", 100]} ]}`,
			wantErr: true,
		},
		{
			name:    "non-numeric value",
			pattern: `{"x": [ {"numeric": ["<", "abc"]} ]}`,
			wantErr: true,
		},
		{
			name:    "missing value",
			pattern: `{"x": [ {"numeric": ["<"]} ]}`,
			wantErr: true,
		},
		{
			name:    "not an array",
			pattern: `{"x": [ {"numeric": "100"} ]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := patternFromJSON([]byte(tt.pattern))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(fields) != 1 {
				t.Fatalf("expected 1 field, got %d", len(fields))
			}

			got := fields[0]
			if got.path != tt.want.path {
				t.Errorf("path = %q, want %q", got.path, tt.want.path)
			}
			if len(got.vals) != 1 {
				t.Fatalf("expected 1 value, got %d", len(got.vals))
			}
			if got.vals[0].vType != tt.want.vals[0].vType {
				t.Errorf("vType = %v, want %v", got.vals[0].vType, tt.want.vals[0].vType)
			}
			if got.vals[0].numericRange == nil {
				t.Fatal("numericRange is nil")
			}
			// Compare range properties
			gr := got.vals[0].numericRange
			wr := tt.want.vals[0].numericRange
			if gr.openBottom != wr.openBottom || gr.openTop != wr.openTop {
				t.Errorf("range bounds openness mismatch: got %v/%v, want %v/%v",
					gr.openBottom, gr.openTop, wr.openBottom, wr.openTop)
			}
			if !bytes.Equal(gr.bottom, wr.bottom) || !bytes.Equal(gr.top, wr.top) {
				t.Errorf("range bounds mismatch: got %v/%v, want %v/%v",
					gr.bottom, gr.top, wr.bottom, wr.top)
			}
		})
	}
}
