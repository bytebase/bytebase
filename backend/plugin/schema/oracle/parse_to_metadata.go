package oracle

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/antlr4-go/antlr/v4"
	plsqlparser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterParseToMetadatas(storepb.Engine_ORACLE, ParseToMetadata)
}

func ParseToMetadata(defaultSchemaName string, schema string) (*storepb.DatabaseSchemaMetadata, error) {
	tree, _, err := plsql.ParsePLSQL(schema)
	if err != nil {
		return nil, err
	}

	transformer := &transformerListener{
		state:         newDatabaseState(),
		defaultSchema: defaultSchemaName,
	}

	antlr.ParseTreeWalkerDefault.Walk(transformer, tree)

	return transformer.state.convertToDatabaseMetadata(), transformer.err
}

type transformerListener struct {
	*plsqlparser.BasePlSqlParserListener

	state         *databaseState
	defaultSchema string
	err           error
}

func (t *transformerListener) EnterCreate_table(ctx *plsqlparser.Create_tableContext) {
	if t.err != nil {
		return
	}

	schemaName := plsql.NormalizeSchemaName(ctx.Schema_name())
	tableName := plsql.NormalizeTableName(ctx.Table_name())

	if schemaName == "" {
		schemaName = t.defaultSchema
	}

	tableDefine := ctx.Relational_table()
	if tableDefine == nil {
		return
	}

	schema := t.state.schemas[schemaName]
	if schema == nil {
		schema = newSchemaState(len(t.state.schemas), schemaName)
		t.state.schemas[schemaName] = schema
	}

	if _, ok := schema.tables[tableName]; ok {
		t.err = errors.New("multiple table names found: " + tableName)
		return
	}

	table := newTableState(len(schema.tables), tableName)
	schema.tables[tableName] = table

	for _, item := range tableDefine.AllRelational_property() {
		switch {
		case item.Column_definition() != nil:
			column := item.Column_definition()
			_, _, columnName := plsql.NormalizeColumnName(column.Column_name())

			if _, ok := table.columns[columnName]; ok {
				t.err = errors.New("multiple column names found: " + columnName + " in table: " + tableName)
				return
			}

			var dataType string
			if column.Datatype() != nil {
				dataType = getDataTypePlainText(column.Datatype())
			} else if column.Regular_id() != nil {
				dataType = column.GetParser().GetTokenStream().GetTextFromRuleContext(column.Regular_id())
			}

			columnState := &columnState{
				id:           len(table.columns),
				name:         columnName,
				tp:           dataType,
				defaultValue: nil,
				comment:      "",
				nullable:     true,
			}
			table.columns[columnName] = columnState

			if column.DEFAULT() != nil && column.Expression() != nil {
				columnState.defaultValue = &defaultValueExpression{
					value: column.GetParser().GetTokenStream().GetTextFromRuleContext(column.Expression()),
				}
			}

			for _, constraint := range column.AllInline_constraint() {
				if constraint.NULL_() != nil {
					columnState.nullable = constraint.NOT() == nil
				}
			}
		case item.Out_of_line_constraint() != nil:
			constraint := item.Out_of_line_constraint()
			switch {
			case constraint.UNIQUE() != nil:
				if constraint.CONSTRAINT() == nil {
					t.err = errors.New("unique constraint without name")
					return
				}
				_, constraintName := plsql.NormalizeConstraintName(constraint.Constraint_name())
				if _, ok := table.indexes[constraintName]; ok {
					t.err = errors.New("multiple index names found: " + constraintName + " in table: " + tableName)
					return
				}

				var keys []string
				for _, column := range constraint.AllColumn_name() {
					_, _, columnName := plsql.NormalizeColumnName(column)
					keys = append(keys, columnName)
				}

				index := &indexState{
					id:      len(table.indexes),
					name:    constraintName,
					keys:    keys,
					unique:  true,
					primary: false,
				}
				table.indexes[constraintName] = index
			case constraint.PRIMARY() != nil:
				if constraint.CONSTRAINT() == nil {
					t.err = errors.New("primary constraint without name")
					return
				}
				_, constraintName := plsql.NormalizeConstraintName(constraint.Constraint_name())
				if _, ok := table.indexes[constraintName]; ok {
					t.err = errors.New("multiple index names found: " + constraintName + " in table: " + tableName)
					return
				}

				var keys []string
				for _, column := range constraint.AllColumn_name() {
					_, _, columnName := plsql.NormalizeColumnName(column)
					keys = append(keys, columnName)
				}

				index := &indexState{
					id:      len(table.indexes),
					name:    constraintName,
					keys:    keys,
					unique:  true,
					primary: true,
				}
				table.indexes[constraintName] = index
			}
		}
	}
}

func (t *transformerListener) EnterCreate_index(ctx *plsqlparser.Create_indexContext) {
	if t.err != nil {
		return
	}

	schemaName, indexName := plsql.NormalizeIndexName(ctx.Index_name())
	if schemaName == "" {
		schemaName = t.defaultSchema
	}

	indexDefine := ctx.Table_index_clause()
	if indexDefine == nil {
		// We only support index on table for now.
		return
	}

	_, tableName := plsql.NormalizeTableViewName("", indexDefine.Tableview_name())

	schema := t.state.schemas[schemaName]
	if schema == nil {
		// Skip index if schema does not exist.
		return
	}

	table := schema.tables[tableName]
	if table == nil {
		// Skip index if table does not exist.
		return
	}

	if _, ok := table.indexes[indexName]; ok {
		t.err = errors.New("multiple index names found: " + indexName + " in table: " + tableName)
		return
	}

	var keys []string
	for _, expr := range indexDefine.AllIndex_expr_option() {
		if expr.Index_expr().Column_name() != nil {
			_, _, columnName := plsql.NormalizeColumnName(expr.Index_expr().Column_name())
			keys = append(keys, fmt.Sprintf("\"%s\"", columnName))
		} else if expr.Index_expr().Expression() != nil {
			keys = append(keys, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr.Index_expr().Expression()))
		}
	}
	index := &indexState{
		id:      len(table.indexes),
		name:    indexName,
		keys:    keys,
		unique:  ctx.UNIQUE() != nil,
		primary: false,
	}
	table.indexes[indexName] = index
}

func getDataTypePlainText(ctx plsqlparser.IDatatypeContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.Native_datatype_element() != nil {
		if ctx.Precision_part() != nil {
			return ctx.GetParser().GetTokenStream().GetTextFromTokens(ctx.GetStart(), ctx.Precision_part().GetStop())
		}
		return ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Native_datatype_element())
	}

	return ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}
