package mysql

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestNamingColumnConvention(t *testing.T) {
	invalidColumnName := advisor.RandomString(65)

	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int, creatorId int)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    1,
				},
			},
		},
		{
			Statement: fmt.Sprintf("CREATE TABLE book(id int, %s int)", invalidColumnName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: fmt.Sprintf("`book`.`%s` mismatches column naming convention, its length should be within 64 characters", invalidColumnName),
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int, creator_id int)",
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
			Statement: `CREATE TABLE book(id int, creator_id int);
						ALTER TABLE book RENAME COLUMN creator_id TO creatorId`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    2,
				},
			},
		},
		{
			Statement: `ALTER TABLE tech_book RENAME COLUMN id TO creator_id;`,
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
						ALTER TABLE book CHANGE COLUMN creator_id creatorId int;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    7,
				},
			},
		},
		{
			Statement: `ALTER TABLE tech_book CHANGE COLUMN id creator_id int;`,
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
			Statement: `ALTER TABLE tech_book DROP COLUMN id;`,
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
						ALTER TABLE book ADD COLUMN contentString varchar(255);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`contentString` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    6,
				},
			},
		},
		{
			Statement: `CREATE TABLE book(
							id int,
							createdTs timestamp,
							updaterId int,
							updated_ts timestamp);
						CREATE TABLE student(
							id int,
							createdTs timestamp,
							updatedTs timestamp);`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`createdTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    3,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`updaterId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    4,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`student`.`createdTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    8,
				},
				{
					Status:  advisor.Warn,
					Code:    advisor.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`student`.`updatedTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
					Line:    9,
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^[a-z]+(_[a-z]+)*$",
		MaxLength: 64,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingColumnConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleColumnNaming,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, advisor.MockMySQLDatabase)
}
