package trino

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_TRINO, extractChangedResources)
}

// extractChangedResources extracts the changed resources from a Trino SQL statement.
func extractChangedResources(currentDatabase string, currentSchema string, dbSchema *model.DatabaseSchema, ast any, statement string) (*base.ChangeSummary, error) {
	// Create a new ChangedResources to track affected tables, columns, etc.
	changedResources := model.NewChangedResources(dbSchema)

	// Parse the SQL statement if not already parsed
	var parseResult *ParseResult
	var ok bool
	var err error

	if ast != nil {
		parseResult, ok = ast.(*ParseResult)
		if !ok {
			// Parse the SQL statement if the AST isn't a ParseResult
			parseResult, err = ParseTrino(statement)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Parse the SQL statement if no AST is provided
		parseResult, err = ParseTrino(statement)
		if err != nil {
			return nil, err
		}
	}

	// Create a listener to track changed resources
	listener := &changedResourceListener{
		currentDatabase:  currentDatabase,
		currentSchema:    currentSchema,
		changedResources: changedResources,
		statement:        statement,
		dmlCount:         0,
		insertCount:      0,
		sampleDMLs:       []string{},
	}

	// Walk the parse tree with our listener
	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)

	// Create a ChangeSummary with our collected data
	changeSummary := &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         listener.dmlCount,
		InsertCount:      listener.insertCount,
		SampleDMLS:       listener.sampleDMLs,
	}

	return changeSummary, nil
}

// changedResourceListener implements the TrinoParserListener interface to track changed resources.
type changedResourceListener struct {
	parser.BaseTrinoParserListener

	currentDatabase  string
	currentSchema    string
	changedResources *model.ChangedResources
	statement        string
	dmlCount         int
	insertCount      int
	sampleDMLs       []string

	// Track the current statement context
	isCreate bool
	isAlter  bool
	isDrop   bool
	isInsert bool
	isUpdate bool
	isDelete bool

	// The maximum number of sample DMLs to collect
	maxSampleDMLs int
}

// EnterSingleStatement is called when entering a singleStatement production
func (l *changedResourceListener) EnterSingleStatement(_ *parser.SingleStatementContext) {
	l.maxSampleDMLs = 5 // Set the maximum number of sample DMLs to collect
}

// EnterCreateTable is called when the parser enters a createTable rule
func (l *changedResourceListener) EnterCreateTable(ctx *parser.CreateTableContext) {
	l.isCreate = true

	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table creation in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)
}

// EnterDropTable is called when the parser enters a dropTable rule
func (l *changedResourceListener) EnterDropTable(ctx *parser.DropTableContext) {
	l.isDrop = true

	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table drop in changedResources - set the last param to true for drop operations
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, true)
}

