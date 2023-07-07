// Package relay is the runner for the relay plugin.
package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/utils"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	relayplugin "github.com/bytebase/bytebase/backend/plugin/app/relay"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewRunner creates a new runner instance.
func NewRunner(store *store.Store, activityManager *activity.Manager, taskScheduler *taskrun.Scheduler, stateCfg *state.State) *Runner {
	return &Runner{
		store:                     store,
		activityManager:           activityManager,
		taskScheduler:             taskScheduler,
		stateCfg:                  stateCfg,
		Client:                    relayplugin.NewClient(),
		CheckExternalApprovalChan: make(chan CheckExternalApprovalChanMessage, 100),
	}
}

// CheckExternalApprovalChanMessage is the message to check external approval status.
type CheckExternalApprovalChanMessage struct {
	ExternalApproval *store.ExternalApprovalMessage
	// ErrChan is used to send back the error message.
	// The channel must be buffered to avoid blocking.
	ErrChan chan error
}

// Runner is the runner for the relay.
type Runner struct {
	store           *store.Store
	activityManager *activity.Manager
	taskScheduler   *taskrun.Scheduler
	stateCfg        *state.State

	Client *relayplugin.Client

	CheckExternalApprovalChan chan CheckExternalApprovalChanMessage
}

const relayRunnerInterval = time.Minute * 10

