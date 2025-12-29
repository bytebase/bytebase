package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/bytebase/bytebase/backend/enterprise"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// token="" => unset
func GetTokenCookie(ctx context.Context, stores *store.Store, licenseService *enterprise.LicenseService, origin, token string) *http.Cookie {
	if token == "" {
		return &http.Cookie{
			Name:    AccessTokenCookieName,
			Value:   "",
			Expires: time.Unix(0, 0),
			Path:    "/",
		}
	}
	isHTTPS := strings.HasPrefix(origin, "https")
	tokenDuration := GetAccessTokenDuration(ctx, stores, licenseService)
	return &http.Cookie{
		Name:  AccessTokenCookieName,
		Value: token,
		// CookieExpDuration expires slightly earlier than the jwt expiration. Client would be logged out if the user
		// cookie expires, thus the client would always logout first before attempting to make a request with the expired jwt.
		// Suppose we have a valid refresh token, we will refresh the token in 2 cases:
		// 1. The access token is about to expire in <<refreshThresholdDuration>>
		// 2. The access token has already expired, we refresh the token so that the ongoing request can pass through.
		Expires: time.Now().Add(tokenDuration - 1*time.Second),
		Path:    "/",
		// Http-only helps mitigate the risk of client side script accessing the protected cookie.
		HttpOnly: true,
		// See https://github.com/bytebase/bytebase/issues/31.
		Secure:   isHTTPS,
		SameSite: http.SameSiteStrictMode,
	}
}

func GetAccessTokenDuration(ctx context.Context, store *store.Store, licenseService *enterprise.LicenseService) time.Duration {
	accessTokenDuration := DefaultAccessTokenDuration

	// If the sign-in frequency control feature is not enabled, return default duration
	if err := licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_TOKEN_DURATION_CONTROL); err != nil {
		return accessTokenDuration
	}

	workspaceProfile, err := store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return accessTokenDuration
	}

	if workspaceProfile.GetAccessTokenDuration().GetSeconds() > 0 {
		accessTokenDuration = workspaceProfile.GetAccessTokenDuration().AsDuration()
	}

	return accessTokenDuration
}

func GetRefreshTokenDuration(ctx context.Context, store *store.Store, licenseService *enterprise.LicenseService) time.Duration {
	refreshTokenDuration := DefaultRefreshTokenDuration

	// If the sign-in frequency control feature is not enabled, return default duration
	if err := licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_TOKEN_DURATION_CONTROL); err != nil {
		return refreshTokenDuration
	}

	workspaceProfile, err := store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return refreshTokenDuration
	}

	if workspaceProfile.GetRefreshTokenDuration().GetSeconds() > 0 {
		refreshTokenDuration = workspaceProfile.GetRefreshTokenDuration().AsDuration()
	}
	// Currently we implement the password rotation restriction in a simple way:
	// 1. Only check if users need to reset their password during login.
	// 2. For the 1st time login, if `RequireResetPasswordForFirstLogin` is true, `require_reset_password` in the response will be true
	// 3. Otherwise if the `PasswordRotation` exists, check the password last updated time to decide if the `require_reset_password` is true.
	// So we will use the minimum value between (`refreshTokenDuration`, `passwordRestriction.PasswordRotation`) to force to expire the token.
	passwordRestriction := workspaceProfile.GetPasswordRestriction()
	if passwordRestriction.GetPasswordRotation().GetSeconds() > 0 {
		passwordRotation := passwordRestriction.GetPasswordRotation().AsDuration()
		if passwordRotation.Seconds() < refreshTokenDuration.Seconds() {
			refreshTokenDuration = passwordRotation
		}
	}

	return refreshTokenDuration
}

// GetRefreshTokenCookie creates a cookie for the refresh token.
// token="" => unset (clears cookie)
// Path is "/" to allow logout to delete the token from database.
// Security is maintained via HttpOnly, Secure, and SameSite=Strict.
func GetRefreshTokenCookie(origin, token string, duration time.Duration) *http.Cookie {
	if token == "" {
		return &http.Cookie{
			Name:    RefreshTokenCookieName,
			Value:   "",
			Expires: time.Unix(0, 0),
			Path:    "/",
		}
	}
	isHTTPS := strings.HasPrefix(origin, "https")
	return &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		MaxAge:   int(duration.Seconds()),
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteStrictMode,
	}
}

// GetRefreshTokenFromCookie extracts the refresh token from request headers.
func GetRefreshTokenFromCookie(header http.Header) string {
	for _, cookie := range header.Values("Cookie") {
		for _, part := range strings.Split(cookie, ";") {
			part = strings.TrimSpace(part)
			if rt, ok := strings.CutPrefix(part, RefreshTokenCookieName+"="); ok {
				return rt
			}
		}
	}
	return ""
}
