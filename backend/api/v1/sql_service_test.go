package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetSQLStatementPrefix(t *testing.T) {
	tests := []struct {
		engine       storepb.Engine
		resourceList []base.SchemaResource
		columnNames  []string
		want         string
	}{
		{
			engine:       storepb.Engine_MYSQL,
			resourceList: nil,
			columnNames:  []string{"a"},
			want:         "INSERT INTO `<table_name>` (`a`) VALUES (",
		},
		{
			engine:       storepb.Engine_MYSQL,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "", Table: "table1"}},
			columnNames:  []string{"a", "b"},
			want:         "INSERT INTO `table1` (`a`,`b`) VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: nil,
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"<table_name>\" (\"a\") VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "", Table: "table1"}},
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"table1\" (\"a\") VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "schema1", Table: "table1"}},
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
		engine          storepb.Engine
		statementPrefix string
		result          *v1pb.QueryResult
		want            string
	}{
		{
			engine:          storepb.Engine_MYSQL,
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
			engine:          storepb.Engine_MYSQL,
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
			engine:          storepb.Engine_MYSQL,
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
			engine:          storepb.Engine_MYSQL,
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
			engine:          storepb.Engine_POSTGRES,
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
			engine:          storepb.Engine_POSTGRES,
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

func TestEncodeToBase64String(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "",
			want:  "",
		},
		{
			input: "select * from employee",
			want:  "c2VsZWN0ICogZnJvbSBlbXBsb3llZQ==",
		},
		{
			input: "select name as 姓名 from employee",
			want:  "c2VsZWN0IG5hbWUgYXMg5aeT5ZCNIGZyb20gZW1wbG95ZWU=",
		},
		{
			input: "Hello 哈喽 👋",
			want:  "SGVsbG8g5ZOI5Za9IPCfkYs=",
		},
	}

	for _, test := range tests {
		got := encodeToBase64String(test.input)
		if got != test.want {
			t.Errorf("encodeToBase64String(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestGetExcelColumnName(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		index int
		want  string
	}{
		{
			index: 0,
			want:  "A",
		},
		{
			index: 3,
			want:  "D",
		},
		{
			index: 25,
			want:  "Z",
		},
		{
			index: 26,
			want:  "AA",
		},
		{
			index: 27,
			want:  "AB",
		},
		{
			index: excelMaxColumn - 1,
			want:  "ZZZ",
		},
	}

	for _, test := range tests {
		got, err := getExcelColumnName(test.index)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}

func TestGetOffsetAndOriginTable(t *testing.T) {
	tests := []struct {
		input     string
		offset    int
		tableName string
	}{
		{
			input:     "_20240614164851_0_TEST_table1",
			offset:    0,
			tableName: "TEST_table1",
		},
		{
			input:     "_20240614164851_11_TEST",
			offset:    11,
			tableName: "TEST",
		},
		{
			input:     "_20240614164851_8_TEST_input_table1",
			offset:    8,
			tableName: "TEST_input_table1",
		},
	}
	a := assert.New(t)

	for _, test := range tests {
		offset, tabaleName, err := getOffsetAndOriginTable(test.input)
		a.NoError(err)
		a.Equal(test.offset, offset)
		a.Equal(test.tableName, tabaleName)
	}
}
