package mysql

import (
	"database/sql"
	"strings"
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

		// MySQL 8.0 created format.
		CREATE TABLE `hello` (
			`id` int NOT NULL AUTO_INCREMENT,
			`name0` varchar(20) DEFAULT NULL,
			`name1` varchar(20) DEFAULT '0',
			`name2` varchar(20) DEFAULT 'hello',
			`age0` int NOT NULL,
			`age1` tinyint NOT NULL DEFAULT '0',
			`age2` tinyint NOT NULL DEFAULT '0',
			`age3` tinyint NOT NULL DEFAULT '0',
			`age4` tinyint NOT NULL DEFAULT '0',
			`price` double(16,2) DEFAULT '0.00',
			`time0` datetime DEFAULT NULL,
			`time1` datetime NOT NULL,
			`time2` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			`time3` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			`time4` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			`time5` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
			`time6` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
			`time7` datetime DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (`id`)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

		// MySQL 5.7 created format.
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
			`price` double(16,2) DEFAULT '0.00',
			`time0` datetime DEFAULT NULL,
			`time1` datetime NOT NULL,
			`time2` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			`time3` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			`time4` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			`time5` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
			`time6` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
			`time7` datetime DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (`id`)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	*/
	tests := []struct {
		name         string
		defaultStr   sql.NullString
		nullableBool bool
		extra        string
		// columnType is information_schema.COLUMNS.COLUMN_TYPE; only binary-family
		// types change the decoding, so most rows leave it empty.
		columnType string
		// binaryFormat is how the server encodes binary-family defaults; zero value
		// (binaryDefaultVerbatim) matches the legacy MariaDB/OceanBase path.
		binaryFormat binaryDefaultFormat
		want         *storepb.ColumnMetadata
	}{
		// MySQL 8.0.
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
			name:         "price",
			defaultStr:   sql.NullString{Valid: true, String: "0.00"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "0.00",
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
			extra:        "DEFAULT_GENERATED",
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
			extra:        "DEFAULT_GENERATED",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP(6)",
			},
		},
		{
			name:         "time6",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP(6)",
			},
		},
		{
			name:         "time7",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP",
			},
		},
		// MySQL 5.7.
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
			name:         "price",
			defaultStr:   sql.NullString{Valid: true, String: "0.00"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				Default: "0.00",
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
			// Different from 8.0, DEFAULT_GENERATED.
			extra: "",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time3",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: false,
			// Different from 8.0, DEFAULT_GENERATED on update CURRENT_TIMESTAMP.
			extra: "on update CURRENT_TIMESTAMP",
			want: &storepb.ColumnMetadata{
				Default:  "CURRENT_TIMESTAMP",
				OnUpdate: "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time4",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			// Different from 8.0, DEFAULT_GENERATED on update CURRENT_TIMESTAMP.
			extra: "on update CURRENT_TIMESTAMP",
			want: &storepb.ColumnMetadata{
				Default:  "CURRENT_TIMESTAMP",
				OnUpdate: "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time5",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			// Different from 8.0, DEFAULT_GENERATED.
			extra: "",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP(6)",
			},
		},
		{
			name:         "time6",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			// Different from 8.0, DEFAULT_GENERATED.
			extra: "",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP(6)",
			},
		},
		{
			name:         "time7",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			// Different from 8.0, DEFAULT_GENERATED.
			extra: "",
			want: &storepb.ColumnMetadata{
				Default: "CURRENT_TIMESTAMP",
			},
		},

		// Binary-family literal defaults. information_schema encodes them per version
		// (verified live on 8.0.32 and 5.7.25): 8.0 reports hex NOTATION text built from
		// the value truncated at its first NUL ("0x", "0x6162"); 5.7 reports the raw
		// bytes, NUL-padded to the declared length for binary(N) and truncated at the
		// first non-convertible byte. Both decode to one canonical form: '' when empty,
		// a plain quoted string for clean text, a hex literal otherwise. defaultStr is
		// what the sync query delivers: QUOTE(COLUMN_DEFAULT).

		// MySQL 8.0 hex notation.
		{
			// binary(16) DEFAULT '' — the openemr uuid shape; I_S reports "0x".
			name:         "bin16_empty_80",
			defaultStr:   sql.NullString{Valid: true, String: `'0x'`},
			nullableBool: false,
			columnType:   "binary(16)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "binary(16)",
				Default: "''",
			},
		},
		{
			// binary(16) DEFAULT 'ab' — padding NULs are already truncated by 8.0.
			name:         "bin16_ab_80",
			defaultStr:   sql.NullString{Valid: true, String: `'0x6162'`},
			nullableBool: false,
			columnType:   "binary(16)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "binary(16)",
				Default: "'ab'",
			},
		},
		{
			// varbinary(24) DEFAULT 'ab'.
			name:         "vb_ab_80",
			defaultStr:   sql.NullString{Valid: true, String: `'0x6162'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "'ab'",
			},
		},
		{
			// varbinary(24) DEFAULT '' — 8.0 reports an empty string, not "0x".
			name:         "vb_empty_80",
			defaultStr:   sql.NullString{Valid: true, String: `''`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "''",
			},
		},
		{
			// varbinary DEFAULT 0xAB — not valid UTF-8, stays a hex literal (lowercase).
			name:         "vb_nonutf8_80",
			defaultStr:   sql.NullString{Valid: true, String: `'0xAB'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "0xab",
			},
		},
		{
			// varbinary DEFAULT 'a''b' — quote byte round-trips with QUOTE()-style escaping.
			name:         "vb_quote_80",
			defaultStr:   sql.NullString{Valid: true, String: `'0x612762'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: `'a\'b'`,
			},
		},
		{
			// varbinary DEFAULT '0x6162' — the literal STRING; 8.0 reports the notation
			// of the text ("0x307836313632") and decoding restores the text.
			name:         "vb_hexstring_80",
			defaultStr:   sql.NullString{Valid: true, String: `'0x307836313632'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "'0x6162'",
			},
		},
		{
			// varchar DEFAULT '0x6162' — NOT binary-family: the value is the plain
			// string and must stay verbatim (the contrast case for the type gate).
			name:         "varchar_hexstring_80",
			defaultStr:   sql.NullString{Valid: true, String: `'0x6162'`},
			nullableBool: false,
			columnType:   "varchar(24)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "varchar(24)",
				Default: "'0x6162'",
			},
		},
		{
			// varbinary DEFAULT (0x6162) — expression default (DEFAULT_GENERATED) keeps
			// the expression path even on a binary column.
			name:         "vb_expression_80",
			defaultStr:   sql.NullString{Valid: true, String: `'0x6162'`},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED",
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "(0x6162)",
			},
		},
		{
			// binary(16) DEFAULT (uuid_to_bin(uuid())) — expression default.
			name:         "bin16_expression_80",
			defaultStr:   sql.NullString{Valid: true, String: `'uuid_to_bin(uuid())'`},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED",
			columnType:   "binary(16)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "binary(16)",
				Default: "(uuid_to_bin(uuid()))",
			},
		},
		{
			// Not hex notation on a >= 8.0 server — unexpected shape falls back verbatim.
			name:         "vb_unexpected_80",
			defaultStr:   sql.NullString{Valid: true, String: `'zz'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultHexNotation,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "'zz'",
			},
		},

		// MySQL 5.7 raw bytes.
		{
			// binary(16) DEFAULT '' — I_S reports sixteen NUL padding bytes.
			name:         "bin16_empty_57",
			defaultStr:   sql.NullString{Valid: true, String: `'\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0'`},
			nullableBool: false,
			columnType:   "binary(16)",
			binaryFormat: binaryDefaultRawBytes,
			want: &storepb.ColumnMetadata{
				Type:    "binary(16)",
				Default: "''",
			},
		},
		{
			// binary(16) DEFAULT 'ab' — value plus fourteen NUL padding bytes.
			name:         "bin16_ab_57",
			defaultStr:   sql.NullString{Valid: true, String: `'ab\0\0\0\0\0\0\0\0\0\0\0\0\0\0'`},
			nullableBool: false,
			columnType:   "binary(16)",
			binaryFormat: binaryDefaultRawBytes,
			want: &storepb.ColumnMetadata{
				Type:    "binary(16)",
				Default: "'ab'",
			},
		},
		{
			// varbinary(24) DEFAULT 'ab'.
			name:         "vb_ab_57",
			defaultStr:   sql.NullString{Valid: true, String: `'ab'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultRawBytes,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "'ab'",
			},
		},
		{
			// varbinary(24) DEFAULT ''.
			name:         "vb_empty_57",
			defaultStr:   sql.NullString{Valid: true, String: `''`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultRawBytes,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "''",
			},
		},
		{
			// varbinary DEFAULT 0x6100 — trailing NUL is significant on varbinary
			// (no padding strip) and forces the hex-literal form.
			name:         "vb_trailnul_57",
			defaultStr:   sql.NullString{Valid: true, String: `'a\0'`},
			nullableBool: false,
			columnType:   "varbinary(8)",
			binaryFormat: binaryDefaultRawBytes,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(8)",
				Default: "0x6100",
			},
		},
		{
			// varbinary DEFAULT 'a''b' — QUOTE() escapes the quote byte.
			name:         "vb_quote_57",
			defaultStr:   sql.NullString{Valid: true, String: `'a\'b'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultRawBytes,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: `'a\'b'`,
			},
		},
		{
			// varbinary DEFAULT '0x6162' — the literal STRING; 5.7 reports the raw text,
			// which is clean and matches the 8.0 decoding of "0x307836313632".
			name:         "vb_hexstring_57",
			defaultStr:   sql.NullString{Valid: true, String: `'0x6162'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultRawBytes,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "'0x6162'",
			},
		},
		{
			// varbinary DEFAULT 'CURRENT_TIMESTAMP' — raw bytes that merely spell the
			// function stay a literal (binary types cannot default to CURRENT_TIMESTAMP).
			name:         "vb_current_timestamp_text_57",
			defaultStr:   sql.NullString{Valid: true, String: `'CURRENT_TIMESTAMP'`},
			nullableBool: false,
			columnType:   "varbinary(20)",
			binaryFormat: binaryDefaultRawBytes,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(20)",
				Default: "'CURRENT_TIMESTAMP'",
			},
		},

		// MariaDB/OceanBase (verbatim) — binary-family decoding is gated off entirely.
		{
			name:         "vb_verbatim_mariadb",
			defaultStr:   sql.NullString{Valid: true, String: `'0x6162'`},
			nullableBool: false,
			columnType:   "varbinary(24)",
			binaryFormat: binaryDefaultVerbatim,
			want: &storepb.ColumnMetadata{
				Type:    "varbinary(24)",
				Default: "'0x6162'",
			},
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		column := &storepb.ColumnMetadata{Type: tc.columnType}
		setColumnMetadataDefault(column, tc.defaultStr, tc.nullableBool, tc.extra, tc.binaryFormat)
		a.Equal(tc.want, column, tc.name)
	}
}

