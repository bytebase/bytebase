# V1 API Audit Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add reliable, non-secret-bearing audit records for the uncovered V1 security and mutation APIs while explicitly preserving the intentional opt-outs for high-volume reads, synchronization, token refresh, and saved query autosave.

**Architecture:** Keep the existing proto-driven audit interceptor as the single audit path. Each covered RPC receives `option (bytebase.v1.audit) = true`; `backend/api/v1/audit.go` remains responsible for canonical resource extraction and cloning/redacting request and response messages before serialization. Standard resource extraction reuses the method-aware proto reflection helper already shared with ACL evaluation, while explicit audit cases are reserved for canonical create names and non-standard request shapes. Unauthenticated handlers resolve a validated workspace through `common.SetAuditWorkspaceID`, and focused end-to-end tests verify that annotations result in persisted rows rather than merely changing descriptors.

**Tech Stack:** Protocol Buffers, Connect RPC, Go, `protoregistry`, `protojson`, Bytebase audit interceptor/store, Testify, backend integration test harness.

## Global Constraints

- Never put credentials, tokens, raw SQL, AI prompts, saved query content, exported files, license text, webhook URLs, or payment-session data in an audit request or response.
- Every redactor must clone before modification; `getRequestString` and `getResponseString` must not mutate live handler messages.
- Audit parents must be canonical `workspaces/{workspace}` or `projects/{project}` names; project-scoped actions must not leak into the workspace-scoped audit stream.
- Keep `Refresh`, `SyncDatabase`, `BatchSyncDatabases`, `SyncInstance`, `BatchSyncInstances`, `UpdateSavedQuery`, `UpdateSavedQueryOrganizer`, and `BatchUpdateSavedQueryOrganizer` unaudited in this change.
- Keep `SwitchWorkspace` unaudited because it changes session/token context rather than workspace resources or authorization grants.
- Keep `TestWebhook`, `TestIdentityProvider`, `TestEmailSetting`, and `AIService.Chat` unaudited because they are non-mutating operations.
- Keep POST-based searches, CEL parse/deparse, schema diffs, release checks, instance database discovery, and rollback previews unaudited.
- Do not add blanket auditing to saved query reads. The saved-query privacy model owns metadata-only search and conditional auditing for another user's private content.
- Run the repository's proto workflow after every proto batch: `buf format -w proto`, `buf lint proto`, then `(cd proto && buf generate)` so later commands remain at the repository root.
- Run `gofmt -w` on modified Go files and the focused Go tests after each task. Run the full Go lint/build gates before completion.

---

## Coverage Decision

Audit these 24 RPCs in this plan:

| Family | RPCs |
| --- | --- |
| Core lifecycle | `SetupSample`, `CreateProject`, `UpdateProject`, `RunPlanChecks`, `CancelPlanCheckRun`, `DeleteRelease`, `UndeleteRelease`, `BatchCreateRevisions`, `DeleteRevision` |
| Content-bearing lifecycle | `CreateRelease`, `UpdateRelease`, `CreateSavedQuery`, `DeleteSavedQuery` |
| Authentication | `RequestPasswordReset`, `ResetPassword` |
| Project webhooks | `AddWebhook`, `UpdateWebhook`, `RemoveWebhook` |
| Subscription | `UploadLicense`, `CreatePurchase`, `UpdatePurchase`, `CancelPurchase`, `ExportVCSProviderUsers` |
| Sensitive external actions | `AuditLogService.ExportAuditLogs` |

The password-reset RPCs have one deliberate limitation: when the unauthenticated flow has no validated workspace, there is nowhere to persist a workspace-owned audit row. Both `RequestPasswordReset` and the corresponding successful `ResetPassword` therefore remain unaudited for codes issued without workspace context. Workspace-bound flows must be recorded.

---

### Task 1: Low-Risk Lifecycle Audit Coverage

