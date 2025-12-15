package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_POSTGRES, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_COCKROACHDB, extractChangedResources)
}

func extractChangedResources(database string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, statement string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)
	searchPath := dbMetadata.GetSearchPath()
	if len(searchPath) == 0 {
		searchPath = []string{"public"} // default search path for PostgreSQL
	}

	// If no parse results, return empty summary
	if len(asts) == 0 {
		return &base.ChangeSummary{
			ChangedResources: changedResources,
			DMLCount:         0,
			SampleDMLS:       []string{},
			InsertCount:      0,
		}, nil
	}

	listener := &changedResourcesListener{
		database:         database,
		searchPath:       searchPath,
		changedResources: changedResources,
		databaseMetadata: dbMetadata,
		statement:        statement,
		dmlCount:         0,
		insertCount:      0,
		sampleDMLs:       []string{},
	}

	// Walk all parse results to extract changed resources
	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for PostgreSQL")
		}
		if antlrAST.Tree == nil {
			return nil, errors.New("ANTLR tree is nil")
		}

		listener.tokenStream = antlrAST.Tokens
		antlr.ParseTreeWalkerDefault.Walk(listener, antlrAST.Tree)
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         listener.dmlCount,
		SampleDMLS:       listener.sampleDMLs,
		InsertCount:      listener.insertCount,
	}, nil
}

type changedResourcesListener struct {
	*parser.BasePostgreSQLParserListener

	database         string
	searchPath       []string
	changedResources *model.ChangedResources
	databaseMetadata *model.DatabaseMetadata
	statement        string
	tokenStream      antlr.TokenStream

	// DML statistics
	dmlCount    int
	insertCount int
	sampleDMLs  []string
}

// EnterVariablesetstmt handles SET search_path statements
func (l *changedResourcesListener) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	setRest := ctx.Set_rest()
	if setRest == nil {
		return
	}
	setRestMore := setRest.Set_rest_more()
	if setRestMore == nil {
		return
	}
	genericSet := setRestMore.Generic_set()
	if genericSet == nil {
		return
	}
	varName := genericSet.Var_name()
	if varName == nil {
		return
	}
	if len(varName.AllColid()) != 1 {
		return
	}
	colid := varName.Colid(0)
	if colid == nil {
		return
	}
	name := colid.GetText()
	if !strings.EqualFold(name, "search_path") {
		return
	}

	// Extract search path values
	varList := genericSet.Var_list()
	if varList == nil {
		return
	}
	values := varList.AllVar_value()
	if len(values) == 0 {
		return
	}

	var newSearchPath []string
	for _, value := range values {
		text := value.GetText()
		// Remove quotes if present
		text = strings.Trim(text, "'\"")
		if text != "" {
			newSearchPath = append(newSearchPath, text)
		}
	}

	if len(newSearchPath) > 0 {
		l.searchPath = newSearchPath
	}
}

// EnterCreatestmt handles CREATE TABLE statements
func (l *changedResourcesListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get qualified name
	qualifiedName := ctx.Qualified_name(0)
	if qualifiedName == nil {
		return
	}

	db, schema, table := l.extractQualifiedName(qualifiedName)
	if db == "" {
		db = l.database
	}
	if schema == "" {
		schema = l.searchPath[0]
	}

	l.changedResources.AddTable(
		db,
		schema,
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)
}

