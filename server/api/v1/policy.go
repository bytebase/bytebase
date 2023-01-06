package v1

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

func convertToPolicy(prefix string, policyMessage *store.PolicyMessage) *v1pb.Policy {
	pType := v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
	switch policyMessage.Type {
	case api.PolicyTypePipelineApproval:
		pType = v1pb.PolicyType_PIPELINE_APPROVAL
	case api.PolicyTypeBackupPlan:
		pType = v1pb.PolicyType_BACKUP_PLAN
	case api.PolicyTypeSQLReview:
		pType = v1pb.PolicyType_SQL_REVIEW
	case api.PolicyTypeEnvironmentTier:
		pType = v1pb.PolicyType_ENVIRONMENT_TIER
	case api.PolicyTypeSensitiveData:
		pType = v1pb.PolicyType_SENSITIVE_DATA
	case api.PolicyTypeAccessControl:
		pType = v1pb.PolicyType_ACCESS_CONTROL
	}

	return &v1pb.Policy{
		Name:              fmt.Sprintf("%s/%s%s", prefix, policyNamePrefix, strings.ToLower(pType.String())),
		Uid:               fmt.Sprintf("%d", policyMessage.UID),
		State:             convertDeletedToState(policyMessage.Deleted),
		InheritFromParent: policyMessage.InheritFromParent,
		Type:              pType,
		Payload:           policyMessage.Payload,
	}
}

func convertPolicyType(pType string) (api.PolicyType, error) {
	var policyType api.PolicyType
	switch strings.ToUpper(pType) {
	case v1pb.PolicyType_PIPELINE_APPROVAL.String():
		return api.PolicyTypePipelineApproval, nil
	case v1pb.PolicyType_BACKUP_PLAN.String():
		return api.PolicyTypeBackupPlan, nil
	case v1pb.PolicyType_SQL_REVIEW.String():
		return api.PolicyTypeSQLReview, nil
	case v1pb.PolicyType_ENVIRONMENT_TIER.String():
		return api.PolicyTypeEnvironmentTier, nil
	case v1pb.PolicyType_SENSITIVE_DATA.String():
		return api.PolicyTypeSensitiveData, nil
	case v1pb.PolicyType_ACCESS_CONTROL.String():
		return api.PolicyTypeAccessControl, nil
	}
	return policyType, errors.Errorf("invalid policy type %v", pType)
}
