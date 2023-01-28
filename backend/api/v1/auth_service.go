package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/idp/oauth2"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// AuthService implements the auth service.
type AuthService struct {
	v1pb.UnimplementedAuthServiceServer
	store          *store.Store
	secret         string
	metricReporter *metricreport.Reporter
	profile        *config.Profile
}

// NewAuthService creates a new AuthService.
func NewAuthService(store *store.Store, secret string, metricReporter *metricreport.Reporter, profile *config.Profile) *AuthService {
	return &AuthService{
		store:          store,
		secret:         secret,
		metricReporter: metricReporter,
		profile:        profile,
	}
}

// GetUser gets a user.
func (s *AuthService) GetUser(ctx context.Context, request *v1pb.GetUserRequest) (*v1pb.User, error) {
	userID, err := getUserID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	return convertToUser(user), nil
}

// ListUsers lists all users.
func (s *AuthService) ListUsers(ctx context.Context, request *v1pb.ListUsersRequest) (*v1pb.ListUsersResponse, error) {
	users, err := s.store.ListUsers(ctx, &store.FindUserMessage{ShowDeleted: request.ShowDeleted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user, error: %v", err)
	}
	response := &v1pb.ListUsersResponse{}
	for _, user := range users {
		response.Users = append(response.Users, convertToUser(user))
	}
	return response, nil
}

// CreateUser creates a user.
func (s *AuthService) CreateUser(ctx context.Context, request *v1pb.CreateUserRequest) (*v1pb.User, error) {
	if request.User == nil {
		return nil, status.Errorf(codes.InvalidArgument, "user must be set")
	}
	if request.User.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email must be set")
	}
	if _, err := mail.ParseAddress(request.User.Email); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email %q address", request.User.Email)
	}
	if request.User.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user title must be set")
	}
	if request.User.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password must be set")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.User.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate password hash, error: %v", err)
	}

	user, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        request.User.Email,
		Name:         request.User.Title,
		Type:         api.EndUser,
		PasswordHash: string(passwordHash),
	}, api.SystemBotID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user, error: %v", err)
	}

	if user.ID == api.PrincipalIDForFirstUser && s.metricReporter != nil {
		s.metricReporter.Report(&metric.Metric{
			Name:  metricAPI.FirstPrincipalMetricName,
			Value: 1,
			Labels: map[string]interface{}{
				"email":         user.Email,
				"name":          user.Name,
				"lark_notified": false,
			},
		})
	}
	bytes, err := json.Marshal(api.ActivityMemberCreatePayload{
		PrincipalID:    user.ID,
		PrincipalName:  user.Name,
		PrincipalEmail: user.Email,
		MemberStatus:   api.Active,
		Role:           user.Role,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to construct activity payload, error: %v", err)
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   user.ID,
		ContainerID: user.ID,
		Type:        api.ActivityMemberCreate,
		Level:       api.ActivityInfo,
		Payload:     string(bytes),
	}
	if _, err := s.store.CreateActivity(ctx, activityCreate); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create activity, error: %v", err)
	}
	return convertToUser(user), nil
}

// UpdateUser updates a user.
func (s *AuthService) UpdateUser(ctx context.Context, request *v1pb.UpdateUserRequest) (*v1pb.User, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.User == nil {
		return nil, status.Errorf(codes.InvalidArgument, "user must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	userID, err := getUserID(request.User.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	if user.MemberDeleted {
		return nil, status.Errorf(codes.InvalidArgument, "user %q has been deleted", userID)
	}

	role := ctx.Value(common.RoleContextKey).(api.Role)
	if principalID != userID && role != api.Owner {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner or user itself can update the user %d", userID)
	}

	patch := &store.UpdateUserMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "user.email":
			if _, err := mail.ParseAddress(request.User.Email); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid email address %q", request.User.Email)
			}
			patch.Email = &request.User.Email
		case "user.title":
			patch.Name = &request.User.Title
		case "user.password":
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.User.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to generate password hash, error: %v", err)
			}
			passwordHashStr := string(passwordHash)
			patch.PasswordHash = &passwordHashStr
		case "user.role":
			if role != api.Owner {
				return nil, status.Errorf(codes.PermissionDenied, "only workspace owner can update user role")
			}
			userRole := convertUserRole(request.User.UserRole)
			if userRole == api.UnknownRole {
				return nil, status.Errorf(codes.InvalidArgument, "invalid user role %s", request.User.UserRole)
			}
			patch.Role = &userRole
		}
	}

	user, err = s.store.UpdateUser(ctx, userID, patch, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user, error: %v", err)
	}
	return convertToUser(user), nil
}

