package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingColumnConvention(t *testing.T) {
	tests := []test{
		{
			statement: "CREATE TABLE book(id int, creatorId int)",
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			statement: "CREATE TABLE book(id int, creator_id int)",
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
			statement: `CREATE TABLE book(id int, creator_id int);
						ALTER TABLE book RENAME COLUMN creator_id TO creatorId`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			statement: `ALTER TABLE book RENAME COLUMN creatorId TO creator_id;`,
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
						ALTER TABLE book CHANGE COLUMN creator_id creatorId int;`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: "`book`.`creatorId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			statement: `ALTER TABLE book CHANGE COLUMN creatorId creator_id int;`,
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
			statement: `ALTER TABLE book DROP COLUMN contentString;`,
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
						ALTER TABLE book ADD COLUMN contentString varchar(255);`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: "`book`.`contentString` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
		{
			statement: `CREATE TABLE book(
							id int,
							createdTs timestamp,
							updaterId int,
							updated_ts timestamp);
						CREATE TABLE student(
							id int,
							createdTs timestamp,
							updatedTs timestamp);`,
			want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: "`book`.`createdTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: "`book`.`updaterId` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: "`student`.`createdTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
				{
					Status:  advisor.Warn,
					Code:    common.NamingColumnConventionMismatch,
					Title:   "Mismatch column naming convention",
					Content: "`student`.`updatedTs` mismatches column naming convention, naming format should be \"^[a-z]+(_[a-z]+)*$\"",
				},
			},
		},
	}

	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "^[a-z]+(_[a-z]+)*$",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingColumnConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleColumnNaming,
		Level:   api.SchemaRuleLevelWarning,
		Payload: string(payload),
	}, &MockCatalogService{})
}
