package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

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
