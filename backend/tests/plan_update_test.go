package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// planUpdateFixture holds a project + Postgres instance + database + two sheets
// + plan + linked issue, ready for tests that mutate the plan and inspect
// the resulting issue_comment audit rows.
type planUpdateFixture struct {
	ctl      *controller
	ctx      context.Context
	instance *v1pb.Instance
	database *v1pb.Database
	plan     *v1pb.Plan
	issue    *v1pb.Issue
	sheet1   *v1pb.Sheet
	sheet2   *v1pb.Sheet
}

// setupPlanUpdateFixture creates the fixture. withIssue=false skips the
// CreateIssue call (used by the no-issue gate test).
func setupPlanUpdateFixture(t *testing.T, withIssue bool) *planUpdateFixture {
	t.Helper()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	t.Cleanup(func() { _ = ctl.Close(ctx) })
	// The fixture creates a database through a bootstrap Plan owned and approved
	// by the test user. Keep that setup flow independent of the feature under
	// test; individual tests can set the project policy they need afterwards.
	_, err = ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project: &v1pb.Project{
			Name:                        ctl.project.Name,
			AllowLastPlanEditorApproval: true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_last_plan_editor_approval"}},
	}))
	a.NoError(err)

	instanceName := "planUpdateInstance_" + generateRandomString("inst")
	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       instanceName,
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	databaseName := "planUpdateDb_" + generateRandomString("db")
	a.NoError(ctl.createDatabase(ctx, ctl.project, instance, nil, databaseName, ""))
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)

	mkSheet := func(content string) *v1pb.Sheet {
		s, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet:  &v1pb.Sheet{Content: []byte(content)},
		}))
		a.NoError(err)
		return s.Msg
	}
	sheet1 := mkSheet("CREATE TABLE plan_update_a (id INTEGER PRIMARY KEY);")
	sheet2 := mkSheet("CREATE TABLE plan_update_b (id INTEGER PRIMARY KEY);")

	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{databaseResp.Msg.Name},
						Sheet:   sheet1.Name,
					},
				},
			}},
		},
	}))
	a.NoError(err)

	f := &planUpdateFixture{
		ctl:      ctl,
		ctx:      ctx,
		instance: instance,
		database: databaseResp.Msg,
		plan:     planResp.Msg,
		sheet1:   sheet1,
		sheet2:   sheet2,
	}
	if withIssue {
		issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
			Parent: ctl.project.Name,
			Issue: &v1pb.Issue{
				Type:        v1pb.Issue_DATABASE_CHANGE,
				Title:       "plan-audit-test",
				Description: "plan-audit-test",
				Plan:        f.plan.Name,
			},
		}))
		a.NoError(err)
		f.issue = issueResp.Msg
	}
	return f
}

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

