package quamina

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// BenchmarkShellstyleSimpleWildcard exercises patterns like "a*b" where the
// full DFA is tiny — just a handful of states. An eager nfa2dfa conversion
// would trivially handle these and produce the fastest possible matcher, but
// Quamina currently falls back to NFA traversal for shellstyle patterns.
// This benchmark exists to show that simple wildcards deserve DFA treatment,
// whether eager or lazy.
func BenchmarkShellstyleSimpleWildcard(b *testing.B) {
	// Simple prefix*suffix patterns — the DFA for each is ~3 states.
	simplePatterns := []struct {
		name       string
		shellstyle string
	}{
		{"a*b", "a*b"},
		{"foo*bar", "foo*bar"},
		{"x*y*z", "x*y*z"},
		{"he*lo", "he*lo"},
	}

	for _, sp := range simplePatterns {
		b.Run(sp.name, func(b *testing.B) {
			q, _ := New()
			pattern := fmt.Sprintf(`{"val": [{"shellstyle": %q}]}`, sp.shellstyle)
			if err := q.AddPattern(sp.name, pattern); err != nil {
				b.Fatal(err)
			}

			// Build events that match — filler is lowercase ASCII.
			rng := rand.New(rand.NewSource(42))
			const poolSize = 64
			events := make([][]byte, poolSize)
			for i := range events {
				var buf strings.Builder
				// For "a*b": produce "a<random chars>b"
				// For "x*y*z": produce "x<random>y<random>z"
				parts := strings.Split(sp.shellstyle, "*")
				for j, part := range parts {
					buf.WriteString(part)
					if j < len(parts)-1 {
						// random filler between fixed parts
						for k := 0; k < 3+rng.Intn(15); k++ {
							buf.WriteByte(byte('a' + rng.Intn(26)))
						}
					}
				}
				events[i] = []byte(fmt.Sprintf(`{"val": %q}`, buf.String()))
			}

			// Verify matches.
			for i, event := range events {
				matches, err := q.MatchesForEvent(event)
				if err != nil {
					b.Fatal(err)
				}
				if len(matches) == 0 {
					b.Fatalf("event %d: no match for %s", i, event)
				}
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matches, err := q.MatchesForEvent(events[i%poolSize])
				if err != nil {
					b.Fatal(err)
				}
				if len(matches) == 0 {
					b.Fatalf("event %d: no match", i%poolSize)
				}
			}
		})
	}
}

