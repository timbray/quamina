package fields

type ArrayPos struct {
	Array int32
	Pos   int32
}
type Field struct {
	Path       []byte
	Val        []byte
	ArrayTrail []ArrayPos
}

// Arrays are invisible in the automaton.  That is to say, if an event has
//  { "a": [ 1, 2, 3 ] }
//  Then the fields will be a/1, a/2, and a/3
//  Same for  {"a": [[1, 2], 3]} or any other permutation
//  So if you have {"a": [ { "b": 1, "c": 2}, {"b": 3, "c": 4}] }
//  then a pattern like { "a": { "b": 1, "c": 4 } } would match.
// To prevent that from happening, each ArrayPos contains two numbers; the first identifies the array in
//  the event that this name/val occurred in, the second the position in the array. We don't allow
//  transitioning between field values that occur in different positions in the same array.
//  See the arrays_test unit test for more examples.
func (f *Field) IsArrayCompatible(other *Field) bool {
	for _, fromAPos := range f.ArrayTrail {
		for _, toAPos := range other.ArrayTrail {
			if fromAPos.Array == toAPos.Array && fromAPos.Pos != toAPos.Pos {
				return false
			}
		}
	}
	return true
}
