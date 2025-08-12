# Dev Guide

1. [Google's Code Review Guideline](https://google.github.io/eng-practices/).
2. Authors are responsible for driving discussions, resolving comments, and promptly merging pull requests..

# Style Guide

1. Adhere to the style guide before consulting existing code patterns.
1. Use American English for naming conventions. Avoid "xxxList" for simplicity and to prevent singular/plural ambiguity stemming from poor design.
1. Prioritize simplicity for effective and maintainable software.

## API

1. Follow [Google AIP](https://google.aip.dev/).
1. When AIP and the proto guide conflict, AIP takes precedence. For example, use HELLO for enum names, not TYPE_HELLO.

## Go

1. https://google.github.io/styleguide/go/

## TypeScript

1. https://google.github.io/styleguide/tsguide.html
1. https://google.github.io/styleguide/jsguide.html
1. [Frontend Style Guide](fe-style-guide.md).

## Dev Environment Setup

### Prerequisites

- [Go](https://golang.org/doc/install)
- [pnpm](https://pnpm.io/installation)

### Steps

1. Pull source.

   ```bash
   git clone https://github.com/bytebase/bytebase
   ```

1. Create an external Postgres database on localhost.

   ```sql
   CREATE USER bbdev SUPERUSER;
   CREATE DATABASE bbdev;
   ```

1. Start backend.

   ```bash
   PG_URL=postgresql://bbdev@localhost/bbdev
   go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go && ./bytebase-build/bytebase --port 8080 --data . --debug
   ```

1. Start frontend (with live reload).

   ```bash
   pnpm --dir frontend i && pnpm --dir frontend dev
   ```

   Bytebase should now be running at http://localhost:3000 and change either frontend or backend code would trigger live reload.

### Tips

- Use [Code Inspector](https://en.inspector.fe-dev.cn/guide/start.html#method1-recommend) to locate
  frontend code from UI. Hold `Option + Shift` on Mac or `Alt + Shift` on Windows
