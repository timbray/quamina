package quamina

import (
	"testing"
)

func TestMakeShellStyleAutomaton(t *testing.T) {
	patterns := []string{
		`"foo*"`,
		`"*foo"`,
		`"*foo*"`,
		`"xx*yy*zz"`,
		`"*xx*yy*"`,
	}
	shouldsForPatterns := [][]string{
		{`"fooabc"`, `"foo"`},
		{`"afoo"`, `"foo"`},
		{`"xxfooyy"`, `"fooyy"`, `"xxfoo"`, `"foo"`},
		{`"xxabyycdzz"`, `"xxyycdzz"`, `"xxabyyzz"`, `"xxyyzz"`},
		{`"abxxcdyyef"`, `"xxcdyyef"`, `"abxxyyef"`, `"abxxcdyy"`, `"abxxyy"`, `"xxcdyy"`, `"xxyyef"`, `"xxyy"`},
	}
	shouldNotForPatterns := [][]string{
		{`"afoo"`, `"fofo"`},
		{`"foox"`, `"afooo"`},
		{`"afoa"`, `"fofofoxooxoo"`},
		{`"xyzyxzy yy zz"`, `"zz yy xx"`},
		{`"ayybyyzxx"`},
	}

	for i, pattern := range patterns {
		myNext := newFieldMatcher()
		a, wanted := makeShellStyleAutomaton([]byte(pattern), myNext)
		if wanted != myNext {
			t.Error("bad next on: " + pattern)
		}
		for _, should := range shouldsForPatterns[i] {
			gotTrans := traverseA(a, should)
			if wanted != gotTrans {
				t.Errorf("Failure for %s on %s", pattern, should)
			}
		}
		for _, shouldNot := range shouldNotForPatterns[i] {
			gotTrans := traverseA(a, shouldNot)
			if gotTrans != nil {
				t.Errorf("bogus match for %s on %s", pattern, shouldNot)
			}
		}
	}
}

func traverseA(t *smallTable, val string) *fieldMatcher {
	var next smallStep
	for _, ch := range []byte(val) {
		next = t.step(ch)
		if next == nil {
			return nil
		}
		if next.HasTransition() {
			return next.SmallTransition().fieldMatchers[0]
		}
		t = next.SmallTable()
	}
	if !next.HasTransition() {
		return nil
	}
	trans := next.SmallTransition()
	return trans.fieldMatchers[0]
}