**Files:**
- Modify: `proto/v1/v1/actuator_service.proto`
- Modify: `proto/v1/v1/project_service.proto`
- Modify: `proto/v1/v1/plan_service.proto`
- Modify: `proto/v1/v1/release_service.proto`
- Modify: `proto/v1/v1/revision_service.proto`
- Modify: `backend/api/v1/audit.go`
- Modify: `backend/api/v1/audit_test.go`
- Modify: `backend/api/v1/audit_redaction_test.go`
- Test: `backend/tests/login_audit_test.go`
- Test: `backend/tests/project_instance_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: existing `getResourceFromSingleRequest`, `maskedString`, `proto.CloneOf`, and proto `bytebase.v1.audit` extension.
- Produces: audited descriptors, canonical resource names, and clone-first project payload redaction for nine lifecycle RPCs.

- [x] **Step 1: Extend the resource extraction test first**

Add cases to `TestProjectLifecycleAuditResource`, rename it to `TestLifecycleAuditResource`, and include the matching full RPC procedure in each case:

```go
{name: "create project", method: v1connect.ProjectServiceCreateProjectProcedure, request: &v1pb.CreateProjectRequest{ProjectId: "project-a", Project: &v1pb.Project{}}, want: "projects/project-a"},
{name: "create project ignores nested name", method: v1connect.ProjectServiceCreateProjectProcedure, request: &v1pb.CreateProjectRequest{ProjectId: "project-a", Project: &v1pb.Project{Name: "projects/wrong-project"}}, want: "projects/project-a"},
{name: "update project", method: v1connect.ProjectServiceUpdateProjectProcedure, request: &v1pb.UpdateProjectRequest{Project: &v1pb.Project{Name: "projects/project-a"}}, want: "projects/project-a"},
{name: "create workspace instance", method: v1connect.InstanceServiceCreateInstanceProcedure, request: &v1pb.CreateInstanceRequest{InstanceId: "instance-a", Instance: &v1pb.Instance{}}, want: "instances/instance-a"},
{name: "create project instance", method: v1connect.InstanceServiceCreateInstanceProcedure, request: &v1pb.CreateInstanceRequest{Parent: new("projects/project-a"), InstanceId: "instance-a", Instance: &v1pb.Instance{}}, want: "projects/project-a/instances/instance-a"},
{name: "run plan checks", method: v1connect.PlanServiceRunPlanChecksProcedure, request: &v1pb.RunPlanChecksRequest{Name: "projects/project-a/plans/101"}, want: "projects/project-a/plans/101"},
{name: "cancel plan checks", method: v1connect.PlanServiceCancelPlanCheckRunProcedure, request: &v1pb.CancelPlanCheckRunRequest{Name: "projects/project-a/plans/101/planCheckRun"}, want: "projects/project-a/plans/101/planCheckRun"},
{name: "delete release", method: v1connect.ReleaseServiceDeleteReleaseProcedure, request: &v1pb.DeleteReleaseRequest{Name: "projects/project-a/releases/release-a"}, want: "projects/project-a/releases/release-a"},
{name: "undelete release", method: v1connect.ReleaseServiceUndeleteReleaseProcedure, request: &v1pb.UndeleteReleaseRequest{Name: "projects/project-a/releases/release-a"}, want: "projects/project-a/releases/release-a"},
{name: "batch create revisions", method: v1connect.RevisionServiceBatchCreateRevisionsProcedure, request: &v1pb.BatchCreateRevisionsRequest{Parent: "instances/instance-a/databases/database-a"}, want: "instances/instance-a/databases/database-a"},
{name: "delete revision", method: v1connect.RevisionServiceDeleteRevisionProcedure, request: &v1pb.DeleteRevisionRequest{Name: "instances/instance-a/databases/database-a/revisions/101"}, want: "instances/instance-a/databases/database-a/revisions/101"},
```

Change the assertion to `require.Equal(t, test.want, getRequestResource(test.request, test.method))`.

- [x] **Step 2: Run the unit test and verify the new cases fail**

Run:

```bash
go test ./backend/api/v1 -run '^TestLifecycleAuditResource$' -count=1
```

Expected: FAIL because `getRequestResource` does not yet accept the RPC method or use descriptor metadata for standard request shapes.

- [x] **Step 3: Make resource extraction method-aware**

Pass the full RPC procedure from the interceptor into `getRequestResource`. Preserve explicit cases only where descriptor metadata cannot express the audit resource: authentication email, canonical create names, and non-standard field or batch shapes such as `UpdateDatabaseCatalog.catalog` and `BatchUpdateDatabases.parent`. For all standard annotated requests, extract the short RPC method name and reuse `getResourceFromSingleRequest`:

```go
case *v1pb.CreateInstanceRequest:
	if r.GetParent() == "" {
		return common.FormatInstance(r.GetInstanceId())
	}
	if projectID, err := common.GetProjectID(r.GetParent()); err == nil {
		return common.FormatProjectInstance(projectID, r.GetInstanceId())
	}
	return ""
case *v1pb.CreateProjectRequest:
	return common.FormatProject(r.GetProjectId())
