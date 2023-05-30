package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPgVersionAndRedshiftVersion(t *testing.T) {
	testCases := []struct {
		version      string
		wantPg       string
		wantRedshift string
		wantErr      bool
	}{
		{
			version:      "PostgreSQL 8.0.2 on i686-pc-linux-gnu, compiled by GCC gcc (GCC) 3.4.2 20041017 (Red Hat 3.4.2-6.fc3), Redshift 1.0.48042",
			wantPg:       "8.0.2",
			wantRedshift: "1.0.48042",
			wantErr:      false,
		},
	}

	for _, tc := range testCases {
		pg, redshift, err := getPgVersionAndRedshiftVersion(tc.version)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.wantPg, pg)
			require.Equal(t, tc.wantRedshift, redshift)
		}
	}
}
