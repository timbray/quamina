# Quamina

### Fast pattern-matching library

Quamina provides APIs to create an interface called 
a **Matcher**,
add multiple **patterns** to it, and then query JSON blobs
called **events** to discover which of the patterns match 
the fields in the event.

### Patterns

Consider the following JSON event, taken from the example
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

The following patterns would match it:

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
    "Thumbnail": {
      "Url": [ { "shellstyle": "http://*.example.com/*" } ]
    }
  }
}
```
The structure of the pattern, in terms of field names
and nesting, must be the same as the structure of the event 
to be matched.  The field values are always given
as an array; if any element of the array matches 
the value in the event, the match is good. If the
field in the event is array-valued, matching is true
if the intersection of the arrays is non-empty.

Fields which are not mentioned in the pattern will
be assumed to match, but all Fields mentioned must match. So the
semantics are effectively an OR on each field's values, 
but an AND on the field names.

Number matching is weak - the number has to appear 
exactly the same in the pattern and the event. I.e.,
Quamina doesn't know that 35, 35.000, and 3.5e1 are the
same number.

## APIs

**Note**: In all the APIs below, field names and values in both
patterns and events must be valid UTF-8.  Unescaped characters
smaller than 0x1F (illegal per JSON), and bytes with value
greater than 0XF4 (can't occur in correctly composed UTF-8)
are rejected by the APIs.

```go
func NewMatcher() *Matcher
```
Creates a new Matcher, takes no arguments.
```go
func (m *Matcher) AddPattern(x X, patternJSON string) error
```

The first argument identifies the pattern and will be
returned by a Matcher when asked to match against events.
X is currently `interface{}` and should become a generic
when Go has them.

The pattern must be provided as a string which is a 
JSON object as exemplified above in this document.

The `error` return is used to signal invalid pattern
structure, which could be bad UTF-8 or malformed JSON 
or leaf values which are not provided as arrays.

As many patterns as desired can be added to a Matcher
but at this time there is no capability of removing any.

The `AddPattern` call is single-threaded; if multiple
threads call it, they will block and execute sequentially.

```go
func (m *Matcher) MatchesForJSONEvent(event []byte) ([]X, error)
```

The `event` argument must be a JSON object encoded in
correct UTF-8. It would be 
easy to extend Matcher to handle other data formats; see the
`Flattener` interface and its implementation in `fj.go`.

The `error` return value is nil unless there was an
error in the event JSON.

The `[]X` return slice may be empty if none of the patterns
match the provided event. 

`MatchesForJSONEvent` is thread-safe. Many threads may
be executing it concurrently, even while `AddPattern` is
also executing.

### Performance

The performance of `MatchesForJSONEvent` is strongly
sublinear in the number of patterns. It’s not quite `O(1)`
as it varies somewhat as a function of the number of 
unique fields that appear in all the patterns that have 
been added to the machine, but remains sublinear in that 
number. 

A word of explanation is in order. Quamina compiles the
patterns into a somewhat-decorated DFA and uses that to
find matches in events; that DFA-based matching process is 
O(1) in the number of patterns.

However, for this to work, the incoming event must be
flattened into a list of pathname/value pairs and 
sorted.  This process exceeds 50% of execution time, 
and is optimized by discarding any fields that
do not appear in one or more of the patterns added
to the matcher. Thus, adding a new pattern that only
mentions fields mentioned in previous patterns is
effectively free in terms of run-time performance.

### Name

From Wikipedia: Quamina Gladstone (1778 – 16 September 
1823), most often referred to simply as Quamina, was a 
Guyanese slave from Africa and father of Jack Gladstone. 
He and his son were involved in the Demerara rebellion 
of 1823, one of the largest slave revolts in the British 
colonies before slavery was abolished.