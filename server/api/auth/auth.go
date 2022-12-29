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

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/store"
)

const (
	issuer = "bytebase"
	// Signing key section. For now, this is only used for signing, not for verifying since we only
	// have 1 version. But it will be used to maintain backward compatibility if we change the signing mechanism.
	keyID = "v1"
	// AccessTokenAudienceFmt is the format of the acccess token audience.
	AccessTokenAudienceFmt = "bb.user.access.%s"
	// RefreshTokenAudienceFmt is the format of the refresh token audience.
	RefreshTokenAudienceFmt = "bb.user.refresh.%s"
	apiTokenDuration        = 2 * time.Hour
	accessTokenDuration     = 24 * time.Hour
	refreshTokenDuration    = 7 * 24 * time.Hour
	// RefreshThresholdDuration is the threshold duration for refreshing token.
	RefreshThresholdDuration = 1 * time.Hour

	// CookieExpDuration expires slightly earlier than the jwt expiration. Client would be logged out if the user
	// cookie expires, thus the client would always logout first before attempting to make a request with the expired jwt.
	// Suppose we have a valid refresh token, we will refresh the token in 2 cases:
	// 1. The access token is about to expire in <<refreshThresholdDuration>>
	// 2. The access token has already expired, we refresh the token so that the ongoing request can pass through.
	CookieExpDuration = refreshTokenDuration - 1*time.Minute
	// AccessTokenCookieName is the cookie name of access token.
	AccessTokenCookieName = "access-token"
	// RefreshTokenCookieName is the cookie name of refresh token.
	RefreshTokenCookieName = "refresh-token"
	// UserIDCookieName is the cookie name of user ID.
	UserIDCookieName = "user"

	// GatewayMetadataAccessTokenKey is the gateway metadata key for access token.
	GatewayMetadataAccessTokenKey = "bytebase-access-token"
	// GatewayMetadataRefreshTokenKey is the gateway metadata key for refresh token.
	GatewayMetadataRefreshTokenKey = "bytebase-refresh-token"
	// GatewayMetadataUserIDKey is the gateway metadata key for user ID.
	GatewayMetadataUserIDKey = "bytebase-user"
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
	methodName := getShortMethodName(serverInfo.FullMethod)
	if isAuthenticationAllowed(methodName) {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "failed to parse metadata from incoming context")
	}
	accessTokenStr, refreshTokenStr, err := getTokenFromMetadata(md)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	claims := &claimsMessage{}
	generateToken := false
	accessToken, err := jwt.ParseWithClaims(accessTokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, status.Errorf(codes.Unauthenticated, "unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(in.secret), nil
			}
		}
		return nil, status.Errorf(codes.Unauthenticated, "unexpected access token kid=%v", t.Header["kid"])
	})
	if err != nil {
		var ve *jwt.ValidationError
		if errors.As(err, &ve) && ve.Errors == jwt.ValidationErrorExpired {
			// If expiration error is the only error, we will clear the err
			// and generate new access token and refresh token
			if refreshTokenStr == "" {
				return nil, status.Errorf(codes.Unauthenticated, "access token is expired")
			}
			generateToken = true
		} else {
			return nil, status.Errorf(codes.Unauthenticated, "failed to parse claim")
		}
	}
	if !audienceContains(claims.Audience, fmt.Sprintf(AccessTokenAudienceFmt, in.mode)) {
		return nil, status.Errorf(codes.Unauthenticated,
			"invalid access token, audience mismatch, got %q, expected %q. you may send request to the wrong environment",
			claims.Audience,
			fmt.Sprintf(AccessTokenAudienceFmt, in.mode),
		)
	}
	if time.Until(claims.ExpiresAt.Time) < RefreshThresholdDuration {
		generateToken = true
	}

	principalID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "malformed ID %q in the access token", claims.Subject)
	}
	user, err := in.store.GetPrincipalByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to find user ID %q in the access token", principalID)
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user ID %q not exists in the access token", principalID)
	}

	if generateToken {
		generateTokenFunc := func() error {
			// Parses token and checks if it's valid.
			refreshTokenClaims := &claimsMessage{}
			refreshToken, err := jwt.ParseWithClaims(refreshTokenStr, refreshTokenClaims, func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() != jwt.SigningMethodHS256.Name {
					return nil, status.Errorf(codes.Unauthenticated, "unexpected refresh token signing method=%v, expected %v", t.Header["alg"], jwt.SigningMethodHS256)
				}

				if kid, ok := t.Header["kid"].(string); ok {
					if kid == "v1" {
						return []byte(in.secret), nil
					}
				}
				return nil, errs.Errorf("unexpected refresh token kid=%v", t.Header["kid"])
			})
			if err != nil {
				if err == jwt.ErrSignatureInvalid {
					return errs.Errorf("failed to generate access token: invalid refresh token signature")
				}
				return errs.Errorf("Server error to refresh expired token, user ID %d", principalID)
			}

			if !audienceContains(refreshTokenClaims.Audience, fmt.Sprintf(RefreshTokenAudienceFmt, in.mode)) {
				return errs.Errorf("Invalid refresh token, audience mismatch, got %q, expected %q. you may send request to the wrong environment",
					refreshTokenClaims.Audience,
					fmt.Sprintf(RefreshTokenAudienceFmt, in.mode),
				)
			}

			// If we have a valid refresh token, we will generate new access token and refresh token
			if refreshToken != nil && refreshToken.Valid {
				if err := generateTokensAndSetCookies(ctx, user, in.mode, in.secret); err != nil {
					return errs.Wrapf(err, "failed to regenerate token")
				}
			}

			return nil
		}

		// It may happen that we still have a valid access token, but we encounter issue when trying to generate new token
		// In such case, we won't return the error.
		if err := generateTokenFunc(); err != nil && !accessToken.Valid {
			return nil, status.Errorf(codes.Unauthenticated, err.Error())
		}
	}

	// Stores principalID into context.
	childCtx := context.WithValue(ctx, common.PrincipalIDContextKey, principalID)
	return handler(childCtx, req)
}

