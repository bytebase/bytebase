package mysql

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetSafeName(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		baseName string
		suffix   string
		expected string
	}{
		{
			baseName: "normal_database_name",
			suffix:   "pitr_1652237293",
			expected: "normal_database_name_pitr_1652237293",
		},
		{
			baseName: "normal_database_name",
			suffix:   "old",
			expected: "normal_database_name_old",
		},
		{
			baseName: "long_database_name1234567890123456789012345678901",
			suffix:   "pitr_1652237293",
			expected: "long_database_name123456789012345678901234567890_pitr_1652237293",
		},
	}

	for _, test := range tests {
		safeName := getSafeName(test.baseName, test.suffix)
		a.Equal(test.expected, safeName)
	}
}

func TestGetPITRDatabaseName(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		database  string
		timestamp int
		expected  string
	}{
		{
			database:  "normal_database_name",
			timestamp: 1652237293,
			expected:  "normal_database_name_pitr_1652237293",
		},
		{
			database:  "long_database_name1234567890123456789012345678901",
			timestamp: 1652237293,
			expected:  "long_database_name123456789012345678901234567890_pitr_1652237293",
		},
	}

	for _, test := range tests {
		name := getPITRDatabaseName(test.database, int64(test.timestamp))
		a.Equal(test.expected, name)
	}
}

func TestGetPITROldDatabaseName(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		database  string
		timestamp int
		expected  string
	}{
		{
			database:  "normal_database_name",
			timestamp: 1652237293,
			expected:  "normal_database_name_pitr_1652237293_old",
		},
		{
			database:  "long_database_name123456789012345678901234567890",
			timestamp: 1652237293,
			expected:  "long_database_name12345678901234567890123456_pitr_1652237293_old",
		},
	}

	for _, test := range tests {
		name := getPITROldDatabaseName(test.database, int64(test.timestamp))
		a.Equal(test.expected, name)
	}
}

func TestParseBinlogFileNameIndex(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		filename string
		expected int64
		err      bool
	}{
		{
			filename: "binlog.000001",
			expected: 1,
			err:      false,
		},
		{
			filename: "binlog.000001.ext",
			expected: -1,
			err:      true,
		},
		{
			filename: "binlog.ext",
			expected: -1,
			err:      true,
		},
	}

	for _, test := range tests {
		index, err := parseBinlogFileNameIndex(test.filename)
		a.Equal(test.expected, index)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestParseBinlogEventTimestampFromTextOutputLine(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		line      string
		timestamp int64
		found     bool
		err       bool
	}{
		{
			line:      "#220421 14:49:26",
			timestamp: time.Date(2022, 4, 21, 14, 49, 26, 0, time.UTC).Unix(),
			found:     true,
			err:       false,
		},
		{
			line:      "220421 14:49:26",
			timestamp: -1,
			found:     false,
			err:       false,
		},
	}

	for _, test := range tests {
		timestamp, found, err := parseBinlogEventTimestampFromTextOutputLine(test.line)
		a.Equal(test.timestamp, timestamp)
		a.Equal(test.found, found)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestParseBinlogEventTimestampImpl(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		binlogText string
		timestamp  int64
		err        bool
	}{
		// This is a real one from mysqlbinlog output.
		{
			binlogText: `# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
DELIMITER /*!*/;
# at 24500
#220421 14:49:26 server id 1  end_log_pos 0 CRC32 0xbb5866d6 	Start: binlog v 4, server v 8.0.28 created 220421 14:49:26
BINLOG '
dv5gYg8BAAAAegAAAAAAAAAAAAQAOC4wLjI4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAEwANAAgAAAAABAAEAAAAYgAEGggAAAAICAgCAAAACgoKKioAEjQA
CigAAdZmWLs=
'/*!*/;`,
			timestamp: time.Date(2022, 4, 21, 14, 49, 26, 0, time.UTC).Unix(),
			err:       false,
		},
		// Cases are that there are multiple occurrence of timestamp, we only consider the first.
		{
			binlogText: `# at 24500
#220425 17:32:13 server id 1  end_log_pos 24604 CRC32 0x6a465388 	Table_map: ` + "`bytebase`.`migration_history`" + ` mapped to number 172
# at 24604
#220425 17:32:14 server id 1  end_log_pos 25671 CRC32 0xfc11091b 	Update_rows: table id 172 flags: STMT_END_F`,
			timestamp: time.Date(2022, 4, 25, 17, 32, 13, 0, time.UTC).Unix(),
			err:       false,
		},
		// Edge case: empty input
		{
			binlogText: "",
			timestamp:  -1,
			err:        true,
		},
	}

	for _, test := range tests {
		timestamp, err := parseBinlogEventTimestampImpl(strings.NewReader(test.binlogText))
		a.Equal(test.timestamp, timestamp)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}
