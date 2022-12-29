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

// Login is the auth login method.
func (s *AuthService) Login(ctx context.Context, request *v1pb.LoginRequest) (*v1pb.LoginResponse, error) {
	user, err := s.store.GetPrincipalByEmail(ctx, request.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get principal by email %q", request.Email)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %q not found", request.Email)
	}

	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		// If the two passwords don't match, return a 401 status.
		return nil, status.Errorf(codes.InvalidArgument, "incorrect password")
	}

	// Test the status of this user.
	member, err := s.store.GetMemberByPrincipalID(ctx, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user member")
	}
	if member == nil {
		return nil, status.Errorf(codes.NotFound, "member not found for email %q", request.Email)
	}
	if member.RowStatus == api.Archived {
		return nil, status.Errorf(codes.NotFound, "This user %q has been deactivated by administrators", request.Email)
	}

	var accessToken string
	if request.Web {
		token, err := auth.GenerateAccessToken(user, s.profile.Mode, s.secret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate API access token")
		}
		accessToken = token
		refreshToken, err := auth.GenerateRefreshToken(user, s.profile.Mode, s.secret)
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
		token, err := auth.GenerateAPIToken(user, s.profile.Mode, s.secret)
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