func TestGetViewDefFromCreateView(t *testing.T) {
	testCases := []struct {
		stmt string
		def  string
	}{
		{
			stmt: "CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`localhost` SQL SECURITY INVOKER VIEW `actor_info` AS select `a`.`actor_id` AS `actor_id`,`a`.`first_name` AS `first_name`,`a`.`last_name` AS `last_name`,group_concat(distinct concat(`c`.`name`,': ',(select group_concat(`f`.`title` order by `f`.`title` ASC separator ', ') from ((`film` `f` join `film_category` `fc` on((`f`.`film_id` = `fc`.`film_id`))) join `film_actor` `fa` on((`f`.`film_id` = `fa`.`film_id`))) where ((`fc`.`category_id` = `c`.`category_id`) and (`fa`.`actor_id` = `a`.`actor_id`)))) order by `c`.`name` ASC separator '; ') AS `film_info` from (((`actor` `a` left join `film_actor` `fa` on((`a`.`actor_id` = `fa`.`actor_id`))) left join `film_category` `fc` on((`fa`.`film_id` = `fc`.`film_id`))) left join `category` `c` on((`fc`.`category_id` = `c`.`category_id`))) group by `a`.`actor_id`,`a`.`first_name`,`a`.`last_name`",
			def:  "select `a`.`actor_id` AS `actor_id`,`a`.`first_name` AS `first_name`,`a`.`last_name` AS `last_name`,group_concat(distinct concat(`c`.`name`,': ',(select group_concat(`f`.`title` order by `f`.`title` ASC separator ', ') from ((`film` `f` join `film_category` `fc` on((`f`.`film_id` = `fc`.`film_id`))) join `film_actor` `fa` on((`f`.`film_id` = `fa`.`film_id`))) where ((`fc`.`category_id` = `c`.`category_id`) and (`fa`.`actor_id` = `a`.`actor_id`)))) order by `c`.`name` ASC separator '; ') AS `film_info` from (((`actor` `a` left join `film_actor` `fa` on((`a`.`actor_id` = `fa`.`actor_id`))) left join `film_category` `fc` on((`fa`.`film_id` = `fc`.`film_id`))) left join `category` `c` on((`fc`.`category_id` = `c`.`category_id`))) group by `a`.`actor_id`,`a`.`first_name`,`a`.`last_name`",
		},
		{
			stmt: "CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`localhost` SQL SECURITY INVOKER VIEW `actor_info` (`id`) AS select idid from t",
			def:  "select idid from t",
		},
		{
			// Comma-separated multi-column list — the stock sys schema form
			// (e.g. x$user_summary_by_file_io) that previously failed to match.
			stmt: "CREATE ALGORITHM=TEMPTABLE DEFINER=`mysql.sys`@`localhost` SQL SECURITY INVOKER VIEW `x$user_summary_by_file_io` (`user`,`ios`,`io_latency`) AS select 1 AS `user`,2 AS `ios`,3 AS `io_latency`",
			def:  "select 1 AS `user`,2 AS `ios`,3 AS `io_latency`",
		},
		{
			// Schema-qualified view name, printed when the session database differs.
			stmt: "CREATE ALGORITHM=TEMPTABLE DEFINER=`mysql.sys`@`localhost` SQL SECURITY INVOKER VIEW `sys`.`x$user_summary_by_file_io` (`user`,`ios`,`io_latency`) AS select 1 AS `user`",
			def:  "select 1 AS `user`",
		},
		{
			// Column identifier with an embedded backtick: SHOW CREATE doubles it
			// (`a``b`). The doubling-aware list segments must match it (8.0-only
			// surface: 5.7 rewrites explicit column lists into per-column aliases).
			stmt: "CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`localhost` SQL SECURITY INVOKER VIEW `v` (`a``b`,`c`) AS select 1 AS `x`, 2 AS `y`",
			def:  "select 1 AS `x`, 2 AS `y`",
		},
		{
			// Column identifier containing ", " must not mis-split the list.
			stmt: "CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`localhost` SQL SECURITY INVOKER VIEW `v` (`a, b`,`c`) AS select 1 AS `x`, 2 AS `y`",
			def:  "select 1 AS `x`, 2 AS `y`",
		},
	}

	for _, tc := range testCases {
		got, err := getViewDefFromCreateView(tc.stmt)
		require.NoError(t, err)
		require.Equal(t, tc.def, got)
	}
}

