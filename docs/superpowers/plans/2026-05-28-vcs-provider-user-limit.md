# VCS Provider User Limit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Track VCS PR/MR creators from bytebase-release, enforce the active VCS user pool against the license seat limit, show the active count on the Subscription page, and export active VCS users as CSV.

**Architecture:** Add a workspace-scoped `vcs_provider_user` table keyed by `(workspace, vcs_type, user_id)` with display metadata in JSONB payload. `CheckRelease` validates attribution early and touches attributed users after target and release-file validation, `ActuatorInfo` exposes the active count, `SubscriptionService` exports active users as `google.api.HttpBody`, and bytebase-release submits optional VCS attribution for non-bot PR/MR creators.

**Tech Stack:** Go, PostgreSQL migrations, Connect/gRPC + grpc-gateway annotations, protobuf + buf, React, Tailwind, Vitest, Go test, bytebase-action CLI.

---

## File Structure

- Create `proto/store/store/vcs_provider_user.proto`: JSONB payload proto for VCS display metadata.
- Modify `proto/v1/v1/release_service.proto`: add `VCSUser` and optional `CheckReleaseRequest.vcs_user`.
- Modify `proto/v1/v1/actuator_service.proto`: add `active_vcs_user_count` to `ActuatorInfo`.
- Modify `proto/v1/v1/subscription_service.proto`: add `google.api.HttpBody` import and `ExportVCSProviderUsers`.
- Create `backend/migrator/migration/3.19/0000##vcs_provider_user.sql`: add table and index.
- Modify `backend/migrator/migration/LATEST.sql`: add table and index.
- Modify `backend/migrator/migrator_test.go`: update latest migration path.
- Create `backend/store/vcs_provider_user.go`: store model, count, touch, list/export helpers.
- Create `backend/store/vcs_provider_user_test.go`: store tests for touch, active window, limit rejection, and export ordering.
- Modify `backend/api/v1/release_service.go`: add `licenseService` dependency to `ReleaseService`.
- Modify `backend/server/grpc_routes.go`: pass `licenseService` into `NewReleaseService`.
- Modify `backend/api/v1/release_service_check.go`: validate VCS attribution early and touch it after target and release-file validation.
- Create `backend/api/v1/vcs_provider_user.go`: active-window constant shared by release, actuator, and subscription services.
- Modify `backend/api/v1/actuator_service.go`: populate `active_vcs_user_count`.
- Modify `backend/api/v1/subscription_service.go`: implement CSV export.
- Modify backend generated protobuf files after `buf generate`.
- Modify `action/world/world.go`: add optional VCS attribution field.
- Create `action/command/vcs_user.go`: provider-specific attribution extraction.
- Create `action/command/vcs_user_test.go`: environment/event parsing tests.
- Modify `action/command/check.go`: attach `VcsUser` to `CheckReleaseRequest`.
- Modify frontend generated proto-es files after `buf generate`.
- Modify `frontend/src/react/stores/app/types.ts`: add `activeVcsUserCount` selector to app-store type.
- Modify `frontend/src/react/stores/app/workspace.ts`: expose `activeVcsUserCount`.
- Modify `frontend/src/react/hooks/useAppState.ts`: return `activeVcsUserCount` from `useServerState`.
- Modify `frontend/src/react/pages/settings/SubscriptionPage.tsx`: render VCS count and CSV download action.
- Modify `frontend/src/react/pages/settings/SubscriptionPage.test.tsx`: add render and download coverage.
- Modify locale files under `frontend/src/locales/` and `frontend/src/react/locales/` for new user-facing strings.

---

### Task 1: Add Schema And Proto Contracts

**Files:**
- Create: `proto/store/store/vcs_provider_user.proto`
- Modify: `proto/v1/v1/release_service.proto`
- Modify: `proto/v1/v1/actuator_service.proto`
- Modify: `proto/v1/v1/subscription_service.proto`
- Create: `backend/migrator/migration/3.19/0000##vcs_provider_user.sql`
- Modify: `backend/migrator/migration/LATEST.sql`
- Modify: `backend/migrator/migrator_test.go`

- [ ] **Step 1: Add the store payload proto**

Create `proto/store/store/vcs_provider_user.proto`:

```proto
syntax = "proto3";

package bytebase.store;

option go_package = "generated-go/store";

message VCSProviderUserPayload {
  string user_name = 1;
  string display_name = 2;
}
```

- [ ] **Step 2: Add request attribution to release proto**

In `proto/v1/v1/release_service.proto`, add this message after `CheckReleaseRequest`:

```proto
message VCSUser {
  VCSType vcs_type = 1;
  string user_id = 2;
  string user_name = 3;
  string display_name = 4;
}
```

Then add field 5 to `CheckReleaseRequest`:

