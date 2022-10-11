package quamina

// SegmentsTreeTracker is an interfaced used by Flattener to represents all living paths
// as segments tree in order to allow efficient selection of fields from hierarchial data structure.
//
// Consider this JSON exampe:
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

	// Do we need to select the given segment from event as Quamina's Field.
	IsSegmentUsed(segment []byte) bool

	// Given a segment, returns the full path.
	// This is caching mechanism.
	PathForSegment(name []byte) []byte

	// NodesCount and FieldsCount returns how much nodes we have to trverse into
	// and how much fields we need to pluck.
	// Once we are on a node, we need to decrement those counts, once we got zero in both - we can do early exit.
	NodesCount() int
	FieldsCount() int

	// String is used only for debugging.
	String() string
}