// BenchmarkShellstyleNarrowInput creates shellstyle patterns whose wildcards can
// match almost any Unicode codepoint, then benchmarks against input drawn from
// a tiny slice of the alphabet. The eager DFA must construct states covering
// the full Unicode byte space implied by "*". A demand-driven approach would
// only need to materialize states for the bytes actually encountered, making
// its effective state space proportional to the input alphabet rather than
// the pattern alphabet.
func BenchmarkShellstyleNarrowInput(b *testing.B) {
	// Anchors are drawn from diverse Unicode blocks so the NFA's wildcard
	// transitions must accommodate the full UTF-8 encoding range. But the
	// text *between* the anchors in the input events only uses a narrow set.
	type anchorSet struct {
		name    string
		anchors []string // characters that appear in patterns as fixed points around "*"
	}

	anchorSets := []anchorSet{
		{
			name:    "ascii_anchors",
			anchors: []string{"X", "Y", "Z", "W", "Q"},
		},
		{
			name:    "cjk_anchors",
			anchors: []string{"東", "京", "北", "海", "山"},
		},
		{
			name:    "mixed_script_anchors",
			anchors: []string{"A", "Ω", "东", "🎯", "Й"},
		},
	}

	// The narrow input alphabets — the characters that fill in between anchors.
	type inputAlphabet struct {
		name  string
		chars []rune
	}

	inputAlphabets := []inputAlphabet{
		{
			name:  "digits_only",
			chars: []rune("0123456789"),
		},
		{
			name:  "lowercase_ascii",
			chars: []rune("abcdefghijklmnopqrstuvwxyz"),
		},
		{
			name:  "narrow_cjk",
			chars: []rune("一二三四五六七八九十"),
		},
	}

	for _, anchors := range anchorSets {
		for _, alphabet := range inputAlphabets {
			for _, patternCount := range []int{8, 32, 128} {
				name := fmt.Sprintf("anchors=%s/input=%s/patterns=%d",
					anchors.name, alphabet.name, patternCount)

				b.Run(name, func(b *testing.B) {
					q, _ := New()

					// Build patterns like: *<anchor1>*<anchor2>*
					// Each wildcard can match any Unicode, but input will
					// only contain chars from the narrow alphabet.
					type anchorPair struct{ a1, a2 string }
					rng := rand.New(rand.NewSource(99))
					pairs := make([]anchorPair, 0, patternCount)
					for i := 0; i < patternCount; i++ {
						a1 := anchors.anchors[rng.Intn(len(anchors.anchors))]
						a2 := anchors.anchors[rng.Intn(len(anchors.anchors))]
						pairs = append(pairs, anchorPair{a1, a2})
						shellstyle := fmt.Sprintf("*%s*%s*", a1, a2)
						pattern := fmt.Sprintf(`{"val": [{"shellstyle": %q}]}`, shellstyle)
						if err := q.AddPattern(fmt.Sprintf("p%d", i), pattern); err != nil {
							b.Fatal(err)
						}
					}

					// Build events whose values contain the anchor characters
					// (so they match) surrounded by padding drawn exclusively
					// from the narrow alphabet.
					const poolSize = 32
					events := make([][]byte, poolSize)
					for i := range events {
						var buf strings.Builder
						// random narrow padding
						for j := 0; j < 5+rng.Intn(10); j++ {
							buf.WriteRune(alphabet.chars[rng.Intn(len(alphabet.chars))])
						}
						// insert two anchors from an actual pattern so the event is guaranteed to match
						pair := pairs[rng.Intn(len(pairs))]
						buf.WriteString(pair.a1)
						for j := 0; j < 5+rng.Intn(10); j++ {
							buf.WriteRune(alphabet.chars[rng.Intn(len(alphabet.chars))])
						}
						buf.WriteString(pair.a2)
						for j := 0; j < 5+rng.Intn(10); j++ {
							buf.WriteRune(alphabet.chars[rng.Intn(len(alphabet.chars))])
						}
						events[i] = []byte(fmt.Sprintf(`{"val": %q}`, buf.String()))
					}

					// Sanity check: at least some events should match.
					matchCount := 0
					for _, event := range events {
						matches, err := q.MatchesForEvent(event)
						if err != nil {
							b.Fatal(err)
						}
						matchCount += len(matches)
					}
					if matchCount == 0 {
						b.Fatal("no matches at all — check pattern/event construction")
					}

					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						matches, err := q.MatchesForEvent(events[i%poolSize])
						if err != nil {
							b.Fatal(err)
						}
						if len(matches) == 0 {
							b.Fatalf("expected matches for event %d", i%poolSize)
						}
					}
				})
			}
		}
	}
}

// BenchmarkShellstyleWidePatternsScaling focuses specifically on the scaling
// behavior as pattern count grows, with maximally broad patterns (every "*"
// accepts all of Unicode) but input restricted to ASCII digits. This isolates
// a demand-driven DFA's advantage: the cache only needs entries for ~10 distinct byte
// values regardless of how many Unicode codepoints the pattern theoretically
// permits.
func BenchmarkShellstyleWidePatternsScaling(b *testing.B) {
	digits := []rune("0123456789")

	// Use anchors from multiple scripts to force the NFA to have transitions
	// spanning the full UTF-8 byte range.
	allAnchors := []string{
		"A", "B", "C", "D", "E", // Latin
		"Α", "Β", "Γ", "Δ", "Ε", // Greek
		"東", "京", "北", "上", "大", // CJK
		"🎯", "🚀", "🌟", "❤", "🎉", // Emoji
		"Д", "Ж", "З", "И", "К", // Cyrillic
	}

	for _, patternCount := range []int{8, 16, 32, 64, 128, 256, 512} {
		b.Run(fmt.Sprintf("patterns=%d", patternCount), func(b *testing.B) {
			q, _ := New()
			rng := rand.New(rand.NewSource(77))

			type anchorPair struct{ a1, a2 string }
			pairs := make([]anchorPair, 0, patternCount)
			for i := 0; i < patternCount; i++ {
				a1 := allAnchors[rng.Intn(len(allAnchors))]
				a2 := allAnchors[rng.Intn(len(allAnchors))]
				pairs = append(pairs, anchorPair{a1, a2})
				shellstyle := fmt.Sprintf("*%s*%s*", a1, a2)
				pattern := fmt.Sprintf(`{"val": [{"shellstyle": %q}]}`, shellstyle)
				if err := q.AddPattern(fmt.Sprintf("p%d", i), pattern); err != nil {
					b.Fatal(err)
				}
			}

			// Events use only ASCII digits as filler — the narrowest possible
			// byte alphabet (10 distinct values, all single-byte).
			const poolSize = 64
			events := make([][]byte, poolSize)
			for i := range events {
				var buf strings.Builder
				// digit padding
				for j := 0; j < 3+rng.Intn(5); j++ {
					buf.WriteRune(digits[rng.Intn(len(digits))])
				}
				// two anchors from an actual pattern embedded in digit soup
				pair := pairs[rng.Intn(len(pairs))]
				buf.WriteString(pair.a1)
				for j := 0; j < 3+rng.Intn(5); j++ {
					buf.WriteRune(digits[rng.Intn(len(digits))])
				}
				buf.WriteString(pair.a2)
				for j := 0; j < 3+rng.Intn(5); j++ {
					buf.WriteRune(digits[rng.Intn(len(digits))])
				}
				events[i] = []byte(fmt.Sprintf(`{"val": %q}`, buf.String()))
			}

			matchCount := 0
			for _, event := range events {
				matches, err := q.MatchesForEvent(event)
				if err != nil {
					b.Fatal(err)
				}
				matchCount += len(matches)
			}
			if matchCount == 0 {
				b.Fatal("no matches — check construction")
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matches, err := q.MatchesForEvent(events[i%poolSize])
				if err != nil {
					b.Fatal(err)
				}
				if len(matches) == 0 {
					b.Fatalf("expected matches for event %d", i%poolSize)
				}
			}
		})
	}
}

