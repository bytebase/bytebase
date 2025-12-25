// Package auth handles the auth of gRPC server.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/golang-jwt/jwt/v5"
	errs "github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
	// OAuth2AccessTokenAudience is the audience for OAuth2 access tokens.
	OAuth2AccessTokenAudience = "bb.oauth2.access"
	apiTokenDuration          = 1 * time.Hour
	// DefaultTokenDuration is the default token expiration duration.
	DefaultTokenDuration = 7 * 24 * time.Hour

	// AccessTokenCookieName is the cookie name of access token.
	AccessTokenCookieName = "access-token"
)

// APIAuthInterceptor is the auth interceptor for gRPC server.
type APIAuthInterceptor struct {
	store          *store.Store
	secret         string
	licenseService *enterprise.LicenseService
	stateCfg       *state.State
	profile        *config.Profile
}

// New returns a new API auth interceptor.
func New(
	store *store.Store,
	secret string,
	licenseService *enterprise.LicenseService,
	stateCfg *state.State,
	profile *config.Profile,
) *APIAuthInterceptor {
	return &APIAuthInterceptor{
		store:          store,
		secret:         secret,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		profile:        profile,
	}
}

// WrapUnary implements the ConnectRPC interceptor interface for unary RPCs.
func (in *APIAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		accessTokenStr, err := GetTokenFromHeaders(req.Header())
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		authContext, err := getAuthContext(req.Spec().Procedure)
		if err != nil {
			return nil, err
		}
		ctx = context.WithValue(ctx, common.AuthContextKey, authContext)

		user, err := in.getUserConnect(ctx, accessTokenStr)
		if err != nil {
			if IsAuthenticationAllowed(req.Spec().Procedure, authContext) {
				return next(ctx, req)
			}
			return nil, err
		}

		ctx = context.WithValue(ctx, common.UserContextKey, user)
		return next(ctx, req)
	}
}

// WrapStreamingClient implements the ConnectRPC interceptor interface for streaming clients.
func (*APIAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

// WrapStreamingHandler implements the ConnectRPC interceptor interface for streaming handlers.
func (in *APIAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		accessTokenStr, err := GetTokenFromHeaders(conn.RequestHeader())
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}

		authContext, err := getAuthContext(conn.Spec().Procedure)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, common.AuthContextKey, authContext)

		user, err := in.getUserConnect(ctx, accessTokenStr)
		if err != nil {
			if IsAuthenticationAllowed(conn.Spec().Procedure, authContext) {
				return next(ctx, conn)
			}
			return err
		}

		ctx = context.WithValue(ctx, common.UserContextKey, user)

		return next(ctx, conn)
	}
}

// authenticate is the shared authentication logic that validates JWT tokens.
// Returns the user and claims, or an error. This is the single source of truth for token validation.
func (in *APIAuthInterceptor) authenticate(ctx context.Context, accessTokenStr string) (*store.UserMessage, *claimsMessage, error) {
	if accessTokenStr == "" {
		return nil, nil, errs.New("access token not found")
	}
	if _, ok := in.stateCfg.ExpireCache.Get(accessTokenStr); ok {
		return nil, nil, errs.New("access token expired")
	}

	claims := &claimsMessage{}
	if _, err := jwt.ParseWithClaims(accessTokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errs.Errorf("unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(in.secret), nil
			}
		}
		return nil, errs.Errorf("unexpected access token kid=%v", t.Header["kid"])
	}); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, nil, errs.New("access token expired")
		}
		return nil, nil, errs.New("failed to parse claim")
	}

	// Accept both user access tokens (bb.user.access.{mode}) and OAuth2 access tokens (bb.oauth2.access)
	validUserAudience := fmt.Sprintf(AccessTokenAudienceFmt, in.profile.Mode)
	if !audienceContains(claims.Audience, validUserAudience) && !audienceContains(claims.Audience, OAuth2AccessTokenAudience) {
		return nil, nil, errs.Errorf(
			"invalid access token, audience mismatch, got %q, expected %q or %q. you may send request to the wrong environment",
			claims.Audience,
			validUserAudience,
			OAuth2AccessTokenAudience,
		)
	}

	// Try to parse subject as integer ID (old tokens), otherwise treat as email (new tokens and OAuth2)
	var user *store.UserMessage
	var err error
	if principalID, parseErr := strconv.Atoi(claims.Subject); parseErr == nil {
		// Old token with numeric ID as subject
		user, err = in.store.GetUserByID(ctx, principalID)
		if err != nil {
			return nil, nil, errs.Errorf("failed to find user ID %q in the access token", principalID)
		}
		if user == nil {
			return nil, nil, errs.Errorf("user ID %q not exists in the access token", principalID)
		}
	} else {
		// New token with email as subject (includes OAuth2 tokens)
		user, err = in.store.GetUserByEmail(ctx, claims.Subject)
		if err != nil {
			return nil, nil, errs.Errorf("failed to find user email %q in the access token", claims.Subject)
		}
		if user == nil {
			return nil, nil, errs.Errorf("user email %q not exists in the access token", claims.Subject)
		}
	}
	if user.MemberDeleted {
		return nil, nil, errs.Errorf("user ID %q has been deactivated by administrators", user.ID)
	}

	return user, claims, nil
}

