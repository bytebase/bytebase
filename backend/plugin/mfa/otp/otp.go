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
	// secondsInMinute is the number of seconds in a minute.
	secondsInMinute = 60
	// maxPastSecretCount is the maximum number of past secret we will check.
	maxPastSecretCount = 5
)

// getCurrentTimestampInMinute returns the current timestamp truncated to the minute.
func getCurrentTimestampInMinute() int64 {
	now := time.Now()
	truncated := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	return truncated.Unix()
}

// removeSecondsFromTimestamp removes the seconds from the timestamp.
func removeSecondsFromTimestamp(timestamp int64) int64 {
	t := time.Unix(timestamp, 0)
	truncated := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	return truncated.Unix()
}

// TimeBasedReader is a reader that returns the same value for the same timestamp.
type TimeBasedReader struct {
	reader *strings.Reader
}

func NewTimeBasedReader(timestamp int64) *TimeBasedReader {
	return &TimeBasedReader{
		reader: strings.NewReader(strconv.FormatInt(removeSecondsFromTimestamp(timestamp), 10)),
	}
}

func (r *TimeBasedReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

// GenerateSecret generates a new secret for the given account name and timestamp.
func GenerateSecret(accountName string, timestamp int64) (string, error) {
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
func GetPastSecrets(accountName string, timestamp int64) ([]string, error) {
	secrets := make([]string, 0)
	for i := 0; i < maxPastSecretCount; i++ {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      issuerName,
			AccountName: accountName,
			Rand:        NewTimeBasedReader(timestamp - int64(i)*secondsInMinute),
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
	currentTimestamp := getCurrentTimestampInMinute()
	secret, err := GenerateSecret(accountName, currentTimestamp)
	if err != nil {
		return false, err
	}

	pastSecrets, err := GetPastSecrets(accountName, currentTimestamp)
	if err != nil {
		return false, err
	}
	if !slices.Contains(pastSecrets, secret) {
		return false, errors.New("secret is outdate")
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
