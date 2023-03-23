package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/store"
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

	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{})
	return nil, status.Errorf(codes.Unimplemented, "method GetReview not implemented")
}

// ApproveReview approves the approval flow of the review.
func (*ReviewService) ApproveReview(context.Context, *v1pb.ApproveReviewRequest) (*v1pb.Review, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ApproveReview not implemented")
}