// EnterDropstmt handles DROP TABLE/VIEW/INDEX statements
func (l *changedResourcesListener) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check object type
	objectTypeAnyName := ctx.Object_type_any_name()
	if objectTypeAnyName == nil {
		return
	}

	// Handle DROP INDEX separately
	if objectTypeAnyName.INDEX() != nil {
		l.handleDropIndex(ctx)
		return
	}

	// Handle DROP TABLE or DROP VIEW
	if objectTypeAnyName.TABLE() == nil && objectTypeAnyName.VIEW() == nil {
		return
	}

	// Get the list of tables/views to drop
	anyNameList := ctx.Any_name_list()
	if anyNameList == nil {
		return
	}

	isView := objectTypeAnyName.VIEW() != nil
	// Skip views since they're not used in risk/approval calculations
	if isView {
		return
	}

	for _, anyName := range anyNameList.AllAny_name() {
		db, schema, name := l.extractAnyName(anyName)
		if db == "" {
			db = l.database
		}
		if schema == "" {
			schemaName, _ := l.databaseMetadata.SearchObject(l.searchPath, name)
			if schemaName == "" {
				schema = l.searchPath[0]
			} else {
				schema = schemaName
			}
		}

		// For tables
		l.changedResources.AddTable(
			db,
			schema,
			&storepb.ChangedResourceTable{
				Name: name,
			},
			true,
		)
	}
}

// EnterAltertablestmt handles ALTER TABLE statements
func (l *changedResourcesListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Skip if this is ALTER VIEW
	if ctx.VIEW() != nil {
		return
	}

	// Get table name
	relationExpr := ctx.Relation_expr()
	if relationExpr == nil {
		return
	}

	db, schema, table := l.extractRelationExpr(relationExpr)
	if db == "" {
		db = l.database
	}
	if schema == "" {
		schemaName, _ := l.databaseMetadata.SearchObject(l.searchPath, table)
		if schemaName == "" {
			schema = l.searchPath[0]
		} else {
			schema = schemaName
		}
	}

	l.changedResources.AddTable(
		db,
		schema,
		&storepb.ChangedResourceTable{
			Name: table,
		},
		true,
	)
}

// EnterRenamestmt handles ALTER TABLE...RENAME TO statements
func (l *changedResourcesListener) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a table rename (ALTER TABLE t1 RENAME TO t2)
	relationExpr := ctx.Relation_expr()
	if relationExpr == nil {
		return
	}

	// Get the new table name
	allNames := ctx.AllName()
	if len(allNames) == 0 {
		return
	}
	newTableName := allNames[0].GetText()

	// Extract old table name from relation_expr
	db, schema, oldTableName := l.extractRelationExpr(relationExpr)
	if db == "" {
		db = l.database
	}
	if schema == "" {
		schemaName, _ := l.databaseMetadata.SearchObject(l.searchPath, oldTableName)
		if schemaName == "" {
			schema = l.searchPath[0]
		} else {
			schema = schemaName
		}
	}

	// Add the old table name (being renamed from)
	l.changedResources.AddTable(
		db,
		schema,
		&storepb.ChangedResourceTable{
			Name: oldTableName,
		},
		true,
	)

	// Add the new table name (being renamed to)
	l.changedResources.AddTable(
		db,
		schema,
		&storepb.ChangedResourceTable{
			Name: newTableName,
		},
		false,
	)
}

// EnterIndexstmt handles CREATE INDEX statements
func (l *changedResourcesListener) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get table name
	relationExpr := ctx.Relation_expr()
	if relationExpr == nil {
		return
	}

	db, schema, table := l.extractRelationExpr(relationExpr)
	if db == "" {
		db = l.database
	}
	if schema == "" {
		schemaName, _ := l.databaseMetadata.SearchObject(l.searchPath, table)
		if schemaName == "" {
			schema = l.searchPath[0]
		} else {
			schema = schemaName
		}
	}

	l.changedResources.AddTable(
		db,
		schema,
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)
}

