# `DelPattern`

The core quamina.Matcher doesn't currently support deleting patterns.
Some of the contemplated implementations of would probably be pretty
difficult.  At least one approach is pretty easy: Wrap the current
matcher to filter removed patterns from match results and periodically
rebuild the matcher from scrach with the live patterns.  More
specifically:

1. Remember patterns that have been added
2. Remember patterns that have been removed (implicitly)
3. Filter `MatchesFor...()` results to remove any removed patterns
4. Rebuilding the matcher state periodically with only the live
   patterns
5. Maintain some statistics to help decide when to rebuild

The implementation of the set of live patterns is pluggable via a Go
interface (`State`).  The default implementation `MemState` is just a
`map[quamina.X]string`.  Other implementations could provide
persistence.

By default, rebuilding is triggered automatically (synchronously
currently) during mutations.  You can also force a manual `Rebuild()`,
and you can `DisableRebuild()` to prevent any automation rebuilds.
The code also supports pluggable rebuilding policies, but those
features are not currently exposed.

