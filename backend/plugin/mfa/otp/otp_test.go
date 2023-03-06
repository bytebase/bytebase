package otp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func TestGenerateTimeBasedSecret(t *testing.T) {
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
			name:            "6min",
			generateTime:    currentTime,
			validateTime:    currentTime.Add(6 * time.Minute),
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
