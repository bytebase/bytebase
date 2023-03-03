package otp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func TestGenerateTimeBasedSecret(t *testing.T) {
	currentTimestamp := time.Now().Unix()
	accountName := "test-user"

	tests := []struct {
		name              string
		generateTimestamp int64
		validateTimestamp int64
		isSecretExpired   bool
	}{
		{
			name:              "20s",
			generateTimestamp: currentTimestamp,
			validateTimestamp: currentTimestamp - 5*secondsInMinute,
			isSecretExpired:   true,
		},
		{
			name:              "20s",
			generateTimestamp: currentTimestamp,
			validateTimestamp: currentTimestamp + 20,
			isSecretExpired:   false,
		},
		{
			name:              "2min",
			generateTimestamp: currentTimestamp,
			validateTimestamp: currentTimestamp + 2*secondsInMinute,
			isSecretExpired:   false,
		},
		{
			name:              "4min - 1s",
			generateTimestamp: currentTimestamp,
			validateTimestamp: currentTimestamp + 4*secondsInMinute - 1,
			isSecretExpired:   false,
		},
		{
			name:              "5min",
			generateTimestamp: currentTimestamp,
			validateTimestamp: currentTimestamp + 5*secondsInMinute,
			isSecretExpired:   true,
		},
		{
			name:              "5min + 1s",
			generateTimestamp: currentTimestamp,
			validateTimestamp: currentTimestamp + 5*secondsInMinute + 1,
			isSecretExpired:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			secret, err := GenerateSecret(accountName, test.generateTimestamp)
			assert.NoError(t, err)
			pastSecrets, err := GetPastSecrets(accountName, test.validateTimestamp)
			assert.NoError(t, err)
			assert.Equal(t, test.isSecretExpired, !slices.Contains(pastSecrets, secret))
		})
	}
}
