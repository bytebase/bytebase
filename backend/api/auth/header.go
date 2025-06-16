package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// GatewayResponseModifier is the response modifier for grpc gateway.
type GatewayResponseModifier struct {
	Store          *store.Store
	LicenseService *enterprise.LicenseService
}

// Modify is the mux option for modifying response header.
func (m *GatewayResponseModifier) Modify(ctx context.Context, response http.ResponseWriter, _ proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return errors.Errorf("failed to get ServerMetadata from context in the gateway response modifier")
	}

	isHTTPS := false
	for _, v := range md.HeaderMD.Get(GatewayMetadataRequestOriginKey) {
		if strings.HasPrefix(v, "https") {
			isHTTPS = true
		}
	}
	m.processMetadata(ctx, md, GatewayMetadataAccessTokenKey, AccessTokenCookieName, true /* httpOnly */, isHTTPS, response)
	return nil
}

func (m *GatewayResponseModifier) processMetadata(ctx context.Context, md runtime.ServerMetadata, metadataKey, cookieName string, httpOnly, isHTTPS bool, response http.ResponseWriter) {
	values := md.HeaderMD.Get(metadataKey)
	if len(values) == 0 {
		return
	}
	value := values[0]
	if value == "" {
		// Unset cookie.
		http.SetCookie(response, &http.Cookie{
			Name:    cookieName,
			Value:   "",
			Expires: time.Unix(0, 0),
			Path:    "/",
		})
		return
	}

	// Set cookie.
	sameSite := http.SameSiteStrictMode
	if isHTTPS {
		sameSite = http.SameSiteNoneMode
	}
	tokenDuration := GetTokenDuration(ctx, m.Store, m.LicenseService)
	http.SetCookie(response, &http.Cookie{
		Name:  cookieName,
		Value: value,
		// CookieExpDuration expires slightly earlier than the jwt expiration. Client would be logged out if the user
		// cookie expires, thus the client would always logout first before attempting to make a request with the expired jwt.
		// Suppose we have a valid refresh token, we will refresh the token in 2 cases:
		// 1. The access token is about to expire in <<refreshThresholdDuration>>
		// 2. The access token has already expired, we refresh the token so that the ongoing request can pass through.
		Expires: time.Now().Add(tokenDuration - 1*time.Second),
		Path:    "/",
		// Http-only helps mitigate the risk of client side script accessing the protected cookie.
		HttpOnly: httpOnly,
		// See https://github.com/bytebase/bytebase/issues/31.
		Secure:   isHTTPS,
		SameSite: sameSite,
	})
}

func GetTokenDuration(ctx context.Context, store *store.Store, licenseService *enterprise.LicenseService) time.Duration {
	tokenDuration := DefaultTokenDuration

	// If the sign-in frequency control feature is not enabled, return default duration
	if err := licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_SIGN_IN_FREQUENCY_CONTROL); err != nil {
		return tokenDuration
	}

	workspaceProfile, err := store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return tokenDuration
	}
	passwordRestriction, err := store.GetPasswordRestrictionSetting(ctx)
	if err != nil {
		return tokenDuration
	}

	if workspaceProfile.TokenDuration != nil && workspaceProfile.TokenDuration.GetSeconds() > 0 {
		tokenDuration = workspaceProfile.TokenDuration.AsDuration()
	}
	// Currently we implement the password rotation restriction in a simple way:
	// 1. Only check if users need to reset their password during login.
	// 2. For the 1st time login, if `RequireResetPasswordForFirstLogin` is true, `require_reset_password` in the response will be true
	// 3. Otherwise if the `PasswordRotation` exists, check the password last updated time to decide if the `require_reset_password` is true.
	// So we will use the minimum value between (`workspaceProfile.TokenDuration`, `passwordRestriction.PasswordRotation`) to force to expire the token.
	if passwordRestriction.PasswordRotation != nil && passwordRestriction.PasswordRotation.GetSeconds() > 0 {
		passwordRotation := passwordRestriction.PasswordRotation.AsDuration()
		if passwordRotation.Seconds() < tokenDuration.Seconds() {
			tokenDuration = passwordRotation
		}
	}

	return tokenDuration
}
