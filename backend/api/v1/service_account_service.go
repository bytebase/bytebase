package v1

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// ServiceAccountService implements the service account service.
type ServiceAccountService struct {
	v1connect.UnimplementedServiceAccountServiceHandler
	store      *store.Store
	iamManager *iam.Manager
}

// NewServiceAccountService creates a new ServiceAccountService.
func NewServiceAccountService(store *store.Store, iamManager *iam.Manager) *ServiceAccountService {
	return &ServiceAccountService{
		store:      store,
		iamManager: iamManager,
	}
}

// CreateServiceAccount creates a new service account.
func (s *ServiceAccountService) CreateServiceAccount(ctx context.Context, request *connect.Request[v1pb.CreateServiceAccountRequest]) (*connect.Response[v1pb.ServiceAccount], error) {
	// Parse parent to determine workspace vs project level
	var projectID *string
	parent := request.Msg.Parent
	if parent != "" {
		// project-level service account: parent = "projects/{project}"
		if strings.HasPrefix(parent, common.ProjectNamePrefix) {
			pid, err := common.GetProjectID(parent)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid parent %q", parent))
			}
			projectID = &pid
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid parent format %q, expected projects/{project} or empty", parent))
		}
	}

	serviceAccountID := request.Msg.ServiceAccountId
	if serviceAccountID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("service_account_id is required"))
	}

	sa := request.Msg.ServiceAccount
	if sa == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("service_account is required"))
	}

	// Build email using helper function
	projectIDStr := ""
	if projectID != nil {
		projectIDStr = *projectID
	}
	email := common.BuildServiceAccountEmail(serviceAccountID, projectIDStr)

	// Check for duplicate email
	existingSA, err := s.store.GetServiceAccountByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check for existing service account"))
	}
	if existingSA != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("service account with email %q already exists", email))
	}

	// Generate service key
	keyRandom, err := common.RandomString(20)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate service key"))
	}
	serviceKey := fmt.Sprintf("%s%s", common.ServiceAccountAccessKeyPrefix, keyRandom)

	// Hash the service key as password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(serviceKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to hash service key"))
	}

	// Create the service account with password hash
	createdSA, err := s.store.CreateServiceAccount(ctx, &store.CreateServiceAccountMessage{
		Email:        email,
		Name:         sa.Title,
		PasswordHash: string(passwordHash),
		Project:      projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create service account"))
	}

	// Convert to API response and include service key
	result := convertToServiceAccount(createdSA)
	result.ServiceKey = serviceKey

	return connect.NewResponse(result), nil
}

// GetServiceAccount gets a service account by name.
func (s *ServiceAccountService) GetServiceAccount(ctx context.Context, request *connect.Request[v1pb.GetServiceAccountRequest]) (*connect.Response[v1pb.ServiceAccount], error) {
	email, err := common.GetServiceAccountEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	sa, err := s.store.GetServiceAccountByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get service account"))
	}
	if sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("service account %q not found", email))
	}

	return connect.NewResponse(convertToServiceAccount(sa)), nil
}

