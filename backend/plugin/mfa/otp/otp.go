// Package otp is the plugin for OTP Multi-Factor Authentication.
package otp

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/exp/slices"
)

const (
	// issuerName is the name of the issuer of the OTP token.
	issuerName = "Bytebase"
	// maxPastSecretCount is the maximum number of past secret we will check.
	maxPastSecretCount = 5
)

// removeSecondsFromTimestamp removes the seconds from the timestamp.
func removeSecondsFromTimestamp(timestamp time.Time) int64 {
	return timestamp.Unix() - timestamp.Unix()%int64(time.Minute.Seconds())
}

// TimeBasedReader is a reader that returns the same value for the same timestamp.
type TimeBasedReader struct {
	reader *strings.Reader
}

// NewTimeBasedReader creates a new TimeBasedReader with the given timestamp.
func NewTimeBasedReader(timestamp time.Time) *TimeBasedReader {
	return &TimeBasedReader{
		reader: strings.NewReader(strconv.FormatInt(removeSecondsFromTimestamp(timestamp), 10)),
	}
}

func (r *TimeBasedReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

// GenerateSecret generates a new secret for the given account name and timestamp.
func GenerateSecret(accountName string, timestamp time.Time) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuerName,
		AccountName: accountName,
		Rand:        NewTimeBasedReader(timestamp),
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

// GetPastSecrets returns the past 5 secrets for the given account name and timestamp.
func GetPastSecrets(accountName string, timestamp time.Time) ([]string, error) {
	var secrets []string
	for i := 0; i < maxPastSecretCount; i++ {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      issuerName,
			AccountName: accountName,
			Rand:        NewTimeBasedReader(time.Unix(timestamp.Unix()-int64(i*int(time.Minute.Seconds())), 0)),
		})
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, key.Secret())
	}
	return secrets, nil
}

// ValidateWithCodeAndAccountName validates the given code against the given account name.
// It will check the current secret and the past 5 secrets.
func ValidateWithCodeAndAccountName(code, accountName string) (bool, error) {
	now := time.Now()
	secret, err := GenerateSecret(accountName, now)
	if err != nil {
		return false, err
	}

	pastSecrets, err := GetPastSecrets(accountName, now)
	if err != nil {
		return false, err
	}
	if !slices.Contains(pastSecrets, secret) {
		return false, errors.New("OTP has expired")
	}

	for _, pastSecret := range pastSecrets {
		if ValidateWithCodeAndSecret(code, pastSecret) {
			return true, nil
		}
	}
	return false, errors.New("invalid code")
}

// ValidateWithCodeAndSecret validates the given code against the given secret.
func ValidateWithCodeAndSecret(code, secret string) bool {
	return totp.Validate(code, secret)
}
