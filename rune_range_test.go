package quamina

import (
	"fmt"
	"testing"
)

func TestSkinnyRuneTree(t *testing.T) {
	// utf-8: E4, B8, AD
	var r rune = 0x4e2d
	utf8 := []byte{0xE4, 0xB8, 0xAD}
	srt := &skinnyRuneTreeNode{}
	pp := newPrettyPrinter(246758)
	tt := newSmallTable()
	pp.labelTable(tt, "Next")
	dest := &faState{table: tt, fieldTransitions: []*fieldMatcher{{}}}
	addSkinnyRuneTreeEntry(srt, r, dest)
	addSkinnyRuneTreeEntry(srt, r+1, dest)
	addSkinnyRuneTreeEntry(srt, r+3, dest)
	fa := nfaFromSkinnyRuneTree(srt, pp)
	precomputeEpsilonClosures(fa)
	fmt.Println("FA:\n" + pp.printNFA(fa))
	trans := []*fieldMatcher{}
	bufs := newNfaBuffers()
	matches := traverseNFA(fa, utf8, trans, bufs, pp)
	if len(matches) != 1 {
		t.Error("MISSED")
	}
}