```

This keeps the audit-specific switch small while preserving plan, release, revision, IdP, data-source, setting, and other annotated resource names through the shared reflection path.

- [x] **Step 4: Add failing project payload redaction tests**

In `backend/api/v1/audit_redaction_test.go`, add request cases for both project mutations with `secretSentinel` in a nested webhook URL:

```go
{"create project webhook URL", &v1pb.CreateProjectRequest{Project: &v1pb.Project{
	Webhooks: []*v1pb.Webhook{{Name: "projects/project-a/webhooks/webhook-a", Url: secretSentinel}},
}}},
{"update project webhook URL", &v1pb.UpdateProjectRequest{Project: &v1pb.Project{
	Name: "projects/project-a",
	Webhooks: []*v1pb.Webhook{{Name: "projects/project-a/webhooks/webhook-a", Url: secretSentinel}},
}}},
```

Add a response case for `Project` with the same nested webhook URL. Extend `TestAuditRedactionDoesNotMutateInput` with a project response and assert its original webhook URL remains `secretSentinel` after serialization.

- [x] **Step 5: Run the redaction tests and verify they fail**

```bash
go test ./backend/api/v1 -run '^TestAudit(Request|Response)RedactsCredentials$' -count=1
```

Expected: FAIL because project request and response payloads currently serialize webhook URLs unchanged.

- [x] **Step 6: Implement clone-first project redaction**

Add the nested webhook and project redactors to `backend/api/v1/audit.go`:

```go
func redactWebhook(w *v1pb.Webhook) *v1pb.Webhook {
	if w == nil {
		return nil
	}
	cloned := proto.CloneOf(w)
	if cloned.Url != "" {
		cloned.Url = maskedString
	}
	return cloned
}

func redactProject(p *v1pb.Project) *v1pb.Project {
	if p == nil {
		return nil
	}
	cloned := proto.CloneOf(p)
	for i, webhook := range cloned.Webhooks {
		cloned.Webhooks[i] = redactWebhook(webhook)
	}
	return cloned
}
```

Register `CreateProjectRequest` and `UpdateProjectRequest` in `getRequestString`, cloning the outer request and replacing its nested project with `redactProject`. Register `Project` in `getResponseString`. Preserve project IDs, update masks, names, webhook names/types/titles, and all other non-secret fields.

- [x] **Step 7: Annotate the low-risk RPCs**

Add this option inside each RPC block for `SetupSample`, `CreateProject`, `UpdateProject`, `RunPlanChecks`, `CancelPlanCheckRun`, `DeleteRelease`, `UndeleteRelease`, `BatchCreateRevisions`, and `DeleteRevision`:

```proto
option (bytebase.v1.audit) = true;
```

- [x] **Step 8: Generate and run focused unit tests**

Run:

```bash
buf format -w proto
buf lint proto
(cd proto && buf generate)
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_test.go backend/api/v1/audit_redaction_test.go
go test ./backend/api/v1 -run '^(TestLifecycleAuditResource|TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput)$' -count=1
```

Expected: all commands PASS.

- [x] **Step 9: Add persistence regression coverage for the original symptom**

In `backend/tests/login_audit_test.go`, reuse `ctl.project`, which `StartServerWithExternalPg` already created through `ProjectService.CreateProject`. Query workspace audit logs with:

```go
Filter: `method == "/bytebase.v1.ProjectService/CreateProject"`,
```

Scan the method-matched rows for `Resource == ctl.project.Name` and require exactly one such row; do not assert that the method filter itself returns only one row because setup or future fixtures may create other projects. Assert that row has the workspace parent. Update `ctl.project`, query its project-scoped audit stream for `ProjectService/UpdateProject`, and assert the resource is `ctl.project.Name`.

In `backend/tests/project_instance_test.go`, immediately after the project-scoped instance is created, query its project audit stream for:

```go
Filter: `method == "/bytebase.v1.InstanceService/CreateInstance"`,
```

Assert a row exists with `Resource == projectInstance.Name`. This pins the distinction between “descriptor is annotated” and “row was actually persisted.”

- [x] **Step 10: Run the lifecycle integration tests**

Run:

```bash
go test -v -count=1 ./backend/tests -run '^(TestAuditLogFormat|TestProjectInstanceCoreBehavior)$' -timeout 10m
```

Expected: PASS, including persisted `CreateProject`, `UpdateProject`, and `CreateInstance` rows.

- [x] **Step 11: Commit the low-risk batch**

```bash
git add proto/v1/v1/actuator_service.proto proto/v1/v1/project_service.proto proto/v1/v1/plan_service.proto proto/v1/v1/release_service.proto proto/v1/v1/revision_service.proto backend/api/v1/audit.go backend/api/v1/audit_test.go backend/api/v1/audit_redaction_test.go backend/tests/login_audit_test.go backend/tests/project_instance_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit core lifecycle APIs"
```

---

### Task 2: Release and Saved Query Content Redaction

**Files:**
- Modify: `proto/v1/v1/release_service.proto`
- Modify: `proto/v1/v1/saved_query_service.proto`
- Modify: `backend/api/v1/audit.go`
- Modify: `backend/api/v1/audit_redaction_test.go`
- Modify: `backend/api/v1/audit_test.go`
- Test: `backend/tests/login_audit_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: `getRequestString`, `getResponseString`, `maskedString`, and `proto.CloneOf`.
- Produces: `redactRelease`, `redactSavedQuery`, and redacted create/update request handling.

