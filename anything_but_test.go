package quamina

import (
	"strings"
	"testing"
)

func TestAnythingButMerging(t *testing.T) {
	pFoo := `{"z": [ "foo" ]}`
	pAbFoot := `{"z": [ {"anything-but": [ "foot"] } ]}`
	q, _ := New()
	var err error

	// can merge with DFA?
	err = q.AddPattern("pFoo", pFoo)
	if err != nil {
		t.Error("add pFoo")
	}
	err = q.AddPattern("pAbFoot", pAbFoot)
	if err != nil {
		t.Error("add pAbFoot: " + err.Error())
	}
	var m []X
	m, err = q.MatchesForEvent([]byte(`{"z": "foo"}`))
	if err != nil {
		t.Error("m4E - foo: " + err.Error())
	}
	if len(m) != 2 {
		t.Errorf("len=%d?!?", len(m))
	}
	m, err = q.MatchesForEvent([]byte(`{"z": "foot"}`))
	if err != nil {
		t.Error("m4E - foo: " + err.Error())
	}
	if len(m) != 0 {
		t.Errorf("len=%d?!?", len(m))
	}

	// can merge with NFA?
	pFooStar := `{"z": [ {"shellstyle": "foo*" } ]}`
	q, _ = New()
	err = q.AddPattern("pFooStar", pFooStar)
	if err != nil {
		t.Error("pFooStar: " + err.Error())
	}
	err = q.AddPattern("pAbFoot", pAbFoot)
	if err != nil {
		t.Error("add pAbFoot: " + err.Error())
	}
	m, err = q.MatchesForEvent([]byte(`{"z": "foo"}`))
	if err != nil {
		t.Error("m4E: " + err.Error())
	}
	if len(m) != 2 {
		t.Errorf("len=%d?!?", len(m))
	}
	m, err = q.MatchesForEvent([]byte(`{"z": "foot"}`))
	if err != nil {
		t.Error("m4E: " + err.Error())
	}
	if len(m) != 1 {
		t.Errorf("len=%d?!?", len(m))
	}
}

func TestAnythingButMatching(t *testing.T) {
	q, _ := New()
	// the idea is we're testing against all the 5-letter Wordle patterns, so we want a 4-letter prefix and
	// suffix of an existing wordle, a 5-letter non-wordle, and a 6-letter where the wordle might match at the start
	// and end. I tried to think of scenarios that would defeat the pretty-simple anything-but DFA but couldn't.
	problemWords := []string{
		`"bloo"`,
		`"aper"`,
		`"fnord"`,
		`"doubts"`,
		`"astern"`,
	}
	pws := strings.Join(problemWords, ",")
	pattern := `{"a": [ {"anything-but": [ ` + pws + `] } ] }"`
	err := q.AddPattern(pattern, pattern)
	if err != nil {
		t.Error("AP: " + err.Error())
	}
	words := readWWords(t)
	template := `{"a": "XX"}`
	problemTemplate := `{"a": XX}`
	for _, word := range problemWords {
		event := strings.ReplaceAll(problemTemplate, "XX", word)
		matches, err := q.MatchesForEvent([]byte(event))
		if err != nil {
			t.Error("on problem word: " + err.Error())
		}
		if len(matches) != 0 {
			t.Error("Matched on : " + word)
		}
	}
	for _, word := range words {
		ws := string(word)
		event := strings.ReplaceAll(template, "XX", ws)
		matches, err := q.MatchesForEvent([]byte(event))
		if err != nil {
			t.Error("m4E: " + err.Error())
		}
		if len(matches) != 1 {
			t.Errorf("missed on (len=%d): "+event, len(matches))
		}
	}
}

func TestParseAnythingButPattern(t *testing.T) {
	goods := []string{
		`{"a": [ {"anything-but": [ "foo" ] } ] }`,
		`{"a": [ {"anything-but": [ "bif", "x", "y", "a;sldkfjas;lkdfjs" ] } ] }`,
	}
	bads := []string{
		`{"a": [ {"anything-but": x } ] }`,
		`{"a": [ {"anything-but": 1 } ] }`,
		`{"a": [ {"anything-but": [ "a"`,
		`{"a": [ {"anything-but": [ x ] } ] }`,
		`{"a": [ {"anything-but": [ {"z": 1} ] } ] }`,
		`{"a": [ {"anything-but": [ true ] } ] }`,
		`{"a": [ {"anything-but": [ "foo" ] x`,
		`{"a": [ {"anything-but": [ "foo" ] ] ] }`,
		`{"a": [ {"anything-but": {"x":1} } ] }`,
		`{"a": [ {"anything-but": "foo" } ] }`,
		`{"a": [ 2, {"anything-but": [ "foo" ] } ] }`,
		`{"a": [ {"anything-but": [ "foo" ] }, 2 ] }`,
		`{"a": [ {"anything-but": [ ] } ] }`,
	}

	for i, good := range goods {
		fields, _, err := patternFromJSON([]byte(good))
		if err != nil {
			t.Errorf("parse anything-but i=%d: "+err.Error(), i)
		}
		if len(fields[0].vals) != 1 {
			t.Errorf("wanted11 fields got %d", len(fields))
		}
	}

	for _, bad := range bads {
		_, _, err := patternFromJSON([]byte(bad))
		if err == nil {
			t.Errorf(`accepted anything-but "%s"`, bad)
		}
	}
}
