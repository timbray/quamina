package quamina

import (
	"os"
	"strings"
	"testing"
)

var (
	topMatches []X
	topFields  []Field
)

const PatternContext = `{ "context": { "user_id": [9034], "friends_count": [158] } }`
const PatternMiddleNestedField = `{ "payload": { "user": { "id_str": ["903487807"] } } }`
const PatternLastField = `{ "payload": { "lang_value": ["ja"] } }`

func Benchmark_JsonFlattener_ContextFields(b *testing.B) {
	RunBenchmarkWithJSONFlattener(b, "context\nuser_id", "context\nfriends_count")
}

func Benchmark_JsonFlattener_MiddleNestedField(b *testing.B) {
	RunBenchmarkWithJSONFlattener(b, "payload\nuser\nid_str")
}

func Benchmark_JsonFlattener_LastField(b *testing.B) {
	RunBenchmarkWithJSONFlattener(b, "payload\nlang_value")
}

func RunBenchmarkWithJSONFlattener(b *testing.B, paths ...string) {
	b.Helper()
	var localFields []Field

	event, err := os.ReadFile("./testdata/status.json")
	if err != nil {
		b.Fatal(err)
	}

	flattener := newJSONFlattener()

	t := newSegmentsIndex(paths...)
	results, err := flattener.Flatten(event, t)
	if err != nil {
		b.Fatal(err)
	}
	PrintFields(b, results)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		fields, err := flattener.Flatten(event, t)
		if err != nil {
			b.Fatal(err)
		}
		localFields = fields
	}
	topFields = localFields
}

func Benchmark_JsonFlattner_Evaluate_ContextFields(b *testing.B) {
	q, err := New()

	if err != nil {
		b.Fatal(err)
	}

	RunBenchmarkEvaluate(b, q, PatternContext)
}

func Benchmark_JsonFlattner_Evaluate_MiddleNestedField(b *testing.B) {
	q, err := New()

	if err != nil {
		b.Fatal(err)
	}

	RunBenchmarkEvaluate(b, q, PatternMiddleNestedField)
}

func Benchmark_JsonFlattner_Evaluate_LastField(b *testing.B) {
	q, err := New()

	if err != nil {
		b.Fatal(err)
	}

	RunBenchmarkEvaluate(b, q, PatternLastField)
}

func RunBenchmarkEvaluate(b *testing.B, q *Quamina, pattern string) {
	b.Helper()

	err := q.AddPattern(1, pattern)
	if err != nil {
		b.Fatalf("Failed adding pattern: %+v", err)
	}

	event, err := os.ReadFile("./testdata/status.json")
	if err != nil {
		b.Fatal(err)
	}

	matches, err := q.MatchesForEvent(event)
	if err != nil {
		b.Fatalf("failed matching: %s", err)
	}

	if len(matches) != 1 {
		b.Fatalf("in-correct matching: %+v", matches)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		matches, err := q.MatchesForEvent(event)
		if err != nil {
			b.Fatalf("failed matching: %s", err)
		}

		if len(matches) != 1 {
			b.Fatalf("in-correct matching: %+v", matches)
		}
	}
}

func PrintFields(tb testing.TB, fields []Field) {
	tb.Helper()

	tb.Logf("> Fields\n")

	for _, field := range fields {
		tb.Logf("Path [%s] Val [%s] ArrayTrail [%+v]\n", strings.ReplaceAll(string(field.Path), "\n", "->"), field.Val, field.ArrayTrail)
	}
	tb.Logf("\n")
}
