package pg

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingPKConvention(t *testing.T) {
	maxLength := 32

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
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^pk_tech_book_id_name$\" but found \"tech_book_id_name\"",
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
			Statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20), CONSTRAINT tech_book_name PRIMARY KEY (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^pk_tech_book_name$\" but found \"tech_book_name\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20), PRIMARY KEY (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^pk_tech_book_name$\" but found \"\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20) PRIMARY KEY)",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^pk_tech_book_name$\" but found \"\"",
				},
			},
		},
		{
			Statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20), PRIMARY KEY (name))",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NamingPKConventionMismatch,
					Title:   "naming.index.pk",
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^pk_tech_book_name$\" but found \"\"",
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
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^pk_tech_book_id_name$\" but found \"pk_tech_book\"",
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
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^pk_tech_book_id_name$\" but found \"pk_tech_book\"",
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
					Content: "Primary key in table \"tech_book\" mismatches the naming convention, expect \"^pk_tech_book_id_name$\" but found \"pk_tech_book\"",
				},
			},
		},
	}

	payload, err := json.Marshal(advisor.NamingRulePayload{
		Format:    "^pk_{{table}}_{{column_list}}$",
		MaxLength: maxLength,
	})
	require.NoError(t, err)
	advisor.RunSQLReviewRuleTests(t, tests, &NamingPKConventionAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRulePKNaming,
		Level:   advisor.SchemaRuleLevelError,
		Payload: string(payload),
	}, advisor.MockPostgreSQLDatabase)
}
