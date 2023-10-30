// Package auth handles the auth of gRPC server.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v4"
	errs "github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	issuer = "bytebase"
	// Signing key section. For now, this is only used for signing, not for verifying since we only
	// have 1 version. But it will be used to maintain backward compatibility if we change the signing mechanism.
	keyID = "v1"
	// AccessTokenAudienceFmt is the format of the acccess token audience.
	AccessTokenAudienceFmt = "bb.user.access.%s"
	// MFATempTokenAudienceFmt is the format of the MFA temp token audience.
	MFATempTokenAudienceFmt = "bb.user.mfa-temp.%s"
	apiTokenDuration        = 1 * time.Hour
	// DefaultTokenDuration is the default token expiration duration.
	DefaultTokenDuration = 7 * 24 * time.Hour

	// AccessTokenCookieName is the cookie name of access token.
	AccessTokenCookieName = "access-token"
	// UserIDCookieName is the cookie name of user ID.
	UserIDCookieName = "user"

	// GatewayMetadataAccessTokenKey is the gateway metadata key for access token.
	GatewayMetadataAccessTokenKey = "bytebase-access-token"
	// GatewayMetadataUserIDKey is the gateway metadata key for user ID.
	GatewayMetadataUserIDKey = "bytebase-user"
)

// APIAuthInterceptor is the auth interceptor for gRPC server.
type APIAuthInterceptor struct {
	store          *store.Store
	secret         string
	tokenDuration  time.Duration
	licenseService enterprise.LicenseService
	stateCfg       *state.State
	mode           common.ReleaseMode
}

// New returns a new API auth interceptor.
func New(store *store.Store, secret string, tokenDuration time.Duration, licenseService enterprise.LicenseService, stateCfg *state.State, mode common.ReleaseMode) *APIAuthInterceptor {
	return &APIAuthInterceptor{
		store:          store,
		secret:         secret,
		tokenDuration:  tokenDuration,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		mode:           mode,
	}
}

// AuthenticationInterceptor is the unary interceptor for gRPC API.
func (in *APIAuthInterceptor) AuthenticationInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse metadata from incoming context")
	}
	accessTokenStr, err := GetTokenFromMetadata(md)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}

	principalID, err := in.authenticate(ctx, accessTokenStr)
	if err != nil {
		if IsAuthenticationAllowed(serverInfo.FullMethod) {
			return handler(ctx, request)
		}
		return nil, err
	}

	// Stores principalID into context.
	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, principalID)
	return handler(childCtx, request)
}

// AuthenticationStreamInterceptor is the unary interceptor for gRPC API.
func (in *APIAuthInterceptor) AuthenticationStreamInterceptor(request any, ss grpc.ServerStream, serverInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "failed to parse metadata from incoming context")
	}
	accessTokenStr, err := GetTokenFromMetadata(md)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, err.Error())
	}

	principalID, err := in.authenticate(ctx, accessTokenStr)
	if err != nil {
		if IsAuthenticationAllowed(serverInfo.FullMethod) {
			return handler(request, ss)
		}
		return err
	}

	// Stores principalID into context.
	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, principalID)
	sss := overrideStream{ServerStream: ss, childCtx: childCtx}
	return handler(request, sss)
}

type overrideStream struct {
	childCtx context.Context
	grpc.ServerStream
}

func (s overrideStream) Context() context.Context {
	return s.childCtx
}

