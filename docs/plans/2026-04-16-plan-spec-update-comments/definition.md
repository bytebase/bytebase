## Background & Context

The reported approval workflow visibility issue occurs when SQL changes during an active approval flow and downstream approvers do not see an in-issue activity entry showing the SQL change. A follow-up proposal narrowed this work to the audit visibility gap: `PlanSpecUpdate` issue comment payloads and frontend rendering remain present, but `UpdatePlan` stopped emitting those events after GitHub PR [#18589](https://github.com/bytebase/bytebase/pull/18589) simplified plan update handling. That PR was merged on 2025-12-23 and its `backend/api/v1/plan_service.go` patch removed prior `PlanSpecUpdate` issue comment creation from the `specs` update branch.

## Issue Statement

When `PlanService.UpdatePlan` processes a `specs` update for a plan attached to an issue, the plan and approval state are updated without an issue activity entry that records the changed statement sheet, even though the issue comment payload, API conversion, and frontend rendering paths still handle `PlanSpecUpdate` events; downstream approvers see the latest SQL and approval activity without in-issue history of the SQL change.

## Current State

- `backend/api/v1/plan_service.go:225` defines `PlanService.UpdatePlan`.
- `backend/api/v1/plan_service.go:307` handles the `specs` update mask path.
- `backend/api/v1/plan_service.go:321` converts and stores the new specs in a cloned plan config.
- `backend/api/v1/plan_service.go:330` looks up the linked issue for the plan.
- `backend/api/v1/plan_service.go:335` through `backend/api/v1/plan_service.go:360` reset approval finding state and, for export issues, reapply the approval template.
- `backend/api/v1/plan_service.go:366` persists the plan update after the spec update branch completes.
- `backend/api/v1/plan_service.go:371` through `backend/api/v1/plan_service.go:383` create and schedule plan check runs after spec changes.
- `backend/store/issue_comment.go:129` defines `CreateIssueComments`, which inserts issue comments with a caller-provided creator and JSON payload.
- `proto/store/store/issue_comment.proto:13` through `proto/store/store/issue_comment.proto:17` define `IssueCommentPayload.event` and include `plan_spec_update`.
- `proto/store/store/issue_comment.proto:35` through `proto/store/store/issue_comment.proto:42` define `PlanSpecUpdate` with spec, previous sheet SHA256, and new sheet SHA256 fields.
- `backend/api/v1/issue_service_converter.go:308` through `backend/api/v1/issue_service_converter.go:310` convert stored `PlanSpecUpdate` payloads to v1 issue comment events.
- `backend/api/v1/issue_service_converter.go:375` through `backend/api/v1/issue_service_converter.go:389` resolve stored sheet SHA256 values to v1 sheet resource names.
- `frontend/src/components/Plan/components/IssueReviewView/ActivitySection/IssueCommentView/ActionSentence.vue:83` through `frontend/src/components/Plan/components/IssueReviewView/ActivitySection/IssueCommentView/ActionSentence.vue:111` render `PLAN_SPEC_UPDATE` comments with a spec link and statement diff when both old and new sheets are present.
- A repository search found current `PlanSpecUpdate` handling in payload, conversion, and frontend rendering paths, but no current `PlanSpecUpdate` issue comment creation path in `PlanService.UpdatePlan`.

## Non-Goals

- Changing proto definitions for issue comments, plans, sheets, or approvals.
- Changing frontend issue activity rendering or the statement diff component.
- Redesigning issue comments, plan comments, or activity timeline storage.
- Changing plan update permissions or the `bb.plans.update` role mapping.
- Changing self-approval eligibility, including tracking or validating the last plan SQL editor.
- Changing rollout execution, task update, or plan check scheduling behavior.
- Recording SQL changes for plans with no linked issue.

## Open Questions

- Should this work cover self-approval eligibility for users who edited plan SQL? (default: no, keep it separate from this audit visibility proposal)
- Should activity entries be limited to statement sheet changes with both previous and new sheet SHA256 values? (default: yes, match the existing `PlanSpecUpdate` payload and frontend diff rendering)
- Should additions or removals of specs be represented separately when no statement sheet diff exists? (default: no, keep the event limited to changed specs whose statement sheets can be compared)
- Should a failed issue comment insert fail the `UpdatePlan` request? (default: no, preserve the existing pattern of treating activity creation as non-fatal where possible)

## Scope

**S** - The current state localizes the visibility gap to the `specs` branch of `backend/api/v1/plan_service.go`, while the existing issue comment payload, converter, storage API, and frontend renderer already support `PlanSpecUpdate` events.
