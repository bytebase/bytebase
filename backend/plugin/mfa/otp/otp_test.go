package otp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func TestGenerateTimeBasedSecret(t *testing.T) {
	currentTimestamp := time.Now()
	accountName := "test-user"

	tests := []struct {
		name              string
		generateTimestamp time.Time
		validateTimestamp time.Time
		isSecretExpired   bool
	}{
		{
			name:              "-5min",
			generateTimestamp: currentTimestamp,
			validateTimestamp: time.Unix(currentTimestamp.Unix()-5*secondsInMinute, 0),
			isSecretExpired:   true,
		},
		{
			name:              "20s",
			generateTimestamp: currentTimestamp,
			validateTimestamp: time.Unix(currentTimestamp.Unix()+20, 0),
			isSecretExpired:   false,
		},
		{
			name:              "2min",
			generateTimestamp: currentTimestamp,
			validateTimestamp: time.Unix(currentTimestamp.Unix()+2*secondsInMinute, 0),
			isSecretExpired:   false,
		},
		{
			name:              "4min - 1s",
			generateTimestamp: currentTimestamp,
			validateTimestamp: time.Unix(currentTimestamp.Unix()+4*secondsInMinute-1, 0),
			isSecretExpired:   false,
		},
		{
			name:              "5min",
			generateTimestamp: currentTimestamp,
			validateTimestamp: time.Unix(currentTimestamp.Unix()+5*secondsInMinute, 0),
			isSecretExpired:   true,
		},
		{
			name:              "5min + 1s",
			generateTimestamp: currentTimestamp,
			validateTimestamp: time.Unix(currentTimestamp.Unix()+5*secondsInMinute+1, 0),
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
