package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
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

func TestBuildExecuteCommandsNormalizesDelimiter(t *testing.T) {
	statement := "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\nDELIMITER ;\n"

	commands, err := buildExecuteCommands(statement)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.NotContains(t, commands[0].Text, "DELIMITER")
	require.Contains(t, commands[0].Text, "CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND")
}

func TestBuildExecuteCommandsDoesNotNormalizeDelimiterForTooManyCommands(t *testing.T) {
	var statement strings.Builder
	statement.WriteString("DELIMITER //\n")
	statement.WriteString("CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\n")
	statement.WriteString("DELIMITER ;\n")
	statement.WriteString("/*!50003 SET @OLD_SQL_MODE=@@SQL_MODE */;\n")
	for i := 0; i < common.MaximumCommands; i++ {
		statement.WriteString("SELECT 1;\n")
	}

	commands, err := buildExecuteCommands(statement.String())
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, statement.String(), commands[0].Text)
}

func TestBuildExecuteCommandsDoesNotNormalizeDelimiterForLargeSheet(t *testing.T) {
	statement := "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\nDELIMITER ;\n" +
		strings.Repeat(" ", common.MaxSheetCheckSize)

	commands, err := buildExecuteCommands(statement)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, statement, commands[0].Text)
}