// TestIsStockMySQL pins the single stock-MySQL predicate gating stock-only
// information_schema surfaces (the SRS_ID column select, the binary-default decode,
// and the column INVISIBLE capture): MariaDB, OceanBase, and TiDB report MySQL-like
// versions but lack them, so a compatible engine registered as MYSQL must not trip
// the stock-only queries.
func TestIsStockMySQL(t *testing.T) {
	testCases := []struct {
		rest string
		want bool
	}{
		{rest: "", want: true},
		{rest: "-log", want: true},
		{rest: "-0ubuntu0.20.04.1", want: true},
		{rest: "-MariaDB-1:10.6.12+maria~ubu2004", want: false},
		{rest: "-OceanBase-v4.2.1", want: false},
		{rest: "-TiDB-v7.5.0", want: false},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.want, isStockMySQL(tc.rest), "rest=%q", tc.rest)
	}
}

// TestInvisibleColumnCaptureGate pins the SyncDBSchema column-scan condition
// (stockMySQL && isInvisibleColumnExtra): MariaDB 10.3+ reports the same INVISIBLE
// token in information_schema.COLUMNS.EXTRA, but the writers emit only the
// MySQL-versioned /*!80023 INVISIBLE */ form, which MariaDB ignores — capturing the
// attribute there would produce a permanently non-converging declarative diff, so it
// is captured on stock MySQL only.
func TestInvisibleColumnCaptureGate(t *testing.T) {
	testCases := []struct {
		rest  string
		extra string
		want  bool
	}{
		// Stock MySQL 8.0.23+ captures the attribute.
		{rest: "", extra: "INVISIBLE", want: true},
		{rest: "-log", extra: "DEFAULT_GENERATED on update CURRENT_TIMESTAMP INVISIBLE", want: true},
		{rest: "", extra: "VIRTUAL GENERATED INVISIBLE", want: true},
		{rest: "", extra: "", want: false},
		{rest: "", extra: "auto_increment", want: false},
		// MariaDB reports INVISIBLE in EXTRA too, but must not capture it.
		{rest: "-MariaDB-1:10.6.12+maria~ubu2004", extra: "INVISIBLE", want: false},
		// OceanBase/TiDB report MySQL-like versions; never capture there either.
		{rest: "-OceanBase-v4.2.1", extra: "INVISIBLE", want: false},
		{rest: "-TiDB-v7.5.0", extra: "INVISIBLE", want: false},
	}
	for _, tc := range testCases {
		got := isStockMySQL(tc.rest) && isInvisibleColumnExtra(tc.extra)
		require.Equal(t, tc.want, got, "rest=%q extra=%q", tc.rest, tc.extra)
	}
}

