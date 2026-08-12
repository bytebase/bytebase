# V1 API Audit Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add reliable, non-secret-bearing audit records for the uncovered V1 security and mutation APIs while explicitly preserving the intentional opt-outs for high-volume reads, synchronization, token refresh, and worksheet autosave.

**Architecture:** Keep the existing proto-driven audit interceptor as the single audit path. Each covered RPC receives `option (bytebase.v1.audit) = true`; `backend/api/v1/audit.go` remains responsible for canonical resource extraction and cloning/redacting request and response messages before serialization. Unauthenticated handlers resolve a validated workspace through `common.SetAuditWorkspaceID`, and focused end-to-end tests verify that annotations result in persisted rows rather than merely changing descriptors.

**Tech Stack:** Protocol Buffers, Connect RPC, Go, `protoregistry`, `protojson`, Bytebase audit interceptor/store, Testify, backend integration test harness.

## Global Constraints

- Never put credentials, tokens, raw SQL, AI prompts, worksheet content, exported files, license text, webhook URLs, or payment-session data in an audit request or response.
- Every redactor must clone before modification; `getRequestString` and `getResponseString` must not mutate live handler messages.
- Audit parents must be canonical `workspaces/{workspace}` or `projects/{project}` names; project-scoped actions must not leak into the workspace-scoped audit stream.
- Keep `Refresh`, `SyncDatabase`, `BatchSyncDatabases`, `SyncInstance`, `BatchSyncInstances`, `UpdateWorksheet`, `UpdateWorksheetOrganizer`, and `BatchUpdateWorksheetOrganizer` unaudited in this change.
- Keep `SwitchWorkspace` unaudited because it changes session/token context rather than workspace resources or authorization grants.
- Keep `TestWebhook` unaudited because it sends a synthetic outbound probe without changing persisted project configuration.
- Keep POST-based searches, CEL parse/deparse, schema diffs, release checks, instance database discovery, and rollback previews unaudited.
- Do not add blanket auditing to worksheet reads in the legacy service. The saved-query migration owns metadata-only search and conditional auditing for another user's private content.
- Run the repository's proto workflow after every proto batch: `buf format -w proto`, `buf lint proto`, then `cd proto && buf generate`.
- Run `gofmt -w` on modified Go files and the focused Go tests after each task. Run the full Go lint/build gates before completion.

---

## Coverage Decision

Audit these 27 RPCs in this plan:

| Family | RPCs |
| --- | --- |
| Core lifecycle | `SetupSample`, `CreateProject`, `UpdateProject`, `RunPlanChecks`, `CancelPlanCheckRun`, `DeleteRelease`, `UndeleteRelease`, `BatchCreateRevisions`, `DeleteRevision` |
| Content-bearing lifecycle | `CreateRelease`, `UpdateRelease`, `CreateWorksheet`, `DeleteWorksheet` |
| Authentication | `RequestPasswordReset`, `ResetPassword` |
| Project webhooks | `AddWebhook`, `UpdateWebhook`, `RemoveWebhook` |
| Subscription | `UploadLicense`, `CreatePurchase`, `UpdatePurchase`, `CancelPurchase`, `ExportVCSProviderUsers` |
| Sensitive external actions | `AIService.Chat`, `AuditLogService.ExportAuditLogs`, `IdentityProviderService.TestIdentityProvider`, `SettingService.TestEmailSetting` |

