package otp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func TestGenerateTimeBasedSecret(t *testing.T) {
	tests := []struct {
		name              string
		generateTimestamp int64
		validateTimestamp int64
		isSecretOutdate   bool
	}{
		{
			name:              "20s",
			generateTimestamp: 1620000000,
			validateTimestamp: 1620000000 + 20,
			isSecretOutdate:   false,
		},
		{
			name:              "2min",
			generateTimestamp: 1620000000,
			validateTimestamp: 1620000000 + 2*secondsInMinute,
			isSecretOutdate:   false,
		},
		{
			name:              "4min - 1s",
			generateTimestamp: 1620000000,
			validateTimestamp: 1620000000 + 4*secondsInMinute - 1,
			isSecretOutdate:   false,
		},
		{
			name:              "5min",
			generateTimestamp: 1620000000,
			validateTimestamp: 1620000000 + 5*secondsInMinute,
			isSecretOutdate:   true,
		},
		{
			name:              "5min + 1s",
			generateTimestamp: 1620000000,
			validateTimestamp: 1620000000 + 5*secondsInMinute + 1,
			isSecretOutdate:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			secret, err := GenerateSecret("test-user", test.generateTimestamp)
			assert.NoError(t, err)
			pastSecrets, err := GetPastSecrets("test-user", test.validateTimestamp)
			assert.NoError(t, err)
			assert.Equal(t, test.isSecretOutdate, !slices.Contains(pastSecrets, secret))
		})
	}
}