// BenchmarkShellstyleSimpleWildcardScaling adds multiple simple patterns to
// show that even a modest collection of small-DFA patterns benefits from DFA
// conversion. Each pattern is independent (different prefix/suffix), so the
// merged DFA stays small.
func BenchmarkShellstyleSimpleWildcardScaling(b *testing.B) {
	prefixes := "abcdefghijklmnopqrstuvwxyz"
	suffixes := "zyxwvutsrqponmlkjihgfedcba"

	for _, patternCount := range []int{1, 4, 8, 16, 26} {
		b.Run(fmt.Sprintf("patterns=%d", patternCount), func(b *testing.B) {
			q, _ := New()

			for i := 0; i < patternCount; i++ {
				shellstyle := fmt.Sprintf("%c*%c", prefixes[i], suffixes[i])
				pattern := fmt.Sprintf(`{"val": [{"shellstyle": %q}]}`, shellstyle)
				if err := q.AddPattern(fmt.Sprintf("p%d", i), pattern); err != nil {
					b.Fatal(err)
				}
			}

			// Build events that match — each targets a random pattern.
			rng := rand.New(rand.NewSource(42))
			const poolSize = 64
			events := make([][]byte, poolSize)
			for i := range events {
				idx := rng.Intn(patternCount)
				var buf strings.Builder
				buf.WriteByte(prefixes[idx])
				for j := 0; j < 5+rng.Intn(20); j++ {
					buf.WriteByte(byte('a' + rng.Intn(26)))
				}
				buf.WriteByte(suffixes[idx])
				events[i] = []byte(fmt.Sprintf(`{"val": %q}`, buf.String()))
			}

			// Verify at least some match.
			matchCount := 0
			for _, event := range events {
				matches, err := q.MatchesForEvent(event)
				if err != nil {
					b.Fatal(err)
				}
				matchCount += len(matches)
			}
			if matchCount == 0 {
				b.Fatal("no matches at all")
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matches, err := q.MatchesForEvent(events[i%poolSize])
				if err != nil {
					b.Fatal(err)
				}
				if len(matches) == 0 {
					b.Fatalf("event %d: no match", i%poolSize)
				}
			}
		})
	}
}

