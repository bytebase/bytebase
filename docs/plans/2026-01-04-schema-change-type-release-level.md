# Move SchemaChangeType to Release Level Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move SchemaChangeType from per-file to release level in both store protos and v1 API protos.

**Architecture:** Update proto definitions to have type at release level instead of file level, create database migration to move existing data, update backend code to use release-level type, preserve revision-level type for history tracking.

**Tech Stack:** Protocol Buffers, Go, PostgreSQL, Buf

---

### Task 1: Update Store Proto (release.proto)

**Files:**
- Modify: `proto/store/store/release.proto`

**Step 1: Add type field to ReleasePayload**

In `proto/store/store/release.proto`, add the type field after `vcs_source`:

```proto
message ReleasePayload {
  string title = 1;

  repeated File files = 2;

  VCSSource vcs_source = 3;

  SchemaChangeType type = 4;

  message File {
    // ... existing fields ...
  }
}
```

**Step 2: Remove type field from ReleasePayload.File**

In the same file, remove line 24 (`SchemaChangeType type = 5;`) from the `File` message.

The File message should now be:

```proto
message File {
  // The unique identifier for the file.
  string id = 1;
  // The path of the file, e.g., `2.2/V0001_create_table.sql`.
  string path = 2;
  reserved 3; // Previously: string sheet (resource name)
  // The SHA256 hash of the sheet content (hex-encoded).
  string sheet_sha256 = 4;
  // Removed: SchemaChangeType type = 5;
  string version = 6;
  // Whether to use gh-ost for online schema migration.
  bool enable_ghost = 7;
}
```

**Step 3: Commit proto changes**

```bash
git add proto/store/store/release.proto
git commit -m "refactor: move SchemaChangeType to release level in store proto

Move type field from ReleasePayload.File to ReleasePayload.
Field number 5 is now unused in File message."
```

---

### Task 2: Update V1 API Proto (release_service.proto)

**Files:**
- Modify: `proto/v1/v1/release_service.proto`

**Step 1: Add Type enum to Release message**

In `proto/v1/v1/release_service.proto`, add the enum and type field to the `Release` message after the `digest` field (around line 299):

```proto
message Release {
  // ... existing fields ...

  // The digest of the release.
  string digest = 8;

  // The type of schema change for all files in this release.
  Type type = 9;

  // The type of schema change.
  enum Type {
    // Unspecified type.
    TYPE_UNSPECIFIED = 0;
    // Versioned schema migration.
    VERSIONED = 1;
    // Declarative schema definition.
    DECLARATIVE = 2;
  }

  // A SQL file in a release.
  message File {
    // ... existing fields ...
  }
}
```

**Step 2: Remove Type enum and type field from Release.File**

In the same file, remove:
1. The `Type type = 5;` field from `Release.File` (line 308)
2. The entire nested `enum Type` from `Release.File` (lines 326-333)

The File message should now be:

```proto
message File {
  // The unique identifier for the file.
  string id = 1;
  // The path of the file. e.g., `2.2/V0001_create_table.sql`.
  string path = 2;
  // Removed: Type type = 5;
  string version = 6;
  // Whether to use gh-ost for online schema migration.
  bool enable_ghost = 9;

  // For inputs, we must either use `sheet` or `statement`.
  // For outputs, we always use `sheet`. `statement` is the preview of the sheet content.
  //
  // The sheet that holds the content.
  // Format: projects/{project}/sheets/{sheet}
  string sheet = 3 [(google.api.resource_reference) = {type: "bytebase.com/Sheet"}];
  // The raw SQL statement content.
  bytes statement = 7 [(google.api.field_behavior) = INPUT_ONLY];
  // The SHA256 hash value of the sheet content or the statement.
  string sheet_sha256 = 4 [(google.api.field_behavior) = OUTPUT_ONLY];
}
```

**Step 3: Commit proto changes**

```bash
git add proto/v1/v1/release_service.proto
git commit -m "refactor: move Type to release level in v1 API proto

Move Type enum and type field from Release.File to Release.
Field number 5 is now unused in File message."
```

---

### Task 3: Generate Proto Code

**Step 1: Format proto files**

Run:
```bash
buf format -w proto
```

Expected: Proto files formatted successfully

**Step 2: Lint proto files**

Run:
```bash
buf lint proto
```

Expected: No linting errors

**Step 3: Generate Go code from protos**

Run:
```bash
cd proto && buf generate
```