```proto
  // The non-bot VCS pull request or merge request creator observed by bytebase-release.
  // If absent, Bytebase skips VCS user tracking and VCS user limit enforcement.
  VCSUser vcs_user = 5;
```

- [ ] **Step 3: Add actuator count field**

In `proto/v1/v1/actuator_service.proto`, append field 28 to `ActuatorInfo`:

```proto
  // The number of active VCS users seen in the active window.
  int32 active_vcs_user_count = 28 [(google.api.field_behavior) = OUTPUT_ONLY];
```

- [ ] **Step 4: Add subscription CSV export RPC**

In `proto/v1/v1/subscription_service.proto`, add:

```proto
import "google/api/httpbody.proto";
```

Add this service method after `GetSubscription`:

```proto
  // Exports active VCS users as CSV.
  rpc ExportVCSProviderUsers(ExportVCSProviderUsersRequest) returns (google.api.HttpBody) {
    option (google.api.http) = {get: "/v1/subscription:vcsProviderUsersExport"};
    option (bytebase.v1.permission) = "bb.subscription.manage";
    option (bytebase.v1.auth_method) = IAM;
  }
```

Add this request message near `GetSubscriptionRequest`:

```proto
message ExportVCSProviderUsersRequest {}
```

- [ ] **Step 5: Add migration**

Create `backend/migrator/migration/3.19/0000##vcs_provider_user.sql`:

```sql
CREATE TABLE vcs_provider_user (
    workspace text NOT NULL REFERENCES workspace(resource_id),
    vcs_type text NOT NULL,
    user_id text NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (workspace, vcs_type, user_id)
);

CREATE INDEX idx_vcs_provider_user_workspace_last_seen_at
    ON vcs_provider_user(workspace, last_seen_at DESC);

CREATE INDEX idx_vcs_provider_user_last_seen_at
    ON vcs_provider_user(last_seen_at);
```

- [ ] **Step 6: Update latest schema**

Insert the same `CREATE TABLE` and `CREATE INDEX` into `backend/migrator/migration/LATEST.sql` in the workspace-scoped tables section after `subscription`.

- [ ] **Step 7: Update migrator latest test**

In `backend/migrator/migrator_test.go`, update:

```go
require.Equal(t, semver.MustParse("3.19.0"), *files[len(files)-1].version)
require.Equal(t, "migration/3.19/0000##vcs_provider_user.sql", files[len(files)-1].path)
```

- [ ] **Step 8: Generate proto code**

Run:

```bash
buf format -w proto
cd proto && buf generate
```

Expected: generated Go and frontend proto-es files update without errors.

- [ ] **Step 9: Verify migration test**

Run:

```bash
go test -v -count=1 ./backend/migrator -run ^TestLatestVersion$
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add proto backend/migrator
git commit -m "feat: add vcs provider user schema"
```

---

### Task 2: Add VCS Provider User Store

**Files:**
- Create: `backend/store/vcs_provider_user.go`
- Create: `backend/store/vcs_provider_user_test.go`

- [ ] **Step 1: Write store tests**

Create `backend/store/vcs_provider_user_test.go` with these test cases:

```go
package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/store"
)

func TestVCSProviderUserTouchAndCount(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)
	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	const workspace = "default"
	activeWindow := 90 * 24 * time.Hour

	ok, err := s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITHUB,
		UserID:  "1001",
		Payload: &storepb.VCSProviderUserPayload{UserName: "alice", DisplayName: "Alice"},
	}, activeWindow, 1)
	require.NoError(t, err)
	require.True(t, ok)

	count, err := s.CountActiveVCSProviderUsers(ctx, workspace, activeWindow)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	ok, err = s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITHUB,
		UserID:  "1001",
		Payload: &storepb.VCSProviderUserPayload{UserName: "alice2", DisplayName: "Alice Cooper"},
	}, activeWindow, 1)
	require.NoError(t, err)
	require.True(t, ok)

	users, err := s.ListActiveVCSProviderUsers(ctx, workspace, activeWindow)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "alice2", users[0].Payload.GetUserName())
}

func TestVCSProviderUserLimitAndInactiveWindow(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)
	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	const workspace = "default"
	activeWindow := 90 * 24 * time.Hour

	_, err = db.ExecContext(ctx, `
		INSERT INTO vcs_provider_user (workspace, vcs_type, user_id, last_seen_at, payload)
		VALUES ($1, $2, $3, $4, '{}'::jsonb)
	`, workspace, v1pb.VCSType_GITHUB.String(), "old", time.Now().Add(-91*24*time.Hour))
	require.NoError(t, err)
	ok, err := s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_GITLAB,
		UserID:  "new",
		Payload: &storepb.VCSProviderUserPayload{UserName: "bob"},
	}, activeWindow, 1)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = s.TouchVCSProviderUser(ctx, workspace, &store.VCSProviderUserMessage{
		VCSType: v1pb.VCSType_BITBUCKET,
		UserID:  "blocked",
		Payload: &storepb.VCSProviderUserPayload{UserName: "carol"},
	}, activeWindow, 1)
	require.NoError(t, err)
	require.False(t, ok)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test -v -count=1 ./backend/store -run ^TestVCSProviderUser
```

