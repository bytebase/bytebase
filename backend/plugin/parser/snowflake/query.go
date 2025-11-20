package snowflake

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_SNOWFLAKE, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	parseResults, err := ParseSnowSQL(statement)
	if err != nil {
		return false, false, err
	}
	l := &queryValidateListener{
		valid: true,
	}
	for _, parseResult := range parseResults {
		antlr.ParseTreeWalkerDefault.Walk(l, parseResult.Tree)
		if !l.valid {
			return false, false, nil
		}
	}
	return true, !l.hasExecute, nil
}

type queryValidateListener struct {
	*parser.BaseSnowflakeParserListener

	valid      bool
	hasExecute bool
}

func (l *queryValidateListener) EnterSql_command(ctx *parser.Sql_commandContext) {
	if !l.valid {
		return
	}
	if ctx.Dml_command() == nil && ctx.Other_command() == nil && ctx.Describe_command() == nil && ctx.Show_command() == nil {
		l.valid = false
		return
	}
	if dml := ctx.Dml_command(); dml != nil {
		if dml.Query_statement() == nil {
			l.valid = false
			return
		}
	}
	if other := ctx.Other_command(); other != nil {
		if other.Set() != nil {
			l.hasExecute = true
			return
		}
		if other.Explain() == nil {
			l.valid = false
			return
		}
	}
}
