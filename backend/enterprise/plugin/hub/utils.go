package hub

import (
	"github.com/golang-jwt/jwt"

	"github.com/bytebase/bytebase/backend/common"
)

func parseJWTToken(tokenString, expectVersion, publicKey string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, common.Errorf(common.Invalid, "unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid != expectVersion {
			return nil, common.Errorf(common.Invalid, "version '%v' is not valid. expect %s", token.Header["kid"], expectVersion)
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
		if err != nil {
			return nil, common.Wrap(err, common.Invalid)
		}

		return key, nil
	})
	if err != nil {
		return common.Wrap(err, common.Invalid)
	}

	if !token.Valid {
		return common.Errorf(common.Invalid, "invalid token")
	}

	return claims.Valid()
}
