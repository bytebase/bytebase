package tsql

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MSSQL, extractChangedResources)
}

func extractChangedResources(currentDatabase string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, statement string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)
	l := &tsqlChangedResourceExtractListener{
		currentDatabase:  currentDatabase,
		currentSchema:    currentSchema,
		dbMetadata:       dbMetadata,
		changedResources: changedResources,
		statement:        statement,
	}

	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for MSSQL")
		}
		antlr.ParseTreeWalkerDefault.Walk(l, antlrAST.Tree)
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS:       l.sampleDMLs,
		DMLCount:         l.dmlCount,
		InsertCount:      l.insertCount,
	}, nil
}

type tsqlChangedResourceExtractListener struct {
	*parser.BaseTSqlParserListener

	currentDatabase  string
	currentSchema    string
	dbMetadata       *model.DatabaseMetadata
	changedResources *model.ChangedResources
	statement        string
	sampleDMLs       []string
	dmlCount         int
	insertCount      int

	// Internal data structure used temporarily.
	text string
}

func (l *tsqlChangedResourceExtractListener) EnterSql_clauses(ctx *parser.Sql_clausesContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (l *tsqlChangedResourceExtractListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	d, s, t := normalizeTableNameSeparated(tableName, l.currentDatabase, l.currentSchema, false /* caseSensitive */)

	l.changedResources.AddTable(
		d,
		s,
		&storepb.ChangedResourceTable{
			Name: t,
		},
		false)
}

func (l *tsqlChangedResourceExtractListener) EnterDrop_table(ctx *parser.Drop_tableContext) {
	if ctx.AllTable_name() == nil {
		return
	}

	for _, tableName := range ctx.AllTable_name() {
		d, s, t := normalizeTableNameSeparated(tableName, l.currentDatabase, l.currentSchema, false /* caseSensitive */)

		l.changedResources.AddTable(
			d,
			s,
			&storepb.ChangedResourceTable{
				Name: t,
			},
			true)
	}
}

func (l *tsqlChangedResourceExtractListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	if ctx.AllTable_name() == nil {
		return
	}

	for _, tableName := range ctx.AllTable_name() {
		d, s, t := normalizeTableNameSeparated(tableName, l.currentDatabase, l.currentSchema, false /* caseSensitive */)

		l.changedResources.AddTable(
			d,
			s,
			&storepb.ChangedResourceTable{
				Name: t,
			},
			true)
	}
}

func (l *tsqlChangedResourceExtractListener) EnterCreate_index(ctx *parser.Create_indexContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	d, s, t := normalizeTableNameSeparated(tableName, l.currentDatabase, l.currentSchema, false /* caseSensitive */)

	l.changedResources.AddTable(
		d,
		s,
		&storepb.ChangedResourceTable{
			Name: t,
		},
		false)
}

// EnterDrop_index is called when production drop_index is entered.
func (l *tsqlChangedResourceExtractListener) EnterDrop_index(ctx *parser.Drop_indexContext) {
	for _, index := range ctx.AllDrop_relational_or_xml_or_spatial_index() {
		fullTable, err := NormalizeFullTableName(index.Full_table_name())
		if err != nil {
			continue
		}
		d, _ := NormalizeTSQLIdentifierText(l.currentDatabase)
		s, _ := NormalizeTSQLIdentifierText(l.currentSchema)
		if fullTable.Database != "" {
			d = fullTable.Database
		}
		if fullTable.Schema != "" {
			s = fullTable.Schema
		}

		l.changedResources.AddTable(
			d,
			s,
			&storepb.ChangedResourceTable{
				Name: fullTable.Table,
			},
			false)
	}
}

func (l *tsqlChangedResourceExtractListener) EnterInsert_statement(ctx *parser.Insert_statementContext) {
	if ctx.Ddl_object() != nil && ctx.Ddl_object().Full_table_name() != nil {
		table, err := NormalizeFullTableName(ctx.Ddl_object().Full_table_name())
		if err == nil && table != nil && table.Table != "" {
			d := table.Database
			if d == "" {
				d = l.currentDatabase
			}
			s := table.Schema
			if s == "" {
				s = l.currentSchema
			}
			l.changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name: table.Table,
				},
				false,
			)
		}
	}

	if ctx.Insert_statement_value() != nil && ctx.Insert_statement_value().Derived_table() != nil && ctx.Insert_statement_value().Derived_table().Table_value_constructor() != nil {
		tvc := ctx.Insert_statement_value().Derived_table().Table_value_constructor()
		l.insertCount += len(tvc.AllExpression_list_())
		return
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, trimStatement(l.text))
	}
}

func (l *tsqlChangedResourceExtractListener) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if ctx.Ddl_object() != nil && ctx.Ddl_object().Full_table_name() != nil {
		table, err := NormalizeFullTableName(ctx.Ddl_object().Full_table_name())
		if err == nil && table != nil && table.Table != "" {
			d := table.Database
			if d == "" {
				d = l.currentDatabase
			}
			s := table.Schema
			if s == "" {
				s = l.currentSchema
			}
			l.changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name: table.Table,
				},
				false,
			)
		}
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, trimStatement(l.text))
	}
}

func (l *tsqlChangedResourceExtractListener) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	from := ctx.Delete_statement_from()
	if from.Ddl_object() != nil && from.Ddl_object().Full_table_name() != nil {
		table, err := NormalizeFullTableName(from.Ddl_object().Full_table_name())
		if err == nil && table != nil && table.Table != "" {
			d := table.Database
			if d == "" {
				d = l.currentDatabase
			}
			s := table.Schema
			if s == "" {
				s = l.currentSchema
			}
			l.changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name: table.Table,
				},
				false,
			)
		}
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, trimStatement(l.text))
	}
}

func trimStatement(statement string) string {
	// TODO(d): why test is "UPDATE t1 SET c1 = 5\n;".
	return strings.TrimLeftFunc(strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon), unicode.IsSpace) + ";"
}
