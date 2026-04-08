package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateServiceAccountEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{
			name:  "valid",
			email: "bot@service.bytebase.com",
			valid: true,
		},
		{
			name:  "invalid syntax",
			email: "Bot@service.bytebase.com",
			valid: false,
		},
		{
			name:  "wrong suffix",
			email: "bot@example.com",
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateServiceAccountEmail(test.email)
			if test.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestValidateWorkloadIdentityEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{
			name:  "valid",
			email: "bot@workload.bytebase.com",
			valid: true,
		},
		{
			name:  "invalid syntax",
			email: "Bot@workload.bytebase.com",
			valid: false,
		},
		{
			name:  "wrong suffix",
			email: "bot@example.com",
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateWorkloadIdentityEmail(test.email)
			if test.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}
