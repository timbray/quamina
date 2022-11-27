package quamina

import (
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
	// only wataGuiterPattern should match
	mickStrummerPattern := `{"bands": { "members": { "given": [ "Mick" ], "surname": [ "Strummer" ] } } }`
	wataDrumsPattern := `{"bands": { "members": { "given": [ "Wata" ], "role": [ "drums" ] } } }`
	wataGuiterPattern := `{"bands": { "members": { "given": [ "Wata" ], "role": [ "guitar" ] } } }`

	m := newCoreMatcher()
	if err := m.addPattern("Mick strummer", mickStrummerPattern); err != nil {
		t.Errorf("Failed adding pattern: %s: %s", mickStrummerPattern, err)
	}

	if err := m.addPattern("Wata drums", wataDrumsPattern); err != nil {
		t.Errorf("Failed adding pattern: %s: %s", wataDrumsPattern, err)
	}
	if err := m.addPattern("Wata guitar", wataGuiterPattern); err != nil {
		t.Errorf("Failed adding pattern: %s: %s", wataGuiterPattern, err)
	}

	matches, err := m.matchesForJSONEvent([]byte(bands))
	if err != nil {
		t.Errorf("Failed 'matchesForJSONEvent': %s", err)
	}

	if len(matches) != 1 || matches[0].(string) != "Wata guitar" {
		t.Errorf("Expected to get a single of 'Wata guiter', but got %d matches: %+v", len(matches), matches)
	}
}
