package api

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
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

	// PolicyTypeWorkspaceIAM is the workspace IAM policy type.
	PolicyTypeWorkspaceIAM PolicyType = "bb.policy.workspace-iam"
	// PolicyTypePipelineApproval is the approval policy type.
	PolicyTypePipelineApproval PolicyType = "bb.policy.pipeline-approval"
	// PolicyTypeBackupPlan is the backup plan policy type.
	PolicyTypeBackupPlan PolicyType = "bb.policy.backup-plan"
	// PolicyTypeSQLReview is the sql review policy type.
	PolicyTypeSQLReview PolicyType = "bb.policy.sql-review"
	// PolicyTypeEnvironmentTier is the tier of an environment.
	PolicyTypeEnvironmentTier PolicyType = "bb.policy.environment-tier"
	// PolicyTypeMasking is the masking policy type.
	PolicyTypeMasking PolicyType = "bb.policy.masking"
	// PolicyTypeMaskingRule is the masking rule policy type.
	PolicyTypeMaskingRule PolicyType = "bb.policy.masking-rule"
	// PolicyTypeMaskingException is the masking exception policy type.
	PolicyTypeMaskingException PolicyType = "bb.policy.masking-exception"
	// PolicyTypeSlowQuery is the slow query policy type.
	PolicyTypeSlowQuery PolicyType = "bb.policy.slow-query"
	// PolicyTypeDisableCopyData is the disable copy data policy type.
	PolicyTypeDisableCopyData PolicyType = "bb.policy.disable-copy-data"

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

	// PolicyResourceTypeUnknown is the unknown resource type.
	PolicyResourceTypeUnknown PolicyResourceType = ""
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
	// AllowedResourceTypes includes allowed resource types for each policy type.
	AllowedResourceTypes = map[PolicyType][]PolicyResourceType{
		PolicyTypeWorkspaceIAM:     {PolicyResourceTypeWorkspace},
		PolicyTypePipelineApproval: {PolicyResourceTypeEnvironment},
		PolicyTypeBackupPlan:       {PolicyResourceTypeEnvironment},
		PolicyTypeSQLReview:        {PolicyResourceTypeEnvironment},
		PolicyTypeEnvironmentTier:  {PolicyResourceTypeEnvironment},
		PolicyTypeMasking:          {PolicyResourceTypeDatabase},
		PolicyTypeSlowQuery:        {PolicyResourceTypeInstance},
		PolicyTypeDisableCopyData:  {PolicyResourceTypeEnvironment},
	}
)

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

// SensitiveDataPolicy is the policy configuration for sensitive data.
// It is only applicable to database resource type.
type SensitiveDataPolicy struct {
	SensitiveDataList []SensitiveData `json:"sensitiveDataList"`
}

// SensitiveData is the value for sensitive data.
type SensitiveData struct {
	Schema string                `json:"schema"`
	Table  string                `json:"table"`
	Column string                `json:"column"`
	Type   SensitiveDataMaskType `json:"maskType"`
}

// SensitiveDataMaskType is the mask type for sensitive data.
type SensitiveDataMaskType string

const (
	// SensitiveDataMaskTypeDefault is the sensitive data type to hide data with a default method.
	// The default method is subject to change.
	SensitiveDataMaskTypeDefault SensitiveDataMaskType = "DEFAULT"
)

// UnmarshalSensitiveDataPolicy will unmarshal payload to sensitive data policy.
func UnmarshalSensitiveDataPolicy(payload string) (*SensitiveDataPolicy, error) {
	var p SensitiveDataPolicy
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal sensitive data policy %q", payload)
	}
	return &p, nil
}

func (p *SensitiveDataPolicy) String() (string, error) {
	s, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// SlowQueryPolicy is the policy configuration for slow query.
type SlowQueryPolicy struct {
	Active bool `json:"active"`
}

// UnmarshalSlowQueryPolicy will unmarshal payload to slow query policy.
func UnmarshalSlowQueryPolicy(payload string) (*SlowQueryPolicy, error) {
	var p SlowQueryPolicy
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal slow query policy %q", payload)
	}
	return &p, nil
}

// String will return the string representation of the policy.
func (p *SlowQueryPolicy) String() (string, error) {
	s, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// DisableCopyDataPolicy is the policy configuration for disabling copying data.
type DisableCopyDataPolicy struct {
	Active bool `json:"active"`
}

// UnmarshalDisableCopyDataPolicyPolicy will unmarshal payload to disable copy data policy.
func UnmarshalDisableCopyDataPolicyPolicy(payload string) (*DisableCopyDataPolicy, error) {
	var p DisableCopyDataPolicy
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal disable copy data policy %q", payload)
	}
	return &p, nil
}

// String will return the string representation of the policy.
func (p *DisableCopyDataPolicy) String() (string, error) {
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

// GetPolicyResourceType gets the policy resource type.
func GetPolicyResourceType(resourceType string) (PolicyResourceType, error) {
	var rt PolicyResourceType
	switch resourceType {
	case "workspace":
		rt = PolicyResourceTypeWorkspace
	case "environment":
		rt = PolicyResourceTypeEnvironment
	case "project":
		rt = PolicyResourceTypeProject
	case "instance":
		rt = PolicyResourceTypeInstance
	case "database":
		rt = PolicyResourceTypeDatabase
	default:
		return PolicyResourceTypeUnknown, errors.Errorf("invalid policy resource type %q", rt)
	}
	return rt, nil
}