Expected: FAIL because store methods and message types do not exist.

- [ ] **Step 3: Implement store methods**

Create `backend/store/vcs_provider_user.go`:

```go
package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

type VCSProviderUserMessage struct {
	Workspace  string
	VCSType    v1pb.VCSType
	UserID     string
	LastSeenAt time.Time
	Payload    *storepb.VCSProviderUserPayload
}

func vcsProviderUserActiveCutoff(activeWindow time.Duration) time.Time {
	return time.Now().Add(-activeWindow)
}

func (s *Store) CountActiveVCSProviderUsers(ctx context.Context, workspace string, activeWindow time.Duration) (int, error) {
	var count int
	if err := s.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM vcs_provider_user
		WHERE workspace = $1 AND last_seen_at >= $2
	`, workspace, vcsProviderUserActiveCutoff(activeWindow)).Scan(&count); err != nil {
		return 0, errors.Wrap(err, "failed to count active VCS provider users")
	}
	return count, nil
}

func (s *Store) TouchVCSProviderUser(ctx context.Context, workspace string, user *VCSProviderUserMessage, activeWindow time.Duration, limit int) (bool, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", "vcs_provider_user/"+workspace); err != nil {
		return false, errors.Wrap(err, "failed to lock VCS provider user workspace")
	}

	cutoff := vcsProviderUserActiveCutoff(activeWindow)
	var lastSeenAt time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT last_seen_at
		FROM vcs_provider_user
		WHERE workspace = $1 AND vcs_type = $2 AND user_id = $3
	`, workspace, user.VCSType.String(), user.UserID).Scan(&lastSeenAt)
	if err != nil && err != sql.ErrNoRows {
		return false, errors.Wrap(err, "failed to get VCS provider user")
	}

	if err == sql.ErrNoRows || lastSeenAt.Before(cutoff) {
		var activeCount int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM vcs_provider_user
			WHERE workspace = $1 AND last_seen_at >= $2
		`, workspace, cutoff).Scan(&activeCount); err != nil {
			return false, errors.Wrap(err, "failed to count active VCS provider users")
		}
		if activeCount >= limit {
			return false, tx.Commit()
		}
	}

	payload := user.Payload
	if payload == nil {
		payload = &storepb.VCSProviderUserPayload{}
	}
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return false, errors.Wrap(err, "failed to marshal VCS provider user payload")
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO vcs_provider_user (workspace, vcs_type, user_id, last_seen_at, payload)
		VALUES ($1, $2, $3, now(), $4)
		ON CONFLICT (workspace, vcs_type, user_id)
		DO UPDATE SET last_seen_at = now(), payload = EXCLUDED.payload
	`, workspace, user.VCSType.String(), user.UserID, string(payloadBytes)); err != nil {
		return false, errors.Wrap(err, "failed to touch VCS provider user")
	}
	return true, tx.Commit()
}

func (s *Store) ListActiveVCSProviderUsers(ctx context.Context, workspace string, activeWindow time.Duration) ([]*VCSProviderUserMessage, error) {
	rows, err := s.GetDB().QueryContext(ctx, `
		SELECT workspace, vcs_type, user_id, last_seen_at, payload
		FROM vcs_provider_user
		WHERE workspace = $1 AND last_seen_at >= $2
		ORDER BY last_seen_at DESC
	`, workspace, vcsProviderUserActiveCutoff(activeWindow))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list active VCS provider users")
	}
	defer rows.Close()

	var users []*VCSProviderUserMessage
	for rows.Next() {
		var user VCSProviderUserMessage
		var vcsType string
		var payloadBytes []byte
		if err := rows.Scan(&user.Workspace, &vcsType, &user.UserID, &user.LastSeenAt, &payloadBytes); err != nil {
			return nil, err
		}
		user.VCSType = v1pb.VCSType(v1pb.VCSType_value[vcsType])
		payload := &storepb.VCSProviderUserPayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal VCS provider user payload")
		}
		user.Payload = payload
		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

```

- [ ] **Step 4: Run store tests**

Run:

```bash
go test -v -count=1 ./backend/store -run ^TestVCSProviderUser
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/store
git commit -m "feat: add vcs provider user store"
```

---

### Task 3: Enforce VCS User Limit In CheckRelease

