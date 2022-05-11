package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestUseInnoDB(t *testing.T) {
	tests := []test{
		{
			statement: "CREATE TABLE book(id int) ENGINE = INNODB",
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
			statement: "CREATE TABLE book(id int)",
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
			statement: "CREATE TABLE book(id int) ENGINE = CSV",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NotInnoDBEngine,
					Title:   "InnoDB engine is not used",
					Content: "\"CREATE TABLE book(id int) ENGINE = CSV\" doesn't use InnoDB engine",
				},
			},
		},
		{
			statement: "ALTER TABLE book ENGINE = INNODB",
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
			statement: "ALTER TABLE book ENGINE = CSV",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NotInnoDBEngine,
					Title:   "InnoDB engine is not used",
					Content: "\"ALTER TABLE book ENGINE = CSV\" doesn't use InnoDB engine",
				},
			},
		},
		{
			statement: "SET default_storage_engine=INNODB",
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
			statement: "SET default_storage_engine=CSV",
			want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    common.NotInnoDBEngine,
					Title:   "InnoDB engine is not used",
					Content: "\"SET default_storage_engine=CSV\" doesn't use InnoDB engine",
				},
			},
		},
	}
	runSchemaReviewRuleTests(t, tests, &UseInnoDBAdvisor{}, &api.SchemaReviewRule{
		Type:    api.SchemaRuleMySQLEngine,
		Level:   api.SchemaRuleLevelError,
		Payload: "",
	})
}
