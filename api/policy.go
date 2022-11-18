package api

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/advisor"
)

// PolicyType is the type or name of a policy.
type PolicyType string

// PipelineApprovalValue is value for approval policy.
type PipelineApprovalValue string

// AssigneeGroupValue is the value for assignee group policy.
type AssigneeGroupValue string

// BackupPlanPolicySchedule is value for backup plan policy.
type BackupPlanPolicySchedule string

// EnvironmentTierValue is the value for environment tier policy.
type EnvironmentTierValue string

// PolicyResourceType is the resource type for a policy.
type PolicyResourceType string

const (
	// DefaultPolicyID is the ID of the default policy.
	DefaultPolicyID int = 0

	// PolicyTypePipelineApproval is the approval policy type.
	PolicyTypePipelineApproval PolicyType = "bb.policy.pipeline-approval"
	// PolicyTypeBackupPlan is the backup plan policy type.
	PolicyTypeBackupPlan PolicyType = "bb.policy.backup-plan"
	// PolicyTypeSQLReview is the sql review policy type.
	PolicyTypeSQLReview PolicyType = "bb.policy.sql-review"
	// PolicyTypeEnvironmentTier is the tier of an environment.
	PolicyTypeEnvironmentTier PolicyType = "bb.policy.environment-tier"

	// PipelineApprovalValueManualNever means the pipeline will automatically be approved without user intervention.
	PipelineApprovalValueManualNever PipelineApprovalValue = "MANUAL_APPROVAL_NEVER"
	// PipelineApprovalValueManualAlways means the pipeline should be manually approved by user to proceed.
	PipelineApprovalValueManualAlways PipelineApprovalValue = "MANUAL_APPROVAL_ALWAYS"

	// AssigneeGroupValueWorkspaceOwnerOrDBA means the assignee can be selected from the workspace owners and DBAs.
	AssigneeGroupValueWorkspaceOwnerOrDBA AssigneeGroupValue = "WORKSPACE_OWNER_OR_DBA"
	// AssigneeGroupValueProjectOwner means the assignee can be selected from the project owners.
	AssigneeGroupValueProjectOwner AssigneeGroupValue = "PROJECT_OWNER"

	// BackupPlanPolicyScheduleUnset is NEVER backup plan policy value.
	BackupPlanPolicyScheduleUnset BackupPlanPolicySchedule = "UNSET"
	// BackupPlanPolicyScheduleDaily is DAILY backup plan policy value.
	BackupPlanPolicyScheduleDaily BackupPlanPolicySchedule = "DAILY"
	// BackupPlanPolicyScheduleWeekly is WEEKLY backup plan policy value.
	BackupPlanPolicyScheduleWeekly BackupPlanPolicySchedule = "WEEKLY"

	// EnvironmentTierValueProtected is PROTECTED environment tier value.
	EnvironmentTierValueProtected EnvironmentTierValue = "PROTECTED"
	// EnvironmentTierValueUnprotected is UNPROTECTED environment tier value.
	EnvironmentTierValueUnprotected EnvironmentTierValue = "UNPROTECTED"

	// PolicyResourceTypeWorkspace is the resource type for workspaces.
	PolicyResourceTypeWorkspace PolicyResourceType = "WORKSPACE"
	// PolicyResourceTypeEnvironment is the resource type for environments.
	PolicyResourceTypeEnvironment PolicyResourceType = "ENVIRONMENT"
	// PolicyResourceTypeProject is the resource type for projects.
	PolicyResourceTypeProject PolicyResourceType = "PROJECT"
	// PolicyResourceTypeInstance is the resource type for instances.
	PolicyResourceTypeInstance PolicyResourceType = "INSTANCE"
	// PolicyResourceTypeDatabase is the resource type for databases.
	PolicyResourceTypeDatabase PolicyResourceType = "DATABASE"
)

var (
	// PolicyTypes is a set of all policy types.
	PolicyTypes = map[PolicyType]bool{
		PolicyTypePipelineApproval: true,
		PolicyTypeBackupPlan:       true,
		PolicyTypeSQLReview:        true,
		PolicyTypeEnvironmentTier:  true,
	}
)

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
	ResourceType PolicyResourceType
	ResourceID   int
	Environment  *Environment `jsonapi:"relation,environment"`

	// Domain specific fields
	Type    PolicyType `jsonapi:"attr,type"`
	Payload string     `jsonapi:"attr,payload"`
}

// PolicyFind is the message to get a policy.
type PolicyFind struct {
	ID *int

	// Related fields
	ResourceType PolicyResourceType
	ResourceID   *int

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
	RowStatus *string `jsonapi:"attr,rowStatus"`

	// Related fields
	ResourceType PolicyResourceType
	ResourceID   int

	// Domain specific fields
	Type    PolicyType
	Payload *string `jsonapi:"attr,payload"`
}

// PolicyDelete is the message to delete a policy.
type PolicyDelete struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int

	// Related fields
	ResourceType PolicyResourceType
	ResourceID   int

	// Domain specific fields
	// Type is the policy type.
	// Currently we only support delete operation for "bb.policy.sql-review", need it here for validation and update query.
	Type PolicyType
}

