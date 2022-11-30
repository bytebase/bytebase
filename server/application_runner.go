package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/app/feishu"
	"github.com/bytebase/bytebase/store"
)

const (
	applicationRunnerInterval = time.Duration(60) * time.Second

	// externalApprovalCancelReasonGeneral is the general reason, used as a default.
	externalApprovalCancelReasonGeneral string = "Canceled because the assignee has been changed, or all tasks of the stage have been approved or the issue is no longer open."
	// externalApprovalCancelReasonIssueNotOpen is used if the issue is not open.
	externalApprovalCancelReasonIssueNotOpen string = "Canceled because the containing issue is no longer open."
	// externalApprovalCancelReasonReassigned is used if the assignee has been changed.
	externalApprovalCancelReasonReassigned string = "Canceled because the assignee has changed."
	// externalApprovalCancelReasonNoTaskPendingApproval is used if there is no pending approval tasks.
	externalApprovalCancelReasonNoTaskPendingApproval string = "Canceled because all tasks have benn approved."
)

// NewApplicationRunner returns a ApplicationRunner.
func NewApplicationRunner(store *store.Store, activityManager *ActivityManager, feishuProvider *feishu.Provider) *ApplicationRunner {
	return &ApplicationRunner{
		store:           store,
		activityManager: activityManager,
		p:               feishuProvider,
	}
}

// ApplicationRunner is a runner which periodically checks external approval status and approve the correspoding stages.
type ApplicationRunner struct {
	store           *store.Store
	activityManager *ActivityManager
	p               *feishu.Provider
}

