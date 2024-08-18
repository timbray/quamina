package quamina

import (
	"fmt"
	"os"
	"testing"
)

/* This test adopted, with thanks, from aws/event-ruler */

func TestRulerArraysBug(t *testing.T) {
	event := "{\n" +
		"  \"requestContext\": { \"obfuscatedCustomerId\": \"AIDACKCEVSQ6C2EXAMPLE\" },\n" +
		"  \"hypotheses\": [\n" +
		"    { \"isBluePrint\": true, \"creator\": \"A123\" },\n" +
		"    { \"isBluePrint\": false, \"creator\": \"A234\" }\n" +
		"  ]\n" +
		"}"
	r1 := "{\n" +
		"  \"hypotheses\": {\n" +
		"    \"isBluePrint\": [ false ],\n" +
		"    \"creator\": [ \"A123\" ]\n" +
		"  }\n" +
		"}"
	r2 := "{\n" +
		"  \"hypotheses\": {\n" +
		"    \"isBluePrint\": [ true ],\n" +
		"    \"creator\": [ \"A234\" ]\n" +
		"  }\n" +
		"}"

	q, _ := New()
	err := q.AddPattern("r1", r1)
	if err != nil {
		t.Error("add r1")
	}
	err = q.AddPattern("r2", r2)
	if err != nil {
		t.Error("add r2")
	}
	matches, err := q.MatchesForEvent([]byte(event))
	if err != nil {
		t.Errorf("MatchesForEvent: %s", err)
	}
	if len(matches) != 0 {
		t.Error("Nonzero matches")
	}
}

func readTestData(t *testing.T, fname string) []byte {
	t.Helper()
	bytes, err := os.ReadFile("testdata/" + fname)
	if err != nil {
		t.Error("couldn't read: " + fname + ": " + err.Error())
	}
	return bytes
}

func TestRulerNestedArrays(t *testing.T) {
	event1 := readTestData(t, "arrayEvent1.json")
	event2 := readTestData(t, "arrayEvent2.json")
	event3 := readTestData(t, "arrayEvent3.json")
	event4 := readTestData(t, "arrayEvent4.json")

	rule1 := string(readTestData(t, "arrayRule1.json"))
	rule2 := string(readTestData(t, "arrayRule2.json"))
	rule3 := string(readTestData(t, "arrayRule3.json"))

	q, _ := New()
	for i, rule := range []string{rule1, rule2, rule3} {
		err := q.AddPattern(fmt.Sprintf("rule%d", i+1), rule)
		if err != nil {
			t.Errorf("add rule%d", i)
		}
	}
	r1, err := q.MatchesForEvent(event1)
	if err != nil {
		t.Error("Matches " + err.Error())
	}
	if len(r1) != 2 {
		t.Errorf("r1 len %d", len(r1))
	}

	r2, err := q.MatchesForEvent(event2)
	if err != nil {
		t.Errorf("Matches " + err.Error())
	}
	if len(r2) != 0 {
		t.Errorf("r2 matchd %d", len(r2))
	}

	r3, err := q.MatchesForEvent(event3)
	if err != nil {
		t.Error("Matches " + err.Error())
	}
	if len(r3) != 0 {
		t.Errorf("r3 matchd %d", len(r2))
	}

	r4, err := q.MatchesForEvent(event4)
	if err != nil {
		t.Errorf("Matches " + err.Error())
	}
	if len(r4) != 1 || r4[0] != "rule3" {
		var msg string
		if len(r4) == 1 {
			msg += "match: " + r4[0].(string)
		} else {
			msg = fmt.Sprintf("r4 matches %d", len(r4))
		}
		t.Error(msg)
	}
}

func TestRulerSimplestPossibleMachine(t *testing.T) {
	rule1 := "{ \"a\" : [ 1 ] }"
	rule2 := "{ \"b\" : [ 2 ] }"
	rule3 := "{ \"c\" : [ 3 ] }"

	q, _ := New()
	_ = q.AddPattern("r1", rule1)
	_ = q.AddPattern("r2", rule2)
	_ = q.AddPattern("r3", rule3)

	event1 := "{ \"a\" :  1 }"
	event2 := "{ \"b\" :  2 }"
	event4 := "{ \"x\" :  true }"
	event5 := "{ \"a\" :  1, \"b\": 2, \"c\" : 3 }"

	var val []X
	var err error
	val, err = q.MatchesForEvent([]byte(event1))
	if err != nil {
		t.Error("e1: " + err.Error())
	}
	if len(val) != 1 || val[0] != "r1" {
		t.Error("event1 fail")
	}

	val, err = q.MatchesForEvent([]byte(event2))
	if err != nil {
		t.Error("e2: " + err.Error())
	}
	if len(val) != 1 || val[0] != "r2" {
		t.Error("event2 fail")
	}

	val, err = q.MatchesForEvent([]byte(event4))
	if err != nil {
		t.Error("e2: " + err.Error())
	}
	if len(val) != 0 {
		t.Error("event4 fail")
	}

	val, err = q.MatchesForEvent([]byte(event5))
	if err != nil {
		t.Error("e2: " + err.Error())
	}
	if len(val) != 3 {
		t.Error("event4 fail")
	}
	matched := 0
	for _, v := range val {
		if v == "r1" || v == "r2" || v == "r3" {
			matched++
		}
	}
	if matched != 3 {
		t.Error("missing match")
	}
}

func TestRulerEmptyInput(t *testing.T) {
	rule1 := `{
  "detail": {
    "c-count": [
      {
        "exists": false
      }
    ]
  },
  "d-count": [
    {
      "exists": false
    }
  ],
  "e-count": [
    {
      "exists": false
    }
  ]
}`
	event := "{}"
	q, _ := New()
	err := q.AddPattern("r", rule1)
	if err != nil {
		t.Error("Empty input add pattern" + err.Error())
	}
	matches, err := q.MatchesForEvent([]byte(event))
	if err != nil {
		t.Error("Empty input matches: " + err.Error())
	}
	if len(matches) != 1 || matches[0] != "r" {
		t.Error("Empty input match botch")
	}
}
