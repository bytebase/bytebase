package v1

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
)

const neonPageTokenDomain = "bytebase/neon-discovery/v1"

func neonPageTokenCipher(secret string) (cipher.AEAD, error) {
	if secret == "" {
		return nil, errors.New("missing discovery token secret")
	}
	key := hmac.New(sha256.New, []byte(secret))
	_, _ = key.Write([]byte(neonPageTokenDomain))
	block, err := aes.NewCipher(key.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCMWithRandomNonce(block)
}

// Scope is authenticated without retaining the API key. The deployment secret
// keeps continuation tokens usable across replicas and process restarts.
func encodeNeonPageToken(secret, cursor, workspace, parent, project string) (string, error) {
	if cursor == "" {
		return "", nil
	}
	aead, err := neonPageTokenCipher(secret)
	if err != nil {
		return "", err
	}
	scope, err := json.Marshal([]string{neonPageTokenDomain, workspace, parent, project})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(aead.Seal(nil, nil, []byte(cursor), scope)), nil
}

func decodeNeonPageToken(secret, token, workspace, parent, project string) (string, error) {
	if token == "" {
		return "", nil
	}
	if len(token) > 65536 {
		return "", errors.New("invalid discovery page token")
	}
	aead, err := neonPageTokenCipher(secret)
	if err != nil {
		return "", err
	}
	scope, err := json.Marshal([]string{neonPageTokenDomain, workspace, parent, project})
	if err != nil {
		return "", err
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", errors.New("invalid discovery page token")
	}
	cursor, err := aead.Open(nil, nil, data, scope)
	if err != nil || len(cursor) == 0 {
		return "", errors.New("invalid discovery page token")
	}
	return string(cursor), nil
}
