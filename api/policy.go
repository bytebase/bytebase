package api

import (
	"context"
	"fmt"
)

// PolicyType is the type or name of a policy.
type PolicyType string

// ApprovalPolicyValue is value for approval policy.
type ApprovalPolicyValue string

// BackupPlanPolicyValue is value for backup plan policy.
type BackupPlanPolicyValue string

const (
	// PolicyTypeApprovalPolicy is the approval policy type.
	PolicyTypeApprovalPolicy PolicyType = "approval_policy"
	// PolicyTypeBackupPlan is the backup plan policy type.
	PolicyTypeBackupPlan PolicyType = "backup_plan"

	// ApprovalPolicyValueManualNever is MANUAL_APPROVAL_NEVER approval policy value.
	ApprovalPolicyValueManualNever ApprovalPolicyValue = "MANUAL_APPROVAL_NEVER"
	// ApprovalPolicyValueManualAlways is MANUAL_APPROVAL_ALWAYS approval policy value.
	ApprovalPolicyValueManualAlways ApprovalPolicyValue = "MANUAL_APPROVAL_ALWAYS"

	// BackupPlanPolicyValueNever is NEVER backup plan policy value.
	BackupPlanPolicyValueNever BackupPlanPolicyValue = "NEVER"
	// BackupPlanPolicyValueDaily is DAILY backup plan policy value.
	BackupPlanPolicyValueDaily BackupPlanPolicyValue = "DAILY"
	// BackupPlanPolicyValueWeekly is WEEKLY backup plan policy value.
	BackupPlanPolicyValueWeekly BackupPlanPolicyValue = "WEEKLY"
)

var (
	// PolicyTypes is a set of all policy types.
	PolicyTypes = map[PolicyType]bool{
		PolicyTypeApprovalPolicy: true,
		PolicyTypeBackupPlan:     true,
	}
)

type Policy struct {
	ID int `jsonapi:"primary,environment"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	EnvironmentId int `jsonapi:"attr,environmentId"`
	// Do not return this to the client since the client always has the database context and fetching the
	// database object and all its own related objects is a bit expensive.
	Environment *Environment

	// Domain specific fields
	Type    PolicyType `jsonapi:"attr,type"`
	Payload string     `jsonapi:"attr,payload"`
}

// PolicyFind is the message to get a policy.
type PolicyFind struct {
	ID *int

	// Related fields
	EnvironmentId *int

	// Domain specific fields
	Type *PolicyType `jsonapi:"attr,type"`
}

// PolicyUpsert is the message to upsert a policy.
// NOTE: We use PATCH for Upsert, this is inspired by https://google.aip.dev/134#patch-and-put
type PolicyUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorId is the ID of the creator.
	UpdaterId int

	// Related fields
	EnvironmentId int

	// Domain specific fields
	Type    PolicyType `jsonapi:"attr,type"`
	Payload string     `jsonapi:"attr,payload"`
}

// PolicyService is the backend for policies.
type PolicyService interface {
	FindPolicy(ctx context.Context, find *PolicyFind) (*Policy, error)
	UpsertPolicy(ctx context.Context, upsert *PolicyUpsert) (*Policy, error)
}

// ValidatePolicy will validate the policy type and payload values.
func ValidatePolicy(pType PolicyType, payload string) error {
	if !PolicyTypes[pType] {
		return fmt.Errorf("invalid policy type: %s", pType)
	}
	if payload == "" {
		return nil
	}

	switch pType {
	case PolicyTypeApprovalPolicy:
		pv := ApprovalPolicyValue(payload)
		if pv != ApprovalPolicyValueManualNever && pv != ApprovalPolicyValueManualAlways {
			return fmt.Errorf("invalid approval policy value: %q", payload)
		}
	case PolicyTypeBackupPlan:
		pv := BackupPlanPolicyValue(payload)
		if pv != BackupPlanPolicyValueNever && pv != BackupPlanPolicyValueDaily && pv != BackupPlanPolicyValueWeekly {
			return fmt.Errorf("invalid backup plan policy value: %q", payload)
		}
	}
	return nil
}

// GetDefaultPolicy will return the default value for the given policy type.
// The default policy can be empty when we don't have anything to enforce at runtime.
func GetDefaultPolicy(pType PolicyType) string {
	switch pType {
	case PolicyTypeApprovalPolicy:
		return ""
	case PolicyTypeBackupPlan:
		return string(BackupPlanPolicyValueNever)
	}
	return ""
}
