// relay is the runner for the relay plugin.
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func NewRunner(store *store.Store, activityManager *activity.Manager, taskScheduler *taskrun.Scheduler, stateCfg *state.State) *Runner {
	return &Runner{
		store:           store,
		activityManager: activityManager,
		taskScheduler:   taskScheduler,
		stateCfg:        stateCfg,
		Client:          relayplugin.NewClient(),
	}
}

type Runner struct {
	store           *store.Store
	activityManager *activity.Manager
	taskScheduler   *taskrun.Scheduler
	stateCfg        *state.State

	Client *relayplugin.Client
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

func (r *Runner) CancelExternalApproval(ctx context.Context, issueUID int) {
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
			if err := r.Client.UpdateStatus(node.Endpoint, payload.URI); err != nil {
				log.Error("failed to update external approval status", zap.String("endpoint", node.Endpoint), zap.String("uri", payload.URI), zap.Error(err))
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
	if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal issue payload, error: %v", err)
	}
	if payload.Approval == nil {
		return status.Errorf(codes.Internal, "issue payload approval is nil")
	}
	if !payload.Approval.ApprovalFindingDone {
		return status.Errorf(codes.FailedPrecondition, "approval template finding is not done")
	}
	if payload.Approval.ApprovalFindingError != "" {
		return status.Errorf(codes.FailedPrecondition, "approval template finding failed: %v", payload.Approval.ApprovalFindingError)
	}
	if len(payload.Approval.ApprovalTemplates) != 1 {
		return status.Errorf(codes.Internal, "expecting one approval template but got %v", len(payload.Approval.ApprovalTemplates))
	}

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep != nil {
		return status.Errorf(codes.InvalidArgument, "cannot approve because the review has been rejected")
	}

	step := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if step == nil {
		return status.Errorf(codes.InvalidArgument, "the review has been approved")
	}
	if len(step.Nodes) != 1 {
		return status.Errorf(codes.Internal, "expecting one node but got %v", len(step.Nodes))
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
		return status.Errorf(codes.Internal, "failed to check if the approval is approved, error: %v", err)
	}

	newApprovers, activityCreates, err := utils.HandleIncomingApprovalSteps(ctx, r.store, r.Client, issue, payload.Approval)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to handle incoming approval steps, error: %v", err)
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, newApprovers...)
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to marshal issue payload, error: %v", err)
	}
	payloadStr := string(payloadBytes)

	issue, err = r.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		Payload: &payloadStr,
	}, api.SystemBotID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to update issue, error: %v", err)
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
		create := &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: issue.UID,
			Type:        api.ActivityIssueCommentCreate,
			Level:       api.ActivityInfo,
			Comment:     "",
			Payload:     string(activityPayload),
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

		create := &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: issue.UID,
			Type:        api.ActivityIssueApprovalNotify,
			Level:       api.ActivityInfo,
			Comment:     "",
			Payload:     string(activityPayload),
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
			return status.Errorf(codes.Internal, "user %v not found", userID)
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
		if _, err := r.activityManager.CreateActivity(ctx, &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: issue.Project.UID,
			Type:        api.ActivityProjectMemberCreate,
			Level:       api.ActivityInfo,
			Comment:     fmt.Sprintf("Granted %s to %s (%s).", newUser.Name, newUser.Email, role),
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

func (r *Runner) CheckExternalApproval(ctx context.Context, approval *store.ExternalApprovalMessage) error {
	payload := &api.ExternalApprovalPayloadRelay{}
	if err := json.Unmarshal([]byte(approval.Payload), payload); err != nil {
		return errors.Wrapf(err, "failed to unmarshal external approval payload")
	}
	node, err := getExternalApprovalByID(ctx, r.store, payload.ExternalApprovalNodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to get external approval node %s", payload.ExternalApprovalNodeID)
	}
	uri := payload.URI
	done, err := r.Client.GetStatus(node.Endpoint, uri)
	if err != nil {
		return errors.Wrapf(err, "failed to get external approval status, id: %v, endpoint: %s, uri: %s", node.Id, node.Endpoint, uri)
	}
	if done {
		if err := r.approveExternalApprovalNode(ctx, approval.IssueUID); err != nil {
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

func (r *Runner) listenIssueExternalApprovalRelayCancelChan(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case issueUID := <-r.stateCfg.IssueExternalApprovalRelayCancelChan:
			r.CancelExternalApproval(ctx, issueUID)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer wg.Done()

	wg.Add(1)
	go r.listenIssueExternalApprovalRelayCancelChan(ctx, wg)

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
					if err := r.CheckExternalApproval(ctx, approval); err != nil {
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
