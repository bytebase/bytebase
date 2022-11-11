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
)

const applicationRunnerInterval = time.Duration(1) * time.Second

// NewApplicationRunner returns a ApplicationRunner.
func NewApplicationRunner(server *Server) *ApplicationRunner {
	return &ApplicationRunner{
		server: server,
		P:      feishu.NewProvider(),
	}
}

// ApplicationRunner is a runner which periodically checks external approval status and approve the correspoding stages.
type ApplicationRunner struct {
	server *Server
	P      *feishu.Provider
}

// Run runs the ApplicationRunner.
func (r *ApplicationRunner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Application runner started and will run every %v", applicationRunnerInterval))
	for {
		select {
		case <-ticker.C:
			func() {
				ctx := context.Background()
				settingName := api.SettingAppIM
				setting, err := r.server.store.GetSetting(ctx, &api.SettingFind{Name: &settingName})
				if err != nil {
					log.Error("failed to get setting by settingName", zap.String("settingName", string(settingName)), zap.Error(err))
					return
				}
				if setting == nil {
					return
				}
				var value api.SettingAppIMValue
				if err := json.Unmarshal([]byte(setting.Value), &value); err != nil {
					log.Error("failed to unmarshal", zap.Error(err))
					return
				}
				if value.ExternalApproval.Enabled == false {
					return
				}

				externalApprovalList, err := r.server.store.FindExternalApproval(ctx, &api.ExternalApprovalFind{})
				if err != nil {
					log.Error("failed to find approval instance list", zap.Error(err))
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

						issue, err := r.server.store.GetIssueByID(ctx, externalApproval.IssueID)
						stage := GetActiveStage(issue.Pipeline.StageList)
						if stage == nil {
							stage = issue.Pipeline.StageList[len(issue.Pipeline.StageList)-1]
						}
						if err != nil {
							log.Error("failed to get issue by issue id", zap.Int("issueID", externalApproval.IssueID), zap.Error(err))
							continue
						}
						if issue.Status != api.IssueOpen {
							if _, err := r.server.cancelOldExternalApprovalIfNeeded(ctx, issue, stage, &value, r.P); err != nil {
								log.Error("failed to cancel external approval", zap.Error(err))
								continue
							}
						}

						status, err := r.P.GetExternalApprovalStatus(ctx, feishu.TokenCtx{
							AppID:     value.AppID,
							AppSecret: value.AppSecret,
							Token:     r.P.Token.Load().(string),
						}, payload.InstanceCode)
						if err != nil {
							log.Error("failed to get approval instance", zap.String("instanceCode", payload.InstanceCode), zap.Error(err))
							continue
						}

						if status == "APPROVED" {
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
									_, err := r.server.store.PatchTaskStatus(ctx, taskStatusPatch)
									if err != nil {
										return errors.Wrapf(err, "Failed to update task %q status", taskIDList)
									}
									if err := r.server.ActivityManager.BatchCreateTaskStatusUpdateApprovalActivity(ctx, taskStatusPatch, issue, stage, tasks); err != nil {
										return errors.Wrapf(err, "failed to create task status update activity")
									}
									return nil
								}(); err != nil {
									log.Error("failed to approve stage", zap.Error(err))
								}

								if _, err := r.server.store.PatchExternalApproval(ctx, &api.ExternalApprovalPatch{
									// archive approval instance
									ID:        externalApproval.ID,
									RowStatus: api.Archived,
								}); err != nil {
									log.Error("failed to archive external apporval", zap.Error(err))
								}
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

func (s *Server) cancelOldExternalApprovalIfNeeded(ctx context.Context, issue *api.Issue, stage *api.Stage, settingValue *api.SettingAppIMValue, p *feishu.Provider) (*api.ExternalApproval, error) {
	approval, err := s.store.GetExternalApprovalByIssueID(ctx, issue.ID)
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

	cancelOld := func() bool {
		if payload.StageID != stage.ID {
			return true
		}
		if payload.AssigneeID != issue.AssigneeID {
			return true
		}
		pendingApprovalCount := 0
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				pendingApprovalCount++
			}
		}
		if pendingApprovalCount == 0 {
			return true
		}
		return false
	}()

	if cancelOld {
		_, err := s.store.PatchExternalApproval(ctx, &api.ExternalApprovalPatch{ID: approval.ID, RowStatus: api.Archived})
		if err != nil {
			return nil, err
		}
		if err := p.CancelExternalApproval(ctx,
			feishu.TokenCtx{
				AppID:     settingValue.AppID,
				AppSecret: settingValue.AppSecret,
				Token:     p.Token.Load().(string),
			},
			settingValue.ExternalApproval.ApprovalCode,
			payload.InstanceCode,
			payload.RequesterID,
		); err != nil {
			return nil, err
		}
	}
	return approval, nil
}

func shouldCreateExternalApproval(issue *api.Issue, stage *api.Stage, oldApproval *api.ExternalApproval) (bool, error) {
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
		for _, taskCheck := range task.TaskCheckRunList {
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

func createExternalApproval(ctx context.Context, s *Server, issue *api.Issue, stage *api.Stage, settingValue *api.SettingAppIMValue, p *feishu.Provider) error {
	users, err := p.GetIDByEmail(ctx,
		feishu.TokenCtx{
			AppID:     settingValue.AppID,
			AppSecret: settingValue.AppSecret,
			Token:     p.Token.Load().(string),
		},
		[]string{issue.Creator.Email, issue.Assignee.Email})
	log.Info("appp", zap.Any("users", users))
	if err != nil {
		log.Error("failed to get id by email", zap.Any("resp", users))
		return err
	}
	instanceCode, err := p.CreateExternalApproval(ctx,
		feishu.TokenCtx{
			AppID:     settingValue.AppID,
			AppSecret: settingValue.AppSecret,
			Token:     p.Token.Load().(string),
		},
		feishu.Content{
			Issue: issue.Name,
			Stage: stage.Name,
		},
		settingValue.ExternalApproval.ApprovalCode,
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
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.store.CreateExternalApproval(ctx, &api.ExternalApprovalCreate{
		IssueID:     issue.ID,
		ApproverID:  issue.AssigneeID,
		RequesterID: issue.CreatorID,
		Type:        api.ExternalApprovalTypeFeishu,
		Payload:     string(b),
	})
	if err != nil {
		return err
	}
	return nil
}

func scheduleApproval(s *Server, pipeline *api.Pipeline) {
	ctx := context.Background()
	settingName := api.SettingAppIM
	setting, err := s.store.GetSetting(ctx, &api.SettingFind{Name: &settingName})
	if err != nil {
		log.Error("failed to get setting by settingName", zap.String("settingName", string(settingName)), zap.Error(err))
		return
	}
	var settingValue api.SettingAppIMValue
	if err := json.Unmarshal([]byte(setting.Value), &settingValue); err != nil {
		log.Error("failed to unmarshal to settingValue", zap.Error(err))
		return
	}

	if settingValue.ExternalApproval.Enabled == false {
		return
	}

	if settingValue.ExternalApproval.ApprovalCode == "" {
		log.Error("no approval code", zap.Any("settingValue", settingValue))
		return
	}

	find := &api.IssueFind{
		PipelineID: &pipeline.ID,
		StatusList: []api.IssueStatus{api.IssueOpen},
	}
	issues, err := s.store.FindIssueStripped(ctx, find)
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
	stage := GetActiveStage(pipeline.StageList)
	if stage == nil {
		stage = pipeline.StageList[len(pipeline.StageList)-1]
	}

	p := feishu.NewProvider()
	oldApproval, err := s.cancelOldExternalApprovalIfNeeded(ctx, issue, stage, &settingValue, p)

	// createExternalApprovalIfNeeded
	// check if we need to create a new approval instance
	// 1. has one or more PENDING_APPROVAL tasks.
	// 2. all task checks are done and the results have no errors.
	ok, err := shouldCreateExternalApproval(issue, stage, oldApproval)
	if err != nil {
		log.Error("failed to check shouldCreateExternalApproval", zap.Error(err))
		return
	}
	if !ok {
		return
	}

	if err := createExternalApproval(ctx, s, issue, stage, &settingValue, p); err != nil {
		log.Error("failed to create approval instance", zap.Error(err))
		return
	}
}