// Run runs the ApplicationRunner.
func (r *ApplicationRunner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(applicationRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Application runner started and will run every %v", applicationRunnerInterval))
	// Try to update approval definition if external approval is enabled, because our approval definition may have changed.
	if err := r.tryUpdateApprovalDefinition(ctx); err != nil {
		log.Error("failed to update approval definition on application runner start", zap.Error(err))
	}
	for {
		select {
		case <-ticker.C:
			func() {
				settingName := api.SettingAppIM
				setting, err := r.store.GetSetting(ctx, &api.SettingFind{Name: &settingName})
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						log.Error("failed to get IM setting", zap.String("settingName", string(settingName)), zap.Error(err))
					}
					return
				}
				if setting == nil {
					log.Error("cannot find IM setting")
					return
				}
				if setting.Value == "" {
					return
				}
				var value api.SettingAppIMValue
				if err := json.Unmarshal([]byte(setting.Value), &value); err != nil {
					log.Error("failed to unmarshal IM setting value", zap.String("settingName", string(settingName)), zap.Any("settingValue", setting.Value), zap.Error(err))
					return
				}
				if !value.ExternalApproval.Enabled {
					return
				}

				externalApprovalList, err := r.store.FindExternalApproval(ctx, &api.ExternalApprovalFind{})
				if err != nil {
					log.Error("failed to find external approval list", zap.Error(err))
					return
				}

				for _, externalApproval := range externalApprovalList {
					switch externalApproval.Type {
					case api.ExternalApprovalTypeFeishu:
						var payload api.ExternalApprovalPayloadFeishu
						if err := json.Unmarshal([]byte(externalApproval.Payload), &payload); err != nil {
							log.Error("failed to unmarshal to ExternalApprovalPayloadFeishu", zap.String("payload", externalApproval.Payload), zap.Error(err))
							continue
						}

						if payload.Rejected {
							continue
						}

						issue, err := r.store.GetIssueByID(ctx, externalApproval.IssueID)
						if err != nil {
							log.Error("failed to get issue by issue id", zap.Int("issueID", externalApproval.IssueID), zap.Error(err))
							continue
						}
						stage := getActiveStage(issue.Pipeline.StageList)
						if stage == nil {
							stage = issue.Pipeline.StageList[len(issue.Pipeline.StageList)-1]
						}
						if issue.Status != api.IssueOpen {
							if err := r.CancelExternalApproval(ctx, issue.ID, externalApprovalCancelReasonIssueNotOpen); err != nil {
								log.Error("failed to cancel external approval", zap.Error(err))
							}
							continue
						}

						status, err := r.p.GetExternalApprovalStatus(ctx, feishu.TokenCtx{
							AppID:     value.AppID,
							AppSecret: value.AppSecret,
						}, payload.InstanceCode)
						if err != nil {
							if errors.Is(err, context.Canceled) {
								break
							}
							log.Error("failed to get external approval", zap.String("instanceCode", payload.InstanceCode), zap.Error(err))
							continue
						}

						switch status {
						case feishu.ApprovalStatusApproved:
							// double check
							if stage.ID == payload.StageID && payload.AssigneeID == issue.AssigneeID {
								// approve stage
								if err := func() error {
									var taskIDList []int
									var tasks []*api.Task
									for _, task := range stage.TaskList {
										if task.Status == api.TaskPendingApproval {
											taskIDList = append(taskIDList, task.ID)
											tasks = append(tasks, task)
										}
									}
									taskStatusPatch := &api.TaskStatusPatch{
										IDList:    taskIDList,
										UpdaterID: externalApproval.ApproverID,
										Status:    api.TaskPending,
									}
									if _, err := r.store.PatchTaskStatus(ctx, taskStatusPatch); err != nil {
										return errors.Wrapf(err, "failed to update task status, task id list: %+v", taskIDList)
									}
									if err := r.activityManager.BatchCreateTaskStatusUpdateApprovalActivity(ctx, taskStatusPatch, issue, stage, tasks); err != nil {
										return errors.Wrapf(err, "failed to create task status update activity")
									}
									return nil
								}(); err != nil {
									log.Error("failed to approve stage", zap.Error(err))
									continue
								}

								if _, err := r.store.PatchExternalApproval(ctx, &api.ExternalApprovalPatch{
									// Archive external approval.
									ID:        externalApproval.ID,
									RowStatus: api.Archived,
								}); err != nil {
									log.Error("failed to archive external apporval", zap.Error(err))
									continue
								}
							}
						case feishu.ApprovalStatusRejected:
							if err := func() error {
								payload := payload
								payload.Rejected = true
								bytes, err := json.Marshal(payload)
								if err != nil {
									return errors.Wrapf(err, "failed to marshal payload %+v", payload)
								}
								payloadString := string(bytes)

								if _, err := r.store.PatchExternalApproval(ctx, &api.ExternalApprovalPatch{
									ID:        externalApproval.ID,
									RowStatus: api.Normal,
									Payload:   &payloadString,
								}); err != nil {
									return errors.Wrap(err, "failed to patch external approval")
								}

								stageName := "UNKNOWN"
								for _, stage := range issue.Pipeline.StageList {
									if stage.ID == payload.StageID {
										stageName = stage.Name
										break
									}
								}
								activityPayload, err := json.Marshal(api.ActivityIssueCommentCreatePayload{
									ExternalApprovalEvent: &api.ExternalApprovalEvent{
										Type:      api.ExternalApprovalTypeFeishu,
										Action:    api.ExternalApprovalEventActionReject,
										StageName: stageName,
									},
								})
								if err != nil {
									return errors.Wrap(err, "failed to marshal ActivityIssueExternalApprovalRejectPayload")
								}

								activityCreate := &api.ActivityCreate{
									CreatorID:   payload.AssigneeID,
									ContainerID: issue.ID,
									Type:        api.ActivityIssueCommentCreate,
									Level:       api.ActivityInfo,
									Comment:     "",
									Payload:     string(activityPayload),
								}

								if _, err = r.activityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
									return errors.Wrap(err, "failed to create activity after external approval rejected")
								}
								return nil
							}(); err != nil {
								log.Error("failed to handle rejected feishu approval", zap.Error(err))
							}
						}
					default:
						log.Error("Unknown ExternalApproval.Type", zap.Any("ExternalApproval", externalApproval))
					}
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}

func (r *ApplicationRunner) cancelOldExternalApprovalIfNeeded(ctx context.Context, issue *api.Issue, stage *api.Stage, settingValue *api.SettingAppIMValue) (*api.ExternalApproval, error) {
	approval, err := r.store.GetExternalApprovalByIssueID(ctx, issue.ID)
	if err != nil {
		return nil, err
	}
	if approval == nil {
		return nil, nil
	}
	var payload api.ExternalApprovalPayloadFeishu
	if err := json.Unmarshal([]byte(approval.Payload), &payload); err != nil {
		return nil, err
	}

	reason := externalApprovalCancelReasonGeneral
	cancelOld := func() bool {
		if payload.StageID != stage.ID {
			reason = externalApprovalCancelReasonNoTaskPendingApproval
			return true
		}
		if payload.AssigneeID != issue.AssigneeID {
			reason = externalApprovalCancelReasonReassigned
			return true
		}
		pendingApprovalCount := 0
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				pendingApprovalCount++
			}
		}
		if pendingApprovalCount == 0 {
			reason = externalApprovalCancelReasonNoTaskPendingApproval
			return true
		}
		return false
	}()

	if cancelOld {
		if _, err := r.store.PatchExternalApproval(ctx, &api.ExternalApprovalPatch{ID: approval.ID, RowStatus: api.Archived}); err != nil {
			return nil, err
		}
		botID, err := r.p.GetBotID(ctx,
			feishu.TokenCtx{
				AppID:     settingValue.AppID,
				AppSecret: settingValue.AppSecret,
			})
		if err != nil {
			return nil, err
		}
		if err := r.p.CancelExternalApproval(ctx,
			feishu.TokenCtx{
				AppID:     settingValue.AppID,
				AppSecret: settingValue.AppSecret,
			},
			settingValue.ExternalApproval.ApprovalDefinitionID,
			payload.InstanceCode,
			botID,
		); err != nil {
			return nil, err
		}
		if err := r.p.CreateExternalApprovalComment(ctx,
			feishu.TokenCtx{
				AppID:     settingValue.AppID,
				AppSecret: settingValue.AppSecret,
			},
			payload.InstanceCode,
			botID,
			string(reason),
		); err != nil {
			return nil, err
		}
	}
	return approval, nil
}