**Files:**
- Modify: `backend/api/v1/release_service.go`
- Modify: `backend/server/grpc_routes.go`
- Modify: `backend/api/v1/release_service_check.go`
- Create: `backend/api/v1/vcs_provider_user.go`
- Modify: `backend/tests/gitops_test.go`

- [ ] **Step 1: Write CheckRelease integration tests**

Add tests to `backend/tests/gitops_test.go`:

```go
func setupGitOpsVCSCheckTarget(t *testing.T, ctx context.Context, ctl *controller, prefix string) (*v1pb.Project, string) {
	t.Helper()
	a := require.New(t)

	projectID := generateRandomString(prefix)
	projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Name:              fmt.Sprintf("projects/%s", projectID),
			Title:             projectID,
			AllowSelfApproval: true,
		},
		ProjectId: projectID,
	}))
	a.NoError(err)

	instanceRootDir := t.TempDir()
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, prefix+"-instance")
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       prefix + "-instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/test"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)

	databaseName := prefix + "_db"
	a.NoError(ctl.createDatabase(ctx, projectResp.Msg, instanceResp.Msg, nil, databaseName, ""))
	return projectResp.Msg, fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, databaseName)
}

func TestGitOpsCheckReleaseVCSUserTracking(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	project, target := setupGitOpsVCSCheckTarget(t, ctx, ctl, "vcs-user-tracking")

	checkResp, err := ctl.releaseServiceClient.CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent: project.Name,
		Release: &v1pb.Release{
			Type: v1pb.Release_VERSIONED,
			Files: []*v1pb.Release_File{{
				Path:      "migrations/001.sql",
				Version:   "001",
				Statement: []byte("CREATE TABLE vcs_tracking_test(id INT);"),
			}},
		},
		Targets: []string{target},
		VcsUser: &v1pb.VCSUser{
			VcsType:     v1pb.VCSType_GITHUB,
			UserId:      "1001",
			UserName:    "alice",
			DisplayName: "Alice",
		},
	}))
	a.NoError(err)
	a.NotNil(checkResp.Msg)
}

func TestGitOpsCheckReleaseVCSUserValidation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	project, target := setupGitOpsVCSCheckTarget(t, ctx, ctl, "vcs-user-validation")

	_, err := ctl.releaseServiceClient.CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent: project.Name,
		Release: &v1pb.Release{
			Type: v1pb.Release_VERSIONED,
			Files: []*v1pb.Release_File{{
				Path:      "migrations/001.sql",
				Version:   "001",
				Statement: []byte("SELECT 1;"),
			}},
		},
		Targets: []string{target},
		VcsUser: &v1pb.VCSUser{VcsType: v1pb.VCSType_GITHUB},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test -v -count=1 ./backend/tests -run '^(TestGitOpsCheckReleaseVCSUserTracking|TestGitOpsCheckReleaseVCSUserValidation)$' -timeout 5m
```

Expected: FAIL until service enforcement is implemented and generated protos are present.

- [ ] **Step 3: Inject LicenseService into ReleaseService**

In `backend/api/v1/release_service.go`, add:

```go
licenseService *enterprise.LicenseService
```

to `ReleaseService`, import `github.com/bytebase/bytebase/backend/enterprise`, and update constructor:

```go
func NewReleaseService(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
	licenseService *enterprise.LicenseService,
) *ReleaseService {
	return &ReleaseService{
		store:          store,
		sheetManager:   sheetManager,
		dbFactory:      dbFactory,
		licenseService: licenseService,
	}
}
```

In `backend/server/grpc_routes.go`, change:

```go
releaseService := apiv1.NewReleaseService(stores, sheetManager, dbFactory, licenseService)
```

- [ ] **Step 4: Add attribution validation and touch helper**

In `backend/api/v1/release_service_check.go`, add:

```go
const vcsProviderUserActiveWindow = 90 * 24 * time.Hour

func (s *ReleaseService) touchVCSProviderUser(ctx context.Context, workspaceID string, vcsUser *v1pb.VCSUser) error {
	if vcsUser == nil {
		return nil
	}
	if vcsUser.VcsType == v1pb.VCSType_VCS_TYPE_UNSPECIFIED {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("vcs_user.vcs_type is required"))
	}
	if vcsUser.UserId == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("vcs_user.user_id is required"))
	}
	ok, err := s.store.TouchVCSProviderUser(ctx, workspaceID, &store.VCSProviderUserMessage{
		VCSType: vcsUser.VcsType,
		UserID:  vcsUser.UserId,
		Payload: &storepb.VCSProviderUserPayload{
			UserName:    vcsUser.UserName,
			DisplayName: vcsUser.DisplayName,
		},
	}, vcsProviderUserActiveWindow, s.licenseService.GetUserLimit(ctx, workspaceID))
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to touch VCS provider user"))
	}
	if !ok {
		return connect.NewError(connect.CodeResourceExhausted, errors.New("new VCS user would exceed the license user limit; increase the license user limit or wait for inactive VCS users to age out of the 90-day window"))
	}
	return nil
}
```