func TestPlanLastEditorAttributionAndApproval(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, false)
	newProjectID := generateRandomString("last-editor-default")
	newProject, err := f.ctl.projectServiceClient.CreateProject(f.ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project:   &v1pb.Project{Title: newProjectID},
		ProjectId: newProjectID,
	}))
	a.NoError(err)
	a.False(newProject.Msg.AllowLastPlanEditorApproval)
	a.Equal("users/demo@example.com", f.plan.LastPlanEditor)
	projectResp, err := f.ctl.projectServiceClient.UpdateProject(f.ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project: &v1pb.Project{
			Name:                        f.ctl.project.Name,
			AllowSelfApproval:           false,
			AllowLastPlanEditorApproval: false,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_self_approval", "allow_last_plan_editor_approval"}},
	}))
	a.NoError(err)
	a.False(projectResp.Msg.AllowLastPlanEditorApproval)

	installWorkspaceApprovalRule(f.ctx, t, f.ctl, f.ctl.project, []string{"roles/projectOwner"})
	editor := provisionApprover(f.ctx, t, f.ctl, f.ctl.project, "last-plan-editor", "roles/projectOwner")

	updateSpecs := func(specs []*v1pb.Plan_Spec) *v1pb.Plan {
		var updated *v1pb.Plan
		withImpersonation(f.ctx, t, f.ctl, editor, func() {
			resp, err := f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
				Plan:       &v1pb.Plan{Name: f.plan.Name, Specs: specs},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
			}))
			a.NoError(err)
			updated = resp.Msg
		})
		return updated
	}

	// An identical specs update is accepted and must still attribute the editor.
	updated := updateSpecs(f.plan.Specs)
	a.Equal("users/"+editor.Email, updated.LastPlanEditor)

	secondSpec := &v1pb.Plan_Spec{
		Id: uuid.NewString(),
		Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
				Targets: []string{f.database.Name},
				Sheet:   f.sheet2.Name,
			},
		},
	}
	updated = updateSpecs(append(updated.Specs, secondSpec))
	// Reordering specs is accepted and must also attribute the editor.
	updated = updateSpecs([]*v1pb.Plan_Spec{updated.Specs[1], updated.Specs[0]})
	a.Equal("users/"+editor.Email, updated.LastPlanEditor)

	// Metadata and lifecycle updates must not transfer Last Plan Editor.
	metadataEditor := provisionApprover(f.ctx, t, f.ctl, f.ctl.project, "metadata-editor", "roles/projectOwner")
	withImpersonation(f.ctx, t, f.ctl, metadataEditor, func() {
		resp, err := f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
			Plan: &v1pb.Plan{
				Name:        updated.Name,
				Title:       "metadata update",
				Description: "metadata update",
				State:       v1pb.State_DELETED,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "description", "state"}},
		}))
		a.NoError(err)
		a.Equal("users/"+editor.Email, resp.Msg.LastPlanEditor)
		resp, err = f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
			Plan:       &v1pb.Plan{Name: updated.Name, State: v1pb.State_ACTIVE},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
		}))
		a.NoError(err)
		a.Equal("users/"+editor.Email, resp.Msg.LastPlanEditor)
		updated = resp.Msg
	})

	issue := createIssueForPlan(f.ctx, t, f.ctl, f.ctl.project, updated, "last plan editor approval")
	waitForIssuePending(f.ctx, t, f.ctl, issue)

	withImpersonation(f.ctx, t, f.ctl, editor, func() {
		_, err := f.ctl.issueServiceClient.ApproveIssue(f.ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{Name: issue.Name}))
		a.Error(err)
		a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	renamedEmail := "renamed-" + editor.Email
	renamed, err := f.ctl.userServiceClient.UpdateEmail(f.ctx, connect.NewRequest(&v1pb.UpdateEmailRequest{
		Name:  common.FormatUserEmail(editor.Email),
		Email: renamedEmail,
	}))
	a.NoError(err)
	a.Equal(renamedEmail, renamed.Msg.Email)
	editor.Email = renamedEmail
	gotPlan, err := f.ctl.planServiceClient.GetPlan(f.ctx, connect.NewRequest(&v1pb.GetPlanRequest{Name: updated.Name}))
	a.NoError(err)
	a.Equal(common.FormatUserEmail(renamedEmail), gotPlan.Msg.LastPlanEditor)

	withImpersonation(f.ctx, t, f.ctl, editor, func() {
		_, err := f.ctl.issueServiceClient.ApproveIssue(f.ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{Name: issue.Name}))
		a.Error(err)
		a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
		_, err = f.ctl.issueServiceClient.RejectIssue(f.ctx, connect.NewRequest(&v1pb.RejectIssueRequest{Name: issue.Name}))
		a.NoError(err)
	})

	projectResp, err = f.ctl.projectServiceClient.UpdateProject(f.ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project: &v1pb.Project{
			Name:                        f.ctl.project.Name,
			AllowLastPlanEditorApproval: true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_last_plan_editor_approval"}},
	}))
	a.NoError(err)
	a.True(projectResp.Msg.AllowLastPlanEditorApproval)
	issueResp, err := f.ctl.issueServiceClient.GetIssue(f.ctx, connect.NewRequest(&v1pb.GetIssueRequest{Name: issue.Name}))
	a.NoError(err)
	a.Len(issueResp.Msg.Approvers, 1)
	a.Equal(v1pb.Issue_Approver_REJECTED, issueResp.Msg.Approvers[0].Status)
	requestIssueAsCreator(f.ctx, t, f.ctl, issue, "updated")

	withImpersonation(f.ctx, t, f.ctl, editor, func() {
		_, err := f.ctl.issueServiceClient.ApproveIssue(f.ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{Name: issue.Name}))
		a.NoError(err)
	})

	// Enabling last-editor approval does not override the self-approval rule.
	var selfPlan *v1pb.Plan
	var selfIssue *v1pb.Issue
	withImpersonation(f.ctx, t, f.ctl, editor, func() {
		planResp, err := f.ctl.planServiceClient.CreatePlan(f.ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
			Parent: f.ctl.project.Name,
			Plan:   &v1pb.Plan{Specs: updated.Specs},
		}))
		a.NoError(err)
		selfPlan = planResp.Msg
		issueResp, err := f.ctl.issueServiceClient.CreateIssue(f.ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
			Parent: f.ctl.project.Name,
			Issue: &v1pb.Issue{
				Type:  v1pb.Issue_DATABASE_CHANGE,
				Title: "self approval remains disabled",
				Plan:  selfPlan.Name,
			},
		}))
		a.NoError(err)
		selfIssue = issueResp.Msg
	})
	waitForIssuePending(f.ctx, t, f.ctl, selfIssue)
	withImpersonation(f.ctx, t, f.ctl, editor, func() {
		_, err := f.ctl.issueServiceClient.ApproveIssue(f.ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{Name: selfIssue.Name}))
		a.Error(err)
		a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))
	})
}