// PipelineApprovalPolicy is the policy configuration for pipeline approval.
type PipelineApprovalPolicy struct {
	Value PipelineApprovalValue `json:"value"`
	// The AssigneeGroup is the final value of the assignee group which overrides the default value.
	// If there is no value provided in the AssigneeGroupList, we use the the workspace owners and DBAs (default) as the available assignee.
	// If the AssigneeGroupValue is PROJECT_OWNER, the available assignee is the project owners.
	AssigneeGroupList []AssigneeGroup `json:"assigneeGroupList"`
}

func (pa *PipelineApprovalPolicy) String() (string, error) {
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
		return nil, errors.Wrapf(err, "failed to unmarshal pipeline approval policy %q", payload)
	}
	return &pa, nil
}

// AssigneeGroup is the configuration of the assignee group.
type AssigneeGroup struct {
	IssueType IssueType          `json:"issueType"`
	Value     AssigneeGroupValue `json:"value"`
}

func (p *AssigneeGroup) String() (string, error) {
	s, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// BackupPlanPolicy is the policy configuration for backup plan.
type BackupPlanPolicy struct {
	Schedule BackupPlanPolicySchedule `json:"schedule"`
	// RetentionPeriodTs is the minimum allowed period that backup data is kept for databases in an environment.
	RetentionPeriodTs int `json:"retentionPeriodTs"`
}

func (bp *BackupPlanPolicy) String() (string, error) {
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
		return nil, errors.Wrapf(err, "failed to unmarshal backup plan policy %q", payload)
	}
	return &bp, nil
}

// UnmarshalSQLReviewPolicy will unmarshal payload to SQL review policy.
func UnmarshalSQLReviewPolicy(payload string) (*advisor.SQLReviewPolicy, error) {
	var sr advisor.SQLReviewPolicy
	if err := json.Unmarshal([]byte(payload), &sr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal SQL review policy %q", payload)
	}
	return &sr, nil
}

// EnvironmentTierPolicy is the tier of an environment.
type EnvironmentTierPolicy struct {
	EnvironmentTier EnvironmentTierValue `json:"environmentTier"`
}

func (p *EnvironmentTierPolicy) String() (string, error) {
	s, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// UnmarshalEnvironmentTierPolicy will unmarshal payload to environment tier policy.
func UnmarshalEnvironmentTierPolicy(payload string) (*EnvironmentTierPolicy, error) {
	var p EnvironmentTierPolicy
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal environment tier policy %q", payload)
	}
	return &p, nil
}

// ValidatePolicy will validate the policy type and payload values.
func ValidatePolicy(pType PolicyType, payload string) error {
	if !PolicyTypes[pType] {
		return errors.Errorf("invalid policy type: %s", pType)
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
			return errors.Errorf("invalid approval policy value: %q", payload)
		}
		issueTypeSeen := make(map[IssueType]bool)
		for _, group := range pa.AssigneeGroupList {
			if group.IssueType != IssueDatabaseSchemaUpdate &&
				group.IssueType != IssueDatabaseSchemaUpdateGhost &&
				group.IssueType != IssueDatabaseDataUpdate {
				return errors.Errorf("invalid assignee group issue type %q", group.IssueType)
			}
			if issueTypeSeen[group.IssueType] {
				return errors.Errorf("duplicate assignee group issue type %q", group.IssueType)
			}
			issueTypeSeen[group.IssueType] = true
		}
	case PolicyTypeBackupPlan:
		bp, err := UnmarshalBackupPlanPolicy(payload)
		if err != nil {
			return err
		}
		if bp.Schedule != BackupPlanPolicyScheduleUnset && bp.Schedule != BackupPlanPolicyScheduleDaily && bp.Schedule != BackupPlanPolicyScheduleWeekly {
			return errors.Errorf("invalid backup plan policy schedule: %q", bp.Schedule)
		}
	case PolicyTypeSQLReview:
		sr, err := UnmarshalSQLReviewPolicy(payload)
		if err != nil {
			return err
		}
		if err := sr.Validate(); err != nil {
			return errors.Wrap(err, "invalid SQL review policy")
		}
	case PolicyTypeEnvironmentTier:
		p, err := UnmarshalEnvironmentTierPolicy(payload)
		if err != nil {
			return err
		}
		if p.EnvironmentTier != EnvironmentTierValueProtected && p.EnvironmentTier != EnvironmentTierValueUnprotected {
			return errors.Errorf("invalid environment tier value %q", p.EnvironmentTier)
		}
	}
	return nil
}

// GetDefaultPolicy will return the default value for the given policy type.
// The default policy can be empty when we don't have anything to enforce at runtime.
func GetDefaultPolicy(pType PolicyType) (string, error) {
	switch pType {
	case PolicyTypePipelineApproval:
		policy := PipelineApprovalPolicy{
			Value: PipelineApprovalValueManualAlways,
		}
		return policy.String()
	case PolicyTypeBackupPlan:
		policy := BackupPlanPolicy{
			Schedule: BackupPlanPolicyScheduleUnset,
		}
		return policy.String()
	case PolicyTypeSQLReview:
		// TODO(ed): we may need to define the default SQL review policy payload in the PR of policy data migration.
		return "{}", nil
	case PolicyTypeEnvironmentTier:
		policy := EnvironmentTierPolicy{
			EnvironmentTier: EnvironmentTierValueUnprotected,
		}
		return policy.String()
	}
	return "", nil
}
