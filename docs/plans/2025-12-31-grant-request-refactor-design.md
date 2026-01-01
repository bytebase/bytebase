# Grant Request Auto-Completion Refactoring

## Problem

Grant request completion logic (grant privilege + close issue + create comment) is duplicated in two places:

1. **Runner** (`backend/runner/approval/runner.go:171-200`): When no approval template is found during issue creation
2. **ApproveIssue** (`backend/api/v1/issue_service.go:619-652`): When approval flow completes

Additionally, there's partial duplication at `issue_service.go:596-601` for granting privilege.

## Solution

1. Create a private helper method `completeGrantRequestIssue` on `IssueService`
2. Remove grant+close logic from the approval runner
3. Call the helper from both `CreateIssue` (when no approval needed) and `ApproveIssue` (when approval completes)

## Benefits

- Eliminates code duplication
- Separates concerns: runner finds approvals, service layer executes business logic
- Single source of truth for grant request completion
- Cleaner, more maintainable code

## Design Details

### 1. New Helper Method

Location: `backend/api/v1/issue_service.go`

```go
// completeGrantRequestIssue grants privilege and closes a grant request issue.
// Called when:
// 1. Issue created without approval template (auto-approved)
// 2. Issue approval flow completes
//
// Returns the updated issue with DONE status.
func (s *IssueService) completeGrantRequestIssue(
    ctx context.Context,
    issue *store.IssueMessage,
    grantRequest *storepb.GrantRequest,
) (*store.IssueMessage, error) {
    // 1. Grant the privilege
    if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, s.store, issue, grantRequest); err != nil {
        return nil, err
    }

    // 2. Update issue status to DONE
    newStatus := storepb.Issue_DONE
    updatedIssue, err := s.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
        Status: &newStatus,
    })
    if err != nil {
        return nil, errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
    }

    // 3. Create issue comment documenting the status change
    if _, err := s.store.CreateIssueComments(ctx, common.SystemBotEmail, &store.IssueCommentMessage{
        IssueUID: issue.UID,
        Payload: &storepb.IssueCommentPayload{
            Event: &storepb.IssueCommentPayload_IssueUpdate_{
                IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
                    FromStatus: &issue.Status,
                    ToStatus:   &updatedIssue.Status,
                },
            },
        },
    }); err != nil {
        // Non-fatal: log warning but continue
        slog.Warn("failed to create issue comment after changing the issue status", log.BBError(err))
    }

    return updatedIssue, nil
}
```

### 2. Modify `FindAndApplyApprovalTemplate`

Update signature to return whether approval template was found:

```go
func FindAndApplyApprovalTemplate(
    ctx context.Context,
    stores *store.Store,
    webhookManager *webhook.Manager,
    licenseService *enterprise.LicenseService,
    issue *store.IssueMessage,
) (approvalTemplateFound bool, error)
```

Changes in `backend/runner/approval/runner.go`:
- Remove grant+close logic (lines 170-200)
- Return `approvalTemplate != nil, nil` at the end
- Update all callers to handle the new return value

### 3. Update `CreateIssue`

Location: `backend/api/v1/issue_service.go:395-415`

```go
// Trigger approval finding based on issue type
switch issue.Type {
case storepb.Issue_GRANT_REQUEST, storepb.Issue_DATABASE_EXPORT:
    approvalTemplateFound, err := approval.FindAndApplyApprovalTemplate(
        ctx, s.store, s.webhookManager, s.licenseService, issue)
    if err != nil {
        slog.Error("failed to find approval template",
            slog.Int("issue_uid", issue.UID),
            slog.String("issue_title", issue.Title),
            log.BBError(err))
        // Continue anyway - non-fatal error
    }

    // For GRANT_REQUEST without approval template, auto-complete it
    if issue.Type == storepb.Issue_GRANT_REQUEST && !approvalTemplateFound {
        issue, err = s.completeGrantRequestIssue(ctx, issue, issue.Payload.GrantRequest)
        if err != nil {
            return nil, connect.NewError(connect.CodeInternal,
                errors.Wrapf(err, "failed to complete grant request"))
        }
    }
case storepb.Issue_DATABASE_CHANGE:
    // DATABASE_CHANGE needs to wait for plan check to complete
    s.bus.ApprovalCheckChan <- int64(issue.UID)
default:
    // For other issue types, no approval finding needed
}
```

### 4. Update `ApproveIssue`

Location: `backend/api/v1/issue_service.go:595-652`

Replace existing logic:

**Before (lines 595-601):**
```go
// Grant the privilege if the issue is approved.
if approved && issue.Type == storepb.Issue_GRANT_REQUEST {
    if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, s.store, issue, payload.GrantRequest); err != nil {
        return nil, err
    }
    // TODO(p0ny): Post project IAM policy update activity.
}
```

**After:**
```go
// Grant the privilege if the issue is approved (will be completed below)
// TODO(p0ny): Post project IAM policy update activity.
```

**Replace lines 619-652 with:**
```go
// If the issue is a grant request and approved, complete it
if issue.Type == storepb.Issue_GRANT_REQUEST && approved {
    var err error
    issue, err = s.completeGrantRequestIssue(ctx, issue, payload.GrantRequest)
    if err != nil {
        slog.Debug("failed to complete grant request issue", log.BBError(err))
    }
}
```

### 5. Update Runner

Location: `backend/runner/approval/runner.go`

**Remove lines 170-200** (entire grant+close block)

**Update return at line 224:**
```go
return approvalTemplate != nil, nil
```

**Update line 62** (signature):
```go
func FindAndApplyApprovalTemplate(...) (bool, error)
```

**Update line 95** (processIssue call):
```go
if _, err := findApprovalTemplateForIssue(...); err != nil {
```

## Error Handling

1. **In `completeGrantRequestIssue`**:
   - If `UpdateProjectPolicyFromGrantIssue` fails → return error (don't close issue)
   - If status update fails → return error (privilege granted, issue not closed)
   - If comment creation fails → log warning, continue (non-critical)

2. **In `CreateIssue`**:
   - If `completeGrantRequestIssue` fails → return error to user

3. **In `ApproveIssue`**:
   - If `completeGrantRequestIssue` fails → log debug, continue (matches current behavior)

## Edge Cases

- ✅ Grant request with approval template → normal approval flow
- ✅ Grant request without approval template → auto-completed in CreateIssue
- ✅ Grant request approval completes → completed in ApproveIssue
- ✅ DATABASE_EXPORT without approval → no change (doesn't need completion)

## Files Modified

1. `backend/api/v1/issue_service.go` - Add helper method, update CreateIssue and ApproveIssue
2. `backend/runner/approval/runner.go` - Remove grant+close logic, update return type

## Testing Considerations

1. **Unit tests**: Test `completeGrantRequestIssue` with various scenarios
2. **Integration tests**:
   - Create grant request without approval → should auto-complete
   - Create grant request with approval → should wait for approval
   - Approve grant request → should complete after approval
3. **Error cases**: Test failures in privilege grant and status update
