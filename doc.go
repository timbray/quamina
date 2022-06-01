// Package quamina suports adding Patterns to Matchers and then
// presenting Events to the matcher, which will report which of
// the Patterns matched it.  Patterns and Events are both represented
// as JSON objects, although there is a provided Flattener interface
// by which structured objects in formats other than JSON can be
// matched by quamina.
package quamina
