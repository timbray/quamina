package quamina

import (
	"testing"
)

func TestSegmentsTreeSanity(t *testing.T) {
	tree := newSegmentsIndex("field1")

	if !tree.IsRoot() {
		t.Errorf(`Expected "newSegmentsIndex" to return a root node: %s`, tree.String())
	}

	expectCounts(t, tree, 1, 0)
	expectSegmentsToBeUsed(t, tree, "field1")

	tree.add("node\nfield")
	tree.add("node\nfield_2")
	tree.add("node\nsub_node\nleaf")

	if !tree.IsRoot() {
		t.Fatalf("Expect tree to be root: %s", tree.String())
	}

	expectCounts(t, tree, 1, 1)
	expectSegmentsToBeUsed(t, tree, "field1", "node")

	n, ok := tree.Get([]byte("node"))
	if !ok {
		t.Fatalf(`Failed to fetch "node" from tree: %s`, tree.String())
	}

	if n.IsRoot() {
		t.Fatalf("Expect node to not be root: %s", n.String())
	}

	expectCounts(t, n, 2, 1)
	expectSegmentsToBeUsed(t, n, "field", "field_2")

	leaf, ok := n.Get([]byte("sub_node"))
	if !ok {
		t.Fatalf(`Failed to fetch "sub_node" from "node": %s`, n.String())
	}

	if leaf.IsRoot() {
		t.Fatalf("Expect sub_node to not be root: %s", leaf.String())
	}

	expectCounts(t, leaf, 1, 0)
	expectSegmentsToBeUsed(t, leaf, "leaf")
}

func TestSegmentsTreeString(t *testing.T) {
	tree := newSegmentsIndex("node\nsub_node\nfield", "root_field")

	expectedString := "root: true, nodes [node], fields: [root_field]"

	if tree.String() != expectedString {
		t.Errorf("Expected tree.String(): [%s] to equal [%s]", tree.String(), expectedString)
	}
}

func expectSegmentsToBeUsed(t *testing.T, tree SegmentsTreeTracker, segments ...string) {
	t.Helper()

	for _, seg := range segments {
		if !tree.IsSegmentUsed([]byte(seg)) {
			t.Fatalf("Expected '%s' segment to be used, but it's not: %s", seg, tree.String())
		}
	}
}

func expectCounts(t *testing.T, tree SegmentsTreeTracker, fieldsCount, nodesCount int) {
	t.Helper()

	if tree.FieldsCount() != fieldsCount || tree.NodesCount() != nodesCount {
		t.Fatalf("Expected to have %v fields & %v nodes: %s", fieldsCount, nodesCount, tree.String())
	}
}
