package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingColumnConvention(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int, creatorId int)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int, creator_id int)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
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
					Code:    common.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: `ALTER TABLE book RENAME COLUMN creatorId TO creator_id;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
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
					Code:    common.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			Statement: `ALTER TABLE book CHANGE COLUMN creatorId creator_id int;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
		{
			Statement: `ALTER TABLE book DROP COLUMN contentString;`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    common.Ok,
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
					Code:    common.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`contentString` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
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
					Code:    common.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`createdTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`book`.`updaterId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`student`.`createdTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "naming.column",
					Content: "`student`.`updatedTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format: "^[a-z]+(_[a-z]+)*$",
	})
	require.NoError(t, err)
	advisor.RunSchemaReviewRuleTests(t, tests, &NamingColumnConventionAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleColumnNaming,
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, &advisor.MockCatalogService{})
}