// authenticateConnect is a ConnectRPC-specific version that returns ConnectRPC errors.
func (in *APIAuthInterceptor) authenticateConnect(ctx context.Context, accessTokenStr string) (*store.UserMessage, error) {
	user, _, err := in.authenticate(ctx, accessTokenStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return user, nil
}

// getUserConnect is a ConnectRPC-specific version that returns ConnectRPC errors.
func (in *APIAuthInterceptor) getUserConnect(ctx context.Context, accessTokenStr string) (*store.UserMessage, error) {
	user, err := in.authenticateConnect(ctx, accessTokenStr)
	if err != nil {
		return nil, err
	}

	// Only update for authorized request.
	in.profile.LastActiveTS.Store(time.Now().Unix())
	return user, nil
}

// GetUserEmailFromMFATempToken returns the user email from the MFA temp token.
func GetUserEmailFromMFATempToken(token string, mode common.ReleaseMode, secret string) (string, error) {
	claims := &claimsMessage{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, connect.NewError(connect.CodeUnauthenticated, errs.Errorf("unexpected MFA temp token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256))
		}
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(secret), nil
			}
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, errs.Errorf("unexpected MFA temp token kid=%v", t.Header["kid"]))
	})
	if err != nil {
		return "", connect.NewError(connect.CodeUnauthenticated, errs.New("failed to parse claim"))
	}
	if !audienceContains(claims.Audience, fmt.Sprintf(MFATempTokenAudienceFmt, mode)) {
		return "", connect.NewError(connect.CodeUnauthenticated, errs.New("invalid MFA temp token, audience mismatch"))
	}
	return claims.Subject, nil
}

// AuthenticateToken validates a JWT access token and returns the user and token expiry.
// This is a non-ConnectRPC version that returns regular errors instead of ConnectRPC errors.
func (in *APIAuthInterceptor) AuthenticateToken(ctx context.Context, accessTokenStr string) (*store.UserMessage, time.Time, error) {
	user, claims, err := in.authenticate(ctx, accessTokenStr)
	if err != nil {
		return nil, time.Time{}, err
	}

	var tokenExpiry time.Time
	if claims.ExpiresAt != nil {
		tokenExpiry = claims.ExpiresAt.Time
	}

	return user, tokenExpiry, nil
}

// GetTokenFromHeaders extracts the access token from HTTP headers for ConnectRPC.
func GetTokenFromHeaders(headers http.Header) (string, error) {
	// Check Authorization header first
	authHeader := headers.Get("Authorization")
	if authHeader != "" {
		authHeaderParts := strings.Fields(authHeader)
		if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
			return "", errs.Errorf("authorization header format must be Bearer {token}")
		}
		return authHeaderParts[1], nil
	}

	// Check HTTP cookies
	var accessToken string
	for _, cookieHeader := range headers.Values("cookie") {
		header := http.Header{}
		header.Add("Cookie", cookieHeader)
		request := http.Request{Header: header}
		if cookie, _ := request.Cookie(AccessTokenCookieName); cookie != nil {
			accessToken = cookie.Value
			break
		}
	}
	return accessToken, nil
}