- [x] **Step 1: Add failing request and response redaction cases**

Add `secretSentinel` as a release statement and saved query content in `TestAuditRequestRedactsCredentials`, and add a saved query response case in `TestAuditResponseRedactsCredentials`:

```go
{"release SQL", &v1pb.CreateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}}},
{"saved query SQL", &v1pb.CreateSavedQueryRequest{SavedQuery: &v1pb.SavedQuery{Content: []byte(secretSentinel)}}},
```

```go
{name: "saved query response SQL", response: &v1pb.SavedQuery{Content: []byte(secretSentinel)}},
```

Because protobuf `bytes` fields are base64-encoded by `protojson`, assert that neither `secretSentinel` nor its base64 representation appears in the serialized payload.

- [x] **Step 2: Run the redaction tests and verify they fail**

```bash
go test ./backend/api/v1 -run '^TestAudit(Request|Response)RedactsCredentials$' -count=1
```

Expected: FAIL because the sentinel appears in serialized audit payloads.

- [x] **Step 3: Implement clone-first content redactors**

Add these helpers to `audit.go`:

```go
func redactRelease(r *v1pb.Release) *v1pb.Release {
	if r == nil {
		return nil
	}
	cloned := proto.CloneOf(r)
	for _, file := range cloned.Files {
		file.Statement = nil
	}
	return cloned
}

func redactSavedQuery(r *v1pb.SavedQuery) *v1pb.SavedQuery {
	if r == nil {
		return nil
	}
	cloned := proto.CloneOf(r)
	cloned.Content = nil
	return cloned
}
```

Add request switch cases for `CreateReleaseRequest`, `UpdateReleaseRequest`, and `CreateSavedQueryRequest`, preserving all non-content fields and update masks. Add response switch cases for `Release` and `SavedQuery` so future handler changes cannot expose statement/content fields through the audit response.

- [x] **Step 4: Verify resource extraction, project ownership, and annotations**

Add method-aware `TestLifecycleAuditResource` cases for release and saved query requests. Their annotated parent, name, and nested resource fields must be resolved by the shared reflection path without adding audit-specific type-switch branches. Declare `SavedQuery` as `bytebase.com/SavedQuery`, and annotate `CreateSavedQueryRequest.parent` and `DeleteSavedQueryRequest.name` with their canonical resource references.

Add `audit = true` to `CreateRelease`, `UpdateRelease`, `CreateSavedQuery`, and `DeleteSavedQuery`.

Ensure both saved query lifecycle RPCs publish their owning project through the authorization context so the interceptor persists their audit rows under `projects/{project}`, not the workspace fallback. The implementation may use the existing declarative resource metadata or an equivalent project-resolution path; the required contract is project-scoped audit ownership.

- [x] **Step 5: Verify redaction and persisted ownership**

Extend `TestAuditRedactionDoesNotMutateInput` with a release and saved query, call the corresponding audit serializer, and assert the original statement/content still equals `secretSentinel`.

Extend the audit integration coverage to create and delete a saved query, then query the project audit stream and assert both rows use `projects/{project}` as their parent. This test must fail if either action is filed only in the workspace audit stream.

- [x] **Step 6: Generate and test**

```bash
buf format -w proto
buf lint proto
(cd proto && buf generate)
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_test.go
go test ./backend/api/v1 -run '^(TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput|TestLifecycleAuditResource)$' -count=1
go test -v -count=1 ./backend/tests -run '^TestAuditLogFormat$' -timeout 10m
```

Expected: PASS, no sentinel in any serialized payload, and saved query lifecycle rows are discoverable in the owning project's audit stream.

- [x] **Step 7: Commit the content-bearing batch**

```bash
git add proto/v1/v1/release_service.proto proto/v1/v1/saved_query_service.proto backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_test.go backend/tests/login_audit_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit release and saved query lifecycle"
```

---

### Task 3: Password Reset Security Events

