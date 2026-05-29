# gh-ost Binlog Validation Error Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix gh-ost plan-save validation so binlog status access failures are not reported as disabled binary logging.

**Architecture:** Keep the change inside `backend/component/ghost`. Add an explicit failure reason to `BinlogValidationResult`; set it in each failing validator branch; make the user-facing formatter switch on that reason instead of inferring from booleans.

**Tech Stack:** Go, `github.com/pkg/errors`, `github.com/stretchr/testify/require`, existing Bytebase gh-ost validation package.

---

## File Structure

- Modify: `backend/component/ghost/validator.go`
  - Owns gh-ost binlog prerequisite validation and conversion of validation failures into plan-check messages.
- Create: `backend/component/ghost/validator_test.go`
  - Table-driven unit tests for `GetUserFriendlyError()` messages and fallback behavior.

---

### Task 1: Add Failing Formatter Tests

**Files:**
- Create: `backend/component/ghost/validator_test.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/component/ghost/validator_test.go` with:

```go
package ghost

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinlogValidationResultGetUserFriendlyError(t *testing.T) {
	tests := []struct {
		name        string
		result      *BinlogValidationResult
		wantTitle   string
		wantContent string
	}{
		{
			name: "valid result",
			result: &BinlogValidationResult{
				Valid: true,
			},
			wantTitle:   "",
			wantContent: "",
		},
		{
			name: "binlog status inaccessible",
			result: &BinlogValidationResult{
				Valid:         false,
				BinlogEnabled: false,
				Error:         errors.New("cannot access binary logs - ensure user has REPLICATION CLIENT privilege"),
				FailureReason: binlogStatusInaccessible,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Cannot access binary log status. Ensure the Bytebase admin user has REPLICATION CLIENT privilege.",
		},
		{
			name: "binary logging disabled",
			result: &BinlogValidationResult{
				Valid:         false,
				BinlogEnabled: false,
				Error:         errors.New("binary logging is not enabled on this MySQL instance"),
				FailureReason: binlogDisabled,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Binary logging is not enabled on this MySQL instance.",
		},
		{
			name: "missing replication privilege",
			result: &BinlogValidationResult{
				Valid:             false,
				BinlogEnabled:     true,
				HasPrivilege:      false,
				MissingPrivileges: []string{"REPLICATION SLAVE"},
				Error:             errors.New("user does not have REPLICATION SLAVE privilege required for gh-ost"),
				FailureReason:     missingReplicationPrivilege,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Database user is missing required privilege: REPLICATION SLAVE\nPlease grant REPLICATION SLAVE or an equivalent replication privilege to the Bytebase admin user.",
		},
		{
			name: "unsupported binlog format",
			result: &BinlogValidationResult{
				Valid:         false,
				BinlogEnabled: true,
				HasPrivilege:  true,
				BinlogFormat:  "statement",
				Error:         errors.New("binlog_format is statement, but gh-ost requires ROW or MIXED format"),
				FailureReason: unsupportedBinlogFormat,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Current binlog_format is statement, but gh-ost requires ROW or MIXED format.\nPlease change it with:\nSET GLOBAL binlog_format='ROW'",
		},
		{
			name: "generic validation query failure",
			result: &BinlogValidationResult{
				Valid:         false,
				BinlogEnabled: false,
				Error:         errors.New("failed to check if binary logging is enabled: access denied"),
				FailureReason: validationQueryFailed,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Validation failed: failed to check if binary logging is enabled: access denied",
		},
		{
			name: "unknown invalid result falls back to error",
			result: &BinlogValidationResult{
				Valid: false,
				Error: errors.New("unexpected validator failure"),
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Validation failed: unexpected validator failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotContent := tt.result.GetUserFriendlyError()
			require.Equal(t, tt.wantTitle, gotTitle)
			require.Equal(t, tt.wantContent, gotContent)
		})
	}
}
```

- [ ] **Step 2: Run the focused test to verify it fails**

Run:

```bash
go test -v -count=1 ./backend/component/ghost -run TestBinlogValidationResultGetUserFriendlyError
```

Expected: FAIL because `FailureReason`, `binlogStatusInaccessible`, `binlogDisabled`, `missingReplicationPrivilege`, `unsupportedBinlogFormat`, and `validationQueryFailed` are not defined.

- [ ] **Step 3: Commit the failing tests**

Run:

```bash
git add backend/component/ghost/validator_test.go
git commit -m "test: cover gh-ost binlog validation messages"
```

Expected: commit succeeds with only the new test file staged.

---

### Task 2: Add Explicit Failure Reasons

**Files:**
- Modify: `backend/component/ghost/validator.go`
- Test: `backend/component/ghost/validator_test.go`

- [ ] **Step 1: Add the failure reason type and field**

In `backend/component/ghost/validator.go`, add this type and constants after the imports and before `BinlogValidationResult`:

```go
type binlogValidationFailureReason string

const (
	binlogStatusInaccessible   binlogValidationFailureReason = "binlog_status_inaccessible"
	binlogDisabled             binlogValidationFailureReason = "binlog_disabled"
	missingReplicationPrivilege binlogValidationFailureReason = "missing_replication_privilege"
	unsupportedBinlogFormat    binlogValidationFailureReason = "unsupported_binlog_format"
	validationQueryFailed      binlogValidationFailureReason = "validation_query_failed"
)
```

Then add this field to `BinlogValidationResult` after `Error error`:

```go
FailureReason binlogValidationFailureReason
```

- [ ] **Step 2: Set failure reasons in validator branches**

Update `ValidateBinlogAccess()` so each failing branch sets `FailureReason` before returning:

