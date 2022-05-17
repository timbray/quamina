// Package pruner supports removing a pattern (DeletePattern) from a
// matcher.
//
// The core quamina.Matcher doesn't currently support deleting
// patterns.  Some of the contemplated implementations would probably
// be pretty difficult.  At least one approach is pretty easy: Wrap
// the current matcher to filter removed patterns from match results
// and periodically rebuild the matcher from scrach with the live
// patterns.
//
// By default, rebuilding is triggered automatically (synchronously
// currently) during mutations.  The code also supports pluggable
// rebuilding policies, but those features are not currently exposed.
// A Rebuild method is available for the application to force a
// rebuild.
package pruner
