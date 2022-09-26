package quamina

// NameTracker is an interface representing a wrapper for a set of byte slices. The intended use is by a Flattener
// which is traversing a data object with the goal of turning it into a list of name/value pairs. The cost of
// Quamina's MatchesForEvent API is strongly related to the number of fields it has to look at.  Therefore, a
// Flattener should call NameTracker for each field it encounters and if it returns false, skip that field
// and not add it to the flattened output. Here's a clarifying example; consider this JSON event:
//
//	{"a": {"b": 1, "c": 2}}
//
// Assuming NameTracker always returns positive, the flattened output should look like
//
//	"a\nb" 1
//	"a\nc" 2
//
// Note that the values stored in NameTracker are not the full paths "a\nb" and "a\nc", but the path segments
// "a", "b", and "c".  This reduces the work the Flattener has to do - whenever it processes an object, it need
// only look up the member names in the NameTracker. A little thought reveals that this can produce false
// positives and some potential wasted work, but is a good trade-off.
type NameTracker interface {
	IsNameUsed(label []byte) bool
}
