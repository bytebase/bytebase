package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// GatewayResponseModifier is the mux option for modifying response header.
func GatewayResponseModifier(ctx context.Context, response http.ResponseWriter, _ proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return errors.Errorf("failed to get ServerMetadata from context in the gateway response modifier")
	}
	processMetadata(md, GatewayMetadataAccessTokenKey, AccessTokenCookieName, true /* httpOnly */, response)
	processMetadata(md, GatewayMetadataRefreshTokenKey, RefreshTokenCookieName, true /* httpOnly */, response)
	processMetadata(md, GatewayMetadataUserIDKey, UserIDCookieName, false /* httpOnly */, response)
	return nil
}

func processMetadata(md runtime.ServerMetadata, metadataKey, cookieName string, httpOnly bool, response http.ResponseWriter) {
	values := md.HeaderMD.Get(metadataKey)
	if len(values) == 0 {
		return
	}
	value := values[0]
	if value == "" {
		unsetCookie(response, cookieName)
	} else {
		setCookie(response, cookieName, value, httpOnly)
	}
}

func setCookie(response http.ResponseWriter, key, value string, httpOnly bool) {
	http.SetCookie(response, &http.Cookie{
		Name:    key,
		Value:   value,
		Expires: time.Now().Add(CookieExpDuration),
		Path:    "/",
		// Http-only helps mitigate the risk of client side script accessing the protected cookie.
		HttpOnly: httpOnly,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})
}

func unsetCookie(response http.ResponseWriter, key string) {
	http.SetCookie(response, &http.Cookie{
		Name:    key,
		Value:   "",
		Expires: time.Unix(0, 0),
		Path:    "/",
	})
}
