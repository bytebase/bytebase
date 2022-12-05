package mysql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestColumnRequirement(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table `book` requires columns: created_ts, creator_id, updated_ts, updater_id",
					Line:    1,
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp)`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book RENAME COLUMN creator_id TO creator;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table `book` requires columns: creator_id",
					Line:    7,
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book RENAME COLUMN creator TO creator_id;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book CHANGE COLUMN creator_id creator int;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table `book` requires columns: creator_id",
					Line:    7,
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book CHANGE COLUMN creator creator_id int;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book DROP COLUMN creator_id;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table `book` requires columns: creator_id",
					Line:    7,
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp,
							content varchar(255));
						ALTER TABLE book DROP COLUMN content;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator_id int,
							created_ts timestamp,
							updated_ts timestamp);
						ALTER TABLE book ADD COLUMN content varchar(255);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table `book` requires columns: updater_id",
					Line:    5,
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						ALTER TABLE book ADD COLUMN creator_id int;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						CREATE TABLE student(
							id int,
							created_ts timestamp,
							updated_ts timestamp)
			
			
			`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table `book` requires columns: creator_id",
					Line:    5,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.NoRequiredColumn,
					Title:   "column.required",
					Content: "Table `student` requires columns: creator_id, updater_id",
					Line:    9,
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							creator int,
							created_ts timestamp,
							updater_id int,
							updated_ts timestamp);
						DROP TABLE book;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}
	payload, err := json.Marshal(advisor.StringArrayTypeRulePayload{
		List: []string{
			"id",
			"created_ts",
			"updated_ts",
			"creator_id",
			"updater_id",
		},
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &ColumnRequirementAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleRequiredColumn,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, advisor.MockMySQLDatabase)
}
