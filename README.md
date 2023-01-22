# Quamina

[![Tests](https://github.com/timbray/quamina/actions/workflows/go-unit-tests.yaml/badge.svg)](https://github.com/timbray/quamina/actions/workflows/go-unit-tests.yaml)
[![Latest Release](https://img.shields.io/github/release/timbray/quamina.svg?logo=github&style=flat-square)](https://github.com/timbray/quamina/releases/latest)
[![codecov](https://codecov.io/gh/timbray/quamina/branch/main/graph/badge.svg?token=TC7MW723JO)](https://codecov.io/gh/timbray/quamina)
[![Go Report Card](https://goreportcard.com/badge/github.com/timbray/quamina)](https://goreportcard.com/report/github.com/timbray/quamina)
[![timbray/quamina](https://img.shields.io/github/go-mod/go-version/timbray/quamina)](https://github.com/timbray/quamina)
[![Go Reference](https://pkg.go.dev/badge/github.com/timbray/quamina.svg)](https://pkg.go.dev/github.com/timbray/quamina)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

### Fast pattern-matching library

**Quamina** implements a data type with APIs to
create an instance and add multiple **Patterns** to it,
and then query data objects called **Events** to
discover which of the Patterns match
the fields in the Event.

Quamina has no run-time dependencies beyond built-in Go libraries.

Quamina [welcomes contributions](CONTRIBUTING.md).

### Status

This is Quamina's first stable release, v1.0.0. Starting now,
we plan to change APIs only additively.

Note that we have documented more APIs than are actually
fully implemented, with the intent of showing direction.

### Patterns

Consider the following JSON Event, taken from the example
in RFC 8259:

```json
{
  "Image": {
    "Width":  800,
    "Height": 600,
    "Title":  "View from 15th Floor",
    "Thumbnail": {
      "Url":    "http://www.example.com/image/481989943",
      "Height": 125,
      "Width":  100
    },
    "Animated" : false,
    "IDs": [116, 943, 234, 38793]
  }
}
```

The following Patterns would match it:
```json
{"Image": {"Width": [800]}}
```
```json
{
  "Image": {
    "Animated": [ false ],
    "Thumbnail": {
      "Height": [ 125 ]
    },
    "IDs": [ 943 ]
  }
}
```
```json
{"Image": { "Title": [ { "exists": true } ] } }
```
```json
{
  "Image":  {
    "Width": [800],
    "Title": [ { "exists": true } ],
    "Animated": [ false ]
  }
}
```
```json
{"Image": { "Width": [800], "IDs": [ { "exists": true } ] } }
```
```json
{"Foo": [ { "exists": false } ] }
```
```json
{
  "Image": {
    "Thumbnail": { "Url": [ { "shellstyle": "*9943" } ] }
  }
}
```
```json
{
  "Image": {
    "Thumbnail": { "Url":
      [ { "shellstyle": "http://www.example.com/*" } ] }
  }
}
```
```json
{
  "Image": {
    "Thumbnail": { "Url":
      [ { "shellstyle": "http://www.example.com/*9943" } ] }
  }
}
```
```json
{
  "Image": {
    "Title": [ {"anything-but":  ["Pikachu", "Eevee"] } ]
  }
}
```
The syntax and semantics of Patterns are fully specified
in [Patterns in Quamina](PATTERNS.md).

The structure of a Pattern, in terms of field names
and nesting, must be the same as the structure of the Event
to be matched. The field values are always given
as an array; if any element of the array matches
the value in the Event, the match is good. If the
field in the Event is array-valued, matching is true
if the intersection of the arrays is non-empty.

Fields which are not mentioned in the Pattern will
be assumed to match, but all fields mentioned must match. So the
semantics are effectively an OR on each field's values,
but an AND on the field names.

Note that the `shellstyle` Patterns can include only
one `*` character. The architecture probably allows
support for a larger subset of regular expressions,
eventually.

The `"exists":true` and `"exists":false` patterns
have corner cases; details are covered in
[Patterns in Quamina](PATTERNS.md).

Number matching is weak - the number has to appear
exactly the same in the Pattern and the Event. I.e.,
Quamina doesn't know that 35, 35.000, and 3.5e1 are the
same number. There's a fix for this in the code which
is not yet activated because it causes a
significant performance penalty, so the API needs to
be enhanced to only ask for it when you need it.

The syntax and semantics of Patterns is described
more fully in [Patterns in Quamina](PATTERNS.md).

## Flattening and Matching

The first step in finding matches for an Event is
“flattening” it, which is to say turning it
into a list of pathname/value pairs called Fields. Quamina
defines a `Flattener` interface type and has a built-in
`Flattener` for JSON.

Note that should you wish to process Events
in a format other than JSON, you can implement
the `Flattener` interface yourself.

## Data Errors
**Note**: Both Patterns and Events are required to be
RFC 8259-conforming JSON. In particular, field names and
values in both Patterns and Events must be valid UTF-8.
Unescaped characters smaller than 0x1F (illegal per JSON),
and bytes with value greater than 0XF4 (can't occur
in correctly composed UTF-8) are rejected by the APIs.

In some cases, JSON errors in Events may not be caught by
the Matching APIs. The Flattener works hard to avoid processing
areas of the Event that cannot match any of the provided
Patterns and, in skipping over them, may miss certain errors.

## APIs
### Control APIs
```go
func New(opts ...Option) (*Quamina, error)

func WithMediaType(mediaType string) Option
func WithFlattener(f Flattener) Option
func WithPatternDeletion(b bool) Option
func WithPatternStorage(ps LivePatternsState) Option
```
For example:

```go
q, err := quamina.New(quamina.WithMediaType("application/json"))
```
The meanings of the `Option` functions are:

`WithMediaType`: In the futue, Quamina will support
Events not just in JSON but in other formats such as
Avro, Protobufs, and so on. This option will make sure
to invoke the correct Flattener. At the moment, the only
supported value is `application/json`, the default.

`WithFlattener`: Requests that Quamina flatten Events with
the provided (presumably user-written) Flattener.

`WithPatternDeletion`: If true, arranges that Quamina
allows Patterns to be deleted from an instance. This is
not free; it can incur extra costs in memory and
occasional stop-the-world Quamina rebuilds. (We plan
to improve this.)

`WithPatternStorage`: If you provide an argument that
supports the `LivePatternStorage` API, Quamina will
use it to maintain a list of which Patterns have currently
been added but not deleted. This could be useful if you
wanted to rebuild Quamina instances for sharded
processing or after a system failure. ***Note: Not
yet implemented.***

### Data APIs

```go
func (q *Quamina) AddPattern(x X, patternJSON string) error
```
The first argument identifies the Pattern and will be
returned by Quamina when asked to match against Events.
X is defined as `any`.

The Pattern is provided in the second argument string which
must be a JSON object as exemplified above in this document.

The `error` return is used to signal invalid Pattern
structure, which could be bad UTF-8 or malformed JSON
or leaf values which are not provided as arrays.

As many Patterns as desired can be added to a Quamina
instance. More than one Pattern can be added with the
same `X` identifier.

The `AddPattern` call is single-threaded; if multiple
threads call it, they will block and execute sequentially.
```go
func (q *Quamina) DeletePatterns(x X) error
```
After calling this API, no list of matches from
`AddPattern` will include the `X` value specified
in the argument.

The `error` return value is nil unless there was an
internal failure of Quamina’s storage system.
```go
func (q *Quamina) MatchesForEvent(event []byte) ([]X, error)
```
The `error` return value is nil unless there was an
error in the encoding of the Event.

The `[]X` return slice may be empty if none of the Patterns
match the provided Event.

### Concurrency

A single Quamina instance can not safely be used by
multiple goroutines at the same time. However, the
underlying data structure is designed for concurrent
access and the `Copy` API is provided to support this.

```go
func (q *Quamina) Copy() *Quamina
```

This generates a copy of the target instance. Such
copies may safely run in parallel in different
goroutines executing any combination of
`MatchesForEvent()`, `AddPattern()`, and
`DeletePattern()` calls. There is a significant
performance penalty if a high proportion of these
calls are `AddPattern()`.

Note that the `Copy()` API is somewhat expensive, and
that a Quamina instance exhibits “warm-up” behavior,
i.e. the performance of `MatchesForEvent()` improves
slightly upon repeated calls, especially over the
first few calls. The conclusion is that, for maximum efficiency, once
you’ve created a Quamina instance, whether through
`New()` or `Copy()`, keep it around and run as many
Events through it as is practical.

### `AddPattern()` Performance

In **most** cases, tens of thousands of Patterns per second can
be added to a Quamina instance; the in-memory data structure will
become larger, but not unreasonably so. The amount of of
available memory is the only significant limit to the
number of patterns an instance can carry.

The exception is `shellstyle` Patterns. Adding many of these
can rapidly lead to degradation in elapsed time and memory
consumption, at a rate which is uneven but at worst
O(2<sup>N</sup>) in the number of patterns. A fuzz test
which adds random 5-letter words with a `*` at a random
location slows to a crawl after 30 or so `AddPattern()`
calls, with the Quamina instance having many millions of
states. Note that such instances, once built, can still
match Events at high speeds.

This is after some optimization. It is possible there is a
bug such that automaton-building is unduly wasteful but it
may remain the case that adding this flavor of Pattern is
simply not something that can be done at large scale.

### `MatchesForEvent()` Performance

I used to say that the performance of
`MatchesForEvent` was O(1) in the number of
Patterns. That’s probably a reasonable way to think
about it, because it’s *almost* right.

To be correct, the performance is a little worse than
O(N) where N is the average number of unique fields in an
event that are used in one or more Patterns that have been
added to the Quamina instance.

For example, suppose you have a list of 50,000 words, and
you add a Pattern for each, of the form:
```json
{"word": ["one of the words"]}
```
The performance in matching events should be about the same
for one word or 50,000, with some marginal loss following on
growth in the size of the necessary data structures.

However, adding another pattern that looks like the
following, in the case that the Events have a top-level
`number` field, would decrease the performance by a factor of
roughly 2:
```json
{"number": [11, 22, 33]}
```
After that, adding a few thousand more `"number"` patterns shouldn’t
decrease the performance observably.

A word of explanation: Quamina compiles the Patterns into a
somewhat-decorated automaton and uses that to find matches in
Events. For Quamina to work, the incoming Events must be flattened
into a list of pathname/value pairs and sorted. This process
exceeds 50% of execution time, and is optimized by discarding
any fields that do not appear in one or more of the Patterns
added to Quamina. This code is optimized and in many cases
can avoid processing every byte of the event. It’s complicated.

Once the list of fields which might match a Pattern is
extracted, the cost of traversing the automaton
is at most N, the number of fields left after discarding.

So, adding a new Pattern that only mentions fields which are
already mentioned in previous Patterns is effectively free,
i.e. O(1) in terms of run-time performance.

### Further documentation

There is a series of blog posts entitled
[Quamina Diary](https://www.tbray.org/ongoing/What/Technology/Quamina%20Diary/)
that provides a detailed discussion of the design decisions
at a length unsuitable for in-code comments.

### Name

From Wikipedia: Quamina Gladstone (1778 – 16 September
1823), most often referred to simply as Quamina, was a
Guyanese slave from Africa and father of Jack Gladstone.
He and his son were involved in the Demerara rebellion
of 1823, one of the largest slave revolts in the British
colonies before slavery was abolished.

### Credits

@timbray: v0.0 and patches.

@jsmorph: `Pruner` and concurrency testing.

@embano1: CI/CD and project structure.

@yosiat: Flattening optimization.
