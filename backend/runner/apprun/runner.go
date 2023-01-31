// Package apprun is an application runner for scanning Feishu approval instances.
package apprun

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/feishu"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewRunner returns a runner.
func NewRunner(store *store.Store, activityManager *activity.Manager, feishuProvider *feishu.Provider, profile config.Profile) *Runner {
	return &Runner{
		store:           store,
		activityManager: activityManager,
		p:               feishuProvider,
		profile:         profile,
	}
}

// Runner is a runner which periodically checks external approval status and approve the correspoding stages.
type Runner struct {
	store           *store.Store
	activityManager *activity.Manager
	p               *feishu.Provider
	profile         config.Profile
}

// Run runs the ApplicationRunner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(r.profile.AppRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Application runner started and will run every %v", r.profile.AppRunnerInterval))
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

				issueByID := make(map[int]*store.IssueMessage)
				stagesByPipelineID := make(map[int][]*store.StageMessage)

				needAttention := true
				issues, err := r.store.ListIssueV2(ctx, &store.FindIssueMessage{
					StatusList:    []api.IssueStatus{api.IssueOpen},
					NeedAttention: &needAttention,
				})
				if err != nil {
					log.Error("failed to find issues", zap.Error(err))
					return
				}
				for _, issue := range issues {
					issueByID[issue.UID] = issue
					stages, err := r.store.ListStageV2(ctx, issue.PipelineUID)
					if err != nil {
						log.Error("failed to list stages", zap.Int("pipeline", issue.PipelineUID), zap.Error(err))
						return
					}
					stagesByPipelineID[issue.PipelineUID] = stages
					r.scheduleApproval(ctx, issue, stages, &value)
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

						issue, ok := issueByID[externalApproval.IssueID]
						if !ok {
							log.Error("expect to have found issue in application runner", zap.Int("issue_id", externalApproval.IssueID))
							continue
						}
						stages, ok := stagesByPipelineID[issue.PipelineUID]
						if !ok {
							log.Error("expect to have found pipeline in application runner", zap.Int("pipeline_id", issue.PipelineUID))
							continue
						}
						activeStage := utils.GetActiveStageV2(stages)

						if issue.Status != api.IssueOpen || activeStage == nil {
							if err := r.CancelExternalApproval(ctx, issue.UID, api.ExternalApprovalCancelReasonIssueNotOpen); err != nil {
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
							if activeStage.ID == payload.StageID && payload.AssigneeID == issue.Assignee.ID {
								// Approve stage.
								if err := func() error {
									// TODO(d): optimize the active stage tasks.
									var taskIDList []int
									var tasks []*store.TaskMessage
									stageTasks, err := r.store.ListTasks(ctx, &api.TaskFind{PipelineID: &activeStage.PipelineID, StageID: &activeStage.ID})
									if err != nil {
										return err
									}
									for _, task := range stageTasks {
										if task.Status == api.TaskPendingApproval {
											taskIDList = append(taskIDList, task.ID)
											tasks = append(tasks, task)
										}
									}
									if err := r.store.BatchPatchTaskStatus(ctx, taskIDList, api.TaskPending, externalApproval.ApproverID); err != nil {
										return errors.Wrapf(err, "failed to update task status, task id list: %+v", taskIDList)
									}
									if err := r.activityManager.BatchCreateTaskStatusUpdateApprovalActivity(ctx, tasks, externalApproval.ApproverID, issue, activeStage.Name); err != nil {
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
								for _, stage := range stages {
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
									ContainerID: issue.UID,
									Type:        api.ActivityIssueCommentCreate,
									Level:       api.ActivityInfo,
									Comment:     "",
									Payload:     string(activityPayload),
								}

								if _, err = r.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
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

func (r *Runner) cancelOldExternalApprovalIfNeeded(ctx context.Context, issue *store.IssueMessage, stage *store.StageMessage, settingValue *api.SettingAppIMValue) (*api.ExternalApproval, error) {
	approval, err := r.store.GetExternalApprovalByIssueID(ctx, issue.UID)
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

	reason := api.ExternalApprovalCancelReasonGeneral
	cancelOld := func() bool {
		if payload.StageID != stage.ID {
			reason = api.ExternalApprovalCancelReasonNoTaskPendingApproval
			return true
		}
		if payload.AssigneeID != issue.Assignee.ID {
			reason = api.ExternalApprovalCancelReasonReassigned
			return true
		}
		pendingApprovalCount := 0
		tasks, err := r.store.ListTasks(ctx, &api.TaskFind{PipelineID: &stage.PipelineID, StageID: &stage.ID})
		if err != nil {
			return false
		}
		for _, task := range tasks {
			if task.Status == api.TaskPendingApproval {
				pendingApprovalCount++
			}
		}
		if pendingApprovalCount == 0 {
			reason = api.ExternalApprovalCancelReasonNoTaskPendingApproval
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
			payload.RequesterID,
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
func (r *Runner) CancelExternalApproval(ctx context.Context, issueID int, reason string) error {
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
		payload.RequesterID,
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

func (r *Runner) shouldCreateExternalApproval(ctx context.Context, issue *store.IssueMessage, stage *store.StageMessage, oldApproval *api.ExternalApproval) (bool, error) {
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
		if oldPayload.StageID == stage.ID && oldPayload.AssigneeID == issue.Assignee.ID {
			return false, nil
		}
	}
	pendingApprovalCount := 0
	tasks, err := r.store.ListTasks(ctx, &api.TaskFind{PipelineID: &stage.PipelineID, StageID: &stage.ID})
	if err != nil {
		return false, err
	}
	taskCheckRuns, err := r.store.ListTaskCheckRuns(ctx, &store.TaskCheckRunFind{PipelineID: &stage.PipelineID, StageID: &stage.ID})
	if err != nil {
		return false, err
	}

	for _, task := range tasks {
		if task.Status == api.TaskPendingApproval {
			pendingApprovalCount++
		}

		instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
		if err != nil {
			return false, err
		}
		ok, err := utils.PassAllCheck(task, api.TaskCheckStatusSuccess, taskCheckRuns, instance.Engine)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	if pendingApprovalCount == 0 {
		return false, nil
	}
	return true, nil
}

func (r *Runner) createExternalApproval(ctx context.Context, issue *store.IssueMessage, stage *store.StageMessage, settingValue *api.SettingAppIMValue) error {
	users, err := r.p.GetIDByEmail(ctx,
		feishu.TokenCtx{
			AppID:     settingValue.AppID,
			AppSecret: settingValue.AppSecret,
		},
		[]string{issue.Creator.Email, issue.Assignee.Email})
	if err != nil {
		return err
	}
	// assignee who approves the external approval is a must-have, so we returns error if we failed to find the user id.
	if _, ok := users[issue.Assignee.Email]; !ok {
		return errors.Errorf("failed to get user_id for issue assignee, email: %s", issue.Assignee.Email)
	}
	// if the creator is not found, the application bot will represent the creator.
	if _, ok := users[issue.Creator.Email]; !ok {
		botID, err := r.p.GetBotID(ctx, feishu.TokenCtx{
			AppID:     settingValue.AppID,
			AppSecret: settingValue.AppSecret,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		users[issue.Creator.Email] = botID
	}

	var taskList []feishu.Task
	tasks, err := r.store.ListTasks(ctx, &api.TaskFind{PipelineID: &stage.PipelineID, StageID: &stage.ID})
	if err != nil {
		return err
	}
	for _, task := range tasks {
		taskList = append(taskList, feishu.Task{
			Name:   task.Name,
			Status: string(task.Status),
		})
	}

	for i, task := range tasks {
		statement, err := utils.GetTaskStatement(task.Payload)
		if err != nil {
			return err
		}
		if statement == "" {
			continue
		}
		taskList[i].Statement = statement
	}

	instanceCode, err := r.p.CreateExternalApproval(ctx,
		feishu.TokenCtx{
			AppID:     settingValue.AppID,
			AppSecret: settingValue.AppSecret,
		},
		feishu.Content{
			Issue:    fmt.Sprintf("#%d %s", issue.UID, issue.Title),
			Stage:    stage.Name,
			Link:     fmt.Sprintf("%s/issue/%s-%d", r.profile.ExternalURL, slug.Make(issue.Title), issue.UID),
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
		AssigneeID:   issue.Assignee.ID,
		InstanceCode: instanceCode,
		RequesterID:  users[issue.Creator.Email],
		Rejected:     false,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := r.store.CreateExternalApproval(ctx, &api.ExternalApprovalCreate{
		IssueID:     issue.UID,
		ApproverID:  issue.Assignee.ID,
		RequesterID: issue.Creator.ID,
		Type:        api.ExternalApprovalTypeFeishu,
		Payload:     string(b),
	}); err != nil {
		return err
	}
	return nil
}

// scheduleApproval tries to cancel old external apporvals and create new external approvals if needed.
func (r *Runner) scheduleApproval(ctx context.Context, issue *store.IssueMessage, stages []*store.StageMessage, settingValue *api.SettingAppIMValue) {
	if !settingValue.ExternalApproval.Enabled {
		return
	}

	if settingValue.ExternalApproval.ApprovalDefinitionID == "" {
		log.Error("no approval code", zap.Any("settingValue", settingValue))
		return
	}

	activeStage := utils.GetActiveStageV2(stages)
	if activeStage == nil {
		return
	}

	oldApproval, err := r.cancelOldExternalApprovalIfNeeded(ctx, issue, activeStage, settingValue)
	if err != nil {
		log.Error("failed to cancelOldExternalApprovalIfNeeded", zap.Error(err))
		return
	}

	// createExternalApprovalIfNeeded
	// check if we need to create a new external approval
	// 1. the approval policy of the stage environment is MANUAL_APPROVAL_ALWAYS.
	// 2. the stage has one or more PENDING_APPROVAL tasks.
	// 3. all task checks of the stage are done and the results have no errors.
	ok, err := r.shouldCreateExternalApproval(ctx, issue, activeStage, oldApproval)
	if err != nil {
		log.Error("failed to check shouldCreateExternalApproval", zap.Error(err))
		return
	}
	if !ok {
		return
	}

	if err := r.createExternalApproval(ctx, issue, activeStage, settingValue); err != nil {
		log.Error("failed to create external approval", zap.Error(err))
		return
	}
}

// tryUpdateApprovalDefinition is run on application runner start.
// The approval definition may have changed so we make idempotent POST request to patch the definition.
func (r *Runner) tryUpdateApprovalDefinition(ctx context.Context) error {
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
