# Plan Mutation Audit (Snapshot) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the typed `PlanSpecAdd` / `PlanSpecRemove` / `PlanSpecUpdate` audit events currently on `feat/plan-spec-mutation-audit-a2` with a single `PlanUpdate` snapshot event carrying `from_specs` / `to_specs`. Move the per-spec diff logic into a pure TypeScript helper next to the renderer.

**Architecture:** Proto replaces three messages with one. Backend emission collapses to a 12-line set-equality guard + direct snapshot copy. Frontend gains a pure `diffPlanSpecs` helper consumed by a new `SpecDiffRow` component that reuses the existing `SpecChangeRow` / `joinFragments` / `IssueDetailStatementUpdateButton` building blocks.

**Tech Stack:** protobuf (buf), Go (backend), React + TypeScript (frontend), vitest (frontend unit tests).

**Pre-state:** This plan rewrites work already on `feat/plan-spec-mutation-audit-a2` (open PR #20276). Steps assume you're on that branch with all PR #20276 commits present. The final task force-pushes the rewrite.

---

## File Structure

### Modified

| File | Change |
|---|---|
| `proto/store/store/issue_comment.proto` | Replace `PlanSpecUpdate` + `PlanSpecAdd` + `PlanSpecRemove` with `PlanUpdate` |
| `proto/v1/v1/issue_service.proto` | Same shape on public API |
| `backend/api/v1/plan_service.go` | Delete 4 helpers + `buildPlanSpecAuditIssueComments`; add `planSpecsEqualSet`; rewire `UpdatePlan` |
| `backend/api/v1/plan_service_test.go` | Delete tests for deleted helpers; add `TestPlanSpecsEqualSet` |
| `backend/api/v1/issue_service_converter.go` | Delete 2 converters; replace third; update dispatch |
| `backend/tests/plan_update_test.go` | Retarget assertions; rename `listPlanSpecAuditEvents` → `listPlanUpdateEvents`; add 2 new tests |
| `backend/tests/plan_audit_collision_test.go` | Retarget action + assertion to `PlanUpdate` |
| `frontend/src/store/modules/v1/issueComment.ts` | Drop `PLAN_SPEC_ADD` / `PLAN_SPEC_REMOVE`; rename `PLAN_SPEC_UPDATE` → `PLAN_UPDATE` |
| `frontend/src/react/pages/project/issue-detail/components/IssueDetailCommentList.tsx` | Replace 3 dispatch branches with 1 `planUpdate` branch + new `SpecDiffRow` component |

### Created

| File | Purpose |
|---|---|
| `frontend/src/react/pages/project/issue-detail/utils/diffPlanSpecs.ts` | Pure diff helper |
| `frontend/src/react/pages/project/issue-detail/utils/__tests__/diffPlanSpecs.test.ts` | Vitest table-test |

### Untouched (already on branch, ship as-is)

- `SpecChangeRow`, `joinFragments`, `IssueDetailStatementUpdateButton` in `IssueDetailCommentList.tsx`
- `?spec=` deep-link wiring in `ProjectIssueDetailPage.tsx`
- View Details dialog height fix
- `UpdateIssue` no-op guard (backend + frontend)
- All 6 i18n keys in both Vue and React locale trees
- `snapshotProject` / `assertProjectUnchanged` extensions covering `issue_comment` rows in `collision_helper_test.go`

---

## Task 1: Replace proto with PlanUpdate

**Files:**
- Modify: `proto/store/store/issue_comment.proto`
- Modify: `proto/v1/v1/issue_service.proto`

- [ ] **Step 1: Rewrite storage proto**

Edit `proto/store/store/issue_comment.proto`. Replace the `IssueCommentPayload` oneof and the three current `PlanSpec*` messages:

```proto
syntax = "proto3";

package bytebase.store;

import "store/approval.proto";
import "store/issue.proto";
import "store/plan.proto";

option go_package = "generated-go/store";

message IssueCommentPayload {
  string comment = 1;

  oneof event {
    Approval approval = 2;
    IssueUpdate issue_update = 3;
    PlanUpdate plan_update = 7;
  }

  message Approval {
    IssuePayloadApproval.Approver.Status status = 1;
  }

  message IssueUpdate {
    optional string from_title = 1;
    optional string to_title = 2;
    optional string from_description = 3;
    optional string to_description = 4;
    optional Issue.Status from_status = 5;
    optional Issue.Status to_status = 6;
    repeated string from_labels = 7;
    repeated string to_labels = 8;
  }

  // PlanUpdate carries before/after snapshots of plan.config.specs,
  // emitted once per PlanService.UpdatePlan call whose specs branch
  // produces a non-cosmetic diff. The renderer computes per-spec
  // add/remove/update from the snapshot pair.
  message PlanUpdate {
    repeated PlanConfig.Spec from_specs = 1;
    repeated PlanConfig.Spec to_specs = 2;
  }
}
```

- [ ] **Step 2: Rewrite v1 proto**

Edit `proto/v1/v1/issue_service.proto`. Locate the `IssueComment` message (around line 595). Replace the `oneof event { ... }` block and the three `PlanSpec*` message definitions with:

```proto
// The event associated with this comment.
oneof event {
  // Approval event.
  Approval approval = 7;
  // Issue update event.
  IssueUpdate issue_update = 8;
  // Plan update event.
  PlanUpdate plan_update = 12;
}
```

And replace the three `message PlanSpec*` blocks with:

```proto
// Plan update event information (snapshot of plan.config.specs before
// and after a PlanService.UpdatePlan call that mutated specs).
message PlanUpdate {
  repeated Plan.Spec from_specs = 1;
  repeated Plan.Spec to_specs = 2;
}
```

- [ ] **Step 3: Format and lint**

```bash
buf format -w proto
buf lint proto
```

Expected: no output (no errors).

- [ ] **Step 4: Regenerate**

```bash
cd proto && buf generate && cd ..
```

Expected: regenerated files under `backend/generated-go/` and `frontend/src/types/proto-es/`. The build is intentionally broken at this point — callers still reference the deleted `IssueCommentPayload_PlanSpec*` types. Subsequent tasks fix them.

- [ ] **Step 5: Stage but do not commit yet**

```bash
git add proto/store/store/issue_comment.proto proto/v1/v1/issue_service.proto backend/generated-go/ frontend/src/types/proto-es/v1/ proto/gen/grpc-doc/ backend/api/mcp/gen/openapi.yaml
```

Don't commit. The commit lands at the end of Task 3 when the backend compiles again.

---

## Task 2: Replace backend emission helpers

**Files:**
- Modify: `backend/api/v1/plan_service.go`
- Modify: `backend/api/v1/plan_service_test.go`

- [ ] **Step 1: Delete the old helpers and tests**

In `backend/api/v1/plan_service.go`, delete:
- `getPlanSpecSheetSha256`
- `getPlanSpecTargets`
- `getPlanSpecEnablePriorBackup`
- `buildPlanSpecAuditIssueComments`

In `backend/api/v1/plan_service_test.go`, delete:
- `TestGetPlanSpecSheetSha256`
- `TestGetPlanSpecTargets`
- `TestGetPlanSpecEnablePriorBackup`
- `TestBuildPlanSpecAuditIssueComments`
- The `cdcSpec` helper (no longer needed)

The test file will be near-empty after deletion. Step 4 below adds one test back.

- [ ] **Step 2: Add the new helper to plan_service.go**

In `backend/api/v1/plan_service.go`, find the convertPlanSpecs function (around line 1090 in the pre-rewrite state) and add **above** it:

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

The `proto` import is already present in the file (used elsewhere). Verify it's there:

```bash
grep -n '"google.golang.org/protobuf/proto"' backend/api/v1/plan_service.go
```

Expected output: one matching line.

- [ ] **Step 3: Update the UpdatePlan call site**

In `backend/api/v1/plan_service.go`, find the call site in the `case "specs":` branch under `UpdatePlan` (currently appends to `issueCommentCreates` via `buildPlanSpecAuditIssueComments`). Locate this block:

```go
			if issue != nil {
				issueCommentCreates = append(issueCommentCreates, buildPlanSpecAuditIssueComments(issue.ProjectID, issue.UID, oldPlan.UID, oldPlan.Config.GetSpecs(), allSpecs)...)
				// Reset approval finding status
				...
```

Replace the `issueCommentCreates = append(...)` line (only that one line) with:

```go
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
```

The surrounding `if issue != nil { ... }` and approval-finding-reset logic stay unchanged.

- [ ] **Step 4: Write the failing test for planSpecsEqualSet**

Append to `backend/api/v1/plan_service_test.go`:

```go
package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func cdcSpec(id, sheet string, targets []string, priorBackup bool) *storepb.PlanConfig_Spec {
	return &storepb.PlanConfig_Spec{
		Id: id,
		Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
				SheetSha256:       sheet,
				Targets:           targets,
				EnablePriorBackup: priorBackup,
			},
		},
	}
}

func TestPlanSpecsEqualSet(t *testing.T) {
	cases := []struct {
		name string
		a, b []*storepb.PlanConfig_Spec
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "identical single spec",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			want: true,
		},
		{
			name: "same set reordered",
			a: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			b: []*storepb.PlanConfig_Spec{
				cdcSpec("s2", "sha2", []string{"db2"}, false),
				cdcSpec("s1", "sha1", []string{"db1"}, false),
			},
			want: true,
		},
		{
			name: "added spec",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			want: false,
		},
		{
			name: "removed spec",
			a: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			want: false,
		},
		{
			name: "same id sheet differs",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha2", []string{"db1"}, false)},
			want: false,
		},
		{
			name: "same id targets differ",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1", "db2"}, false)},
			want: false,
		},
		{
			name: "same id prior_backup differs",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, true)},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, planSpecsEqualSet(tc.a, tc.b))
		})
	}
}
```

- [ ] **Step 5: Build and run the unit test**

```bash
go build ./backend/...
go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run '^TestPlanSpecsEqualSet$'
```

Expected: build succeeds (proto generated types now match call sites in plan_service.go). All 8 sub-cases PASS.

If the build still fails with references to `IssueCommentPayload_PlanSpecUpdate_`, `IssueCommentPayload_PlanSpecAdd_`, or `IssueCommentPayload_PlanSpecRemove_`, those references are in `issue_service_converter.go` — they are addressed in Task 3, which must run before the build will succeed end-to-end. In that case, complete Task 3 first, then re-run this step.

- [ ] **Step 6: Stage but do not commit yet**

```bash
git add backend/api/v1/plan_service.go backend/api/v1/plan_service_test.go
```

Commit lands at the end of Task 3.

---

## Task 3: Replace backend converter

**Files:**
- Modify: `backend/api/v1/issue_service_converter.go`

- [ ] **Step 1: Update the dispatch switch**

In `backend/api/v1/issue_service_converter.go`, find `convertToIssueComment` (around line 294). Locate the dispatch switch:

```go
	switch e := ic.Payload.Event.(type) {
	case *storepb.IssueCommentPayload_Approval_:
		r.Event = convertToIssueCommentEventApproval(e)
	case *storepb.IssueCommentPayload_IssueUpdate_:
		r.Event = convertToIssueCommentEventIssueUpdate(e)
	case *storepb.IssueCommentPayload_PlanSpecUpdate_:
		projectID, _, _ := common.GetProjectIDIssueUID(issueName)
		r.Event = convertToIssueCommentEventPlanSpecUpdate(projectID, e)
	case *storepb.IssueCommentPayload_PlanSpecAdd_:
		r.Event = convertToIssueCommentEventPlanSpecAdd(e)
	case *storepb.IssueCommentPayload_PlanSpecRemove_:
		r.Event = convertToIssueCommentEventPlanSpecRemove(e)
	default:
	}
```

Replace it with:

```go
	switch e := ic.Payload.Event.(type) {
	case *storepb.IssueCommentPayload_Approval_:
		r.Event = convertToIssueCommentEventApproval(e)
	case *storepb.IssueCommentPayload_IssueUpdate_:
		r.Event = convertToIssueCommentEventIssueUpdate(e)
	case *storepb.IssueCommentPayload_PlanUpdate_:
		projectID, _, _ := common.GetProjectIDIssueUID(issueName)
		r.Event = convertToIssueCommentEventPlanUpdate(projectID, e)
	default:
	}
```

- [ ] **Step 2: Replace the converters**

In the same file, find `convertToIssueCommentEventPlanSpecUpdate`, `convertToIssueCommentEventPlanSpecAdd`, and `convertToIssueCommentEventPlanSpecRemove`. Delete all three. Replace them with a single function:

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

`convertToPlanSpecs` lives in `plan_service.go` and is already used by the live-plan view; we reuse it.

- [ ] **Step 3: Build**

```bash
go build ./backend/...
```

Expected: no output (clean build).

- [ ] **Step 4: Commit Tasks 1–3 together**

```bash
git add backend/api/v1/issue_service_converter.go
git commit -m "$(cat <<'EOF'
refactor(plan): switch spec audit to snapshot-based PlanUpdate event

Replaces the typed PlanSpecAdd / PlanSpecRemove / PlanSpecUpdate
events with a single PlanUpdate message carrying before/after
snapshots of plan.config.specs. Driven by PR #20276 review:
the snapshot shape is forward-compatible with new Spec fields,
covers reorder / CreatePlan / DeletePlan trivially, and collapses
the backend emission to a 12-line set-equality guard.

Deletes:
- buildPlanSpecAuditIssueComments
- getPlanSpecSheetSha256 / getPlanSpecTargets / getPlanSpecEnablePriorBackup
- convertToIssueCommentEventPlanSpecAdd / ...PlanSpecRemove
- their unit tests

Adds:
- planSpecsEqualSet (reorder-as-cosmetic filter)
- TestPlanSpecsEqualSet
- convertToIssueCommentEventPlanUpdate

EOF
)"
```

---

## Task 4: Retarget backend integration tests

**Files:**
- Modify: `backend/tests/plan_update_test.go`

- [ ] **Step 1: Rename listPlanSpecAuditEvents**

In `backend/tests/plan_update_test.go`, find the function:

```go
func listPlanSpecAuditEvents(t *testing.T, f *planUpdateFixture) (
	adds []*v1pb.IssueComment_PlanSpecAdd,
	removes []*v1pb.IssueComment_PlanSpecRemove,
	updates []*v1pb.IssueComment_PlanSpecUpdate,
) {
```

Replace the function entirely with:

```go
// listPlanUpdateEvents returns the PlanUpdate event payloads from the
// issue's comment list, in the order the API returned them.
func listPlanUpdateEvents(t *testing.T, f *planUpdateFixture) []*v1pb.IssueComment_PlanUpdate {
	t.Helper()
	a := require.New(t)
	resp, err := f.ctl.issueServiceClient.ListIssueComments(f.ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
		Parent: f.issue.Name,
	}))
	a.NoError(err)
	var out []*v1pb.IssueComment_PlanUpdate
	for _, c := range resp.Msg.IssueComments {
		if x := c.GetPlanUpdate(); x != nil {
			out = append(out, x)
		}
	}
	return out
}
```

- [ ] **Step 2: Retarget `TestPlanSpecUpdate_SheetChange_EmitsIssueComment`**

Find the test (it asserts on sheet sha changing via the old `updates` slice). Replace its assertion block (everything from `adds, removes, updates := listPlanSpecAuditEvents(t, f)` onward) with:

```go
	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1)
	ev := events[0]
	a.Len(ev.FromSpecs, 1)
	a.Len(ev.ToSpecs, 1)
	a.Equal(originalSpec.Id, ev.FromSpecs[0].Id)
	a.Equal(originalSpec.Id, ev.ToSpecs[0].Id)
	a.Equal(f.sheet1.Name, ev.FromSpecs[0].GetChangeDatabaseConfig().GetSheet())
	a.Equal(f.sheet2.Name, ev.ToSpecs[0].GetChangeDatabaseConfig().GetSheet())
```

- [ ] **Step 3: Retarget `TestPlanSpecAdd_EmitsIssueComment`**

Replace its assertion block with:

```go
	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1)
	ev := events[0]
	a.Len(ev.FromSpecs, 1)
	a.Len(ev.ToSpecs, 2)
	a.Equal(originalSpec.Id, ev.FromSpecs[0].Id)
	// Both old and new ids are present in the after-snapshot.
	idsAfter := map[string]bool{}
	for _, s := range ev.ToSpecs {
		idsAfter[s.Id] = true
	}
	a.True(idsAfter[originalSpec.Id])
	a.True(idsAfter[newSpecID])
```

- [ ] **Step 4: Retarget `TestPlanSpecRemove_EmitsIssueComment`**

This test does two `UpdatePlan` calls (seed add + remove). Both produce a `PlanUpdate` row now. Replace its assertion block with:

```go
	events := listPlanUpdateEvents(t, f)
	a.Len(events, 2, "expected two PlanUpdate rows: seed add + removal")
	// Second event is the removal: from has both specs, to has only the original.
	removal := events[1]
	a.Len(removal.FromSpecs, 2)
	a.Len(removal.ToSpecs, 1)
	a.Equal(originalSpec.Id, removal.ToSpecs[0].Id)
```

- [ ] **Step 5: Retarget `TestPlanSpecUpdate_TargetsChange_EmitsIssueComment`**

Replace its assertion block with:

```go
	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1)
	ev := events[0]
	a.Len(ev.FromSpecs, 1)
	a.Len(ev.ToSpecs, 1)
	a.Equal([]string{f.database.Name}, ev.FromSpecs[0].GetChangeDatabaseConfig().GetTargets())
	a.Equal([]string{f.database.Name, db2Resp.Msg.Name}, ev.ToSpecs[0].GetChangeDatabaseConfig().GetTargets())
	a.Equal(f.sheet1.Name, ev.FromSpecs[0].GetChangeDatabaseConfig().GetSheet())
	a.Equal(f.sheet1.Name, ev.ToSpecs[0].GetChangeDatabaseConfig().GetSheet())
```

- [ ] **Step 6: Retarget `TestPlanSpecUpdate_PriorBackupToggle_EmitsIssueComment`**

Replace its assertion block with:

```go
	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1)
	ev := events[0]
	a.Len(ev.FromSpecs, 1)
	a.Len(ev.ToSpecs, 1)
	a.False(ev.FromSpecs[0].GetChangeDatabaseConfig().GetEnablePriorBackup())
	a.True(ev.ToSpecs[0].GetChangeDatabaseConfig().GetEnablePriorBackup())
```

- [ ] **Step 7: `TestPlanSpecAudit_NoIssue_NoEmission` stays unchanged**

Verify with `grep`:

```bash
grep -n "TestPlanSpecAudit_NoIssue_NoEmission" backend/tests/plan_update_test.go
```

Expected: one match. The test asserts no audit is emitted when there's no issue. The current assertion (`a.Nil(f.issue, ...)`) is correct regardless of audit shape — no changes needed.

- [ ] **Step 8: Add `TestPlanUpdate_ReorderOnly_NoEmission`**

Append to `backend/tests/plan_update_test.go`:

```go
// TestPlanUpdate_ReorderOnly_NoEmission verifies that reordering specs
// without any other change is treated as cosmetic and produces no
// PlanUpdate audit row.
func TestPlanUpdate_ReorderOnly_NoEmission(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, true)

	// Seed: add a second spec so we have something to reorder.
	originalSpec := f.plan.Specs[0]
	secondSpecID := uuid.NewString()
	_, err := f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: f.plan.Name,
			Specs: []*v1pb.Plan_Spec{
				originalSpec,
				{
					Id: secondSpecID,
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets: []string{f.database.Name},
							Sheet:   f.sheet2.Name,
						},
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	// Read back to get the actual second spec (with any normalized fields).
	planResp, err := f.ctl.planServiceClient.GetPlan(f.ctx, connect.NewRequest(&v1pb.GetPlanRequest{Name: f.plan.Name}))
	a.NoError(err)
	a.Len(planResp.Msg.Specs, 2)
	specs := planResp.Msg.Specs

	// Reorder: swap the two specs, no other change.
	_, err = f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name:  f.plan.Name,
			Specs: []*v1pb.Plan_Spec{specs[1], specs[0]},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	// Only the seed step's PlanUpdate row should exist; reorder is suppressed.
	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1, "expected only the seed-add PlanUpdate row; reorder must not emit")
}
```

- [ ] **Step 9: Add `TestPlanUpdate_MultiSpec_OneRow`**

Append to the same file:

```go
// TestPlanUpdate_MultiSpec_OneRow verifies that mutating multiple specs
// in one UpdatePlan call produces exactly one PlanUpdate row whose
// snapshots capture all the changes.
func TestPlanUpdate_MultiSpec_OneRow(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, true)

	// Seed: add a second spec.
	originalSpec := f.plan.Specs[0]
	secondSpecID := uuid.NewString()
	_, err := f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: f.plan.Name,
			Specs: []*v1pb.Plan_Spec{
				originalSpec,
				{
					Id: secondSpecID,
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets: []string{f.database.Name},
							Sheet:   f.sheet1.Name,
						},
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	// Multi-mutation: change spec 1's sheet AND toggle spec 2's prior_backup.
	_, err = f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: f.plan.Name,
			Specs: []*v1pb.Plan_Spec{
				{
					Id: originalSpec.Id,
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets: []string{f.database.Name},
							Sheet:   f.sheet2.Name, // changed
						},
					},
				},
				{
					Id: secondSpecID,
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets:           []string{f.database.Name},
							Sheet:             f.sheet1.Name,
							EnablePriorBackup: true, // flipped
						},
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	events := listPlanUpdateEvents(t, f)
	a.Len(events, 2, "expected seed-add and multi-mutation PlanUpdate rows")
	multi := events[1]
	a.Len(multi.FromSpecs, 2)
	a.Len(multi.ToSpecs, 2)
	fromByID := map[string]*v1pb.Plan_Spec{}
	for _, s := range multi.FromSpecs {
		fromByID[s.Id] = s
	}
	toByID := map[string]*v1pb.Plan_Spec{}
	for _, s := range multi.ToSpecs {
		toByID[s.Id] = s
	}
	a.Equal(f.sheet1.Name, fromByID[originalSpec.Id].GetChangeDatabaseConfig().GetSheet())
	a.Equal(f.sheet2.Name, toByID[originalSpec.Id].GetChangeDatabaseConfig().GetSheet())
	a.False(fromByID[secondSpecID].GetChangeDatabaseConfig().GetEnablePriorBackup())
	a.True(toByID[secondSpecID].GetChangeDatabaseConfig().GetEnablePriorBackup())
}
```

- [ ] **Step 10: Verify the test file compiles**

```bash
go vet ./backend/tests/
```

Expected: no output. (Don't actually run the integration tests — they need Docker. CI will run them.)

- [ ] **Step 11: Commit**

```bash
git add backend/tests/plan_update_test.go
git commit -m "test(plan): retarget integration tests to PlanUpdate shape + add reorder/multi-spec coverage"
```

---

## Task 5: Retarget collision test

**Files:**
- Modify: `backend/tests/plan_audit_collision_test.go`

- [ ] **Step 1: Update the assertion**

In `backend/tests/plan_audit_collision_test.go`, find the block:

```go
	// Positive sanity: project A's issue gained the audit row.
	commentsA, err := ctl.issueServiceClient.ListIssueComments(ctx,
		connect.NewRequest(&v1pb.ListIssueCommentsRequest{Parent: fixture.IssueA.Name}))
	a.NoError(err)
	var sawUpdateA bool
	for _, c := range commentsA.Msg.IssueComments {
		if c.GetPlanSpecUpdate() != nil {
			sawUpdateA = true
			break
		}
	}
	a.True(sawUpdateA, "project A's issue should have received a PlanSpecUpdate audit row")
```

Replace with:

```go
	// Positive sanity: project A's issue gained the audit row.
	commentsA, err := ctl.issueServiceClient.ListIssueComments(ctx,
		connect.NewRequest(&v1pb.ListIssueCommentsRequest{Parent: fixture.IssueA.Name}))
	a.NoError(err)
	var sawUpdateA bool
	for _, c := range commentsA.Msg.IssueComments {
		if c.GetPlanUpdate() != nil {
			sawUpdateA = true
			break
		}
	}
	a.True(sawUpdateA, "project A's issue should have received a PlanUpdate audit row")
```

- [ ] **Step 2: Verify it compiles**

```bash
go vet ./backend/tests/
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/plan_audit_collision_test.go
git commit -m "test(collision): retarget plan audit collision test to PlanUpdate shape"
```

---

## Task 6: Backend lint and full build gate

- [ ] **Step 1: Format**

```bash
gofmt -w backend/api/v1/plan_service.go backend/api/v1/plan_service_test.go backend/api/v1/issue_service_converter.go backend/tests/plan_update_test.go backend/tests/plan_audit_collision_test.go
```

- [ ] **Step 2: Lint with auto-fix**

```bash
golangci-lint run --fix --allow-parallel-runners
```

- [ ] **Step 3: Lint until clean**

```bash
golangci-lint run --allow-parallel-runners
```

Re-run repeatedly until no output remains. Address any reported issues by hand.

- [ ] **Step 4: Full build**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: clean build.

- [ ] **Step 5: Re-run unit tests**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run '^TestPlanSpecsEqualSet$'
```

Expected: PASS.

- [ ] **Step 6: Commit any lint/format changes**

```bash
git add -u
git diff --cached --quiet || git commit -m "chore(plan): gofmt / golangci-lint cleanup for snapshot rewrite"
```

(`git diff --cached --quiet` exits non-zero when there are staged changes — the `||` runs the commit only when there is something to commit.)

---

## Task 7: Frontend diff helper

**Files:**
- Create: `frontend/src/react/pages/project/issue-detail/utils/diffPlanSpecs.ts`
- Create: `frontend/src/react/pages/project/issue-detail/utils/__tests__/diffPlanSpecs.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/react/pages/project/issue-detail/utils/__tests__/diffPlanSpecs.test.ts`:

```ts
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  Plan_ChangeDatabaseConfigSchema,
  type Plan_Spec,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { diffPlanSpecs, type SpecDiffEntry } from "../diffPlanSpecs";

function cdcSpec(opts: {
  id: string;
  sheet?: string;
  targets?: string[];
  enablePriorBackup?: boolean;
}): Plan_Spec {
  return create(Plan_SpecSchema, {
    id: opts.id,
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, {
        sheet: opts.sheet ?? "",
        targets: opts.targets ?? [],
        enablePriorBackup: opts.enablePriorBackup ?? false,
      }),
    },
  });
}

