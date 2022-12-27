// Package auth handles the auth of gRPC server.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/store"
)

const (
	issuer = "bytebase"
	// Signing key section. For now, this is only used for signing, not for verifying since we only
	// have 1 version. But it will be used to maintain backward compatibility if we change the signing mechanism.
	keyID                  = "v1"
	accessTokenAudienceFmt = "bb.user.access.%s"
	apiTokenDuration       = 2 * time.Hour
)

// APIAuthInterceptor is the auth interceptor for gRPC server.
type APIAuthInterceptor struct {
	store          *store.Store
	secret         string
	licenseService enterpriseAPI.LicenseService
	mode           common.ReleaseMode
}

// New returns a new API auth interceptor.
func New(store *store.Store, secret string, licenseService enterpriseAPI.LicenseService, mode common.ReleaseMode) *APIAuthInterceptor {
	return &APIAuthInterceptor{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		mode:           mode,
	}
}

// AuthenticationInterceptor is the unary interceptor for gRPC API.
func (in *APIAuthInterceptor) AuthenticationInterceptor(ctx context.Context, req interface{}, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// TODO(d): skips actuator, GET /subscription request, OpenAPI SQL endpoint.
	if isAuthenticationAllowed(serverInfo.FullMethod) {
		return handler(ctx, req)
	}

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

// GenerateAPIToken generates an API token.
func GenerateAPIToken(user *api.Principal, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(apiTokenDuration)
	return generateToken(user, fmt.Sprintf(accessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// Pay attention to this function. It holds the main JWT token generation logic.
func generateToken(user *api.Principal, aud string, expirationTime time.Time, secret []byte) (string, error) {
	// Create the JWT claims, which includes the username and expiry time.
	claims := &claims{
		Name: user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{aud},
			// In JWT, the expiry time is expressed as unix milliseconds.
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
			Subject:   strconv.Itoa(user.ID),
		},
	}

	// Declare the token with the HS256 algorithm used for signing, and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID

	// Create the JWT string.
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
