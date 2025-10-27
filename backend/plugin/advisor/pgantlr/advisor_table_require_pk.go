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
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type tableRequirePKChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
	catalog        *catalog.Finder
}

// EnterCreatestmt handles CREATE TABLE - must have PK
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

	hasPK := false

	// Check table-level constraints
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Check if this is a table constraint with PRIMARY KEY
			if elem.Tableconstraint() != nil {
				constraint := elem.Tableconstraint()
				if constraint.Constraintelem() != nil {
					constraintElem := constraint.Constraintelem()
					// Check for PRIMARY KEY or PRIMARY KEY USING INDEX
					if constraintElem.PRIMARY() != nil && constraintElem.KEY() != nil {
						hasPK = true
						break
					}
				}
			}

			// Check column-level constraints
			if elem.ColumnDef() != nil {
				columnDef := elem.ColumnDef()
				if columnDef.Colquallist() != nil {
					allQuals := columnDef.Colquallist().AllColconstraint()
					for _, qual := range allQuals {
						if qual.Colconstraintelem() != nil {
							constraintElem := qual.Colconstraintelem()
							// Check for PRIMARY KEY
							if constraintElem.PRIMARY() != nil && constraintElem.KEY() != nil {
								hasPK = true
								break
							}
						}
					}
					if hasPK {
						break
					}
				}
			}
		}
	}

	if !hasPK {
		c.addMissingPKAdvice(schemaName, tableName, ctx)
	}
}

// EnterAltertablestmt handles ALTER TABLE DROP CONSTRAINT / DROP COLUMN
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

	// Check all alter table commands
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// DROP CONSTRAINT
			if cmd.DROP() != nil && cmd.CONSTRAINT() != nil {
				allColIDs := cmd.AllColid()
				if len(allColIDs) > 0 {
					constraintName := pgparser.NormalizePostgreSQLColid(allColIDs[0])
					// Check if this constraint is a primary key using catalog
					if c.catalog != nil {
						_, index := c.catalog.Origin.FindIndex(&catalog.IndexFind{
							SchemaName: schemaName,
							TableName:  tableName,
							IndexName:  constraintName,
						})
						if index != nil && index.Primary() {
							c.addMissingPKAdvice(schemaName, tableName, ctx)
							return
						}
					}
				}
			}

			// DROP COLUMN (check for COLUMN keyword to distinguish from DROP CONSTRAINT)
			if cmd.DROP() != nil && cmd.Opt_column() != nil {
				allColIDs := cmd.AllColid()
				if len(allColIDs) > 0 && cmd.CONSTRAINT() == nil {
					columnName := pgparser.NormalizePostgreSQLColid(allColIDs[0])
					// Check if this column is part of the primary key using catalog
					if c.catalog != nil {
						pk := c.catalog.Origin.FindPrimaryKey(&catalog.PrimaryKeyFind{
							SchemaName: schemaName,
							TableName:  tableName,
						})
						if pk != nil {
							for _, column := range pk.ExpressionList() {
								if column == columnName {
									c.addMissingPKAdvice(schemaName, tableName, ctx)
									return
								}
							}
						}
					}
				}
			}
		}
	}
}

func (c *tableRequirePKChecker) addMissingPKAdvice(schemaName, tableName string, ctx antlr.ParserRuleContext) {
	stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.TableNoPK.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf("Table %q.%q requires PRIMARY KEY, related statement: %q", schemaName, tableName, stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