Expected: Go files regenerated in `backend/generated-go/`

**Step 4: Verify generated code compiles**

Run:
```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: Build fails with compilation errors (expected - we need to fix backend code)

**Step 5: Commit generated code**

```bash
git add backend/generated-go/
git commit -m "build: regenerate proto code after moving type to release level"
```

---

### Task 4: Create Database Migration

**Files:**
- Create: `backend/migrator/migration/3.14/0018##move_release_type_to_release_level.sql`
- Modify: `backend/migrator/migrator_test.go`

**Step 1: Create migration SQL file**

Create `backend/migrator/migration/3.14/0018##move_release_type_to_release_level.sql`:

```sql
-- Move SchemaChangeType from per-file to release level in JSONB payloads.
-- Extract type from first file and set it at release level, then remove from all files.

UPDATE project_release
SET payload = jsonb_set(
    payload #- '{files}',
    '{type}',
    COALESCE(
        payload #> '{files,0,type}',
        '0'::jsonb  -- Default to SCHEMA_CHANGE_TYPE_UNSPECIFIED if no files
    )
) || jsonb_build_object(
    'files',
    (
        SELECT jsonb_agg(file_obj - 'type')
        FROM jsonb_array_elements(payload -> 'files') AS file_obj
    )
)
WHERE payload -> 'files' IS NOT NULL
  AND jsonb_array_length(payload -> 'files') > 0;
```

**Step 2: Update migrator test**

In `backend/migrator/migrator_test.go`, update line 15:

```go
func TestLatestVersion(t *testing.T) {
	files, err := getSortedVersionedFiles()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("3.14.18"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.14/0018##move_release_type_to_release_level.sql", files[len(files)-1].path)
}
```

**Step 3: Test migration file**

Run:
```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run ^TestLatestVersion$
```

Expected: PASS

**Step 4: Commit migration**

```bash
git add backend/migrator/migration/3.14/0018##move_release_type_to_release_level.sql backend/migrator/migrator_test.go
git commit -m "feat: add migration to move release type to release level

Migrate JSONB payloads in project_release table to move type field
from per-file to release level."
```

---

### Task 5: Update Release Service Converter (convertToRelease)

**Files:**
- Modify: `backend/api/v1/release_service.go:396-420`

**Step 1: Update convertToRelease function**

In `backend/api/v1/release_service.go` around line 396, update the function:

```go
func convertToRelease(release *store.ReleaseMessage) *v1pb.Release {
	r := &v1pb.Release{
		Name:       common.FormatReleaseName(release.ProjectID, release.UID),
		Title:      release.Payload.Title,
		Creator:    common.FormatUserEmail(release.Creator),
		CreateTime: timestamppb.New(release.At),
		VcsSource:  convertToReleaseVcsSource(release.Payload.VcsSource),
		State:      convertDeletedToState(release.Deleted),
		Digest:     release.Digest,
		Type:       v1pb.Release_Type(release.Payload.Type),
	}

	for _, f := range release.Payload.Files {
		// Sheets are now project-agnostic, no need to check projectID
		r.Files = append(r.Files, &v1pb.Release_File{
			Id:          f.Id,
			Path:        f.Path,
			Sheet:       common.FormatSheet(release.ProjectID, f.SheetSha256),
			SheetSha256: f.SheetSha256,
			// Removed: Type field
			Version:     f.Version,
			EnableGhost: f.EnableGhost,
		})
	}
	return r
}
```

**Step 2: Verify the code compiles**

Run:
```bash
go build ./backend/api/v1/...
```

Expected: Successful compilation

**Step 3: Commit changes**

```bash
git add backend/api/v1/release_service.go
git commit -m "refactor: update convertToRelease to use release-level type

Use release.Payload.Type instead of file.Type when converting
from store to v1 API format."
```

---

### Task 6: Update CreateRelease to Set Release-Level Type

**Files:**
- Modify: `backend/api/v1/release_service.go:45-144`

**Step 1: Find where payload is created in CreateRelease**

Read `backend/api/v1/release_service.go` starting at line 100 to find where `storepb.ReleasePayload` is created.

**Step 2: Add type field to ReleasePayload creation**

Around line 102-140 in the `CreateRelease` function, find where the payload is created and add the type field:

