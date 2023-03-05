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
	// secretMaxExpiredDuration is the maximum duration that a secret is valid.
	// The secret is valid for 5 minutes.
	secretMaxExpiredDuration = 5 * time.Minute
)

// TimeBasedReader is a reader that returns the same value for the same timestamp.
type TimeBasedReader struct {
	reader *strings.Reader
}

// NewTimeBasedReader creates a new TimeBasedReader for the given timestamp.
func NewTimeBasedReader(timestamp time.Time) *TimeBasedReader {
	formatedTimestampUnix := timestamp.Unix() - int64(timestamp.Second())
	return &TimeBasedReader{
		reader: strings.NewReader(strconv.FormatInt(formatedTimestampUnix, 10)),
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
		Rand:        NewTimeBasedReader(timestamp),
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

// GetValidSecrets returns a list of valid secrets for the given account name and timestamp.
func GetValidSecrets(accountName string, timestamp time.Time) ([]string, error) {
	var secrets []string
	tempTime := timestamp
	expiredTime := timestamp.Add(-1 * secretMaxExpiredDuration)
	// Iterate from the current time to the expired time, and generate a secret for each time.
	for tempTime.After(expiredTime) {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      issuerName,
			AccountName: accountName,
			Rand:        NewTimeBasedReader(tempTime),
		})
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, key.Secret())
		// Move the time back by 1 minute.
		tempTime = tempTime.Add(-1 * time.Minute)
	}
	return secrets, nil
}

// ValidateWithCodeAndAccountName validates the given code against the given account name.
func ValidateWithCodeAndAccountName(code, accountName string) (bool, error) {
	currentTime := time.Now()
	secret, err := GenerateSecret(accountName, currentTime)
	if err != nil {
		return false, err
	}

	validSecrets, err := GetValidSecrets(accountName, currentTime)
	if err != nil {
		return false, err
	}
	if !slices.Contains(validSecrets, secret) {
		return false, errors.New("OTP has expired")
	}

	for _, secret := range validSecrets {
		if ValidateWithCodeAndSecret(code, secret) {
			return true, nil
		}
	}
	return false, errors.New("invalid code")
}

// ValidateWithCodeAndSecret validates the given code against the given secret.
func ValidateWithCodeAndSecret(code, secret string) bool {
	return totp.Validate(code, secret)
}
