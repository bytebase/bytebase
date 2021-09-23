package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// PolicyType is the type or name of a policy.
type PolicyType string

// ApprovalPolicyValue is value for approval policy.
type ApprovalPolicyValue string

// BackupPlanPolicySchedule is value for backup plan policy.
type BackupPlanPolicySchedule string

const (
	// PolicyTypeApprovalPolicy is the approval policy type.
	PolicyTypeApprovalPolicy PolicyType = "approval_policy"
	// PolicyTypeBackupPlan is the backup plan policy type.
	PolicyTypeBackupPlan PolicyType = "backup_plan"

	// ApprovalPolicyValueManualNever is MANUAL_APPROVAL_NEVER approval policy value.
	ApprovalPolicyValueManualNever ApprovalPolicyValue = "MANUAL_APPROVAL_NEVER"
	// ApprovalPolicyValueManualAlways is MANUAL_APPROVAL_ALWAYS approval policy value.
	ApprovalPolicyValueManualAlways ApprovalPolicyValue = "MANUAL_APPROVAL_ALWAYS"

	// BackupPlanPolicyScheduleUnset is NEVER backup plan policy value.
	BackupPlanPolicyScheduleUnset BackupPlanPolicySchedule = "UNSET"
	// BackupPlanPolicyScheduleDaily is DAILY backup plan policy value.
	BackupPlanPolicyScheduleDaily BackupPlanPolicySchedule = "DAILY"
	// BackupPlanPolicyScheduleWeekly is WEEKLY backup plan policy value.
	BackupPlanPolicyScheduleWeekly BackupPlanPolicySchedule = "WEEKLY"
)

var (
	// PolicyTypes is a set of all policy types.
	PolicyTypes = map[PolicyType]bool{
		PolicyTypeApprovalPolicy: true,
		PolicyTypeBackupPlan:     true,
	}
)

type Policy struct {
	ID int `jsonapi:"primary,policy"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	EnvironmentId int
	Environment   *Environment `jsonapi:"relation,environment"`

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
	Type    PolicyType
	Payload string `jsonapi:"attr,payload"`
}

// PolicyService is the backend for policies.
type PolicyService interface {
	FindPolicy(ctx context.Context, find *PolicyFind) (*Policy, error)
	UpsertPolicy(ctx context.Context, upsert *PolicyUpsert) (*Policy, error)
	GetBackupPlanPolicy(ctx context.Context, environmentID int) (*BackupPlanPolicy, error)
}

// BackupPlanPolicy is the policy configuration for backup plan.
type BackupPlanPolicy struct {
	Schedule BackupPlanPolicySchedule `json:"schedule"`
}

func (bp BackupPlanPolicy) String() (string, error) {
	s, err := json.Marshal(bp)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// UnmarshalBackupPlanPolicy will unmarshal payload to backup plan policy.
func UnmarshalBackupPlanPolicy(payload string) (*BackupPlanPolicy, error) {
	var bp BackupPlanPolicy
	if err := json.Unmarshal([]byte(payload), &bp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup plan policy %q: %q", payload, err)
	}
	return &bp, nil
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
		bp, err := UnmarshalBackupPlanPolicy(payload)
		if err != nil {
			return err
		}
		if bp.Schedule != BackupPlanPolicyScheduleUnset && bp.Schedule != BackupPlanPolicyScheduleDaily && bp.Schedule != BackupPlanPolicyScheduleWeekly {
			return fmt.Errorf("invalid backup plan policy schedule: %q", bp.Schedule)
		}
	}
	return nil
}

// GetDefaultPolicy will return the default value for the given policy type.
// The default policy can be empty when we don't have anything to enforce at runtime.
func GetDefaultPolicy(pType PolicyType) (string, error) {
	switch pType {
	case PolicyTypeApprovalPolicy:
		return "", nil
	case PolicyTypeBackupPlan:
		return BackupPlanPolicy{
			Schedule: BackupPlanPolicyScheduleUnset,
		}.String()
	}
	return "", nil
}
