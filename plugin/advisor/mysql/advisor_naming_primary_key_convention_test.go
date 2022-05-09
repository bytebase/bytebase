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
					Content: "\"ALTER TABLE tech_book ADD PRIMARY KEY tech_book_id (id)\" mismatches primary key naming convention, expect \"^pk_{{table}}_{{column_list}}$\" but found \"tech_book_id\"",
				},
			},
		},
		{
			// TODO: Test "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20))",
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
					Content: "\"CREATE TABLE tech_book(id INT, name VARCHAR(20), PRIMARY KEY (name))\" mismatches primary key naming convention, expect \"^pk_{{table}}_{{column_list}}$\" but found \"\"",
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
	})
}