// handleDropIndex handles DROP INDEX statements
func (l *changedResourcesListener) handleDropIndex(ctx *parser.DropstmtContext) {
	// Get index names
	anyNameList := ctx.Any_name_list()
	if anyNameList == nil {
		return
	}

	for _, anyName := range anyNameList.AllAny_name() {
		db, schema, indexName := l.extractAnyName(anyName)
		if db == "" {
			db = l.database
		}

		// If schema is specified, try to find the table
		if schema != "" {
			// Search for the index in metadata to find its table
			schemaName, indexMetadata := l.databaseMetadata.SearchIndex([]string{schema}, indexName)
			if indexMetadata != nil && schemaName != "" {
				tableProto := indexMetadata.GetTableProto()
				if tableProto != nil {
					l.changedResources.AddTable(
						db,
						schemaName,
						&storepb.ChangedResourceTable{
							Name: tableProto.GetName(),
						},
						false,
					)
				}
			}
		} else {
			// Search in search_path
			schemaName, indexMetadata := l.databaseMetadata.SearchIndex(l.searchPath, indexName)
			if indexMetadata != nil && schemaName != "" {
				tableProto := indexMetadata.GetTableProto()
				if tableProto != nil {
					l.changedResources.AddTable(
						db,
						schemaName,
						&storepb.ChangedResourceTable{
							Name: tableProto.GetName(),
						},
						false,
					)
				}
			}
		}
	}
}

// EnterInsertstmt handles INSERT statements
func (l *changedResourcesListener) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get table name
	insertTarget := ctx.Insert_target()
	if insertTarget == nil {
		return
	}
	qualifiedName := insertTarget.Qualified_name()
	if qualifiedName == nil {
		return
	}

	db, schema, table := l.extractQualifiedName(qualifiedName)
	if db == "" {
		db = l.database
	}
	if schema == "" {
		schemaName, _ := l.databaseMetadata.SearchObject(l.searchPath, table)
		if schemaName == "" {
			schema = l.searchPath[0]
		} else {
			schema = schemaName
		}
	}

	l.changedResources.AddTable(
		db,
		schema,
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)

	// Count insert rows
	insertRest := ctx.Insert_rest()
	if insertRest != nil {
		selectStmt := insertRest.Selectstmt()
		if selectStmt != nil {
			// Try to count VALUES clauses
			rowCount := l.countInsertRows(selectStmt)
			if rowCount > 0 {
				l.insertCount += rowCount
			}
		}
	}
}

// EnterUpdatestmt handles UPDATE statements
func (l *changedResourcesListener) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get table name
	relationExprOptAlias := ctx.Relation_expr_opt_alias()
	if relationExprOptAlias == nil {
		return
	}
	relationExpr := relationExprOptAlias.Relation_expr()
	if relationExpr == nil {
		return
	}

	db, schema, table := l.extractRelationExpr(relationExpr)
	if db == "" {
		db = l.database
	}
	if schema == "" {
		schemaName, _ := l.databaseMetadata.SearchObject(l.searchPath, table)
		if schemaName == "" {
			schema = l.searchPath[0]
		} else {
			schema = schemaName
		}
	}

	l.changedResources.AddTable(
		db,
		schema,
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)

	// Count as DML
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		// Use getStatementText to preserve whitespace and include semicolon
		l.sampleDMLs = append(l.sampleDMLs, l.getStatementText(ctx))
	}
}

// EnterDeletestmt handles DELETE statements
func (l *changedResourcesListener) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get table name
	relationExprOptAlias := ctx.Relation_expr_opt_alias()
	if relationExprOptAlias == nil {
		return
	}
	relationExpr := relationExprOptAlias.Relation_expr()
	if relationExpr == nil {
		return
	}

	db, schema, table := l.extractRelationExpr(relationExpr)
	if db == "" {
		db = l.database
	}
	if schema == "" {
		schemaName, _ := l.databaseMetadata.SearchObject(l.searchPath, table)
		if schemaName == "" {
			schema = l.searchPath[0]
		} else {
			schema = schemaName
		}
	}

	l.changedResources.AddTable(
		db,
		schema,
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)

	// Count as DML
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		// Use getStatementText to preserve whitespace and include semicolon
		l.sampleDMLs = append(l.sampleDMLs, l.getStatementText(ctx))
	}
}

// Helper functions

