package tidb

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestSetColumnMetadataDefault(t *testing.T) {
	/*
		CREATE TABLE hello (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name0 varchar(20),
			name1 varchar(20) DEFAULT '0',
			name2 varchar(20) DEFAULT 'hello',
			age0 int NOT NULL,
			age1 tinyint(4) NOT NULL DEFAULT '0',
			age2 tinyint(4) NOT NULL DEFAULT 0,
			age3 tinyint NOT NULL DEFAULT '0',
			age4 tinyint NOT NULL DEFAULT 0,
			price double(16,2) DEFAULT '0.00',
			time0 datetime,
			time1 datetime NOT NULL,
			time2 datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			time3 datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			time4 datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			time5 datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
			time6 datetime(6) DEFAULT NOW(6),
			time7 datetime DEFAULT CURRENT_TIMESTAMP()
		);

		// 8.0.11-TiDB-v7.6.0 created format.
		// 5.7.28-TiDB-v7.1.1-serverless created format.
		// 5.7.25-TiDB-v6.5.8 created format.
		CREATE TABLE `hello` (
			`id` int(11) NOT NULL AUTO_INCREMENT,
			`name0` varchar(20) DEFAULT NULL,
			`name1` varchar(20) DEFAULT '0',
			`name2` varchar(20) DEFAULT 'hello',
			`age0` int(11) NOT NULL,
			`age1` tinyint(4) NOT NULL DEFAULT '0',
			`age2` tinyint(4) NOT NULL DEFAULT '0',
			`age3` tinyint(4) NOT NULL DEFAULT '0',
			`age4` tinyint(4) NOT NULL DEFAULT '0',
			`price` double(16,2) DEFAULT '0',
			`time0` datetime DEFAULT NULL,
			`time1` datetime NOT NULL,
			`time2` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			`time3` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			`time4` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			`time5` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
			`time6` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
			`time7` datetime DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (`id`) ... // T![clustered_index] CLUSTERED
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
	*/
	tests := []struct {
		name         string
		defaultStr   sql.NullString
		nullableBool bool
		extra        string
		want         *storepb.ColumnMetadata
	}{
		// 8.0.11-TiDB-v7.6.0.
		// 5.7.28-TiDB-v7.1.1-serverless.
		// 5.7.25-TiDB-v6.5.8.
		{
			name:         "id",
			defaultStr:   sql.NullString{},
			nullableBool: false,
			extra:        "auto_increment",
			want: &storepb.ColumnMetadata{
				Default: "AUTO_INCREMENT",
			},
		},
		{
			name:         "name0",
			defaultStr:   sql.NullString{},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "NULL",
			},
		},
		{
			name:         "name1",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "0",
			},
		},
		{
			name:         "name2",
			defaultStr:   sql.NullString{Valid: true, String: "hello"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "hello",
			},
		},
		{
			name:         "age0",
			defaultStr:   sql.NullString{},
			nullableBool: false,
			extra:        "",
			want:         &storepb.ColumnMetadata{},
		},
		{
			name:         "age1",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "0",
			},
		},
		{
			name:         "age2",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "0",
			},
		},
		{
			name:         "age3",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "0",
			},
		},
		{
			name:         "age4",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "0",
			},
		},
		{
			name: "price",
			// This is strange, not "0.00".
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "0",
			},
		},
		{
			name:         "time0",
			defaultStr:   sql.NullString{},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "NULL",
			},
		},
		{
			name:         "time1",
			defaultStr:   sql.NullString{},
			nullableBool: false,
			extra:        "",
			want:         &storepb.ColumnMetadata{},
		},
		{
			name:         "time2",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time3",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: false,
			extra:        "DEFAULT_GENERATED on update CURRENT_TIMESTAMP",
			want: &storepb.ColumnMetadata{
				Default:  "CURRENT_TIMESTAMP",
				OnUpdate: "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time4",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED on update CURRENT_TIMESTAMP",
			want: &storepb.ColumnMetadata{
				Default:  "CURRENT_TIMESTAMP",
				OnUpdate: "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time5",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP(6)",
			},
		},
		{
			name:         "time6",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP(6)",
			},
		},
		{
			name:         "time7",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP",
			},
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		column := &storepb.ColumnMetadata{}
		setColumnMetadataDefault(column, tc.defaultStr, tc.nullableBool, tc.extra)
		a.Equal(tc.want, column, tc.name)
	}
}

func TestConvertCollationToCharset(t *testing.T) {
	tests := []struct {
		collation string
		want      string
	}{
		{
			collation: "ascii_bin",
			want:      "ascii",
		},
		{
			collation: "binary",
			want:      "binary",
		},
		{
			collation: "gbk_chinese_ci",
			want:      "gbk",
		},
		{
			collation: "latin1_bin",
			want:      "latin1",
		},
		{
			collation: "utf8_bin",
			want:      "utf8",
		},
		{
			collation: "utf8mb4_bin",
			want:      "utf8mb4",
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		got := convertCollationToCharset(tc.collation)
		a.Equal(tc.want, got, tc.collation)
	}
}

func TestStripSingleQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"'quoted'", "quoted"},
		{"''", ""},
		{"'single'", "single"},
		{"no quotes", "no quotes"},
		{"'partial", "'partial"},
		{"partial'", "partial'"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("input: %s", test.input), func(t *testing.T) {
			result := stripSingleQuote(test.input)
			require.Equal(t, test.expected, result)
		})
	}
}
