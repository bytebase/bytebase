package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/pkg/errors"
)

// VerifyPKCE verifies that the code_verifier matches the code_challenge.
// Only S256 method is supported per OAuth 2.1 requirements.
func VerifyPKCE(codeVerifier, codeChallenge, codeChallengeMethod string) error {
	if codeChallengeMethod != "S256" {
		return errors.New("only S256 code_challenge_method is supported")
	}

	// S256: BASE64URL(SHA256(code_verifier)) == code_challenge
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])

	if computed != codeChallenge {
		return errors.New("code_verifier does not match code_challenge")
	}

	return nil
}

// ValidateCodeVerifier checks that the code_verifier meets RFC 7636 requirements.
func ValidateCodeVerifier(verifier string) error {
	// code_verifier must be 43-128 characters
	if len(verifier) < 43 || len(verifier) > 128 {
		return errors.New("code_verifier must be 43-128 characters")
	}

	// Must only contain unreserved characters: [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
	for _, c := range verifier {
		if !isUnreserved(c) {
			return errors.New("code_verifier contains invalid characters")
		}
	}

	return nil
}

// ValidateCodeChallenge checks that the code_challenge is valid base64url.
func ValidateCodeChallenge(challenge string) error {
	// S256 challenge is 43 characters (256 bits in base64url)
	if len(challenge) != 43 {
		return errors.New("code_challenge must be 43 characters for S256")
	}

	// Must be valid base64url (no padding)
	if strings.ContainsAny(challenge, "+/=") {
		return errors.New("code_challenge must be base64url encoded without padding")
	}

	return nil
}

func isUnreserved(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}