func (*changedResourcesListener) extractQualifiedName(ctx parser.IQualified_nameContext) (string, string, string) {
	if ctx == nil {
		return "", "", ""
	}

	colid := ctx.Colid()
	if colid == nil {
		return "", "", ""
	}

	name := colid.GetText()

	// Check for indirection (schema.table or db.schema.table)
	indirection := ctx.Indirection()
	if indirection == nil {
		return "", "", name
	}

	indirectionEls := indirection.AllIndirection_el()
	if len(indirectionEls) == 0 {
		return "", "", name
	}

	// If there's one indirection, it's schema.table
	if len(indirectionEls) == 1 {
		attrName := indirectionEls[0].Attr_name()
		if attrName != nil {
			return "", name, attrName.GetText()
		}
	}

	// If there are two indirections, it's db.schema.table
	if len(indirectionEls) == 2 {
		schema := indirectionEls[0].Attr_name()
		table := indirectionEls[1].Attr_name()
		if schema != nil && table != nil {
			return name, schema.GetText(), table.GetText()
		}
	}

	return "", "", name
}

func (*changedResourcesListener) extractAnyName(ctx parser.IAny_nameContext) (string, string, string) {
	if ctx == nil {
		return "", "", ""
	}

	// Collect all parts: first from Colid(), then from Attrs()
	var parts []string
	if ctx.Colid() != nil {
		parts = append(parts, NormalizePostgreSQLColid(ctx.Colid()))
	}

	if ctx.Attrs() != nil {
		for _, attr := range ctx.Attrs().AllAttr_name() {
			// Use GetText() and normalize it - PostgreSQL identifiers are case-insensitive by default
			text := attr.GetText()
			// Remove quotes if present
			if (strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"")) || (strings.HasPrefix(text, "'") && strings.HasSuffix(text, "'")) {
				text = text[1 : len(text)-1]
			} else {
				text = strings.ToLower(text)
			}
			parts = append(parts, text)
		}
	}

	if len(parts) == 0 {
		return "", "", ""
	}

	// Simple name
	if len(parts) == 1 {
		return "", "", parts[0]
	}

	// schema.name
	if len(parts) == 2 {
		return "", parts[0], parts[1]
	}

	// db.schema.name
	if len(parts) >= 3 {
		return parts[0], parts[1], parts[2]
	}

	return "", "", ""
}

func (l *changedResourcesListener) extractRelationExpr(ctx parser.IRelation_exprContext) (string, string, string) {
	if ctx == nil {
		return "", "", ""
	}

	qualifiedName := ctx.Qualified_name()
	if qualifiedName == nil {
		return "", "", ""
	}

	return l.extractQualifiedName(qualifiedName)
}

// getStatementText returns the text of a statement, including trailing semicolon if present
func (l *changedResourcesListener) getStatementText(ctx antlr.ParserRuleContext) string {
	start := ctx.GetStart().GetTokenIndex()
	stop := ctx.GetStop().GetTokenIndex()

	// Check if the next token is a semicolon
	tokens, ok := l.tokenStream.(*antlr.CommonTokenStream)
	if ok {
		nextToken := tokens.Get(stop + 1)
		if nextToken != nil && nextToken.GetText() == ";" {
			// Include the semicolon
			return l.tokenStream.GetTextFromInterval(antlr.NewInterval(start, stop+1))
		}
	}

	// No semicolon or can't access token stream
	return l.tokenStream.GetTextFromInterval(antlr.NewInterval(start, stop))
}

func (*changedResourcesListener) countInsertRows(ctx parser.ISelectstmtContext) int {
	if ctx == nil {
		return 0
	}

	// Count VALUES clauses
	counter := &valuesRowCounter{count: 0}
	antlr.ParseTreeWalkerDefault.Walk(counter, ctx)
	return counter.count
}

// valuesRowCounter counts the number of rows in VALUES clauses
type valuesRowCounter struct {
	*parser.BasePostgreSQLParserListener
	count int
}

func (v *valuesRowCounter) EnterValues_clause(ctx *parser.Values_clauseContext) {
	// Count the number of expr_list (rows) in the VALUES clause
	if ctx == nil {
		return
	}

	// Each expr_list represents one row
	exprLists := ctx.AllExpr_list()
	v.count += len(exprLists)
}
