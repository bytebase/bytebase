package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

// PolicyService implements the policy service.
type PolicyService struct {
	v1pb.UnimplementedPolicyServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewPolicyService creates a new PolicyService.
func NewPolicyService(store *store.Store, licenseService enterpriseAPI.LicenseService) *PolicyService {
	return &PolicyService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetPolicy gets a policy.
func (s *PolicyService) GetPolicy(ctx context.Context, request *v1pb.GetPolicyRequest) (*v1pb.Policy, error) {
	// the policy request should be policies/{policy-type}/{policy-id}
	sections := strings.Split(request.Name, "/")
	if len(sections) != 3 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid request path %s", request.Name))
	}

	policyID := sections[2]
	rType, err := getPolicyType(strings.Join(sections[0:2], "/"))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	resourceType, err := api.GetPolicyResourceType(rType)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		UID:          &policyID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if policy == nil {
		return nil, status.Errorf(codes.NotFound, "policy %s/%s not found", rType, policyID)
	}

	return convertToPolicy(policy), nil
}

// ListPolicies lists all policies.
func (s *PolicyService) ListPolicies(ctx context.Context, request *v1pb.ListPoliciesRequest) (*v1pb.ListPoliciesResponse, error) {
	rType, err := getPolicyType(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	resourceType, err := api.GetPolicyResourceType(rType)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	policies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListPoliciesResponse{}
	for _, policy := range policies {
		response.Policies = append(response.Policies, convertToPolicy(policy))
	}
	return response, nil
}

func convertToPolicy(policyMessage *store.PolicyMessage) *v1pb.Policy {
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
		Name:              fmt.Sprintf("%s%s/%s", policyNamePrefix, strings.ToLower(string(policyMessage.ResourceType)), policyMessage.ResourceID),
		Uid:               fmt.Sprintf("%d", policyMessage.UID),
		State:             convertDeletedToState(policyMessage.Deleted),
		InheritFromParent: policyMessage.InheritFromParent,
		Type:              pType,
		Payload:           policyMessage.Payload,
	}
}
