package otp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func TestValidateTimeBasedSecretDuration(t *testing.T) {
	currentTime := time.Now()
	accountName := "test-user"

	tests := []struct {
		name            string
		generateTime    time.Time
		validateTime    time.Time
		isSecretExpired bool
	}{
		{
			name:            "-5min",
			generateTime:    currentTime,
			validateTime:    currentTime.Add(-5 * time.Minute),
			isSecretExpired: true,
		},
		{
			name:            "0",
			generateTime:    currentTime,
			validateTime:    currentTime,
			isSecretExpired: false,
		},
		{
			name:            "20s",
			generateTime:    currentTime,
			validateTime:    currentTime.Add(20 * time.Second),
			isSecretExpired: false,
		},
		{
			name:            "2min",
			generateTime:    currentTime,
			validateTime:    currentTime.Add(2 * time.Minute),
			isSecretExpired: false,
		},
		{
			name:            "4min - 1s",
			generateTime:    currentTime,
			validateTime:    currentTime.Add(4*time.Minute - 1*time.Second),
			isSecretExpired: false,
		},
		{
			name:            "5min - 1s",
			generateTime:    currentTime,
			validateTime:    currentTime.Add(5*time.Minute - 1*time.Second),
			isSecretExpired: false,
		},
		{
			name:            "5min",
			generateTime:    currentTime,
			validateTime:    currentTime.Add(5 * time.Minute),
			isSecretExpired: false,
		},
		{
			name:            "10min",
			generateTime:    currentTime,
			validateTime:    currentTime.Add(10 * time.Minute),
			isSecretExpired: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			secret, err := GenerateSecret(accountName, test.generateTime)
			assert.NoError(t, err)
			validSecrets, err := GetValidSecrets(accountName, test.validateTime)
			assert.NoError(t, err)
			assert.Equal(t, test.isSecretExpired, !slices.Contains(validSecrets, secret))
		})
	}
}

func TestGenerateTimeBasedSecret(t *testing.T) {
	accountName := "test-user"

	tests := []struct {
		accountName  string
		timestamp    time.Time
		wantedSecret string
	}{
		{
			accountName:  "test-user",
			timestamp:    time.Unix(1678115520, 0),
			wantedSecret: "ORSXG5BNOVZWK4RVGU4TGNZRHAAAAAAA",
		},
	}

	for _, test := range tests {
		t.Run(test.accountName, func(t *testing.T) {
			secret, err := GenerateSecret(accountName, test.timestamp)
			assert.NoError(t, err)
			assert.Equal(t, test.wantedSecret, secret)
		})
	}
}