// ListServiceAccounts lists service accounts.
func (s *ServiceAccountService) ListServiceAccounts(ctx context.Context, request *connect.Request[v1pb.ListServiceAccountsRequest]) (*connect.Response[v1pb.ListServiceAccountsResponse], error) {
	// Parse parent to determine workspace vs project level
	var projectID *string
	parent := request.Msg.Parent
	if parent != "" {
		if strings.HasPrefix(parent, common.ProjectNamePrefix) {
			pid, err := common.GetProjectID(parent)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid parent %q", parent))
			}
			projectID = &pid
		} else {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid parent format %q, expected projects/{project} or empty", parent))
		}
	} else {
		// workspace-level list - use empty string to filter workspace-level SAs
		emptyProjectID := ""
		projectID = &emptyProjectID
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.Msg.PageToken,
		limit:   int(request.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	// List service accounts using the store method with project filtering
	sas, err := s.store.ListServiceAccounts(ctx, &store.FindServiceAccountMessage{
		Project:     projectID,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		ShowDeleted: request.Msg.ShowDeleted,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list service accounts"))
	}

	nextPageToken := ""
	if len(sas) == limitPlusOne {
		sas = sas[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	response := &v1pb.ListServiceAccountsResponse{
		NextPageToken: nextPageToken,
	}
	for _, sa := range sas {
		response.ServiceAccounts = append(response.ServiceAccounts, convertToServiceAccount(sa))
	}

	return connect.NewResponse(response), nil
}

// UpdateServiceAccount updates a service account.
func (s *ServiceAccountService) UpdateServiceAccount(ctx context.Context, request *connect.Request[v1pb.UpdateServiceAccountRequest]) (*connect.Response[v1pb.ServiceAccount], error) {
	if request.Msg.ServiceAccount == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("service_account is required"))
	}
	if request.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask is required"))
	}

	email, err := common.GetServiceAccountEmail(request.Msg.ServiceAccount.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	sa, err := s.store.GetServiceAccountByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get service account"))
	}
	if sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("service account %q not found", email))
	}
	if sa.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("service account %q has been deleted", email))
	}

	var newServiceKey string
	patch := &store.UpdateServiceAccountMessage{}

	for _, path := range request.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Name = &request.Msg.ServiceAccount.Title
		case "service_key":
			// Rotate service key
			keyRandom, err := common.RandomString(20)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate service key"))
			}
			newServiceKey = fmt.Sprintf("%s%s", common.ServiceAccountAccessKeyPrefix, keyRandom)

			passwordHash, err := bcrypt.GenerateFromPassword([]byte(newServiceKey), bcrypt.DefaultCost)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to hash service key"))
			}
			passwordHashStr := string(passwordHash)
			patch.PasswordHash = &passwordHashStr
		default:
			// Ignore unknown fields
		}
	}

	updatedSA, err := s.store.UpdateServiceAccount(ctx, sa, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update service account"))
	}

	result := convertToServiceAccount(updatedSA)
	if newServiceKey != "" {
		result.ServiceKey = newServiceKey
	}

	return connect.NewResponse(result), nil
}

// DeleteServiceAccount deletes a service account.
func (s *ServiceAccountService) DeleteServiceAccount(ctx context.Context, request *connect.Request[v1pb.DeleteServiceAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	email, err := common.GetServiceAccountEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	sa, err := s.store.GetServiceAccountByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get service account"))
	}
	if sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("service account %q not found", email))
	}
	if sa.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("service account %q has been deleted", email))
	}

	// Soft delete
	if err := s.store.DeleteServiceAccount(ctx, sa); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to delete service account"))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// UndeleteServiceAccount restores a deleted service account.
func (s *ServiceAccountService) UndeleteServiceAccount(ctx context.Context, request *connect.Request[v1pb.UndeleteServiceAccountRequest]) (*connect.Response[v1pb.ServiceAccount], error) {
	email, err := common.GetServiceAccountEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	sa, err := s.store.GetServiceAccountByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get service account"))
	}
	if sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("service account %q not found", email))
	}
	if !sa.MemberDeleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("service account %q is already active", email))
	}

	// Restore
	restoredSA, err := s.store.UndeleteServiceAccount(ctx, sa)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to undelete service account"))
	}

	return connect.NewResponse(convertToServiceAccount(restoredSA)), nil
}

// convertToServiceAccount converts a store.ServiceAccountMessage to a v1pb.ServiceAccount.
// Note: service_key is NOT populated by this function; it should only be returned on create/rotate.
func convertToServiceAccount(sa *store.ServiceAccountMessage) *v1pb.ServiceAccount {
	return &v1pb.ServiceAccount{
		Name:  common.FormatServiceAccountEmail(sa.Email),
		State: convertDeletedToState(sa.MemberDeleted),
		Email: sa.Email,
		Title: sa.Name,
	}
}