`RequestPasswordReset` has one deliberate limitation: when the unauthenticated request has no validated workspace, there is nowhere to persist a workspace-owned audit row, so the existing silent-success behavior remains unaudited. Requests tied to a validated workspace must be recorded.

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
- Test: `backend/tests/login_audit_test.go`
- Test: `backend/tests/project_instance_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: existing `getRequestResource(any) string` and proto `bytebase.v1.audit` extension.
- Produces: audited descriptors and canonical resource names for nine low-risk lifecycle RPCs.

- [ ] **Step 1: Extend the resource extraction test first**

Add cases to `TestProjectLifecycleAuditResource` and rename it to `TestLifecycleAuditResource`:

```go
{name: "create project", request: &v1pb.CreateProjectRequest{ProjectId: "project-a", Project: &v1pb.Project{}}, want: "projects/project-a"},
{name: "update project", request: &v1pb.UpdateProjectRequest{Project: &v1pb.Project{Name: "projects/project-a"}}, want: "projects/project-a"},
{name: "run plan checks", request: &v1pb.RunPlanChecksRequest{Name: "projects/project-a/plans/101"}, want: "projects/project-a/plans/101"},
{name: "cancel plan checks", request: &v1pb.CancelPlanCheckRunRequest{Name: "projects/project-a/plans/101/planCheckRun"}, want: "projects/project-a/plans/101/planCheckRun"},
{name: "delete release", request: &v1pb.DeleteReleaseRequest{Name: "projects/project-a/releases/release-a"}, want: "projects/project-a/releases/release-a"},
{name: "undelete release", request: &v1pb.UndeleteReleaseRequest{Name: "projects/project-a/releases/release-a"}, want: "projects/project-a/releases/release-a"},
{name: "batch create revisions", request: &v1pb.BatchCreateRevisionsRequest{Parent: "instances/instance-a/databases/database-a"}, want: "instances/instance-a/databases/database-a"},
{name: "delete revision", request: &v1pb.DeleteRevisionRequest{Name: "instances/instance-a/databases/database-a/revisions/101"}, want: "instances/instance-a/databases/database-a/revisions/101"},
```

Change the assertion to `require.Equal(t, test.want, getRequestResource(test.request))`.

- [ ] **Step 2: Run the unit test and verify the new cases fail**

Run:

```bash
go test ./backend/api/v1 -run '^TestLifecycleAuditResource$' -count=1
```

Expected: FAIL for plan, release, and revision cases because `getRequestResource` currently returns an empty string.

- [ ] **Step 3: Add canonical resource extraction**

Add direct cases to `getRequestResource`; do not introduce another dispatcher:

```go
case *v1pb.RunPlanChecksRequest:
	return r.GetName()
case *v1pb.CancelPlanCheckRunRequest:
	return r.GetName()
case *v1pb.DeleteReleaseRequest:
	return r.GetName()
case *v1pb.UndeleteReleaseRequest:
	return r.GetName()
case *v1pb.BatchCreateRevisionsRequest:
	return r.GetParent()
case *v1pb.DeleteRevisionRequest:
	return r.GetName()
```

- [ ] **Step 4: Annotate the low-risk RPCs**

Add this option inside each RPC block for `SetupSample`, `CreateProject`, `UpdateProject`, `RunPlanChecks`, `CancelPlanCheckRun`, `DeleteRelease`, `UndeleteRelease`, `BatchCreateRevisions`, and `DeleteRevision`:

```proto
option (bytebase.v1.audit) = true;
```

- [ ] **Step 5: Generate and run focused unit tests**

Run:

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_test.go
go test ./backend/api/v1 -run '^(TestLifecycleAuditResource|TestAuditRedactionDoesNotMutateInput)$' -count=1
```

Expected: all commands PASS.

- [ ] **Step 6: Add persistence regression coverage for the original symptom**

In `backend/tests/login_audit_test.go`, create a project, query workspace audit logs with:

```go
Filter: `method == "/bytebase.v1.ProjectService/CreateProject"`,
```

Assert exactly one matching row has the workspace parent and the canonical project resource. Update that project, query its project-scoped audit stream for `ProjectService/UpdateProject`, and assert the resource is the same project name.

In `backend/tests/project_instance_test.go`, immediately after the project-scoped instance is created, query its project audit stream for:

```go
Filter: `method == "/bytebase.v1.InstanceService/CreateInstance"`,
```

Assert a row exists with `Resource == projectInstance.Name`. This pins the distinction between “descriptor is annotated” and “row was actually persisted.”

- [ ] **Step 7: Run the lifecycle integration tests**

Run:

```bash
go test -v -count=1 ./backend/tests -run '^(TestAuditLogFormat|TestProjectInstanceCoreBehavior)$' -timeout 10m
```

Expected: PASS, including persisted `CreateProject`, `UpdateProject`, and `CreateInstance` rows.

- [ ] **Step 8: Commit the low-risk batch**

```bash
git add proto/v1/v1/actuator_service.proto proto/v1/v1/project_service.proto proto/v1/v1/plan_service.proto proto/v1/v1/release_service.proto proto/v1/v1/revision_service.proto backend/api/v1/audit.go backend/api/v1/audit_test.go backend/tests/login_audit_test.go backend/tests/project_instance_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit core lifecycle APIs"
```

---

### Task 2: Release and Worksheet Content Redaction