// Run runs the runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(relayRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Relay runner started and will run every %v", relayRunnerInterval))

	wg.Add(1)
	go r.listenIssueExternalApprovalRelayCancelChan(ctx, wg)
	wg.Add(1)
	go r.listenCheckExternalApprovalChan(ctx, wg)

	for {
		select {
		case <-ticker.C:
			err := func() error {
				externalApprovalType := api.ExternalApprovalTypeRelay
				approvals, err := r.store.ListExternalApprovalV2(ctx, &store.ListExternalApprovalMessage{
					Type: &externalApprovalType,
				})
				if err != nil {
					return err
				}
				var errs error
				for _, approval := range approvals {
					msg := CheckExternalApprovalChanMessage{
						ExternalApproval: approval,
						ErrChan:          make(chan error, 1),
					}
					r.CheckExternalApprovalChan <- msg
					err := <-msg.ErrChan
					if err != nil {
						err = errors.Wrapf(err, "failed to check external approval status, issueUID %d", approval.IssueUID)
						errs = multierr.Append(errs, err)
					}
				}
				return errs
			}()
			if err != nil {
				log.Error("relay runner: failed to check external approval", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) listenCheckExternalApprovalChan(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case msg := <-r.CheckExternalApprovalChan:
			err := r.checkExternalApproval(ctx, msg.ExternalApproval)
			msg.ErrChan <- err
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) checkExternalApproval(ctx context.Context, approval *store.ExternalApprovalMessage) error {
	payload := &api.ExternalApprovalPayloadRelay{}
	if err := json.Unmarshal([]byte(approval.Payload), payload); err != nil {
		return errors.Wrapf(err, "failed to unmarshal external approval payload")
	}
	node, err := getExternalApprovalByID(ctx, r.store, payload.ExternalApprovalNodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to get external approval node %s", payload.ExternalApprovalNodeID)
	}
	id := payload.ID
	resp, err := r.Client.GetApproval(node.Endpoint, id)
	if err != nil {
		return errors.Wrapf(err, "failed to get external approval status, id: %v, endpoint: %s, id: %s", node.Id, node.Endpoint, id)
	}
	if resp.Status == relayplugin.StatusApproved {
		if err := r.approveExternalApprovalNode(ctx, approval.IssueUID); err != nil {
			return err
		}
		if _, err := r.store.UpdateExternalApprovalV2(ctx, &store.UpdateExternalApprovalMessage{
			ID:        approval.ID,
			RowStatus: api.Archived,
		}); err != nil {
			return err
		}
	} else if resp.Status == relayplugin.StatusRejected {
		if err := r.rejectExternalApprovalNode(ctx, approval.IssueUID); err != nil {
			return err
		}
		if _, err := r.store.UpdateExternalApprovalV2(ctx, &store.UpdateExternalApprovalMessage{
			ID:        approval.ID,
			RowStatus: api.Archived,
		}); err != nil {
			return err
		}
	}
	return nil
}

func getExternalApprovalByID(ctx context.Context, s *store.Store, externalApprovalID string) (*storepb.ExternalApprovalSetting_Node, error) {
	setting, err := s.GetWorkspaceExternalApprovalSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace external approval setting")
	}
	for _, node := range setting.Nodes {
		if node.Id == externalApprovalID {
			return node, nil
		}
	}
	return nil, nil
}

func (r *Runner) cancelExternalApproval(ctx context.Context, issueUID int) {
	approvals, err := r.store.ListExternalApprovalV2(ctx, &store.ListExternalApprovalMessage{
		IssueUID: &issueUID,
	})
	if err != nil {
		log.Error("failed to list external approvals", zap.Error(err))
		return
	}
	if len(approvals) == 0 {
		return
	}
	for _, approval := range approvals {
		if _, err := r.store.UpdateExternalApprovalV2(ctx, &store.UpdateExternalApprovalMessage{
			ID:        approval.ID,
			RowStatus: api.Archived,
		}); err != nil {
			log.Error("failed to archive external approval", zap.Error(err))
			continue
		}
		payload := &api.ExternalApprovalPayloadRelay{}
		if err := json.Unmarshal([]byte(approval.Payload), payload); err != nil {
			log.Error("failed to unmarshal external approval payload", zap.Error(err))
			continue
		}
		// don't wait for http requests, just fire and forget
		node, err := getExternalApprovalByID(ctx, r.store, payload.ExternalApprovalNodeID)
		if err != nil {
			log.Error("failed to get external approval node", zap.Error(err))
			continue
		}
		if node == nil {
			log.Error("external approval node not found", zap.String("id", payload.ExternalApprovalNodeID))
			continue
		}
		go func() {
			if err := r.Client.UpdateApproval(node.Endpoint, payload.ID, &relayplugin.UpdatePayload{}); err != nil {
				log.Error("failed to update external approval status", zap.String("endpoint", node.Endpoint), zap.String("id", payload.ID), zap.Error(err))
			}
		}()
	}
}

func (r *Runner) approveExternalApprovalNode(ctx context.Context, issueUID int) error {
	issue, err := r.store.GetIssueV2(ctx, &store.FindIssueMessage{
		UID: &issueUID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to get issue")
	}
	payload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
		return errors.Wrapf(err, "failed to unmarshal issue payload")
	}
	if payload.Approval == nil {
		return errors.Wrapf(err, "issue payload approval is nil")
	}
	if !payload.Approval.ApprovalFindingDone {
		return errors.Wrapf(err, "approval template finding is not done")
	}
	if payload.Approval.ApprovalFindingError != "" {
		return errors.Wrapf(err, "approval template finding failed: %v", payload.Approval.ApprovalFindingError)
	}
	if len(payload.Approval.ApprovalTemplates) != 1 {
		return errors.Wrapf(err, "expecting one approval template but got %v", len(payload.Approval.ApprovalTemplates))
	}

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep != nil {
		return errors.Wrapf(err, "cannot approve because the review has been rejected")
	}

	step := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if step == nil {
		return errors.Wrapf(err, "cannot approve because the review has been approved")
	}
	if len(step.Nodes) != 1 {
		return errors.Wrapf(err, "expecting one node but got %v", len(step.Nodes))
	}

	node := step.Nodes[0]
	_, ok := node.Payload.(*storepb.ApprovalNode_ExternalNodeId)
	if !ok {
		return errors.Errorf("expecting external node id type but got %T", node.Payload)
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, &storepb.IssuePayloadApproval_Approver{
		Status:      storepb.IssuePayloadApproval_Approver_APPROVED,
		PrincipalId: api.SystemBotID,
	})

	approved, err := utils.CheckApprovalApproved(payload.Approval)
	if err != nil {
		return errors.Wrapf(err, "failed to check if the approval is approved")
	}

	newApprovers, activityCreates, err := utils.HandleIncomingApprovalSteps(ctx, r.store, r.Client, issue, payload.Approval)
	if err != nil {
		return errors.Wrapf(err, "failed to handle incoming approval steps")
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, newApprovers...)
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal issue payload")
	}
	payloadStr := string(payloadBytes)

	issue, err = r.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		Payload: &payloadStr,
	}, api.SystemBotID)
	if err != nil {
		return errors.Wrapf(err, "failed to update issue")
	}

	// It's ok to fail to create activity.
	if err := func() error {
		activityPayload, err := protojson.Marshal(&storepb.ActivityIssueCommentCreatePayload{
			Event: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_{
				ApprovalEvent: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent{
					Status: storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_APPROVED,
				},
			},
			IssueName: issue.Title,
		})
		if err != nil {
			return err
		}
		create := &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueCommentCreate,
			Level:        api.ActivityInfo,
			Comment:      "",
			Payload:      string(activityPayload),
		}
		if _, err := r.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
			return err
		}

		for _, create := range activityCreates {
			if _, err := r.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
				return err
			}
		}

		return nil
	}(); err != nil {
		log.Error("failed to create skipping steps activity after approving review", zap.Error(err))
	}

	if err := func() error {
		if len(payload.Approval.ApprovalTemplates) != 1 {
			return nil
		}
		approvalStep := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
		if approvalStep == nil {
			return nil
		}
		protoPayload, err := protojson.Marshal(&storepb.ActivityIssueApprovalNotifyPayload{
			ApprovalStep: approvalStep,
		})
		if err != nil {
			return err
		}
		activityPayload, err := json.Marshal(api.ActivityIssueApprovalNotifyPayload{
			ProtoPayload: string(protoPayload),
		})
		if err != nil {
			return err
		}

		create := &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueApprovalNotify,
			Level:        api.ActivityInfo,
			Comment:      "",
			Payload:      string(activityPayload),
		}
		if _, err := r.activityManager.CreateActivity(ctx, create, &activity.Metadata{Issue: issue}); err != nil {
			return err
		}

		return nil
	}(); err != nil {
		log.Error("failed to create approval step pending activity after creating review", zap.Error(err))
	}

	// Grant the privilege if the issue is approved.
	if approved && issue.Type == api.IssueGrantRequest {
		policy, err := r.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &issue.Project.ResourceID})
		if err != nil {
			return err
		}
		var newConditionExpr string
		if payload.GrantRequest.Condition != nil {
			newConditionExpr = payload.GrantRequest.Condition.Expression
		}
		updated := false

		userID, err := strconv.Atoi(strings.TrimPrefix(payload.GrantRequest.User, "users/"))
		if err != nil {
			return err
		}
		newUser, err := r.store.GetUserByID(ctx, userID)
		if err != nil {
			return err
		}
		if newUser == nil {
			return errors.Errorf("user %v not found", userID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Role(payload.GrantRequest.Role) {
				continue
			}
			var oldConditionExpr string
			if binding.Condition != nil {
				oldConditionExpr = binding.Condition.Expression
			}
			if oldConditionExpr != newConditionExpr {
				continue
			}
			// Append
			binding.Members = append(binding.Members, newUser)
			updated = true
			break
		}
		role := api.Role(strings.TrimPrefix(payload.GrantRequest.Role, "roles/"))
		if !updated {
			condition := payload.GrantRequest.Condition
			condition.Description = fmt.Sprintf("#%d", issue.UID)
			policy.Bindings = append(policy.Bindings, &store.PolicyBinding{
				Role:      role,
				Members:   []*store.UserMessage{newUser},
				Condition: condition,
			})
		}
		if _, err := r.store.SetProjectIAMPolicy(ctx, policy, api.SystemBotID, issue.Project.UID); err != nil {
			return err
		}
		// Post project IAM policy update activity.
		if _, err := r.activityManager.CreateActivity(ctx, &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.Project.UID,
			Type:         api.ActivityProjectMemberCreate,
			Level:        api.ActivityInfo,
			Comment:      fmt.Sprintf("Granted %s to %s (%s).", newUser.Name, newUser.Email, role),
		}, &activity.Metadata{}); err != nil {
			log.Warn("Failed to create project activity", zap.Error(err))
		}
	}
	if issue.Type == api.IssueGrantRequest {
		if err := func() error {
			payload := &storepb.IssuePayload{}
			if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
				return errors.Wrap(err, "failed to unmarshal issue payload")
			}
			approved, err := utils.CheckApprovalApproved(payload.Approval)
			if err != nil {
				return errors.Wrap(err, "failed to check if the approval is approved")
			}
			if approved {
				if err := r.taskScheduler.ChangeIssueStatus(ctx, issue, api.IssueDone, api.SystemBotID, ""); err != nil {
					return errors.Wrap(err, "failed to update issue status")
				}
			}
			return nil
		}(); err != nil {
			log.Debug("failed to update issue status to done if grant request issue is approved", zap.Error(err))
		}
	}

	return nil
}

