package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDomains(t *testing.T) {
	a := require.New(t)

	testCases := []struct {
		domain  string
		wantErr bool
	}{
		{
			domain:  "",
			wantErr: true,
		},
		{
			domain:  "hello@world.com",
			wantErr: true,
		},
		{
			domain:  "BYTEBASE.COM",
			wantErr: true,
		},
		{
			domain:  "bytebase.com",
			wantErr: false,
		},
		{
			domain:  "x.y",
			wantErr: true,
		},
		{
			domain:  "abc.xyz",
			wantErr: false,
		},
		{
			domain:  "gmail.com",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		err := validateDomains([]string{tc.domain})
		if tc.wantErr {
			a.Error(err, tc.domain)
		} else {
			a.NoError(err, tc.domain)
		}
	}
}
