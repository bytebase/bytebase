package pg

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

	rule := &tableRequirePKRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		statementsText: checkCtx.Statements,
		tableState:     make(map[string]*tableState),
		catalog:        checkCtx.Catalog,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	// Validate final state after walking all statements
	rule.validateFinalState()

	return checker.GetAdviceList(), nil
}

type tableState struct {
	hasPK        bool
	pkConstraint string // Name of the PK constraint (if known)
	startLine    int
	endLine      int
}

type tableRequirePKRule struct {
	BaseRule
	statementsText string
	catalog        *catalog.Finder

	// Track table state: whether they have a primary key
	tableState map[string]*tableState // key: "schema.table", value: state
}

// Name returns the rule name.
func (*tableRequirePKRule) Name() string {
	return "table.require-pk"
}

// OnEnter is called when the parser enters a rule context.
func (r *tableRequirePKRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	case "Dropstmt":
		r.handleDropstmt(ctx.(*parser.DropstmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*tableRequirePKRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt records CREATE TABLE statements
func (r *tableRequirePKRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
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

	key := makeTableKey(schemaName, tableName)

	// Check if this CREATE TABLE has a PRIMARY KEY
	hasPK := false
	pkConstraint := ""
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Check column-level PRIMARY KEY
			if elem.ColumnDef() != nil && hasPrimaryKeyConstraint(elem.ColumnDef()) {
				hasPK = true
				// PostgreSQL generates constraint name as tablename_pkey
				pkConstraint = getDefaultPKConstraintName(tableName)
				break
			}

			// Check table-level PRIMARY KEY
			if elem.Tableconstraint() != nil && isTablePrimaryKey(elem.Tableconstraint()) {
				hasPK = true
				// Try to get explicit constraint name, or use default
				if elem.Tableconstraint().Name() != nil && elem.Tableconstraint().Name().Colid() != nil {
					pkConstraint = pgparser.NormalizePostgreSQLColid(elem.Tableconstraint().Name().Colid())
				} else {
					pkConstraint = getDefaultPKConstraintName(tableName)
				}
				break
			}
		}
	}

	r.tableState[key] = &tableState{
		hasPK:        hasPK,
		pkConstraint: pkConstraint,
		startLine:    ctx.GetStart().GetLine(),
		endLine:      ctx.GetStop().GetLine(),
	}
}

// handleAltertablestmt records ALTER TABLE statements
func (r *tableRequirePKRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
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

	key := makeTableKey(schemaName, tableName)

	// Check if we're dropping a PRIMARY KEY (either via DROP CONSTRAINT or DROP COLUMN)
	droppedPK := false
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			if r.isDropPKConstraint(cmd, key, schemaName, tableName) {
				droppedPK = true
				break
			}
			if r.isDropPKColumn(cmd, schemaName, tableName) {
				droppedPK = true
				break
			}
		}
	}

	// Check if we're already tracking this table (from CREATE TABLE)
	// or if we just detected we're dropping the PK
	if r.tableState[key] == nil && !droppedPK {
		// Not tracking this table and not dropping PK, so nothing to do
		return
	}

	// Track that we're modifying this table
	if r.tableState[key] == nil {
		r.tableState[key] = &tableState{
			hasPK:     false,
			startLine: ctx.GetStart().GetLine(),
			endLine:   ctx.GetStop().GetLine(),
		}
	}

	// If we dropped the PK, mark it and update the line to point to this ALTER statement
	if droppedPK {
		r.tableState[key].hasPK = false
		r.tableState[key].startLine = ctx.GetStart().GetLine()
	}

	// Check if ALTER TABLE adds a PRIMARY KEY
	addedPK := false
	addedPKConstraint := ""
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD COLUMN with PRIMARY KEY
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				if hasPrimaryKeyConstraint(cmd.ColumnDef()) {
					addedPK = true
					addedPKConstraint = tableName + "_pkey"
					break
				}
			}

			// ADD CONSTRAINT PRIMARY KEY
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				if isTablePrimaryKey(cmd.Tableconstraint()) {
					addedPK = true
					// Try to get explicit constraint name, or use default
					if cmd.Tableconstraint().Name() != nil && cmd.Tableconstraint().Name().Colid() != nil {
						addedPKConstraint = pgparser.NormalizePostgreSQLColid(cmd.Tableconstraint().Name().Colid())
					} else {
						addedPKConstraint = tableName + "_pkey"
					}
					break
				}
			}
		}
	}

	if addedPK {
		r.tableState[key].hasPK = true
		r.tableState[key].pkConstraint = addedPKConstraint
	}

	// Update end line
	r.tableState[key].endLine = ctx.GetStop().GetLine()
}

// handleDropstmt handles DROP TABLE - remove from tracking
func (r *tableRequirePKRule) handleDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is DROP TABLE
	if ctx.Object_type_any_name() == nil || ctx.Object_type_any_name().TABLE() == nil {
		return
	}

	// Remove all dropped tables from tracking
	if ctx.Any_name_list() != nil {
		allNames := ctx.Any_name_list().AllAny_name()
		for _, anyName := range allNames {
			if anyName.Colid() != nil {
				// Simple table name (most common case)
				name := pgparser.NormalizePostgreSQLColid(anyName.Colid())
				key := fmt.Sprintf("public.%s", name)
				delete(r.tableState, key)
			}
			// For qualified names, we skip for simplicity
		}
	}
}

