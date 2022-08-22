package mysql

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db/util"
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
			suffix:   "del",
			expected: "normal_database_name_del",
		},
		{
			baseName: "long_database_name1234567890123456789012345678901",
			suffix:   "pitr_1652237293",
			expected: "long_database_name123456789012345678901234567890_pitr_1652237293",
		},
	}

	for _, test := range tests {
		safeName := util.GetSafeName(test.baseName, test.suffix)
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
		name := util.GetPITRDatabaseName(test.database, int64(test.timestamp))
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
			expected:  "normal_database_name_pitr_1652237293_del",
		},
		{
			database:  "long_database_name123456789012345678901234567890",
			timestamp: 1652237293,
			expected:  "long_database_name12345678901234567890123456_pitr_1652237293_del",
		},
	}

	for _, test := range tests {
		name := util.GetPITROldDatabaseName(test.database, int64(test.timestamp))
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
		ext, err := GetBinlogNameSeq(test.name)
		a.EqualValues(test.expect, ext)
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

		result, err := GetBinlogReplayList(test.startBinlogInfo, tmpDir)
		if test.err {
			a.Error(err)
		} else {
			a.EqualValues(len(test.expect), len(result))
			for idx := range test.expect {
				a.EqualValues(filepath.Join(tmpDir, test.expect[idx]), result[idx])
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
			a.Equal(test.expect[i].Seq, sorted[i].Seq)
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

func TestParseBinlogEventTsInLine(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		line    string
		eventTs int64
		found   bool
		err     bool
	}{
		// normal case
		{
			line: "#220620 13:23:55 server id 1  end_log_pos 126 CRC32 0x9a60fe57 	Start: binlog v 4, server v 8.0.28 created 220620 13:23:55 at startup",
			eventTs: time.Date(2022, 6, 20, 13, 23, 55, 0, time.Local).Unix(),
			found:   true,
			err:     false,
		},
		// no "server id"
		{
			line:    "/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;",
			eventTs: 0,
			found:   false,
			err:     false,
		},
		// fake event with "end_log_pos 0"
		{
			line: "#220609 11:59:57 server id 1  end_log_pos 0 CRC32 0x031d41f6 	Start: binlog v 4, server v 8.0.28 created 220609 11:59:57",
			eventTs: 0,
			found:   false,
			err:     false,
		},
		// incomplete line
		{
			line:    "#220620 13:23:55 server id 1  end_log_",
			eventTs: 0,
			found:   false,
			err:     true,
		},
		// parse time error
		{
			line: "#220620 99:99:99 server id 1  end_log_pos 126 CRC32 0x9a60fe57 	Start: binlog v 4, server v 8.0.28 created 220620 13:23:55 at startup",
			eventTs: 0,
			found:   false,
			err:     true,
		},
	}
	for _, test := range tests {
		eventTs, found, err := parseBinlogEventTsInLine(test.line)
		a.Equal(test.eventTs, eventTs)
		a.Equal(test.found, found)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestParseBinlogEventPosInLine(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		line  string
		pos   int64
		found bool
		err   bool
	}{
		// normal case
		{
			line:  "# at 34794",
			pos:   34794,
			found: true,
			err:   false,
		},
		// no "# at "
		{
			line:  "/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;",
			pos:   0,
			found: false,
			err:   false,
		},
		// incomplete line
		{
			line:  "# at ",
			pos:   0,
			found: false,
			err:   true,
		},
		// parse int error
		{
			line:  "# at x",
			pos:   0,
			found: false,
			err:   true,
		},
	}
	for _, test := range tests {
		pos, found, err := parseBinlogEventPosInLine(test.line)
		a.Equal(test.pos, pos)
		a.Equal(test.found, found)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}
