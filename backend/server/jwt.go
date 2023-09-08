package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	pkgerrors "github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	// Context section
	// The key name used to store principal id in the context
	// principal id is extracted from the jwt token subject field.
	principalIDContextKey = "principal-id"
)

// Claims creates a struct that will be encoded to a JWT.
// We add jwt.RegisteredClaims as an embedded type, to provide fields such as name.
type Claims struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

func getPrincipalIDContextKey() string {
	return principalIDContextKey
}

func extractTokenFromHeader(c echo.Context) (string, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return "", nil
	}

	authHeaderParts := strings.Fields(authHeader)
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", common.Errorf(common.Invalid, "Authorization header format must be Bearer {token}")
	}

	return authHeaderParts[1], nil
}

func findAccessToken(c echo.Context) (string, error) {
	if common.HasPrefixes(c.Path(), openAPIPrefix) {
		return extractTokenFromHeader(c)
	}

	cookie, err := c.Cookie(auth.AccessTokenCookieName)
	if err != nil {
		// TODO(ed): support trigger the schema sync via terraform in a quick but dirty way.
		// We should support trigger the sync via OpenAPI later.
		if c.Path() == "/api/sql/sync-schema" {
			return extractTokenFromHeader(c)
		}
		return "", err
	}

	return cookie.Value, nil
}

// JWTMiddleware validates the access token.
// If the access token is about to expire or has expired and the request has a valid refresh token, it
// will try to generate new access token and refresh token.
func JWTMiddleware(pathPrefix string, principalStore *store.Store, next echo.HandlerFunc, mode common.ReleaseMode, secret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := strings.TrimPrefix(c.Request().URL.Path, pathPrefix)

		// Skips auth, actuator
		if common.HasPrefixes(path, "/auth", "/actuator") {
			return next(c)
		}

		method := c.Request().Method

		// Skip GET /feature request
		if common.HasPrefixes(path, "/feature") && method == "GET" {
			return next(c)
		}

		// Skips OpenAPI SQL endpoint
		if common.HasPrefixes(c.Path(), fmt.Sprintf("%s/sql", openAPIPrefix)) {
			return next(c)
		}

		token, err := findAccessToken(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing access token")
		}

		claims := &Claims{}
		_, err = jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Name {
				return nil, pkgerrors.Errorf("unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
			}
			if kid, ok := t.Header["kid"].(string); ok {
				if kid == "v1" {
					return []byte(secret), nil
				}
			}
			return nil, pkgerrors.Errorf("unexpected access token kid=%v", t.Header["kid"])
		})

		if !audienceContains(claims.Audience, fmt.Sprintf(auth.AccessTokenAudienceFmt, mode)) {
			return echo.NewHTTPError(http.StatusUnauthorized,
				fmt.Sprintf("Invalid access token, audience mismatch, got %q, expected %q. you may send request to the wrong environment",
					claims.Audience,
					fmt.Sprintf(auth.AccessTokenAudienceFmt, mode),
				))
		}

		if err != nil {
			var ve *jwt.ValidationError
			if errors.As(err, &ve) {
				// If expiration error is the only error, we will clear the err
				// and generate new access token and refresh token
				if ve.Errors == jwt.ValidationErrorExpired {
					return echo.NewHTTPError(http.StatusUnauthorized, "access token expired")
				}
			}
		}

		// We either have a valid access token or we will attempt to generate new access token and refresh token
		if err == nil {
			ctx := c.Request().Context()
			principalID, err := strconv.Atoi(claims.Subject)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Malformed ID in the token.")
			}

			// Even if there is no error, we still need to make sure the user still exists.
			user, err := principalStore.GetUserByID(ctx, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user ID: %d", principalID)).SetInternal(err)
			}
			if user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find user ID: %d", principalID))
			}

			// Stores principalID into context.
			c.Set(getPrincipalIDContextKey(), principalID)
			return next(c)
		}

		return &echo.HTTPError{
			Code:     http.StatusUnauthorized,
			Message:  "Invalid or expired access token",
			Internal: err,
		}
	}
}

func audienceContains(audience jwt.ClaimStrings, token string) bool {
	for _, v := range audience {
		if v == token {
			return true
		}
	}
	return false
}
