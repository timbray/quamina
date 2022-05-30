package core

import (
	"github.com/timbray/quamina/flattener"
	"testing"
)

const bands = `{
  "bands": [
    {
      "name": "The Clash",
      "members": [
        {
          "given": "Joe",
          "surname": "Strummer",
          "role": [
            "guitar",
            "vocals"
            ]
        },
        {
          "given": "Mick",
          "surname": "Jones",
          "role": [
            "guitar",
            "vocals"
          ]
        },
        {
          "given": "Paul",
          "surname": "Simonon",
          "role": [
            "bass"
          ]
        },
        {
          "given": "Topper",
          "surname": "Headon",
          "role": [
            "drums"
          ]
        }
      ]
    },
    {
      "name": "Boris",
      "members": [
        {
          "given": "Wata",
          "role": [
            "guitar",
            "vocals"
          ]
        },
        {
          "given": "Atsuo",
          "role": [
            "drums"
          ]
        },
        {
          "given": "Takeshi",
          "role": [
            "bass",
            "vocals"
          ]
        }
      ]
    }
  ]
}`

func TestArrayCorrectness(t *testing.T) {

	// only pattern3 should match
	pattern1 := `{"bands": { "members": { "given": [ "Mick" ], "surname": [ "Strummer" ] } } }`
	pattern2 := `{"bands": { "members": { "given": [ "Wata" ], "role": [ "drums" ] } } }`
	pattern3 := `{"bands": { "members": { "given": [ "Wata" ], "role": [ "guitar" ] } } }`
	m := NewCoreMatcher()
	err := m.AddPattern("Mick strummer", pattern1)
	if err != nil {
		t.Error(err.Error())
	}
	err = m.AddPattern("Wata drums", pattern2)
	if err != nil {
		t.Error(err.Error())
	}
	err = m.AddPattern("Wata guitar", pattern3)
	if err != nil {
		t.Error(err.Error())
	}

	fields, _ := flattener.NewFJ().Flatten([]byte(bands), m)
	matches, err := m.MatchesForFields(fields)
	if err != nil {
		t.Error(err.Error())
	}

	if len(matches) != 1 || matches[0].(string) != "Wata guitar" {
		t.Error("Matches across array boundaries")
	}
}
