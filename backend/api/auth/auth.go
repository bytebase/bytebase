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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/golang-jwt/jwt/v5"
	errs "github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	// GatewayMetadataRequestOriginKey is the gateway metadata key for the request origin header.
	GatewayMetadataRequestOriginKey = "bytebase-request-origin"
)

// APIAuthInterceptor is the auth interceptor for gRPC server.
type APIAuthInterceptor struct {
	store          *store.Store
	secret         string
	tokenDuration  time.Duration
	licenseService enterprise.LicenseService
	stateCfg       *state.State
	profile        *config.Profile
}

// New returns a new API auth interceptor.
func New(
	store *store.Store,
	secret string,
	tokenDuration time.Duration,
	licenseService enterprise.LicenseService,
	stateCfg *state.State,
	profile *config.Profile,
) *APIAuthInterceptor {
	return &APIAuthInterceptor{
		store:          store,
		secret:         secret,
		tokenDuration:  tokenDuration,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		profile:        profile,
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

	authContext, err := getAuthContext(serverInfo.FullMethod)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, common.AuthContextKey, authContext)

	principalID, err := in.getPrincipalID(ctx, accessTokenStr)
	if err != nil {
		if IsAuthenticationAllowed(serverInfo.FullMethod, authContext) {
			return handler(ctx, request)
		}
		return nil, err
	}

	ctx = context.WithValue(ctx, common.PrincipalIDContextKey, principalID)
	return handler(ctx, request)
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

	authContext, err := getAuthContext(serverInfo.FullMethod)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, common.AuthContextKey, authContext)

	principalID, err := in.getPrincipalID(ctx, accessTokenStr)
	if err != nil {
		if IsAuthenticationAllowed(serverInfo.FullMethod, authContext) {
			return handler(request, ss)
		}
		return err
	}

	ctx = context.WithValue(ctx, common.PrincipalIDContextKey, principalID)
	sss := overrideStream{ServerStream: ss, childCtx: ctx}
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
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, status.Errorf(codes.Unauthenticated, "access token expired")
		}
		return 0, status.Errorf(codes.Unauthenticated, "failed to parse claim")
	}
	if !audienceContains(claims.Audience, fmt.Sprintf(AccessTokenAudienceFmt, in.profile.Mode)) {
		return 0, status.Errorf(codes.Unauthenticated,
			"invalid access token, audience mismatch, got %q, expected %q. you may send request to the wrong environment",
			claims.Audience,
			fmt.Sprintf(AccessTokenAudienceFmt, in.profile.Mode),
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

func (in *APIAuthInterceptor) getPrincipalID(ctx context.Context, accessTokenStr string) (int, error) {
	principalID, err := in.authenticate(ctx, accessTokenStr)
	if err != nil {
		return 0, err
	}

	// Only update for authorized request.
	in.profile.LastActiveTs = time.Now().Unix()
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

func getAuthContext(fullMethod string) (*common.AuthContext, error) {
	methodTokens := strings.Split(fullMethod, "/")
	if len(methodTokens) != 3 {
		return nil, errs.Errorf("invalid full method name %q", fullMethod)
	}
	rd, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(methodTokens[1]))
	if err != nil {
		return nil, errs.Wrapf(err, "invalid full method name %q", fullMethod)
	}
	sd, ok := rd.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, errs.Errorf("invalid full method name %q", fullMethod)
	}
	md, ok := sd.Methods().ByName(protoreflect.Name(methodTokens[2])).Options().(*descriptorpb.MethodOptions)
	if !ok {
		return nil, errs.Errorf("invalid full method name %q", fullMethod)
	}
	allowWithoutCredentialAny := proto.GetExtension(md, v1pb.E_AllowWithoutCredential)
	allowWithoutCredential, ok := allowWithoutCredentialAny.(bool)
	if !ok {
		return nil, errs.Errorf("invalid full method name %q", fullMethod)
	}
	permissionAny := proto.GetExtension(md, v1pb.E_Permission)
	permission, ok := permissionAny.(string)
	if !ok {
		return nil, errs.Errorf("invalid full method name %q", fullMethod)
	}
	authMethodAny := proto.GetExtension(md, v1pb.E_AuthMethod)
	am, ok := authMethodAny.(v1pb.AuthMethod)
	if !ok {
		return nil, errs.Errorf("invalid full method name %q", fullMethod)
	}
	var authMethod common.AuthMethod
	switch am {
	case v1pb.AuthMethod_AUTH_METHOD_UNSPECIFIED:
		authMethod = common.AuthMethodUnspecified
	case v1pb.AuthMethod_IAM:
		authMethod = common.AuthMethodIAM
	case v1pb.AuthMethod_CUSTOM:
		authMethod = common.AuthMethodCustom
	}

	return &common.AuthContext{
		AllowWithoutCredential: allowWithoutCredential,
		Permission:             permission,
		AuthMethod:             authMethod,
	}, nil
}
