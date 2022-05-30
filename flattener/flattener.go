package flattener

import "github.com/timbray/quamina/fields"

// Flattener is provided as an interface in the hope that flatterners for other non-JSON message formats might
//  be implemented.
// How it needs to work, by JSON example:
// { "a": 1, "b": "two", "c": true", "d": nil, "e": { "e1": 2, "e2":, 3.02e-5} "f": [33, "x"]} }
// should produce
// "a", "1"
// "b", "\"two\"",
// "c", "true"
// "d", "nil",
// "e\ne1", "2"
// "e\ne2", "3.02e-5"
// "f", "33"
// "f", "\"x\""
//
// Let's call the first column, eg "d" and "e\ne1", the pathSegments. For each step i the pathSegments, e.g. "d" and "e1", the
//  Flattener shold call nameTracker.IsNameUsed(step) and if that comes back negative, not include any paths which
//  don't contain that step.
// So in the example above, if nameTracker.IsNameUsed() only came back true for "a" and "f", then the output
//  would be
// "a", "1"
// "f", "33"
// "f", "\"x\""
type Flattener interface {
	Flatten(event []byte, tracker NameTracker) ([]fields.Field, error)
}