```go
payload := &storepb.ReleasePayload{
	Title:     req.Msg.Release.Title,
	VcsSource: convertReleaseVcsSource(req.Msg.Release.VcsSource),
	Type:      storepb.SchemaChangeType(req.Msg.Release.Type),
}

for _, file := range sanitizedFiles {
	h := sha256.Sum256(file.Statement)
	payload.Files = append(payload.Files, &storepb.ReleasePayload_File{
		Id:          file.Id,
		Path:        file.Path,
		SheetSha256: hex.EncodeToString(h[:]),
		// Removed: Type field
		Version:     file.Version,
		EnableGhost: file.EnableGhost,
	})
}
```

**Step 3: Verify the code compiles**

Run:
```bash
go build ./backend/api/v1/...
```

Expected: Successful compilation

**Step 4: Commit changes**

```bash
git add backend/api/v1/release_service.go
git commit -m "refactor: set release-level type in CreateRelease

Store type at release level instead of per-file when creating releases."
```

---

### Task 7: Update UpdateRelease to Handle Release-Level Type

**Files:**
- Modify: `backend/api/v1/release_service.go:277-333`

**Step 1: Find where payload is updated in UpdateRelease**

Read the `UpdateRelease` function around line 277-333 to find where payload fields are updated.

**Step 2: Add type field to update paths**

In the `UpdateRelease` function, find the section that handles field updates and add type handling:

```go
for _, path := range req.Msg.UpdateMask.Paths {
	switch path {
	case "title":
		patch.Payload.Title = req.Msg.Release.Title
	case "type":
		patch.Payload.Type = storepb.SchemaChangeType(req.Msg.Release.Type)
	case "files":
		// existing file handling code
		// Remove Type from file conversion
	// ... other cases
	}
}
```

**Step 3: Verify the code compiles**

Run:
```bash
go build ./backend/api/v1/...
```

Expected: Successful compilation

**Step 4: Commit changes**

```bash
git add backend/api/v1/release_service.go
git commit -m "refactor: handle release-level type in UpdateRelease

Allow updating type field at release level via UpdateRelease."
```

---

### Task 8: Update Release File Validation

**Files:**
- Modify: `backend/api/v1/release_service.go:450-540`

**Step 1: Update validateAndSanitizeReleaseFiles function**

Find the `validateAndSanitizeReleaseFiles` function and update type validation:

Remove the `fileTypeCount` map (line 456) and related validation since type is now at release level, not per-file.

The function should no longer validate or count file types.

**Step 2: Verify the code compiles**

Run:
```bash
go build ./backend/api/v1/...
```

Expected: Successful compilation

**Step 3: Commit changes**

```bash
git add backend/api/v1/release_service.go
git commit -m "refactor: remove per-file type validation

Type is now at release level, so file-level type validation is no longer needed."
```

---

### Task 9: Update CheckRelease to Use Release-Level Type

**Files:**
- Modify: `backend/api/v1/release_service_check.go:100-141`

**Step 1: Update CheckRelease function**

In `backend/api/v1/release_service_check.go`, update the function around line 120:

Change from:
```go
releaseFileType := sanitizedFiles[0].Type
```

To:
```go
releaseType := request.Release.Type
```

**Step 2: Update switch statement**

Update the switch statement to use `releaseType`:

```go
var response *v1pb.CheckReleaseResponse
switch releaseType {
case v1pb.Release_DECLARATIVE:
	resp, err := s.checkReleaseDeclarative(ctx, sanitizedFiles, targetDatabases, request.CustomRules)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check release declarative"))
	}
	response = resp
case v1pb.Release_VERSIONED:
	resp, err := s.checkReleaseVersioned(ctx, project, sanitizedFiles, targetDatabases, request.CustomRules)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check release versioned"))
	}
	response = resp
default:
	return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected release type %q", releaseType.String()))
}
```

**Step 3: Verify the code compiles**

Run:
```bash
go build ./backend/api/v1/...
```

Expected: Successful compilation

**Step 4: Commit changes**

```bash
git add backend/api/v1/release_service_check.go
git commit -m "refactor: use release-level type in CheckRelease

Read type from release instead of first file when checking release."
```

---

### Task 10: Update Database Migration Executor

**Files:**
- Modify: `backend/runner/taskrun/database_migrate_executor.go:287-411`

**Step 1: Update runReleaseTask to use release-level type**

In `backend/runner/taskrun/database_migrate_executor.go`, update the file iteration loop around line 289:

Change from reading `file.Type` to reading `release.Payload.Type` once before the loop.

