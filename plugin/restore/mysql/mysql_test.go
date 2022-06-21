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
			name:   "binlog.999999",
			expect: 999999,
			err:    false,
		},
		{
			name:   "binlog.1000000",
			expect: 1000000,
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
		ext, err := getBinlogNameSeq(test.name)
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
		startLine  int
		event      binlogEvent
		err        bool
	}{
		// A real one from `mysqlbinlog --start-position 24500 --stop-position 24501`.
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
			startLine: 0,
			event:     binlogEvent{ts: time.Date(2022, 4, 25, 17, 32, 13, 0, time.Local).Unix(), startPos: 24500},
			err:       false,
		},
		// A real one from `mysqlbinlog --start-datetime`.
		{
			binlogText: `# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
DELIMITER /*!*/;
# at 4
#220620 13:23:55 server id 1  end_log_pos 126 CRC32 0x9a60fe57 	Start: binlog v 4, server v 8.0.28 created 220620 13:23:55 at startup
ROLLBACK/*!*/;
BINLOG '
awSwYg8BAAAAegAAAH4AAAAAAAQAOC4wLjI4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAABrBLBiEwANAAgAAAAABAAEAAAAYgAEGggAAAAICAgCAAAACgoKKioAEjQA
CigAAVf+YJo=
'/*!*/;
# at 428
#220620 17:00:23 server id 1  end_log_pos 507 CRC32 0x7dbdc0dc 	Anonymous_GTID	last_committed=1	sequence_number=2	rbr_only=yes	original_committed_timestamp=1655715623024386	immediate_commit_timestamp=1655715623024386	transaction_length=271
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1655715623024386 (2022-06-20 17:00:23.024386 CST)
# immediate_commit_timestamp=1655715623024386 (2022-06-20 17:00:23.024386 CST)
/*!80001 SET @@session.original_commit_timestamp=1655715623024386*//*!*/;
/*!80014 SET @@session.original_server_version=80028*//*!*/;
/*!80014 SET @@session.immediate_server_version=80028*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;`,
			startLine: 7,
			event:     binlogEvent{ts: time.Date(2022, 6, 20, 17, 00, 23, 0, time.Local).Unix(), startPos: 428},
			err:       false,
		},
		// Edge case: invalid mysqlbinlog option
		{
			binlogText: "mysqlbinlog: [ERROR] mysqlbinlog: unknown option '-n'.",
			startLine:  0,
			event:      binlogEvent{},
			err:        true,
		},
	}

	for _, test := range tests {
		event, _, err := parseBinlogEventImpl(test.binlogText, test.startLine)
		a.Equal(test.event.ts, event.ts)
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

func TestSortBinlogFiles(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		binlogFileNames []BinlogFile
		expect          []BinlogFile
	}{
		// Normal
		{
			binlogFileNames: []BinlogFile{{Seq: 1}, {Seq: 2}, {Seq: 3}},
			expect:          []BinlogFile{{Seq: 1}, {Seq: 2}, {Seq: 3}},
		},
		// gap
		{
			binlogFileNames: []BinlogFile{{Seq: 1}, {Seq: 7}, {Seq: 4}},
			expect:          []BinlogFile{{Seq: 1}, {Seq: 4}, {Seq: 7}},
		},
		// Empty
		{
			binlogFileNames: []BinlogFile{},
			expect:          []BinlogFile{},
		},
	}

	for _, test := range tests {
		sorted := sortBinlogFiles(test.binlogFileNames)
		a.Equal(len(test.expect), len(sorted))
		for i := range sorted {
			a.Equal(sorted[i].Seq, test.expect[i].Seq)
		}
	}
}

func TestBinlogFilesAreContinuous(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		binlogFiles []BinlogFile
		expect      bool
	}{
		{
			binlogFiles: []BinlogFile{},
			expect:      true,
		},
		{
			binlogFiles: []BinlogFile{{Seq: 1}},
			expect:      true,
		},
		{
			binlogFiles: []BinlogFile{{Seq: 1}, {Seq: 2}},
			expect:      true,
		},
		{
			binlogFiles: []BinlogFile{{Seq: 1}, {Seq: 3}},
			expect:      false,
		},
	}
	for _, test := range tests {
		result := binlogFilesAreContinuous(test.binlogFiles)
		a.Equal(test.expect, result)
	}
}

func TestGetLastTsIndexBeforeTargetTs(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		firstEventTsList []int64
		targetTs         int64
		expect           int
		err              bool
	}{
		{
			firstEventTsList: []int64{1, 2, 3},
			targetTs:         0,
			expect:           0,
			err:              true,
		},
		{
			firstEventTsList: []int64{1, 2, 3},
			targetTs:         2,
			expect:           1,
			err:              false,
		},
		{
			firstEventTsList: []int64{1, 2, 3},
			targetTs:         3,
			expect:           2,
			err:              false,
		},
		{
			firstEventTsList: []int64{1, 2, 3},
			targetTs:         4,
			expect:           2,
			err:              false,
		},
	}
	for _, test := range tests {
		i, err := getLastTsIndexBeforeTargetTs(test.firstEventTsList, test.targetTs)
		a.Equal(test.expect, i)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}
