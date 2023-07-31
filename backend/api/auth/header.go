package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// GatewayResponseModifier is the response modifier for grpc gateway.
type GatewayResponseModifier struct {
	ExternalURL          string
	RefreshTokenDuration time.Duration
}

// Modify is the mux option for modifying response header.
func (m *GatewayResponseModifier) Modify(ctx context.Context, response http.ResponseWriter, _ proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return errors.Errorf("failed to get ServerMetadata from context in the gateway response modifier")
	}
	isHTTPS := strings.HasPrefix(m.ExternalURL, "https")
	m.processMetadata(md, GatewayMetadataAccessTokenKey, AccessTokenCookieName, true /* httpOnly */, isHTTPS, response)
	m.processMetadata(md, GatewayMetadataRefreshTokenKey, RefreshTokenCookieName, true /* httpOnly */, isHTTPS, response)
	m.processMetadata(md, GatewayMetadataUserIDKey, UserIDCookieName, false /* httpOnly */, isHTTPS, response)
	return nil
}

func (m *GatewayResponseModifier) processMetadata(md runtime.ServerMetadata, metadataKey, cookieName string, httpOnly, isHTTPS bool, response http.ResponseWriter) {
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
	} else {
		// Set cookie.
		sameSite := http.SameSiteStrictMode
		if isHTTPS {
			sameSite = http.SameSiteNoneMode
		}
		http.SetCookie(response, &http.Cookie{
			Name:  cookieName,
			Value: value,
			// CookieExpDuration expires slightly earlier than the jwt expiration. Client would be logged out if the user
			// cookie expires, thus the client would always logout first before attempting to make a request with the expired jwt.
			// Suppose we have a valid refresh token, we will refresh the token in 2 cases:
			// 1. The access token is about to expire in <<refreshThresholdDuration>>
			// 2. The access token has already expired, we refresh the token so that the ongoing request can pass through.
			Expires: time.Now().Add(m.RefreshTokenDuration - 1*time.Minute),
			Path:    "/",
			// Http-only helps mitigate the risk of client side script accessing the protected cookie.
			HttpOnly: httpOnly,
			// See https://github.com/bytebase/bytebase/issues/31.
			Secure:   isHTTPS,
			SameSite: sameSite,
		})
	}
}
