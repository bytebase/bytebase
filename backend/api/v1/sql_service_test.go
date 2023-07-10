package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetSQLStatementPrefix(t *testing.T) {
	tests := []struct {
		engine       db.Type
		resourceList []parser.SchemaResource
		columnNames  []string
		want         string
	}{
		{
			engine:       db.MySQL,
			resourceList: nil,
			columnNames:  []string{"a"},
			want:         "INSERT INTO `<table_name>` (`a`) VALUES (",
		},
		{
			engine:       db.MySQL,
			resourceList: []parser.SchemaResource{{Database: "db1", Schema: "", Table: "table1"}},
			columnNames:  []string{"a", "b"},
			want:         "INSERT INTO `table1` (`a`,`b`) VALUES (",
		},
		{
			engine:       db.Postgres,
			resourceList: nil,
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"<table_name>\" (\"a\") VALUES (",
		},
		{
			engine:       db.Postgres,
			resourceList: []parser.SchemaResource{{Database: "db1", Schema: "", Table: "table1"}},
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"table1\" (\"a\") VALUES (",
		},
		{
			engine:       db.Postgres,
			resourceList: []parser.SchemaResource{{Database: "db1", Schema: "schema1", Table: "table1"}},
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"schema1\".\"table1\" (\"a\") VALUES (",
		},
	}
	a := assert.New(t)

	for _, test := range tests {
		got, err := getSQLStatementPrefix(test.engine, test.resourceList, test.columnNames)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}

func TestExportSQL(t *testing.T) {
	tests := []struct {
		engine          db.Type
		statementPrefix string
		result          *v1pb.QueryResult
		want            string
	}{
		{
			engine:          db.MySQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_BoolValue{BoolValue: true},
							},
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "abc"},
							},
							{
								Kind: &v1pb.RowValue_NullValue{},
							},
						},
					},
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_BoolValue{BoolValue: false},
							},
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "abc"},
							},
							{
								Kind: &v1pb.RowValue_NullValue{},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES (true,'abc',NULL);\nINSERT INTO `<table_name>` (`a`) VALUES (false,'abc',NULL);",
		},
		{
			engine:          db.MySQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\nbc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\\nbc');",
		},
		{
			engine:          db.MySQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a'b"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a''b');",
		},
		{
			engine:          db.MySQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\b"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\\b');",
		},
		{
			engine:          db.Postgres,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\nbc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\nbc');",
		},
		{
			engine:          db.Postgres,
			statementPrefix: "INSERT INTO `<table_name>` (`b`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\\bc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`b`) VALUES ( E'a\\\\bc');",
		},
	}
	a := assert.New(t)

	for _, test := range tests {
		got, err := exportSQL(test.engine, test.statementPrefix, test.result)
		a.NoError(err)
		a.Equal(test.want, string(got))
	}
}

func TestDecodeBase64String(t *testing.T) {
	tests := []struct {
		statement string
		want      string
	}{
		{
			statement: "",
			want:      "",
		},
		{
			statement: "U0VMRUNUIG5hbWUgQVMg5aeT5ZCNIEZST00gZW1wbG95ZWU7",
			want:      "SELECT name AS 姓名 FROM employee;",
		},
		{
			statement: "aGVsbG8g5ZOI5Za9IPCfkYs=",
			want:      "hello 哈喽 👋",
		},
	}

	for _, test := range tests {
		got, err := DecodeBase64String(test.statement)
		assert.NoError(t, err)
		assert.Equal(t, test.want, got)
	}
}
