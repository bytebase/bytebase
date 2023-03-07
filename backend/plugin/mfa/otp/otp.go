// Package otp is the plugin for OTP Multi-Factor Authentication.
package otp

import (
	"github.com/pquerna/otp/totp"
)

const (
	// issuerName is the name of the issuer of the OTP token.
	issuerName = "Bytebase"
)

// GenerateRandSecret generates a random secret for the given account name.
func GenerateRandSecret(accountName string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuerName,
		AccountName: accountName,
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

// ValidateWithCodeAndSecret validates the given code against the given secret.
func ValidateWithCodeAndSecret(code, secret string) bool {
	return totp.Validate(code, secret)
}
