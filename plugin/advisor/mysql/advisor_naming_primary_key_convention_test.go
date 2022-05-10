package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingPKConvention(t *testing.T) {
	tests := []test{
		{
			statement: "ALTER TABLE tech_book ADD PRIMARY KEY pk_tech_book_id (id)",
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
			statement: "ALTER TABLE tech_book ADD PRIMARY KEY tech_book_id (id)",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingPKConventionMismatch,
					Title:   "Mismatch primary key naming convention",
					Content: "Primary key mismatches the naming convention, expect \"^pk_tech_book_id$\" but found `tech_book_id`",
				},
			},
		},
		{
			statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20), PRIMARY KEY pk_tech_book_name (name))",
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
			statement: "CREATE TABLE tech_book(id INT, name VARCHAR(20), PRIMARY KEY (name))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingPKConventionMismatch,
					Title:   "Mismatch primary key naming convention",
					Content: "Primary key mismatches the naming convention, expect \"^pk_tech_book_name$\" but found ``",
				},
			},
		},
	}

	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "^pk_{{table}}_{{column_list}}$",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingPKConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRulePKNaming,
		Level:   api.SchemaRuleLevelError,
		Payload: string(payload),
	}, &MockCatalogService{})
}
