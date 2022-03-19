package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// PolicyType is the type or name of a policy.
type PolicyType string

// PipelineApprovalValue is value for approval policy.
type PipelineApprovalValue string

// BackupPlanPolicySchedule is value for backup plan policy.
type BackupPlanPolicySchedule string

const (
	// PolicyTypePipelineApproval is the approval policy type.
	PolicyTypePipelineApproval PolicyType = "bb.policy.pipeline-approval"
	// PolicyTypeBackupPlan is the backup plan policy type.
	PolicyTypeBackupPlan PolicyType = "bb.policy.backup-plan"

	// PipelineApprovalValueManualNever is MANUAL_APPROVAL_NEVER approval policy value.
	PipelineApprovalValueManualNever PipelineApprovalValue = "MANUAL_APPROVAL_NEVER"
	// PipelineApprovalValueManualAlways is MANUAL_APPROVAL_ALWAYS approval policy value.
	PipelineApprovalValueManualAlways PipelineApprovalValue = "MANUAL_APPROVAL_ALWAYS"

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
		PolicyTypePipelineApproval: true,
		PolicyTypeBackupPlan:       true,
	}
)

// PolicyRaw is the store model for an Policy.
// Fields have exactly the same meanings as Policy.
type PolicyRaw struct {
	ID int

	// Standard fields
	RowStatus RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	EnvironmentID int

	// Domain specific fields
	Type    PolicyType
	Payload string
}

// ToPolicy creates an instance of Policy based on the PolicyRaw.
// This is intended to be called when we need to compose an Policy relationship.
func (raw *PolicyRaw) ToPolicy() *Policy {
	return &Policy{
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		EnvironmentID: raw.EnvironmentID,

		// Domain specific fields
		Type:    raw.Type,
		Payload: raw.Payload,
	}
}

// Policy is the API message for a policy.
type Policy struct {
	ID int `jsonapi:"primary,policy"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	EnvironmentID int
	Environment   *Environment `jsonapi:"relation,environment"`

	// Domain specific fields
	Type    PolicyType `jsonapi:"attr,type"`
	Payload string     `jsonapi:"attr,payload"`
}

// PolicyFind is the message to get a policy.
type PolicyFind struct {
	ID *int

	// Related fields
	EnvironmentID *int

	// Domain specific fields
	Type *PolicyType `jsonapi:"attr,type"`
}

// PolicyUpsert is the message to upsert a policy.
// NOTE: We use PATCH for Upsert, this is inspired by https://google.aip.dev/134#patch-and-put
type PolicyUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorID is the ID of the creator.
	UpdaterID int

	// Related fields
	EnvironmentID int

	// Domain specific fields
	Type    PolicyType
	Payload string `jsonapi:"attr,payload"`
}

// PolicyService is the backend for policies.
type PolicyService interface {
	FindPolicy(ctx context.Context, find *PolicyFind) (*PolicyRaw, error)
	UpsertPolicy(ctx context.Context, upsert *PolicyUpsert) (*PolicyRaw, error)
	GetBackupPlanPolicy(ctx context.Context, environmentID int) (*BackupPlanPolicy, error)
	GetPipelineApprovalPolicy(ctx context.Context, environmentID int) (*PipelineApprovalPolicy, error)
}

// PipelineApprovalPolicy is the policy configuration for pipeline approval
type PipelineApprovalPolicy struct {
	Value PipelineApprovalValue `json:"value"`
}

func (pa PipelineApprovalPolicy) String() (string, error) {
	s, err := json.Marshal(pa)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// UnmarshalPipelineApprovalPolicy will unmarshal payload to pipeline approval policy.
func UnmarshalPipelineApprovalPolicy(payload string) (*PipelineApprovalPolicy, error) {
	var pa PipelineApprovalPolicy
	if err := json.Unmarshal([]byte(payload), &pa); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline approval policy %q: %q", payload, err)
	}
	return &pa, nil
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
	case PolicyTypePipelineApproval:
		pa, err := UnmarshalPipelineApprovalPolicy(payload)
		if err != nil {
			return err
		}
		if pa.Value != PipelineApprovalValueManualNever && pa.Value != PipelineApprovalValueManualAlways {
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
	case PolicyTypePipelineApproval:
		return PipelineApprovalPolicy{
			Value: PipelineApprovalValueManualAlways,
		}.String()
	case PolicyTypeBackupPlan:
		return BackupPlanPolicy{
			Schedule: BackupPlanPolicyScheduleUnset,
		}.String()
	}
	return "", nil
}
