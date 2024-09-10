# Patterns in Quamina

**Patterns** are added to Quamina instances using the
`AddPattern` API. This document specifies the syntax and
semantics of Patterns.

The discussion of JSON constructs in this document uses
the terminology specified in [RFC 8259](https://www.rfc-editor.org/rfc/rfc8259.html).

## Fields

Patterns exist to match **Fields** in incoming **Events**. At
launch, Quamina supports only JSON syntax for Events. An
Event **MUST** be equivalent to a JSON Object in the
sense that it consists of an unordered set of members,
each identified by a name which is string composed of
Unicode characters.

As in JSON, the allowed member values may be strings,
numbers, the literals `true`, `false`, and `null`, arrays,
and objects. We refer to values which are neither arrays
nor objects as **Leaf** values.

A **Field** is the combination of a Leaf value and a
**Path**, a list of strings which are the Field names
that must be traversed to reach it from the Event root.

For example, in the Event
```json
{"alpha": {"beta": 1}}
```
There is only one Field whose Leaf value is `1`
and whose Path is `"alpha","beta"`.

Paths omit arrays. So in the Event
```json
{"alpha": [ {"beta": [1, 2]}, {"beta": [3, 4]} ] }
```
The Path for the Leaf value `1` is still `"alpha","beta"`

### Pattern Syntax and Semantics
A Pattern **MUST** be a JSON object all of whose Leaf
values **MUST** be in arrays.

Note that a Field in an Event may have multiple Leaf Values
if the Field's value is an array.

To match a Field in an Event, the Pattern **MUST** contain
an exactly-matching Path whose value **MUST** be an array
which contains either an Event Field Leaf value or an
**Extended Pattern** which matches an Event Field Leaf
value.

Thus, the following Pattern would match both JSON events above:
```json
{"alpha": {"beta": [1]}}
```

### Numeric Values

Quamina can match numeric values with precision and range exactly the same as that provided by 
Go's `float64` data type, which is said to conform to IEE 754 `binary64`.

## Extended Patterns
An **Extended Pattern** **MUST** be a JSON object containing
a single field whose name is called the **Pattern Type**.

### Prefix Pattern

The Pattern Type of a Prefix Pattern is `prefix` and its value
**MUST** be a string.

The following event:

```json
{"a": "alpha"}
```

would be matched by this Prefix Pattern:

```json
{"a": [ { "prefix":  "al" } ] }
```

### Exists Pattern

The Pattern Type of an Exists Pattern is `exists` and its
value **MUST** be `true` or `false`. Here
are two Exists Patterns that would match the Events above:
```json
{"alpha": {"beta": [ {"exists": true} ]}}
{"alpha": {"gamma": [ {"exists": false} ]}}
```

If a Field in a Pattern contains an Exists Pattern, it
**MUST NOT** contain any other values.

Exists Patterns currently only work on leaf nodes. That is to
say, given this event:

```json
{ "a": { "b": 1 } }
```

The following pattern will not match:

```json
{ "a": [ {"exists": true} ] }
```

We may be able to change this in future.

The case of empty arrays is interesting, both in Patterns and Events. Consider this event:

```json
{ "a": [] }
```

Then `"exists": true` does not match but `"exists": false` does.
I.e., only the first of the two sample patterns below matches.

```json
{ "a": [ { "exists": false } ] }
```
```json
{ "a": [ { "exists": true } ] }
```
This makes sense in the context of the leaf-node semantics; there
really is no value for the `"a"` field.

In Patterns, the following never matches any Event:

```json
{ "a": [] }
```

Once again, there is nothing in the array of candidate values in the Pattern that can match any value of an `"a"`
field in an Event.



### Anything-But Pattern

The Pattern Type of an Anything-But Pattern is
`anything-but` and its value **MUST** be an array
of strings. It will match a string value which
is not equal to any of the strings in the array.

If a Field in a Pattern contains an Anything-But Pattern,
it **MUST NOT** contain any other values.

### Wildcard Pattern

The Pattern Type of a Wildcard Pattern is `wildcard`
and its value **MUST** be a string which **MAY** contain
`*` (“star”) characters. The star character
functions exactly as the same character does in
command-line processors which descend from Unix’s
shell; i.e., matches the regular expression `.*`

Adjacent `*` characters are not allowed.

Consider the following Event:
```json
{"img": "https://example.com/9943.jpg"}
```
The following Wildcard Patterns would match it:
```json
{"img": [ {"wildcard": "*.jpg"} ] }
{"img": [ {"wildcard": "https://example.com/*"} ] }
{"img": [ {"wildcard": "https://example.com/*.jpg"} ] }
{"img": [ {"wildcard": "https://example.*/*.jpg"} ] }
```

If it is desired to match the actual character "*", it may be “escaped”
with backslash, "\". For example, consider the following Event:

```json
{"example-regex": "a**\\.b"}
```

The following Wildcard pattern would match it.

```json
{"example-regex": [ {"wildcard":  "a\\*\\*\\\\.b"}]}
```

Note that the "\" backslashes must be doubled to deal with the
fact that they are escape characters for JSON as well as for Quamina.

After a "\", the appearance of any character other than "*" or "\" is an error.

### Shellstyle Pattern

This is an earlier version of the Wildcard pattern, differing only that 
\-escaping the "*" and "\" characters is not supported.

### Equals-Ignore-Case Pattern

The Pattern Type of an Equals-Ignore-Case pattern is `equals-ignore-case`
and its value **MUST** be a string. Quamina attempts to match with
case folding in effect, as discussed in Section 3.13 of the Unicode
Standard. Quamina uses the case-folding mappings provided in the file
CaseFolding.txt in the Unicode Character Database to generate its mappings.
Note that case-folding is highly dependent on the specifics of the language
in use and in certain locales, this default mapping may not produce satisfactory
results, although results are good for ASCII and "simple" characters from
other alphabets.

## EventBridge Patterns

Quamina’s Patterns are inspired by those offered by
the AWS EventBridge service, as documented in
[Amazon EventBridge event patterns](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-event-patterns.html).

As of release 1.0, Quamina supports Exists and
Anything-But Patterns, but does not yet support any other
EventBridge patterns. Note that a
Wildcard Pattern with a trailing `*` is equivalent
to a `prefix` pattern.

