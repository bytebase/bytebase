package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

// OrgPolicyService implements the workspace policy service.
type OrgPolicyService struct {
	v1pb.UnimplementedOrgPolicyServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewOrgPolicyService creates a new OrgPolicyService.
func NewOrgPolicyService(store *store.Store, licenseService enterpriseAPI.LicenseService) *OrgPolicyService {
	return &OrgPolicyService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetPolicy gets a policy in a specific resource.
func (s *OrgPolicyService) GetPolicy(ctx context.Context, request *v1pb.GetPolicyRequest) (*v1pb.Policy, error) {
	tokens := strings.Split(request.Name, policyNamePrefix)
	if len(tokens) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request %s", request.Name)
	}

	token := tokens[0]
	if strings.HasSuffix(token, "/") {
		token = token[:(len(token) - 1)]
	}
	resourceType, resourceID, err := s.getPolicyResourceTypeAndID(ctx, token)
	if err != nil {
		return nil, err
	}

	policyType, err := convertPolicyType(tokens[1])
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
		ResourceUID:  &resourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if policy == nil {
		return nil, status.Errorf(codes.NotFound, "policy %q not found", request.Name)
	}

	return convertToPolicy(token, policy), nil
}

// ListPolicies lists policies in a specific resource.
func (s *OrgPolicyService) ListPolicies(ctx context.Context, request *v1pb.ListPoliciesRequest) (*v1pb.ListPoliciesResponse, error) {
	resourceType, resourceID, err := s.getPolicyResourceTypeAndID(ctx, request.Parent)
	if err != nil {
		return nil, err
	}

	policies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &resourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &v1pb.ListPoliciesResponse{}
	for _, policy := range policies {
		response.Policies = append(response.Policies, convertToPolicy(request.Parent, policy))
	}
	return response, nil
}

func (s *OrgPolicyService) getPolicyResourceTypeAndID(ctx context.Context, requestName string) (api.PolicyResourceType, int, error) {
	if requestName == "" {
		return api.PolicyResourceTypeWorkspace, 0, nil
	}

	if strings.HasPrefix(requestName, projectNamePrefix) {
		projectID, err := getProjectID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, err.Error())
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID:  &projectID,
			ShowDeleted: true,
		})
		if err != nil {
			return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.Internal, err.Error())
		}
		if project == nil {
			return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.NotFound, "project %q not found", projectID)
		}
		if project.Deleted {
			return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, "project %q has been deleted", projectID)
		}

		return api.PolicyResourceTypeProject, project.UID, nil
	}

	if strings.HasPrefix(requestName, environmentNamePrefix) {
		sections := strings.Split(requestName, "/")

		// environment policy request name should be environments/{environment id}
		if len(sections) == 2 {
			environmentID, err := getEnvironmentID(requestName)
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, err.Error())
			}
			environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
				ResourceID: &environmentID,
			})
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.Internal, err.Error())
			}
			if environment == nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
			}
			if environment.Deleted {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, "environment %q has been deleted", environmentID)
			}

			return api.PolicyResourceTypeEnvironment, environment.UID, nil
		}

		// instance policy request name should be environments/{environment id}/instances/{instance id}
		if len(sections) == 4 {
			environmentID, instanceID, err := getEnvironmentInstanceID(requestName)
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, err.Error())
			}

			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
				EnvironmentID: &environmentID,
				ResourceID:    &instanceID,
				ShowDeleted:   true,
			})
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.Internal, err.Error())
			}
			if instance == nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
			}
			if instance.Deleted {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", instanceID)
			}

			return api.PolicyResourceTypeInstance, instance.UID, nil
		}

		// database policy request name should be environments/{environment id}/instances/{instance id}/databases/{db name}
		if len(sections) == 6 {
			environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(requestName)
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, err.Error())
			}
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				EnvironmentID: &environmentID,
				InstanceID:    &instanceID,
				DatabaseName:  &databaseName,
			})
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.Internal, err.Error())
			}
			if database == nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.NotFound, "database %q not found", databaseName)
			}

			return api.PolicyResourceTypeDatabase, database.UID, nil
		}
	}

	return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, "unknown request name %s", requestName)
}

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