```go
if !canAccessBinlog {
	result.Valid = false
	result.FailureReason = binlogStatusInaccessible
	result.Error = errors.New("cannot access binary logs - ensure user has REPLICATION CLIENT privilege")
	slog.Error("binlog access validation failed: cannot access binary logs",
		slog.String("host", adminDataSource.GetHost()),
		slog.String("user", adminDataSource.GetUsername()))
	return result
}
```

```go
if err := row.Scan(&logBin); err != nil {
	result.Valid = false
	result.FailureReason = validationQueryFailed
	result.Error = errors.Wrap(err, "failed to check if binary logging is enabled")
	return result
}
```

```go
if !result.BinlogEnabled {
	result.Valid = false
	result.FailureReason = binlogDisabled
	result.Error = errors.New("binary logging is not enabled on this MySQL instance")
	return result
}
```

```go
if err != nil {
	result.Valid = false
	result.FailureReason = validationQueryFailed
	result.Error = errors.Wrap(err, "failed to check user grants")
	return result
}
```

```go
if err := rows.Err(); err != nil {
	result.Valid = false
	result.FailureReason = validationQueryFailed
	result.Error = errors.Wrap(err, "error reading grants")
	return result
}
```

```go
if !result.HasPrivilege {
	result.Valid = false
	result.FailureReason = missingReplicationPrivilege
	result.MissingPrivileges = append(result.MissingPrivileges, "REPLICATION SLAVE")
	result.Error = errors.New("user does not have REPLICATION SLAVE privilege required for gh-ost")
	slog.Error("missing REPLICATION SLAVE privilege",
		slog.String("host", adminDataSource.GetHost()),
		slog.String("user", adminDataSource.GetUsername()),
		slog.Any("grants", result.CurrentGrants))
	return result
}
```

```go
if err := row.Scan(&result.BinlogFormat); err != nil {
	result.Valid = false
	result.FailureReason = validationQueryFailed
	result.Error = errors.Wrap(err, "failed to check binlog format")
	return result
}
```

```go
if strings.ToUpper(result.BinlogFormat) == "STATEMENT" {
	result.Valid = false
	result.FailureReason = unsupportedBinlogFormat
	result.Error = errors.Errorf("binlog_format is %s, but gh-ost requires ROW or MIXED format", result.BinlogFormat)
	return result
}
```

- [ ] **Step 3: Replace formatter inference with a reason switch**

Replace the body of `GetUserFriendlyError()` after `title := "gh-ost migration prerequisites not met"` with:

```go
switch r.FailureReason {
case binlogStatusInaccessible:
	return title, "Cannot access binary log status. Ensure the Bytebase admin user has REPLICATION CLIENT privilege."
case binlogDisabled:
	return title, "Binary logging is not enabled on this MySQL instance."
case missingReplicationPrivilege:
	missingPrivileges := strings.Join(r.MissingPrivileges, ", ")
	if missingPrivileges == "" {
		missingPrivileges = "REPLICATION SLAVE"
	}
	return title, fmt.Sprintf("Database user is missing required privilege: %s\n", missingPrivileges) +
		"Please grant REPLICATION SLAVE or an equivalent replication privilege to the Bytebase admin user."
case unsupportedBinlogFormat:
	return title, fmt.Sprintf("Current binlog_format is %s, but gh-ost requires ROW or MIXED format.\n", r.BinlogFormat) +
		"Please change it with:\n" +
		"SET GLOBAL binlog_format='ROW'"
case validationQueryFailed:
	if r.Error != nil {
		return title, fmt.Sprintf("Validation failed: %v", r.Error)
	}
	return title, "Validation failed"
}

if r.Error != nil {
	return title, fmt.Sprintf("Validation failed: %v", r.Error)
}
return title, "Unknown validation error occurred"
```

This ensures `BinlogEnabled=false` no longer causes the disabled-binlog message unless the validator explicitly sets `binlogDisabled`.

- [ ] **Step 4: Format the changed Go files**

Run:

```bash
gofmt -w backend/component/ghost/validator.go backend/component/ghost/validator_test.go
```

Expected: no command output.

- [ ] **Step 5: Run the focused test**

Run:

```bash
go test -v -count=1 ./backend/component/ghost -run TestBinlogValidationResultGetUserFriendlyError
```

Expected: PASS.

- [ ] **Step 6: Commit the implementation**

Run:

```bash
git add backend/component/ghost/validator.go backend/component/ghost/validator_test.go
git commit -m "fix: classify gh-ost binlog validation errors"
```

Expected: commit succeeds with validator and test changes.

---

### Task 3: Verify Package and Required Go Checks

**Files:**
- Verify: `backend/component/ghost/validator.go`
- Verify: `backend/component/ghost/validator_test.go`

- [ ] **Step 1: Run the package tests**

Run:

```bash
go test -v -count=1 ./backend/component/ghost
```

Expected: PASS for existing gh-ost package tests plus the new formatter test.

- [ ] **Step 2: Run repository lint**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: PASS. If it reports issues, fix only issues caused by this change, then rerun the same command until it reports no issues.

- [ ] **Step 3: Run repository lint auto-fix if lint reports fixable issues**

Only if Step 2 reports fixable issues, run:

```bash
golangci-lint run --fix --allow-parallel-runners
golangci-lint run --allow-parallel-runners
```

Expected: final lint run passes. Review any modified files before committing.

- [ ] **Step 4: Commit verification fixes if any were needed**

If Step 3 changed files, run:

```bash
git add backend/component/ghost/validator.go backend/component/ghost/validator_test.go
git commit -m "chore: address gh-ost validator lint"
```

Expected: commit succeeds only if lint or formatting produced additional changes. If no files changed, skip this step.

- [ ] **Step 5: Record final status**

Run:

```bash
git status --short
```

Expected: no unstaged or uncommitted changes unless unrelated pre-existing files were already dirty.
