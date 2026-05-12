# Plan Mutation Audit — Snapshot Design (supersedes 2026-05-09)

Date: 2026-05-11
Author: Steven
Status: Draft (supersedes `2026-05-09-plan-spec-mutation-audit-design.md`)

## TL;DR

Audit `PlanService.UpdatePlan` spec mutations by writing a **before/after snapshot of `plan.config.specs`** to `issue_comment`, instead of typed per-attribute events. One audit row per `UpdatePlan` call (issue-gated). The frontend diffs the snapshot pair and renders per-spec changes.

This supersedes the earlier typed-event design (`PlanSpecAdd` / `PlanSpecRemove` / `PlanSpecUpdate` with `from_*` / `to_*` attribute pairs). The trigger was [PR #20276 review feedback][1]: every new auditable attribute under the typed design costs a coordinated proto + helper + converter + renderer + i18n change, and reorder / `CreatePlan` / `DeletePlan` were explicitly deferred because they didn't fit the typed shape.

[1]: https://github.com/bytebase/bytebase/pull/20276#discussion_r3212945032

## Background

The visibility gap is the same as the earlier design — `BYT-9175`: approvers can't see what an editor changed on a plan during an active approval flow. PR #20032 (still unmerged) covers sheet-only changes. The earlier `2026-05-09` design extended that with typed `PlanSpecUpdate` (sheet + targets + prior_backup) and `PlanSpecAdd` / `PlanSpecRemove`. Reviewer pushed back on the typed approach in PR #20276.

The reviewer's argument, summarized: snapshotting `repeated PlanConfig.Spec` directly is forward-compatible with any future field, covers the deferred non-goals (reorder, `CreatePlan`, `DeletePlan`) for free, and collapses the backend emission to two lines. Storage cost is acceptable because sheets are content-addressed (sha256 only) and `Spec` configs are typically <1KB each. The cost is moving the diff logic from Go into TypeScript.

This design accepts that proposal.

## Goals

1. Approvers can see what changed on a plan they are reviewing — sheet, targets, prior-backup flag, spec adds, spec removes — for plans linked to an issue.
2. Schema is **forward-compatible**: a new field added to `PlanConfig.Spec` is audited automatically with zero backend churn. Only the renderer decides whether to surface it specially or fall through to a JSON-diff fallback.
3. Single uniform event shape — no per-attribute proto messages.
4. No schema migration; no DDL.

## Non-goals

- Plans without a linked issue (G3 in the umbrella doc; deferred to a future `plan_audit` table).
- Self-approval eligibility / editor tracking (the *control* gap from `BYT-9175`).
- Reorder-only audit — order is treated as cosmetic; reorder produces no audit row.
- `CreatePlan` / `DeletePlan` emission paths — the snapshot shape supports them trivially (empty `from_specs` / `to_specs`), but they introduce their own questions (who "created" a plan for audit purposes; semantics of deletion when an issue is still attached). Out of scope here.
- Auditing non-spec plan attributes (title, description, state). Tracked separately if needed.
- Real-time notifications / webhook delivery for plan mutations.

## Decision: Snapshot instead of typed events

| Dimension | Typed events (superseded) | Snapshot (this design) |
|---|---|---|
| New auditable attribute cost | proto + helper + converter + renderer + i18n | renderer only |
| Reorder coverage | non-goal | natural (filtered as cosmetic) |
| `CreatePlan` / `DeletePlan` coverage | non-goal | natural extension (deferred) |
| Storage per row | small typed payload | full Spec snapshot ×2 (sheets are sha256, ~1KB/spec) |
| Diff lives in | Go (backend) | TypeScript (frontend) |
| API consumer cost | event-typed: ready to consume | snapshot: must re-diff if they want per-attribute |
| Field-by-field opt-in to audit | yes | no (audits everything on `Spec`) |

The decision lands on the snapshot side because the forward-compat win is durable and the deferred non-goals fall out for free. The renderer carries more logic, but the diff helper is a pure function with a clean test surface.

## Detailed Design

### 1. Storage proto (`proto/store/store/issue_comment.proto`)

Replace the three typed variants with one snapshot-shaped event:

```proto
import "store/plan.proto";

oneof event {
  Approval approval = 2;
  IssueUpdate issue_update = 3;
  PlanUpdate plan_update = 7;
}

// PlanUpdate carries before/after snapshots of plan.config.specs,
// emitted once per PlanService.UpdatePlan call whose specs branch
// produces a non-cosmetic diff (id-keyed set inequality).
message PlanUpdate {
  repeated PlanConfig.Spec from_specs = 1;
  repeated PlanConfig.Spec to_specs = 2;
}
```

Field number `7` is the same number previously used by `PlanSpecUpdate`. Since neither `PlanSpecUpdate` nor its companions ever landed on `main` (PR #20032 and PR #20276 are both unmerged), there is no wire compatibility constraint and no `reserved` declaration is needed.

### 2. v1 proto (`proto/v1/v1/issue_service.proto`)

Mirror on the public API:

```proto
oneof event {
  Approval approval = 7;
  IssueUpdate issue_update = 8;
  PlanUpdate plan_update = 12;
}

message PlanUpdate {
  repeated Plan.Spec from_specs = 1;
  repeated Plan.Spec to_specs = 2;
}
```

`Plan.Spec` is the v1 type — same shape as storage but with sheet resource names instead of raw sha256s. The converter handles the translation.

### 3. Backend emission (`backend/api/v1/plan_service.go`)

Delete the four helpers introduced in the earlier design (`getPlanSpecSheetSha256`, `getPlanSpecTargets`, `getPlanSpecEnablePriorBackup`, `buildPlanSpecAuditIssueComments`). Replace with a single 12-line helper that detects cosmetic-only diffs:

```go
// planSpecsEqualSet reports whether two spec slices have the same set of
// specs keyed by id, with each pair byte-equal under proto.Equal. Order
// is ignored — reorder-only diffs are not audited.
func planSpecsEqualSet(a, b []*storepb.PlanConfig_Spec) bool {
    if len(a) != len(b) {
        return false
    }
    byID := make(map[string]*storepb.PlanConfig_Spec, len(a))
    for _, s := range a {
        byID[s.GetId()] = s
    }
    for _, s := range b {
        other, ok := byID[s.GetId()]
        if !ok || !proto.Equal(s, other) {
            return false
        }
    }
    return true
}
```

The call site under `case "specs":` becomes:

```go
if issue != nil {
    if !planSpecsEqualSet(oldPlan.Config.GetSpecs(), allSpecs) {
        issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
            ProjectID: issue.ProjectID,
            IssueUID:  issue.UID,
            Payload: &storepb.IssueCommentPayload{
                Event: &storepb.IssueCommentPayload_PlanUpdate_{
                    PlanUpdate: &storepb.IssueCommentPayload_PlanUpdate{
                        FromSpecs: oldPlan.Config.GetSpecs(),
                        ToSpecs:   allSpecs,
                    },
                },
            },
        })
    }
    // ... existing approval-finding reset logic unchanged ...
}
```

Self-approval reset and approval-template re-run logic are unchanged.

### 4. Backend converter (`backend/api/v1/issue_service_converter.go`)

Delete two trivial converters (`...PlanSpecAdd`, `...PlanSpecRemove`). Replace the third:

```go
func convertToIssueCommentEventPlanUpdate(projectID string, u *storepb.IssueCommentPayload_PlanUpdate_) *v1pb.IssueComment_PlanUpdate_ {
    return &v1pb.IssueComment_PlanUpdate_{
        PlanUpdate: &v1pb.IssueComment_PlanUpdate{
            FromSpecs: convertToPlanSpecs(projectID, u.PlanUpdate.GetFromSpecs()),
            ToSpecs:   convertToPlanSpecs(projectID, u.PlanUpdate.GetToSpecs()),
        },
    }
}
```

`convertToPlanSpecs` is the existing storage → v1 converter used by the live-plan view; we reuse it verbatim.

### 5. Frontend renderer (`IssueDetailCommentList.tsx`)

The diff logic moves into a pure helper alongside the renderer:

```ts
// frontend/src/react/pages/project/issue-detail/utils/diffPlanSpecs.ts

type SpecDiffEntry =
  | { kind: "added"; spec: Plan_Spec }
  | { kind: "removed"; spec: Plan_Spec }
  | {
      kind: "updated";
      specId: string;
      from: Plan_Spec;
      to: Plan_Spec;
      sheetChanged: boolean;
      targetsChanged: boolean;
      priorBackupChanged: boolean;
      otherChanged: boolean;
    };

function diffPlanSpecs(from: Plan_Spec[], to: Plan_Spec[]): SpecDiffEntry[];
```

`otherChanged` flags the case where the audit payload contains a `Spec` field difference the renderer doesn't recognize (a future attribute). The renderer falls back to a generic "Change <id8> updated" row with a `<details>` JSON-diff toggle.

The renderer dispatch becomes:

```tsx
if (commentType === IssueCommentType.PLAN_UPDATE &&
    issueComment.event.case === "planUpdate") {
  const { fromSpecs, toSpecs } = issueComment.event.value;
  const entries = diffPlanSpecs(fromSpecs, toSpecs);
  if (entries.length === 0) return null;
  return (
    <div className="flex flex-col gap-1">
      {entries.map((entry) => <SpecDiffRow key={diffEntryKey(entry)} entry={entry} />)}
    </div>
  );
}
```

`SpecDiffRow` switches on `entry.kind` and renders the same three shapes the typed-event design rendered, reusing the existing `SpecChangeRow`, `joinFragments`, `IssueDetailStatementUpdateButton` components.

#### Reused from PR #20276

- `SpecChangeRow` (chip + trailing slot for `[View Details]`)
- `joinFragments` (`common.and`-joined attribute fragments inside a row)
- `IssueDetailStatementUpdateButton` (including one-sided-sheet fallback)
- `?spec=<id>` deep-link in `ProjectIssueDetailPage` (chip click selects spec inline)
- All six i18n keys (`added`, `removed`, `modified-sql-of`, `changed-targets-of`, `enabled-prior-backup-on`, `disabled-prior-backup-on`) in both Vue and React locale trees
- Spec UUID-prefix chip (first 8 chars, stable across add/remove churn)
- Diff viewer height fix (viewport-sized Monaco)
- `UpdateIssue` no-op guard (backend `continue` + frontend `saveTitle` short-circuit)

#### `IssueCommentType` classifier

Drop `PLAN_SPEC_ADD` / `PLAN_SPEC_REMOVE`. Rename `PLAN_SPEC_UPDATE` → `PLAN_UPDATE`. Classifier dispatches on `event.case === "planUpdate"`.

#### `CommentActionIcon`

A single `PLAN_UPDATE` icon (`Pencil`). Per-entry add/remove icons aren't shown at the row level — they live inside `SpecDiffRow` if needed, but for now the entry text alone is enough.

### 6. Issue-gating (G3 still deferred)

Audit emission is still gated on `issue != nil`. Plans without a linked issue produce no audit row. This matches PR #20276 and the umbrella design's G3 deferral. The future `plan_audit` table (Approach 3) remains the path to closing the gap, and is independent of this PR's shape decision.

### 7. Migration from PR #20276

PR #20276 is open with the typed-event implementation. We force-push the snapshot rewrite onto the same branch (`feat/plan-spec-mutation-audit-a2`). Field number `7` (storage) and `12` (v1) are reused since no production data carries the old shape. The existing review thread stays attached to the diff.

### 8. Storage cost

Sheets are content-addressed via `sheet_sha256`, so snapshots do not carry SQL bytes. A typical `ChangeDatabaseConfig` spec serializes to <1KB protojson. A plan with 20 specs edited 100 times = ~4MB of `issue_comment` payload across the issue's timeline. JSONB tolerates this comfortably. Worst-case (hundreds of specs per plan + many edits) gets larger but stays within reasonable bounds; revisit if a customer surfaces a problem.

## Testing

### Backend unit

- `TestPlanSpecsEqualSet` — table-driven, 6 cases (identical, reorder, added, removed, sheet diff, targets diff, prior-backup diff).
- The 9-case `TestBuildPlanSpecAuditIssueComments` from the typed design is **deleted** along with the helper.

### Backend integration (`backend/tests/plan_update_test.go`)

All five original scenarios kept, assertions retargeted to the new `PlanUpdate` shape via `listPlanUpdateEvents` (renamed from `listPlanSpecAuditEvents`). Two new tests:

- `TestPlanUpdate_ReorderOnly_NoEmission` — verifies the cosmetic-reorder filter.
- `TestPlanUpdate_MultiSpec_OneRow` — verifies that mutating multiple specs in one `UpdatePlan` call produces exactly one `PlanUpdate` row.

The shared fixture (`setupPlanUpdateFixture`) and the no-issue test (`TestPlanSpecAudit_NoIssue_NoEmission`) carry over unchanged.

### Composite-PK collision test

`TestCollision_PlanSpecAuditEmission` retargeted to emit a `PlanUpdate` row in project A; assertion that project B's snapshot is untouched. `snapshotProject` / `assertProjectUnchanged` extensions covering `issue_comment` rows (added in PR #20276) carry over.

### Frontend unit (new — `diffPlanSpecs.test.ts`)

Table-driven, ~10 cases:

- empty / empty → []
- one added
- one removed
- sheet sha changed → `updated`, `sheetChanged: true`
- targets changed → `updated`, `targetsChanged: true`
- prior_backup flipped → `updated`, `priorBackupChanged: true`
- all three changed in one spec → one `updated` entry with all three flags set
- reorder-only → []
- mixed (add + remove + update) → three entries
- unknown attribute changed → `updated` with `otherChanged: true`

### Manual UI verification

Same four scenarios as PR #20276 plus two new ones:

- Multi-spec change in one save → one comment row with multiple visual entries.
- Reorder-only edit → no new comment row appears.

## Open questions

- The fallback rendering for `otherChanged: true` (unknown attribute) uses a generic "Change <id8> updated" with a JSON-diff toggle. The visual treatment can be polished later — the design just commits to "render *something* recognizable rather than silently dropping the row."
- Spec id is the only identity we have. If a user adds and immediately removes a spec with the same UUID across separate `UpdatePlan` calls, the timeline shows two rows; the rendering correctly distinguishes them.

## Future work

- **Approach 3** (`plan_audit` table parented to `plan`) closes G3. With the snapshot shape, A2→A3 migration is a near-1:1 row copy: same payload, different table.
- **`CreatePlan` / `DeletePlan` emission** via the same `PlanUpdate` shape (empty `from_specs` or `to_specs`).
- **Reorder audit** if a real user need surfaces. The diff helper already detects reorder; we just chose to filter it out.
- **Self-approval eligibility / editor tracking** — separate workstream addressing the *control* gap from `BYT-9175`.

## References

- PR #20276 review comment driving this redesign: https://github.com/bytebase/bytebase/pull/20276#discussion_r3212945032
- Superseded design: `docs/plans/2026-05-09-plan-spec-mutation-audit-design.md`
- Superseded implementation plan: `docs/plans/2026-05-09-plan-spec-mutation-audit.md`
- Umbrella design: `Design Doc: Plan Mutation Audit` (Apr 20 2026)
- `BYT-9175` — approval-workflow visibility issue
- `proto/store/store/issue_comment.proto`
- `proto/v1/v1/issue_service.proto`
- `proto/store/store/plan.proto` (`PlanConfig.Spec` shape)
- `backend/api/v1/plan_service.go` (`UpdatePlan` `case "specs":` branch)
- `backend/api/v1/issue_service_converter.go` (`convertToIssueComment` dispatch)
- `frontend/src/react/pages/project/issue-detail/components/IssueDetailCommentList.tsx`
