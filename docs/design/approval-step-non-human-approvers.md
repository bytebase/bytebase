# Approval steps held by service accounts and workload identities

Status: implemented in this PR · 2026-10-12

A custom approval step whose role is held only by service accounts shows "No reviewers" on the
plan page, and the code comment beside that string says the step can never be approved. Both are
wrong: the backend already lets a service account or workload identity approve any step whose role
it holds, and QVC's ServiceNow integration does exactly that. The review UI applies a display
filter the authorization model does not have. **A principal that holds a step's role is an
approver whatever its type: the review surfaces list it, name it, and attribute its decision the
same way they do for a person. Notifications stay human-only.**

## Problem

QVC (2026-10-08) ends every change's approval flow with a custom role "ServiceNow Approver" held
by two service accounts: a workspace account `bytebase-integration-service` bound at workspace and
project level, and a project account `testdt` bound at project level. The plan page shows the step
as "No reviewers", so the customer asked whether the flow is supported at all.

Reproduced on a throwaway 3.24-dev instance with the same shape (Developer → Workspace DBA →
ServiceNow Approver). What each surface showed before this change, from real screenshots in the PR
body:

| Surface | Before | Where |
| --- | --- | --- |
| Reviewer line under a pending or current step | "No reviewers" | `ReviewApprovalFlow.tsx` `NodeReviewers`; comment claims the step "can never be approved" (BYT-9711) |
| Folded-pending popover row | avatars omitted silently | `PendingPopoverRow`, `candidates.length > 0` guard |
| Header "Review in progress" popover | "No reviewers" | `PlanStatusAction.tsx` → `ReviewApprovalFlow compact` |
| "Approved by …" after the SA approves via API | `Approved by bytebase-integration-service` (email local part, no title, no account-type mark) | `FlowNode` subtitle, `ApprovedPopoverRow`, `ChipAvatars`: `getUserByIdentifier` misses, `unknownUser(principal)` fills in |
| Activity row for the approval comment | `bytebase-integration-service approved issue` | `IssueCommentActivity.tsx` `IssueCommentHeader`, same fallback |
| Collapsed REVIEW summary | `Last approved by bytebase-integration-service` | `phaseSummary.ts`: local part for every principal, humans included |
| Legacy issue page (grant issues) | approver hidden: "Approved by" followed by nothing; candidates filtered to humans | `IssueDetailApprovalFlow.tsx` `ApprovalUserText` returns null for non-`USER`; `useApprovalStep` filters `AccountType.USER` |

The authorization side is consistent and needs no change. `canReview` in
`backend/component/review/workflow.go` resolves roles through `GetUserFormattedRolesMap`, which
matches `serviceAccounts/{email}` and `workloadIdentities/{email}` members directly. The three QVC
binding shapes all pass. The approval is recorded as `users/<sa-email>`
(`common.FormatUserEmail(actor.Email)`), the approval comment's creator is the SA email, the
webhook actor is the SA's `UserMessage`, and the audit log names the actor
`serviceAccounts/<sa-email>`.

## Principle

> Who may approve is decided by IAM, not by principal type. The UI explains that decision; it never
> narrows it. A step with no human holder is still a step someone can act on, and the page must
> say who.

## Approver truth table

One row per kind of role holder. Cells that change are marked →.

| Principal | Can approve (backend) | Listed as reviewer | Mentioned in approval-requested notification | "Approved by" / activity attribution |
| --- | --- | --- | --- | --- |
| End user, direct or via group | yes | yes | yes | title |
| Service account (workspace), bound at workspace IAM | yes | no → **yes, badge "Service Account"** | no (by design) | local part → **title + badge** |
| Service account (workspace), bound at project IAM | yes | no → **yes** | no | local part → **title + badge** |
| Service account (project `x@proj.service.bytebase.com`), bound at project IAM | yes | no → **yes** | no | local part → **title + badge** |
| Workload identity, bound at either level | yes (`canReview` matches `workloadIdentities/`) | no → **yes, badge "Workload Identity"** | no | local part → **title + badge** |
| Deleted service account or workload identity still bound | no (cannot authenticate) | filtered like a deleted user (`State.ACTIVE` check) | no | as recorded |

