package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestMySQLDatabaseAllowDropIfEmpty(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "DROP DATABASE IF EXISTS foo",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.DatabaseNotEmpty,
					Title:   "database.drop-empty-database",
					Content: "Database `foo` is not allowed to drop if not empty",
				},
			},
		},
	}

	advisor.RunSchemaReviewRuleTests(t, tests, &DatabaseAllowDropIfEmptyAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleDropEmptyDatabase,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, &advisor.MockCatalogService{})
}