func audienceContains(audience jwt.ClaimStrings, token string) bool {
	return slices.Contains(audience, token)
}

type claimsMessage struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

// GenerateAPIToken generates an API token.
func GenerateAPIToken(userEmail string, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(apiTokenDuration)
	return generateToken(userEmail, fmt.Sprintf(AccessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// GenerateAccessToken generates an access token for web.
func GenerateAccessToken(userEmail string, mode common.ReleaseMode, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userEmail, fmt.Sprintf(AccessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// GenerateMFATempToken generates a temporary token for MFA.
func GenerateMFATempToken(userEmail string, mode common.ReleaseMode, secret string, tokenDuration time.Duration) (string, error) {
	expirationTime := time.Now().Add(tokenDuration)
	return generateToken(userEmail, fmt.Sprintf(MFATempTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// Pay attention to this function. It holds the main JWT token generation logic.
func generateToken(userEmail string, aud string, expirationTime time.Time, secret []byte) (string, error) {
	// Create the JWT claims, which includes the user email and expiry time.
	claims := &claimsMessage{
		Name: userEmail,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{aud},
			// In JWT, the expiry time is expressed as unix milliseconds.
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
			Subject:   userEmail,
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

func getAuthContext(fullMethod string) (*common.AuthContext, error) {
	methodTokens := strings.Split(fullMethod, "/")
	if len(methodTokens) != 3 {
		return nil, errs.Errorf("invalid full method name %q", fullMethod)
	}
	rd, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(methodTokens[1]))
	if err != nil {
		return nil, errs.Wrapf(err, "invalid registry service descriptor, full method name %q", fullMethod)
	}
	sd, ok := rd.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, errs.Errorf("invalid service descriptor, full method name %q", fullMethod)
	}
	md, ok := sd.Methods().ByName(protoreflect.Name(methodTokens[2])).Options().(*descriptorpb.MethodOptions)
	if !ok {
		return nil, errs.Errorf("invalid method options, full method name %q", fullMethod)
	}
	allowWithoutCredentialAny := proto.GetExtension(md, v1pb.E_AllowWithoutCredential)
	allowWithoutCredential, ok := allowWithoutCredentialAny.(bool)
	if !ok {
		return nil, errs.Errorf("invalid allow without credential extension, full method name %q", fullMethod)
	}
	permissionAny := proto.GetExtension(md, v1pb.E_Permission)
	permission, ok := permissionAny.(string)
	if !ok {
		return nil, errs.Errorf("invalid permission extension, full method name %q", fullMethod)
	}
	authMethodAny := proto.GetExtension(md, v1pb.E_AuthMethod)
	am, ok := authMethodAny.(v1pb.AuthMethod)
	if !ok {
		return nil, errs.Errorf("invalid auth method extension, full method name %q", fullMethod)
	}
	var authMethod common.AuthMethod
	switch am {
	case v1pb.AuthMethod_AUTH_METHOD_UNSPECIFIED:
		authMethod = common.AuthMethodUnspecified
	case v1pb.AuthMethod_IAM:
		authMethod = common.AuthMethodIAM
	case v1pb.AuthMethod_CUSTOM:
		authMethod = common.AuthMethodCustom
	default:
		return nil, errs.Errorf("unknown auth method %v for full method name %q", am, fullMethod)
	}
	auditAny := proto.GetExtension(md, v1pb.E_Audit)
	audit, ok := auditAny.(bool)
	if !ok {
		return nil, errs.Errorf("invalid audit extension, full method name %q", fullMethod)
	}

	return &common.AuthContext{
		AllowWithoutCredential: allowWithoutCredential,
		Permission:             permission,
		AuthMethod:             authMethod,
		Audit:                  audit,
	}, nil
}
