package snowflake

import (
	parser "github.com/bytebase/parser/snowflake"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type queryTypeListener struct {
	*parser.BaseSnowflakeParserListener

	allSystems bool
	result     base.QueryType
	err        error
}

func (l *queryTypeListener) EnterBatch(ctx *parser.BatchContext) {
	if l.err != nil {
		return
	}

	l.result, l.err = l.getQueryTypeForBatch(ctx)
}

func (l *queryTypeListener) getQueryTypeForBatch(batch parser.IBatchContext) (base.QueryType, error) {
	sqlCommand := batch.Sql_command()
	switch {
	case sqlCommand.Ddl_command() != nil:
		return base.DDL, nil
	case sqlCommand.Dml_command() != nil:
		return l.getQueryTypeForDmlCommand(sqlCommand.Dml_command())
	case sqlCommand.Show_command() != nil:
		return base.SelectInfoSchema, nil
	case sqlCommand.Use_command() != nil:
		return base.Select, nil
	case sqlCommand.Describe_command() != nil:
		return base.SelectInfoSchema, nil
	case sqlCommand.Other_command() != nil:
		return l.getQueryTypeForOtherCommand(sqlCommand.Other_command())
	default:
		return base.QueryTypeUnknown, nil
	}
}

func (l *queryTypeListener) getQueryTypeForDmlCommand(dmlCommand parser.IDml_commandContext) (base.QueryType, error) {
	switch {
	case dmlCommand.Query_statement() != nil:
		if l.allSystems {
			return base.SelectInfoSchema, nil
		}
		return base.Select, nil
	default:
		return base.DML, nil
	}
}

func (*queryTypeListener) getQueryTypeForOtherCommand(otherCommand parser.IOther_commandContext) (base.QueryType, error) {
	switch {
	case otherCommand.Copy_into_table() != nil:
		return base.DML, nil
	case otherCommand.Comment() != nil:
		return base.DDL, nil
	case otherCommand.Set() != nil:
		return base.Select, nil
	default:
		return base.QueryTypeUnknown, nil
	}
}
