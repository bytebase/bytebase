package mysql

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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
			version: "8.0.33",
			err:     false,
		},
		{
			version: "8.0.33-debug",
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
		base   string
		seq    int64
		err    bool
	}{
		{
			name: "binlog.096865",
			base: "binlog",
			seq:  96865,
			err:  false,
		},
		{
			name: "binlog.999999",
			base: "binlog",
			seq:  999999,
			err:  false,
		},
		{
			name: "binlog.1000000",
			base: "binlog",
			seq:  1000000,
			err:  false,
		},
		{
			name: "binlog.000001",
			base: "binlog",
			seq:  1,
			err:  false,
		},
		{
			name: "binlog.x8dft",
			base: "",
			seq:  0,
			err:  true,
		},
	}
	for _, test := range tests {
		base, ext, err := ParseBinlogName(test.name)
		a.EqualValues(test.seq, ext)
		a.Equal(test.base, base)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
		}
	}
}

func TestGenBinlogFileNames(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		seqStart int64
		seqEnd   int64
		want     []string
	}{
		{
			name:     "empty",
			seqStart: 2,
			seqEnd:   1,
			want:     nil,
		},
		{
			name:     "single",
			base:     "binlog",
			seqStart: 1,
			seqEnd:   1,
			want:     []string{"binlog.000001"},
		},
		{
			name:     "less than 6 digits",
			base:     "binlog",
			seqStart: 1,
			seqEnd:   4,
			want:     []string{"binlog.000001", "binlog.000002", "binlog.000003", "binlog.000004"},
		},
		{
			name:     "more than 6 digits",
			base:     "binlog",
			seqStart: 1000001,
			seqEnd:   1000004,
			want:     []string{"binlog.1000001", "binlog.1000002", "binlog.1000003", "binlog.1000004"},
		},
	}

	for _, test := range tests {
		a := require.New(t)
		result := GenBinlogFileNames(test.base, test.seqStart, test.seqEnd)
		a.Equal(test.want, result)
	}
}

func TestGetReplayBinlogPathList(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		binlogFileNames  []string
		startBinlogInfo  api.BinlogInfo
		targetBinlogInfo api.BinlogInfo
		expect           []string
		err              bool
	}{
		{
			// skip stale binlog
			binlogFileNames:  []string{"binlog.000001", "binlog.000002", "binlog.000003"},
			startBinlogInfo:  api.BinlogInfo{FileName: "binlog.000002"},
			targetBinlogInfo: api.BinlogInfo{FileName: "binlog.000002"},
			expect:           []string{"binlog.000002"},
			err:              false,
		},
		{
			// binlog files not continuous
			binlogFileNames:  []string{"binlog.000001", "binlog.000002", "binlog.000004"},
			startBinlogInfo:  api.BinlogInfo{FileName: "binlog.000002"},
			targetBinlogInfo: api.BinlogInfo{FileName: "binlog.000004"},
			expect:           nil,
			err:              true,
		},
		{
			// binlog files not continuous, but replayed list is continuous
			binlogFileNames:  []string{"binlog.000001", "binlog.000002", "binlog.000004"},
			startBinlogInfo:  api.BinlogInfo{FileName: "binlog.000001"},
			targetBinlogInfo: api.BinlogInfo{FileName: "binlog.000002"},
			expect:           []string{"binlog.000001", "binlog.000002"},
			err:              false,
		},
		{
			// mysql-bin prefix
			binlogFileNames:  []string{"mysql-bin.000001", "mysql-bin.000002", "mysql-bin.000003"},
			startBinlogInfo:  api.BinlogInfo{FileName: "mysql-bin.000001"},
			targetBinlogInfo: api.BinlogInfo{FileName: "mysql-bin.000002"},
			expect:           []string{"mysql-bin.000001", "mysql-bin.000002"},
			err:              false,
		},
		{
			// out of binlog.999999
			binlogFileNames:  []string{"binlog.999999", "binlog.1000000", "binlog.1000001"},
			startBinlogInfo:  api.BinlogInfo{FileName: "binlog.999999"},
			targetBinlogInfo: api.BinlogInfo{FileName: "binlog.1000001"},
			expect:           []string{"binlog.999999", "binlog.1000000", "binlog.1000001"},
			err:              false,
		},
		{
			// start seq not exist
			binlogFileNames:  []string{"binlog.000001", "binlog.000003", "binlog.000004"},
			startBinlogInfo:  api.BinlogInfo{FileName: "binlog.000002"},
			targetBinlogInfo: api.BinlogInfo{FileName: "binlog.000003"},
			expect:           nil,
			err:              true,
		},
		{
			// target seq not exist
			binlogFileNames:  []string{"binlog.000001", "binlog.000003", "binlog.000004"},
			startBinlogInfo:  api.BinlogInfo{FileName: "binlog.000001"},
			targetBinlogInfo: api.BinlogInfo{FileName: "binlog.000002"},
			expect:           nil,
			err:              true,
		},
	}

	for _, test := range tests {
		tmpDir := t.TempDir()

		for _, binlogFileName := range test.binlogFileNames {
			f, err := os.Create(filepath.Join(tmpDir, binlogFileName+binlogMetaSuffix))
			a.NoError(err)
			content, err := json.Marshal(binlogFileMeta{})
			a.NoError(err)
			_, err = f.Write(content)
			a.NoError(err)
			err = f.Close()
			a.NoError(err)
		}

		result, err := GetBinlogReplayList(test.startBinlogInfo, test.targetBinlogInfo, tmpDir)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
			a.EqualValues(len(test.expect), len(result))
			for idx := range test.expect {
				a.EqualValues(filepath.Join(tmpDir, test.expect[idx]), result[idx])
			}
		}
	}
}