// BenchmarkShellstyleZWJEmoji exercises NFA traversal on input containing
// ZWJ (Zero Width Joiner) emoji sequences mixed with Japanese text. This is
// a worst case for byte-level automaton traversal because:
//
//  1. ZWJ emoji sequences encode a single visible glyph as many codepoints
//     joined by U+200D (ZWJ), producing 15-25+ bytes per "character".
//  2. The ZWJ byte sequence (0xE2 0x80 0x8D) shares its leading byte 0xE2
//     with CJK characters, hiragana, katakana, and other BMP codepoints,
//     creating massive byte-level ambiguity in the NFA.
//  3. Variation selectors (U+FE0F = 0xEF 0xB8 0x8F) add further multi-byte
//     sequences that interleave with the emoji and Japanese text.
//
// The NFA must branch at nearly every byte because 0xE2 and 0xEF are
// shared prefixes across many unrelated codepoints in the input.
func BenchmarkShellstyleZWJEmoji(b *testing.B) {
	// ZWJ emoji sequences — each is a single glyph but many bytes.
	zwjEmoji := []string{
		"👨\u200D👩\u200D👧\u200D👦", // family
		"👩\u200D🚀",               // woman astronaut
		"🏳\uFE0F\u200D🌈",         // rainbow flag
		"👨\u200D💻",               // man technologist
		"🧑\u200D🎤",               // singer
		"👩\u200D🔬",               // woman scientist
		"🐻\u200D❄\uFE0F",         // polar bear
		"👁\uFE0F\u200D🗨\uFE0F",   // eye in speech bubble
	}

	// Japanese text that shares leading UTF-8 bytes with ZWJ sequences.
	// Hiragana (U+3040-309F): 0xE3 0x81 0x80 - 0xE3 0x82 0x9F
	// Katakana (U+30A0-30FF): 0xE3 0x82 0xA0 - 0xE3 0x83 0xBF
	// CJK (U+4E00+):          0xE4-0xE9 ...
	// All start with 0xE3/0xE4+ which the NFA cannot distinguish from
	// 0xE2 (ZWJ prefix) without reading the second byte.
	japaneseFiller := []string{
		"東京都渋谷区",
		"新宿駅前通り",
		"こんにちは",
		"カタカナテスト",
		"令和七年",
		"人工知能研究所",
		"品川駅南口",
		"秋葉原電気街",
	}

	// Patterns use ZWJ emoji as anchors with wildcards between them.
	// The "*" must handle both Japanese multi-byte text and ZWJ byte
	// sequences, forcing the NFA to branch heavily on shared leading bytes.
	type benchCase struct {
		name         string
		patternCount int
	}

	cases := []benchCase{
		{"patterns=4", 4},
		{"patterns=8", 8},
		{"patterns=16", 16},
		{"patterns=32", 32},
		{"patterns=64", 64},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			q, _ := New()
			rng := rand.New(rand.NewSource(2025))

			type emojiPair struct{ e1, e2 string }
			ePairs := make([]emojiPair, 0, bc.patternCount)
			for i := 0; i < bc.patternCount; i++ {
				e1 := zwjEmoji[rng.Intn(len(zwjEmoji))]
				e2 := zwjEmoji[rng.Intn(len(zwjEmoji))]
				ePairs = append(ePairs, emojiPair{e1, e2})
				shellstyle := fmt.Sprintf("*%s*%s*", e1, e2)
				pattern := fmt.Sprintf(`{"val": [{"shellstyle": %q}]}`, shellstyle)
				if err := q.AddPattern(fmt.Sprintf("p%d", i), pattern); err != nil {
					b.Fatal(err)
				}
			}

			// Events: Japanese filler interspersed with ZWJ emoji anchors.
			// The NFA sees a stream of 0xE2, 0xE3, 0xE4, 0xEF bytes and
			// must disambiguate at every step.
			const poolSize = 64
			events := make([][]byte, poolSize)
			for i := range events {
				pair := ePairs[rng.Intn(len(ePairs))]
				var buf strings.Builder
				buf.WriteString(japaneseFiller[rng.Intn(len(japaneseFiller))])
				buf.WriteString(pair.e1)
				buf.WriteString(japaneseFiller[rng.Intn(len(japaneseFiller))])
				buf.WriteString(pair.e2)
				buf.WriteString(japaneseFiller[rng.Intn(len(japaneseFiller))])
				events[i] = []byte(fmt.Sprintf(`{"val": %q}`, buf.String()))
			}

			matchCount := 0
			for _, event := range events {
				matches, err := q.MatchesForEvent(event)
				if err != nil {
					b.Fatal(err)
				}
				matchCount += len(matches)
			}
			if matchCount == 0 {
				b.Fatal("no matches — check pattern/event construction")
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matches, err := q.MatchesForEvent(events[i%poolSize])
				if err != nil {
					b.Fatal(err)
				}
				if len(matches) == 0 {
					b.Fatalf("event %d: no match", i%poolSize)
				}
			}
		})
	}
}