// TestPlanSpecUpdate_SheetChange_EmitsIssueComment is the original test from
// PR #20032: changing a spec's sheet emits a single PlanSpecUpdate row.
func TestPlanSpecUpdate_SheetChange_EmitsIssueComment(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, true)

	originalSpec := f.plan.Specs[0]
	_, err := f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: f.plan.Name,
			Specs: []*v1pb.Plan_Spec{{
				Id: originalSpec.Id,
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{f.database.Name},
						Sheet:   f.sheet2.Name, // changed
					},
				},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1)
	ev := events[0]
	a.Len(ev.FromSpecs, 1)
	a.Len(ev.ToSpecs, 1)
	a.Equal(originalSpec.Id, ev.FromSpecs[0].Id)
	a.Equal(originalSpec.Id, ev.ToSpecs[0].Id)
	a.Equal(f.sheet1.Name, ev.FromSpecs[0].GetChangeDatabaseConfig().GetSheet())
	a.Equal(f.sheet2.Name, ev.ToSpecs[0].GetChangeDatabaseConfig().GetSheet())
}

func TestPlanSpecAdd_EmitsIssueComment(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, true)

	originalSpec := f.plan.Specs[0]
	newSpecID := uuid.NewString()
	_, err := f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: f.plan.Name,
			Specs: []*v1pb.Plan_Spec{
				originalSpec,
				{
					Id: newSpecID,
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

	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1)
	ev := events[0]
	a.Len(ev.FromSpecs, 1)
	a.Len(ev.ToSpecs, 2)
	a.Equal(originalSpec.Id, ev.FromSpecs[0].Id)
	idsAfter := map[string]bool{}
	for _, s := range ev.ToSpecs {
		idsAfter[s.Id] = true
	}
	a.True(idsAfter[originalSpec.Id])
	a.True(idsAfter[newSpecID])
}

