# Release ID with Iteration Tracking Design

## Overview

Add a release ID system with iteration tracking similar to Google Rapid's approach. This enables sequential versioning of releases within a "train" (e.g., `release_20260115-RC00`, `release_20260115-RC01`) with atomic increment guarantees and customizable templates.

## Motivation

Currently, GitOps releases use auto-generated titles (`[project] timestamp`) and UID-based resource names (`projects/xxx/releases/101`). This design introduces:

- **Human-readable release IDs** with semantic meaning
- **Iteration tracking** for multiple releases on the same train
- **Template flexibility** for different naming conventions
- **Atomic increment** to prevent race conditions

## Design

### 1. Schema Changes

**Database Schema:**

Add three columns to the `release` table:

```sql
ALTER TABLE release ADD COLUMN release_id TEXT NOT NULL DEFAULT '';
ALTER TABLE release ADD COLUMN train TEXT NOT NULL DEFAULT '';
ALTER TABLE release ADD COLUMN iteration INTEGER NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX idx_release_project_train_iteration ON release(project, train, iteration);
CREATE INDEX idx_release_project_release_id ON release(project, release_id);
```

**Column Definitions:**
- `release_id`: The full rendered ID (e.g., `hello_server_20260115_1430-RC00`)
- `train`: The template without iteration (e.g., `hello_server_20260115_1430-RC`)
- `iteration`: Zero-based integer (0 = RC00, 1 = RC01, etc.)

**Constraints:**
- `UNIQUE(project, train, iteration)`: Prevents duplicate iterations for same train
- Index on `(project, release_id)`: Fast lookups by release ID

**Go Struct Changes:**

Update `ReleaseMessage` in `backend/store/release.go`:

```go
type ReleaseMessage struct {
	ProjectID string
	Digest    string
	Payload   *storepb.ReleasePayload

	// output only
	UID        int64
	Deleted    bool
	Creator    string
	At         time.Time
	ReleaseID  string  // New: full rendered ID (e.g., "hello_server_20260115_1430-RC00")
	Train      string  // New: template without iteration (e.g., "hello_server_20260115_1430-RC")
	Iteration  int32   // New: zero-based integer (0 = RC00, 1 = RC01, etc.)
}
```

Update `FindReleaseMessage` in `backend/store/release.go`:

```go
type FindReleaseMessage struct {
	ProjectID   *string
	UID         *int64
	ReleaseID   *string  // New: query by release_id
	Limit       *int
	Offset      *int
	ShowDeleted bool
}
```

### 2. Template Rendering & Iteration Logic

**Template Variables:**

Available variables for `--release-id-template`:
- `{date}`: YYYYMMDD format (e.g., `20260115`)
- `{time}`: HHMM format (e.g., `1430`)
- `{timestamp}`: YYYYMMDD_HHMM format (e.g., `20260115_1430`)
- `{iteration}`: 00, 01, ..., 99, 100, etc. (min 2-digit zero-padded)

**Template Rendering Process:**

1. Parse template (e.g., `hello_server_{date}_{time}-RC{iteration}`)
2. Get current time in specified timezone (default UTC)
3. Render train = template without `{iteration}` → `hello_server_20260115_1430-RC`
4. Query database atomically:
   ```sql
   BEGIN;
   SELECT COALESCE(MAX(iteration), -1) FROM release
   WHERE project = ? AND train = ? FOR UPDATE;
   -- next_iteration = result + 1
   COMMIT;
   ```
5. Render full release_id with `fmt.Sprintf("%s%02d", train, iteration)` → `hello_server_20260115_1430-RC00`

**Iteration Format:**
- Format: `%02d` (minimum 2 digits, zero-padded)
- Examples: `00`, `01`, `99`, `100`, `999`
- No upper limit

**Iteration Scope:**
- Iteration is scoped to `(project_id, train)` tuple
- Each project maintains independent iteration sequences
- Each unique train value starts at iteration 0

**Default Template:**
- If `--release-id-template` not provided: `release_{date}-RC{iteration}`
- Example: `release_20260115-RC00`

**Default Timezone:**
- If `--release-id-timezone` not provided: `UTC`

### 3. CLI Interface

**New Command-Line Flags:**

Add to `bytebase-action rollout` command:

```go
cmdRollout.Flags().StringVar(&w.ReleaseIDTemplate, "release-id-template", "release_{date}-RC{iteration}",
    "Template for release ID. Available variables: {date}, {time}, {timestamp}, {iteration}")
cmdRollout.Flags().StringVar(&w.ReleaseIDTimezone, "release-id-timezone", "UTC",
    "Timezone for {date} and {time} variables (e.g., 'UTC', 'America/Los_Angeles')")
```

