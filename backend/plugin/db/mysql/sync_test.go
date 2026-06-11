package mysql

import (
	"database/sql"
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
	}

	a := require.New(t)
	for _, tc := range tests {
		column := &storepb.ColumnMetadata{}
		setColumnMetadataDefault(column, tc.defaultStr, tc.nullableBool, tc.extra)
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
	}

	for _, tc := range testCases {
		got, err := getViewDefFromCreateView(tc.stmt)
		require.NoError(t, err)
		require.Equal(t, tc.def, got)
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
