package api

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// PolicyType is the type or name of a policy.
type PolicyType string

// PipelineApprovalValue is value for approval policy.
type PipelineApprovalValue string

// AssigneeGroupValue is the value for assignee group policy.
type AssigneeGroupValue string

// EnvironmentTierValue is the value for environment tier policy.
type EnvironmentTierValue string

// PolicyResourceType is the resource type for a policy.
type PolicyResourceType string

// ReservedTag is the reserved tags for bb.policy.tag.
type ReservedTag string

const (
	// DefaultPolicyID is the ID of the default policy.
	DefaultPolicyID int = 0

	// PolicyTypeRollout is the rollout policy type.
	PolicyTypeRollout PolicyType = "bb.policy.rollout"
	// PolicyTypeEnvironmentTier is the tier of an environment.
	PolicyTypeEnvironmentTier PolicyType = "bb.policy.environment-tier"
	// PolicyTypeMasking is the masking policy type.
	PolicyTypeMasking PolicyType = "bb.policy.masking"
	// PolicyTypeMaskingException is the masking exception policy type.
	PolicyTypeMaskingException PolicyType = "bb.policy.masking-exception"
	// PolicyTypeSlowQuery is the slow query policy type.
	PolicyTypeSlowQuery PolicyType = "bb.policy.slow-query"
	// PolicyTypeDisableCopyData is the disable copy data policy type.
	PolicyTypeDisableCopyData PolicyType = "bb.policy.disable-copy-data"
	// PolicyTypeMaskingRule is the masking rule policy type.
	PolicyTypeMaskingRule PolicyType = "bb.policy.masking-rule"
	// PolicyTypeRestrictIssueCreationForSQLReview is the policy type for restricting issue creation for SQL review.
	PolicyTypeRestrictIssueCreationForSQLReview PolicyType = "bb.policy.restrict-issue-creation-for-sql-review"
	// PolicyTypeProjectIAM is the policy for IAM in the project.
	PolicyTypeProjectIAM PolicyType = "bb.policy.project-iam"
	// PolicyTypeTag is the policy type for resource tags.
	PolicyTypeTag PolicyType = "bb.policy.tag"

	// PipelineApprovalValueManualNever means the pipeline will automatically be approved without user intervention.
	PipelineApprovalValueManualNever PipelineApprovalValue = "MANUAL_APPROVAL_NEVER"
	// PipelineApprovalValueManualAlways means the pipeline should be manually approved by user to proceed.
	PipelineApprovalValueManualAlways PipelineApprovalValue = "MANUAL_APPROVAL_ALWAYS"

	// AssigneeGroupValueWorkspaceOwnerOrDBA means the assignee can be selected from the workspace owners and DBAs.
	AssigneeGroupValueWorkspaceOwnerOrDBA AssigneeGroupValue = "WORKSPACE_OWNER_OR_DBA"
	// AssigneeGroupValueProjectOwner means the assignee can be selected from the project owners.
	AssigneeGroupValueProjectOwner AssigneeGroupValue = "PROJECT_OWNER"

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

	// ReservedTagReviewConfig is the tag for review config.
	ReservedTagReviewConfig ReservedTag = "bb.tag.review_config"
)

var (
	// AllowedResourceTypes includes allowed resource types for each policy type.
	AllowedResourceTypes = map[PolicyType][]PolicyResourceType{
		PolicyTypeRollout:                           {PolicyResourceTypeEnvironment},
		PolicyTypeEnvironmentTier:                   {PolicyResourceTypeEnvironment},
		PolicyTypeTag:                               {PolicyResourceTypeEnvironment, PolicyResourceTypeProject, PolicyResourceTypeDatabase},
		PolicyTypeMasking:                           {PolicyResourceTypeDatabase},
		PolicyTypeSlowQuery:                         {PolicyResourceTypeInstance},
		PolicyTypeDisableCopyData:                   {PolicyResourceTypeEnvironment, PolicyResourceTypeProject},
		PolicyTypeMaskingRule:                       {PolicyResourceTypeWorkspace},
		PolicyTypeMaskingException:                  {PolicyResourceTypeProject},
		PolicyTypeRestrictIssueCreationForSQLReview: {PolicyResourceTypeWorkspace, PolicyResourceTypeProject},
	}
)

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

// String will return the string representation of the policy.
func (p *DisableCopyDataPolicy) String() (string, error) {
	s, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

// RestrictIssueCreationForSQLReviewPolicy is the policy configuration for restricting issue creation for SQL review.
type RestrictIssueCreationForSQLReviewPolicy struct {
	Disallow bool `json:"disallow"`
}

// UnmarshalRestrictIssueCreationForSQLReviewPolicy will unmarshal payload to restrict issue creation for SQL review policy.
func UnmarshalRestrictIssueCreationForSQLReviewPolicy(payload string) (*RestrictIssueCreationForSQLReviewPolicy, error) {
	var p RestrictIssueCreationForSQLReviewPolicy
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal restrict issue creation for SQL review policy %q", payload)
	}
	return &p, nil
}

func (p *RestrictIssueCreationForSQLReviewPolicy) String() (string, error) {
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
