package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

const (
	accessTokenCookieName  = "access-token"
	refreshTokenCookieName = "refresh-token"
	// Just for the demo purpose, I declared a secret here. In the real-world application, you might need to get it from the env variables.
	jwtSecretKey             = "some-secret-key"
	jwtRefreshSecretKey      = "some-refresh-secret-key"
	refreshThresholdDuration = 15 * time.Minute
	accessTokenDuration      = 1 * time.Hour
	refreshTokenDuration     = 24 * time.Hour
	// Cookie expiration is the same as the refresh token.
	// Suppose we have a valid refresh token, we should refresh the token in 2 cases:
	// 1. The access token is about to expire in <<refreshThresholdDuration>>
	// 2. The access token has already expired, we refresh the token so that the ongoing request can pass through
	cookieExpDuration = refreshTokenDuration
)

func GetJWTSecret() string {
	return jwtSecretKey
}

func GetRefreshJWTSecret() string {
	return jwtRefreshSecretKey
}

// Create a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time.
type Claims struct {
	Name string `json:"name"`
	jwt.StandardClaims
}

// GenerateTokensAndSetCookies generates jwt token and saves it to the http-only cookie.
func GenerateTokensAndSetCookies(user *api.Principal, c echo.Context) error {
	accessToken, err := generateAccessToken(user)
	if err != nil {
		return err
	}

	cookieExp := time.Now().Add(cookieExpDuration)
	setTokenCookie(accessTokenCookieName, accessToken, cookieExp, c)
	setUserCookie(user, cookieExp, c)

	// We generate here a new refresh token and saving it to the cookie.
	refreshToken, err := generateRefreshToken(user)
	if err != nil {
		return err
	}
	setTokenCookie(refreshTokenCookieName, refreshToken, cookieExp, c)

	return nil
}

func generateAccessToken(user *api.Principal) (string, error) {
	expirationTime := time.Now().Add(accessTokenDuration)
	return generateToken(user, expirationTime, []byte(GetJWTSecret()))
}

func generateRefreshToken(user *api.Principal) (string, error) {
	expirationTime := time.Now().Add(refreshTokenDuration)
	return generateToken(user, expirationTime, []byte(GetRefreshJWTSecret()))
}

// Pay attention to this function. It holds the main JWT token generation logic.
func generateToken(user *api.Principal, expirationTime time.Time, secret []byte) (string, error) {
	// Create the JWT claims, which includes the username and expiry time.
	claims := &Claims{
		Name: user.Name,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds.
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Declare the token with the HS256 algorithm used for signing, and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

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
	cookie.Value = user.Name
	cookie.Expires = expiration
	cookie.Path = "/"
	c.SetCookie(cookie)
}

func TokenRefresherMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skips auth end point
		if strings.HasPrefix(c.Path(), "/api/auth") {
			return next(c)
		}

		// If the user is not authenticated (no user token data in the context), don't do anything.
		if c.Get("user") == nil {
			return next(c)
		}
		// Gets user token from the context.
		u := c.Get("user").(*jwt.Token)

		claims := u.Claims.(*Claims)

		// We ensure that a new token is not issued until enough time has elapsed.
		// In this case, a new token will only be issued if the old token is within
		// 15 mins of expiry.
		if time.Until(time.Unix(claims.ExpiresAt, 0)) < refreshThresholdDuration {
			// Gets the refresh token from the cookie.
			fmt.Println("Token about to expire, generate new token...")
			rc, err := c.Cookie(refreshTokenCookieName)
			if err == nil && rc != nil {
				// Parses token and checks if it valid.
				tkn, err := jwt.ParseWithClaims(rc.Value, claims, func(token *jwt.Token) (interface{}, error) {
					return []byte(GetRefreshJWTSecret()), nil
				})
				if err != nil {
					if err == jwt.ErrSignatureInvalid {
						c.Response().Writer.WriteHeader(http.StatusUnauthorized)
					}
				}

				if tkn != nil && tkn.Valid {
					// If everything is good, update tokens.
					_ = GenerateTokensAndSetCookies(&api.Principal{
						Name: claims.Name,
					}, c)
				}
			}
		}

		return next(c)
	}
}

// JWTErrorChecker will be executed when user try to access a protected path.
func JWTErrorChecker(err error, c echo.Context) error {
	// Redirects to the signIn form.
	fmt.Printf("Unauthorized to access protected route %s, err: %v\n", c.Path(), err)
	return err
}
