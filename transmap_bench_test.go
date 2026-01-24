package quamina

import (
	"testing"
)

// BenchmarkCoreMatcherTransmap demonstrates the allocation savings from pooling transmap
// by using the CoreMatcher API directly (which didn't have pooling before) with a
// pattern that requires NFA traversal (shellstyle).
func BenchmarkCoreMatcherTransmap(b *testing.B) {
	cm := newCoreMatcher()
	// Add a shellstyle pattern which uses NFA traversal
	// This forces the use of traverseNFA and thus transmap
	err := cm.addPattern("shell", `{"key": [ {"shellstyle": "*val*"} ]}`)
	if err != nil {
		b.Fatal(err)
	}

	event := []byte(`{"key": "somevalue"}`)

	// Pre-allocate flattener to isolate buffer allocations
	flattener := newJSONFlattener()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// matchesForJSONWithFlattener is the lower-level API that:
		// 1. Before PR: Allocated new nfaBuffers every time
		// 2. Inside traverseNFA: Allocated new transmap every time
		// 3. After PR: Reuses everything from sync.Pool
		_, err := cm.matchesForJSONWithFlattener(event, flattener)
		if err != nil {
			b.Fatal(err)
		}
	}
}
