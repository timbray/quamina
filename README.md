# Quamina

### Fast pattern-matching library

Quamina provides APIs to create an interface called 
a **Matcher**,
add multiple **Patterns** to it, and then query JSON blobs
called **Events** to discover which of the patterns match 
the fields in the event.

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
support for a larger subset of regular expressions
but currently,  the testing for just the single -`*`
patterns is a bit lacking.

Number matching is weak - the number has to appear 
exactly the same in the pattern and the event. I.e.,
Quamina doesn't know that 35, 35.000, and 3.5e1 are the
same number.

There's a fix for this in the code which is commented
out because it causes a significant performance penalty.

## Flattening and Matching

The first step in finding matches for an Event is 
“flattening” it, which is to say turning it 
into a list of pathname/value pairs called Fields. Quamina 
defines a `Flattener` interface type and provides a 
JSON-specific implementation in the `FJ` type.

`Flattener` implementations in general will have
internal state and thus not be thread-safe.

The `MatchesForJSONEvent` API must create a new 
`FJ` instance for each event so that it 
can be thread-safe.  This works fine, but creating a 
new `FJ` instance is expensive enough to slow the 
rate at which events can be matched by 15% or so.

For maximum performance in matching JSON events, 
you should create your own `FJ` instance with the 
`NewFJ(Matcher)` method. You can then use 
`FJ.Flatten(event)` API to turn multiple successive
JSON events into `Field` lists and pass them to 
`Matcher`'s `MatchesForFields()` API, but `FJ` 
includes a convenience method `FlattenAndMatch(event)` 
which will call the `Matcher` for you.  As long as 
each thread has its own `Flattener` instance, 
everything will remain thread-safe.

Also note that should you wish to process events 
in a format other than JSON, you can implement 
the `Flattener` interface and use that to process 
events in whatever format into Field lists.

## APIs

**Note**: In all the APIs below, field names and values in both
Patterns and Events must be valid UTF-8.  Unescaped characters
smaller than 0x1F (illegal per JSON), and bytes with value
greater than 0XF4 (can't occur in correctly composed UTF-8)
are rejected by the APIs.

```go
type Matcher interface {
	AddPattern(x X, pat string) error
	MatchesForJSONEvent(event []byte) ([]X, error)
	MatchesForFields(fields []Field) []X
	DeletePattern(x X) error
}
```

Above are the operations provided by a Matcher. Quamina
includes an implementation called `CoreMatcher` which
implements `Matcher`.  In a forthcoming release it will
provider alternate implementations that offer extra
features.

```go
func NewCoreMatcher() *Matcher
```

Creates a new Matcher, takes no arguments.
```go
func (m *Matcher) AddPattern(x X, patternJSON string) error
```

The first argument identifies the Pattern and will be
returned by a Matcher when asked to match against Events.
X is currently `interface{}`. Should it be a generic now
that Go has them?

The Pattern must be provided as a string which is a 
JSON object as exemplified above in this document.

The `error` return is used to signal invalid Pattern
structure, which could be bad UTF-8 or malformed JSON 
or leaf values which are not provided as arrays.

As many Patterns as desired can be added to a Matcher
but at this time there is no capability of removing any.

The `AddPattern` call is single-threaded; if multiple
threads call it, they will block and execute sequentially.

```go
func (m *Matcher) MatchesForJSONEvent(event []byte) ([]X, error)
```

The `event` argument must be a JSON object encoded in
correct UTF-8. 

The `error` return value is nil unless there was an
error in the Event JSON.

The `[]X` return slice may be empty if none of the Patterns
match the provided Event. 

```go
func (m *Matcher) MatchesForFields([]Field) []X
```
Performs the functions of `MatchesForJSON` on an 
Event which has been flattened into a list of `Field`
instances.

`MatchesForJSONEvent` is thread-safe. Many threads may
be executing it concurrently, even while `AddPattern` is
also executing.

```go
func NewFJ(*Matcher) Flattener
```
Creates a new JSON-specific Flattener.
```go
func (fj *FJ) Flatten([]byte event) []Field
```
Transforms an event, which must be JSON object
encoded in UTF-8 into a list of `Field` instances.

```go
func (fj *FJ) FlattenAndMatch([]byte event) ([]X, error)
```
Utility function which combines the functions of the 
`FJ.Flatten` and `Matcher.MatchesForFields` APIs.

### Performance

I used to say that the performance of 
`MatchesForJSONEvent` was `O(1)` in the number of 
Patterns. While that’s probably the right way to think
about it, it’s not *quite* true,
as it varies somewhat as a function of the number of 
unique fields that appear in all the patterns that have 
been added to the matcher, but still remains sublinear 
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
to the matcher. Thus, adding a new pattern that only
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