Validate attribution in `CheckRelease` immediately after the project exists check:

```go
if err := validateVCSProviderUser(request.VcsUser); err != nil {
	return nil, err
}
```

Call the touch helper after target resolution and release-file validation, before SQL review and database-heavy checks:

```go
if err := s.touchVCSProviderUser(ctx, workspaceID, request.VcsUser); err != nil {
	return nil, err
}
```

- [ ] **Step 5: Run CheckRelease tests**

Run:

```bash
go test -v -count=1 ./backend/tests -run '^(TestGitOpsCheckReleaseVCSUserTracking|TestGitOpsCheckReleaseVCSUserValidation)$' -timeout 5m
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/api/v1 backend/server backend/tests
git commit -m "feat: enforce vcs user limit in check release"
```

---

### Task 4: Add Actuator Count And Subscription CSV Export

**Files:**
- Modify: `backend/api/v1/actuator_service.go`
- Modify: `backend/api/v1/subscription_service.go`
- Modify: `backend/tests/gitops_test.go`

- [ ] **Step 1: Write service test for active count and export**

Add a focused backend test to `backend/tests/gitops_test.go`:

```go
func TestVCSProviderUserActuatorAndExport(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	project, target := setupGitOpsVCSCheckTarget(t, ctx, ctl, "vcs-user-export")
	_, err = ctl.releaseServiceClient.CheckRelease(ctx, connect.NewRequest(&v1pb.CheckReleaseRequest{
		Parent: project.Name,
		Release: &v1pb.Release{
			Type: v1pb.Release_VERSIONED,
			Files: []*v1pb.Release_File{{
				Path:      "migrations/001.sql",
				Version:   "001",
				Statement: []byte("CREATE TABLE vcs_export_test(id INT);"),
			}},
		},
		Targets: []string{target},
		VcsUser: &v1pb.VCSUser{
			VcsType:     v1pb.VCSType_GITHUB,
			UserId:      "1001",
			UserName:    "alice",
			DisplayName: "Alice",
		},
	}))
	a.NoError(err)

	info, err := ctl.actuatorServiceClient.GetActuatorInfo(ctx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	a.NoError(err)
	a.Equal(int32(1), info.Msg.ActiveVcsUserCount)

	body, err := ctl.subscriptionServiceClient.ExportVCSProviderUsers(ctx, connect.NewRequest(&v1pb.ExportVCSProviderUsersRequest{}))
	a.NoError(err)
	a.Equal("text/csv; charset=utf-8", body.Msg.ContentType)
	a.Contains(string(body.Msg.Data), "vcs_type,user_id,user_name,display_name,last_seen_at")
	a.Contains(string(body.Msg.Data), "GITHUB,1001,alice,Alice,")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test -v -count=1 ./backend/tests -run ^TestVCSProviderUserActuatorAndExport$ -timeout 5m
```

Expected: FAIL because actuator field and export RPC implementation are missing.

- [ ] **Step 3: Populate actuator count**

In `backend/api/v1/actuator_service.go`, after `UserCountInIam` is computed, add:

```go
activeVCSUserCount, err := s.store.CountActiveVCSProviderUsers(ctx, workspaceID, vcsProviderUserActiveWindow)
if err != nil {
	return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to count active VCS users"))
}
serverInfo.ActiveVcsUserCount = int32(activeVCSUserCount)
```

Create `backend/api/v1/vcs_provider_user.go` with the shared active-window constant:

```go
package v1

import "time"

const vcsProviderUserActiveWindow = 90 * 24 * time.Hour
```

- [ ] **Step 4: Implement CSV export**

In `backend/api/v1/subscription_service.go`, import:

```go
import (
	"bytes"
	"encoding/csv"
)

httpbody "google.golang.org/genproto/googleapis/api/httpbody"
```

Add:

```go
func (s *SubscriptionService) ExportVCSProviderUsers(ctx context.Context, _ *connect.Request[v1pb.ExportVCSProviderUsersRequest]) (*connect.Response[httpbody.HttpBody], error) {
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	users, err := s.store.ListActiveVCSProviderUsers(ctx, workspaceID, vcsProviderUserActiveWindow)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to list active VCS provider users"))
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write([]string{"vcs_type", "user_id", "user_name", "display_name", "last_seen_at"}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, user := range users {
		if err := writer.Write([]string{
			user.VCSType.String(),
			user.UserID,
			user.Payload.GetUserName(),
			user.Payload.GetDisplayName(),
			user.LastSeenAt.UTC().Format(time.RFC3339),
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&httpbody.HttpBody{
		ContentType: "text/csv; charset=utf-8",
		Data:        buf.Bytes(),
	}), nil
}
```

