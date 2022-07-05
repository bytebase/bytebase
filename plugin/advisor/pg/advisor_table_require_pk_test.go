package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestRequirePK(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE t(id INT PRIMARY KEY)",
			Want:      []advisor.Advice{},
		},
	}

	advisor.RunSchemaReviewRuleTests(t, tests, &TableRequirePKAdvisor{}, &advisor.SchemaReviewRule{
		Type:    advisor.SchemaRuleTableRequirePK,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, &advisor.MockCatalogService{})
}
