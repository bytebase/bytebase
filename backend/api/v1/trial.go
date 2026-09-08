package v1

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/enterprise"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	trialDuration  = 14 * 24 * time.Hour
	trialSeats     = 20
	trialInstances = 10
)

func newTrialLicenseParams(workspaceID string, startedAt time.Time) *enterprise.LicenseParams {
	startedAt = startedAt.UTC().Truncate(time.Second)
	return &enterprise.LicenseParams{
		Plan:        v1pb.PlanType_TEAM.String(),
		Seats:       trialSeats,
		Instances:   trialInstances,
		WorkspaceID: workspaceID,
		Trialing:    true,
		ExpiresAt:   startedAt.Add(trialDuration),
	}
}

func subscriptionFromTrialParams(params *enterprise.LicenseParams) *v1pb.Subscription {
	return &v1pb.Subscription{
		Plan:            v1pb.PlanType_TEAM,
		Seats:           int32(params.Seats),
		Instances:       int32(params.Instances),
		ActiveInstances: int32(params.Instances),
		ExpiresTime:     timestamppb.New(params.ExpiresAt),
		Trialing:        true,
	}
}