- [ ] **Step 5: Run service test**

Run:

```bash
go test -v -count=1 ./backend/tests -run ^TestVCSProviderUserActuatorAndExport$ -timeout 5m
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/api/v1 backend/tests
git commit -m "feat: expose active vcs user usage"
```

---

### Task 5: Add bytebase-release VCS Attribution

**Files:**
- Modify: `action/world/world.go`
- Create: `action/command/vcs_user.go`
- Create: `action/command/vcs_user_test.go`
- Modify: `action/command/check.go`

- [ ] **Step 1: Write attribution tests**

Create `action/command/vcs_user_test.go`:

```go
package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/action/world"
)

func TestGetVCSUserFromGitHubPullRequest(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":1001,"login":"alice","name":"Alice","type":"User"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	user := getVCSUser(world.GitHub)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_GITHUB, user.VcsType)
	require.Equal(t, "1001", user.UserId)
	require.Equal(t, "alice", user.UserName)
	require.Equal(t, "Alice", user.DisplayName)
}

func TestGetVCSUserSkipsGitHubBot(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":41898282,"login":"github-actions[bot]","type":"Bot"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)
	require.Nil(t, getVCSUser(world.GitHub))
}

func TestGetVCSUserFromGitLabMergeRequest(t *testing.T) {
	t.Setenv("CI_PIPELINE_SOURCE", "merge_request_event")
	t.Setenv("GITLAB_USER_ID", "2002")
	t.Setenv("GITLAB_USER_LOGIN", "bob")
	t.Setenv("GITLAB_USER_NAME", "Bob")
	user := getVCSUser(world.GitLab)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_GITLAB, user.VcsType)
	require.Equal(t, "2002", user.UserId)
	require.Equal(t, "bob", user.UserName)
	require.Equal(t, "Bob", user.DisplayName)
}

func TestGetVCSUserFromBitbucketPullRequest(t *testing.T) {
	t.Setenv("BITBUCKET_PR_ID", "10")
	t.Setenv("BITBUCKET_STEP_TRIGGERER_UUID", "{3003}")
	t.Setenv("BITBUCKET_STEP_TRIGGERER_USERNAME", "carol")
	user := getVCSUser(world.Bitbucket)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_BITBUCKET, user.VcsType)
	require.Equal(t, "3003", user.UserId)
	require.Equal(t, "carol", user.UserName)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test -v -count=1 ./action/command -run ^TestGetVCSUser
```

Expected: FAIL because `getVCSUser` does not exist.

- [ ] **Step 3: Implement attribution helper**

Create `action/command/vcs_user.go`:

```go
package command

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func getVCSUser(platform world.JobPlatform) *v1pb.VCSUser {
	switch platform {
	case world.GitHub:
		return getGitHubVCSUser()
	case world.GitLab:
		return getGitLabVCSUser()
	case world.Bitbucket:
		return getBitbucketVCSUser()
	default:
		return nil
	}
}

func getGitHubVCSUser() *v1pb.VCSUser {
	if os.Getenv("GITHUB_EVENT_NAME") != "pull_request" {
		return nil
	}
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil
	}
	data, err := os.ReadFile(eventPath)
	if err != nil {
		return nil
	}
	var event struct {
		PullRequest struct {
			User struct {
				ID    int64  `json:"id"`
				Login string `json:"login"`
				Name  string `json:"name"`
				Type  string `json:"type"`
			} `json:"user"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return nil
	}
	if event.PullRequest.User.ID == 0 || strings.EqualFold(event.PullRequest.User.Type, "Bot") {
		return nil
	}
	return &v1pb.VCSUser{
		VcsType:     v1pb.VCSType_GITHUB,
		UserId:      strconv.FormatInt(event.PullRequest.User.ID, 10),
		UserName:    event.PullRequest.User.Login,
		DisplayName: event.PullRequest.User.Name,
	}
}

func getGitLabVCSUser() *v1pb.VCSUser {
	if os.Getenv("CI_PIPELINE_SOURCE") != "merge_request_event" {
		return nil
	}
	userID := os.Getenv("GITLAB_USER_ID")
	if userID == "" {
		return nil
	}
	userName := os.Getenv("GITLAB_USER_LOGIN")
	if userName == "" {
		userName = os.Getenv("GITLAB_USER_NAME")
	}
	return &v1pb.VCSUser{
		VcsType:     v1pb.VCSType_GITLAB,
		UserId:      userID,
		UserName:    userName,
		DisplayName: os.Getenv("GITLAB_USER_NAME"),
	}
}