// DeleteUser deletes a user.
func (s *AuthService) DeleteUser(ctx context.Context, request *v1pb.DeleteUserRequest) (*emptypb.Empty, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	userID, err := getUserID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	if user.MemberDeleted {
		return nil, status.Errorf(codes.InvalidArgument, "user %q has been deleted", userID)
	}

	role := ctx.Value(common.RoleContextKey).(api.Role)
	if role != api.Owner {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner can delete the user %d", userID)
	}

	if _, err := s.store.UpdateUser(ctx, userID, &store.UpdateUserMessage{Delete: &deletePatch}, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// UndeleteUser undeletes a user.
func (s *AuthService) UndeleteUser(ctx context.Context, request *v1pb.UndeleteUserRequest) (*v1pb.User, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	userID, err := getUserID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
	}
	if !user.MemberDeleted {
		return nil, status.Errorf(codes.InvalidArgument, "user %q is already active", userID)
	}

	role := ctx.Value(common.RoleContextKey).(api.Role)
	if role != api.Owner {
		return nil, status.Errorf(codes.PermissionDenied, "only workspace owner can undelete the user %d", userID)
	}

	user, err = s.store.UpdateUser(ctx, userID, &store.UpdateUserMessage{Delete: &undeletePatch}, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToUser(user), nil
}

func convertToUser(user *store.UserMessage) *v1pb.User {
	role := v1pb.UserRole_USER_ROLE_UNSPECIFIED
	switch user.Role {
	case api.Owner:
		role = v1pb.UserRole_OWNER
	case api.DBA:
		role = v1pb.UserRole_DBA
	case api.Developer:
		role = v1pb.UserRole_DEVELOPER
	}
	userType := v1pb.UserType_USER_TYPE_UNSPECIFIED
	switch user.Type {
	case api.EndUser:
		userType = v1pb.UserType_USER
	case api.SystemBot:
		userType = v1pb.UserType_SYSTEM_BOT
	case api.ServiceAccount:
		userType = v1pb.UserType_SERVICE_ACCOUNT
	}

	convertedUser := &v1pb.User{
		Name:     fmt.Sprintf("%s%d", userNamePrefix, user.ID),
		State:    convertDeletedToState(user.MemberDeleted),
		Email:    user.Email,
		Title:    user.Name,
		UserType: userType,
		UserRole: role,
	}
	if common.FeatureFlag(common.FeatureFlagNoop) {
		if user.IdentityProviderResourceID != nil {
			convertedUser.IdentityProvider = fmt.Sprintf("%s%s", identityProviderNamePrefix, *user.IdentityProviderResourceID)
		}
	}
	return convertedUser
}

func convertUserRole(userRole v1pb.UserRole) api.Role {
	switch userRole {
	case v1pb.UserRole_OWNER:
		return api.Owner
	case v1pb.UserRole_DBA:
		return api.DBA
	case v1pb.UserRole_DEVELOPER:
		return api.Developer
	}
	return api.UnknownRole
}

// Login is the auth login method.
func (s *AuthService) Login(ctx context.Context, request *v1pb.LoginRequest) (*v1pb.LoginResponse, error) {
	user, err := s.store.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get principal by email %q", request.Email)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user %q not found", request.Email)
	}
	if user.MemberDeleted {
		return nil, status.Errorf(codes.Unauthenticated, "user %q has been deactivated by administrators", request.Email)
	}
	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		// If the two passwords don't match, return a 401 status.
		return nil, status.Errorf(codes.InvalidArgument, "incorrect password")
	}

	var accessToken string
	if request.Web {
		token, err := auth.GenerateAccessToken(user.Name, user.ID, s.profile.Mode, s.secret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate API access token")
		}
		accessToken = token
		refreshToken, err := auth.GenerateRefreshToken(user.Name, user.ID, s.profile.Mode, s.secret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate API access token")
		}

		if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
			auth.GatewayMetadataAccessTokenKey:  accessToken,
			auth.GatewayMetadataRefreshTokenKey: refreshToken,
			auth.GatewayMetadataUserIDKey:       fmt.Sprintf("%d", user.ID),
		})); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to set grpc header, error: %v", err)
		}
	} else {
		token, err := auth.GenerateAPIToken(user.Name, user.ID, s.profile.Mode, s.secret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate API access token")
		}
		accessToken = token
	}
	return &v1pb.LoginResponse{
		Token: accessToken,
	}, nil
}

