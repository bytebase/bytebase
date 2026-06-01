# VCS Provider User Limit Design

## Context

Bytebase license seats currently count Bytebase IAM users. In GitOps workflows, one Bytebase automation identity can represent many GitHub, GitLab, or Bitbucket users. The product needs to track VCS pull request or merge request creators, enforce a VCS user pool against the same license user limit, and show workspace admins the active VCS usage count.

This design follows the linked "GitOps User Limit Check" document with one explicit lifecycle change: Bytebase records VCS activity after a `CheckRelease` request has valid attribution, valid targets, and valid release files, before running the heavier release checks.

## Goals

- Track non-bot GitHub, GitLab, and Bitbucket pull request or merge request creators observed through bytebase-release `CheckRelease`.
- Enforce the active VCS user pool against `LicenseService.GetUserLimit`.
- Keep VCS users separate from Bytebase IAM users. The two pools are not unioned and are not deduplicated by email.
- Show `active_vcs_user_count` on the Subscription page through `ActuatorInfo`.
- Provide a CSV export of active VCS users.
- Preserve backward compatibility for old bytebase-release clients and manual callers that do not send VCS attribution.

## Non-Goals

- Do not create Bytebase principals for VCS users.
- Do not combine IAM user count and VCS user count into one limit calculation.
- Do not use email for VCS user identity or deduplication.
- Do not store bot-authored checks.
- Do not build a paginated VCS user table in the UI for this iteration.
- Do not add manual delete, deactivate, or override controls for VCS users.

## Definitions

- VCS user: a non-bot pull request or merge request creator submitted by bytebase-release in `CheckReleaseRequest.vcs_user`.
- Active VCS user: a tracked VCS user whose `last_seen_at` is within the active window.
- Active window: 90 days.
- Identity key: `(workspace, vcs_type, user_id)`.

## Database Schema

Add a workspace-scoped table:

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

Add a store payload proto:

```proto
message VCSProviderUserPayload {
  string user_name = 1;
  string display_name = 2;
}
```

Only identity and query fields are columns. Mutable display metadata lives in `payload` to match repository conventions for JSONB table payloads.

The composite primary key is intentional. `(workspace, vcs_type, user_id)` is the natural identity for a VCS user, prevents duplicate identity rows, and provides the `ON CONFLICT` target for race-safe touch/upsert behavior. The table does not need a surrogate `id` because no API or child table addresses a VCS user independently of that identity tuple.

## API Shape

Extend `CheckReleaseRequest` with optional attribution:

```proto
message CheckReleaseRequest {
  string parent = 1;
  Release release = 2;
  repeated string targets = 3;
  string custom_rules = 4;
  VCSUser vcs_user = 5;
}

message VCSUser {
  VCSType vcs_type = 1;
  string user_id = 2;
  string user_name = 3;
  string display_name = 4;
}
```

`vcs_user` is optional. When it is absent, Bytebase runs the existing `CheckRelease` behavior without VCS tracking or VCS user limit enforcement.

Add `active_vcs_user_count` to `ActuatorInfo`. `ActuatorService.GetActuatorInfo` computes it for the current workspace using the same 90-day active window.

Add a CSV export RPC to `SubscriptionService`:

```proto
import "google/api/httpbody.proto";

rpc ExportVCSProviderUsers(ExportVCSProviderUsersRequest) returns (google.api.HttpBody) {
  option (google.api.http) = {
    get: "/v1/subscription:vcsProviderUsersExport"
  };
  option (bytebase.v1.permission) = "bb.subscription.manage";
  option (bytebase.v1.auth_method) = IAM;
}

message ExportVCSProviderUsersRequest {}
```

The response uses:

- `content_type`: `text/csv; charset=utf-8`
- `data`: CSV bytes

CSV columns:

```text
vcs_type,user_id,user_name,display_name,last_seen_at
```

The export includes only active users, sorted by `last_seen_at DESC`. Empty exports still return the header row.

## CheckRelease Flow

For an attributed request:

1. Parse workspace and project from the request.
2. Validate the project exists.
3. Validate `vcs_user.vcs_type != VCS_TYPE_UNSPECIFIED` and `vcs_user.user_id != ""`.
4. Validate target databases or database groups exist.
5. Validate and sanitize release files.
6. Touch the VCS user before SQL review and database-heavy release checks.
7. If touch succeeds, continue with existing `CheckRelease`.
8. If touch rejects because the active VCS user limit is reached, return `ResourceExhausted`.