func getBitbucketVCSUser() *v1pb.VCSUser {
	if os.Getenv("BITBUCKET_PR_ID") == "" {
		return nil
	}
	userID := strings.Trim(os.Getenv("BITBUCKET_STEP_TRIGGERER_UUID"), "{}")
	if userID == "" {
		return nil
	}
	return &v1pb.VCSUser{
		VcsType:  v1pb.VCSType_BITBUCKET,
		UserId:   userID,
		UserName: os.Getenv("BITBUCKET_STEP_TRIGGERER_USERNAME"),
	}
}
```

- [ ] **Step 4: Attach attribution in check command**

In `action/command/check.go`, change request construction:

```go
checkReleaseRequest := &v1pb.CheckReleaseRequest{
	Parent: w.Project,
	Release: &v1pb.Release{
		Files: releaseFiles,
		Type:  releaseType,
	},
	Targets:     w.Targets,
	CustomRules: w.CustomRules,
	VcsUser:     getVCSUser(platform),
}
checkReleaseResponse, err := client.checkRelease(cmd.Context(), checkReleaseRequest)
```

- [ ] **Step 5: Run action tests**

Run:

```bash
go test -v -count=1 ./action/command -run ^TestGetVCSUser
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add action
git commit -m "feat(action): send vcs user attribution"
```

---

### Task 6: Add Subscription Page VCS Count And Download

**Files:**
- Modify: `frontend/src/react/stores/app/types.ts`
- Modify: `frontend/src/react/stores/app/workspace.ts`
- Modify: `frontend/src/react/hooks/useAppState.ts`
- Modify: `frontend/src/react/pages/settings/SubscriptionPage.tsx`
- Modify: `frontend/src/react/pages/settings/SubscriptionPage.test.tsx`
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json`
- Modify: `frontend/src/locales/es-ES.json`
- Modify: `frontend/src/locales/ja-JP.json`
- Modify: `frontend/src/locales/vi-VN.json`
- Modify: `frontend/src/react/locales/en-US.json`
- Modify: `frontend/src/react/locales/zh-CN.json`
- Modify: `frontend/src/react/locales/es-ES.json`
- Modify: `frontend/src/react/locales/ja-JP.json`
- Modify: `frontend/src/react/locales/vi-VN.json`

- [ ] **Step 1: Write frontend tests**

In `frontend/src/react/pages/settings/SubscriptionPage.test.tsx`, add tests:

```tsx
it("renders active VCS user count separately from IAM user count", () => {
  mockUseServerState.mockReturnValue({
    isSaaSMode: false,
    userCountInIam: 4,
    activeVcsUserCount: 2,
    totalInstanceCount: 3,
    activatedInstanceCount: 3,
    workspaceResourceName: "workspaces/default",
  });
  mockUseSubscriptionState.mockReturnValue({
    userCountLimit: 20,
    instanceCountLimit: 10,
    instanceLicenseCount: Number.MAX_VALUE,
    currentPlan: PlanType.TEAM,
    isFreePlan: false,
    isTrialing: false,
    isExpired: false,
    showTrial: false,
    trialingDays: 0,
    expireAt: "",
    hasUnifiedInstanceLicense: true,
    uploadLicense: vi.fn(),
  });

  render(<SubscriptionPage />);
  expect(screen.getByText("Active VCS users")).toBeInTheDocument();
  expect(screen.getByText("2")).toBeInTheDocument();
  expect(screen.getAllByText("20")).not.toHaveLength(0);
});

it("downloads active VCS users csv", async () => {
  subscriptionServiceClientConnect.exportVCSProviderUsers = vi.fn().mockResolvedValue({
    contentType: "text/csv; charset=utf-8",
    data: new TextEncoder().encode("vcs_type,user_id,user_name,display_name,last_seen_at\n"),
  });
  render(<SubscriptionPage />);
  await userEvent.click(screen.getByRole("button", { name: "Download active VCS users" }));
  expect(subscriptionServiceClientConnect.exportVCSProviderUsers).toHaveBeenCalledWith({});
});
```

