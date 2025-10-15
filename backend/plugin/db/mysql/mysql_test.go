package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateMySQLExtraConnectionParameters(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]string
		wantError bool
	}{
		{
			name:      "empty parameters",
			params:    map[string]string{},
			wantError: false,
		},
		{
			name: "safe parameters",
			params: map[string]string{
				"timeout":      "10s",
				"readTimeout":  "30s",
				"writeTimeout": "30s",
			},
			wantError: false,
		},
		{
			name: "dangerous parameter - allowAllFiles",
			params: map[string]string{
				"allowAllFiles": "true",
			},
			wantError: true,
		},
		{
			name: "dangerous parameter - allowAllFiles lowercase",
			params: map[string]string{
				"allowallfiles": "true",
			},
			wantError: true,
		},
		{
			name: "dangerous parameter - allowAllFiles with mixed case",
			params: map[string]string{
				"AllowAllFiles": "true",
			},
			wantError: true,
		},
		{
			name: "mixed safe and dangerous parameters",
			params: map[string]string{
				"timeout":       "10s",
				"allowAllFiles": "true",
			},
			wantError: true,
		},
		{
			name: "parameter with whitespace",
			params: map[string]string{
				"  allowAllFiles  ": "true",
			},
			wantError: true,
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		t.Run(tc.name, func(_ *testing.T) {
			err := validateMySQLExtraConnectionParameters(tc.params)
			if tc.wantError {
				a.Error(err, "expected error for test case: %s", tc.name)
			} else {
				a.NoError(err, "expected no error for test case: %s", tc.name)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version  string
		want     string
		wantRest string
	}{
		{
			version:  "8.0.27",
			want:     "8.0.27",
			wantRest: "",
		},
		{
			version:  "5.7.22-log",
			want:     "5.7.22",
			wantRest: "-log",
		},
		{
			version:  "5.6.29_ddm_3.0.1.7",
			want:     "5.6.29",
			wantRest: "_ddm_3.0.1.7",
		},
		{
			version:  "10.4.7-MariaDB",
			want:     "10.4.7",
			wantRest: "-MariaDB",
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		version, rest, err := parseVersion(tc.version)
		a.NoError(err)
		a.Equal(tc.want, version)
		a.Equal(tc.wantRest, rest)
	}
}
