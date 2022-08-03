package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestUseInnoDB(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int) ENGINE = INNODB",
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
			Statement: "CREATE TABLE book(id int)",
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
			Statement: "CREATE TABLE book(id int) ENGINE = CSV",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NotInnoDBEngine,
					Title:   "engine.mysql.use-innodb",
					Content: "\"CREATE TABLE book(id int) ENGINE = CSV\" doesn't use InnoDB engine",
				},
			},
		},
		{
			Statement: "ALTER TABLE book ENGINE = INNODB",
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
			Statement: "ALTER TABLE book ENGINE = CSV",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NotInnoDBEngine,
					Title:   "engine.mysql.use-innodb",
					Content: "\"ALTER TABLE book ENGINE = CSV\" doesn't use InnoDB engine",
				},
			},
		},
		{
			Statement: "SET default_storage_engine=INNODB",
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
			Statement: "SET default_storage_engine=CSV",
			Want: []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.NotInnoDBEngine,
					Title:   "engine.mysql.use-innodb",
					Content: "\"SET default_storage_engine=CSV\" doesn't use InnoDB engine",
				},
			},
		},
	}
	advisor.RunSQLReviewRuleTests(t, tests, &UseInnoDBAdvisor{}, &advisor.SQLReviewRule{
		Type:    advisor.SchemaRuleMySQLEngine,
		Level:   advisor.SchemaRuleLevelError,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
