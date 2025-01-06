# Regular Expressions in Quamina

**Regular Expressions** (hereinafter “regexps”) may appear in Quamina Regexp Patterns. 

## Syntax

The regexp syntax supported in Regexp Patterns are that specified in 
[RFC 9485](https://datatracker.ietf.org/doc/rfc9485/), 
*I-Regexp: An Interoperable Regular Expression Format*.

There is one important syntactic difference. The backslash character “\” commonly
used in regexp constructs for escaping metacharacters (as in `Stop\.`) and in such 
constructs such as `\P{Lu}`, is “~” in Quamina regexps.

“~” is used for this purpose because “\” is also used in Go string literals and
in JSON. Thus, complexity is added to unit testing and fragments such as `\\\\` and even
`\\\\\\\\` are regularly needed.  Conventional regexps may be turned into Quamina regexps
by replacing occurrences of “\” with “~” wherever “\” is being used as a metacharacter. If a
Quamina regexp needs to match the literal character “\”, it need not be escaped. For
example, the common Go-language syntax for representing whitespace characters in code can
be matched with the Quamina regexp `\[nrt]`, but the newline character, U+000A, would
be matched by the Quamina regexp `~n`.

When a regexp is used in a Quamina `addPattern()` call, an error is returned if the regexp
contains a syntax error or if it uses a regexp feature that is not yet supported in the
current release.

## Features

Regexps are being added to Quamina incrementally. The following list identifies the full
set of planned features; it is not in any particular order. Those that are supported in the
current release are bold-faced.

`.` : **single-character matcher**

`*` : zero-or-more matcher

`+` : one-or-more matcher

`?` : optional matcher

`{lo,hi}` : occurrence-count matcher

`()` : parenthetized sub-regexp

`~p{}` : Unicode property matcher

`~P{}` : Unicode property-complement matcher

`[]` : character-class matcher

`[^]` : complementary character-class matcher

`|` logical alternatives

## Semantics of “.”

In Quamina regexps, the `.` metacharacter matches any Unicode character whose code point is
among the *Unicode Scalars* as defined in Definition D76 in Section 3.9 of the Unicode Standard.
This is the range of codepoints between U+0000 - U+D7FF inclusive, and U+E000 - U+10FFFF
inclusive.

Put another way, `.` matches all of the Unicode code points except those defined in the Unicode Standard as “Surrogates”.