Adjust mocks to match the existing test file helpers and i18n test setup.

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
pnpm --dir frontend test -- SubscriptionPage.test.tsx
```

Expected: FAIL because state selector and UI do not exist.

- [ ] **Step 3: Add app-store selector**

In `frontend/src/react/stores/app/types.ts`, add:

```ts
activeVcsUserCount: () => number;
```

In `frontend/src/react/stores/app/workspace.ts`, add:

```ts
activeVcsUserCount: () => get().serverInfo?.activeVcsUserCount ?? 0,
```

In `frontend/src/react/hooks/useAppState.ts`, read and return it:

```ts
const activeVcsUserCount = useAppStore((state) => state.activeVcsUserCount());
```

and include `activeVcsUserCount` in the returned object.

- [ ] **Step 4: Add UI and download action**

In `frontend/src/react/pages/settings/SubscriptionPage.tsx`, import:

```ts
import dayjs from "dayjs";
import { Download } from "lucide-react";
import { subscriptionServiceClientConnect } from "@/connect";
```

Read the new state:

```ts
const {
  isSaaSMode,
  userCountInIam,
  activeVcsUserCount,
  totalInstanceCount,
  activatedInstanceCount,
  workspaceResourceName,
} = useServerState();
```

Add a helper:

```ts
const handleDownloadVcsUsers = async () => {
  try {
    const response =
      await subscriptionServiceClientConnect.exportVCSProviderUsers({});
    const blob = new Blob([response.data], {
      type: response.contentType || "text/csv; charset=utf-8",
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `active-vcs-users.${dayjs().format("YYYY-MM-DDTHH-mm-ss")}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  } catch (error) {
    notify({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.failed"),
      description: (error as { message?: string }).message,
    });
  }
};
```

Add a stat next to the existing user count:

```tsx
<div className="flex flex-col text-left">
  <div className="text-main flex items-center gap-x-2">
    {t("subscription.active-vcs-users")}
    {allowManage && (
      <Button
        variant="ghost"
        size="icon"
        aria-label={t("subscription.download-active-vcs-users")}
        onClick={handleDownloadVcsUsers}
      >
        <Download className="size-4" />
      </Button>
    )}
  </div>
  <div className="mt-1 text-4xl flex items-center gap-2">
    {activeVcsUserCount}
    <span className="font-mono text-control-light">/</span>
    {userLimit}
  </div>
</div>
```

- [ ] **Step 5: Add locale strings**

Add these keys under the existing `subscription` namespace in every locale file listed in this task. Use the English strings for locales where no translation is available:

```json
"active-vcs-users": "Active VCS users",
"download-active-vcs-users": "Download active VCS users"
```

Place the keys under the existing `subscription` namespace. Add matching keys to other locale JSON files using the English strings if translations are not available.

- [ ] **Step 6: Run frontend tests**

Run:

```bash
pnpm --dir frontend test -- SubscriptionPage.test.tsx
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend
git commit -m "feat(frontend): show active vcs user usage"
```

---

### Task 7: Final Verification

**Files:**
- All modified files from Tasks 1-6.

- [ ] **Step 1: Format Go files**

Run:

```bash
gofmt -w backend/store/vcs_provider_user.go backend/store/vcs_provider_user_test.go backend/api/v1/release_service.go backend/api/v1/release_service_check.go backend/api/v1/actuator_service.go backend/api/v1/subscription_service.go backend/tests/gitops_test.go action/command/vcs_user.go action/command/vcs_user_test.go action/command/check.go
```

Expected: command exits 0.

- [ ] **Step 2: Format and lint proto**

Run:

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
```

Expected: all commands exit 0.

- [ ] **Step 3: Run focused backend tests**

Run:

```bash
go test -v -count=1 ./backend/migrator -run ^TestLatestVersion$
go test -v -count=1 ./backend/store -run ^TestVCSProviderUser
go test -v -count=1 ./backend/tests -run '^(TestGitOpsCheckReleaseVCSUserTracking|TestGitOpsCheckReleaseVCSUserValidation|TestVCSProviderUserActuatorAndExport)$' -timeout 5m
go test -v -count=1 ./action/command -run ^TestGetVCSUser
```

Expected: all PASS.

- [ ] **Step 4: Run frontend checks**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test -- SubscriptionPage.test.tsx
```

Expected: all PASS.

- [ ] **Step 5: Run Go lint repeatedly until clean**

Run:

```bash
golangci-lint run --fix --allow-parallel-runners
golangci-lint run --allow-parallel-runners
```

If the second command reports issues, fix them and rerun `golangci-lint run --allow-parallel-runners` until it reports no issues.

- [ ] **Step 6: Build backend**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: build exits 0.

- [ ] **Step 7: Inspect final diff**

Run:

```bash
git status --short
git diff --stat
git diff --check
```

Expected: only intended files are modified and `git diff --check` exits 0.

- [ ] **Step 8: Final commit**

```bash
git add .
git commit -m "feat: track vcs provider users"
```

---

## Self-Review

- Spec coverage: Tasks cover schema, JSONB payload, request attribution, validated-request touch behavior, active limit enforcement, actuator count, `HttpBody` CSV export, Subscription UI count/download, bytebase-release provider attribution, bot omission, and required tests.
- Placeholder scan: This plan uses concrete filenames, commands, field names, messages, and test names. It does not leave open implementation markers.
- Type consistency: The plan consistently uses `VCSUser`, `vcs_user`, `vcs_provider_user`, `user_id`, `user_name`, `display_name`, `last_seen_at`, and `active_vcs_user_count`.