Self-approval and last-plan-editor rules apply to every row by email: the backend compares
`issue.CreatorEmail` and `plan.LastPlanEditor` with `actor.Email`, and SA emails cannot collide
with human emails because `validateEndUserEmail` refuses the `service.bytebase.com` and
`workload.bytebase.com` suffixes for people.

## Decisions

**D1 · The reviewer list is every principal holding the step's role.** Project IAM plus workspace
IAM, with `serviceAccounts/` and `workloadIdentities/` names kept where `useApprovalCandidates`
used to drop them. "No reviewers" then means what it says: no principal holds the role. A listed
holder can still be barred by the self-approval or last-plan-editor rule, exactly as a person is
today; the list answers who holds the role, not who may act right now. Groups are expanded as
today, as people: the Groups page adds only `users/` members. A service account written into a
group through the API can approve, because the backend matches the stored member string, but it is
not listed; that case is left alone. Rejected: keep the human filter and reword to "No human
reviewers". It still hides who can act, which is the question the customer asked.

**D2 · Non-human principals resolve through their own services, never the user service.**
`BatchGetUsers` and `GetUser` reject SA and WI emails (`validateEndUserEmail`), and one such name
fails the whole batch. The candidate hook therefore splits IAM names by prefix: `users/` through
`batchGetOrFetchUsers`, `serviceAccounts/` through `fetchServiceAccount`, `workloadIdentities/`
through `fetchWorkloadIdentity`. A recorded approver or creator is `users/<email>` whatever its
type (D5), so there the split is by email suffix, which `usePrincipal.ts` on the issue-detail page
already does. It becomes the shared `hooks/usePrincipal.ts` and serves "Approved by", the approved
chip avatars, the activity header and avatar, the header status, and the rejection banner. The plan
page makes no failing user-service call today only because it never fetches approvers. The plan and
issue lists do: they prefetch creators through `batchGetOrFetchUsers`, and a page holding one
SA-created plan gets a 400 for the whole batch, people included. The user store's fetch entry
points therefore skip non-human emails instead of sending them, which fixes every caller at once.
Rejected: teach `BatchGetUsers` to return SAs. The split services are deliberate, and the Members
page already lives with it.

**D3 · The account-type mark is the existing `AccountTypeBadge`.** It derives the type from the
email suffix, which the server enforces, so a person cannot carry a "Service Account" badge and an
SA cannot shed it. That holds once the frontend applies the server's rule: `getAccountTypeByEmail`
accepts any email that ends in the suffix, while `IsServiceAccountEmail` requires the suffix to be
the whole domain or to follow a project label and a dot, so a person at `x@myservice.bytebase.com`
is a service account to the frontend alone. D2 routes fetches on this classifier, so it is
tightened to the server's rule. The badge gains a `compact` form with the geometry of the "Current"
badge (`px-1.5 py-0 text-xs`): it sits after the title and never shrinks, and the title truncates.
The legacy issue page's sidebar is too narrow for both on one line, so there the badge sits
beneath the name. Rejected: a distinct robot avatar glyph. The Members page sets
the precedent of initials avatar plus badge, and reviewers scan the badge text, not the avatar.

**D4 · A candidate of any type carries the attribution name `users/<email>`.** `issue.creator` and
`plan.lastPlanEditor` are `users/<email>` for every actor, SAs included. A service account or
workload identity is therefore listed through the same `User` projection the Members page uses
(`workloadIdentityToUser`, and a `serviceAccountToUser` twin), so `getApprovalEligibility`'s name
comparison holds unchanged. Naming the candidate `serviceAccounts/<email>` would show an SA creator
as eligible to approve its own plan when the backend refuses.

