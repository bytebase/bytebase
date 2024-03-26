package mysql

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "AUTO_INCREMENT"},
			},
		},
		{
			name:         "name0",
			defaultStr:   sql.NullString{},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultNull{DefaultNull: true},
			},
		},
		{
			name:         "name1",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "name2",
			defaultStr:   sql.NullString{Valid: true, String: "hello"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "hello"}},
			},
		},
		{
			name:         "age0",
			defaultStr:   sql.NullString{},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: nil,
			},
		},
		{
			name:         "age1",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "age2",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "age3",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "age4",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "price",
			defaultStr:   sql.NullString{Valid: true, String: "0.00"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0.00"}},
			},
		},
		{
			name:         "time0",
			defaultStr:   sql.NullString{},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultNull{DefaultNull: true},
			},
		},
		{
			name:         "time1",
			defaultStr:   sql.NullString{},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: nil,
			},
		},
		{
			name:         "time2",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: false,
			extra:        "DEFAULT_GENERATED",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP"},
			},
		},
		{
			name:         "time3",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: false,
			extra:        "DEFAULT_GENERATED on update CURRENT_TIMESTAMP",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP"},
				OnUpdate:     "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time4",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED on update CURRENT_TIMESTAMP",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP"},
				OnUpdate:     "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time5",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP(6)"},
			},
		},
		{
			name:         "time6",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP(6)"},
			},
		},
		{
			name:         "time7",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			extra:        "DEFAULT_GENERATED",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP"},
			},
		},
		// MySQL 5.7.
		{
			name:         "id",
			defaultStr:   sql.NullString{},
			nullableBool: false,
			extra:        "auto_increment",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "AUTO_INCREMENT"},
			},
		},
		{
			name:         "name0",
			defaultStr:   sql.NullString{},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultNull{DefaultNull: true},
			},
		},
		{
			name:         "name1",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "name2",
			defaultStr:   sql.NullString{Valid: true, String: "hello"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "hello"}},
			},
		},
		{
			name:         "age0",
			defaultStr:   sql.NullString{},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: nil,
			},
		},
		{
			name:         "age1",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "age2",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "age3",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "age4",
			defaultStr:   sql.NullString{Valid: true, String: "0"},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0"}},
			},
		},
		{
			name:         "price",
			defaultStr:   sql.NullString{Valid: true, String: "0.00"},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: "0.00"}},
			},
		},
		{
			name:         "time0",
			defaultStr:   sql.NullString{},
			nullableBool: true,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultNull{DefaultNull: true},
			},
		},
		{
			name:         "time1",
			defaultStr:   sql.NullString{},
			nullableBool: false,
			extra:        "",
			want: &storepb.ColumnMetadata{
				DefaultValue: nil,
			},
		},
		{
			name:         "time2",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: false,
			// Different from 8.0, DEFAULT_GENERATED.
			extra: "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP"},
			},
		},
		{
			name:         "time3",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: false,
			// Different from 8.0, DEFAULT_GENERATED on update CURRENT_TIMESTAMP.
			extra: "on update CURRENT_TIMESTAMP",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP"},
				OnUpdate:     "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time4",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			// Different from 8.0, DEFAULT_GENERATED on update CURRENT_TIMESTAMP.
			extra: "on update CURRENT_TIMESTAMP",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP"},
				OnUpdate:     "CURRENT_TIMESTAMP",
			},
		},
		{
			name:         "time5",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			// Different from 8.0, DEFAULT_GENERATED.
			extra: "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP(6)"},
			},
		},
		{
			name:         "time6",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP(6)"},
			nullableBool: true,
			// Different from 8.0, DEFAULT_GENERATED.
			extra: "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP(6)"},
			},
		},
		{
			name:         "time7",
			defaultStr:   sql.NullString{Valid: true, String: "CURRENT_TIMESTAMP"},
			nullableBool: true,
			// Different from 8.0, DEFAULT_GENERATED.
			extra: "",
			want: &storepb.ColumnMetadata{
				DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "CURRENT_TIMESTAMP"},
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