// CancelExternalApproval cancels the active external approval of an issue.
func (r *ApplicationRunner) CancelExternalApproval(ctx context.Context, issueID int, reason string) error {
	settingName := api.SettingAppIM
	setting, err := r.store.GetSetting(ctx, &api.SettingFind{Name: &settingName})
	if err != nil {
		return errors.Wrapf(err, "failed to get IM setting by settingName %s", string(settingName))
	}
	if setting == nil {
		return errors.New("cannot find IM setting")
	}
	if setting.Value == "" {
		return nil
	}
	var value api.SettingAppIMValue
	if err := json.Unmarshal([]byte(setting.Value), &value); err != nil {
		return errors.Wrapf(err, "failed to unmarshal IM setting, settingName %s", string(settingName))
	}
	if !value.ExternalApproval.Enabled {
		return nil
	}
	approval, err := r.store.GetExternalApprovalByIssueID(ctx, issueID)
	if err != nil {
		return err
	}
	if approval == nil {
		return nil
	}
	var payload api.ExternalApprovalPayloadFeishu
	if err := json.Unmarshal([]byte(approval.Payload), &payload); err != nil {
		return err
	}
	if _, err := r.store.PatchExternalApproval(ctx, &api.ExternalApprovalPatch{ID: approval.ID, RowStatus: api.Archived}); err != nil {
		return err
	}
	botID, err := r.p.GetBotID(ctx,
		feishu.TokenCtx{
			AppID:     value.AppID,
			AppSecret: value.AppSecret,
		})
	if err != nil {
		return err
	}
	if err := r.p.CancelExternalApproval(ctx,
		feishu.TokenCtx{
			AppID:     value.AppID,
			AppSecret: value.AppSecret,
		},
		value.ExternalApproval.ApprovalDefinitionID,
		payload.InstanceCode,
		botID,
	); err != nil {
		return err
	}
	return r.p.CreateExternalApprovalComment(ctx,
		feishu.TokenCtx{
			AppID:     value.AppID,
			AppSecret: value.AppSecret,
		},
		payload.InstanceCode,
		botID,
		reason,
	)
}

