// Package auth handles the auth of gRPC server.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v4"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/store"
)

const (
	accessTokenAudienceFmt = "bb.user.access.%s"
)

// APIAuthInterceptor is the auth interceptor for gRPC server.
type APIAuthInterceptor struct {
	store  *store.Store
	secret string
	mode   common.ReleaseMode
}

// New returns a new API auth interceptor.
func New(store *store.Store, secret string, mode common.ReleaseMode) *APIAuthInterceptor {
	return &APIAuthInterceptor{
		store:  store,
		secret: secret,
		mode:   mode,
	}
}

// UnaryInterceptor is the unary interceptor for gRPC API.
func (in *APIAuthInterceptor) UnaryInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// TODO(d): skips auth, actuator, GET /subscription request, OpenAPI SQL endpoint.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "failed to parse metadata from incoming context")
	}
	if len(md.Get("Authorization")) == 0 {
		return nil, status.Error(codes.PermissionDenied, "failed to get authorization metadata from incoming context")
	}
	authorization := md.Get("Authorization")[0]
	authHeaderParts := strings.Fields(authorization)
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return nil, status.Error(codes.PermissionDenied, "authorization header format must be Bearer {token}")
	}
	token := authHeaderParts[1]

	claims := &claims{}
	if _, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, status.Errorf(codes.PermissionDenied, "unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(in.secret), nil
			}
		}
		return nil, status.Errorf(codes.PermissionDenied, "unexpected access token kid=%v", t.Header["kid"])
	}); err != nil {
		var ve *jwt.ValidationError
		if errors.As(err, &ve) {
			// If expiration error is the only error, we will clear the err
			// and generate new access token and refresh token
			if ve.Errors == jwt.ValidationErrorExpired {
				return nil, status.Errorf(codes.PermissionDenied, "access token is expired")
			}
		}
		return nil, status.Errorf(codes.PermissionDenied, "failed to parse claim")
	}
	if !audienceContains(claims.Audience, fmt.Sprintf(accessTokenAudienceFmt, in.mode)) {
		return nil, status.Errorf(codes.PermissionDenied,
			"invalid access token, audience mismatch, got %q, expected %q. you may send request to the wrong environment",
			claims.Audience,
			fmt.Sprintf(accessTokenAudienceFmt, in.mode),
		)
	}

	principalID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "malformed ID %q in the access token", claims.Subject)
	}
	user, err := in.store.GetPrincipalByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "failed to find user ID %q in the access token", principalID)
	}
	if user == nil {
		return nil, status.Errorf(codes.PermissionDenied, "user ID %q not exists in the access token", principalID)
	}
	// Stores principalID into context.
	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, principalID)
	return handler(childCtx, req)
}

func audienceContains(audience jwt.ClaimStrings, token string) bool {
	for _, v := range audience {
		if v == token {
			return true
		}
	}
	return false
}

type claims struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}
