package mysql

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytebase/bytebase/api"
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

func TestCheckVersionForPITR(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		version string
		err     bool
	}{
		{
			version: "5.6.1",
			err:     true,
		},
		{
			version: "5.7.0",
			err:     true,
		},
		{
			version: "8.0.28",
			err:     false,
		},
		{
			version: "8.0.28-debug",
			err:     false,
		},
		{
			version: "invalid.semver",
			err:     true,
		},
	}

	for _, test := range tests {
		err := checkVersionForPITR(test.version)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestGetBinlogFileNameSeqNumber(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		name   string
		prefix string
		expect int64
		err    bool
	}{
		{
			name:   "binlog.096865",
			expect: 96865,
			err:    false,
		},
		{
			name:   "binlog.178923",
			expect: 178923,
			err:    false,
		},
		{
			name:   "binlog.000001",
			expect: 1,
			err:    false,
		},
		{
			name:   "binlog.x8dft",
			expect: 0,
			err:    true,
		},
	}
	for _, test := range tests {
		ext, err := getBinlogNameExtension(test.name)
		a.EqualValues(ext, test.expect)
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
'/*!*/;
# at 24500
#220425 17:32:13 server id 1  end_log_pos 24604 CRC32 0x6a465388 	Table_map: ` + "`bytebase`.`migration_history`" + ` mapped to number 172
WARNING: The range of printed events ends with a row event or a table map event that does not have the STMT_END_F flag set. This might be because the last statement was not fully written to the log, or because you are using a --stop-position or --stop-datetime that refers to an event in the middle of a statement. The event(s) from the partial statement have not been written to output.
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
DELIMITER ;
# End of log file
/*!50003 SET COMPLETION_TYPE=@OLD_COMPLETION_TYPE*/;
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=0*/;`,
			timestamp: time.Date(2022, 4, 21, 14, 49, 26, 0, time.Local).Unix(),
			err:       false,
		},
		// Edge case: invalid mysqlbinlog option
		{
			binlogText: "mysqlbinlog: [ERROR] mysqlbinlog: unknown option '-n'.",
			timestamp:  0,
			err:        true,
		},
	}

	for _, test := range tests {
		timestamp, err := parseBinlogEventTimestampImpl(test.binlogText)
		a.Equal(test.timestamp, timestamp)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestGetLatestBackupBeforeOrEqualTsImpl(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		backupList   []*api.Backup
		eventTsList  []int64
		targetTs     int64
		targetBackup *api.Backup
		err          bool
	}{
		// normal case
		{
			backupList: []*api.Backup{
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 10}}},
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 20}}},
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000002", Position: 10}}},
			},
			eventTsList: []int64{1, 2, 3},
			targetTs:    2,
			targetBackup: &api.Backup{
				Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 20}},
			},
			err: false,
		},
		// backup list not in sorted order
		{
			backupList: []*api.Backup{
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 20}}},
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 10}}},
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000002", Position: 10}}},
			},
			eventTsList: []int64{2, 1, 3},
			targetTs:    2,
			targetBackup: &api.Backup{
				Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 20}},
			},
			err: false,
		},
		// backup with empty binlog info does not count
		{
			backupList: []*api.Backup{
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 10}}},
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{}}},
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000002", Position: 10}}},
			},
			eventTsList: []int64{1, 2, 3},
			targetTs:    2,
			targetBackup: &api.Backup{
				Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 10}},
			},
			err: false,
		},
		// no valid backup found
		{
			backupList: []*api.Backup{
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 10}}},
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000001", Position: 20}}},
				{Payload: api.BackupPayload{BinlogInfo: api.BinlogInfo{FileName: "binlog.000002", Position: 10}}},
			},
			eventTsList:  []int64{1, 2, 3},
			targetTs:     0,
			targetBackup: nil,
			err:          true,
		},
	}
	for _, test := range tests {
		backup, err := getLatestBackupBeforeOrEqualTsImpl(test.backupList, test.eventTsList, test.targetTs)
		a.Equal(test.targetBackup, backup)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestGetReplayBinlogPathList(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		subDirNames     []string
		binlogFileNames []string
		startBinlogInfo api.BinlogInfo
		expect          []string
		err             bool
	}{
		{
			// Test skip directory
			subDirNames:     []string{"subdir_a", "subdir_b"},
			binlogFileNames: []string{},
			startBinlogInfo: api.BinlogInfo{},
			expect:          []string{},
			err:             false,
		},
		{
			// Test skip stale binlog
			subDirNames:     []string{},
			binlogFileNames: []string{"binlog.000001", "binlog.000002", "binlog.000003"},
			startBinlogInfo: api.BinlogInfo{
				FileName: "binlog.000002",
				Position: 0xdeadbeaf,
			},
			expect: []string{"binlog.000002", "binlog.000003"},
			err:    false,
		},
		{
			// Test binlogs no hole
			subDirNames:     []string{},
			binlogFileNames: []string{"binlog.000001", "binlog.000002", "binlog.000004"},
			startBinlogInfo: api.BinlogInfo{
				FileName: "binlog.000002",
				Position: 0xdeadbeaf,
			},
			expect: []string{},
			err:    true,
		},
		{
			// Test mysql-bin prefix
			subDirNames:     []string{},
			binlogFileNames: []string{"mysql-bin.000001", "mysql-bin.000002", "mysql-bin.000003"},
			startBinlogInfo: api.BinlogInfo{
				FileName: "bin.000001",
				Position: 0xdeadbeaf,
			},
			expect: []string{"mysql-bin.000001", "mysql-bin.000002", "mysql-bin.000003"},
			err:    false,
		},
		{
			// Test out of binlog.999999
			subDirNames:     []string{},
			binlogFileNames: []string{"binlog.999999", "binlog.1000000", "binlog.1000001"},
			startBinlogInfo: api.BinlogInfo{
				FileName: "binlog.999999",
				Position: 0xdeadbeaf,
			},
			expect: []string{"binlog.999999", "binlog.1000000", "binlog.1000001"},
			err:    false,
		},
		{
			subDirNames:     []string{"sub_dir"},
			binlogFileNames: []string{"binlog.000001", "binlog.000002", "binlog.000003"},
			startBinlogInfo: api.BinlogInfo{
				FileName: "binlog.000002",
				Position: 0xdeadbeaf,
			},
			expect: []string{"binlog.000002", "binlog.000003"},
			err:    false,
		},
	}

	for _, test := range tests {
		tmpDir := t.TempDir()

		for _, subDir := range test.subDirNames {
			err := os.MkdirAll(filepath.Join(tmpDir, subDir), os.ModePerm)
			a.NoError(err)
		}

		for _, binlogFileName := range test.binlogFileNames {
			f, err := os.Create(filepath.Join(tmpDir, binlogFileName))
			a.NoError(err)
			err = f.Close()
			a.NoError(err)
		}

		result, err := getBinlogReplayList(test.startBinlogInfo, tmpDir)
		if test.err {
			a.Error(err)
		} else {
			a.EqualValues(len(result), len(test.expect))
			for idx := range test.expect {
				a.EqualValues(result[idx], filepath.Join(tmpDir, test.expect[idx]))
			}
		}
	}
}
