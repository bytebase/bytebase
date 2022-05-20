package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestColumnRequirement(t *testing.T) {
	tests := []test{
		{
			statement: "CREATE TABLE book(id int)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NoRequiredColumn,
					Title:   "Require columns",
					Content: "Table `book` requires columns: created_ts, creator_id, updated_ts, updater_id",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp)`,
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book RENAME COLUMN creator_id TO creator;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NoRequiredColumn,
					Title:   "Require columns",
					Content: "Table `book` requires columns: creator_id",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book RENAME COLUMN creator TO creator_id;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book CHANGE COLUMN creator_id creator int;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NoRequiredColumn,
					Title:   "Require columns",
					Content: "Table `book` requires columns: creator_id",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book CHANGE COLUMN creator creator_id int;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book DROP COLUMN creator_id;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NoRequiredColumn,
					Title:   "Require columns",
					Content: "Table `book` requires columns: creator_id",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp,
							content varchar(255));
						ALTER TABLE book DROP COLUMN content;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updated_ts timestamp);
						ALTER TABLE book ADD COLUMN content varchar(255);`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NoRequiredColumn,
					Title:   "Require columns",
					Content: "Table `book` requires columns: updater_id",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book ADD COLUMN creator_id int;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						CREATE TABLE student(
							id int,
							created_ts timestamp,
							updated_ts timestamp);`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NoRequiredColumn,
					Title:   "Require columns",
					Content: "Table `book` requires columns: creator_id",
				},
				{
					Status:  advisor.Warn,
					Code:    common.NoRequiredColumn,
					Title:   "Require columns",
					Content: "Table `student` requires columns: creator_id, updater_id",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							creator int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						DROP TABLE book;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}
	payload, err := json.Marshal(api.RequiredColumnRulePayload{
		ColumnList: []string{
			"id",
			"created_ts",
			"updated_ts",
			"creator_id",
			"updater_id",
		},
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &ColumnRequirementAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleRequiredColumn,
		Level:   api.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, &MockCatalogService{})
}
