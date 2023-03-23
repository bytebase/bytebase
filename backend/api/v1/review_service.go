package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ReviewService implements the review service.
type ReviewService struct {
	v1pb.UnimplementedReviewServiceServer
	store *store.Store
}

// NewReviewService creates a new ReviewService.
func NewReviewService(store *store.Store) *ReviewService {
	return &ReviewService{
		store: store,
	}
}

// GetReview gets a review.
func (s *ReviewService) GetReview(ctx context.Context, request *v1pb.GetReviewRequest) (*v1pb.Review, error) {
	reviewID, err := getReviewID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &reviewID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get review, error: %v", err)
	}
	var issuePayload storepb.IssuePayload
	if err := protojson.Unmarshal([]byte(issue.Payload), &issuePayload); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal issue payload, error: %v", err)
	}
	return &v1pb.Review{
		ApprovalTemplates: issuePayload.Approval.ApprovalTemplates,
	}
	return nil, status.Errorf(codes.Unimplemented, "method GetReview not implemented")
}

// ApproveReview approves the approval flow of the review.
func (*ReviewService) ApproveReview(context.Context, *v1pb.ApproveReviewRequest) (*v1pb.Review, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ApproveReview not implemented")
}

func convertToApprovalTemplates(templates []*storepb.ApprovalTemplate) []*v1pb.ApprovalTemplate {
}

func convertToApprovalTemplate(template *storepb.ApprovalTemplate) *v1pb.ApprovalTemplate {

}

func convertToApprovalFlow(flow *storepb.ApprovalFlow) *v1pb.ApprovalFlow {
	convertedFlow := &v1pb.ApprovalFlow{}
	for _, step := range flow.Steps {
		convertedFlow.Steps = append(convertedFlow.Steps, convertToApprovalStep(step))
	}
	return convertedFlow
}

func convertToApprovalStep(step *storepb.ApprovalStep) *v1pb.ApprovalStep {
	convertedStep := &v1pb.ApprovalStep{
		Type: v1pb.ApprovalStep_Type(step.Type),
	}
	for _, node := range step.Nodes {
		convertedStep.Nodes = append(convertedStep.Nodes, convertToApprovalNode(node))
	}
	return convertedStep
}

func convertToApprovalNode(node *storepb.ApprovalNode) *v1pb.ApprovalNode {
	return &v1pb.ApprovalNode{
		Type: v1pb.ApprovalNode_Type(node.Type),
		Payload: node.Payload,
	}
}

func convertToReview(issue *store.IssueMessage, issuePayload *storepb.IssuePayload) *v1pb.Review {
	review := &v1pb.Review{}
	review.ApprovalTemplates = 
}
