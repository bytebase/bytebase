package pg

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestNamingPKConvention(t *testing.T) {
	maxLength := 32
	invalidPKName := advisor.RandomString(33)

	tests := []advisor.TestCase{
		{
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id_name PRIMARY KEY (id, name)",
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
			Statement: "ALTER TABLE tech_book ADD CONSTRAINT tech_book_id_name PRIMARY KEY (id, name)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^$|^pk_tech_book_id_name$\" but found \"tech_book_id_name\"",
					Line:    1,
				},
			},
		},
		{
			Statement: fmt.Sprintf("ALTER TABLE tech_book ADD CONSTRAINT %s PRIMARY KEY (id, name)", invalidPKName),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: fmt.Sprintf("Primary key in table \"tech_book\" mismatches the naming convention, expect \"^$|^pk_tech_book_id_name$\" but found \"%s\"", invalidPKName),
					Line:    1,
				},
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: fmt.Sprintf(`Primary key "%s" in table "tech_book" mismatches the naming convention, its length should be within %d characters`, invalidPKName, maxLength),
					Line:    1,
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20), CONSTRAINT pk_tech_book_name PRIMARY KEY (name))",
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
			Statement: `-- this is the first line.
				CREATE TABLE tech_book(
					id INT,
					name VARCHAR(20),
					CONSTRAINT tech_book_name PRIMARY KEY (name)
				)`,
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^$|^pk_tech_book_name$\" but found \"tech_book_name\"",
					Line:    5,
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20), PRIMARY KEY (name))",
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
			Statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20) PRIMARY KEY)",
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
			Statement: fmt.Sprintf(
				"ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_%s PRIMARY KEY USING INDEX %s",
				strings.Join(advisor.MockIndexColumnList, "_"),
				advisor.MockOldIndexName,
			),
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
			Statement: fmt.Sprintf(
				"ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book PRIMARY KEY USING INDEX %s",
				advisor.MockOldIndexName,
			),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^$|^pk_tech_book_id_name$\" but found \"pk_tech_book\"",
					Line:    1,
				},
			},
		},
		{
			Statement: fmt.Sprintf(
				"ALTER TABLE tech_book RENAME CONSTRAINT %s TO pk_tech_book_%s",
				advisor.MockOldPostgreSQLPKName,
				strings.Join(advisor.MockIndexColumnList, "_"),
			),
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
			Statement: fmt.Sprintf(
				"ALTER TABLE tech_book RENAME CONSTRAINT %s TO pk_tech_book",
				advisor.MockOldPostgreSQLPKName,
			),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^$|^pk_tech_book_id_name$\" but found \"pk_tech_book\"",
					Line:    1,
				},
			},
		},
		{
			Statement: fmt.Sprintf(
				"ALTER INDEX %s RENAME TO pk_tech_book_%s",
				advisor.MockOldPostgreSQLPKName,
				strings.Join(advisor.MockIndexColumnList, "_"),
			),
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
			Statement: fmt.Sprintf(
				"ALTER INDEX %s RENAME TO pk_tech_book",
				advisor.MockOldPostgreSQLPKName,
			),
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^$|^pk_tech_book_id_name$\" but found \"pk_tech_book\"",
					Line:    1,
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^$|^pk_{{table}}_{{column_list}}$",
		MaxLength: maxLength,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingPKConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRulePKNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
