package store

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// ComputeApprovalStatus derives an issue's approval status from its payload.
// approvalStatusExpr is the SQL derivation of the same rule, which the
// approval_status list filter runs; TestIssueApprovalStatusFilterMatchesComputation
// keeps the two in step.
func ComputeApprovalStatus(approval *storepb.IssuePayloadApproval) v1pb.ApprovalStatus {
	// If approval finding is not done, status is checking.
	// approval.GetApprovalFindingDone() returns false when approval is nil.
	if !approval.GetApprovalFindingDone() {
		return v1pb.ApprovalStatus_CHECKING
	}

	// If no approval template, approval is skipped (not required).
	if approval.GetApprovalTemplate() == nil {
		return v1pb.ApprovalStatus_SKIPPED
	}

	approvers := approval.GetApprovers()
	totalSteps := len(approval.GetApprovalTemplate().GetFlow().GetRoles())

	// If no approvers are assigned yet, it's pending.
	if len(approvers) == 0 {
		return v1pb.ApprovalStatus_PENDING
	}

	// Short-circuit: if any approver rejected, overall status is rejected.
	for _, approver := range approvers {
		if approver.GetStatus() == storepb.IssuePayloadApproval_Approver_REJECTED {
			return v1pb.ApprovalStatus_REJECTED
		}
	}

	// Each approver corresponds to one step in the approval flow, so every
	// step is approved once there are as many approvers as steps and all of
	// them approved.
	if len(approvers) >= totalSteps {
		allApproved := true
		for _, approver := range approvers {
			if approver.GetStatus() != storepb.IssuePayloadApproval_Approver_APPROVED {
				allApproved = false
				break
			}
		}
		if allApproved {
			return v1pb.ApprovalStatus_APPROVED
		}
	}

	// Otherwise, approval is pending (more steps to complete or waiting for approvals).
	return v1pb.ApprovalStatus_PENDING
}