**Files:**
- Modify: `proto/v1/v1/auth_service.proto`
- Modify: `backend/api/v1/auth_service.go`
- Modify: `backend/api/v1/audit.go`
- Modify: `backend/api/v1/audit_redaction_test.go`
- Test: `backend/tests/login_audit_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: `common.SetAuditWorkspaceID`, `parseOptionalWorkspace`, and existing `redactLoginResponse`.
- Produces: `redactResetPasswordRequest` and validated workspace attribution for password-reset events.

- [x] **Step 1: Add failing credential-redaction tests**

Add a request case containing the sentinel in password-reset `code` and `new_password`. Assert the email remains visible while both secrets are absent.

- [x] **Step 2: Implement request redactors**

```go
func redactResetPasswordRequest(r *v1pb.ResetPasswordRequest) *v1pb.ResetPasswordRequest {
	if r == nil {
		return nil
	}
	cloned := proto.CloneOf(r)
	cloned.Code = maskedString
	cloned.NewPassword = maskedString
	return cloned
}
```

Register the helper in `getRequestString`. Keep `RequestPasswordResetRequest` unchanged because email and workspace are audit identifiers, not credentials.

- [x] **Step 3: Attribute successful and target-valid auth events**

For `RequestPasswordReset`, set the audit workspace only after establishing both of these facts:

- the normalized email belongs to an active `END_USER` account;
- the requested workspace is valid for that account under the existing membership rules.

The password-reset delivery path must apply the same active `END_USER` check before resolving email configuration, storing a verification code, or sending email. Unknown, deleted, service-account, and workload-identity targets remain silent no-ops.

An `allUsers` workspace binding alone must not make an unknown, deleted, service-account, or workload-identity email auditable. Keep the endpoint's current always-success response and do not surface lookup failures; an absent, malformed, unknown, or invalid workspace produces no audit row.

For both password-reset RPCs, the interceptor must accept only the workspace explicitly validated and announced by the handler. An authenticated caller's token workspace must not become a fallback audit parent when validation fails or no workspace was supplied.

For `ResetPassword`, attribute the event only to the validated non-empty workspace captured with the verification code. Set that workspace early enough that membership, password-policy, or update failures after successful code verification can be recorded. Do not invent or select an arbitrary workspace when the code has no workspace context; that path is the explicit no-workspace limitation described above.

- [x] **Step 4: Add audit annotations**

Add `audit = true` to `RequestPasswordReset` and `ResetPassword`.

- [x] **Step 5: Add auth persistence tests**

Extend `backend/tests/login_audit_test.go` to assert:

- a workspace-bound successful password reset row contains the email but neither the submitted code nor new password;
- a workspace-bound request for an active end user produces a row while preserving the endpoint's always-success response;
- unknown, deleted, service-account, and workload-identity emails do not create request audit rows, including when the workspace grants `allUsers`;
- those exclusions also hold when the caller supplies a valid access token;
- no-workspace request and reset flows preserve their existing API behavior but create no audit row, documenting the storage limitation explicitly.

- [x] **Step 6: Generate and run auth tests**

```bash
buf format -w proto
buf lint proto
(cd proto && buf generate)
gofmt -w backend/api/v1/auth_service.go backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/tests/login_audit_test.go
go test ./backend/api/v1 -run '^(TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput)$' -count=1
go test -v -count=1 ./backend/tests -run '^TestAuditLogFormat$' -timeout 10m
```

Expected: PASS with no auth credential present in stored request/response JSON.

- [x] **Step 7: Commit the authentication batch**

Merged in PR #21162.

```bash
git add proto/v1/v1/auth_service.proto backend/api/v1/auth_service.go backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/tests/login_audit_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit authentication security events"
```

---

### Task 3A: Email-Code Login Audit Attribution

**Files:**
- Modify: `backend/api/v1/auth_service.go`
- Modify: `backend/api/v1/audit.go`
- Test: `backend/api/v1/audit_test.go`
- Test: `backend/tests/login_audit_test.go`

**Interfaces:**
- Consumes: the existing `SendEmailLoginCode` audit annotation, `store.GetAccountByEmail`, `store.FindWorkspace`, and `common.SetAuditWorkspaceID`.
- Produces: workspace attribution for `SendEmailLoginCode` without changing its delivery or signup behavior.

- [x] **Step 1: Attribute requests for existing workspace users**

Normalize the requested email and parse the optional workspace once. Announce that workspace to the audit interceptor only when the email belongs to an active `END_USER` account that is a member of the requested workspace under the existing membership rules.

This lookup affects audit attribution only. The parsed workspace ID is still used by the existing email-setting and verification-code paths, and the RPC keeps its existing login and auto-signup behavior.

- [x] **Step 2: Keep signup-capable requests out of the send audit**

An email that does not yet have an active workspace account, including an invited email that may later sign up, does not create a `SendEmailLoginCode` audit row because there is no existing user membership to attribute. Supplying an authenticated caller token must not make such a target auditable.

The successful `Login` request is audited after email-code authentication resolves or creates the user and determines the workspace. This provides the durable security event for email-code signup.

Delivery eligibility and self-host email-code signup policy require separate design discussion and are explicitly outside this audit-only task.

- [x] **Step 3: Constrain and define audit attribution**

`SendEmailLoginCode` may use only the workspace explicitly validated and announced by the handler. Do not fall back to ACL resources or the caller token for this unauthenticated target operation.

For an attributed workspace-bound request, the audit row contains:

- `Parent`: the validated `workspaces/{workspace}`;
- `Resource`: the normalized email address;
- `Request`: email and workspace, with no verification code because this RPC does not accept one;
- `Response`: the empty response on success;
- the standard method, caller when authenticated, status, latency, request metadata, and MCP delegation fields.

An unknown, inactive, non-user, non-member, or workspace-less target produces no row because there is no validated workspace parent. An authenticated caller's token workspace must not change that result.

- [x] **Step 4: Add regression coverage**

Extend resource extraction coverage for normalized `SendEmailLoginCodeRequest.email`. Extend the audit integration test to cover an active member, an invited email without an account, an authenticated unknown-address request, and the subsequent successful email-code `Login` audit row.

```bash
gofmt -w backend/api/v1/auth_service.go backend/api/v1/audit.go backend/api/v1/audit_test.go backend/tests/login_audit_test.go
go test ./backend/api/v1 -run '^TestLifecycleAuditResource$' -count=1
go test -v -count=1 ./backend/tests -run '^TestAuditLogFormat$' -timeout 10m
```

Expected: the existing member request creates a workspace `SendEmailLoginCode` audit row; signup-capable targets do not; and a successful email-code signup creates a workspace `Login` audit row with the code redacted.

---

### Task 4: Subscription and Export Audit Coverage

**Files:**
- Modify: `proto/v1/v1/subscription_service.proto`
- Modify: `backend/api/v1/audit.go`
- Modify: `backend/api/v1/audit_redaction_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: workspace parent fallback from the authenticated context.
- Produces: `redactUploadLicenseRequest`, empty/redacted purchase responses, and metadata-only VCS provider user export responses.

