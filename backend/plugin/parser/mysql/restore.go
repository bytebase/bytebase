package mysql

import (
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

func GenerateRestoreSQL(statement string, backupDatabase string, backupTable string, originalDatabase string, originalTable string) (string, error) {
	parseResult, err := ParseMySQL(statement)
	if err != nil {
		return "", err
	}

	if len(parseResult) != 1 {
		return "", errors.Errorf("expected 1 statement, but got %d", len(parseResult))
	}

	g := &generator{
		backupDatabase:   backupDatabase,
		backupTable:      backupTable,
		originalDatabase: originalDatabase,
		originalTable:    originalTable,
	}
	antlr.ParseTreeWalkerDefault.Walk(g, parseResult[0].Tree)
	return g.result, g.err
}

type generator struct {
	*parser.BaseMySQLParserListener

	backupDatabase   string
	backupTable      string
	originalDatabase string
	originalTable    string
	result           string
	err              error
}

func (g *generator) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	g.result = fmt.Sprintf("INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s`;", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable)
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
	if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s` ON DUPLICATE KEY UPDATE ", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable); err != nil {
		g.err = err
		return
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