describe("diffPlanSpecs", () => {
  it("empty/empty -> []", () => {
    expect(diffPlanSpecs([], [])).toEqual([]);
  });

  it("one added", () => {
    const entries = diffPlanSpecs([], [cdcSpec({ id: "a", sheet: "s1" })]);
    expect(entries).toHaveLength(1);
    expect(entries[0].kind).toBe("added");
    expect((entries[0] as Extract<SpecDiffEntry, { kind: "added" }>).spec.id).toBe("a");
  });

  it("one removed", () => {
    const entries = diffPlanSpecs([cdcSpec({ id: "a", sheet: "s1" })], []);
    expect(entries).toHaveLength(1);
    expect(entries[0].kind).toBe("removed");
    expect((entries[0] as Extract<SpecDiffEntry, { kind: "removed" }>).spec.id).toBe("a");
  });

  it("sheet changed -> updated with sheetChanged", () => {
    const entries = diffPlanSpecs(
      [cdcSpec({ id: "a", sheet: "projects/p/sheets/s1" })],
      [cdcSpec({ id: "a", sheet: "projects/p/sheets/s2" })]
    );
    expect(entries).toHaveLength(1);
    expect(entries[0].kind).toBe("updated");
    const u = entries[0] as Extract<SpecDiffEntry, { kind: "updated" }>;
    expect(u.sheetChanged).toBe(true);
    expect(u.targetsChanged).toBe(false);
    expect(u.priorBackupChanged).toBe(false);
    expect(u.otherChanged).toBe(false);
  });

  it("targets changed -> updated with targetsChanged", () => {
    const entries = diffPlanSpecs(
      [cdcSpec({ id: "a", targets: ["db1"] })],
      [cdcSpec({ id: "a", targets: ["db1", "db2"] })]
    );
    expect(entries).toHaveLength(1);
    const u = entries[0] as Extract<SpecDiffEntry, { kind: "updated" }>;
    expect(u.sheetChanged).toBe(false);
    expect(u.targetsChanged).toBe(true);
    expect(u.priorBackupChanged).toBe(false);
  });

  it("prior_backup flipped -> updated with priorBackupChanged", () => {
    const entries = diffPlanSpecs(
      [cdcSpec({ id: "a", enablePriorBackup: false })],
      [cdcSpec({ id: "a", enablePriorBackup: true })]
    );
    expect(entries).toHaveLength(1);
    const u = entries[0] as Extract<SpecDiffEntry, { kind: "updated" }>;
    expect(u.priorBackupChanged).toBe(true);
  });

  it("all three changed -> one updated entry with all flags set", () => {
    const entries = diffPlanSpecs(
      [cdcSpec({ id: "a", sheet: "projects/p/sheets/s1", targets: ["db1"], enablePriorBackup: false })],
      [cdcSpec({ id: "a", sheet: "projects/p/sheets/s2", targets: ["db1", "db2"], enablePriorBackup: true })]
    );
    expect(entries).toHaveLength(1);
    const u = entries[0] as Extract<SpecDiffEntry, { kind: "updated" }>;
    expect(u.sheetChanged).toBe(true);
    expect(u.targetsChanged).toBe(true);
    expect(u.priorBackupChanged).toBe(true);
  });

  it("reorder-only -> []", () => {
    const a = cdcSpec({ id: "a", sheet: "projects/p/sheets/s1" });
    const b = cdcSpec({ id: "b", sheet: "projects/p/sheets/s2" });
    expect(diffPlanSpecs([a, b], [b, a])).toEqual([]);
  });

  it("mixed add + remove + update -> three entries, ADDED, REMOVED, UPDATED order", () => {
    const entries = diffPlanSpecs(
      [
        cdcSpec({ id: "old", sheet: "projects/p/sheets/x" }),
        cdcSpec({ id: "keep", sheet: "projects/p/sheets/v1" }),
      ],
      [
        cdcSpec({ id: "keep", sheet: "projects/p/sheets/v2" }),
        cdcSpec({ id: "new", sheet: "projects/p/sheets/y" }),
      ]
    );
    expect(entries.map((e) => e.kind)).toEqual(["added", "removed", "updated"]);
  });

  it("unchanged content -> []", () => {
    const a = cdcSpec({ id: "a", sheet: "projects/p/sheets/s1", targets: ["db1"], enablePriorBackup: false });
    const b = cdcSpec({ id: "a", sheet: "projects/p/sheets/s1", targets: ["db1"], enablePriorBackup: false });
    expect(diffPlanSpecs([a], [b])).toEqual([]);
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
pnpm --dir frontend test diffPlanSpecs 2>&1 | tail -10
```

Expected: failure — `Cannot find module '../diffPlanSpecs'` or equivalent.

- [ ] **Step 3: Implement diffPlanSpecs**

Create `frontend/src/react/pages/project/issue-detail/utils/diffPlanSpecs.ts`:

```ts
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";

export type SpecDiffEntry =
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

function specSheet(s: Plan_Spec): string {
  switch (s.config?.case) {
    case "changeDatabaseConfig":
      return s.config.value.sheet ?? "";
    case "exportDataConfig":
      return s.config.value.sheet ?? "";
    default:
      return "";
  }
}

function specTargets(s: Plan_Spec): string[] {
  switch (s.config?.case) {
    case "changeDatabaseConfig":
      return s.config.value.targets ?? [];
    case "exportDataConfig":
      return s.config.value.targets ?? [];
    default:
      return [];
  }
}

function specPriorBackup(s: Plan_Spec): boolean {
  switch (s.config?.case) {
    case "changeDatabaseConfig":
      return s.config.value.enablePriorBackup ?? false;
    default:
      return false;
  }
}

function arrayEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
}

// "Other" detection: zero out the fields we render specially and compare
// the JSON-stringified specs. Any residual diff is an unknown attribute
// change that we'll surface via the fallback row in the renderer.
function otherFieldsDiffer(a: Plan_Spec, b: Plan_Spec): boolean {
  const stripped = (s: Plan_Spec) => {
    const clone = JSON.parse(JSON.stringify(s));
    if (clone.config?.case === "changeDatabaseConfig" && clone.config.value) {
      clone.config.value.sheet = "";
      clone.config.value.targets = [];
      clone.config.value.enablePriorBackup = false;
    }
    if (clone.config?.case === "exportDataConfig" && clone.config.value) {
      clone.config.value.sheet = "";
      clone.config.value.targets = [];
    }
    return JSON.stringify(clone);
  };
  return stripped(a) !== stripped(b);
}

export function diffPlanSpecs(from: Plan_Spec[], to: Plan_Spec[]): SpecDiffEntry[] {
  const fromById = new Map(from.map((s) => [s.id, s]));
  const toById = new Map(to.map((s) => [s.id, s]));
  const out: SpecDiffEntry[] = [];

  for (const s of to) {
    if (!fromById.has(s.id)) out.push({ kind: "added", spec: s });
  }
  for (const s of from) {
    if (!toById.has(s.id)) out.push({ kind: "removed", spec: s });
  }
  for (const newSpec of to) {
    const oldSpec = fromById.get(newSpec.id);
    if (!oldSpec) continue;
    const sheetChanged = specSheet(oldSpec) !== specSheet(newSpec);
    const targetsChanged = !arrayEqual(specTargets(oldSpec), specTargets(newSpec));
    const priorBackupChanged = specPriorBackup(oldSpec) !== specPriorBackup(newSpec);
    const otherChanged = otherFieldsDiffer(oldSpec, newSpec);
    if (sheetChanged || targetsChanged || priorBackupChanged || otherChanged) {
      out.push({
        kind: "updated",
        specId: newSpec.id,
        from: oldSpec,
        to: newSpec,
        sheetChanged,
        targetsChanged,
        priorBackupChanged,
        otherChanged,
      });
    }
  }
  return out;
}

// diffEntryKey produces a stable React key for a diff entry.
export function diffEntryKey(entry: SpecDiffEntry): string {
  switch (entry.kind) {
    case "added":
      return `add:${entry.spec.id}`;
    case "removed":
      return `rm:${entry.spec.id}`;
    case "updated":
      return `up:${entry.specId}`;
  }
}
```

- [ ] **Step 4: Run the test**

```bash
pnpm --dir frontend test diffPlanSpecs 2>&1 | tail -10
```

Expected: all 10 cases PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/project/issue-detail/utils/diffPlanSpecs.ts \
        frontend/src/react/pages/project/issue-detail/utils/__tests__/diffPlanSpecs.test.ts
git commit -m "feat(frontend): pure diff helper for PlanUpdate snapshot pairs"
```

---

## Task 8: Frontend renderer + classifier rewrite

**Files:**
- Modify: `frontend/src/store/modules/v1/issueComment.ts`
- Modify: `frontend/src/react/pages/project/issue-detail/components/IssueDetailCommentList.tsx`

These two files must be changed together — renaming the enum entry without updating the renderer would leave the file type-broken.

- [ ] **Step 1: Update the classifier**

In `frontend/src/store/modules/v1/issueComment.ts`, replace the `IssueCommentType` enum block:

```ts
export enum IssueCommentType {
  USER_COMMENT = "USER_COMMENT",
  APPROVAL = "APPROVAL",
  ISSUE_UPDATE = "ISSUE_UPDATE",
  PLAN_SPEC_UPDATE = "PLAN_SPEC_UPDATE",
  PLAN_SPEC_ADD = "PLAN_SPEC_ADD",
  PLAN_SPEC_REMOVE = "PLAN_SPEC_REMOVE",
}
```

with:

```ts
export enum IssueCommentType {
  USER_COMMENT = "USER_COMMENT",
  APPROVAL = "APPROVAL",
  ISSUE_UPDATE = "ISSUE_UPDATE",
  PLAN_UPDATE = "PLAN_UPDATE",
}
```

Replace the `getIssueCommentType` function:

```ts
export const getIssueCommentType = (
  issueComment: IssueComment
): IssueCommentType => {
  if (issueComment.event?.case === "approval") {
    return IssueCommentType.APPROVAL;
  } else if (issueComment.event?.case === "issueUpdate") {
    return IssueCommentType.ISSUE_UPDATE;
  } else if (issueComment.event?.case === "planSpecUpdate") {
    return IssueCommentType.PLAN_SPEC_UPDATE;
  } else if (issueComment.event?.case === "planSpecAdd") {
    return IssueCommentType.PLAN_SPEC_ADD;
  } else if (issueComment.event?.case === "planSpecRemove") {
    return IssueCommentType.PLAN_SPEC_REMOVE;
  }
  return IssueCommentType.USER_COMMENT;
};
```

with:

```ts
export const getIssueCommentType = (
  issueComment: IssueComment
): IssueCommentType => {
  if (issueComment.event?.case === "approval") {
    return IssueCommentType.APPROVAL;
  } else if (issueComment.event?.case === "issueUpdate") {
    return IssueCommentType.ISSUE_UPDATE;
  } else if (issueComment.event?.case === "planUpdate") {
    return IssueCommentType.PLAN_UPDATE;
  }
  return IssueCommentType.USER_COMMENT;
};
```

- [ ] **Step 2: Replace the three dispatch branches in IssueDetailCommentList.tsx**

Open `frontend/src/react/pages/project/issue-detail/components/IssueDetailCommentList.tsx`. Find the three sequential `if` blocks for `PLAN_SPEC_UPDATE`, `PLAN_SPEC_ADD`, and `PLAN_SPEC_REMOVE` inside `CommentActionSentence`. Replace **all three** with one block:

```tsx
  if (
    commentType === IssueCommentType.PLAN_UPDATE &&
    issueComment.event.case === "planUpdate"
  ) {
    const { fromSpecs, toSpecs } = issueComment.event.value;
    const entries = diffPlanSpecs(fromSpecs, toSpecs);
    if (entries.length === 0) return null;
    return (
      <div className="flex w-full flex-col gap-1">
        {entries.map((entry) => (
          <SpecDiffRow key={diffEntryKey(entry)} entry={entry} />
        ))}
      </div>
    );
  }
```

Add the imports at the top of the file:

```ts
import { diffEntryKey, diffPlanSpecs, type SpecDiffEntry } from "../utils/diffPlanSpecs";
```

- [ ] **Step 3: Add the SpecDiffRow component**

Insert `SpecDiffRow` immediately after `SpecChangeRow` in the same file:

```tsx
function SpecDiffRow({ entry }: { entry: SpecDiffEntry }) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const planName = page.plan?.name ?? "";

  if (entry.kind === "added") {
    return (
      <SpecChangeRow specRef={specResourceName(planName, entry.spec)}>
        {t("activity.sentence.added-spec")}
      </SpecChangeRow>
    );
  }

  if (entry.kind === "removed") {
    return (
      <SpecChangeRow specRef={specResourceName(planName, entry.spec)}>
        {t("activity.sentence.removed-spec")}
      </SpecChangeRow>
    );
  }

  // updated
  const fragments: ReactNode[] = [];
  let trailing: ReactNode = null;
  if (entry.sheetChanged) {
    const fromSheet = sheetFromSpec(entry.from);
    const toSheet = sheetFromSpec(entry.to);
    fragments.push(
      <span key="sheet">{t("activity.sentence.modified-sql-of")}</span>
    );
    trailing = (
      <IssueDetailStatementUpdateButton
        newSheet={toSheet}
        oldSheet={fromSheet}
      />
    );
  }
  if (entry.targetsChanged) {
    const fromTargets = targetsFromSpec(entry.from);
    const toTargets = targetsFromSpec(entry.to);
    const fromSet = new Set(fromTargets);
    const toSet = new Set(toTargets);
    const added = toTargets.filter((x) => !fromSet.has(x));
    const removed = fromTargets.filter((x) => !toSet.has(x));
    fragments.push(
      <span key="targets" className="inline-flex items-center gap-1">
        {t("activity.sentence.changed-targets-of")}{" "}
        <span className="text-xs">
          {added.length > 0 ? `+${added.join(", ")}` : ""}
          {added.length > 0 && removed.length > 0 ? "  " : ""}
          {removed.length > 0 ? `-${removed.join(", ")}` : ""}
        </span>
      </span>
    );
  }
  if (entry.priorBackupChanged) {
    const flipped = priorBackupFromSpec(entry.to);
    fragments.push(
      <span key="backup">
        {flipped
          ? t("activity.sentence.enabled-prior-backup-on")
          : t("activity.sentence.disabled-prior-backup-on")}
      </span>
    );
  }
  if (fragments.length === 0 && entry.otherChanged) {
    // Unknown attribute change — generic fallback. The renderer doesn't
    // know what changed; show a "spec updated" row with a details toggle
    // for the raw before/after JSON.
    return (
      <SpecChangeRow specRef={specResourceName(planName, entry.to)}>
        <span>{t("common.updated")}</span>
        <details className="ml-2 text-xs">
          <summary className="cursor-pointer text-control-light">
            {t("common.details")}
          </summary>
          <pre className="mt-1 max-h-48 overflow-auto rounded bg-control-bg p-2">
            {JSON.stringify(
              { from: entry.from, to: entry.to },
              null,
              2
            )}
          </pre>
        </details>
      </SpecChangeRow>
    );
  }

  return (
    <SpecChangeRow specRef={specResourceName(planName, entry.to)} trailing={trailing}>
      {joinFragments(fragments, t("common.and"))}
    </SpecChangeRow>
  );
}

// SpecChangeRow expects a full spec resource name so getSpecDisplayInfo
// and the UUID-prefix fallback regex both work. The audit payload only
// carries Plan_Spec objects (with a bare UUID in `id`), so we glue the
// live plan's resource name onto it.
function specResourceName(planName: string, spec: Plan_Spec): string {
  return planName ? `${planName}/specs/${spec.id}` : `specs/${spec.id}`;
}

function sheetFromSpec(s: Plan_Spec): string {
  switch (s.config?.case) {
    case "changeDatabaseConfig":
      return s.config.value.sheet ?? "";
    case "exportDataConfig":
      return s.config.value.sheet ?? "";
    default:
      return "";
  }
}

function targetsFromSpec(s: Plan_Spec): string[] {
  switch (s.config?.case) {
    case "changeDatabaseConfig":
      return s.config.value.targets ?? [];
    case "exportDataConfig":
      return s.config.value.targets ?? [];
    default:
      return [];
  }
}

function priorBackupFromSpec(s: Plan_Spec): boolean {
  return s.config?.case === "changeDatabaseConfig"
    ? s.config.value.enablePriorBackup ?? false
    : false;
}
```

Add the `Plan_Spec` type import at the top of the file:

```ts
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
```

**About `specResourceName`:** the audit payload contains `Plan_Spec` objects whose `id` field is the spec UUID. `SpecChangeRow` expects a full resource name (`projects/{p}/plans/{plan}/specs/{specId}`) for resolving the chip. Since this audit row lives on an issue page that already has `page.plan`, the existing `SpecChangeRow` code falls back to parsing the UUID out of whatever string we give it. Passing just the UUID works — `getSpecDisplayInfo` matches by spec id against `page.plan.specs`, so we don't need the resource-name prefix. **Verify by inspecting `getSpecDisplayInfo`'s implementation in `@/utils`** before relying on this — if it requires the full resource name, change `specResourceName` to construct it via `${page.plan?.name}/specs/${spec.id}`.

- [ ] **Step 4: Update CommentActionIcon**

Find the icon dispatch in `CommentActionIcon`. Delete the three blocks:

```tsx
  if (commentType === IssueCommentType.PLAN_SPEC_UPDATE) { ... }
  if (commentType === IssueCommentType.PLAN_SPEC_ADD) { ... }
  if (commentType === IssueCommentType.PLAN_SPEC_REMOVE) { ... }
```

Replace with a single block:

```tsx
  if (commentType === IssueCommentType.PLAN_UPDATE) {
    return (
      <CommentIconBadge
        className="bg-control-bg text-control"
        icon={<Pencil className="h-4 w-4" />}
      />
    );
  }
```

Remove the now-unused `Plus` and `Minus` imports from `lucide-react` if they aren't referenced elsewhere in the file:

```bash
grep -n 'Plus\|Minus' frontend/src/react/pages/project/issue-detail/components/IssueDetailCommentList.tsx
```

If the only matches are in the import statement, remove `Plus` and `Minus` from the `lucide-react` import. If they're used elsewhere, leave them.

- [ ] **Step 5: Type-check**

```bash
pnpm --dir frontend type-check 2>&1 | tail -5
```

Expected: clean (no TS errors).

- [ ] **Step 6: Run the unit test again**

```bash
pnpm --dir frontend test diffPlanSpecs 2>&1 | tail -5
```

Expected: 10/10 PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/store/modules/v1/issueComment.ts \
        frontend/src/react/pages/project/issue-detail/components/IssueDetailCommentList.tsx
git commit -m "$(cat <<'EOF'
feat(frontend): render snapshot-based PlanUpdate audit rows via diff helper

Replace the three planSpecUpdate / planSpecAdd / planSpecRemove
dispatch branches with one planUpdate branch that calls diffPlanSpecs
and maps each entry to a SpecDiffRow. Reuses SpecChangeRow,
joinFragments, IssueDetailStatementUpdateButton, and all i18n keys.
Adds a generic fallback row (with JSON-diff details toggle) for the
case where a Spec field changed that the renderer doesn't yet handle
specially. Classifier collapses to a single PLAN_UPDATE type.

EOF
)"
```

---

## Task 9: Frontend full gate

- [ ] **Step 1: Auto-fix**

```bash
pnpm --dir frontend fix 2>&1 | tail -5
```

Expected: minor formatting fixes only.

- [ ] **Step 2: Check**

```bash
pnpm --dir frontend check 2>&1 | tail -10
```

Expected:
```
React i18n: all checks passed (missing keys, unused keys, cross-locale consistency).
React layering policy: all checks passed.
Locale sorter: all 30 file(s) are normalized.
```

If unused-key errors surface for the existing locale keys (`added`, `removed`, etc.), it likely means the renderer isn't referencing them. Re-check Step 3 of Task 8.

- [ ] **Step 3: Type-check**

```bash
pnpm --dir frontend type-check 2>&1 | tail -3
```

Expected: clean.

- [ ] **Step 4: Run all frontend tests**

```bash
pnpm --dir frontend test 2>&1 | tail -10
```

Expected: all suites pass, including `diffPlanSpecs.test.ts`.

- [ ] **Step 5: Commit any fix/check changes**

```bash
git add -u
git diff --cached --quiet || git commit -m "chore(frontend): formatting cleanup after PlanUpdate rewrite"
```

---

## Task 10: Manual UI verification

**Files:** none (manual, per CLAUDE.md UI rule)

- [ ] **Step 1: Start dev backend**

```bash
PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug
```

Leave running.

- [ ] **Step 2: Start dev frontend**

```bash
pnpm --dir frontend dev
```

Open the local URL.

- [ ] **Step 3: Walk through every audit-row scenario**

For each, create an issue with a linked plan, then perform the action and verify the comment row.

1. **Add a spec** → one row, content reads "Adela added Change <id8>". Chip clickable, updates `?spec=<id>`, selects the spec inline.
2. **Remove a spec** → one row, "Adela removed Change <id8>". Chip non-clickable (spec gone from live plan).
3. **Change a spec's SQL only** → one row, "Adela modified SQL of Change <id8> [View Details]". Diff button after the chip. Dialog fills the viewport.
4. **Change a spec's targets only** → one row, "Adela changed targets of Change <id8> +db_added −db_removed".
5. **Toggle `enable_prior_backup`** → one row, "Adela enabled/disabled prior backup on Change <id8>".
6. **Multi-spec change in one save** → one comment row with multiple visual entries (one per affected spec) stacked.
7. **Reorder specs without other changes** → no new comment row appears.
8. **Rename issue title to itself (autoblur with no edit)** → no new comment row (the bundled `UpdateIssue` no-op guard).

- [ ] **Step 4: Fix and re-verify if anything broke**

If any scenario fails, fix the renderer / helper / diff logic and re-verify before continuing.

---

## Task 11: Force-push the rewrite

- [ ] **Step 1: Final sanity sweep**

```bash
git log --oneline origin/main..HEAD
go build ./backend/...
pnpm --dir frontend type-check 2>&1 | tail -3
pnpm --dir frontend check 2>&1 | tail -3
```

- [ ] **Step 2: Force-push with lease**

```bash
git push --force-with-lease origin feat/plan-spec-mutation-audit-a2
```

`--force-with-lease` aborts the push if origin moved unexpectedly (someone else pushed to the same branch).

- [ ] **Step 3: Post a follow-up comment on PR #20276**

```bash
gh pr comment 20276 --body "$(cat <<'EOF'
Rewrote per the snapshot proposal in your review. Summary:

- `PlanUpdate` message replaces `PlanSpecAdd` / `PlanSpecRemove` / `PlanSpecUpdate`. One audit row per `UpdatePlan` call, carrying `from_specs` / `to_specs`.
- Backend emission collapses to a 12-line `planSpecsEqualSet` guard.
- Per-spec diff lives in `frontend/src/react/pages/project/issue-detail/utils/diffPlanSpecs.ts` (pure helper, 10-case test).
- Reorder filtered as cosmetic on the backend; renderer also no-ops empty diffs defensively.
- Unknown attribute changes fall back to a generic "updated" row with a JSON-diff `<details>` toggle.
- New design doc: `docs/plans/2026-05-11-plan-mutation-audit-snapshot-design.md`.
- Implementation plan: `docs/plans/2026-05-11-plan-mutation-audit-snapshot.md`.

PTAL.
EOF
)"
```

- [ ] **Step 4: Verify PR state**

```bash
gh pr view 20276 --json state,headRefName,reviewDecision
```

Expected: `state: OPEN`, branch name unchanged, review decision unchanged (reviewer needs to re-approve after seeing the rewrite).

---

## Out of scope (deferred — same as superseded design)

- Plans without a linked issue (G3) — needs a future `plan_audit` table.
- Self-approval eligibility / editor tracking (the *control* gap from BYT-9175).
- `CreatePlan` / `DeletePlan` audit emission — snapshot shape supports them trivially, but they introduce their own semantic questions; out of scope here.
- Auditing non-spec plan attributes (title, description, state).
- Real-time notifications / webhook delivery for plan mutations.
