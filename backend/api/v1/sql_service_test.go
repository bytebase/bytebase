package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

func TestEvalMaskingLevelOfDatabaseColumn(t *testing.T) {
	defaultDBSchemaMetadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "hiring",
				Tables: []*storepb.TableMetadata{
					{
						Name: "employees",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "id",
							},
							{
								Name:           "name",
								Classification: "1-1-1",
							},
							{
								Name: "remote",
							},
						},
					},
					{
						Name: "salary",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "employee_id",
							},
							{
								Name: "salary",
							},
						},
					},
				},
			},
			{
				Name: "company",
				Tables: []*storepb.TableMetadata{
					{
						Name: "office",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "office_id",
							},
							{
								Name: "city",
							},
						},
					},
				},
			},
		},
	}

	defaultProjectMessage := &store.ProjectMessage{
		DataClassificationConfigID: "2b599739-41da-4c35-a9ff-4a73c6cfe32c",
		ResourceID:                 "bytebase",
	}
	defaultClassificationSetting := &storepb.DataClassificationSetting{
		Configs: []*storepb.DataClassificationSetting_DataClassificationConfig{
			{
				Id: "2b599739-41da-4c35-a9ff-4a73c6cfe32c",
				Levels: []*storepb.DataClassificationSetting_DataClassificationConfig_Level{
					{
						Id: "S1",
					},
					{
						Id: "S2",
					},
				},
				Classification: map[string]*storepb.DataClassificationSetting_DataClassificationConfig_DataClassification{
					"1-1-1": {
						Id:    "1-1-1",
						Title: "personal",
						LevelId: func() *string {
							a := "S2"
							return &a
						}(),
					},
				},
			},
		},
	}

	defaultDatabaseMessage := &store.DatabaseMessage{
		EnvironmentID: "prod",
		ProjectID:     "bytebase",
		InstanceID:    "neon-host",
		DatabaseName:  "bb",
	}

	defaultEmail := "zp@bytebase.com"
	defaultCurrentPrincipal := &store.UserMessage{
		Email: defaultEmail,
	}

	testCases := []struct {
		name                   string
		database               *store.DatabaseMessage
		databaseSchemaMetadata *storepb.DatabaseSchemaMetadata
		maskingPolicy          *storepb.MaskingPolicy
		maskingRulePolicy      *storepb.MaskingRulePolicy
		maskingExceptionPolicy *storepb.MaskingExceptionPolicy
		currentPrincipal       *store.UserMessage
		requestTime            time.Time
		want                   db.DatabaseSchema
	}{
		{
			name:                   "Respect Masking Policy",
			database:               defaultDatabaseMessage,
			databaseSchemaMetadata: defaultDBSchemaMetadata,
			currentPrincipal:       defaultCurrentPrincipal,
			maskingPolicy: &storepb.MaskingPolicy{
				MaskData: []*storepb.MaskData{
					{
						Schema:       "hiring",
						Table:        "employees",
						Column:       "id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "hiring",
						Table:        "employees",
						Column:       "name",
						MaskingLevel: storepb.MaskingLevel_PARTIAL,
					},
					{
						Schema:       "hiring",
						Table:        "employees",
						Column:       "remote",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "hiring",
						Table:        "salary",
						Column:       "employee_id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "hiring",
						Table:        "salary",
						Column:       "salary",
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Schema:       "company",
						Table:        "office",
						Column:       "office_id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "company",
						Table:        "office",
						Column:       "city",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
				},
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						Condition: &expr.Expr{
							Expression: `(environment_id == "prod") && (table_name == "employees")`,
						},
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
				},
			},
			maskingExceptionPolicy: &storepb.MaskingExceptionPolicy{},
			want: db.DatabaseSchema{
				Name: "bb",
				SchemaList: []db.SchemaSchema{
					{
						Name: "hiring",
						TableList: []db.TableSchema{
							{
								Name: "employees",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									{
										Name:         "name",
										MaskingLevel: storepb.MaskingLevel_PARTIAL,
										Sensitive:    true,
									},
									{
										Name:         "remote",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
								},
							},
							{
								Name: "salary",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "employee_id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									{
										Name:         "salary",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
								},
							},
						},
					},
					{
						Name: "company",
						TableList: []db.TableSchema{
							{
								Name: "office",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "office_id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									{
										Name:         "city",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
								},
							},
						},
					},
				},
			},
			requestTime: time.Now(),
		},
		{
			name:                   "Fallback To Masking Rule",
			database:               defaultDatabaseMessage,
			databaseSchemaMetadata: defaultDBSchemaMetadata,
			currentPrincipal:       defaultCurrentPrincipal,
			maskingPolicy: &storepb.MaskingPolicy{
				MaskData: []*storepb.MaskData{
					{
						Schema:       "hiring",
						Table:        "employees",
						Column:       "id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "hiring",
						Table:        "salary",
						Column:       "employee_id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "company",
						Table:        "office",
						Column:       "office_id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
				},
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						Condition: &expr.Expr{
							Expression: `(environment_id == "prod") && (schema_name == "hiring") && ((table_name == "employees") || (table_name == "salary"))`,
						},
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Condition: &expr.Expr{
							Expression: `(environment_id == "prod") && (schema_name == "company") && (table_name == "office")`,
						},
						MaskingLevel: storepb.MaskingLevel_PARTIAL,
					},
				},
			},
			maskingExceptionPolicy: &storepb.MaskingExceptionPolicy{},
			want: db.DatabaseSchema{
				Name: "bb",
				SchemaList: []db.SchemaSchema{
					{
						Name: "hiring",
						TableList: []db.TableSchema{
							{
								Name: "employees",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									{
										Name:         "name",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
									{
										Name:         "remote",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
								},
							},
							{
								Name: "salary",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "employee_id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									{
										Name:         "salary",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
								},
							},
						},
					},
					{
						Name: "company",
						TableList: []db.TableSchema{
							{
								Name: "office",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "office_id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									{
										Name:         "city",
										MaskingLevel: storepb.MaskingLevel_PARTIAL,
										Sensitive:    true,
									},
								},
							},
						},
					},
				},
			},
			requestTime: time.Now(),
		},
		{
			name:                   "Find Lower Level In Masking Exception Policy",
			database:               defaultDatabaseMessage,
			databaseSchemaMetadata: defaultDBSchemaMetadata,
			currentPrincipal:       defaultCurrentPrincipal,
			maskingPolicy: &storepb.MaskingPolicy{
				MaskData: []*storepb.MaskData{
					{
						Schema:       "hiring",
						Table:        "employees",
						Column:       "id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "hiring",
						Table:        "employees",
						Column:       "name",
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Schema:       "hiring",
						Table:        "employees",
						Column:       "remote",
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Schema:       "hiring",
						Table:        "salary",
						Column:       "employee_id",
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Schema:       "hiring",
						Table:        "salary",
						Column:       "salary",
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Schema:       "company",
						Table:        "office",
						Column:       "office_id",
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Schema:       "company",
						Table:        "office",
						Column:       "city",
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
				},
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{},
			},
			maskingExceptionPolicy: &storepb.MaskingExceptionPolicy{
				MaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
					{
						Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
						Condition: &expr.Expr{
							Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "id")`,
						},
						Members:      []string{defaultEmail},
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
						Condition: &expr.Expr{
							Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "salary") && (resource.column_name == "salary")`,
						},
						Members:      []string{defaultEmail},
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
				},
			},
			want: db.DatabaseSchema{
				Name: "bb",
				SchemaList: []db.SchemaSchema{
					{
						Name: "hiring",
						TableList: []db.TableSchema{
							{
								Name: "employees",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									{
										Name:         "name",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
									{
										Name:         "remote",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
								},
							},
							{
								Name: "salary",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "employee_id",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
									{
										Name:         "salary",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
								},
							},
						},
					},
					{
						Name: "company",
						TableList: []db.TableSchema{
							{
								Name: "office",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "office_id",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
									{
										Name:         "city",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
								},
							},
						},
					},
				},
			},
			requestTime: time.Now(),
		},
		{
			name:                   "Mixed",
			database:               defaultDatabaseMessage,
			databaseSchemaMetadata: defaultDBSchemaMetadata,
			currentPrincipal:       defaultCurrentPrincipal,
			maskingPolicy: &storepb.MaskingPolicy{
				MaskData: []*storepb.MaskData{
					{
						Schema:       "hiring",
						Table:        "employees",
						Column:       "id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "hiring",
						Table:        "salary",
						Column:       "employee_id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
					{
						Schema:       "hiring",
						Table:        "salary",
						Column:       "salary",
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
					{
						Schema:       "company",
						Table:        "office",
						Column:       "office_id",
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
				},
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						Condition: &expr.Expr{
							Expression: `(column_classification_level == "S2")`,
						},
						MaskingLevel: storepb.MaskingLevel_PARTIAL,
					},
					{
						Condition: &expr.Expr{
							Expression: `(table_name == "employees")`,
						},
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
				},
			},
			maskingExceptionPolicy: &storepb.MaskingExceptionPolicy{
				MaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
					{
						Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
						Condition: &expr.Expr{
							Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "salary") && (resource.column_name == "salary")`,
						},
						Members:      []string{defaultEmail},
						MaskingLevel: storepb.MaskingLevel_NONE,
					},
				},
			},
			want: db.DatabaseSchema{
				Name: "bb",
				SchemaList: []db.SchemaSchema{
					{
						Name: "hiring",
						TableList: []db.TableSchema{
							{
								Name: "employees",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									// Inherited from masking rule.
									{
										Name:         "name",
										MaskingLevel: storepb.MaskingLevel_PARTIAL,
										Sensitive:    true,
									},
									// Inherited from masking rule.
									{
										Name:         "remote",
										MaskingLevel: storepb.MaskingLevel_FULL,
										Sensitive:    true,
									},
								},
							},
							{
								Name: "salary",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "employee_id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									// Hit Exception.
									{
										Name:         "salary",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
								},
							},
						},
					},
					{
						Name: "company",
						TableList: []db.TableSchema{
							{
								Name: "office",
								ColumnList: []db.ColumnInfo{
									{
										Name:         "office_id",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
									// Do not included in masking policy and masking rule, ignore the exception.
									{
										Name:         "city",
										MaskingLevel: storepb.MaskingLevel_NONE,
										Sensitive:    false,
									},
								},
							},
						},
					},
				},
			},
			requestTime: time.Now(),
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		result, err := evalMaskingLevelOfDatabaseColumn(defaultProjectMessage, tc.database, tc.databaseSchemaMetadata, tc.maskingPolicy, tc.maskingRulePolicy, tc.maskingExceptionPolicy, tc.currentPrincipal, tc.requestTime, defaultClassificationSetting)
		a.NoError(err, tc.name)
		a.Equal(tc.want, result, tc.name)
	}
}
