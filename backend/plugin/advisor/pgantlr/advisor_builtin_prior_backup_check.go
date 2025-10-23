package pgantlr

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

const (
	defaultSchema = "public"
)

var (
	_ advisor.Advisor = (*BuiltinPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.BuiltinRulePriorBackupCheck, &BuiltinPriorBackupCheckAdvisor{})
}

// BuiltinPriorBackupCheckAdvisor is the advisor checking for disallow mix DDL and DML.
type BuiltinPriorBackupCheckAdvisor struct {
}

// Check checks for disallow mix DDL and DML.
func (*BuiltinPriorBackupCheckAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup || checkCtx.ChangeType != storepb.PlanCheckRunConfig_DML {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(checkCtx.Rule.Type)

	var adviceList []*storepb.Advice

	// Check for DDL statements in DML context
	tree, err := pg.ParsePostgreSQL(checkCtx.Statements)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	ddlChecker := &ddlChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        title,
		tokens:                       tree.Tokens,
	}
	antlr.ParseTreeWalkerDefault.Walk(ddlChecker, tree.Tree)
	adviceList = append(adviceList, ddlChecker.adviceList...)

	// Check if backup schema exists
	schemaName := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	if !checkCtx.Catalog.Origin.HasSchema(schemaName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Need schema %q to do prior backup but it does not exist", schemaName),
			Code:          advisor.SchemaNotExists.Int32(),
			StartPosition: nil,
		})
	}

	// Check statement type consistency for each table
	statementInfoList, err := prepareTransformation(checkCtx.Statements)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare transformation")
	}

	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s", item.table.Schema, item.table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements on the same table.
	for key, list := range groupByTable {
		statementType := StatementTypeUnknown
		for _, item := range list {
			if statementType == StatementTypeUnknown {
				statementType = item.table.StatementType
			}
			if statementType != item.table.StatementType {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("The statement type is not the same for all statements on the same table %q", key),
					Code:          advisor.BuiltinPriorBackupCheck.Int32(),
					StartPosition: nil,
				})
				break
			}
		}
	}

	return adviceList, nil
}

// isDDLStatement checks if a context is a DDL statement
func isDDLStatement(ctx antlr.Tree) bool {
	switch ctx.(type) {
	// CREATE statements
	case *parser.CreatestmtContext,
		*parser.CreateasstmtContext,
		*parser.CreateassertionstmtContext,
		*parser.CreatecaststmtContext,
		*parser.CreateconversionstmtContext,
		*parser.CreatedomainstmtContext,
		*parser.CreateextensionstmtContext,
		*parser.CreatefdwstmtContext,
		*parser.CreateforeignserverstmtContext,
		*parser.CreateforeigntablestmtContext,
		*parser.CreatefunctionstmtContext,
		*parser.CreategroupstmtContext,
		*parser.CreatematviewstmtContext,
		*parser.CreateopclassstmtContext,
		*parser.CreateopfamilystmtContext,
		*parser.CreatepublicationstmtContext,
		*parser.CreatepolicystmtContext,
		*parser.CreateplangstmtContext,
		*parser.CreateschemastmtContext,
		*parser.CreateseqstmtContext,
		*parser.CreatesubscriptionstmtContext,
		*parser.CreatestatsstmtContext,
		*parser.CreatetablespacestmtContext,
		*parser.CreatetransformstmtContext,
		*parser.CreatetrigstmtContext,
		*parser.CreateeventtrigstmtContext,
		*parser.CreaterolestmtContext,
		*parser.CreateuserstmtContext,
		*parser.CreateusermappingstmtContext,
		*parser.CreatedbstmtContext,
		*parser.CreateamstmtContext,
		*parser.IndexstmtContext, // CREATE INDEX
		// ALTER statements
		*parser.AltereventtrigstmtContext,
		*parser.AltercollationstmtContext,
		*parser.AlterdatabasestmtContext,
		*parser.AlterdatabasesetstmtContext,
		*parser.AlterdefaultprivilegesstmtContext,
		*parser.AlterdomainstmtContext,
		*parser.AlterenumstmtContext,
		*parser.AlterextensionstmtContext,
		*parser.AlterextensioncontentsstmtContext,
		*parser.AlterfdwstmtContext,
		*parser.AlterforeignserverstmtContext,
		*parser.AlterfunctionstmtContext,
		*parser.AltergroupstmtContext,
		*parser.AlterobjectdependsstmtContext,
		*parser.AlterobjectschemastmtContext,
		*parser.AlterownerstmtContext,
		*parser.AlteroperatorstmtContext,
		*parser.AltertypestmtContext,
		*parser.AlterpolicystmtContext,
		*parser.AlterseqstmtContext,
		*parser.AltersystemstmtContext,
		*parser.AltertablestmtContext,
		*parser.AltertblspcstmtContext,
		*parser.AltercompositetypestmtContext,
		*parser.AlterpublicationstmtContext,
		*parser.AlterrolesetstmtContext,
		*parser.AlterrolestmtContext,
		*parser.AltersubscriptionstmtContext,
		*parser.AlterstatsstmtContext,
		*parser.AltertsconfigurationstmtContext,
		*parser.AltertsdictionarystmtContext,
		*parser.AlterusermappingstmtContext,
		*parser.AlteropfamilystmtContext,
		// DROP statements
		*parser.DropstmtContext,
		*parser.DropcaststmtContext,
		*parser.DropopclassstmtContext,
		*parser.DropopfamilystmtContext,
		*parser.DropownedstmtContext,
		*parser.DropsubscriptionstmtContext,
		*parser.DroptablespacestmtContext,
		*parser.DroptransformstmtContext,
		*parser.DroprolestmtContext,
		*parser.DropusermappingstmtContext,
		*parser.DropdbstmtContext,
		// Other DDL statements
		*parser.TruncatestmtContext,
		*parser.RenamestmtContext,
		*parser.CommentstmtContext,
		*parser.DefinestmtContext,
		*parser.RemoveaggrstmtContext,
		*parser.RemovefuncstmtContext,
		*parser.RemoveoperstmtContext,
		*parser.ReindexstmtContext,
		*parser.ClusterstmtContext,
		*parser.RefreshmatviewstmtContext,
		*parser.RulestmtContext,
		*parser.SeclabelstmtContext,
		*parser.ReassignownedstmtContext:
		return true
	default:
		return false
	}
}

