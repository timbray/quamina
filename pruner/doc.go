package pruner

// Support for removing a pattern from a matcher
//
// The core quamina.Matcher doesn't currently support deleting
// patterns.  Some of the contemplated implementations of would
// probably be pretty difficult.  At least one approach is pretty
// easy: Wrap the current matcher to filter removed patterns from
// match results and periodically rebuild the matcher from scrach with
// the live patterns.  More specifically:
//
// 1.  Remember patterns that have been added
// 2. Remember patterns that have been removed (implicitly)
// 3. Filter MatchedFor...() results to remove any removed patterns
// 4. Support rebuilding the matcher state periodically with only the
//    live patterns.
// 5. Maintain some statistics to help decide when to rebuild
//
// The implementation of the set of live patterns is pluggable via a
// Go interface (State).  The default implementation is a
// `map[quamina.X]string (MemState)`.  Other implementations can
// provide persistence.
//
// By default, rebuilding is triggered automatically (synchronously
// currently) during mutations.  The code also supports pluggable
// rebuilding policies, but those features are not currently exposed.
// A Rebuild method is available for the application to force a
// rebuild.
