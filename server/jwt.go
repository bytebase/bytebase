package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/store"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const (
	issuer                  = "bytebase"
	accessTokenAudienceFmt  = "bb.user.access.%s"
	refreshTokenAudienceFmt = "bb.user.refresh.%s"

	// Cookie section
	accessTokenCookieName  = "access-token"
	refreshTokenCookieName = "refresh-token"

	// Signing key section. For now, this is only used for signing, not for verifying since we only
	// have 1 version. But it will be used to maintain backward compatibility if we change the signing mechanism.
	keyID = "v1"

	// Expiration section
	refreshThresholdDuration = 1 * time.Hour
	accessTokenDuration      = 24 * time.Hour
	refreshTokenDuration     = 7 * 24 * time.Hour
	// Make cookie expire slightly earlier than the jwt expiration. Client would be logged out if the user
	// cookie expires, thus the client would always logout first before attempting to make a request with the expired jwt.
	// Suppose we have a valid refresh token, we will refresh the token in 2 cases:
	// 1. The access token is about to expire in <<refreshThresholdDuration>>
	// 2. The access token has already expired, we refresh the token so that the ongoing request can pass through
	cookieExpDuration = refreshTokenDuration - 1*time.Minute

	// Context section
	// The key name used to store principal id in the context
	// principal id is extracted from the jwt token subject field.
	principalIDContextKey = "principal-id"
)

// Claims creates a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like name.
type Claims struct {
	Name string `json:"name"`
	jwt.StandardClaims
}

func getPrincipalIDContextKey() string {
	return principalIDContextKey
}

// GenerateTokensAndSetCookies generates jwt token and saves it to the http-only cookie.
func GenerateTokensAndSetCookies(c echo.Context, user *api.Principal, mode common.ReleaseMode, secret string) error {
	accessToken, err := generateAccessToken(user, mode, secret)
	if err != nil {
		return fmt.Errorf("failed to generate access token: %w", err)
	}

	cookieExp := time.Now().Add(cookieExpDuration)
	setTokenCookie(c, accessTokenCookieName, accessToken, cookieExp)
	setUserCookie(c, user, cookieExp)

	// We generate here a new refresh token and saving it to the cookie.
	refreshToken, err := generateRefreshToken(user, mode, secret)
	if err != nil {
		return fmt.Errorf("failed to generate refresh token: %w", err)
	}
	setTokenCookie(c, refreshTokenCookieName, refreshToken, cookieExp)

	return nil
}

