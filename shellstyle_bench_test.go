package quamina

import (
	"fmt"
	"testing"
)

// BenchmarkShellstyleMultiMatch exercises shellstyle pattern matching with wildcards
// across a variety of character sets including ASCII, CJK, and emoji. This benchmark
// is useful for measuring allocation patterns in NFA traversal code paths.
func BenchmarkShellstyleMultiMatch(b *testing.B) {
	q, _ := New()

	// Add multiple shellstyle patterns like in TestBigShellStyle
	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P"} {
		pattern := fmt.Sprintf(`{"STREET": [ {"shellstyle": "%s*"} ]}`, letter)
		q.AddPattern(letter, pattern)
	}

	// Add some funky patterns with multiple wildcards that trigger more complex NFA traversal
	q.AddPattern("funky1", `{"STREET": [ {"shellstyle": "*E*E*E*"} ]}`)
	q.AddPattern("funky2", `{"STREET": [ {"shellstyle": "*A*B*"} ]}`)
	q.AddPattern("funky3", `{"STREET": [ {"shellstyle": "*N*P*"} ]}`)
	q.AddPattern("funky4", `{"STREET": [ {"shellstyle": "*O*O*O*"} ]}`)

	// Add CJK patterns to test Unicode handling
	q.AddPattern("jp1", `{"STREET": [ {"shellstyle": "*東京*"} ]}`)
	q.AddPattern("jp2", `{"STREET": [ {"shellstyle": "新*"} ]}`)
	q.AddPattern("cn1", `{"STREET": [ {"shellstyle": "*北京*"} ]}`)
	q.AddPattern("cn2", `{"STREET": [ {"shellstyle": "上海*"} ]}`)
	q.AddPattern("kr1", `{"STREET": [ {"shellstyle": "*서울*"} ]}`)

	// Add emoji patterns to test multi-byte UTF-8 sequences
	q.AddPattern("emoji1", `{"STREET": [ {"shellstyle": "*🎉*"} ]}`)
	q.AddPattern("emoji2", `{"STREET": [ {"shellstyle": "🚀*"} ]}`)
	q.AddPattern("emoji3", `{"STREET": [ {"shellstyle": "*❤️*"} ]}`)
	q.AddPattern("emoji4", `{"STREET": [ {"shellstyle": "*🌟*🎯*"} ]}`)

	// Events that will match and require NFA traversal
	events := [][]byte{
		// English streets
		[]byte(`{"STREET": "ASHBURY"}`),
		[]byte(`{"STREET": "BELVEDERE"}`),
		[]byte(`{"STREET": "CRANLEIGH"}`),
		[]byte(`{"STREET": "DEER PARK"}`),
		[]byte(`{"STREET": "EMBARCADERO"}`),
		[]byte(`{"STREET": "FULTON"}`),
		[]byte(`{"STREET": "GEARY"}`),
		[]byte(`{"STREET": "HAIGHT"}`),
		[]byte(`{"STREET": "IRVING"}`),
		[]byte(`{"STREET": "JUDAH"}`),
		[]byte(`{"STREET": "KEARNY"}`),
		[]byte(`{"STREET": "LOMBARD"}`),
		[]byte(`{"STREET": "MARKET"}`),
		[]byte(`{"STREET": "NORIEGA"}`),
		[]byte(`{"STREET": "OCTAVIA"}`),
		[]byte(`{"STREET": "POLK"}`),
		// Streets with multiple vowels for funky patterns
		[]byte(`{"STREET": "EMBARCADERO STREET"}`),
		[]byte(`{"STREET": "ALABAMA"}`),
		[]byte(`{"STREET": "NAPOLEON"}`),
		[]byte(`{"STREET": "COLORADO"}`),
		// CJK streets
		[]byte(`{"STREET": "東京タワー通り"}`),
		[]byte(`{"STREET": "新宿駅前"}`),
		[]byte(`{"STREET": "北京路"}`),
		[]byte(`{"STREET": "上海南京路"}`),
		[]byte(`{"STREET": "서울대로"}`),
		[]byte(`{"STREET": "江南大路"}`),
		// Emoji streets (fun test case!)
		[]byte(`{"STREET": "Party Street 🎉"}`),
		[]byte(`{"STREET": "🚀 Rocket Road"}`),
		[]byte(`{"STREET": "Love ❤️ Lane"}`),
		[]byte(`{"STREET": "Star 🌟 Plaza 🎯"}`),
		// Mixed
		[]byte(`{"STREET": "Tokyo 東京 Street"}`),
		[]byte(`{"STREET": "Happy 😊 Avenue"}`),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, event := range events {
			_, _ = q.MatchesForEvent(event)
		}
	}
}