**Files:**
- Modify: `proto/v1/v1/release_service.proto`
- Modify: `proto/v1/v1/worksheet_service.proto`
- Modify: `backend/api/v1/audit.go`
- Modify: `backend/api/v1/audit_redaction_test.go`
- Modify: `backend/api/v1/audit_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: `getRequestString`, `getResponseString`, `maskedString`, and `proto.CloneOf`.
- Produces: `redactRelease`, `redactWorksheet`, and redacted create/update request handling.

- [ ] **Step 1: Add failing request and response redaction cases**

Add `secretSentinel` as a release statement and worksheet content in `TestAuditRequestRedactsCredentials`, and add a worksheet response case in `TestAuditResponseRedactsCredentials`:

```go
{"release SQL", &v1pb.CreateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}}},
{"worksheet SQL", &v1pb.CreateWorksheetRequest{Worksheet: &v1pb.Worksheet{Content: []byte(secretSentinel)}}},
```

```go
{name: "worksheet response SQL", response: &v1pb.Worksheet{Content: []byte(secretSentinel)}},
```

- [ ] **Step 2: Run the redaction tests and verify they fail**

```bash
go test ./backend/api/v1 -run '^TestAudit(Request|Response)RedactsCredentials$' -count=1
```

Expected: FAIL because the sentinel appears in serialized audit payloads.

- [ ] **Step 3: Implement clone-first content redactors**

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

func redactWorksheet(r *v1pb.Worksheet) *v1pb.Worksheet {
	if r == nil {
		return nil
	}
	cloned := proto.CloneOf(r)
	cloned.Content = nil
	return cloned
}
```

Add request switch cases for `CreateReleaseRequest`, `UpdateReleaseRequest`, and `CreateWorksheetRequest`, preserving all non-content fields and update masks. Add response switch cases for `Release` and `Worksheet` so future handler changes cannot expose statement/content fields through the audit response.

- [ ] **Step 4: Add resource extraction and annotations**

Extend `getRequestResource`:

```go
case *v1pb.CreateReleaseRequest:
	return r.GetParent()
case *v1pb.UpdateReleaseRequest:
	return r.GetRelease().GetName()
case *v1pb.CreateWorksheetRequest:
	return r.GetParent()
case *v1pb.DeleteWorksheetRequest:
	return r.GetName()
```

Add `audit = true` to `CreateRelease`, `UpdateRelease`, `CreateWorksheet`, and `DeleteWorksheet`.

- [ ] **Step 5: Verify redaction does not mutate input**

Extend `TestAuditRedactionDoesNotMutateInput` with a release and worksheet, call the corresponding audit serializer, and assert the original statement/content still equals `secretSentinel`.

- [ ] **Step 6: Generate and test**

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_test.go
go test ./backend/api/v1 -run '^(TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput|TestLifecycleAuditResource)$' -count=1
```

Expected: PASS and no sentinel in any serialized payload.

- [ ] **Step 7: Commit the content-bearing batch**

```bash
git add proto/v1/v1/release_service.proto proto/v1/v1/worksheet_service.proto backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit release and worksheet lifecycle"
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

- [ ] **Step 1: Add failing credential-redaction tests**

Add a request case containing the sentinel in password-reset `code` and `new_password`. Assert the email remains visible while both secrets are absent.

- [ ] **Step 2: Implement request redactors**

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

- [ ] **Step 3: Attribute successful and target-valid auth events**

In `ResetPassword`, call:

```go
common.SetAuditWorkspaceID(ctx, codeRow.Workspace)
```

immediately after `verifyEmailCode` succeeds, before membership, password-policy, or update failures can return.

For `RequestPasswordReset`, resolve the optional workspace without changing the externally visible result. Before sending the email, use the normalized email plus parsed workspace ID in `FindWorkspace`; set the audit workspace only when the account is an actual member:

```go
if workspaceID, err := parseOptionalWorkspace(req.Msg.Workspace); err == nil && workspaceID != "" {
	if workspace, findErr := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		WorkspaceID:    &workspaceID,
		Email:          email,
		IncludeAllUser: !s.profile.SaaS,
	}); findErr == nil && workspace != nil {
		common.SetAuditWorkspaceID(ctx, workspaceID)
	}
}
```

Do not return lookup errors from this preflight and do not change the current always-success behavior; an absent, malformed, unknown, or non-member workspace produces no audit row.

- [ ] **Step 4: Add audit annotations**

Add `audit = true` to `RequestPasswordReset` and `ResetPassword`.

- [ ] **Step 5: Add auth persistence tests**

Extend `backend/tests/login_audit_test.go` to assert:

- a successful password reset row contains the email but neither the submitted code nor new password;
- a workspace-bound password reset request produces a row without exposing whether the account exists through the API response.