// LoginWithIdentityProvider is the auth login method with an identity provider.
func (s *AuthService) LoginWithIdentityProvider(ctx context.Context, request *v1pb.LoginWithIdentityProviderRequest) (*v1pb.LoginResponse, error) {
	idpID, _ := getIdentityProviderID(request.IdpName)
	idp, err := s.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &idpID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get identity provider")
	}
	if idp == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider not found")
	}

	var userInfo *storepb.IdentityProviderUserInfo
	if idp.Type == storepb.IdentityProviderType_OAUTH2 {
		oauth2Context := request.Context.GetOauth2Context()
		if oauth2Context == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing OAuth2 context")
		}
		oauth2IdentityProvider, err := oauth2.NewIdentityProvider(idp.Config.GetOauth2Config())
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "failed to new oauth2 identity provider")
		}
		redirectURL := fmt.Sprintf("%s/oauth/callback", s.profile.ExternalURL)
		token, err := oauth2IdentityProvider.ExchangeToken(ctx, redirectURL, oauth2Context.Code)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		userInfo, err = oauth2IdentityProvider.UserInfo(token)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider type %s not supported", idp.Type.String())
	}

	if userInfo == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider user info not found")
	}
	user, err := s.store.GetUser(ctx, &store.FindUserMessage{
		IdentityProviderID:       &idp.UID,
		IdentityProviderUserInfo: fmt.Sprintf("principal.idp_user_info->>'identifier' = '%s'", userInfo.Identifier),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get principal")
	}
	if user == nil {
		// Generate random password for new users from identity provider.
		password, err := common.RandomString(20)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate random password")
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate password hash")
		}
		newUser, err := s.store.CreateUser(ctx, &store.UserMessage{
			Name:                       userInfo.DisplayName,
			Email:                      userInfo.Email,
			Type:                       api.EndUser,
			PasswordHash:               string(passwordHash),
			IdentityProviderResourceID: &idp.ResourceID,
			IdentityProviderUserInfo:   userInfo,
		}, api.SystemBotID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create user, error: %v", err)
		}
		user = newUser
	} else if user.MemberDeleted {
		return nil, status.Errorf(codes.Unauthenticated, "user has been deactivated by administrators")
	}

	var accessToken string
	token, err := auth.GenerateAccessToken(user.Name, user.ID, s.profile.Mode, s.secret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate API access token")
	}
	accessToken = token
	refreshToken, err := auth.GenerateRefreshToken(user.Name, user.ID, s.profile.Mode, s.secret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate API access token")
	}

	if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
		auth.GatewayMetadataAccessTokenKey:  accessToken,
		auth.GatewayMetadataRefreshTokenKey: refreshToken,
		auth.GatewayMetadataUserIDKey:       fmt.Sprintf("%d", user.ID),
	})); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set grpc header, error: %v", err)
	}
	return &v1pb.LoginResponse{
		Token: accessToken,
	}, nil
}

// Logout is the auth logtou method.
func (*AuthService) Logout(ctx context.Context, _ *v1pb.LogoutRequest) (*emptypb.Empty, error) {
	if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
		auth.GatewayMetadataAccessTokenKey:  "",
		auth.GatewayMetadataRefreshTokenKey: "",
		auth.GatewayMetadataUserIDKey:       "",
	})); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set grpc header, error: %v", err)
	}
	return &emptypb.Empty{}, nil
}