func (r *ApplicationRunner) shouldCreateExternalApproval(ctx context.Context, issue *api.Issue, stage *api.Stage, oldApproval *api.ExternalApproval) (bool, error) {
	policy, err := r.store.GetPipelineApprovalPolicy(ctx, stage.EnvironmentID)
	if err != nil {
		return false, err
	}
	// don't send approvals for auto-approval stages.
	if policy.Value == api.PipelineApprovalValueManualNever {
		return false, nil
	}
	if oldApproval != nil {
		var oldPayload api.ExternalApprovalPayloadFeishu
		if err := json.Unmarshal([]byte(oldApproval.Payload), &oldPayload); err != nil {
			return false, err
		}
		// nothing changes
		if oldPayload.StageID == stage.ID && oldPayload.AssigneeID == issue.AssigneeID {
			return false, nil
		}
	}
	pendingApprovalCount := 0
	for _, task := range stage.TaskList {
		if task.Status == api.TaskPendingApproval {
			pendingApprovalCount++
		}

		// get the most recent task check run result for each type of task check
		taskCheckRun := make(map[api.TaskCheckType]*api.TaskCheckRun)
		for _, run := range task.TaskCheckRunList {
			v, ok := taskCheckRun[run.Type]
			if !ok {
				taskCheckRun[run.Type] = run
			} else if run.ID > v.ID {
				taskCheckRun[run.Type] = run
			}
		}

		for _, taskCheck := range taskCheckRun {
			if taskCheck.Status != api.TaskCheckRunDone {
				return false, nil
			}
			var payload api.TaskCheckRunResultPayload
			if err := json.Unmarshal([]byte(taskCheck.Result), &payload); err != nil {
				return false, err
			}
			for _, result := range payload.ResultList {
				if result.Status == api.TaskCheckStatusError {
					return false, nil
				}
			}
		}
	}
	if pendingApprovalCount == 0 {
		return false, nil
	}
	return true, nil
}

