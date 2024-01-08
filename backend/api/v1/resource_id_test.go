package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateResourceID(t *testing.T) {
	a := require.New(t)

	testCases := []struct {
		resourceID string
		wantErr    bool
	}{
		{
			resourceID: "",
			wantErr:    true,
		},
		{
			resourceID: "a",
			wantErr:    false,
		},
		{
			resourceID: "A",
			wantErr:    true,
		},
		{
			resourceID: "a-1",
			wantErr:    false,
		},
		{
			resourceID: "a-1-",
			wantErr:    true,
		},
		{
			resourceID: "1a",
			wantErr:    true,
		},
		{
			resourceID: "environment-ds8gr3lx",
			wantErr:    false,
		},
	}

	for _, tc := range testCases {
		err := validateResourceID(tc.resourceID)
		if tc.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}
