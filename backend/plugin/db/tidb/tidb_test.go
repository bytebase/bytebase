package tidb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestGetTiDBConnectionUsesExtraConnectionParameters(t *testing.T) {
	d := &Driver{}
	dsn, err := d.getTiDBConnection(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "4000",
			Username: "root",
			ExtraConnectionParameters: map[string]string{
				"readTimeout":  "30s",
				"writeTimeout": "45s",
			},
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "test",
		},
	})
	require.NoError(t, err)
	require.Contains(t, dsn, "readTimeout=30s")
	require.Contains(t, dsn, "writeTimeout=45s")
}

func TestGetTiDBConnectionRejectsAllowAllFiles(t *testing.T) {
	d := &Driver{}
	_, err := d.getTiDBConnection(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "4000",
			Username: "root",
			ExtraConnectionParameters: map[string]string{
				"allowAllFiles": "true",
			},
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "test",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowAllFiles")
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{
			version: "8.0.11-TiDB-v8.5.0",
			want:    "v8.5.0",
		},
		{
			version: "8.0.11-TiDB-v7.5.2-serverless",
			want:    "v7.5.2",
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		version, err := parseVersion(tc.version)
		a.NoError(err)
		a.Equal(tc.want, version)
	}
}

func TestTiDBVersionAtLeast(t *testing.T) {
	tests := []struct {
		version   string
		threshold string
		want      bool
	}{
		{version: "v7.1.1", threshold: "7.4.0", want: false},
		{version: "v7.2.0", threshold: "7.4.0", want: false},
		{version: "v7.3.0", threshold: "7.4.0", want: false},
		{version: "v7.4.0", threshold: "7.4.0", want: true},
		{version: "v7.5.2", threshold: "7.4.0", want: true},
		{version: "v8.0.0", threshold: "7.4.0", want: true},
		{version: "v8.5.0", threshold: "7.4.0", want: true},
	}

	a := require.New(t)
	for _, tc := range tests {
		got, err := tidbVersionAtLeast(tc.version, tc.threshold)
		a.NoError(err)
		a.Equal(tc.want, got, "version=%s threshold=%s", tc.version, tc.threshold)
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