// ddlChecker checks for DDL statements in DML context
type ddlChecker struct {
	*parser.BasePostgreSQLParserListener

	level      storepb.Advice_Status
	title      string
	tokens     *antlr.CommonTokenStream
	adviceList []*storepb.Advice
}

func (c *ddlChecker) getStatementTextWithSemicolon(ctx antlr.ParserRuleContext) string {
	text := c.tokens.GetTextFromRuleContext(ctx)

	// Check if the next token after the context is a semicolon
	stopIndex := ctx.GetStop().GetTokenIndex()
	if stopIndex+1 < c.tokens.Size() {
		nextToken := c.tokens.Get(stopIndex + 1)
		if nextToken.GetText() == ";" {
			text += ";"
		}
	}

	return text
}

func (c *ddlChecker) EnterEveryRule(ctx antlr.ParserRuleContext) {
	// Check if this is a top-level DDL statement
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if isDDLStatement(ctx) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Title:   c.title,
			Content: fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", c.getStatementTextWithSemicolon(ctx)),
			Code:    advisor.BuiltinPriorBackupCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: int32(ctx.GetStart().GetColumn()),
			},
		})
	}
}

// StatementType represents the type of DML statement
type StatementType int

const (
	StatementTypeUnknown StatementType = iota
	StatementTypeUpdate
	StatementTypeInsert
	StatementTypeDelete
)

type TableReference struct {
	Database      string
	Schema        string
	Table         string
	Alias         string
	StatementType StatementType
}

type statementInfo struct {
	offset    int
	statement string
	tree      antlr.ParserRuleContext
	table     *TableReference
}

func prepareTransformation(statement string) ([]statementInfo, error) {
	tree, err := pg.ParsePostgreSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	extractor := &dmlExtractor{}
	antlr.ParseTreeWalkerDefault.Walk(extractor, tree.Tree)
	return extractor.dmls, nil
}

type dmlExtractor struct {
	*parser.BasePostgreSQLParserListener

	dmls   []statementInfo
	offset int
}

func isTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}

	switch ctx := ctx.(type) {
	case *parser.RootContext, *parser.StmtblockContext:
		return true
	case *parser.StmtmultiContext, *parser.StmtContext:
		return isTopLevel(ctx.GetParent())
	default:
		return false
	}
}

func (e *dmlExtractor) ExitStmt(ctx *parser.StmtContext) {
	if isTopLevel(ctx) {
		e.offset++
	}
}

func (e *dmlExtractor) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		table := extractTableReference(ctx.Relation_expr_opt_alias())
		if table == nil {
			return
		}
		table.StatementType = StatementTypeUpdate
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
		})
	}
}

func (e *dmlExtractor) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if isTopLevel(ctx.GetParent()) {
		table := extractTableReference(ctx.Relation_expr_opt_alias())
		if table == nil {
			return
		}
		table.StatementType = StatementTypeDelete
		e.dmls = append(e.dmls, statementInfo{
			offset:    e.offset,
			statement: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx),
			tree:      ctx,
			table:     table,
		})
	}
}

func extractTableReference(ctx parser.IRelation_expr_opt_aliasContext) *TableReference {
	if ctx == nil {
		return nil
	}

	table := TableReference{}

	relationExpr := ctx.Relation_expr()
	if relationExpr == nil {
		return nil
	}

	list := pg.NormalizePostgreSQLQualifiedName(relationExpr.Qualified_name())
	switch len(list) {
	case 3:
		table.Database = list[0]
		table.Schema = list[1]
		table.Table = list[2]
	case 2:
		table.Schema = list[0]
		table.Table = list[1]
	case 1:
		table.Schema = defaultSchema
		table.Table = list[0]
	default:
		slog.Debug("Invalid table name", log.BBError(errors.Errorf("Invalid table name: %v", list)))
		return nil
	}

	if ctx.Colid() != nil {
		table.Alias = pg.NormalizePostgreSQLColid(ctx.Colid())
	}
	return &table
}