- [ ] **Step 6: Generate and run auth tests**

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
gofmt -w backend/api/v1/auth_service.go backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/tests/login_audit_test.go
go test ./backend/api/v1 -run '^(TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput)$' -count=1
go test -v -count=1 ./backend/tests -run '^TestAuditLogFormat$' -timeout 10m
```

Expected: PASS with no auth credential present in stored request/response JSON.

- [ ] **Step 7: Commit the authentication batch**

```bash
git add proto/v1/v1/auth_service.proto backend/api/v1/auth_service.go backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/tests/login_audit_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit authentication security events"
```

---

### Task 4: Subscription and Export Audit Coverage

**Files:**
- Modify: `proto/v1/v1/subscription_service.proto`
- Modify: `backend/api/v1/audit.go`
- Modify: `backend/api/v1/audit_redaction_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: workspace parent fallback from the authenticated context.
- Produces: `redactUploadLicenseRequest`, empty/redacted purchase responses, and metadata-only HTTP export responses.

- [ ] **Step 1: Add failing redaction cases**

Cover `UploadLicenseRequest.License`, `PurchaseResponse.PaymentUrl`, `PurchaseResponse.SessionId`, and `google.api.HttpBody.Data` with `secretSentinel` in the request/response redaction tables.

- [ ] **Step 2: Implement subscription redactors**

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

For `google.api.HttpBody`, retain only `ContentType` and drop `Data` and `Extensions`. Register these types in the request/response serializer switches.

- [ ] **Step 3: Add subscription annotations**

Add `audit = true` to `UploadLicense`, `CreatePurchase`, `UpdatePurchase`, `CancelPurchase`, and `ExportVCSProviderUsers`.

Leave `Resource` empty for these workspace-singleton operations; the authenticated workspace remains the audit parent. Do not invent a non-resource-format subscription name.

- [ ] **Step 4: Generate and test**

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go
go test ./backend/api/v1 -run '^(TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput)$' -count=1
```

Expected: PASS; license, checkout URL/session, and CSV data are absent.

- [ ] **Step 5: Commit the subscription batch**

```bash
git add proto/v1/v1/subscription_service.proto backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
git commit -m "feat: audit subscription management actions"
```

---

### Task 5: Webhook, Configuration Test, AI, and Audit Export Actions

**Files:**
- Modify: `proto/v1/v1/project_service.proto`
- Modify: `proto/v1/v1/idp_service.proto`
- Modify: `proto/v1/v1/setting_service.proto`
- Modify: `proto/v1/v1/ai_service.proto`
- Modify: `proto/v1/v1/audit_log_service.proto`
- Modify: `backend/api/v1/audit.go`
- Modify: `backend/api/v1/audit_redaction_test.go`
- Modify: `backend/api/v1/audit_test.go`
- Generated: `backend/generated-go/v1/`, `frontend/src/types/proto-es/v1/`, `proto/gen/grpc-doc/`, `backend/api/mcp/gen/`

**Interfaces:**
- Consumes: existing `redactIdentityProvider`, `redactSetting`, and `maskedString`.
- Produces: metadata-only audit payloads for outbound tests, AI calls, and audit-log exports.

- [ ] **Step 1: Add failing secret/content tests**

Add cases for webhook URL, OAuth/OIDC authorization code, LDAP password, SMTP password, AI message content/tool arguments/provider metadata, and exported audit-log bytes. Keep safe identifiers such as project name, webhook name/title, recipient email, IdP name, AI token usage, and export page token.

- [ ] **Step 2: Extract a reusable email-setting redactor**

Create `redactEmailSetting(*v1pb.EmailSetting) *v1pb.EmailSetting` by moving the existing SMTP-password masking logic out of `redactSetting`; make `redactSetting` call it. Use the same helper for `TestEmailSettingRequest` so the two paths cannot drift.

- [ ] **Step 3: Implement request/response redactors**

Implement these exact contracts:

```go
func redactWebhook(w *v1pb.Webhook) *v1pb.Webhook
// Clone; replace a non-empty URL with maskedString; keep name, type, title, and direct-message metadata.

func redactTestIdentityProviderRequest(r *v1pb.TestIdentityProviderRequest) *v1pb.TestIdentityProviderRequest
// Clone; call redactIdentityProvider; mask OAuth/OIDC code and LDAP password.

func redactAIChatRequest(r *v1pb.AIChatRequest) *v1pb.AIChatRequest
// Return an empty AIChatRequest so prompts, tool schemas, tool arguments, and provider metadata are never logged.

func redactAIChatResponse(r *v1pb.AIChatResponse) *v1pb.AIChatResponse
// Return a response containing only a cloned Usage message.

