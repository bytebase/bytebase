package mysql

import (
	"encoding/json"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

func TestNamingIndexConvention(t *testing.T) {
	tests := []test{
		{
			statement: "CREATE INDEX idx_tech_book_id_name ON tech_book(id, name)",
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
			statement: "CREATE INDEX tech_book_id_name ON tech_book(id, name)",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"CREATE INDEX tech_book_id_name ON tech_book(id, name)\" mismatches index naming convention, expect \"^idx_{{table}}_{{column_list}}$\" but found \"tech_book_id_name\"",
				},
			},
		},
		{
			statement: "ALTER TABLE tech_book RENAME INDEX tech_book_id_name TO idx_tech_book_id_name",
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
			statement: "ALTER TABLE tech_book ADD INDEX idx_tech_book_id_name (id, name)",
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
			statement: "ALTER TABLE tech_book ADD INDEX tech_book_id_name (id, name)",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"ALTER TABLE tech_book ADD INDEX tech_book_id_name (id, name)\" mismatches index naming convention, expect \"^idx_{{table}}_{{column_list}}$\" but found \"tech_book_id_name\"",
				},
			},
		},
		{
			statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), INDEX idx_tech_book_name (name))",
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
			statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), INDEX (name))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), INDEX (name))\" mismatches index naming convention, expect \"^idx_{{table}}_{{column_list}}$\" but found \"\"",
				},
			},
		},
	}

	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "^idx_{{table}}_{{column_list}}$",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingIndexConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleIDXNaming,
		Level:   api.SchemaRuleLevelError,
		Payload: string(payload),
	})
}