func (r *Runner) rejectExternalApprovalNode(ctx context.Context, issueUID int) error {
	issue, err := r.store.GetIssueV2(ctx, &store.FindIssueMessage{
		UID: &issueUID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to get issue")
	}
	payload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
		return errors.Wrapf(err, "failed to unmarshal issue payload")
	}
	if payload.Approval == nil {
		return errors.Wrapf(err, "issue payload approval is nil")
	}
	if !payload.Approval.ApprovalFindingDone {
		return errors.Wrapf(err, "approval template finding is not done")
	}
	if payload.Approval.ApprovalFindingError != "" {
		return errors.Wrapf(err, "approval template finding failed: %v", payload.Approval.ApprovalFindingError)
	}
	if len(payload.Approval.ApprovalTemplates) != 1 {
		return errors.Wrapf(err, "expecting one approval template but got %v", len(payload.Approval.ApprovalTemplates))
	}

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep != nil {
		return errors.Wrapf(err, "cannot reject because the review has been rejected")
	}

	step := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if step == nil {
		return errors.Wrapf(err, "cannot reject because the review has been approved")
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, &storepb.IssuePayloadApproval_Approver{
		Status:      storepb.IssuePayloadApproval_Approver_REJECTED,
		PrincipalId: int32(api.SystemBotID),
	})

	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal issue payload")
	}
	payloadStr := string(payloadBytes)

	issue, err = r.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		Payload: &payloadStr,
	}, api.SystemBotID)
	if err != nil {
		return errors.Wrapf(err, "failed to update issue")
	}

	// It's ok to fail to create activity.
	if err := func() error {
		activityPayload, err := protojson.Marshal(&storepb.ActivityIssueCommentCreatePayload{
			Event: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_{
				ApprovalEvent: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent{
					Status: storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_REJECTED,
				},
			},
			IssueName: issue.Title,
		})
		if err != nil {
			return err
		}
		create := &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueCommentCreate,
			Level:        api.ActivityInfo,
			Comment:      "",
			Payload:      string(activityPayload),
		}
		if _, err := r.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		log.Error("failed to create activity after rejecting review", zap.Error(err))
	}

	return nil
}

func (r *Runner) listenIssueExternalApprovalRelayCancelChan(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case issueUID := <-r.stateCfg.IssueExternalApprovalRelayCancelChan:
			r.cancelExternalApproval(ctx, issueUID)
		case <-ctx.Done():
			return
		}
	}
}