- [x] **Step 1: Add failing redaction cases**

Cover `UploadLicenseRequest.License`, `PurchaseResponse.PaymentUrl`, `PurchaseResponse.SessionId`, and `ExportVCSProviderUsersResponse.Content` with `secretSentinel` in the request/response redaction tables.

- [x] **Step 2: Implement subscription redactors**

Replace the generic `google.api.HttpBody` export result with an RPC-specific response:

```proto
message ExportVCSProviderUsersResponse {
  bytes content = 1;
}
```

Return this message from `ExportVCSProviderUsers` and update the Subscription page download path to read `Content` and use the fixed `text/csv; charset=utf-8` media type. The HTTP/JSON representation now carries base64-encoded `content` instead of returning a raw CSV body.

```go
func redactUploadLicenseRequest(r *v1pb.UploadLicenseRequest) *v1pb.UploadLicenseRequest {
	if r == nil {
		return nil
	}
	return &v1pb.UploadLicenseRequest{License: maskedString}
}

func redactPurchaseResponse(r *v1pb.PurchaseResponse) *v1pb.PurchaseResponse {
	if r == nil {
		return nil
	}
	return &v1pb.PurchaseResponse{}
}
```

For `ExportVCSProviderUsersResponse`, drop `Content` and serialize an empty response. Register these types in the request/response serializer switches.

- [x] **Step 3: Add subscription annotations**

Add `audit = true` to `UploadLicense`, `CreatePurchase`, `UpdatePurchase`, `CancelPurchase`, and `ExportVCSProviderUsers`.

Leave `Resource` empty for these workspace-singleton operations; the authenticated workspace remains the audit parent. Do not invent a non-resource-format subscription name.

- [x] **Step 4: Generate and test**

```bash
buf format -w proto
buf lint proto
(cd proto && buf generate)
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go
go test ./backend/api/v1 -run '^(TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput)$' -count=1
```

Expected: PASS; license, checkout URL/session, and CSV content are absent.

- [ ] **Step 5: Commit the subscription batch**

Left uncommitted for the author to review and commit explicitly.

```bash
git add proto/v1/v1/subscription_service.proto backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit subscription management actions"
```

---

### Task 5: Webhook and Audit Export Actions

