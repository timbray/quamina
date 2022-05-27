# Contributing to `quamina`

## Basics

Quamina is hosted in this GitHub repository 
at `github.com/timbray/quamina` and welcomes 
contributions.

Typically, the first step in making a change is to 
raise an Issue to allow for discussion of the idea. 
This is important because possibly Quamina already
does what you want, in which case perhaps what’s 
needed is a documentation fix. Possibly the idea 
has been raised before but failed to convince Quamina’s
maintainers. (Doesn’t mean it won’t find favor now;
times change.)

Assuming there is agreement that a change in Quamina
is a good idea, the mechanics of forking the repository,
committing changes, and submitting a pull request are
well-described in many places; there is nothing 
unusual about Quamina.

### Code Style

The coding style suggested by the Go community is 
used in Quamina. See the
[style doc](https://github.com/golang/go/wiki/CodeReviewComments) for details.

Try to limit column width to 120 characters for both code and markdown documents
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
- `docs:` - Use for changes to the documentation

If your contribution falls into multiple categories, e.g. `api` and `pat` it
is recommended to break up your commits using distinct prefixes.

### Running Tests

In any repo subdirectory, `go test` runs unit tests
with all the defaults, which is a decent check for basic
sanity and correctness.

Running the following command in the root repository runs
all the available tests with race-detection enabled, and 
is an essential step before submitting any changes:

```shell
go test -race -v -count 1 ./...
```

The following command runs the Go linter; submissions 
need to be free of lint errors.

```shell
golangci-lint run  
```

At the moment we don’t have a script for running this 
in all the Quamina subdirectories so you’ll have to do
this by hand.  `golangci-lint` has a home page with
instructions for installing it.

## Reporting Bugs and Creating Issues

When opening a new issue, try to roughly follow the commit message format
conventions above.