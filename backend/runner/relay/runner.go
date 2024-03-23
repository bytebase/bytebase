// Package relay is the runner for the relay plugin.
package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/utils"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	relayplugin "github.com/bytebase/bytebase/backend/plugin/app/relay"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewRunner creates a new runner instance.
func NewRunner(store *store.Store, activityManager *activity.Manager, stateCfg *state.State) *Runner {
	return &Runner{
		store:                     store,
		activityManager:           activityManager,
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
	slog.Debug(fmt.Sprintf("Relay runner started and will run every %v", relayRunnerInterval))

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
				slog.Error("relay runner: failed to check external approval", log.BBError(err))
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
		if err := r.ApproveExternalApprovalNode(ctx, approval.IssueUID); err != nil {
			return err
		}
		if _, err := r.store.UpdateExternalApprovalV2(ctx, &store.UpdateExternalApprovalMessage{
			ID:        approval.ID,
			RowStatus: api.Archived,
		}); err != nil {
			return err
		}
	} else if resp.Status == relayplugin.StatusRejected {
		if err := r.RejectExternalApprovalNode(ctx, approval.IssueUID); err != nil {
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
		slog.Error("failed to list external approvals", log.BBError(err))
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
			slog.Error("failed to archive external approval", log.BBError(err))
			continue
		}
		payload := &api.ExternalApprovalPayloadRelay{}
		if err := json.Unmarshal([]byte(approval.Payload), payload); err != nil {
			slog.Error("failed to unmarshal external approval payload", log.BBError(err))
			continue
		}
		// don't wait for http requests, just fire and forget
		node, err := getExternalApprovalByID(ctx, r.store, payload.ExternalApprovalNodeID)
		if err != nil {
			slog.Error("failed to get external approval node", log.BBError(err))
			continue
		}
		if node == nil {
			slog.Error("external approval node not found", slog.String("id", payload.ExternalApprovalNodeID))
			continue
		}
		go func() {
			if err := r.Client.UpdateApproval(node.Endpoint, payload.ID, &relayplugin.UpdatePayload{}); err != nil {
				slog.Error("failed to update external approval status", slog.String("endpoint", node.Endpoint), slog.String("id", payload.ID), log.BBError(err))
			}
		}()
	}
}

// ApproveExternalApprovalNode will approve the external approval node and update the issue.
func (r *Runner) ApproveExternalApprovalNode(ctx context.Context, issueUID int) error {
	issue, err := r.store.GetIssueV2(ctx, &store.FindIssueMessage{
		UID: &issueUID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to get issue")
	}
	payload := issue.Payload
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

	issue, err = r.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.IssuePayload{
			Approval: payload.Approval,
		},
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
			CreatorUID:        api.SystemBotID,
			ResourceContainer: issue.Project.GetName(),
			ContainerUID:      issue.UID,
			Type:              api.ActivityIssueCommentCreate,
			Level:             api.ActivityInfo,
			Comment:           "",
			Payload:           string(activityPayload),
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
		slog.Error("failed to create skipping steps activity after approving review", log.BBError(err))
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
			CreatorUID:        api.SystemBotID,
			ResourceContainer: issue.Project.GetName(),
			ContainerUID:      issue.UID,
			Type:              api.ActivityIssueApprovalNotify,
			Level:             api.ActivityInfo,
			Comment:           "",
			Payload:           string(activityPayload),
		}
		if _, err := r.activityManager.CreateActivity(ctx, create, &activity.Metadata{Issue: issue}); err != nil {
			return err
		}

		return nil
	}(); err != nil {
		slog.Error("failed to create approval step pending activity after creating review", log.BBError(err))
	}

	// Grant the privilege if the issue is approved.
	if approved && issue.Type == api.IssueGrantRequest {
		if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, r.store, r.activityManager, issue, payload.GrantRequest); err != nil {
			return errors.Wrapf(err, "failed to update project iam policy for grant request issue %q", issue.Title)
		}
	}
	if issue.Type == api.IssueGrantRequest {
		if err := func() error {
			approved, err := utils.CheckApprovalApproved(issue.Payload.Approval)
			if err != nil {
				return errors.Wrap(err, "failed to check if the approval is approved")
			}
			if approved {
				if err := utils.ChangeIssueStatus(ctx, r.store, r.activityManager, issue, api.IssueDone, api.SystemBotID, ""); err != nil {
					return errors.Wrap(err, "failed to update issue status")
				}
			}
			return nil
		}(); err != nil {
			slog.Debug("failed to update issue status to done if grant request issue is approved", log.BBError(err))
		}
	}

	return nil
}

// ApproveExternalApprovalNode will reject the external approval node and update the issue.
func (r *Runner) RejectExternalApprovalNode(ctx context.Context, issueUID int) error {
	issue, err := r.store.GetIssueV2(ctx, &store.FindIssueMessage{
		UID: &issueUID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to get issue")
	}
	payload := issue.Payload
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

	issue, err = r.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.IssuePayload{
			Approval: payload.Approval,
		},
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
			CreatorUID:        api.SystemBotID,
			ResourceContainer: issue.Project.GetName(),
			ContainerUID:      issue.UID,
			Type:              api.ActivityIssueCommentCreate,
			Level:             api.ActivityInfo,
			Comment:           "",
			Payload:           string(activityPayload),
		}
		if _, err := r.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		slog.Error("failed to create activity after rejecting review", log.BBError(err))
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