func TestUnescapeCheckClause(t *testing.T) {
	testCases := []struct {
		clause string
		want   string
	}{
		{
			// information_schema.CHECK_CONSTRAINTS escapes single quotes as \'.
			clause: `((` + "`type`" + ` = _utf8mb4\'A\') and (` + "`amount`" + ` is not null))`,
			want:   "((`type` = _utf8mb4'A') and (`amount` is not null))",
		},
		{
			clause: "(`amount` > 0)",
			want:   "(`amount` > 0)",
		},
		{
			clause: `(` + "`status`" + ` in (_utf8mb4\'open\',_utf8mb4\'closed\'))`,
			want:   "(`status` in (_utf8mb4'open',_utf8mb4'closed'))",
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.want, unescapeCheckClause(tc.clause))
	}
}

func TestGetCheckConstraintQuery(t *testing.T) {
	t.Run("MariaDB uses native table name column", func(t *testing.T) {
		query := strings.ToUpper(getCheckConstraintQuery(true))

		require.Contains(t, query, "TABLE_NAME")
		require.Contains(t, query, "FROM INFORMATION_SCHEMA.CHECK_CONSTRAINTS")
		require.Contains(t, query, "WHERE CONSTRAINT_SCHEMA = ?")
		require.NotContains(t, query, "JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS")
		require.NotContains(t, query, "TC.TABLE_SCHEMA")
	})

	t.Run("MySQL keeps table constraints join", func(t *testing.T) {
		query := strings.ToUpper(getCheckConstraintQuery(false))

		require.Contains(t, query, "TC.TABLE_NAME")
		require.Contains(t, query, "JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS")
		require.Contains(t, query, "TC.CONSTRAINT_TYPE = 'CHECK'")
		require.Contains(t, query, "TC.TABLE_SCHEMA = ?")
	})
}

