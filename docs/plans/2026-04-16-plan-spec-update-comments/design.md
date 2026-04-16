## References

- [GitHub Docs: Reviewing proposed changes in a pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/reviewing-proposed-changes-in-a-pull-request?tool=codespaces) - verified. GitHub places review decisions next to file diffs, commit context, and stale-review behavior.
- [GitLab Docs: Merge requests](https://docs.gitlab.com/user/project/merge_requests/) - verified. GitLab merge requests present code changes, inline reviews, comments, CI state, mergeability, and commit lists together.
- [GitLab Docs: Merge request commits](https://docs.gitlab.com/user/project/merge_requests/commits/) - verified. GitLab keeps a per-merge-request commit history and supports viewing diffs between commits.
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) - verified. OWASP describes audit logs as chronological, attributable records and recommends enough event data to reconstruct who did what, when, and to which object.
- [Bytebase PR #18589: refactor: remove deployment snapshot and simplify plan updates](https://github.com/bytebase/bytebase/pull/18589) - verified. The `backend/api/v1/plan_service.go` patch simplified `UpdatePlan` and removed the previous `PlanSpecUpdate` issue comment creation path.

## Industry Baseline

Code review systems keep content changes visible inside the review workflow. GitHub documents pull request review around changed files, diffs, commits, approvals, and stale-review behavior in one review surface. GitLab merge requests similarly combine code changes, inline reviews, comments, pipelines, mergeability, and commits, and provide commit history plus diff comparison when reviewers need change progression.

Audit guidance favors chronological, attributable, scoped event records instead of separate ad hoc channels. The OWASP Logging Cheat Sheet describes audit records as a trail that supports reconstruction and review, and recommends recording enough information to identify the event time, actor, action, and object without over-logging.

For this codebase, the baseline maps to issue activity rather than a new plan activity subsystem. `IssueCommentPayload.PlanSpecUpdate` already stores the changed spec and previous/new sheet SHA256 values, `issue_comment` already stores creator and timestamps, and the frontend already renders the event as a statement diff.

## Research Summary

The relevant pattern across GitHub and GitLab is to surface changes to review content in the same workflow where approvals happen. Reviewers can inspect what changed without switching to an unrelated audit channel.

OWASP's logging guidance supports a small, purpose-built activity event: the existing issue comment row supplies when and who; the `PlanSpecUpdate` payload supplies what changed and the affected object; the frontend diff view supplies reviewable context. That matches the current Bytebase model without expanding proto contracts or logging raw SQL into an additional structure.

The prior `UpdatePlan` implementation removed in PR #18589 already followed this direction by creating `PlanSpecUpdate` comments for sheet changes. The current code still has the downstream payload conversion and frontend rendering, so the main design question is how to reconstruct changed sheet pairs in the simplified `UpdatePlan` flow.

## Design Goals

- Emit one issue activity entry for each updated plan spec whose statement sheet changes while the plan is attached to an issue; this is verifiable by updating a plan spec and listing issue comments.
- Preserve existing API and frontend contracts; this is verifiable by no proto changes and no frontend source changes.
- Keep issue activity scoped to reviewable statement diffs; this is verifiable by `PlanSpecUpdate` comments containing both previous and new sheet references.
- Preserve existing plan update behavior for validation, approval reset, and plan check scheduling; this is verifiable by existing plan update tests plus a focused regression test around issue comments.

## Non-Goals

- Changing proto definitions for issue comments, plans, sheets, or approvals.
- Changing frontend issue activity rendering or the statement diff component.
- Redesigning issue comments, plan comments, or activity timeline storage.
- Changing plan update permissions or the `bb.plans.update` role mapping.
- Changing self-approval eligibility, including tracking or validating the last plan SQL editor.
- Changing rollout execution, task update, or plan check scheduling behavior.
- Recording SQL changes for plans with no linked issue.

## Proposed Design

Keep `PlanService.UpdatePlan` as the emission point for plan spec update activity. This follows the GitHub and GitLab pattern of keeping content-change history in the same review object where reviewers make approval decisions, and it avoids a separate plan activity system because `issue_comment` already backs issue activity.

Detect changed statement sheets inside the existing `specs` update branch in `backend/api/v1/plan_service.go`. Compare the old stored specs from `oldPlan.Config.Specs` with the new stored specs produced from the request, keyed by spec ID. For each spec present in both old and new versions, read the old and new sheet SHA256 from supported sheet-backed configs. Create an activity entry only when both SHA256 values are non-empty and different. This keeps the event aligned with the current `PlanSpecUpdate` payload and frontend renderer, which both expect before/after sheet references.

Use the existing `IssueCommentPayload_PlanSpecUpdate` event. Set `Spec` with `common.FormatSpec(issue.ProjectID, oldPlan.UID, specID)`, `FromSheetSha256` with the old sheet SHA256, and `ToSheetSha256` with the new sheet SHA256. `issue_comment.creator` and row timestamps provide the actor and time fields recommended by OWASP, while the payload identifies the action object and before/after sheet state.

Create comments only when an issue is found for the plan. That matches the current visibility problem, which is specifically about downstream issue approvers, and respects the non-goal of recording SQL changes for standalone plans. If multiple specs change in one request, create one comment per changed spec so each timeline entry can link to the affected spec and statement diff independently.

Create the comments after the plan update succeeds, using `store.CreateIssueComments` with the requesting user's email and the accumulated comment payloads. Treat comment insertion failure as non-fatal and log it, matching the pre-simplification behavior removed in PR #18589. This keeps the plan update behavior stable while restoring the review-visible audit entry.

Do not change `proto/store/store/issue_comment.proto`, `backend/api/v1/issue_service_converter.go`, or the frontend activity components. The existing converter resolves stored SHA256 values back to sheet resource names, and the existing frontend renders the linked spec and statement diff when both sheets are present.

Add a focused backend regression test for a plan attached to an issue: update a sheet-backed spec to reference a different sheet, assert the plan update succeeds, then assert an issue comment exists with a `PlanSpecUpdate` payload containing the expected spec resource and old/new sheet SHA256 values. A companion negative case should cover unchanged sheet SHA or no linked issue, where no `PlanSpecUpdate` comment is created.
