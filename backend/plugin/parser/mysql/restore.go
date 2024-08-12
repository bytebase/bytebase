package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MYSQL, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupDatabase string, backupTable string, originalDatabase string, originalTable string) (string, error) {
	parseResult, err := ParseMySQL(statement)
	if err != nil {
		return "", err
	}

	if len(parseResult) != 1 {
		return "", errors.Errorf("expected 1 statement, but got %d", len(parseResult))
	}

	generatedColumns, normalColumns, err := classifyColumns(ctx, rCtx.GetDatabaseMetadataFunc, rCtx.InstanceID, &TableReference{
		Database: originalDatabase,
		Table:    originalTable,
	})

	if err != nil {
		return "", errors.Wrapf(err, "failed to classify columns for %s.%s", originalDatabase, originalTable)
	}

	g := &generator{
		ctx:              ctx,
		rCtx:             rCtx,
		backupDatabase:   backupDatabase,
		backupTable:      backupTable,
		originalDatabase: originalDatabase,
		originalTable:    originalTable,
		generatedColumns: generatedColumns,
		normalColumns:    normalColumns,
	}
	var buf strings.Builder
	antlr.ParseTreeWalkerDefault.Walk(g, parseResult[0].Tree)
	if g.err != nil {
		return "", g.err
	}

	if _, err := fmt.Fprintf(&buf, "/*\nOriginal SQL:\n%s\n*/\n%s", statement, g.result); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type generator struct {
	*parser.BaseMySQLParserListener

	ctx  context.Context
	rCtx base.RestoreContext

	backupDatabase   string
	backupTable      string
	originalDatabase string
	originalTable    string
	generatedColumns []string
	normalColumns    []string
	result           string
	err              error
}

func (g *generator) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if len(g.generatedColumns) == 0 {
		g.result = fmt.Sprintf("INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s`;", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable)
	} else {
		var quotedColumns []string
		for _, column := range g.normalColumns {
			quotedColumns = append(quotedColumns, fmt.Sprintf("`%s`", column))
		}
		quotedColumnList := strings.Join(quotedColumns, ", ")
		g.result = fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) SELECT %s FROM `%s`.`%s`;", g.originalDatabase, g.originalTable, quotedColumnList, quotedColumnList, g.backupDatabase, g.backupTable)
	}
}

func (g *generator) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	singleTables := &singleTableListener{
		databaseName: g.originalDatabase,
		singleTables: make(map[string]*TableReference),
	}

	antlr.ParseTreeWalkerDefault.Walk(singleTables, ctx.TableReferenceList())

	tableName := g.originalTable

	for _, table := range singleTables.singleTables {
		if table.Table == g.originalTable && table.Alias != "" {
			tableName = table.Alias
		}
	}

	setFields := &setFieldListener{
		table: tableName,
	}

	antlr.ParseTreeWalkerDefault.Walk(setFields, ctx.UpdateList())

	var buf strings.Builder
	if len(g.generatedColumns) == 0 {
		if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s` ON DUPLICATE KEY UPDATE ", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable); err != nil {
			g.err = err
			return
		}
	} else {
		var quotedColumns []string
		for _, column := range g.normalColumns {
			quotedColumns = append(quotedColumns, fmt.Sprintf("`%s`", column))
		}
		quotedColumnList := strings.Join(quotedColumns, ", ")
		if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` (%s) SELECT %s FROM `%s`.`%s` ON DUPLICATE KEY UPDATE ", g.originalDatabase, g.originalTable, quotedColumnList, quotedColumnList, g.backupDatabase, g.backupTable); err != nil {
			g.err = err
			return
		}
	}

	for i, field := range setFields.result {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				g.err = err
				return
			}
		}

		if _, err := fmt.Fprintf(&buf, "`%s` = VALUES(`%s`)", field, field); err != nil {
			g.err = err
			return
		}
	}
	if _, err := buf.WriteString(";"); err != nil {
		g.err = err
		return
	}
	g.result = buf.String()
}

type setFieldListener struct {
	*parser.BaseMySQLParserListener

	table  string
	result []string
}

func (l *setFieldListener) EnterUpdateElement(ctx *parser.UpdateElementContext) {
	_, tableName, columnName := NormalizeMySQLColumnRef(ctx.ColumnRef())
	if tableName == l.table || tableName == "" {
		l.result = append(l.result, columnName)
	}
}