**Usage Examples:**

```bash
# Use default template (release_20260115-RC00)
bytebase-action rollout --project projects/my-project ...

# Custom template with date and time
bytebase-action rollout \
  --release-id-template "myapp_{date}_{time}-RC{iteration}" \
  --project projects/my-project ...

# Date-only train (all releases on same day increment together)
bytebase-action rollout \
  --release-id-template "v{date}.{iteration}" \
  --project projects/my-project ...

# Custom timezone
bytebase-action rollout \
  --release-id-template "release_{date}_{time}-RC{iteration}" \
  --release-id-timezone "America/Los_Angeles" \
  --project projects/my-project ...

# Custom prefix without RC
bytebase-action rollout \
  --release-id-template "hello_server_{date}_{time}_{iteration}" \
  --project projects/my-project ...
```

**World Struct Updates:**

Add fields to `action/world/world.go`:

```go
type World struct {
    // ... existing fields ...
    ReleaseIDTemplate string
    ReleaseIDTimezone string
}
```

### 4. Implementation Flow

**API Changes (`proto/v1/release_service.proto`):**

Update `CreateReleaseRequest` to accept train:

```protobuf
message CreateReleaseRequest {
  string parent = 1;
  Release release = 2;

  // Train for iteration tracking (template rendered without {iteration})
  string train = 3;
}
```

**Resource Name Change:**

```go
// backend/common/resource_name.go
func FormatReleaseName(projectID string, releaseID string) string {
    return fmt.Sprintf("%s/%s%s", FormatProject(projectID), ReleaseNamePrefix, releaseID)
}
// Example: projects/my-project/releases/hello_server_20260115_1430-RC00
```

**Store Layer (`backend/store/release.go`):**

```go
func (s *Store) CreateRelease(ctx context.Context, release *ReleaseMessage, creator string) (*ReleaseMessage, error) {
    p, err := protojson.Marshal(release.Payload)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to marshal release payload")
    }

    tx, err := s.GetDB().BeginTx(ctx, nil)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to begin tx")
    }
    defer tx.Rollback()

    // Atomically get next iteration for (project, train)
    var maxIteration sql.NullInt64
    err = tx.QueryRowContext(ctx,
        "SELECT MAX(iteration) FROM release WHERE project = ? AND train = ? FOR UPDATE",
        release.ProjectID, release.Train,
    ).Scan(&maxIteration)
    if err != nil && err != sql.ErrNoRows {
        return nil, errors.Wrapf(err, "failed to get max iteration")
    }

    nextIteration := int32(0)
    if maxIteration.Valid {
        nextIteration = int32(maxIteration.Int64) + 1
    }

    // Compute release_id = train + formatted iteration
    releaseID := fmt.Sprintf("%s%02d", release.Train, nextIteration)

    // Insert with all fields
    q := qb.Q().Space(`
        INSERT INTO release (
            creator,
            project,
            digest,
            payload,
            release_id,
            train,
            iteration
        ) VALUES (
            ?, ?, ?, ?, ?, ?, ?
        ) RETURNING id, created_at
    `, creator, release.ProjectID, release.Digest, p,
       releaseID, release.Train, nextIteration)

    query, args, err := q.ToSQL()
    if err != nil {
        return nil, errors.Wrapf(err, "failed to build sql")
    }

    var id int64
    var createdTime time.Time
    if err := tx.QueryRowContext(ctx, query, args...).Scan(&id, &createdTime); err != nil {
        return nil, errors.Wrapf(err, "failed to insert release")
    }

    if err := tx.Commit(); err != nil {
        return nil, errors.Wrapf(err, "failed to commit tx")
    }

    release.UID = id
    release.Creator = creator
    release.At = createdTime
    release.ReleaseID = releaseID
    release.Iteration = nextIteration

    return release, nil
}
```

Update `ListReleases` to include new columns:

```go
func (s *Store) ListReleases(ctx context.Context, find *FindReleaseMessage) ([]*ReleaseMessage, error) {
    q := qb.Q().Space(`
        SELECT
            id,
            deleted,
            project,
            digest,
            creator,
            created_at,
            payload,
            release_id,
            train,
            iteration
        FROM release
        WHERE TRUE
    `)

    if v := find.ProjectID; v != nil {
        q.And("project = ?", *v)
    }
    if v := find.UID; v != nil {
        q.And("id = ?", *v)
    }
    if v := find.ReleaseID; v != nil {
        q.And("release_id = ?", *v)
    }
    // ... rest of query ...

    // In scan:
    if err := rows.Scan(
        &r.UID,
        &r.Deleted,
        &r.ProjectID,
        &r.Digest,
        &r.Creator,
        &r.At,
        &payload,
        &r.ReleaseID,
        &r.Train,
        &r.Iteration,
    ); err != nil {
        return nil, errors.Wrapf(err, "failed to scan rows")
    }
}
```

**Service Layer (`backend/api/v1/release_service.go`):**

```go
func (s *ReleaseService) CreateRelease(ctx context.Context, request *v1pb.CreateReleaseRequest) (*v1pb.Release, error) {
    // ... existing validation ...

    releaseMessage := &store.ReleaseMessage{
        ProjectID: projectID,
        Train:     request.Train, // Passed from action layer
        Payload:   convertReleasePayload(request.Release),
        Digest:    request.Release.Digest,
    }

    created, err := s.store.CreateRelease(ctx, releaseMessage, creator)
    if err != nil {
        return nil, err
    }

    // Convert to proto with new name format
    return convertToRelease(created), nil
}

func convertToRelease(release *store.ReleaseMessage) *v1pb.Release {
    r := &v1pb.Release{
        Name:       common.FormatReleaseName(release.ProjectID, release.ReleaseID), // Use release_id instead of UID
        Title:      release.Payload.Title,
        Creator:    common.FormatUserEmail(release.Creator),
        CreateTime: timestamppb.New(release.At),
        VcsSource:  convertToReleaseVcsSource(release.Payload.VcsSource),
        State:      convertDeletedToState(release.Deleted),
        Digest:     release.Digest,
        Type:       v1pb.Release_Type(release.Payload.Type),
    }
    // ... file processing ...
    return r
}
```

**Action Layer (`action/command/rollout.go`):**

```go
func runRollout(w *world.World) func(command *cobra.Command, _ []string) error {
    return func(command *cobra.Command, _ []string) error {
        // ... existing setup ...

        // Render train from template
        train, err := renderTrain(w.ReleaseIDTemplate, w.ReleaseIDTimezone)
        if err != nil {
            return errors.Wrapf(err, "failed to render train")
        }

        createReleaseResponse, err := client.CreateRelease(
            ctx,
            w.Project,
            &v1pb.Release{
                Title:     w.ReleaseTitle,
                Files:     releaseFiles,
                VcsSource: getVCSSource(w),
                Digest:    releaseDigest,
                Type:      releaseType,
            },
            train, // Pass computed train
        )
        if err != nil {
            return errors.Wrapf(err, "failed to create release")
        }

        // Response name is now: projects/xxx/releases/hello_server_20260115_1430-RC00
        w.Logger.Info("release created", "url", fmt.Sprintf("%s/%s", client.url, createReleaseResponse.Name))

        // ... rest of rollout ...
    }
}

func renderTrain(template, timezone string) (string, error) {
    // Validate template
    if err := validateTemplate(template); err != nil {
        return "", err
    }

    // Validate timezone
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        return "", errors.Wrapf(err, "invalid timezone: %s", timezone)
    }

    now := time.Now().In(loc)

    train := template
    train = strings.ReplaceAll(train, "{date}", now.Format("20060102"))
    train = strings.ReplaceAll(train, "{time}", now.Format("1504"))
    train = strings.ReplaceAll(train, "{timestamp}", now.Format("20060102_1504"))
    train = strings.ReplaceAll(train, "{iteration}", "")

    return train, nil
}

func validateTemplate(template string) error {
    // Must contain {iteration}
    if !strings.Contains(template, "{iteration}") {
        return errors.New("template must contain {iteration} placeholder")
    }

    // {iteration} must be at the end of the template
    if !strings.HasSuffix(template, "{iteration}") {
        return errors.New("{iteration} must be at the end of the template")
    }

    // Must contain at least one time variable
    hasTimeVar := strings.Contains(template, "{date}") ||
                  strings.Contains(template, "{time}") ||
                  strings.Contains(template, "{timestamp}")
    if !hasTimeVar {
        return errors.New("template must contain at least one of: {date}, {time}, {timestamp}")
    }

    return nil
}
```

### 5. Migration & Default Behavior

**Migration Script:**

Backfill existing releases with meaningful values based on `created_at`:

