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
	ExternalURL string
}

// Modify is the mux option for modifying response header.
func (m *GatewayResponseModifier) Modify(ctx context.Context, response http.ResponseWriter, _ proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return errors.Errorf("failed to get ServerMetadata from context in the gateway response modifier")
	}
	isHTTPS := strings.HasPrefix(m.ExternalURL, "https")
	processMetadata(md, GatewayMetadataAccessTokenKey, AccessTokenCookieName, true /* httpOnly */, isHTTPS, response)
	processMetadata(md, GatewayMetadataRefreshTokenKey, RefreshTokenCookieName, true /* httpOnly */, isHTTPS, response)
	processMetadata(md, GatewayMetadataUserIDKey, UserIDCookieName, false /* httpOnly */, isHTTPS, response)
	return nil
}

func processMetadata(md runtime.ServerMetadata, metadataKey, cookieName string, httpOnly, isHTTPS bool, response http.ResponseWriter) {
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
			Name:    cookieName,
			Value:   value,
			Expires: time.Now().Add(CookieExpDuration),
			Path:    "/",
			// Http-only helps mitigate the risk of client side script accessing the protected cookie.
			HttpOnly: httpOnly,
			// See https://github.com/bytebase/bytebase/issues/31.
			Secure:   isHTTPS,
			SameSite: sameSite,
		})
	}
}
