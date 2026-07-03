# Approval CEL Issue Labels Design

## Context

The linked design note splits the approval-flow work into three areas:
approval input correctness, issue labels in approval-flow CEL, and creating
plans and issues together. Approval input correctness is mostly complete. This
spec covers only the second slice: exposing issue labels to approval-flow CEL.

Issue labels are already stored in `issue.payload.labels`, normalized through
`store.CanonicalizeIssueLabels`, and used by the issue store. Approval-flow CEL
currently evaluates source-specific rules with resource, statement, request,
and risk attributes. Fallback approval rules are deliberately narrower and only
allow `resource.project_id`.

## Goals

- Let `CHANGE_DATABASE` approval rules use `issue.labels`.
- Represent labels as a CEL `list<string>`, so expressions such as
  `"prod" in issue.labels` work naturally.
- Use canonical issue labels for evaluation.
- Reject approval payload writes computed from stale issue labels.
- Surface `issue.labels` in the custom approval CEL editor.
- Keep fallback approval rules unchanged.

## Non-Goals

- Do not expose labels to `CREATE_DATABASE`, `REQUEST_ROLE`, `REQUEST_ACCESS`,
  or fallback approval rules.
- Do not add a project-label dropdown in the CEL editor.
- Do not change issue-label update, approval reset, or plan/issue lifecycle
  behavior in this slice, except for the stale-label write guard required by
  label-based approval CEL.
- Do not introduce a separate label matcher outside CEL.

## Backend Design

Add a new CEL attribute constant named `issue.labels` in the shared CEL
attribute definitions. Add it to `common.ApprovalFactors` as a
`cel.ListType(cel.StringType)` variable. Do not add it to
`common.FallbackApprovalFactors`.

When building CEL variables for `DATABASE_CHANGE` issues, read labels from
`issue.Payload.GetLabels()`, canonicalize them with
`store.CanonicalizeIssueLabels`, and attach the resulting string slice to every
CEL variable map generated for that issue. This keeps the existing approval
matching model intact: a source-specific rule matches if any generated
per-target CEL variable map evaluates to true.

The attribute is attached only in the `DATABASE_CHANGE` path. Other issue types
continue to receive the variables they receive today.

Because labels become approval-affecting input, the approval runner must keep
the canonical label slice it used while computing the approval template. The
final database write must be conditional on both freshness dimensions:

- The plan still has the approval input version observed by the runner.
- The issue still has the same canonical labels observed by the runner.

If the conditional update affects zero rows, the runner discards the computed
approval payload. This matches the existing stale plan-version behavior and
prevents an older runner from writing approval state after issue labels changed.
The label comparison should use canonical value equality. Empty and missing
labels compare as the same empty list.

This design does not require a new plan-version mechanism. It uses the existing
`approvalInputVersion` guard for plan inputs and adds the missing issue-label
guard for the new approval-affecting input.

## Frontend Design

Add `CEL_ATTRIBUTE_ISSUE_LABELS = "issue.labels"` to the frontend CEL
attribute constants. Include it in the `CHANGE_DATABASE` factor list for custom
approval rules.

The existing expression editor can treat `issue.labels` as a normal factor.
This iteration does not need a project-label dropdown or project-label
existence validation. Users can author expressions with typed values, for
example:

```cel
"prod" in issue.labels
```

## Error Handling

Malformed expressions continue to fail through existing approval-rule CEL
compilation. Rules that reference `issue.labels` under fallback approval rules
remain invalid because fallback rules use the restricted fallback CEL
environment.

Missing labels evaluate as an empty canonical string list. Duplicate or
out-of-order labels are normalized before evaluation.

If labels change while approval finding is running, the stale runner may finish,
but its approval payload write is rejected by the label freshness guard. The
current reset/retry path remains responsible for scheduling a fresh approval
finding pass.

## Testing

Add backend tests that verify:

- `common.ApprovalFactors` accepts `issue.labels` list expressions.
- `common.FallbackApprovalFactors` rejects `issue.labels`.
- A `CHANGE_DATABASE` approval rule using `"prod" in issue.labels` matches when
  the issue has that label.
- Canonicalization makes duplicate or unordered labels behave consistently.
- A stale approval payload write is rejected when issue labels changed after the
  runner observed them.
- The existing plan approval input version guard still rejects stale plan-input
  writes.

Add frontend coverage where practical for the factor list, or otherwise rely on
the existing frontend check and type-check gates for the small constant/map
change.

## Rollout

This is additive for source-specific `CHANGE_DATABASE` approval rules. Existing
rules keep their behavior. Fallback rules remain intentionally unchanged, so no
migration is required.
