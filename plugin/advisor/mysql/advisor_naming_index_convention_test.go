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
					Content: "\"CREATE INDEX tech_book_id_name ON tech_book(id, name)\" mismatches index naming convention. Expect \"idx_tech_book_id_name\" but found \"tech_book_id_name\"",
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
					Content: "\"ALTER TABLE tech_book ADD INDEX tech_book_id_name (id, name)\" mismatches index naming convention. Expect \"idx_tech_book_id_name\" but found \"tech_book_id_name\"",
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
					Content: "\"CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), INDEX (name))\" mismatches index naming convention. Expect \"idx_tech_book_name\" but found \"\"",
				},
			},
		},
	}

	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "idx_{{table}}_{{column_list}}",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingIndexConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleIDXNaming,
		Level:   api.SchemaRuleLevelError,
		Payload: string(payload),
	})
}

func TestNamingUKConvention(t *testing.T) {
	tests := []test{
		{
			statement: "CREATE UNIQUE INDEX uk_tech_book_id_name ON tech_book(id, name)",
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
			statement: "CREATE UNIQUE INDEX tech_book_id_name ON tech_book(id, name)",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"CREATE UNIQUE INDEX tech_book_id_name ON tech_book(id, name)\" mismatches index naming convention. Expect \"uk_tech_book_id_name\" but found \"tech_book_id_name\"",
				},
			},
		},
		{
			statement: "ALTER TABLE tech_book ADD UNIQUE uk_tech_book_id_name (id, name)",
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
			statement: "ALTER TABLE tech_book ADD UNIQUE tech_book_id_name (id, name)",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"ALTER TABLE tech_book ADD UNIQUE tech_book_id_name (id, name)\" mismatches index naming convention. Expect \"uk_tech_book_id_name\" but found \"tech_book_id_name\"",
				},
			},
		},
		{
			statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE INDEX uk_tech_book_name (name))",
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
			statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))\" mismatches index naming convention. Expect \"uk_tech_book_name\" but found \"\"",
				},
			},
		},
		{
			statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE INDEX (name))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE INDEX (name))\" mismatches index naming convention. Expect \"uk_tech_book_name\" but found \"\"",
				},
			},
		},
		{
			statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))\" mismatches index naming convention. Expect \"uk_tech_book_name\" but found \"\"",
				},
			},
		},
	}

	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "uk_{{table}}_{{column_list}}",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingIndexConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleUKNaming,
		Level:   api.SchemaRuleLevelError,
		Payload: string(payload),
	})
}

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
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"ALTER TABLE tech_book ADD PRIMARY KEY tech_book_id (id)\" mismatches index naming convention. Expect \"pk_tech_book_id\" but found \"tech_book_id\"",
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
					Code:    common.NamingIndexConventionMismatch,
					Title:   "Mismatch index naming convention",
					Content: "\"CREATE TABLE tech_book(id INT, name VARCHAR(20), PRIMARY KEY (name))\" mismatches index naming convention. Expect \"pk_tech_book_name\" but found \"\"",
				},
			},
		},
	}

	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "pk_{{table}}_{{column_list}}",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingIndexConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRulePKNaming,
		Level:   api.SchemaRuleLevelError,
		Payload: string(payload),
	})
}
