# Oracle Query Span Omni Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate Oracle query span extraction from the ANTLR parser path to the omni Oracle AST path while preserving the existing query span fixture behavior.

**Architecture:** Add a package-internal omni golden harness that compares the new extractor against `backend/plugin/parser/plsql/test-data/query_span.yaml`. Implement an Oracle omni query span extractor that reuses the existing metadata lookup and column-source semantics where practical, then switch the public `GetQuerySpan` registration to the omni path. Keep the legacy ANTLR extractor available during the migration for existing access-table tests and as a reference until follow-up cleanup.

**Tech Stack:** Go, `github.com/bytebase/omni/oracle/ast`, `github.com/bytebase/omni/oracle/parser`, existing Bytebase parser base query span model.

### Task 1: Oracle Omni Golden Harness

**Files:**
- Create: `backend/plugin/parser/plsql/query_span_omni_parity_test.go`

**Step 1: Write the failing test**

Add a test that loads `backend/plugin/parser/plsql/test-data/query_span.yaml`, builds the same mock metadata context as `TestGetQuerySpan`, and calls `newOmniQuerySpanExtractor(...).getOmniQuerySpan(...)`.

**Step 2: Run test to verify it fails**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/plsql -run '^TestOracleOmniQuerySpanGoldenHarness$'`

Expected: FAIL because `newOmniQuerySpanExtractor` is undefined.

### Task 2: Upgrade Omni

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Update dependency**

Run: `go get github.com/bytebase/omni@v0.0.0-20260528064644-6ef6c2c2b826`

**Step 2: Verify parser coverage**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/plsql -run '^TestOracleOmniQuerySpanMigrationProbe$'`

Expected: PASS, including inline external table support through the upgraded omni parser.

### Task 3: Implement Oracle Omni Extractor

**Files:**
- Create: `backend/plugin/parser/plsql/query_span_extractor_omni.go`

**Step 1: Add minimal extractor scaffold**

Implement `newOmniQuerySpanExtractor` and `getOmniQuerySpan` with parsing, statement classification, access-table collection, and select result extraction.

**Step 2: Run golden harness**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/plsql -run '^TestOracleOmniQuerySpanGoldenHarness$'`

Expected: PASS.

### Intentional Behavior Changes

- `JSON_TABLE`, `XMLTABLE`, `CONTAINERS`, and inline external table are supported by the omni query span extractor.
- `PIVOT`, `UNPIVOT`, `MODEL`, and `TABLE(collection)` are intentionally typed unsupported for this migration. In particular, ANTLR could return a successful approximate span for some `UNPIVOT` and `MODEL` queries, but it did not model their transformation semantics accurately. The omni path should fail explicitly until those output columns can be represented correctly.

### Task 4: Switch Public Oracle Query Span Entry

**Files:**
- Modify: `backend/plugin/parser/plsql/query_span.go`

**Step 1: Change registration**

Replace the public Oracle `GetQuerySpan` implementation with the omni extractor path.

**Step 2: Run public fixture test**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/plsql -run '^TestGetQuerySpan$'`

Expected: PASS.

### Task 5: Final Verification

**Files:**
- All modified Go files

**Step 1: Format**

Run: `gofmt -w backend/plugin/parser/plsql/query_span_omni_parity_test.go backend/plugin/parser/plsql/query_span_extractor_omni.go backend/plugin/parser/plsql/query_span.go`

**Step 2: Package tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/plsql`

Expected: PASS.

**Step 3: Lint**

Run: `golangci-lint run --allow-parallel-runners`

Expected: `0 issues.`

**Step 4: Auto-fix lint**

Run: `golangci-lint run --fix --allow-parallel-runners`

Expected: `0 issues.`

**Step 5: Build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`

Expected: PASS.