Touch behavior:

- If the VCS user exists and is active, update `last_seen_at = now()` and refresh payload.
- If the VCS user is new or inactive, count active VCS users in the workspace.
- If the active count is below `LicenseService.GetUserLimit`, insert or update the row with `last_seen_at = now()`.
- If the active count is at or above the limit, reject and leave the row unchanged.

This intentionally records activity only after Bytebase verifies the project exists, attribution fields pass validation, targets exist, and release files are valid. Invalid targets, empty targets, missing releases, malformed attribution, and invalid release files do not consume VCS seats. Later SQL review failures or internal check failures do not roll back `last_seen_at`.

For an unattributed request:

- Skip VCS tracking and VCS user limit enforcement.
- Preserve existing behavior for older bytebase-release versions and manual API callers.

For bot-authored provider events:

- bytebase-release sends no `vcs_user`.
- Bytebase does not track the bot event and does not show it in export output.

## Concurrency

The active-count decision and upsert must be atomic for new or inactive users. The store will provide `TouchVCSProviderUser`, taking the workspace, VCS user, active window, and limit.

The method should use a workspace-scoped advisory lock or equivalent transaction-level serialization around:

- checking whether the target user is already active,
- counting active VCS users,
- inserting or updating the row.

The lock must not wrap release SQL review or target database checks.

## UI

The Subscription page keeps the existing IAM user usage stat unchanged and adds a separate stat:

```text
Active VCS users
N / user limit
```

The value comes from `ActuatorInfo.active_vcs_user_count` and the existing subscription user limit computation.

Next to the stat, show a download action for users with `bb.subscription.manage`. The action calls `ExportVCSProviderUsers`, creates a CSV file from the `HttpBody` data, and downloads it as:

```text
active-vcs-users.<timestamp>.csv
```

No VCS user table is shown in this iteration.

## bytebase-release Attribution

bytebase-release should populate `CheckReleaseRequest.vcs_user` when it can identify a non-bot PR/MR creator:

- GitHub: pull request creator ID, login, and display name when available.
- GitLab: merge request creator ID, username, and display name when available.
- Bitbucket: pull request creator ID, username or nickname, and display name when available.

If the provider event identifies the creator as a bot, bytebase-release omits `vcs_user`.

## Error Handling

- Missing `vcs_user`: allowed, no VCS tracking.
- `vcs_user.vcs_type` unspecified: `InvalidArgument`.
- `vcs_user.user_id` empty: `InvalidArgument`.
- Active VCS user limit reached for a new or inactive user: `ResourceExhausted`.
- Export failures: surface the backend error through the standard frontend critical notification.

The limit error should explain that a new VCS user would exceed the license user limit and that admins can increase the license user limit or wait for inactive VCS users to age out of the 90-day window.

## Testing

Backend store tests:

- touching a new user inserts a row.
- touching an active existing user refreshes `last_seen_at`.
- inactive users older than 90 days do not count against the active limit.
- a new or inactive user is rejected when active count equals the limit.
- CSV export includes the header, active rows, sorted order, and excludes inactive rows.

Backend service tests:

- `CheckRelease` without `vcs_user` preserves existing behavior.
- valid `vcs_user` records activity after request target and release-file validation.
- invalid targets and empty targets do not record VCS activity.
- limit rejection returns `ResourceExhausted`.
- malformed attribution returns `InvalidArgument`.
- oversized or control-character attribution fields return `InvalidArgument`.
- `ActuatorInfo.active_vcs_user_count` reflects the active window.

Frontend tests:

- Subscription page renders active VCS user usage separately from IAM user usage.
- Download action calls the export RPC and downloads a CSV Blob.
- Locale strings cover the new label and download action.

Action tests:

- GitHub PR creator maps to `VCSUser`.
- GitLab MR creator maps to `VCSUser`.
- Bitbucket PR creator maps to `VCSUser`.
- Bot creators produce no attribution.
- The check command includes `vcs_user` when attribution is available.

## Migration And Generation

- Add a new migration file under the next migration version and update `backend/migrator/migration/LATEST.sql`.
- Update `TestLatestVersion` after adding the migration.
- Add the store payload proto and run proto generation.
- Update generated frontend proto-es files through the existing `buf generate` workflow.
