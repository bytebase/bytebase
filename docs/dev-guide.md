# Code Review Guide

_Note: also applies to [bytebase.com](https://github.com/bytebase/bytebase.com)._

1. [Google's Code Review Guideline](https://google.github.io/eng-practices/).
1. Effective communication. Expect responses for **up to one business day**.
   1. [Re-requesting a review](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/about-pull-request-reviews#about-pull-request-reviews) with comment "PTAL".
   1. "LGTM".
   1. "I'm not around. Ask @xxx for reviews."
   2. "PTAL".
2. Authors are the drivers for discussions, resolving comments, and merging PRs. Discussions or PRs should not hang around.

# Style Guide

1. Follow the style guide before referring to existing code patterns and conventions.
1. Use American English for naming. For simplicity, avoid using "xxxList". The ambiguity between singular and plural forms often arises from inadequate design.
1. Prioritizing simplicity leading to effective and maintainable software.

## API

1. Follow [Google AIP](https://google.aip.dev/).
1. In cases of conflicts between AIP and the proto guide, follow AIP. For instance, the enum name should be `HELLO` instead of `TYPE_HELLO`.

## Go

1. https://google.github.io/styleguide/go/

## TypeScript

1. https://google.github.io/styleguide/tsguide.html
1. https://google.github.io/styleguide/jsguide.html
1. [Frontend Style Guide](fe-style-guide.md).

## Database Schema

1. [Schema Update Guide](schema-update-guide.md)

## Writing

1. [Writing Guide](writing-guide.md).