func TestStripReturnsCharset(t *testing.T) {
	testCases := []struct {
		name string
		def  string
		want string
	}{
		{
			name: "multiline parameter list with RETURNS CHARSET and COLLATE",
			def: "CREATE FUNCTION `f1`(\n" +
				"    p_a BIGINT,\n" +
				"    p_b VARCHAR(36)\n" +
				") RETURNS char(36) CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci\n" +
				"    NO SQL\n" +
				"    DETERMINISTIC\n" +
				"BEGIN\n" +
				"    RETURN p_b;\n" +
				"END",
			want: "CREATE FUNCTION `f1`(\n" +
				"    p_a BIGINT,\n" +
				"    p_b VARCHAR(36)\n" +
				") RETURNS char(36)\n" +
				"    NO SQL\n" +
				"    DETERMINISTIC\n" +
				"BEGIN\n" +
				"    RETURN p_b;\n" +
				"END",
		},
		{
			name: "single-line parameter list with RETURNS CHARSET",
			def: "CREATE FUNCTION `f2`(i INT) RETURNS char(36) CHARSET utf8mb4\n" +
				"BEGIN\n" +
				"    RETURN 'x';\n" +
				"END",
			want: "CREATE FUNCTION `f2`(i INT) RETURNS char(36)\n" +
				"BEGIN\n" +
				"    RETURN 'x';\n" +
				"END",
		},
		{
			name: "RETURNS without CHARSET is unchanged",
			def: "CREATE FUNCTION `f3`(i INT) RETURNS int\n" +
				"RETURN i + 1",
			want: "CREATE FUNCTION `f3`(i INT) RETURNS int\n" +
				"RETURN i + 1",
		},
		{
			name: "CHARSET in parameter list only is unchanged",
			def: "CREATE FUNCTION `f4`(\n" +
				"    p_a VARCHAR(10) CHARSET utf8mb4,\n" +
				"    p_b INT\n" +
				") RETURNS int\n" +
				"RETURN p_b",
			want: "CREATE FUNCTION `f4`(\n" +
				"    p_a VARCHAR(10) CHARSET utf8mb4,\n" +
				"    p_b INT\n" +
				") RETURNS int\n" +
				"RETURN p_b",
		},
		{
			name: "no RETURNS clause is unchanged",
			def:  "CREATE PROCEDURE `p1`(IN a INT)\nBEGIN\nEND",
			want: "CREATE PROCEDURE `p1`(IN a INT)\nBEGIN\nEND",
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.want, stripReturnsCharset(tc.def), tc.name)
	}
}
