package v1

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/api"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/server/api/auth"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/store"
)

const userNamePrefix = "users/"

// AuthService implements the auth service.
type AuthService struct {
	v1pb.UnimplementedAuthServiceServer
	store   *store.Store
	secret  string
	profile *config.Profile
}

// NewAuthService creates a new AuthService.
func NewAuthService(store *store.Store, secret string, profile *config.Profile) *AuthService {
	return &AuthService{
		store:   store,
		secret:  secret,
		profile: profile,
	}
}

// CreateUser creates a user.
func (s *AuthService) CreateUser(ctx context.Context, request *v1pb.CreateUserRequest) (*v1pb.User, error) {
	if request.User == nil {
		return nil, status.Errorf(codes.InvalidArgument, "user must be set")
	}
	if request.User.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email must be set")
	}
	if request.User.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user title must be set")
	}
	if request.User.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password must be set")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.User.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate password hash, error %v", err)
	}

	user, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        request.User.Email,
		Name:         request.User.Title,
		Type:         api.EndUser,
		PasswordHash: string(passwordHash),
	}, api.SystemBotID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user, error %v", err)
	}
	return convertToUser(user), nil
}

func convertToUser(user *store.UserMessage) *v1pb.User {
	role := v1pb.UserRole_USER_ROLE_UNSPECIFIED
	switch user.Role {
	case api.Owner:
		role = v1pb.UserRole_USER_ROLE_OWNER
	case api.DBA:
		role = v1pb.UserRole_USER_ROLE_DBA
	case api.Developer:
		role = v1pb.UserRole_USER_ROLE_DEVELOPER
	}
	return &v1pb.User{
		Name:     fmt.Sprintf("%s%d", userNamePrefix, user.ID),
		State:    convertDeletedToState(user.MemberDeleted),
		Email:    user.Email,
		Title:    user.Name,
		UserRole: role,
	}
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
			return nil, status.Errorf(codes.Internal, "failed to set grpc header, error %v", err)
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

// Logout is the auth logtou method.
func (*AuthService) Logout(ctx context.Context, _ *v1pb.LogoutRequest) (*emptypb.Empty, error) {
	if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
		auth.GatewayMetadataAccessTokenKey:  "",
		auth.GatewayMetadataRefreshTokenKey: "",
		auth.GatewayMetadataUserIDKey:       "",
	})); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set grpc header, error %v", err)
	}
	return &emptypb.Empty{}, nil
}