func redactExportAuditLogsResponse(r *v1pb.ExportAuditLogsResponse) *v1pb.ExportAuditLogsResponse
// Return only NextPageToken; drop Content.
```

Register Add/Update/Remove webhook requests, `TestIdentityProviderRequest`, `TestEmailSettingRequest`, `AIChatRequest`, `AIChatResponse`, and `ExportAuditLogsResponse` in the serializer switches.

- [ ] **Step 4: Add canonical resource extraction**

```go
case *v1pb.AddWebhookRequest:
	return r.GetProject()
case *v1pb.UpdateWebhookRequest:
	return r.GetWebhook().GetName()
case *v1pb.RemoveWebhookRequest:
	return r.GetWebhook().GetName()
case *v1pb.ExportAuditLogsRequest:
	return r.GetParent()
case *v1pb.TestEmailSettingRequest:
	return r.GetParent()
```

The IdP test and AI chat remain workspace-parented with an empty resource because neither request names a canonical Bytebase resource.

- [ ] **Step 5: Add annotations**

Add `audit = true` to `AddWebhook`, `UpdateWebhook`, `RemoveWebhook`, `TestIdentityProvider`, `TestEmailSetting`, `AIService.Chat`, and `ExportAuditLogs`.

- [ ] **Step 6: Generate and test**

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
gofmt -w backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_test.go
go test ./backend/api/v1 -run '^(TestAudit(Request|Response)RedactsCredentials|TestAuditRedactionDoesNotMutateInput|TestLifecycleAuditResource)$' -count=1
```

Expected: PASS; audit rows retain action metadata without outbound payloads or credentials.

- [ ] **Step 7: Commit the sensitive-action batch**

```bash
git add proto/v1/v1/project_service.proto proto/v1/v1/idp_service.proto proto/v1/v1/setting_service.proto proto/v1/v1/ai_service.proto proto/v1/v1/audit_log_service.proto backend/api/v1/audit.go backend/api/v1/audit_redaction_test.go backend/api/v1/audit_test.go backend/generated-go/v1 frontend/src/types/proto-es/v1 proto/gen/grpc-doc backend/api/mcp/gen
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
		"bytebase.v1.AIService.Chat",
		"bytebase.v1.AuditLogService.ExportAuditLogs",
		"bytebase.v1.AuthService.RequestPasswordReset",
		"bytebase.v1.AuthService.ResetPassword",
		"bytebase.v1.IdentityProviderService.TestIdentityProvider",
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
		"bytebase.v1.SettingService.TestEmailSetting",
		"bytebase.v1.SubscriptionService.CancelPurchase",
		"bytebase.v1.SubscriptionService.CreatePurchase",
		"bytebase.v1.SubscriptionService.ExportVCSProviderUsers",
		"bytebase.v1.SubscriptionService.UpdatePurchase",
		"bytebase.v1.SubscriptionService.UploadLicense",
		"bytebase.v1.WorksheetService.CreateWorksheet",
		"bytebase.v1.WorksheetService.DeleteWorksheet",
	} {
		t.Run(method, func(t *testing.T) {
			requireAuditedMethod(t, method)
		})
	}
}
```

- [ ] **Step 2: Add the intentional-exclusion test**

Add `requireUnauditedMethod` and enumerate the ten deliberate exclusions from Global Constraints. This makes adding audit to sync, autosave, session mechanics, or non-mutating probe paths a conscious policy change instead of accidental drift:

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
		"bytebase.v1.AuthService.Refresh",
		"bytebase.v1.AuthService.SwitchWorkspace",
		"bytebase.v1.DatabaseService.BatchSyncDatabases",
		"bytebase.v1.DatabaseService.SyncDatabase",
		"bytebase.v1.InstanceService.BatchSyncInstances",
		"bytebase.v1.InstanceService.SyncInstance",
		"bytebase.v1.ProjectService.TestWebhook",
		"bytebase.v1.WorksheetService.BatchUpdateWorksheetOrganizer",
		"bytebase.v1.WorksheetService.UpdateWorksheet",
		"bytebase.v1.WorksheetService.UpdateWorksheetOrganizer",
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
- the deliberate exclusions and their reasons, including session-only workspace switching and non-mutating webhook tests;
- the rule that serializers must redact credentials/content before annotations are enabled;
- the workspace-resolution limitation for unauthenticated password-reset requests;
- the legacy worksheet-read deferral to the saved-query privacy model.

- [ ] **Step 5: Run all required verification gates**

```bash
buf format -w proto
buf lint proto
cd proto && buf generate
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