func TestPlanSpecRemove_EmitsIssueComment(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, true)

	originalSpec := f.plan.Specs[0]
	secondSpecID := uuid.NewString()
	// Seed: add a second spec we can later remove.
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

	// Now remove the second spec.
	_, err = f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name:  f.plan.Name,
			Specs: []*v1pb.Plan_Spec{originalSpec},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	events := listPlanUpdateEvents(t, f)
	a.Len(events, 2, "expected two PlanUpdate rows: seed add + removal")
	removal := events[1]
	a.Len(removal.FromSpecs, 2)
	a.Len(removal.ToSpecs, 1)
	a.Equal(originalSpec.Id, removal.ToSpecs[0].Id)
	idsBefore := map[string]bool{}
	for _, s := range removal.FromSpecs {
		idsBefore[s.Id] = true
	}
	a.True(idsBefore[secondSpecID])
}

func TestPlanSpecUpdate_TargetsChange_EmitsIssueComment(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, true)

	dbName2 := "planUpdateDb2_" + generateRandomString("db")
	a.NoError(f.ctl.createDatabase(f.ctx, f.ctl.project, f.instance, nil, dbName2, ""))
	db2Resp, err := f.ctl.databaseServiceClient.GetDatabase(f.ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", f.instance.Name, dbName2),
	}))
	a.NoError(err)

	originalSpec := f.plan.Specs[0]
	_, err = f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: f.plan.Name,
			Specs: []*v1pb.Plan_Spec{{
				Id: originalSpec.Id,
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{f.database.Name, db2Resp.Msg.Name}, // added db2
						Sheet:   f.sheet1.Name,                               // unchanged
					},
				},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1)
	ev := events[0]
	a.Len(ev.FromSpecs, 1)
	a.Len(ev.ToSpecs, 1)
	a.Equal([]string{f.database.Name}, ev.FromSpecs[0].GetChangeDatabaseConfig().GetTargets())
	a.Equal([]string{f.database.Name, db2Resp.Msg.Name}, ev.ToSpecs[0].GetChangeDatabaseConfig().GetTargets())
	a.Equal(f.sheet1.Name, ev.FromSpecs[0].GetChangeDatabaseConfig().GetSheet())
	a.Equal(f.sheet1.Name, ev.ToSpecs[0].GetChangeDatabaseConfig().GetSheet())
}

func TestPlanSpecUpdate_PriorBackupToggle_EmitsIssueComment(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, true)

	originalSpec := f.plan.Specs[0]
	_, err := f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: f.plan.Name,
			Specs: []*v1pb.Plan_Spec{{
				Id: originalSpec.Id,
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets:           []string{f.database.Name}, // unchanged
						Sheet:             f.sheet1.Name,             // unchanged
						EnablePriorBackup: true,                      // flipped from false
					},
				},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)

	events := listPlanUpdateEvents(t, f)
	a.Len(events, 1)
	ev := events[0]
	a.Len(ev.FromSpecs, 1)
	a.Len(ev.ToSpecs, 1)
	a.False(ev.FromSpecs[0].GetChangeDatabaseConfig().GetEnablePriorBackup())
	a.True(ev.ToSpecs[0].GetChangeDatabaseConfig().GetEnablePriorBackup())
}

// TestPlanSpecAudit_NoIssue_NoEmission documents the G3 deferral: when the
// plan is not linked to an issue, UpdatePlan emits no audit rows. The
// assertion is indirect (no panic / FK violation when the audit emission
// is skipped), since there is no issue to ListIssueComments on.
func TestPlanSpecAudit_NoIssue_NoEmission(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupPlanUpdateFixture(t, false) // no issue linked

	originalSpec := f.plan.Specs[0]
	_, err := f.ctl.planServiceClient.UpdatePlan(f.ctx, connect.NewRequest(&v1pb.UpdatePlanRequest{
		Plan: &v1pb.Plan{
			Name: f.plan.Name,
			Specs: []*v1pb.Plan_Spec{{
				Id: originalSpec.Id,
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{f.database.Name},
						Sheet:   f.sheet2.Name, // sheet change
					},
				},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"specs"}},
	}))
	a.NoError(err)
	a.Nil(f.issue, "fixture intentionally has no linked issue")
}

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