func getTokenFromMetadata(md metadata.MD) (string, string, error) {
	authorizationHeaders := md.Get("Authorization")
	if len(md.Get("Authorization")) > 0 {
		authHeaderParts := strings.Fields(authorizationHeaders[0])
		if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
			return "", "", errs.Errorf("authorization header format must be Bearer {token}")
		}
		return authHeaderParts[1], "", nil
	}
	// check the HTTP cookie
	var accessToken, refreshToken string
	for _, t := range md["grpcgateway-cookie"] {
		header := http.Header{}
		header.Add("Cookie", t)
		request := http.Request{Header: header}
		if v, _ := request.Cookie(AccessTokenCookieName); v != nil {
			accessToken = v.Value
		}
		if v, _ := request.Cookie(RefreshTokenCookieName); v != nil {
			refreshToken = v.Value
		}
	}
	if accessToken != "" && refreshToken != "" {
		return accessToken, refreshToken, nil
	}
	return "", "", errs.Errorf("access token not found from metadata")
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

// generateTokensAndSetCookies generates jwt token and saves it to the http-only cookie.
func generateTokensAndSetCookies(ctx context.Context, user *api.Principal, mode common.ReleaseMode, secret string) error {
	accessToken, err := GenerateAccessToken(user, mode, secret)
	if err != nil {
		return errs.Wrap(err, "failed to generate access token")
	}
	// We generate here a new refresh token and saving it to the cookie.
	refreshToken, err := GenerateRefreshToken(user, mode, secret)
	if err != nil {
		return errs.Wrap(err, "failed to generate refresh token")
	}

	if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
		GatewayMetadataAccessTokenKey:  accessToken,
		GatewayMetadataRefreshTokenKey: refreshToken,
		GatewayMetadataUserIDKey:       fmt.Sprintf("%d", user.ID),
	})); err != nil {
		return errs.Wrapf(err, "failed to set grpc header")
	}
	return nil
}

// GenerateAPIToken generates an API token.
func GenerateAPIToken(user *api.Principal, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(apiTokenDuration)
	return generateToken(user, fmt.Sprintf(AccessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// GenerateAccessToken generates an access token for web.
func GenerateAccessToken(user *api.Principal, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(accessTokenDuration)
	return generateToken(user, fmt.Sprintf(AccessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// GenerateRefreshToken generates a refresh token for web.
func GenerateRefreshToken(user *api.Principal, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(refreshTokenDuration)
	return generateToken(user, fmt.Sprintf(RefreshTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// Pay attention to this function. It holds the main JWT token generation logic.
func generateToken(user *api.Principal, aud string, expirationTime time.Time, secret []byte) (string, error) {
	// Create the JWT claims, which includes the username and expiry time.
	claims := &claimsMessage{
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
