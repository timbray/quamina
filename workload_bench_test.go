package quamina

import (
	"fmt"
	"testing"
)

// Workload characterization benchmarks. These exercise representative
// match-time workloads against the current matcher implementation. They
// are intended as stable baselines: subsequent matcher work (e.g., a
// shared lazy DFA cache, eager nfa2Dfa, or other optimizations) can be
// evaluated by re-running these benchmarks unchanged and comparing.
//
// Each benchmark warms the matcher with a few hundred iterations of the
// chosen event(s) before resetting the timer so allocations from
// first-call laziness do not pollute the steady-state measurement.

// BenchmarkWorkload_ExactString — 1 exact pattern, uniform event.
func BenchmarkWorkload_ExactString(b *testing.B) {
	q, _ := New()
	_ = q.AddPattern("p", `{"x": ["foobar"]}`)
	ev := []byte(`{"x":"foobar"}`)
	for i := 0; i < 100; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
}

// BenchmarkWorkload_SingleShellstyle — 1 wildcard pattern, uniform
// matching event. Hot single-NFA traversal.
func BenchmarkWorkload_SingleShellstyle(b *testing.B) {
	q, _ := New()
	_ = q.AddPattern("p", `{"x": [{"shellstyle": "*foo*"}]}`)
	ev := []byte(`{"x":"abcdefoobarghi"}`)
	for i := 0; i < 100; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
}

// BenchmarkWorkload_ManyOverlappingWildcards — N overlapping shellstyle
// wildcards on the same field. Cost scales with N as the merged NFA's
// epsilon closure grows; this is the textbook NFA-explosion workload.
func BenchmarkWorkload_ManyOverlappingWildcards(b *testing.B) {
	for _, n := range []int{8, 16, 32, 64, 128} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			q, _ := New()
			for i := 0; i < n; i++ {
				a := byte('a' + (i % 13))
				c := byte('a' + ((i + 1) % 13))
				d := byte('a' + ((i + 2) % 13))
				p := fmt.Sprintf(`{"x": [{"shellstyle": "*%c*%c*%c*"}]}`, a, c, d)
				_ = q.AddPattern(fmt.Sprintf("p%d", i), p)
			}
			ev := []byte(`{"x":"abcdefghijklm"}`)
			for i := 0; i < 100; i++ {
				_, _ = q.MatchesForEvent(ev)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = q.MatchesForEvent(ev)
			}
		})
	}
}

// BenchmarkWorkload_RegexAlternation — 20 regex patterns with
// alternation (foo|bar|...). Exercises dense epsilon closure structure.
func BenchmarkWorkload_RegexAlternation(b *testing.B) {
	q, _ := New()
	keywords := []string{"foo", "bar", "baz", "quux", "xyzzy"}
	for i := 0; i < 20; i++ {
		kw := keywords[i%len(keywords)]
		p := fmt.Sprintf(`{"x": [{"regex": "(%s|alt%d)\\d+"}]}`, kw, i)
		_ = q.AddPattern(fmt.Sprintf("p%d", i), p)
	}
	ev := []byte(`{"x":"foo42"}`)
	for i := 0; i < 100; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
}

// BenchmarkWorkload_LiteralInRegex — long literal substring inside a
// regex. Real log-line shape.
func BenchmarkWorkload_LiteralInRegex(b *testing.B) {
	q, _ := New()
	_ = q.AddPattern("p", `{"x": [{"regex": ".*ERROR.*\\d+.*"}]}`)
	ev := []byte(`{"x":"2026-05-18T10:00:00 ERROR request_id=42 connection refused"}`)
	for i := 0; i < 100; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
}