func (in *APIAuthInterceptor) authenticate(ctx context.Context, accessTokenStr string) (int, error) {
	if accessTokenStr == "" {
		return 0, status.Errorf(codes.Unauthenticated, "access token not found")
	}
	if _, ok := in.stateCfg.ExpireCache.Get(accessTokenStr); ok {
		return 0, status.Errorf(codes.Unauthenticated, "access token expired")
	}
	claims := &claimsMessage{}
	if _, err := jwt.ParseWithClaims(accessTokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, status.Errorf(codes.Unauthenticated, "unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(in.secret), nil
			}
		}
		return nil, status.Errorf(codes.Unauthenticated, "unexpected access token kid=%v", t.Header["kid"])
	}); err != nil {
		var ve *jwt.ValidationError
		if errors.As(err, &ve) && ve.Errors == jwt.ValidationErrorExpired {
			return 0, status.Errorf(codes.Unauthenticated, "access token expired")
		}
		return 0, status.Errorf(codes.Unauthenticated, "failed to parse claim")
	}
	if !audienceContains(claims.Audience, fmt.Sprintf(AccessTokenAudienceFmt, in.mode)) {
		return 0, status.Errorf(codes.Unauthenticated,
			"invalid access token, audience mismatch, got %q, expected %q. you may send request to the wrong environment",
			claims.Audience,
			fmt.Sprintf(AccessTokenAudienceFmt, in.mode),
		)
	}

	principalID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, status.Errorf(codes.Unauthenticated, "malformed ID %q in the access token", claims.Subject)
	}
	user, err := in.store.GetUserByID(ctx, principalID)
	if err != nil {
		return 0, status.Errorf(codes.Unauthenticated, "failed to find user ID %q in the access token", principalID)
	}
	if user == nil {
		return 0, status.Errorf(codes.Unauthenticated, "user ID %q not exists in the access token", principalID)
	}
	if user.MemberDeleted {
		return 0, status.Errorf(codes.Unauthenticated, "user ID %q has been deactivated by administrators", principalID)
	}

	return principalID, nil
}

// GetUserIDFromMFATempToken returns the user ID from the MFA temp token.
func GetUserIDFromMFATempToken(token string, mode common.ReleaseMode, secret string) (int, error) {
	claims := &claimsMessage{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, status.Errorf(codes.Unauthenticated, "unexpected MFA temp token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(secret), nil
			}
		}
		return nil, status.Errorf(codes.Unauthenticated, "unexpected MFA temp token kid=%v", t.Header["kid"])
	})
	if err != nil {
		return 0, status.Errorf(codes.Unauthenticated, "failed to parse claim")
	}
	if !audienceContains(claims.Audience, fmt.Sprintf(MFATempTokenAudienceFmt, mode)) {
		return 0, status.Errorf(codes.Unauthenticated, "invalid MFA temp token, audience mismatch")
	}
	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, status.Errorf(codes.Unauthenticated, "malformed ID %q in the MFA temp token", claims.Subject)
	}
	return userID, nil
}

func GetTokenFromMetadata(md metadata.MD) (string, error) {
	authorizationHeaders := md.Get("Authorization")
	if len(md.Get("Authorization")) > 0 {
		authHeaderParts := strings.Fields(authorizationHeaders[0])
		if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
			return "", errs.Errorf("authorization header format must be Bearer {token}")
		}
		return authHeaderParts[1], nil
	}
	// check the HTTP cookie
	var accessToken string
	for _, t := range append(md.Get("grpcgateway-cookie"), md.Get("cookie")...) {
		header := http.Header{}
		header.Add("Cookie", t)
		request := http.Request{Header: header}
		if v, _ := request.Cookie(AccessTokenCookieName); v != nil {
			accessToken = v.Value
		}
	}
	return accessToken, nil
}

func audienceContains(audience jwt.ClaimStrings, token string) bool {
	for _, v := range audience {
		if v == token {
			return true
		}
	}
	return false
}

type claimsMessage struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

// GenerateAPIToken generates an API token.
func GenerateAPIToken(userName string, userID int, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(apiTokenDuration)
	return generateToken(userName, userID, fmt.Sprintf(AccessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// GenerateAccessToken generates an access token for web.
func GenerateAccessToken(userName string, userID int, mode common.ReleaseMode, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userName, userID, fmt.Sprintf(AccessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// GenerateMFATempToken generates a temporary token for MFA.
func GenerateMFATempToken(userName string, userID int, mode common.ReleaseMode, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userName, userID, fmt.Sprintf(MFATempTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// Pay attention to this function. It holds the main JWT token generation logic.
func generateToken(userName string, userID int, aud string, expirationTime time.Time, secret []byte) (string, error) {
	// Create the JWT claims, which includes the username and expiry time.
	claims := &claimsMessage{
		Name: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{aud},
			// In JWT, the expiry time is expressed as unix milliseconds.
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
			Subject:   strconv.Itoa(userID),
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
