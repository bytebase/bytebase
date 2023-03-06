// Package otp is the plugin for OTP Multi-Factor Authentication.
package otp

import (
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
)

const (
	// issuerName is the name of the issuer of the OTP token.
	issuerName = "Bytebase"
	// generateSecretPeriod is the period that we generate a new secret.
	generateSecretPeriod = 5 * time.Minute
)

// TimeBasedReader is a reader that returns the same value for the same timestamp.
type TimeBasedReader struct {
	reader *strings.Reader
}

// NewTimeBasedReader creates a new TimeBasedReader with the given account name and timestamp.
func NewTimeBasedReader(accountName string, timestamp time.Time) *TimeBasedReader {
	// Convert the timestamp to Unix time and divide by the secretMaxExpiredDuration.
	// We generate a new secret every 5 minutes. e.g. 15:00, 15:05
	formatedTimestampUnix := timestamp.Unix() / int64(generateSecretPeriod/time.Second)
	return &TimeBasedReader{
		reader: strings.NewReader(fmt.Sprintf("%s-%d", accountName, formatedTimestampUnix)),
	}
}

func (r *TimeBasedReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

// GenerateSecret generates a secret for the given account name and timestamp.
func GenerateSecret(accountName string, timestamp time.Time) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuerName,
		AccountName: accountName,
		Rand:        NewTimeBasedReader(accountName, timestamp),
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

// GetValidSecrets returns a list of valid secrets for the given account name and timestamp.
func GetValidSecrets(accountName string, timestamp time.Time) ([]string, error) {
	var secrets []string
	// validTimestamps is a list of timestamps that are valid for the secret.
	// The secret is valid for 5 minutes. So we need to check the current timestamp and the timestamp 5 minutes ago.
	validTimestamps := []time.Time{timestamp, timestamp.Add(-1 * generateSecretPeriod)}
	for _, t := range validTimestamps {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      issuerName,
			AccountName: accountName,
			Rand:        NewTimeBasedReader(accountName, t),
		})
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, key.Secret())
	}
	return secrets, nil
}

// ValidateWithCodeAndAccountName validates the given code against the given account name.
func ValidateWithCodeAndAccountName(code, accountName string) (bool, error) {
	validSecrets, err := GetValidSecrets(accountName, time.Now())
	if err != nil {
		return false, err
	}

	for _, secret := range validSecrets {
		if ValidateWithCodeAndSecret(code, secret) {
			return true, nil
		}
	}
	return false, nil
}

// ValidateWithCodeAndSecret validates the given code against the given secret.
func ValidateWithCodeAndSecret(code, secret string) bool {
	return totp.Validate(code, secret)
}