```sql
-- Migration: XXXX##backfill_release_id.sql

-- Step 1: Calculate iteration numbers using window function
-- Group by (project, date) and assign iteration based on creation order
WITH numbered_releases AS (
  SELECT
    id,
    project,
    TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYYMMDD') AS date_str,
    ROW_NUMBER() OVER (
      PARTITION BY project, TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYYMMDD')
      ORDER BY created_at, id
    ) - 1 AS iteration_num
  FROM release
  WHERE release_id = ''
)
UPDATE release r
SET
  train = 'release_' || n.date_str || '-RC',
  iteration = n.iteration_num,
  release_id = 'release_' || n.date_str || '-RC' || LPAD(n.iteration_num::TEXT, 2, '0')
FROM numbered_releases n
WHERE r.id = n.id;

-- Step 2: Ensure columns are not nullable
ALTER TABLE release ALTER COLUMN release_id SET NOT NULL;
ALTER TABLE release ALTER COLUMN train SET NOT NULL;
ALTER TABLE release ALTER COLUMN iteration SET NOT NULL;
```

**Migration Result Examples:**

Project `hello-server` with 3 releases on 2026-01-15:
- Release created at `2026-01-15 10:30:00 UTC` → `release_20260115-RC00` (train: `release_20260115-RC`, iteration: 0)
- Release created at `2026-01-15 14:45:00 UTC` → `release_20260115-RC01` (train: `release_20260115-RC`, iteration: 1)
- Release created at `2026-01-15 18:00:00 UTC` → `release_20260115-RC02` (train: `release_20260115-RC`, iteration: 2)

**Default Behavior:**

- Always use release ID template (no opt-out)
- Default template: `release_{date}-RC{iteration}`
- Default timezone: `UTC`

**Resource Name Format:**

```go
// backend/common/resource_name.go
func FormatReleaseName(projectID string, releaseID string) string {
    return fmt.Sprintf("%s/%s%s", FormatProject(projectID), ReleaseNamePrefix, releaseID)
}
// Example: projects/hello-server/releases/release_20260115-RC00
```

### 6. Error Handling & Edge Cases

**Template Validation:**

```go
func validateTemplate(template string) error {
    // Must contain {iteration}
    if !strings.Contains(template, "{iteration}") {
        return errors.New("template must contain {iteration} placeholder")
    }

    // {iteration} must be at the end of the template
    if !strings.HasSuffix(template, "{iteration}") {
        return errors.New("{iteration} must be at the end of the template")
    }

    // Must contain at least one time variable
    hasTimeVar := strings.Contains(template, "{date}") ||
                  strings.Contains(template, "{time}") ||
                  strings.Contains(template, "{timestamp}")
    if !hasTimeVar {
        return errors.New("template must contain at least one of: {date}, {time}, {timestamp}")
    }

    return nil
}
```

**Timezone Validation:**

```go
func validateTimezone(tz string) error {
    _, err := time.LoadLocation(tz)
    if err != nil {
        return errors.Wrapf(err, "invalid timezone: %s", tz)
    }
    return nil
}
```

**Race Condition Handling:**

The `SELECT ... FOR UPDATE` ensures no two releases get the same iteration for a (project, train) pair. If two releases are created simultaneously:
- Both start transactions
- First one acquires lock, gets iteration N
- Second one waits for lock, gets iteration N+1
- Both commit successfully with unique iterations

**Unique Constraint Violation:**

If the unique constraint `UNIQUE(project, train, iteration)` is violated:
- Database returns constraint violation error
- Return error to user: "Failed to create release: iteration conflict"
- User can retry (should succeed on retry)

**Query Performance:**

With `SELECT MAX(iteration) ... FOR UPDATE`:
- Index on `(project, train, iteration)` ensures fast lookup
- Lock scope limited to specific (project, train) pair
- Minimal contention between different trains

## Summary

| Aspect | Details |
|--------|---------|
| **Release ID** | Template-based, customizable (default: `release_{date}-RC{iteration}`) |
| **Train** | Template without iteration placeholder |
| **Iteration** | Zero-based, scoped to (project, train), atomic increment |
| **Resource Name** | `projects/{project}/releases/{release_id}` |
| **CLI Flags** | `--release-id-template`, `--release-id-timezone` |
| **Migration** | Backfill existing releases with date-based iterations |
| **Atomicity** | `SELECT ... FOR UPDATE` prevents race conditions |

This design provides flexible, human-readable release IDs while maintaining strong consistency guarantees.
