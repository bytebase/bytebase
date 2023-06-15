// relay is the runner for the relay plugin.
package relay

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/relay"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func NewRunner() *Runner {
	return &Runner{}
}

type Runner struct {
	store  *store.Store
	client *relay.Client
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

func (r *Runner) CreateExternalApproval(ctx context.Context, externalNodeID string) error {
	node, err := getExternalApprovalByID(ctx, r.store, externalNodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to get external approval node %s", externalNodeID)
	}
	if node == nil {
		return errors.Errorf("external approval node %s not found", externalNodeID)
	}
	uri, err := r.client.Create(node.Endpoint, relay.CreatePayload{})
	if err != nil {
		return errors.Wrapf(err, "failed to create external approval")
	}
	payload, err := json.Marshal(&api.ExternalApprovalPayloadRelay{
		URI: uri,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal external approval payload")
	}
	if _, err := r.store.CreateExternalApprovalV2(ctx, &store.ExternalApprovalMessage{
		IssueUID:     1,
		ApproverUID:  api.SystemBotID,
		Type:         api.ExternalApprovalTypeRelay,
		Payload:      string(payload),
		RequesterUID: api.SystemBotID,
	}); err != nil {
		return errors.Wrapf(err, "failed to create external approval")
	}
	return nil
}

func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer wg.Done()
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
				for _, approval := range approvals {
				}

				return nil
			}()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) doIssue(ctx context.Context, issue *store.IssueMessage) error {
	issuePayload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), issuePayload); err != nil {
		return errors.Wrapf(err, "failed to unmarshal issue payload")
	}
	approval := issuePayload.GetApproval()
	if approval == nil || approval.ApprovalFindingDone == false || approval.ApprovalFindingError != "" {
		return nil
	}
	if len(approval.ApprovalTemplates) != 1 {
		return errors.Errorf("expecting only 1 approval template, but got %d", len(approval.ApprovalTemplates))
	}
	approvalTemplate := approval.ApprovalTemplates[0]
	rejectedStep := utils.FindRejectedStep(approvalTemplate, approval.Approvers)
	if rejectedStep != nil {
		return nil
	}
	pendingStep := utils.FindNextPendingStep(approvalTemplate, approval.Approvers)

	return nil
}
