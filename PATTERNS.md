# Patterns in Quamina

**Patterns** are added to Quamina instances using the 
`AddPattern` API.  This document specifies the syntax and
semantics of Patterns.

## Fields

Patterns exist to match **Fields** in incoming **Events**. At
launch, Quamina supports only JSON syntax for Events. An 
Event **MUST** be equivalent to a JSON Object in the
sense that it consists of an unordered set of members,
each identified by a name which is string composed of
Unicode characters.

As in JSON, the allowed member values may be strings, 
numbers, the literals `true`, `false`, and `null`, arrays, 
and objects.  We refer to values which are neither arrays
nor objects as **Leaf** values.  

A **Field** is the combination of a Leaf value and a 
**Path**, a list of strings which are the Field names 
that must be traversed to reach it from the Event root.

For example, in the Event
```json
{"alpha": {"beta":  1}}
```
There is only one Field whose Leaf value is `1`
and whose Path is `"alpha","beta"`.

Paths omit arrays.  So in the Event
```json
{"alpha": [ {"beta":  [1, 2]}, {"beta":  [3, 4]} ] }
```
The Path for the Leaf value `1` is still `"alpha","beta"`

### Pattern Syntax and Semantics
A Pattern **MUST** be a JSON object all of whose Leaf
values **MUST** be in arrays.

To match a Field in an Event, the Pattern **MUST** contain
an exactly-matching Path whose value **MUST** be an array 
which contains either the Field’s Leaf value or an
**Extended Pattern**

Thus, the following Pattern would match both JSON events above:
```json
{"alpha": {"beta":  [1]}}
```

## Extended Patterns
An **Extended Pattern** **MUST** be a JSON object containing
a single field whose name is called the **Pattern Type**. 

### Exists Pattern

The Pattern Type of an Exists Pattern is `exists` and its
value **MUST** be `true` or `false`.  Here
are two Exists Patterns that would match the Events above:
```json
{"alpha": {"beta": [ {"exists": true} ]}}
{"alpha": {"gamma": [ {"exists": false} ]}}
```

If a Field in a Pattern contains an Exists Pattern, it
**MUST NOT** contain any other values.

### Anything-But Pattern

The Pattern Type of an Anything-But Pattern is
`anything-but` and its value **MUST** be an array
of strings.  It will match a string value which
is not equal to any of the strings in the array.

If a Field in a Pattern contains an Anything-But Pattern, 
it **MUST NOT** contain any other values.

### Shellstyle Pattern

The Pattern Type of a Shellstyle Pattern is `shellstyle` 
and its value **MUST** be a string which **MAY** contain
a single `*` (“star”) character. The star character 
functions exactly as the same character does in 
command-line processors which descend from Unix’s 
shell; i.e., matches the regular expression `.*`

Consider the following Event:
```json
{"img": "https://example.com/9943.jpg"}
```
The following Shellstyle Patterns would match it:
```json
{"img": [ {"shellstyle": "*.jpg"} ]}
{"img": [ {"shellstyle": "https://example.com/*"} ]}
{"img": [ {"shellstyle": "https://example.com/*.jpg"} ]}
```
## EventBridge Patterns

Quamina’s Patterns are inspired by those offered by 
the AWS EventBridge service, as documented in
[Amazon EventBridge event patterns](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-event-patterns.html).

As of release 0.1.1, Quamina supports Exists and
Anything-But patterns but does not yet support AWS’s 
`numeric` or `prefix` patterns.  Note that a 
Shellstyle Pattern with a trailing `*` is equivalent
to a `prefix` pattern.



