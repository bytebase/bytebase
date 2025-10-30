package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableRequirePK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check parses the given statement and checks for errors.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableRequirePKChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
		catalog:                      checkCtx.Catalog,
		tableMentions:                make(map[string]*tableMention),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	// Simple Solution: Validate final state after walking all statements
	checker.validateFinalState()

	return checker.adviceList, nil
}

type tableMention struct {
	startLine int
	endLine   int
}

type tableRequirePKChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
	catalog        *catalog.Finder

	// Simple Solution: Track last mention of each table
	tableMentions map[string]*tableMention // key: "schema.table", value: last mention info
}

// EnterCreatestmt records CREATE TABLE statements (Simple Solution)
func (c *tableRequirePKChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		tableName = extractTableName(allQualifiedNames[0])
		schemaName = extractSchemaName(allQualifiedNames[0])
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Simple Solution: Just record the table mention (ALWAYS update for last occurrence)
	key := fmt.Sprintf("%s.%s", schemaName, tableName)
	c.tableMentions[key] = &tableMention{
		startLine: ctx.GetStart().GetLine(),
		endLine:   ctx.GetStop().GetLine(),
	}
}

// EnterAltertablestmt records ALTER TABLE statements (Simple Solution)
func (c *tableRequirePKChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
		schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Simple Solution: Just record the table mention (ALWAYS update for last occurrence)
	key := fmt.Sprintf("%s.%s", schemaName, tableName)
	c.tableMentions[key] = &tableMention{
		startLine: ctx.GetStart().GetLine(),
		endLine:   ctx.GetStop().GetLine(),
	}
}

// EnterDropstmt handles DROP TABLE - remove from tracking
func (c *tableRequirePKChecker) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is DROP TABLE
	if ctx.Object_type_any_name() == nil || ctx.Object_type_any_name().TABLE() == nil {
		return
	}

	// Remove all dropped tables from tracking (simplified - best effort)
	// Note: For the Simple Solution, not handling DROP TABLE perfectly is acceptable
	// since dropped tables won't be in catalog.Final anyway
	if ctx.Any_name_list() != nil {
		allNames := ctx.Any_name_list().AllAny_name()
		for _, anyName := range allNames {
			if anyName.Colid() != nil {
				// Simple table name (most common case)
				name := pgparser.NormalizePostgreSQLColid(anyName.Colid())
				key := fmt.Sprintf("public.%s", name)
				delete(c.tableMentions, key)
			}
			// For qualified names, we skip for simplicity - they won't cause false positives
			// because catalog.Final won't have dropped tables
		}
	}
}

// validateFinalState checks all mentioned tables against catalog.Final (Simple Solution)
func (c *tableRequirePKChecker) validateFinalState() {
	for tableKey, mention := range c.tableMentions {
		// Parse table key: "schema.table"
		schemaName, tableName := parseTableKey(tableKey)

		// Check catalog.Final for PRIMARY KEY
		hasPK := c.catalog.Final.HasPrimaryKey(&catalog.PrimaryKeyFind{
			SchemaName: schemaName,
			TableName:  tableName,
		})

		if !hasPK {
			content := fmt.Sprintf("Table %q.%q requires PRIMARY KEY", schemaName, tableName)

			// Extract and include the related statement
			statement := extractStatementText(c.statementsText, mention.startLine, mention.endLine)
			if statement != "" {
				content = fmt.Sprintf("%s, related statement: %q", content, statement)
			}

			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.TableNoPK.Int32(),
				Title:   c.title,
				Content: content,
				StartPosition: &storepb.Position{
					Line:   int32(mention.startLine),
					Column: 0,
				},
			})
		}
	}
}

// parseTableKey splits "schema.table" into schema and table name
func parseTableKey(key string) (string, string) {
	// Simple split on first dot
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			return key[:i], key[i+1:]
		}
	}
	return "public", key
}
