// Package quamina instances support adding Patterns and then
// presenting Events, generating a report of which Patterns
// match the Event.  Patterns and Events are both represented
// as JSON objects, although there is a provided Flattener interface
// by which structured objects in formats other than JSON can be
// matched by quamina. Quamina instances match Events quickly and
// with a latency that is not strongly affected by the number of
// Patterns which have been added.
package quamina