func TestGetMetaReplayList(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		metaList  []binlogFileMeta
		startSeq  int64
		targetSeq int64
		expect    []binlogFileMeta
		err       bool
	}{
		{
			metaList:  []binlogFileMeta{{seq: 1}, {seq: 2}, {seq: 3}},
			startSeq:  1,
			targetSeq: 3,
			expect:    []binlogFileMeta{{seq: 1}, {seq: 2}, {seq: 3}},
			err:       false,
		},
		{
			metaList:  []binlogFileMeta{{seq: 1}, {seq: 3}},
			startSeq:  1,
			targetSeq: 3,
			expect:    []binlogFileMeta{{seq: 1}, {seq: 3}},
			err:       false,
		},
		{
			metaList:  []binlogFileMeta{{seq: 1}, {seq: 2}, {seq: 3}},
			startSeq:  1,
			targetSeq: 1,
			expect:    []binlogFileMeta{{seq: 1}},
			err:       false,
		},
		{
			metaList:  []binlogFileMeta{{seq: 1}, {seq: 2}, {seq: 3}},
			startSeq:  3,
			targetSeq: 1,
			expect:    nil,
			err:       true,
		},
		{
			metaList:  []binlogFileMeta{{seq: 1}, {seq: 2}, {seq: 3}},
			startSeq:  1,
			targetSeq: 4,
			expect:    nil,
			err:       true,
		},
		{
			metaList:  []binlogFileMeta{{seq: 1}, {seq: 2}, {seq: 3}},
			startSeq:  0,
			targetSeq: 3,
			expect:    nil,
			err:       true,
		},
	}

	for _, test := range tests {
		result, err := getMetaReplayList(test.metaList, test.startSeq, test.targetSeq)
		a.Equal(test.expect, result)
		if test.err {
			a.Error(err)
		} else {
			a.NoError(err)
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

func TestBinlogMetaAreContinuous(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		metaList []binlogFileMeta
		expect   bool
	}{
		{
			metaList: []binlogFileMeta{},
			expect:   true},
		{
			metaList: []binlogFileMeta{{seq: 1}},
			expect:   true,
		},
		{
			metaList: []binlogFileMeta{{seq: 1}, {seq: 2}},
			expect:   true,
		},
		{
			metaList: []binlogFileMeta{{seq: 1}, {seq: 3}},
			expect:   false,
		},
	}
	for _, test := range tests {
		result := binlogMetaAreContinuous(test.metaList)
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
			line:    "#220620 13:23:55 server id 1  end_log_pos 126 CRC32 0x9a60fe57 	Start: binlog v 4, server v 8.0.33 created 220620 13:23:55 at startup",
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
			line:    "#220609 11:59:57 server id 1  end_log_pos 0 CRC32 0x031d41f6 	Start: binlog v 4, server v 8.0.33 created 220609 11:59:57",
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
			line:    "#220620 99:99:99 server id 1  end_log_pos 126 CRC32 0x9a60fe57 	Start: binlog v 4, server v 8.0.33 created 220620 13:23:55 at startup",
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
