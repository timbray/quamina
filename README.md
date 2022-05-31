# Quamina

[![Tests](https://github.com/timbray/quamina/actions/workflows/go-unit-tests.yaml/badge.svg)](https://github.com/timbray/quamina/actions/workflows/go-unit-tests.yaml)
[![Latest Release](https://img.shields.io/github/release/timbray/quamina.svg?logo=github&style=flat-square)](https://github.com/timbray/quamina/releases/latest)
[![codecov](https://codecov.io/gh/timbray/quamina/branch/main/graph/badge.svg?token=TC7MW723JO)](https://codecov.io/gh/timbray/quamina) 
[![Go Report Card](https://goreportcard.com/badge/github.com/timbray/quamina)](https://goreportcard.com/report/github.com/timbray/quamina)
[![timbray/quamina](https://img.shields.io/github/go-mod/go-version/timbray/quamina)](https://github.com/timbray/quamina)
[![Go Reference](https://pkg.go.dev/badge/github.com/timbray/quamina.svg)](https://pkg.go.dev/github.com/timbray/quamina)


### Fast pattern-matching library

**Quamina** implements a data type that has APIs to 
create an instance and add multiple **Patterns** to it, 
and then query data objects called **Events** to
discover  which of the patterns match 
the fields in the event.

Quamina [welcomes contributions](CONTRIBUTING.md).

### Status

As of late May 2022, Quamina has a lot of unit tests and
they're all passing.  It has a reasonable basis of
GitHub-based CI/CD working. We intend to press the
“release” button any day now, but for the moment 
we reserve the right to change APIs.

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
The structure of a Pattern, in terms of field names
and nesting, must be the same as the structure of the Event 
to be matched.  The field values are always given
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

Number matching is weak - the number has to appear 
exactly the same in the pattern and the event. I.e.,
Quamina doesn't know that 35, 35.000, and 3.5e1 are the
same number. There's a fix for this in the code which 
is not yet activated because it causes a 
significant performance penalty, so the API needs to
be enhanced to only ask for it when you need it.

## Flattening and Matching

The first step in finding matches for an Event is 
“flattening” it, which is to say turning it 
into a list of pathname/value pairs called Fields. Quamina 
defines a `Flattener` interface type and has a built-in
`Flattener` for JSON.

`Flattener` implementations in general will have
internal state and thus not be thread-safe.

Note that should you wish to process events 
in a format other than JSON, you can implement 
the `Flattener` interface yourself.

## APIs
**Note**: In all the APIs below, field names and values in both
Patterns and Events must be valid UTF-8.  Unescaped characters
smaller than 0x1F (illegal per JSON), and bytes with value
greater than 0XF4 (can't occur in correctly composed UTF-8)
are rejected by the APIs.
### Control APIs
```go
func New(...Option) (*Quamina, error)

func WithMediaType(mediaType string) Option
func WithFlattener(f Flattener) Option
func WithPatternDeletion(b bool) Option
func WithPatternStorage(ps LivePatternsState) Option 
```
For example:

```go
q, err := quamina.New(quamina.New(quamina.WithMediaType("application/json")))
```
The meanings of the `Option` functions are:

`WithMediaType`: In the futue, Quamina will support 
Events not just in JSON but in other formats such as
Avro, Protobufs, and so on. This option will make sure
to invoke the correct Flattener. At the moment, the only
supported value is `application/json`, the default.

`WithFlattener`: Requests that Quamina flatten events with
the provided (presumably user-written) Flattener.

`WithPatternDeletion`: If true, arranges that Quamina
allows Patterns to be deleted from an instance. This is 
not free; it can incur extra costs in memory and 
occasional stop-the-world Quamina rebuilds. (We plan
to improve this.)

`WithPatternStorage`: If you provide an argument that
supports the `LivePatternStorage` API, Quamina will
use it to 
maintain a list of which patterns have currently been
added but not deleted.  This could be useful if you
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

The Pattern must be provided as a string which is a 
JSON object as exemplified above in this document.

The `error` return is used to signal invalid Pattern
structure, which could be bad UTF-8 or malformed JSON 
or leaf values which are not provided as arrays.

As many Patterns as desired can be added to a Quamina
instance. 

The `AddPattern` call is single-threaded; if multiple
threads call it, they will block and execute sequentially.

```go
func (q *Quamina) MatchesForEvent(event []byte) ([]X, error)
```

The `error` return value is nil unless there was an
error in the encoding of the Event.

The `[]X` return slice may be empty if none of the Patterns
match the provided Event. 

A single Quamina instance is not thread-safe. But 
instances can share the underlying data structures
in a safe way.

```json
func (q *Quamina) Copy() *Quamina
```
This generates a copy of of the target instance 
which may be used in parallel on another thread, 
while sharing the underlying data structure. Many
instances can execute `MatchesForEvent()` calls 
concurrently, even while one or more of them are 
also executing `AddPattern()`.  There is a 
significant performance penalty if there is a high 
rate of `AddPattern` in parallel with matching.

### Performance

I used to say that the performance of 
`MatchesForEvent` was `O(1)` in the number of 
Patterns. While that’s probably the right way to think
about it, it’s not *quite* true,
as it varies somewhat as a function of the number of 
unique fields that appear in all the patterns that have 
been added to Quamina, but still remains sublinear 
in that number. 

A word of explanation: Quamina compiles the
patterns into a somewhat-decorated automaton and uses 
that to find matches in events; the matching process is 
O(1) in the number of patterns.

However, for this to work, the incoming event must be
flattened into a list of pathname/value pairs and 
sorted.  This process exceeds 50% of execution time, 
and is optimized by discarding any fields that
do not appear in one or more of the patterns added
to Quamina. Thus, adding a new pattern that only
mentions fields mentioned in previous patterns is
effectively free i.e. `O(1)` in terms of run-time 
performance.

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