Before the loop (around line 287):
```go
}

// Get release type once
releaseType := release.Payload.Type

// Execute unapplied files in order
for _, file := range release.Payload.Files {
	switch releaseType {
	case storepb.SchemaChangeType_VERSIONED:
		// ... existing versioned logic, now uses releaseType
	case storepb.SchemaChangeType_DECLARATIVE:
		// ... existing declarative logic, now uses releaseType
	default:
		return true, nil, errors.Errorf("unsupported release type %q", releaseType)
	}
}
```

**Step 2: Verify the code compiles**

Run:
```bash
go build ./backend/runner/taskrun/...
```

Expected: Successful compilation

**Step 3: Commit changes**

```bash
git add backend/runner/taskrun/database_migrate_executor.go
git commit -m "refactor: use release-level type in migration executor

Read type from release.Payload.Type instead of file.Type when executing migrations."
```

---

### Task 11: Update Revision Creation to Use Release Type

**Files:**
- Modify: `backend/store/revision.go`

**Step 1: Find revision creation code**

Read `backend/store/revision.go` to find where revisions are created from release files.

Run:
```bash
grep -n "SchemaChangeType" backend/store/revision.go
```

**Step 2: Update revision creation**

Find where `RevisionPayload` is created and ensure it copies the type from the release, not from individual files.

The revision should store the release's type for proper history tracking.

**Step 3: Verify the code compiles**

Run:
```bash
go build ./backend/store/...
```

Expected: Successful compilation

**Step 4: Commit changes**

```bash
git add backend/store/revision.go
git commit -m "refactor: copy release type when creating revisions

Revisions now copy type from release instead of individual files."
```

---

### Task 12: Run Linters and Format Code

**Step 1: Format Go code**

Run:
```bash
gofmt -w backend/api/v1/release_service.go backend/api/v1/release_service_check.go backend/runner/taskrun/database_migrate_executor.go backend/store/revision.go
```

Expected: Files formatted

**Step 2: Run golangci-lint (first pass)**

Run:
```bash
golangci-lint run --allow-parallel-runners
```

Expected: May show some issues

**Step 3: Auto-fix linting issues**

Run:
```bash
golangci-lint run --fix --allow-parallel-runners
```

Expected: Some issues auto-fixed

**Step 4: Run golangci-lint again until clean**

Run:
```bash
golangci-lint run --allow-parallel-runners
```

Expected: No issues (repeat if needed)

**Step 5: Commit formatting and lint fixes**

```bash
git add -u
git commit -m "style: format code and fix linting issues"
```

---

### Task 13: Build and Run Tests

**Step 1: Build the project**

Run:
```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: Successful build

**Step 2: Run release service tests**

Run:
```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run Release
```

Expected: Tests pass

**Step 3: Run migration executor tests**

Run:
```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/runner/taskrun -run DatabaseMigrate
```

Expected: Tests pass

**Step 4: Run revision store tests**

Run:
```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/store -run Revision
```

Expected: Tests pass

---

### Task 14: Final Verification and Cleanup

**Step 1: Search for remaining file.Type references**

Run:
```bash
grep -r "file\.Type" backend/ --include="*.go" | grep -v "file\.Type_" | grep -v "// file\.Type"
```

Expected: No results (or only comments)

**Step 2: Search for Files[].Type references**

Run:
```bash
grep -r "Files\[.*\]\.Type" backend/ --include="*.go"
```

Expected: No results

**Step 3: Verify all tests pass**

Run:
```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/...
```

Expected: All tests pass (this may take several minutes)

**Step 4: Create final commit if needed**

If any final cleanup was done:
```bash
git add -u
git commit -m "chore: final cleanup for release-level type refactoring"
```

---

### Task 15: Verify Branch State

**Step 1: Check git status**

Run:
```bash
git status
```

Expected: Clean working tree on feature branch

**Step 2: Review commit history**

Run:
```bash
git log --oneline main..HEAD
```

Expected: Shows all commits for this feature

**Step 3: Verify on correct branch**

Run:
```bash
git branch --show-current
```

Expected: `feature/schema-change-type-release-level`

---

## Implementation Complete

All tasks completed. The SchemaChangeType has been successfully moved from per-file to release level in both store protos and v1 API protos.

**Next Steps:**
1. Test the changes manually with a local Bytebase instance
2. Create a pull request
3. Mark PR with `breaking` label
4. Request code review
