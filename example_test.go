package quamina_test

import (
	"fmt"
	"log"

	"github.com/timbray/quamina"
)

const userRegisteredEvent = `{
  "id": "1c0e1ce4-3d88-4786-a09d-7133c170d02a",
  "type": "UserRegistered",
  "user": {
    "name": "Doe, John",
    "premiumAccount": true
  }
}
`

const premiumUserPattern = `{
  "type":["UserRegistered"],
  "user": {"premiumAccount": [true]}
}`

func ExampleNew() {
	q, err := quamina.New()
	if err != nil {
		log.Fatalf("could not create quamina instance: %v", err)
	}

	const patternName = "premium user"
	err = q.AddPattern(patternName, premiumUserPattern)
	if err != nil {
		log.Fatalf("could not add pattern: %v", err)
	}

	matches, err := q.MatchesForEvent([]byte(userRegisteredEvent))
	if err != nil {
		log.Fatalf("could not match for event: %v", err)
	}

	for _, m := range matches {
		if m == patternName {
			fmt.Printf("pattern matched for event: %q", patternName)
			return
		}
	}

	// you would typically handle no matches cases here, but in this example no
	// match is a bug, hence panic :)
	panic("no pattern match")

	// Output: pattern matched for event: "premium user"
}