**Files:**
- Modify: `proto/v1/v1/project_service.proto`
- Modify: `proto/v1/v1/audit_log_service.proto`
- Modify: `backend/api/v1/audit.go`
- Modify: `backend/api/v1/audit_redaction_test.go`
- Modify: `backend/api/v1/audit_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: existing `redactWebhook` and `maskedString`.
- Produces: metadata-only audit payloads for webhook mutations and audit-log exports.

- [x] **Step 1: Add failing secret/content tests**

Add cases for exported audit-log bytes. The shared project redactor from Task 1 already covers webhook URLs in webhook requests and returned `Project` responses. Keep safe identifiers such as project name, webhook name/title, and export page token.

- [x] **Step 2: Confirm configuration probes remain excluded**

Keep `TestIdentityProvider`, `TestEmailSetting`, and `AIService.Chat` unaudited alongside `TestWebhook`; none of these methods mutates a Bytebase resource.

- [x] **Step 3: Implement request/response redactors**

Implement these exact contracts:

```go
func redactExportAuditLogsResponse(r *v1pb.ExportAuditLogsResponse) *v1pb.ExportAuditLogsResponse
// Return only NextPageToken; drop Content.
```

Register Add/Update/Remove webhook requests using Task 1's `redactWebhook`, and register `ExportAuditLogsResponse` in the response serializer switch.

- [x] **Step 4: Add canonical resource extraction and regression tests**

Add method-aware cases for webhook and export-audit-log requests so their annotated fields are covered by the shared resource path.

- [x] **Step 5: Add annotations**

Add `audit = true` to `AddWebhook`, `UpdateWebhook`, `RemoveWebhook`, and `ExportAuditLogs`.

- [x] **Step 6: Generate and test**

```bash
buf format -w proto
buf lint proto
(cd proto && buf generate)
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_test.go
go test ./backend/api/v1 -run '^(TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput|TestLifecycleAuditResource)$' -count=1
```

Expected: PASS; webhook resources remain canonical and audit rows contain no exported audit-log content.

- [ ] **Step 7: Commit the sensitive-action batch**

Left uncommitted for the author to review and commit explicitly.

```bash
git add proto/v1/v1/project_service.proto proto/v1/v1/audit_log_service.proto backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit sensitive outbound actions"
```

---

### Task 6: Lock the Audit Policy and Run Full Verification

**Files:**
- Create: `backend/api/v1/audit_policy_test.go`
- Modify: `docs/design/v1-api-audit-2026-08.md`

**Interfaces:**
- Consumes: generated method descriptors and `v1pb.E_Audit`.
- Produces: a regression guard for required annotations and documented intentional exclusions.

- [ ] **Step 1: Add the descriptor policy test**

Create a table-driven test using `protoregistry.GlobalFiles.FindDescriptorByName`. The helper must fail when a named method is missing or its audit extension is absent/false:

```go
func requireAuditedMethod(t *testing.T, fullName string) {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(fullName))
	require.NoError(t, err)
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	options := method.Options().(*descriptorpb.MethodOptions)
	require.True(t, proto.HasExtension(options, v1pb.E_Audit), fullName)
	audited, ok := proto.GetExtension(options, v1pb.E_Audit).(bool)
	require.True(t, ok)
	require.True(t, audited, fullName)
}
```

`TestRequiredAPIAuditCoverage` must enumerate every RPC in the Coverage Decision table. Keep the method list explicit so reviewers can see policy changes in code review.

```go
func TestRequiredAPIAuditCoverage(t *testing.T) {
	for _, method := range []string{
		"bytebase.v1.ActuatorService.SetupSample",
		"bytebase.v1.AuditLogService.ExportAuditLogs",
		"bytebase.v1.AuthService.RequestPasswordReset",
		"bytebase.v1.AuthService.ResetPassword",
		"bytebase.v1.PlanService.CancelPlanCheckRun",
		"bytebase.v1.PlanService.RunPlanChecks",
		"bytebase.v1.ProjectService.AddWebhook",
		"bytebase.v1.ProjectService.CreateProject",
		"bytebase.v1.ProjectService.RemoveWebhook",
		"bytebase.v1.ProjectService.UpdateProject",
		"bytebase.v1.ProjectService.UpdateWebhook",
		"bytebase.v1.ReleaseService.CreateRelease",
		"bytebase.v1.ReleaseService.DeleteRelease",
		"bytebase.v1.ReleaseService.UndeleteRelease",
		"bytebase.v1.ReleaseService.UpdateRelease",
		"bytebase.v1.RevisionService.BatchCreateRevisions",
		"bytebase.v1.RevisionService.DeleteRevision",
		"bytebase.v1.SubscriptionService.CancelPurchase",
		"bytebase.v1.SubscriptionService.CreatePurchase",
		"bytebase.v1.SubscriptionService.ExportVCSProviderUsers",
		"bytebase.v1.SubscriptionService.UpdatePurchase",
		"bytebase.v1.SubscriptionService.UploadLicense",
		"bytebase.v1.SavedQueryService.CreateSavedQuery",
		"bytebase.v1.SavedQueryService.DeleteSavedQuery",
	} {
		t.Run(method, func(t *testing.T) {
			requireAuditedMethod(t, method)
		})
	}
}
```

- [ ] **Step 2: Add the intentional-exclusion test**

Add `requireUnauditedMethod` and enumerate the thirteen deliberate exclusions from Global Constraints. This makes adding audit to sync, autosave, session mechanics, or non-mutating operations a conscious policy change instead of accidental drift:

```go
func requireUnauditedMethod(t *testing.T, fullName string) {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(fullName))
	require.NoError(t, err)
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	options := method.Options().(*descriptorpb.MethodOptions)
	if !proto.HasExtension(options, v1pb.E_Audit) {
		return
	}
	audited, ok := proto.GetExtension(options, v1pb.E_Audit).(bool)
	require.True(t, ok)
	require.False(t, audited, fullName)
}

