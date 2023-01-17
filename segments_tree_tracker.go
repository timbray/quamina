package quamina

// SegmentsTreeTracker is an interface used by Flattener to represents all the paths mentioned
// Patterns added to a Quamina instance in AddPattern() calls. It allows a Flattener to determine
// which Event fields may safely be ignored, and also caches the runtime form of the Field.Path
// value.
//
// Consider this JSON example:
//
//	{ "a": {"b": 1, "c": 2}}
//
// The tree will look like that:
//
//	[ root ]
//	   |
//	 [ "a" ] -> as node
//	   |-> with fields of: "b" and "c"
//
// This allow us to traverse the hierarchial data together with the segments tree,
// fetch a node and answer:
//   - Is the current segment is used? (JSON - is the current property needs to be selected)
//   - Do we need to traverse into this Node as well? (JSON - do we need traverse this object?)
//   - How much fields & nodes we have to traverse in the current hierarchy until we are finished?
//     for example: in the current level, in the tree node we have 1 node and 2 fields
//     we finishded selecting them, can we finish traversing this node?
type SegmentsTreeTracker interface {
	// Get returns another level of the hierarchy, referred as "Node"
	// If a node is returned we will need to traverse into (in JSON/CBOR/ProtoBuf/etc..)
	Get(segment []byte) (SegmentsTreeTracker, bool)

	// IsRoot - are we root node?
	// NOTE: need for early exit, can be solved differently maybe.
	IsRoot() bool

	// Called by the Flattener looking at a member name in a JSON object to ascertain
	// whether this particular member of the object is mentioned in any Patterns added
	// to the Quamina instance.
	IsSegmentUsed(segment []byte) bool

	// When a Flattener reaches the last (leaf) step of a path, this returns the full
	// path-name for that Field.  This is an optimization; since these need to be calculated
	// while executing `ddPattern, we might as wewll remember them for use during Flattening.
	PathForSegment(name []byte) []byte

	// Called by the Flattener to return the number of nodes (non-leaf children) and fields
	// (field values) contained in any node.  When processing through the node, once we've
	// hit the right number of nodes and fields we can terminate the Flattening process.
	NodesCount() int
	FieldsCount() int

	// String is used only for debugging.
	String() string
}
