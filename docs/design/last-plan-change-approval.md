# Approval by the person who last changed a plan

## Decision

Bytebase treats the authenticated user on the most recent accepted `Plan.specs` mutation as the
Last Plan Editor. Project owners independently control whether that user and the Issue creator may
approve the related database change Issue. Both permissions are cumulative: a reviewer who is both
actors may approve only when both permissions allow it.

The public API retains `last_plan_editor` and `allow_last_plan_editor_approval` as precise resource
names. User-facing text describes the action — “the person who last changed the plan” — because
“Plan Editor” otherwise reads as a role.

## Product behavior

A plan creation or any accepted update whose update mask includes `specs` transfers Last Plan
Editor attribution to the authenticated caller. Identical and reorder-only specs updates count
because accepting the mutation makes that caller responsible for the submitted plan.

The following operations do not transfer attribution:

- changing the plan title or description;
- changing lifecycle state; and
- changing the content of a referenced Sheet without updating `Plan.specs`.

When `allow_last_plan_editor_approval` is disabled, the attributed user remains visible as a review
candidate but cannot Approve. The UI explains that the user last changed the plan and that another
reviewer must approve. Reject and request-review remain available.

`allow_self_approval` is a separate permission based on Issue creator identity. Keeping two controls
lets a project prohibit either authorship relationship independently. Combining them would prevent a
project from allowing Issue creators while requiring an independent reviewer after plan changes, or
the reverse.

## Attribution and compatibility

The Plan stores Last Plan Editor as a normalized email and updates that reference when the user's
email changes. The mutation, attribution transfer, and approval reset commit atomically. Approval is
authorized by the backend; UI eligibility is explanatory and cannot grant approval.

Historical plans without stored attribution use the Plan creator as the effective Last Plan Editor.
Existing projects are migrated to allow approval so upgrades preserve their prior behavior. New
projects use the restrictive default until a project owner enables the permission.

Changing either project permission affects future approval actions only. It does not remove or
rewrite recorded approval steps.