// validateFinalState checks all tables for PRIMARY KEY
func (r *tableRequirePKRule) validateFinalState() {
	for tableKey, state := range r.tableState {
		// Parse table key: "schema.table"
		schemaName, tableName := parseTableKey(tableKey)

		// Only report error if we didn't add a PK in the statements
		// (Tables we CREATE without PK, or tables we ALTER without adding PK are reported)
		if !state.hasPK {
			content := fmt.Sprintf("Table %q.%q requires PRIMARY KEY", schemaName, tableName)

			// Extract and include the related statement
			statement := extractStatementText(r.statementsText, state.startLine, state.endLine)
			if statement != "" {
				content = fmt.Sprintf("%s, related statement: %q", content, statement)
			}

			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    advisor.TableNoPK.Int32(),
				Title:   r.title,
				Content: content,
				StartPosition: &storepb.Position{
					Line:   int32(state.startLine),
					Column: 0,
				},
			})
		}
	}
}

// getConstraintName extracts the constraint name from an ALTER TABLE command.
// Returns empty string if no constraint name found.
func getConstraintName(cmd parser.IAlter_table_cmdContext) string {
	if cmd.Name() != nil && cmd.Name().Colid() != nil {
		return pgparser.NormalizePostgreSQLColid(cmd.Name().Colid())
	}
	allColids := cmd.AllColid()
	if len(allColids) > 0 {
		return pgparser.NormalizePostgreSQLColid(allColids[0])
	}
	return ""
}

// isPKConstraintInCatalog checks if the given constraint name is a primary key in catalog.Origin.
func (r *tableRequirePKRule) isPKConstraintInCatalog(schemaName, tableName, constraintName string) bool {
	tableInfo := r.catalog.Origin.FindTable(&catalog.TableFind{
		SchemaName: schemaName,
		TableName:  tableName,
	})
	if tableInfo == nil {
		return false
	}

	indexSet := tableInfo.Index(nil)
	if indexSet == nil {
		return false
	}

	for indexName, indexState := range *indexSet {
		if indexState.Primary() && indexName == constraintName {
			return true
		}
	}
	return false
}

// isColumnInPK checks if a column is part of the primary key in catalog.Origin.
func (r *tableRequirePKRule) isColumnInPK(schemaName, tableName, columnName string) bool {
	tableInfo := r.catalog.Origin.FindTable(&catalog.TableFind{
		SchemaName: schemaName,
		TableName:  tableName,
	})
	if tableInfo == nil {
		return false
	}

	indexSet := tableInfo.Index(nil)
	if indexSet == nil {
		return false
	}

	for _, indexState := range *indexSet {
		if indexState.Primary() {
			for _, expr := range indexState.ExpressionList() {
				if expr == columnName {
					return true
				}
			}
		}
	}
	return false
}

// isDropPKConstraint checks if the ALTER TABLE command drops a primary key constraint.
func (r *tableRequirePKRule) isDropPKConstraint(cmd parser.IAlter_table_cmdContext, key, schemaName, tableName string) bool {
	if cmd.DROP() == nil || cmd.CONSTRAINT() == nil {
		return false
	}

	constraintName := getConstraintName(cmd)
	if constraintName == "" {
		return false
	}

	// First check if this table is being tracked and has a known PK constraint
	if r.tableState[key] != nil && r.tableState[key].pkConstraint == constraintName {
		return true
	}

	// Check if this constraint is the primary key in catalog.Origin
	return r.isPKConstraintInCatalog(schemaName, tableName, constraintName)
}

// isDropPKColumn checks if the ALTER TABLE command drops a column that's part of the primary key.
func (r *tableRequirePKRule) isDropPKColumn(cmd parser.IAlter_table_cmdContext, schemaName, tableName string) bool {
	if cmd.DROP() == nil || cmd.CONSTRAINT() != nil {
		return false
	}

	allColids := cmd.AllColid()
	if len(allColids) == 0 {
		return false
	}

	columnName := pgparser.NormalizePostgreSQLColid(allColids[0])
	return r.isColumnInPK(schemaName, tableName, columnName)
}

// hasPrimaryKeyConstraint checks if a column definition has PRIMARY KEY constraint
func hasPrimaryKeyConstraint(colDef parser.IColumnDefContext) bool {
	if colDef.Colquallist() == nil {
		return false
	}

	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		if constraint.Colconstraintelem() != nil {
			elem := constraint.Colconstraintelem()
			// PRIMARY KEY
			if elem.PRIMARY() != nil && elem.KEY() != nil {
				return true
			}
		}
	}

	return false
}

// isTablePrimaryKey checks if a table constraint is PRIMARY KEY
func isTablePrimaryKey(constraint parser.ITableconstraintContext) bool {
	if constraint == nil || constraint.Constraintelem() == nil {
		return false
	}

	elem := constraint.Constraintelem()

	// PRIMARY KEY
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		return true
	}

	return false
}