**D5 · The stored approver principal stays `users/<email>`.** Switching `applyReviewAction` to
`common.FormatPrincipalMember` was inventoried against every consumer: `issue_service_converter.go`
passes the string through; `utils.FindNextPendingRole`, `FindRejectedRole` and
`store/issue_approval_status.go` read only status and count; `approvalsEqual` and the staleness
check in `ReviewIssue` compare whole protos; `store/principal.go` rewrites `users/<old>` →
`users/<new>` when a user's email changes, which cannot happen to an SA (`UpdateServiceAccount`
accepts only `title` and `service_key`); webhooks use the actor's `UserMessage`, not the stored
string. Nothing needs the type in the string, because the email suffix already carries it. The
codebase has two conventions and this field belongs to the first: attribution fields
(`issue.creator`, `plan.creator`, `plan.last_plan_editor`, comment creators) are `users/<email>`
for every actor, while policy members and the audit-log actor carry the typed prefix. A new format
would need a backfill for every SA approval already stored (QVC's included) and dual-format
readers forever, for no information gain.

**D6 · Notifications are unchanged.** `getApproversForRole` keeps `END_USER` only and
`webhook/manager.go` filters mentions to end users; a machine has no IM identity to mention. The
UI's reviewer list and the notification's mention list are therefore different sets, and the
comment on `getApproversForRole` says so.

**D7 · The legacy issue page uses the same two hooks.** `IssueDetailApprovalFlow.tsx` still renders
grant issues (create-database, access, role) and carried its own copy of the candidate pipeline.
That copy is removed: `useApprovalCandidates` moves beside `approvalEligibility.ts` and both pages
call it. `ApprovalUserText` adopts the principal hook, so a role-grant request approved by an SA
no longer reads "Approved by" followed by a blank. It also stops hiding an approver who has since
been deactivated: a decided step names whoever decided it.

## States of the reviewer line

| Role holders | Line | Popover |
| --- | --- | --- |
| Humans only | avatar stack + "N reviewers" (unchanged) | rows: title, email, ineligibility note |
| Service accounts and workload identities only | avatar stack + "N reviewers" | rows carry the account-type badge |
| Mixed | humans first, current user first, then by title | same |
| Nobody | "No reviewers" | none |
| Policy not loaded | nothing (unchanged) | none |
| Holder record does not resolve | avatar from the email local part, badge still correct | row shows local part as title |

The last row is rare. `GetServiceAccount` and `GetWorkloadIdentity` need `bb.serviceAccounts.get`
/ `bb.workloadIdentities.get`, and every predefined workspace role carries both, Workspace Member
included (`backend/store/predefined_roles.go`). It takes a workspace that replaces Workspace
Member with a narrower custom role, or a binding that names an account that was never created.
The badge never depends on the fetch because the suffix is in the name.

## Risks

**R1 · Showing machines as approvers makes an existing capability visible.** Nothing new becomes
possible: an SA could already approve, and the audit log already records it. What changes is that a
reviewer reading the page learns that a machine can close the step. That is the governed path
(IAM binding, reviewed as code) and the shape Bytebase itself recommended when external approval
was removed. The design does not add a project setting to forbid non-human approvers; a workspace
that does not want them removes the binding.

**R2 · Self-approval by the integration's own account.** An SA that creates the plan through the
API and holds the approver role is refused by the backend unless the project allows self-approval,
exactly as a person is. D4 keeps the UI's eligibility note in step; getting it wrong would only
mislead, not grant.

**R3 · Spoofing the badge.** Impossible by construction: the suffixes are reserved server-side for
SA and WI records, and the badge reads the suffix.

**R4 · Fetch noise and permission.** Resolving the recorded `users/<sa-email>` through the user
store would add a failing request per plan view (D2). Fetching through the SA/WI services is
permitted for every predefined workspace role. Where a custom role lacks the read permission the
fetch fails silently and the row shows the local-part title. The SA and WI stores dedupe in-flight
requests and cache successes, not failures, so such a viewer pays one silent request per holder
per mount, never a loop; the user store treats a person it cannot resolve the same way. No
negative cache is added for that case.

**R5 · Stale bindings.** A deleted SA or WI stays in the IAM policy until an administrator removes
the binding, and it can no longer authenticate. Its record still resolves: deletion is soft, and
`GetServiceAccount` / `GetWorkloadIdentity` return it with `state: DELETED`. The hook applies the
`State.ACTIVE` filter that already drops a deleted user, so the line never lists it. The hook does
not branch on why a fetch failed, which the stores do not expose: a holder whose record does not
resolve keeps its placeholder and stays listed, as a person does today through `unknownUser`.

**R6 · Approval-flow width.** The "Approved by" subtitle is a fixed-height truncating line; the
badge is `shrink-0` and the title truncates, so a long SA title never pushes the badge out of
view. The reviewer popover widens from 240px to 288px so a title keeps room beside the badge;
the vertical layout below 560px stacks the same row.

**R7 · Existing stored data.** Every SA approval ever recorded is `users/<sa-email>` and renders
correctly under D5 without migration. Customers on older versions keep seeing the local-part
fallback until they upgrade; no data changes.

## Scope

| File | Change |
| --- | --- |
| `frontend/src/routes/project/useApprovalCandidates.ts` (moved from `plan-detail/components/review/`) | keep SA/WI names; resolve by prefix; list each as a `users/<email>` user; filter resolved records by state; people first |
| `frontend/src/hooks/usePrincipal.ts` (moved from `issue-detail/components/`) | shared; `principalLabel` names a principal by title, or by its email until a record arrives |
| `frontend/src/components/AccountTypeBadge.tsx` | `compact` form |
| `frontend/src/routes/project/plan-detail/components/review/ReviewApprovalFlow.tsx` | badge in `CandidateRow`, `FlowNode` subtitle, `ApprovedPopoverRow`; chip avatars through the hook; popover width; the BYT-9711 comment |
| `frontend/src/components/issue-activity/IssueCommentActivity.tsx` | `CommentCreator` and `ActivityUserIcon` take the principal and resolve it through the hook, badge after the name |
| `PlanStatusAction.tsx`, `ReviewRejectionBanner.tsx`, `ReviewActivityTimeline.tsx` | whoever rejected or acted, through the hook |
| `IssueDetailApprovalFlow.tsx`, `IssueDetailCommentList.tsx` | shared candidates hook in place of the local pipeline; approver identity with the badge beneath |
| `frontend/src/stores/app/user.ts` | `batchGetOrFetchUsers` and `fetchUser` skip non-human emails instead of sending them to the user service |
| `frontend/src/stores/app/serviceAccount.ts` | `serviceAccountToUser` |
| `frontend/src/types/v1/user.ts` | `getAccountTypeByEmail` applies the server's domain-boundary rule |
| `backend/component/review/evaluator.go` | comment on `getApproversForRole`: it returns the people to notify, not everyone who can approve |

Tests:

| File | Covers |
| --- | --- |
| `backend/component/review/workflow_test.go` | an SA approves via workspace IAM, via project IAM, and as a project SA; a WI via project IAM; each is recorded as `users/<email>`; an SA bound under `users/` and an SA that created the issue are refused |
| `useApprovalCandidates.test.tsx` | SA-only role; people before machines; nobody; deleted SA dropped, unresolved SA kept; SA creator held to self-approval; SA bound as `user:` not listed |
| `ReviewApprovalFlow.test.tsx` | SA-held step shows its reviewer; no holder says "No reviewers"; a step decided by an SA shows title and badge; a person without a record falls back to the email name, unmarked |
| `IssueCommentActivity.test.tsx` | an SA's approval names it with the badge; a person's does not |
| `stores/app/user.test.ts` | a batch that contains an SA email still resolves the people in it and sends nothing for the SA; a lone SA is never asked of the user service |
| `types/v1/user.test.ts` | the classifier against the server's rule, including `x@myservice.bytebase.com` |
| `tests/e2e/plan-detail/plan-detail-review.spec.ts` | in a browser: an SA-held step lists the account with its badge; after the SA approves through the API the step and the activity name it |

## Not in this PR

- Mentioning non-human approvers in IM notifications.
- Fetching SA/WI records on the Members page so it shows titles instead of the local part.
- Approver-type restrictions as a project setting.
- Listing a service account that was added to a group through the API.
- Showing an SA creator's title in the plan and issue lists; they keep the local-part fallback.
- Stating the `users/{email}` convention in the `Approver.principal` proto comments: a comment-only
  proto edit regenerates the API docs and can ride on a change that already touches those files.