func (r *ApplicationRunner) createExternalApproval(ctx context.Context, issue *api.Issue, stage *api.Stage, settingValue *api.SettingAppIMValue) error {
	users, err := r.p.GetIDByEmail(ctx,
		feishu.TokenCtx{
			AppID:     settingValue.AppID,
			AppSecret: settingValue.AppSecret,
		},
		[]string{issue.Creator.Email, issue.Assignee.Email})
	if err != nil {
		return err
	}

	var taskList []feishu.Task
	for _, task := range stage.TaskList {
		taskList = append(taskList, feishu.Task{
			Name:   task.Name,
			Status: string(task.Status),
		})
	}

	instanceCode, err := r.p.CreateExternalApproval(ctx,
		feishu.TokenCtx{
			AppID:     settingValue.AppID,
			AppSecret: settingValue.AppSecret,
		},
		feishu.Content{
			Issue:    issue.Name,
			Stage:    stage.Name,
			Link:     fmt.Sprintf("%s/issue/%s", r.activityManager.s.profile.ExternalURL, api.IssueSlug(issue)),
			TaskList: taskList,
		},
		settingValue.ExternalApproval.ApprovalDefinitionID,
		users[issue.Creator.Email],
		users[issue.Assignee.Email])
	if err != nil {
		return err
	}
	payload := api.ExternalApprovalPayloadFeishu{
		StageID:      stage.ID,
		AssigneeID:   issue.AssigneeID,
		InstanceCode: instanceCode,
		RequesterID:  users[issue.Creator.Email],
		Rejected:     false,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := r.store.CreateExternalApproval(ctx, &api.ExternalApprovalCreate{
		IssueID:     issue.ID,
		ApproverID:  issue.AssigneeID,
		RequesterID: issue.CreatorID,
		Type:        api.ExternalApprovalTypeFeishu,
		Payload:     string(b),
	}); err != nil {
		return err
	}
	return nil
}

// ScheduleApproval tries to cancel old external apporvals and create new external approvals if needed.
func (r *ApplicationRunner) ScheduleApproval(ctx context.Context, pipeline *api.Pipeline) {
	settingName := api.SettingAppIM
	setting, err := r.store.GetSetting(ctx, &api.SettingFind{Name: &settingName})
	if err != nil {
		log.Error("failed to get IM setting", zap.String("settingName", string(settingName)), zap.Error(err))
		return
	}
	if setting == nil {
		log.Error("cannot find IM setting")
		return
	}
	if setting.Value == "" {
		return
	}
	var settingValue api.SettingAppIMValue
	if err := json.Unmarshal([]byte(setting.Value), &settingValue); err != nil {
		log.Error("failed to unmarshal IM setting", zap.String("settingName", string(settingName)), zap.Error(err))
		return
	}

	if !settingValue.ExternalApproval.Enabled {
		return
	}

	if settingValue.ExternalApproval.ApprovalDefinitionID == "" {
		log.Error("no approval code", zap.Any("settingValue", settingValue))
		return
	}

	find := &api.IssueFind{
		PipelineID: &pipeline.ID,
		StatusList: []api.IssueStatus{api.IssueOpen},
	}
	issues, err := r.store.FindIssueStripped(ctx, find)
	if err != nil {
		log.Error("failed to find issues", zap.Any("issueFind", find), zap.Error(err))
		return
	}
	if len(issues) > 1 {
		log.Error("expect 0 or 1 issue, get more than 1 issues", zap.Any("issues", issues))
		return
	}
	if len(issues) == 0 {
		// no containing issue, skip
		return
	}
	issue := issues[0]
	stage := getActiveStage(pipeline.StageList)
	if stage == nil {
		stage = pipeline.StageList[len(pipeline.StageList)-1]
	}

	oldApproval, err := r.cancelOldExternalApprovalIfNeeded(ctx, issue, stage, &settingValue)
	if err != nil {
		log.Error("failed to cancelOldExternalApprovalIfNeeded", zap.Error(err))
		return
	}

	// createExternalApprovalIfNeeded
	// check if we need to create a new external approval
	// 1. the approval policy of the stage environment is MANUAL_APPROVAL_ALWAYS.
	// 2. the stage has one or more PENDING_APPROVAL tasks.
	// 3. all task checks of the stage are done and the results have no errors.
	ok, err := r.shouldCreateExternalApproval(ctx, issue, stage, oldApproval)
	if err != nil {
		log.Error("failed to check shouldCreateExternalApproval", zap.Error(err))
		return
	}
	if !ok {
		return
	}

	if err := r.createExternalApproval(ctx, issue, stage, &settingValue); err != nil {
		log.Error("failed to create external approval", zap.Error(err))
		return
	}
}

// tryUpdateApprovalDefinition is run on application runner start.
// The approval definition may have changed so we make idempotent POST request to patch the definition.
func (r *ApplicationRunner) tryUpdateApprovalDefinition(ctx context.Context) error {
	settingName := api.SettingAppIM
	setting, err := r.store.GetSetting(ctx, &api.SettingFind{Name: &settingName})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			return errors.Wrapf(err, "failed to get IM setting")
		}
		return nil
	}
	if setting == nil {
		return errors.New("cannot find IM setting")
	}
	if setting.Value == "" {
		return nil
	}
	var value api.SettingAppIMValue
	if err := json.Unmarshal([]byte(setting.Value), &value); err != nil {
		return errors.Wrapf(err, "failed to unmarshal setting value %+v", setting.Value)
	}
	if !value.ExternalApproval.Enabled {
		return nil
	}
	// pass in ApprovalDefinitionID so that this would be a PATCH.
	if _, err := r.p.CreateApprovalDefinition(ctx, feishu.TokenCtx{
		AppID:     value.AppID,
		AppSecret: value.AppSecret,
	}, value.ExternalApproval.ApprovalDefinitionID); err != nil {
		return errors.Wrap(err, "failed to update approval definition")
	}
	return nil
}
