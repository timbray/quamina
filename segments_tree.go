package quamina

import (
	"fmt"
	"strings"
)

const SegmentSeparator = "\n"

type segmentsTree struct {
	root bool

	// "nodes" stores a map from a segment to it's childrens
	// in an hierarchial data like JSON, a node can be Object or Array.
	// for example, in this path "context\nuser\nid", both "context" and "user" will be nodes.
	nodes map[string]*segmentsTree

	// "fields" is storing leaf fields here mapped to their full paths
	// for example:
	//  leaf "id" will be mapped to it's full path "context\nuser\nid"
	//  leaf "user" will be mapped to it's full path "context\nuser"
	fields map[string][]byte
}

// newSegmentsIndex creates a segmentsTree node which is the root
// passing paths will auto-add them to the tree, useful for testing.
func newSegmentsIndex(paths ...string) *segmentsTree {
	st := newSegmentsIndexNode(true)

	for _, path := range paths {
		st.add(path)
	}

	return st
}

// newSegmentsIndexNode initializes a segmentsTree node
func newSegmentsIndexNode(root bool) *segmentsTree {
	return &segmentsTree{
		root:   root,
		nodes:  make(map[string]*segmentsTree),
		fields: make(map[string][]byte),
	}
}

func (p *segmentsTree) add(path string) {
	segments := strings.Split(path, SegmentSeparator)

	// If we have only one segment, it's a field on the root.
	if len(segments) == 1 {
		// It's a direct field.
		p.fields[path] = []byte(path)
		return
	}

	var node *segmentsTree
	node = p

	for i, segment := range segments {
		// If this the last segment, add it as field
		// example: context\nuser\nid, in this case "id" is the field ("context" & "user" are nodes)
		if i == len(segments)-1 {
			node.addSegment(segment, []byte(path))
		} else {
			node = node.getOrCreate(segment)
		}
	}
}

func (p *segmentsTree) getOrCreate(name string) *segmentsTree {
	_, ok := p.nodes[name]
	if !ok {
		p.nodes[name] = newSegmentsIndexNode(false)
	}

	return p.nodes[name]
}

func (p *segmentsTree) addSegment(segment string, path []byte) {
	_, ok := p.fields[segment]
	if !ok {
		p.fields[segment] = path
	}
}

// Get implements SegmentsTreeTracker
func (p *segmentsTree) Get(name []byte) (SegmentsTreeTracker, bool) {
	n, ok := p.nodes[string(name)]
	return n, ok
}

// IsRoot implements SegmentsTreeTracker
func (p *segmentsTree) IsRoot() bool {
	return p.root
}

// IsSegmentUsed implements SegmentsTreeTracker
func (p *segmentsTree) IsSegmentUsed(segment []byte) bool {
	// In the next path: "context\nuser\nid"
	// "context" / "user" are nodes, while "id" is a field
	// As a result a segment can be both node and field, we need to check
	// in both maps.
	_, isField := p.fields[string(segment)]
	if isField {
		return true
	}
	_, isNode := p.nodes[string(segment)]
	return isNode
}

// PathForSegment implements SegmentsTreeTracker
func (p *segmentsTree) PathForSegment(segment []byte) []byte {
	return p.fields[string(segment)]
}

// NodesCount implements SegmentsTreeTracker
func (p *segmentsTree) NodesCount() int {
	return len(p.nodes)
}

// FieldsCount implements SegmentsTreeTracker
func (p *segmentsTree) FieldsCount() int {
	return len(p.fields)
}

// String used for debugging purposes
func (p *segmentsTree) String() string {
	nodeNames := make([]string, 0)
	for n := range p.nodes {
		nodeNames = append(nodeNames, n)
	}

	fieldNames := make([]string, 0)
	for f := range p.fields {
		fieldNames = append(fieldNames, f)
	}

	return fmt.Sprintf("root: %v, nodes [%s], fields: [%s]", p.root, strings.Join(nodeNames, ","), strings.Join(fieldNames, ","))
}

func (p *segmentsTree) copy() *segmentsTree {
	np := newSegmentsIndexNode(p.root)

	// copy fields
	for name, path := range p.fields {
		np.fields[name] = path
	}

	// copy nodes
	for name, node := range p.nodes {
		np.nodes[name] = node.copy()
	}

	return np
}
