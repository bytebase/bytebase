package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const (
	issuer               = "bytebase"
	accessTokenAudience  = "bb.user.access"
	refreshTokenAudience = "bb.user.refresh"

	// Cookie section
	accessTokenCookieName  = "access-token"
	refreshTokenCookieName = "refresh-token"

	// Signing key section
	keyId = "v1"

	// TODO: Uses better keys
	jwtSecretKey        = "some-secret-key"
	jwtRefreshSecretKey = "some-refresh-secret-key"

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

	// Conext section
	// The key name used to store principal id in the context
	// principal id is extracted from the jwt token subject field.
	principalIdContextKey = "principal-id"
)

// Create a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like name.
type Claims struct {
	Name string `json:"name"`
	jwt.StandardClaims
}

func getJWTSecret(kid string) string {
	if kid == keyId {
		return jwtSecretKey
	}
	return ""
}

func getRefreshJWTSecret(kid string) string {
	if kid == keyId {
		return jwtRefreshSecretKey
	}
	return ""
}

func GetPrincipalIdContextKey() string {
	return principalIdContextKey
}

// GenerateTokensAndSetCookies generates jwt token and saves it to the http-only cookie.
func GenerateTokensAndSetCookies(user *api.Principal, c echo.Context) error {
	accessToken, err := generateAccessToken(user)
	if err != nil {
		return fmt.Errorf("failed to generate access token: %w", err)
	}

	cookieExp := time.Now().Add(cookieExpDuration)
	setTokenCookie(accessTokenCookieName, accessToken, cookieExp, c)
	setUserCookie(user, cookieExp, c)

	// We generate here a new refresh token and saving it to the cookie.
	refreshToken, err := generateRefreshToken(user)
	if err != nil {
		return fmt.Errorf("failed to generate refresh token: %w", err)
	}
	setTokenCookie(refreshTokenCookieName, refreshToken, cookieExp, c)

	return nil
}

func generateAccessToken(user *api.Principal) (string, error) {
	expirationTime := time.Now().Add(accessTokenDuration)
	return generateToken(user, accessTokenAudience, expirationTime, []byte(getJWTSecret(keyId)))
}

func generateRefreshToken(user *api.Principal) (string, error) {
	expirationTime := time.Now().Add(refreshTokenDuration)
	return generateToken(user, refreshTokenAudience, expirationTime, []byte(getRefreshJWTSecret(keyId)))
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
	token.Header["kid"] = keyId

	// Create the JWT string.
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Here we are creating a new cookie, which will store the valid JWT token.
func setTokenCookie(name, token string, expiration time.Time, c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = token
	cookie.Expires = expiration
	cookie.Path = "/"
	// Http-only helps mitigate the risk of client side script accessing the protected cookie.
	cookie.HttpOnly = true

	c.SetCookie(cookie)
}

// Purpose of this cookie is to store the user's name.
func setUserCookie(user *api.Principal, expiration time.Time, c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = "user"
	cookie.Value = strconv.Itoa(user.ID)
	cookie.Expires = expiration
	cookie.Path = "/"
	c.SetCookie(cookie)
}

// JWTMiddleware validates the access token.
// If the access token is about to expire or has expired and the request has a valid refresh token, it
// will try to generate new access token and refresh token.
func JWTMiddleware(l *zap.Logger, p api.PrincipalService, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skips auth, webhook end point
		if strings.HasPrefix(c.Path(), "/api/auth") {
			return next(c)
		}

		cookie, err := c.Cookie(accessTokenCookieName)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing access token")
		}

		claims := &Claims{}
		accessToken, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Name {
				return nil, fmt.Errorf("unexpected access token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
			}
			if kid, ok := t.Header["kid"].(string); ok {
				if key := getJWTSecret(kid); key != "" {
					return []byte(key), nil
				}
			}
			return nil, fmt.Errorf("unexpected access token kid=%v", t.Header["kid"])
		})

		generateToken := time.Until(time.Unix(claims.ExpiresAt, 0)) < refreshThresholdDuration
		generateReason := "Token about to expire, generate new token..."
		if err != nil {
			var ve *jwt.ValidationError
			if errors.As(err, &ve) {
				// If expiration error is the only error,  we will clear the err
				// and generate new access token and refresh token
				if ve.Errors == jwt.ValidationErrorExpired {
					err = nil
					generateToken = true
					generateReason = "Token has expired, generate new token..."
				}
			}
		}

		// We either have a valid access token or we will attempt to generate new access token and refresh token
		if err == nil {
			principalId, err := strconv.Atoi(claims.Subject)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Malformatted ID in the token.")
			}

			// Even if there is no error, we still need to make sure the user still exists.
			principalFind := &api.PrincipalFind{
				ID: &principalId,
			}
			user, err := p.FindPrincipal(context.Background(), principalFind)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find to refresh expired token. User ID %d", principalId))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user to refresh expired token. User ID %d", principalId)).SetInternal(err)
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
							return nil, fmt.Errorf("unexpected refresh token signing method=%v, expect %v", t.Header["alg"], jwt.SigningMethodHS256)
						}
						if kid, ok := t.Header["kid"].(string); ok {
							if key := getRefreshJWTSecret(kid); key != "" {
								return []byte(key), nil
							}
						}
						return nil, fmt.Errorf("unexpected refresh token kid=%v", t.Header["kid"])
					})
					if err != nil {
						if err == jwt.ErrSignatureInvalid {
							return echo.NewHTTPError(http.StatusUnauthorized, "Failed to generate access token. Invalid refresh token signature.")
						}
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to refresh expired token. User Id %d", principalId)).SetInternal(err)
					}

					// If we have a valid refresh token, we will generate new access token and refresh token
					if refreshToken != nil && refreshToken.Valid {
						l.Info(generateReason)
						if err := GenerateTokensAndSetCookies(user, c); err != nil {
							return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to refresh expired token. User Id %d", principalId)).SetInternal(err)
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

			// Stores principalId into context.
			c.Set(GetPrincipalIdContextKey(), principalId)
			return next(c)
		}

		return &echo.HTTPError{
			Code:     http.StatusUnauthorized,
			Message:  "Invalid or expired access token",
			Internal: err,
		}
	}
}
