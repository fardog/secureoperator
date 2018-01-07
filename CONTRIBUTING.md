# Contributing Guidelines

Contributions welcome!

**Before spending lots of time on something, ask for feedback on your idea
first!**

Please search issues and pull requests before adding something new to avoid
duplicating efforts and conversations.

This project welcomes non-code contributions, too! The following types of
contributions are welcome:

* **Ideas**: participate in an issue thread or start your own to have your voice
  heard.
* **Writing**: contribute your expertise in an area by helping expand the
  included docs.
* **Copy editing**: fix typos, clarify language, and improve the quality of the
  docs.
* **Formatting**: help keep docs easy to read with consistent formatting.

## Code of Conduct

This project is intended to be a safe, welcoming space for collaboration. All
contributors are expected to adhere to the [code of conduct][].

[code of conduct]: ./CODE_OF_CONDUCT.md

## Code and Documentation Style

Go provides a standard tool for formatting: [`gofmt`][]. Any code contributions
to this repository should be formatted with `gofmt` prior to a Pull Request
being filed. Additionally, we try to keep lines to under 80 characters where
possible, to avoid line wrapping in an editor.

Documentation should be written in [CommonMark][], a strongly-defined spec for
Markdown. Lines should be hard-wrapped to 80 characters, with exceptions for
code blocks as appropriate.

[gofmt]: https://golang.org/cmd/gofmt/
[CommonMark]: http://commonmark.org/

## Project Governance

Individuals making significant and valuable contributions are given
commit-access to the project to contribute as they see fit. This project is more
like an open wiki than a standard guarded open source project.

### Rules

There are a few basic ground-rules for contributors:

1. **No `--force` pushes** or modifying the Git history in any way.
2. **No commits directly to master** outside of correcting trivial typos, and
   release tasks such as version bumps.
3. **Pull requests** for all other changes.
4. **Tests must pass** before any changes are merged to master. We have a
   Continuous Integration setup to ensure tests are run against multiple
   versions of Go.
5. **Changes must support** the current and previous minor release within the
   same major version of Go, e.g. if version 1.9 is current, 1.8 and 1.9 must be
   supported. Supporting new major versions of Go will be exempt from this, e.g.
   when version 2.0 is adopted, only that version will be supported until 2.1 is
   released.

### Releases

Declaring formal releases remains the prerogative of the project maintainer.

### Changes to this arrangement

This is an experiment and feedback is welcome! This document may also be subject
to pull requests or changes by contributors where you believe you have something
valuable to add or change.

## Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

* (a) The contribution was created in whole or in part by me and I have the
  right to submit it under the open source license indicated in the file; or

* (b) The contribution is based upon previous work that, to the best of my
  knowledge, is covered under an appropriate open source license and I have the
  right under that license to submit that work with modifications, whether
  created in whole or in part by me, under the same open source license (unless
  I am permitted to submit under a different license), as indicated in the file;
  or

* (c) The contribution was provided directly to me by some other person who
  certified (a), (b) or (c) and I have not modified it.

* (d) I understand and agree that this project and the contribution are public
  and that a record of the contribution (including all personal information I
  submit with it, including my sign-off) is maintained indefinitely and may be
  redistributed consistent with this project or the open source license(s)
  involved.

## Attribution

This document is adapted from the following sources:

* [CONTRIBUTING.md][]
* [OPEN Open Source][openopen]
* [WebTorrent][]

[CONTRIBUTING.md]: https://github.com/ungoldman/CONTRIBUTING.md/
[openopen]: http://openopensource.org/
[WebTorrent]: https://github.com/webtorrent/webtorrent/blob/master/CONTRIBUTING.md