// BenchmarkWorkload_QuantifiedCharClass — regex with {n,m} quantifier
// on a character class. Eager NFA→DFA conversion blows up here.
func BenchmarkWorkload_QuantifiedCharClass(b *testing.B) {
	q, _ := New()
	for i := 0; i < 5; i++ {
		p := fmt.Sprintf(`{"x": [{"regex": "[a-z]{8,16}sfx%d"}]}`, i)
		_ = q.AddPattern(fmt.Sprintf("p%d", i), p)
	}
	ev := []byte(`{"x":"abcdefghijksfx3"}`)
	for i := 0; i < 100; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
}

// BenchmarkWorkload_ManyAnchoredRegex — 200 anchored regex patterns
// with shared prefix/suffix. Cost scales with the cross-product of
// merged regex paths.
func BenchmarkWorkload_ManyAnchoredRegex(b *testing.B) {
	q, _ := New()
	for i := 0; i < 200; i++ {
		p := fmt.Sprintf(`{"x": [{"regex": "PFX[0-9]+SFX%d"}]}`, i)
		_ = q.AddPattern(fmt.Sprintf("p%d", i), p)
	}
	events := [][]byte{
		[]byte(`{"x":"PFX42SFX17"}`),
		[]byte(`{"x":"PFX99SFX42"}`),
		[]byte(`{"x":"PFX1SFX199"}`),
		[]byte(`{"x":"PFX9999SFX0"}`),
	}
	for i := 0; i < 100; i++ {
		_, _ = q.MatchesForEvent(events[i%len(events)])
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.MatchesForEvent(events[i%len(events)])
	}
}

// BenchmarkWorkload_DeepEpsilonNest — regex designed to maximize
// epsilon closure depth via nested alternation and quantifiers.
func BenchmarkWorkload_DeepEpsilonNest(b *testing.B) {
	q, _ := New()
	_ = q.AddPattern("p", `{"x": [{"regex": "((a|b|c)*(d|e|f)*)+"}]}`)
	ev := []byte(`{"x":"abcdefabcdefabcdef"}`)
	for i := 0; i < 100; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.MatchesForEvent(ev)
	}
}

// BenchmarkWorkload_CacheThrashing — pattern admits a huge state space
// (5 wildcards) and events are permuted so each visits different state
// trajectories. Adversarial input for any state-set-caching strategy.
func BenchmarkWorkload_CacheThrashing(b *testing.B) {
	q, _ := New()
	_ = q.AddPattern("p", `{"x": [{"shellstyle": "*X*Y*Z*W*V*"}]}`)
	perms := []string{
		`{"x":"XYZWVabcdefghij"}`,
		`{"x":"jihgfedcbaVWZYX"}`,
		`{"x":"aXbYcZdWeVfghij"}`,
		`{"x":"VWZXYjihgfedcba"}`,
		`{"x":"ZYXWVbacdefghij"}`,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.MatchesForEvent([]byte(perms[i%len(perms)]))
	}
}

// BenchmarkWorkload_ParallelMatchers — N goroutines run MatchesForEvent
// on Copy() instances sharing one matcher. Measures contention behavior
// under concurrent match load.
func BenchmarkWorkload_ParallelMatchers(b *testing.B) {
	for _, gor := range []int{8, 16, 32, 64} {
		b.Run(fmt.Sprintf("G=%d", gor), func(b *testing.B) {
			q, _ := New()
			// Reuse the ManyOverlapping pattern set at N=64.
			n := 64
			for i := 0; i < n; i++ {
				a := byte('a' + (i % 13))
				c := byte('a' + ((i + 1) % 13))
				d := byte('a' + ((i + 2) % 13))
				p := fmt.Sprintf(`{"x": [{"shellstyle": "*%c*%c*%c*"}]}`, a, c, d)
				_ = q.AddPattern(fmt.Sprintf("p%d", i), p)
			}
			ev := []byte(`{"x":"abcdefghijklm"}`)
			for i := 0; i < 200; i++ {
				_, _ = q.MatchesForEvent(ev)
			}
			b.SetParallelism(gor)
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				cp := q.Copy()
				for pb.Next() {
					_, _ = cp.MatchesForEvent(ev)
				}
			})
		})
	}
}
