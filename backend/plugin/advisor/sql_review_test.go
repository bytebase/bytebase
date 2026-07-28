package advisor

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestSQLReviewCheckMaximumSQLSizeDoesNotParseExceededStatement(t *testing.T) {
	sm := sheet.NewManager()
	statement := "this is intentionally not valid sql"

	adviceList, err := SQLReviewCheck(context.Background(), sm, statement, []*storepb.SQLReviewRule{
		{
			Type:   storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_ORACLE,
			Payload: &storepb.SQLReviewRule_NumberPayload{
				NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{
					Number: 10,
				},
			},
		},
	}, Context{
		DBType:          storepb.Engine_ORACLE,
		NoAppendBuiltin: true,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_ERROR, adviceList[0].Status)
	require.Equal(t, code.StatementExceedMaximumSQLSize.Int32(), adviceList[0].Code)
	require.Equal(t, storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE.String(), adviceList[0].Title)
	require.Contains(t, adviceList[0].Content, "exceeds the maximum SQL size")
}

func TestSQLReviewCheckMaximumSQLSizeBuiltinDoesNotParseLargeStatement(t *testing.T) {
	sm := sheet.NewManager()
	statement := strings.Repeat("not sql\n", common.MaxSheetCheckSize/len("not sql\n")+1)

	adviceList, err := SQLReviewCheck(context.Background(), sm, statement, nil, Context{
		DBType: storepb.Engine_ORACLE,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_WARNING, adviceList[0].Status)
	require.Equal(t, code.StatementExceedMaximumSQLSize.Int32(), adviceList[0].Code)
	require.Equal(t, storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE.String(), adviceList[0].Title)
}

func TestSQLReviewCheckMaximumSQLSizeBuiltinIsEngineAware(t *testing.T) {
	sm := sheet.NewManager()
	statement := strings.Repeat("not sql\n", common.MaxSheetCheckSize/len("not sql\n")+1)

	adviceList, err := SQLReviewCheck(context.Background(), sm, statement, []*storepb.SQLReviewRule{
		{
			Type:   storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_MYSQL,
			Payload: &storepb.SQLReviewRule_NumberPayload{
				NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{
					Number: int32(len(statement) + 1),
				},
			},
		},
	}, Context{
		DBType: storepb.Engine_POSTGRES,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_WARNING, adviceList[0].Status)
	require.Equal(t, code.StatementExceedMaximumSQLSize.Int32(), adviceList[0].Code)
	require.Equal(t, storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE.String(), adviceList[0].Title)
}

func TestSQLReviewCheckMaximumSQLSizeBuiltinRunsWhenConfigHasOnlyOtherEngineRules(t *testing.T) {
	sm := sheet.NewManager()
	statement := strings.Repeat("not sql\n", common.MaxSheetCheckSize/len("not sql\n")+1)

	adviceList, err := SQLReviewCheck(context.Background(), sm, statement, []*storepb.SQLReviewRule{
		{
			Type:   storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_MYSQL,
		},
	}, Context{
		DBType: storepb.Engine_POSTGRES,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_WARNING, adviceList[0].Status)
	require.Equal(t, code.StatementExceedMaximumSQLSize.Int32(), adviceList[0].Code)
	require.Equal(t, storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE.String(), adviceList[0].Title)
}

func TestSQLReviewCheckMaximumSQLSizeBuiltinRunsWithMySQLOnlyConfigForPostgres(t *testing.T) {
	sm := sheet.NewManager()
	statement := strings.Repeat("not sql\n", common.MaxSheetCheckSize/len("not sql\n")+1)

	adviceList, err := SQLReviewCheck(context.Background(), sm, statement, []*storepb.SQLReviewRule{
		{
			Type:   storepb.SQLReviewRule_TABLE_REQUIRE_PK,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_MYSQL,
		},
		{
			Type:   storepb.SQLReviewRule_COLUMN_REQUIRED,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_MYSQL,
			Payload: &storepb.SQLReviewRule_StringArrayPayload{
				StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{
					List: []string{"id", "created_ts", "updated_ts", "creator_id", "updater_id"},
				},
			},
		},
		{
			Type:   storepb.SQLReviewRule_COLUMN_NO_NULL,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: storepb.Engine_MYSQL,
		},
		{
			Type:   storepb.SQLReviewRule_NAMING_COLUMN,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_MYSQL,
			Payload: &storepb.SQLReviewRule_NamingPayload{
				NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{
					Format: "^[a-z]+(_[a-z]+)*$",
				},
			},
		},
		{
			Type:   storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_MYSQL,
			Payload: &storepb.SQLReviewRule_NumberPayload{
				NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{
					Number: 4194304,
				},
			},
		},
		{
			Type:   storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: storepb.Engine_MYSQL,
		},
		{
			Type:   storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: storepb.Engine_MYSQL,
		},
	}, Context{
		DBType: storepb.Engine_POSTGRES,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_WARNING, adviceList[0].Status)
	require.Equal(t, code.StatementExceedMaximumSQLSize.Int32(), adviceList[0].Code)
	require.Equal(t, storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE.String(), adviceList[0].Title)
}

func TestSQLReviewCheckMaximumSQLSizeUsesExactEngineRule(t *testing.T) {
	sm := sheet.NewManager()
	statement := strings.Repeat("not sql\n", common.MaxSheetCheckSize/len("not sql\n")+1)

	adviceList, err := SQLReviewCheck(context.Background(), sm, statement, []*storepb.SQLReviewRule{
		{
			Type:   storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE,
			Level:  storepb.SQLReviewRule_WARNING,
			Engine: storepb.Engine_ENGINE_UNSPECIFIED,
			Payload: &storepb.SQLReviewRule_NumberPayload{
				NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{
					Number: int32(len(statement) + 1),
				},
			},
		},
		{
			Type:   storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_POSTGRES,
			Payload: &storepb.SQLReviewRule_NumberPayload{
				NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{
					Number: 10,
				},
			},
		},
	}, Context{
		DBType:          storepb.Engine_POSTGRES,
		NoAppendBuiltin: true,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_ERROR, adviceList[0].Status)
	require.Equal(t, code.StatementExceedMaximumSQLSize.Int32(), adviceList[0].Code)
	require.Equal(t, storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE.String(), adviceList[0].Title)
}

func TestSQLReviewCheckMaximumSQLSizeUsesConfiguredMaximum(t *testing.T) {
	sm := sheet.NewManager()
	statement := strings.Repeat("not sql\n", common.MaxSheetCheckSize/len("not sql\n")+1)

	adviceList, err := SQLReviewCheck(context.Background(), sm, statement, []*storepb.SQLReviewRule{
		{
			Type:   storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE,
			Level:  storepb.SQLReviewRule_ERROR,
			Engine: storepb.Engine_ORACLE,
			Payload: &storepb.SQLReviewRule_NumberPayload{
				NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{
					Number: int32(len(statement) + 1),
				},
			},
		},
	}, Context{
		DBType:          storepb.Engine_ORACLE,
		NoAppendBuiltin: true,
	})

	require.NoError(t, err)
	require.NotEmpty(t, adviceList)
	require.NotEqual(t, code.StatementExceedMaximumSQLSize.Int32(), adviceList[0].Code)
}

func TestMaximumSQLSizeBuiltinRule(t *testing.T) {
	for engineValue := range storepb.Engine_name {
		engine := storepb.Engine(engineValue)
		if engine == storepb.Engine_ENGINE_UNSPECIFIED {
			continue
		}

		rules := GetBuiltinRules(engine)

		var rule *storepb.SQLReviewRule
		for _, r := range rules {
			if r.Type == storepb.SQLReviewRule_BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE {
				rule = r
				break
			}
		}

		require.NotNil(t, rule)
		require.Equal(t, storepb.SQLReviewRule_WARNING, rule.Level)
		require.Equal(t, engine, rule.Engine)
		require.Equal(t, int32(common.MaxSheetCheckSize), rule.GetNumberPayload().Number)
	}
}
