package mysql

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/stretchr/testify/require"
)

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
					Code:    common.NamingUKConventionMismatch,
					Title:   "Mismatch unique key naming convention",
					Content: "\"CREATE UNIQUE INDEX tech_book_id_name ON tech_book(id, name)\" mismatches unique key naming convention, expect \"^uk_tech_book_id_name$\" but found \"tech_book_id_name\"",
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
					Code:    common.NamingUKConventionMismatch,
					Title:   "Mismatch unique key naming convention",
					Content: "\"ALTER TABLE tech_book ADD UNIQUE tech_book_id_name (id, name)\" mismatches unique key naming convention, expect \"^uk_tech_book_id_name$\" but found \"tech_book_id_name\"",
				},
			},
		},
		{
			statement: fmt.Sprintf(
				"ALTER TABLE tech_book RENAME INDEX %s TO uk_tech_book_%s",
				MockOldUKName,
				strings.Join(MockIndexColumnList, "_"),
			),
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
			statement: fmt.Sprintf(
				"ALTER TABLE tech_book RENAME INDEX %s TO uk_tech_book",
				MockOldUKName,
			),
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingUKConventionMismatch,
					Title:   "Mismatch unique key naming convention",
					Content: "\"ALTER TABLE tech_book RENAME INDEX old_uk TO uk_tech_book\" mismatches unique key naming convention, expect \"^uk_tech_book_id_name$\" but found \"uk_tech_book\"",
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
					Code:    common.NamingUKConventionMismatch,
					Title:   "Mismatch unique key naming convention",
					Content: "\"CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))\" mismatches unique key naming convention, expect \"^uk_tech_book_name$\" but found \"\"",
				},
			},
		},
		{
			statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE INDEX (name))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingUKConventionMismatch,
					Title:   "Mismatch unique key naming convention",
					Content: "\"CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE INDEX (name))\" mismatches unique key naming convention, expect \"^uk_tech_book_name$\" but found \"\"",
				},
			},
		},
		{
			statement: "CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NamingUKConventionMismatch,
					Title:   "Mismatch unique key naming convention",
					Content: "\"CREATE TABLE tech_book(id INT PRIMARY KEY, name VARCHAR(20), UNIQUE KEY (name))\" mismatches unique key naming convention, expect \"^uk_tech_book_name$\" but found \"\"",
				},
			},
		},
	}

	payload, err := json.Marshal(api.NamingRulePayload{
		Format: "^uk_{{table}}_{{column_list}}$",
	})
	require.NoError(t, err)
	runSchemaReviewRuleTests(t, tests, &NamingUKConventionAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleUKNaming,
		Level:   api.SchemaRuleLevelError,
		Payload: string(payload),
	}, &MockCatalogService{})
}