func generateAccessToken(user *api.Principal, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(accessTokenDuration)
	return generateToken(user, fmt.Sprintf(accessTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

func generateRefreshToken(user *api.Principal, mode common.ReleaseMode, secret string) (string, error) {
	expirationTime := time.Now().Add(refreshTokenDuration)
	return generateToken(user, fmt.Sprintf(refreshTokenAudienceFmt, mode), expirationTime, []byte(secret))
}

// Pay attention to this function. It holds the main JWT token generation logic.
func generateToken(user *api.Principal, aud string, expirationTime time.Time, secret []byte) (string, error) {
	// Create the JWT claims, which includes the username and expiry time.
	claims := &Claims{
		Name: user.Name,
		StandardClaims: jwt.StandardClaims{
			Audience: aud,
			// In JWT, the expiry time is expressed as unix milliseconds.
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
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

// Here we are creating a new cookie, which will store the valid JWT token.
func setTokenCookie(c echo.Context, name, token string, expiration time.Time) {
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = token
	cookie.Expires = expiration
	cookie.Path = "/"
	// Http-only helps mitigate the risk of client side script accessing the protected cookie.
	cookie.HttpOnly = true
	// For now, we allow bytebase to run on non-https host, see https://github.com/bytebase/bytebase/issues/31
	// cookie.Secure = true
	cookie.SameSite = http.SameSiteStrictMode
	c.SetCookie(cookie)
}

func removeTokenCookie(c echo.Context, name string) {
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	cookie.Path = "/"
	c.SetCookie(cookie)
}

// Purpose of this cookie is to store the user's id.
func setUserCookie(c echo.Context, user *api.Principal, expiration time.Time) {
	cookie := new(http.Cookie)
	cookie.Name = "user"
	cookie.Value = strconv.Itoa(user.ID)
	cookie.Expires = expiration
	cookie.Path = "/"
	// For now, we allow bytebase to run on non-https host, see https://github.com/bytebase/bytebase/issues/31
	// cookie.Secure = true
	cookie.SameSite = http.SameSiteStrictMode
	c.SetCookie(cookie)
}

func removeUserCookie(c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = "user"
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	cookie.Path = "/"
	c.SetCookie(cookie)
}

// JWTMiddleware validates the access token.
// If the access token is about to expire or has expired and the request has a valid refresh token, it
// will try to generate new access token and refresh token.
func JWTMiddleware(l *zap.Logger, principalStore *store.Store, next echo.HandlerFunc, mode common.ReleaseMode, secret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skips auth, actuator, plan
		if common.HasPrefixes(c.Path(), "/api/auth", "/api/actuator", "/api/plan") {
			return next(c)
		}

		method := c.Request().Method
		// Skip GET /subscription request
		if common.HasPrefixes(c.Path(), "/api/subscription") && method == "GET" {
			return next(c)
		}

		cookie, err := c.Cookie(accessTokenCookieName)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing access token")
		}

		claims := &Claims{}
		accessToken, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Name {
				return nil, fmt.Errorf("unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
			}
			if kid, ok := t.Header["kid"].(string); ok {
				if kid == "v1" {
					return []byte(secret), nil
				}
			}
			return nil, fmt.Errorf("unexpected access token kid=%v", t.Header["kid"])
		})

		if claims.Audience != fmt.Sprintf(accessTokenAudienceFmt, mode) {
			return echo.NewHTTPError(http.StatusUnauthorized,
				fmt.Sprintf("Invalid access token, audience mismatch, got %q, expected %q. you may send request to the wrong environment",
					claims.Audience,
					fmt.Sprintf(accessTokenAudienceFmt, mode),
				))
		}

		generateToken := time.Until(time.Unix(claims.ExpiresAt, 0)) < refreshThresholdDuration
		if err != nil {
			var ve *jwt.ValidationError
			if errors.As(err, &ve) {
				// If expiration error is the only error, we will clear the err
				// and generate new access token and refresh token
				if ve.Errors == jwt.ValidationErrorExpired {
					err = nil
					generateToken = true
				}
			}
		}

		// We either have a valid access token or we will attempt to generate new access token and refresh token
		if err == nil {
			ctx := c.Request().Context()
			principalID, err := strconv.Atoi(claims.Subject)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Malformatted ID in the token.")
			}

			// Even if there is no error, we still need to make sure the user still exists.
			principalFind := &api.PrincipalFind{
				ID: &principalID,
			}
			user, err := principalStore.FindPrincipal(ctx, principalFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user ID: %d", principalID)).SetInternal(err)
			}
			if user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find user ID: %d", principalID))
			}

			if generateToken {
				generateTokenFunc := func() error {
					rc, err := c.Cookie(refreshTokenCookieName)

					if err != nil {
						return echo.NewHTTPError(http.StatusUnauthorized, "Failed to generate access token. Missing refresh token.")
					}

					// Parses token and checks if it's valid.
					refreshTokenClaims := &Claims{}
					refreshToken, err := jwt.ParseWithClaims(rc.Value, refreshTokenClaims, func(t *jwt.Token) (interface{}, error) {
						if t.Method.Alg() != jwt.SigningMethodHS256.Name {
							return nil, fmt.Errorf("unexpected refresh token signing method=%v, expected %v", t.Header["alg"], jwt.SigningMethodHS256)
						}

						if kid, ok := t.Header["kid"].(string); ok {
							if kid == "v1" {
								return []byte(secret), nil
							}
						}
						return nil, fmt.Errorf("unexpected refresh token kid=%v", t.Header["kid"])
					})
					if err != nil {
						if err == jwt.ErrSignatureInvalid {
							return echo.NewHTTPError(http.StatusUnauthorized, "Failed to generate access token. Invalid refresh token signature.")
						}
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to refresh expired token. User Id %d", principalID)).SetInternal(err)
					}

					if refreshTokenClaims.Audience != fmt.Sprintf(refreshTokenAudienceFmt, mode) {
						return echo.NewHTTPError(http.StatusUnauthorized,
							fmt.Sprintf("Invalid refresh token, audience mismatch, got %q, expected %q. you may send request to the wrong environment",
								refreshTokenClaims.Audience,
								fmt.Sprintf(refreshTokenAudienceFmt, mode),
							))
					}

					// If we have a valid refresh token, we will generate new access token and refresh token
					if refreshToken != nil && refreshToken.Valid {
						if err := GenerateTokensAndSetCookies(c, user, mode, secret); err != nil {
							return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to refresh expired token. User Id %d", principalID)).SetInternal(err)
						}
					}

					return nil
				}

				// It may happen that we still have a valid access token, but we encounter issue when trying to generate new token
				// In such case, we won't return the error.
				if err := generateTokenFunc(); err != nil && !accessToken.Valid {
					return err
				}
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