// EnterCreateView is called when the parser enters a createView rule
func (l *changedResourceListener) EnterCreateView(ctx *parser.CreateViewContext) {
	l.isCreate = true

	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and view name
	catalog, schema, view := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the view creation in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	viewResource := &storepb.ChangedResourceView{
		Name:   view,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddView(catalog, schema, viewResource)
}

// EnterDropView is called when the parser enters a dropView rule
func (l *changedResourceListener) EnterDropView(ctx *parser.DropViewContext) {
	l.isDrop = true

	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and view name
	catalog, schema, view := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the view drop in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	viewResource := &storepb.ChangedResourceView{
		Name:   view,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddView(catalog, schema, viewResource)
}

// EnterCreateSchema is called when the parser enters a createSchema rule
func (l *changedResourceListener) EnterCreateSchema(ctx *parser.CreateSchemaContext) {
	l.isCreate = true

	if ctx.QualifiedName() == nil {
		return
	}

	parts := ExtractQualifiedNameParts(ctx.QualifiedName())
	if len(parts) == 0 {
		return
	}

	// In Trino, schema can be specified as catalog.schema or just schema
	catalog := l.currentDatabase
	schema := parts[0]

	if len(parts) >= 2 {
		catalog = parts[0]
		schema = parts[1]
	}

	// Track the schema creation in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	// Since there's no explicit AddSchema method for the new ChangedResources struct,
	// we'll create a dummy table in that schema to ensure the schema is created
	tableResource := &storepb.ChangedResourceTable{
		Name:   "__schema_change__",
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)
}

// EnterDropSchema is called when the parser enters a dropSchema rule
func (l *changedResourceListener) EnterDropSchema(ctx *parser.DropSchemaContext) {
	l.isDrop = true

	if ctx.QualifiedName() == nil {
		return
	}

	parts := ExtractQualifiedNameParts(ctx.QualifiedName())
	if len(parts) == 0 {
		return
	}

	// In Trino, schema can be specified as catalog.schema or just schema
	catalog := l.currentDatabase
	schema := parts[0]

	if len(parts) >= 2 {
		catalog = parts[0]
		schema = parts[1]
	}

	// Track the schema drop in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	// Since there's no explicit AddSchema method for the new ChangedResources struct,
	// we'll create a dummy table in that schema to ensure the schema is recognized
	tableResource := &storepb.ChangedResourceTable{
		Name:   "__schema_change__",
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, true)
}

// EnterRenameTable handles rename table statements
func (l *changedResourceListener) EnterRenameTable(ctx *parser.RenameTableContext) {
	l.isAlter = true

	if len(ctx.AllQualifiedName()) < 1 || ctx.AllQualifiedName()[0] == nil {
		return
	}

	// Extract catalog, schema, and table name for the source table
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.AllQualifiedName()[0],
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table alteration in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)
}

// EnterRenameColumn handles rename column statements
func (l *changedResourceListener) EnterRenameColumn(ctx *parser.RenameColumnContext) {
	l.isAlter = true

	if len(ctx.AllQualifiedName()) < 1 || ctx.AllQualifiedName()[0] == nil {
		return
	}

	// Extract catalog, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.AllQualifiedName()[0],
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table alteration in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)
}

// EnterDropColumn handles drop column statements
func (l *changedResourceListener) EnterDropColumn(ctx *parser.DropColumnContext) {
	l.isAlter = true

	if len(ctx.AllQualifiedName()) < 1 || ctx.AllQualifiedName()[0] == nil {
		return
	}

	// Extract catalog, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.AllQualifiedName()[0],
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table alteration in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)
}

// EnterAddColumn handles add column statements
func (l *changedResourceListener) EnterAddColumn(ctx *parser.AddColumnContext) {
	l.isAlter = true

	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table alteration in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)
}

// EnterInsertInto is called when the parser enters an insertInto rule
func (l *changedResourceListener) EnterInsertInto(ctx *parser.InsertIntoContext) {
	l.isInsert = true
	l.dmlCount++
	l.insertCount++

	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table insertion in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)

	// Add to sample DMLs if we haven't reached the maximum
	if len(l.sampleDMLs) < l.maxSampleDMLs {
		l.sampleDMLs = append(l.sampleDMLs, l.statement)
	}
}

// EnterDelete is called when the parser enters a delete rule
func (l *changedResourceListener) EnterDelete(ctx *parser.DeleteContext) {
	l.isDelete = true
	l.dmlCount++

	// Trino's DELETE syntax: DELETE FROM table [WHERE condition]
	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table deletion in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)

	// Add to sample DMLs if we haven't reached the maximum
	if len(l.sampleDMLs) < l.maxSampleDMLs {
		l.sampleDMLs = append(l.sampleDMLs, l.statement)
	}
}

// EnterUpdate is called when the parser enters an update rule
func (l *changedResourceListener) EnterUpdate(ctx *parser.UpdateContext) {
	l.isUpdate = true
	l.dmlCount++

	// Trino's UPDATE syntax: UPDATE table SET col = expr [WHERE condition]
	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	// Track the table update in changedResources
	rng := base.NewRange(l.statement, ctx.GetText())

	tableResource := &storepb.ChangedResourceTable{
		Name:   table,
		Ranges: []*storepb.Range{rng},
	}

	l.changedResources.AddTable(catalog, schema, tableResource, false)

	// Add to sample DMLs if we haven't reached the maximum
	if len(l.sampleDMLs) < l.maxSampleDMLs {
		l.sampleDMLs = append(l.sampleDMLs, l.statement)
	}
}