func TestIntentionalAPIAuditExclusions(t *testing.T) {
	for _, method := range []string{
		"bytebase.v1.AIService.Chat",
		"bytebase.v1.AuthService.Refresh",
		"bytebase.v1.AuthService.SwitchWorkspace",
		"bytebase.v1.DatabaseService.BatchSyncDatabases",
		"bytebase.v1.DatabaseService.SyncDatabase",
		"bytebase.v1.InstanceService.BatchSyncInstances",
		"bytebase.v1.InstanceService.SyncInstance",
		"bytebase.v1.IdentityProviderService.TestIdentityProvider",
		"bytebase.v1.ProjectService.TestWebhook",
		"bytebase.v1.SavedQueryService.BatchUpdateSavedQueryOrganizer",
		"bytebase.v1.SavedQueryService.UpdateSavedQuery",
		"bytebase.v1.SavedQueryService.UpdateSavedQueryOrganizer",
		"bytebase.v1.SettingService.TestEmailSetting",
	} {
		t.Run(method, func(t *testing.T) {
			requireUnauditedMethod(t, method)
		})
	}
}
```

- [ ] **Step 3: Run the policy test**

```bash
gofmt -w backend/api/v1/audit_policy_test.go
go test ./backend/api/v1 -run '^Test(RequiredAPIAuditCoverage|IntentionalAPIAuditExclusions)$' -count=1
```

Expected: PASS with all required descriptors audited and all deliberate exclusions unaudited.

- [ ] **Step 4: Document the final policy**

Add an “Audit annotation coverage” section to `docs/design/v1-api-audit-2026-08.md` containing:

- the audited RPC families from the Coverage Decision table;
- the deliberate exclusions and their reasons, including session-only workspace switching and non-mutating configuration probes;
- the rule that serializers must redact credentials/content before annotations are enabled;
- the workspace-resolution limitation for both unauthenticated password-reset RPCs;
- the saved-query read deferral to the saved-query privacy model.

- [ ] **Step 5: Run all required verification gates**

```bash
buf format -w proto
buf lint proto
(cd proto && buf generate)
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_test.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_policy_test.go backend/api/v1/auth_service.go backend/tests/login_audit_test.go backend/tests/project_instance_test.go
go test ./backend/api/v1 -count=1
go test -v -count=1 ./backend/tests -run '^(TestAuditLogFormat|TestProjectInstanceCoreBehavior|TestAdminExecuteAuditLog)$' -timeout 10m
golangci-lint run --allow-parallel-runners
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
git diff --check
```

Expected: every command PASS. Run `golangci-lint run --allow-parallel-runners` repeatedly until it reports no issues, per repository instructions.

- [ ] **Step 6: Inspect generated and security-sensitive diffs**

```bash
git diff --stat
git diff -- proto/v1/v1 backend/api/v1/audit.go backend/api/v1/auth_service.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_policy_test.go backend/tests docs/design/v1-api-audit-2026-08.md
```

Confirm no unrelated generated churn, no raw sentinel/credential fixtures outside tests, and no accidental audit annotation on the deliberate exclusions.

- [ ] **Step 7: Commit the policy guard and documentation**

```bash
git add backend/api/v1/audit_policy_test.go docs/design/v1-api-audit-2026-08.md
git commit -m "test: lock v1 API audit policy"
```

---

## Completion Criteria

- Every RPC in the Coverage Decision table has `option (bytebase.v1.audit) = true` in source and generated descriptors.
- `CreateProject`, `UpdateProject`, and the already-annotated `CreateInstance` have end-to-end persistence assertions.
- Project-scoped rows use project parents and canonical resources; workspace-scoped rows use workspace parents.
- All credential/content sentinels are absent from serialized audit payloads, and redaction does not mutate handler messages.
- Deliberate exclusions remain unaudited and are protected by a descriptor policy test.
- Proto format/lint/generation, focused integration tests, backend unit tests, lint, build, and `git diff --check` pass.
