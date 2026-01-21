# Contributing to Quamina

## Basics

Most of this document is concerned with the mechanics of raising issues
and posting Pull Requests to offer improvements to Quamina. Following
this, there is a section entitled **Developing** that describes
technology issues that potential contributors will face
and tools that might be helpful.

Quamina is hosted in this GitHub repository 
at `github.com/timbray/quamina` and welcomes 
contributions.

Typically, the first step in making a change is to 
raise an Issue to allow for discussion of the idea. 
This is important because possibly Quamina already
does what you want, in which case perhaps what’s 
needed is a documentation fix. Possibly the idea 
has been raised before but failed to convince Quamina’s
maintainers. (Doesn't mean it won’t find favor now;
times change.)

Assuming there is agreement that a change in Quamina
is a good idea, the mechanics of forking the repository,
committing changes, and submitting a pull request are
well-described in many places; there is nothing 
unusual about Quamina.

### Code Style

The code style produced by `gofmt`, with all the defaults, is 
used in Quamina. Most IDEs will take care of this for you
and the `golangci-lint` tool mentioned below will tell you
what's wrong and keep you from merging PRs with problems.

Try to limit column width to 120 characters for both code and Markdown documents
such as this one.

### Format of the Commit Message

We follow the conventions described in [How to Write a Git Commit
Message](http://chris.beams.io/posts/git-commit/).

Be sure to include any related GitHub issue references in the commit message,
e.g. `Closes: #<number>`.

The [`CHANGELOG.md`](./CHANGELOG.md) and release page uses **commit message
prefixes** for grouping and highlighting. A commit message that
starts with `[prefix:] ` will place this commit under the respective
section in the `CHANGELOG`.

The following example creates a commit referencing the `issue: 1234` and puts
the commit message in the `pat` `CHANGELOG` section:

```bash
git commit -s -m "pat: Add complex-number predicate" -m "Closes: #1234"
```

Currently the following prefixes are used:

- `api:` - Use for API-related changes
- `pat:` - Use for changes to the Quamina pattern language
- `chore:` - Use for repository related activities
- `fix:` - Use for bug fixes
- `kaizen:` - Use for code improvements or performance optimization
- `docs:` - Use for changes to the documentation

If your contribution falls into multiple categories, e.g. `api` and `pat` it
is recommended to break up your commits using distinct prefixes.

### Signing commits

Commits should be signed (not just the `-s` “signed off on”) with
any of the [styles GitHub supports](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits).
Note that you can use `git config` to arrange that your commits are
automatically signed with the right key.

### Directory structure

There really isn't one; all of the .go files are at the top level,
with the exception of the Unicode-table-generation code in the 
`code_gen` subdirectory.

This causes a mild problem in that new arrivals to Quamina have to
scroll down a lot to get past the filenames and see the README. If
this bothers you, propose a reorganization, none of us love the
current setup so minds are open.

### Running Tests

As with most Go projects `go test` runs unit tests
with all the defaults, which is a decent check for basic
sanity and correctness. They take less than 15 seconds to
run and we really want to keep that time short to encourage
people to run them all the time.

Running the following command in the root repository runs
all the available tests with race-detection enabled, and 
is an essential step before submitting any changes:

```shell
go test -race -v -count 1
```

The following command runs the Go linter; submissions 
need to be free of lint errors.

```shell
golangci-lint run  
```

### Rebuilding the Unicode Tables

Quamina's `ignore-case` patterns, and its regular-expression
property matching, rely on mappings found
in the generated source files `case_folding.go`
and `character_properties.go`. Quamina
includes a program called `code_gen` in the `code_gen/`
directory that generates them. It is very unlikely that
you will ever want to change them, but if you do, you
can't change them directly, you have to re-run the program,
whose source is in `code_gen/build_unicode_tables.go`. It’s
not pretty.

There is a `Makefile` whose only function is
to check the Unicode files and rebuild them if they are older
than three months, because a Unicode version release may
have added characters.

As a result, it is a good practice, sometime in the process
of building and submitting a PR, to type `make` at some
point, which will rebuild and re-run `code_gen`; that program
will display a message saying whether or not it rebuilt the
case-folding mappings. If it did rebuild those mappings, please
include the generated files in your commmit
and PR.

## Reporting Bugs and Creating Issues

When opening a new issue, try to roughly follow the commit message format
conventions above.

## Developing

### Automata

Quamina works by compiling the Patterns together into a Nondeterministic
Finite Automaton (NFA) which proceeds byte-at-a-time through the UTF-8-encoded
fields and values. NFAs are nondeterministic in the sense that a byte value
may cause multiple transitions to different states.

The general workflow, for some specific pattern type, is to write code to build 
an automaton that matches that type. Examples are the functions `makeStringFA()` in
`value_matcher.go` and `makeShellStyleAutomaton()` in `shell_style.go`. Then,
insert calls to the automaton builder in `value_matcher.go`, which is reasonably
straightforward code.  It takes care of merging new automata with existing ones
as required.

### Testing

A straightforward way to test a new feature is exemplified by `TestLongCase()` in
`shell_style_test.go`:

1. Make a `coreMatcher` by calling `newCoreMatcher()`
2. Add patterns to it by calling `addPattern()`
3. Make test data and examine matching behavior by calling `matchesForJSONEvent()`

We track test coverage carefully and while we don't have a target coverage number,
the majority of Quamina’s source files hit 100%. Don’t scrimp on unit tsting.

### Prettyprinting NFAs

NFAs can be difficult to build and to debug.  For this reason, code 
is provided in `prettyprinter.go` which produces human-readable NFA
representations.

To use the prettyprinter, make an instance with `newPrettyPrinter()` - the only
argument is a seed used to generate state numbers. Then, instead of calling
`addPattern()`, call `addPatternWithPrinter()`, passing your prettyprinter into
the automaton-building code. New automata are created by `valueMatcher` calls,
see `value_matcher.go`. Ensure that the prettyprinter is passed to your
automaton-matching code; an example of this is in the `makeShellStyleAutomaton()`
function.  Then, in your automaton-building code, use `prettyprinter.labelTable()`
to attach meaningful labels to the states of your automaton. Then at
some convenient point, call `prettyprinter.printNFA()` to generate the NFA printout;
real programmers debug with Print statements.

### Prettyprinter output

Rather than take space here to describe the prettyprinter output, read the blog
[here](https://www.tbray.org/ongoing/When/202x/2024/06/17/Epsilon-Love#p-5).

