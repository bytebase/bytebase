package pg

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterGetSDLDiff(storepb.Engine_POSTGRES, GetSDLDiff)
	schema.RegisterGetSDLDiff(storepb.Engine_COCKROACHDB, GetSDLDiff)
}

func GetSDLDiff(currentSDLText, previousUserSDLText string, currentSchema, previousSchema *model.DatabaseSchema) (*schema.MetadataDiff, error) {
	// Check for initialization scenario: previousUserSDLText is empty
	if strings.TrimSpace(previousUserSDLText) == "" && currentSchema != nil {
		// Initialization scenario: convert currentSchema to SDL format as baseline
		generatedSDL, err := convertDatabaseSchemaToSDL(currentSchema)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert current schema to SDL format for initialization")
		}
		previousUserSDLText = generatedSDL
	}

	currentChunks, err := ChunkSDLText(currentSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk current SDL text")
	}

	previousChunks, err := ChunkSDLText(previousUserSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk previous SDL text")
	}

	// Check for drift scenario: when both schemas are provided, apply minimal changes to previousChunks
	if currentSchema != nil && previousSchema != nil {
		err = applyMinimalChangesToChunks(previousChunks, currentSchema, previousSchema)
		if err != nil {
			return nil, errors.Wrap(err, "failed to apply minimal changes to SDL chunks")
		}
	}

	// Pre-compute SDL chunks from current database metadata for performance optimization
	// This avoids repeated calls to convertDatabaseSchemaToSDL and ChunkSDLText during usability checks
	currentDBSDLChunks, err := buildCurrentDatabaseSDLChunks(currentSchema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build current database SDL chunks")
	}

	// Initialize MetadataDiff
	diff := &schema.MetadataDiff{
		DatabaseName:            "",
		SchemaChanges:           []*schema.SchemaDiff{},
		TableChanges:            []*schema.TableDiff{},
		ViewChanges:             []*schema.ViewDiff{},
		MaterializedViewChanges: []*schema.MaterializedViewDiff{},
		FunctionChanges:         []*schema.FunctionDiff{},
		ProcedureChanges:        []*schema.ProcedureDiff{},
		SequenceChanges:         []*schema.SequenceDiff{},
		EnumTypeChanges:         []*schema.EnumTypeDiff{},
	}

	// Process table changes
	err = processTableChanges(currentChunks, previousChunks, currentSchema, previousSchema, currentDBSDLChunks, diff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process table changes")
	}

	// Process index changes (standalone CREATE INDEX statements)
	processStandaloneIndexChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process view changes
	processViewChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process function changes
	processFunctionChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process sequence changes
	processSequenceChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	return diff, nil
}

func ChunkSDLText(sdlText string) (*schema.SDLChunks, error) {
	if strings.TrimSpace(sdlText) == "" {
		return &schema.SDLChunks{
			Tables:    make(map[string]*schema.SDLChunk),
			Views:     make(map[string]*schema.SDLChunk),
			Functions: make(map[string]*schema.SDLChunk),
			Indexes:   make(map[string]*schema.SDLChunk),
			Sequences: make(map[string]*schema.SDLChunk),
		}, nil
	}

	parseResult, err := pgparser.ParsePostgreSQL(sdlText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse SDL text")
	}

	extractor := &sdlChunkExtractor{
		sdlText: sdlText,
		chunks: &schema.SDLChunks{
			Tables:    make(map[string]*schema.SDLChunk),
			Views:     make(map[string]*schema.SDLChunk),
			Functions: make(map[string]*schema.SDLChunk),
			Indexes:   make(map[string]*schema.SDLChunk),
			Sequences: make(map[string]*schema.SDLChunk),
		},
		tokens: parseResult.Tokens,
	}

	antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)

	return extractor.chunks, nil
}

type sdlChunkExtractor struct {
	*parser.BasePostgreSQLParserListener
	sdlText string
	chunks  *schema.SDLChunks
	tokens  *antlr.CommonTokenStream
}

func (l *sdlChunkExtractor) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if ctx.Qualified_name(0) == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name(0))
	identifierStr := strings.Join(identifier, ".")

	// Ensure schema.tableName format
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.Tables[schemaQualifiedName] = chunk
}

func (l *sdlChunkExtractor) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	// Ensure schema.sequenceName format
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.Sequences[schemaQualifiedName] = chunk
}

func (l *sdlChunkExtractor) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	funcNameCtx := ctx.Func_name()
	if funcNameCtx == nil {
		return
	}

	// Extract function name using proper normalization
	funcNameParts := pgparser.NormalizePostgreSQLFuncName(funcNameCtx)
	if len(funcNameParts) == 0 {
		return
	}

	// Parse schema and function name directly from parts
	var schemaName string
	if len(funcNameParts) == 2 {
		// Schema qualified: schema.function_name
		schemaName = funcNameParts[0]
	} else if len(funcNameParts) == 1 {
		// Unqualified: function_name (assume public schema)
		schemaName = "public"
	} else {
		// Unexpected format
		return
	}

	// Use the unified function signature extraction
	signature := extractFunctionSignatureFromAST(ctx)
	if signature == "" {
		return
	}

	schemaQualifiedSignature := schemaName + "." + signature

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedSignature,
		ASTNode:    ctx,
	}

	l.chunks.Functions[schemaQualifiedSignature] = chunk
}

func (l *sdlChunkExtractor) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	// Check if this is CREATE INDEX
	if ctx.CREATE() == nil || ctx.INDEX() == nil {
		return
	}

	// Extract index name
	var indexName string
	if name := ctx.Name(); name != nil {
		indexName = pgparser.NormalizePostgreSQLName(name)
	}

	// If no explicit name, we skip it as PostgreSQL will generate one
	if indexName == "" {
		return
	}

	// Ensure schema.indexName format
	var schemaQualifiedName string
	if strings.Contains(indexName, ".") {
		schemaQualifiedName = indexName
	} else {
		schemaQualifiedName = "public." + indexName
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.Indexes[schemaQualifiedName] = chunk
}

func (l *sdlChunkExtractor) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	// Ensure schema.viewName format
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.Views[schemaQualifiedName] = chunk
}

// processTableChanges processes changes to tables by comparing SDL chunks
// nolint:unparam
func processTableChanges(currentChunks, previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) error {
	// Process current table chunks to find created and modified tables
	for _, currentChunk := range currentChunks.Tables {
		schemaName, tableName := parseIdentifier(currentChunk.Identifier)

		if previousChunk, exists := previousChunks.Tables[currentChunk.Identifier]; exists {
			// Table exists in both - check if modified
			currentText := currentChunk.GetText()
			previousText := previousChunk.GetText()
			if currentText != previousText {
				// Apply usability check: skip diff if current chunk matches database metadata SDL
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}
				// Table was modified - process column changes
				oldASTNode, ok := previousChunk.ASTNode.(*parser.CreatestmtContext)
				if !ok {
					return errors.Errorf("expected CreatestmtContext for previous table %s", previousChunk.Identifier)
				}
				newASTNode, ok := currentChunk.ASTNode.(*parser.CreatestmtContext)
				if !ok {
					return errors.Errorf("expected CreatestmtContext for current table %s", currentChunk.Identifier)
				}

				columnChanges := processColumnChanges(oldASTNode, newASTNode, currentSchema, previousSchema, currentDBSDLChunks, currentChunk.Identifier)
				foreignKeyChanges := processForeignKeyChanges(oldASTNode, newASTNode, currentDBSDLChunks, currentChunk.Identifier)
				checkConstraintChanges := processCheckConstraintChanges(oldASTNode, newASTNode, currentDBSDLChunks, currentChunk.Identifier)
				primaryKeyChanges := processPrimaryKeyChanges(oldASTNode, newASTNode, currentDBSDLChunks, currentChunk.Identifier)
				uniqueConstraintChanges := processUniqueConstraintChanges(oldASTNode, newASTNode, currentDBSDLChunks, currentChunk.Identifier)

				tableDiff := &schema.TableDiff{
					Action:                  schema.MetadataDiffActionAlter,
					SchemaName:              schemaName,
					TableName:               tableName,
					OldTable:                nil, // Will be populated when SDL drift detection is implemented
					NewTable:                nil, // Will be populated when SDL drift detection is implemented
					OldASTNode:              oldASTNode,
					NewASTNode:              newASTNode,
					ColumnChanges:           columnChanges,
					ForeignKeyChanges:       foreignKeyChanges,
					CheckConstraintChanges:  checkConstraintChanges,
					PrimaryKeyChanges:       primaryKeyChanges,
					UniqueConstraintChanges: uniqueConstraintChanges,
				}
				diff.TableChanges = append(diff.TableChanges, tableDiff)
			}
		} else {
			// New table
			// Handle generated chunks (from drift detection) that don't have AST nodes
			if currentChunk.ASTNode == nil {
				// This is a generated chunk, create a simplified diff entry
				tableDiff := &schema.TableDiff{
					Action:        schema.MetadataDiffActionCreate,
					SchemaName:    schemaName,
					TableName:     tableName,
					OldTable:      nil,
					NewTable:      nil,
					OldASTNode:    nil,
					NewASTNode:    nil,                    // No AST node for generated content
					ColumnChanges: []*schema.ColumnDiff{}, // Empty - table creation automatically creates all columns
				}
				diff.TableChanges = append(diff.TableChanges, tableDiff)
			} else {
				newASTNode, ok := currentChunk.ASTNode.(*parser.CreatestmtContext)
				if !ok {
					return errors.Errorf("expected CreatestmtContext for new table %s", currentChunk.Identifier)
				}

				tableDiff := &schema.TableDiff{
					Action:        schema.MetadataDiffActionCreate,
					SchemaName:    schemaName,
					TableName:     tableName,
					OldTable:      nil,
					NewTable:      nil, // Will be populated when SDL drift detection is implemented
					OldASTNode:    nil,
					NewASTNode:    newASTNode,
					ColumnChanges: []*schema.ColumnDiff{}, // Empty - table creation automatically creates all columns
				}
				diff.TableChanges = append(diff.TableChanges, tableDiff)
			}
		}
	}

	// Process previous table chunks to find dropped tables
	for identifier, previousChunk := range previousChunks.Tables {
		if _, exists := currentChunks.Tables[identifier]; !exists {
			// Table was dropped
			schemaName, tableName := parseIdentifier(previousChunk.Identifier)

			// Handle generated chunks (from drift detection) that don't have AST nodes
			if previousChunk.ASTNode == nil {
				// This is a generated chunk that was removed, skip it in the diff
				// It means this table was added during drift detection but doesn't exist in current SDL
				continue
			}

			oldASTNode, ok := previousChunk.ASTNode.(*parser.CreatestmtContext)
			if !ok {
				return errors.Errorf("expected CreatestmtContext for dropped table %s", previousChunk.Identifier)
			}

			tableDiff := &schema.TableDiff{
				Action:        schema.MetadataDiffActionDrop,
				SchemaName:    schemaName,
				TableName:     tableName,
				OldTable:      nil, // Will be populated when SDL drift detection is implemented
				NewTable:      nil,
				OldASTNode:    oldASTNode,
				NewASTNode:    nil,
				ColumnChanges: []*schema.ColumnDiff{}, // Empty - table drop doesn't need column analysis
			}
			diff.TableChanges = append(diff.TableChanges, tableDiff)
		}
	}

	return nil
}

// processColumnChanges analyzes column changes between old and new table definitions
// Following the same pattern as processTableChanges: compare text first, then analyze differences
// If schemas are nil, operates in AST-only mode without metadata extraction
func processColumnChanges(oldTable, newTable *parser.CreatestmtContext, currentSchema, previousSchema *model.DatabaseSchema, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.ColumnDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.ColumnDiff{}
	}

	// Detect AST-only mode: when both schemas are nil, operate without metadata extraction
	astOnlyMode := currentSchema == nil && previousSchema == nil

	// Step 1: Extract all column definitions with their AST nodes for text comparison
	oldColumns := extractColumnDefinitionsWithAST(oldTable)
	newColumns := extractColumnDefinitionsWithAST(newTable)

	var columnDiffs []*schema.ColumnDiff

	// Step 2: Process current columns to find created and modified columns
	// Use the order from the new table's AST to maintain original column order
	for _, columnName := range newColumns.Order {
		newColumnDef := newColumns.Map[columnName]
		if oldColumnDef, exists := oldColumns.Map[columnName]; exists {
			// Column exists in both - check if modified by comparing text first
			currentText := getColumnText(newColumnDef.ASTNode)
			previousText := getColumnText(oldColumnDef.ASTNode)
			if currentText != previousText {
				// Apply column-level usability check: skip diff if current column matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipColumnDiffForUsability(currentText, schemaName, tableName, columnName) {
					continue
				}
				// Column was modified - extract metadata only if not in AST-only mode
				var oldColumn, newColumn *storepb.ColumnMetadata
				if !astOnlyMode {
					oldColumn = extractColumnMetadata(oldColumnDef.ASTNode)
					newColumn = extractColumnMetadata(newColumnDef.ASTNode)
				}

				columnDiffs = append(columnDiffs, &schema.ColumnDiff{
					Action:     schema.MetadataDiffActionAlter,
					OldColumn:  oldColumn,
					NewColumn:  newColumn,
					OldASTNode: oldColumnDef.ASTNode,
					NewASTNode: newColumnDef.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New column - extract metadata only if not in AST-only mode
			var newColumn *storepb.ColumnMetadata
			if !astOnlyMode {
				newColumn = extractColumnMetadata(newColumnDef.ASTNode)
			}
			columnDiffs = append(columnDiffs, &schema.ColumnDiff{
				Action:     schema.MetadataDiffActionCreate,
				OldColumn:  nil,
				NewColumn:  newColumn,
				OldASTNode: nil,
				NewASTNode: newColumnDef.ASTNode,
			})
		}
	}

	// Step 3: Process previous columns to find dropped columns
	// Use the order from the old table's AST to maintain original column order
	for _, columnName := range oldColumns.Order {
		oldColumnDef := oldColumns.Map[columnName]
		if _, exists := newColumns.Map[columnName]; !exists {
			// Column was dropped - extract metadata only if not in AST-only mode
			var oldColumn *storepb.ColumnMetadata
			if !astOnlyMode {
				oldColumn = extractColumnMetadata(oldColumnDef.ASTNode)
			}
			columnDiffs = append(columnDiffs, &schema.ColumnDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldColumn:  oldColumn,
				NewColumn:  nil,
				OldASTNode: oldColumnDef.ASTNode,
				NewASTNode: nil,
			})
		}
	}

	return columnDiffs
}

// ColumnDefWithAST holds a column definition AST node with its name for efficient lookup
type ColumnDefWithAST struct {
	Name    string
	ASTNode parser.IColumnDefContext
}

// ColumnDefWithASTOrdered holds column definitions in AST order for deterministic processing
type ColumnDefWithASTOrdered struct {
	Map   map[string]*ColumnDefWithAST
	Order []string // Column names in the order they appear in the AST
}

// extractColumnDefinitionsWithAST extracts column definitions and preserves their AST order
// This ensures deterministic processing while maintaining the original column order
func extractColumnDefinitionsWithAST(createStmt *parser.CreatestmtContext) *ColumnDefWithASTOrdered {
	result := &ColumnDefWithASTOrdered{
		Map:   make(map[string]*ColumnDefWithAST),
		Order: []string{},
	}

	if createStmt == nil {
		return result
	}

	// Get the optTableElementList which contains column definitions
	if createStmt.Opttableelementlist() != nil && createStmt.Opttableelementlist().Tableelementlist() != nil {
		elementList := createStmt.Opttableelementlist().Tableelementlist()

		for _, element := range elementList.AllTableelement() {
			// Check if this is a columnDef (column definition)
			if element.ColumnDef() != nil {
				columnDef := element.ColumnDef()
				if columnDef.Colid() != nil {
					columnName := pgparser.NormalizePostgreSQLColid(columnDef.Colid())

					result.Map[columnName] = &ColumnDefWithAST{
						Name:    columnName,
						ASTNode: columnDef,
					}
					// Preserve the order columns appear in the AST
					result.Order = append(result.Order, columnName)
				}
			}
		}
	}

	return result
}

// extractColumnMetadata extracts full column metadata from a single column AST node
// This is called only when we actually need the detailed information
func extractColumnMetadata(columnDef parser.IColumnDefContext) *storepb.ColumnMetadata {
	if columnDef == nil {
		return nil
	}

	columnName := pgparser.NormalizePostgreSQLColid(columnDef.Colid())

	return &storepb.ColumnMetadata{
		Name:      columnName,
		Type:      extractColumnType(columnDef.Typename()),
		Nullable:  extractColumnNullable(columnDef),
		Default:   extractColumnDefault(columnDef),
		Comment:   extractColumnComment(columnDef),
		Collation: extractColumnCollation(columnDef),
	}
}

// parseIdentifier parses a table identifier and returns schema name and table name
func parseIdentifier(identifier string) (schemaName, objectName string) {
	parts := strings.Split(identifier, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", identifier
}

// createTableExtractor extracts CREATE TABLE AST node from parse tree
type createTableExtractor struct {
	*parser.BasePostgreSQLParserListener
	result **parser.CreatestmtContext
}

func (e *createTableExtractor) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	*e.result = ctx
}

// getColumnText extracts the text representation of a column definition
func getColumnText(columnAST parser.IColumnDefContext) string {
	if columnAST == nil {
		return ""
	}

	// Get tokens from the parser
	if parser := columnAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := columnAST.GetStart()
			stop := columnAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return columnAST.GetText()
}

// extractColumnType extracts column type information from the typename AST node
func extractColumnType(typename parser.ITypenameContext) string {
	if typename == nil {
		return "unknown"
	}

	// Get tokens from the parser
	if parser := typename.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := typename.GetStart()
			stop := typename.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return typename.GetText()
}

// extractColumnNullable extracts whether a column is nullable from column constraints
func extractColumnNullable(columnDef parser.IColumnDefContext) bool {
	if columnDef == nil || columnDef.Colquallist() == nil {
		return true // Default is nullable
	}

	// Check all column constraints
	for _, constraint := range columnDef.Colquallist().AllColconstraint() {
		if constraint.Colconstraintelem() != nil {
			elem := constraint.Colconstraintelem()
			if elem.NOT() != nil && elem.NULL_P() != nil {
				return false // NOT NULL
			}
			if elem.PRIMARY() != nil && elem.KEY() != nil {
				return false // PRIMARY KEY implies NOT NULL
			}
		}
	}

	return true // Default is nullable
}

// extractColumnDefault extracts the default value from column constraints
func extractColumnDefault(columnDef parser.IColumnDefContext) string {
	if columnDef == nil || columnDef.Colquallist() == nil {
		return ""
	}

	// Check all column constraints for DEFAULT
	for _, constraint := range columnDef.Colquallist().AllColconstraint() {
		if constraint.Colconstraintelem() != nil {
			elem := constraint.Colconstraintelem()
			if elem.DEFAULT() != nil && elem.B_expr() != nil {
				// Extract default value text
				if parser := elem.GetParser(); parser != nil {
					if tokenStream := parser.GetTokenStream(); tokenStream != nil {
						start := elem.B_expr().GetStart()
						stop := elem.B_expr().GetStop()
						if start != nil && stop != nil {
							return tokenStream.GetTextFromTokens(start, stop)
						}
					}
				}
			}
		}
	}

	return ""
}

// extractColumnComment extracts column comment (PostgreSQL doesn't store comments in CREATE TABLE syntax)
func extractColumnComment(_ parser.IColumnDefContext) string {
	// PostgreSQL column comments are set using COMMENT ON COLUMN, not in CREATE TABLE
	// This information is not available in the CREATE TABLE AST
	return ""
}

// extractColumnCollation extracts collation from column constraints
func extractColumnCollation(columnDef parser.IColumnDefContext) string {
	if columnDef == nil || columnDef.Colquallist() == nil {
		return ""
	}

	// Check all column constraints for COLLATE
	for _, constraint := range columnDef.Colquallist().AllColconstraint() {
		if constraint.COLLATE() != nil && constraint.Any_name() != nil {
			// Extract collation name
			return extractAnyName(constraint.Any_name())
		}
	}

	return ""
}

// extractAnyName extracts the string representation from an Any_name context
func extractAnyName(anyName parser.IAny_nameContext) string {
	if anyName == nil {
		return ""
	}

	// Get tokens from the parser
	if parser := anyName.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := anyName.GetStart()
			stop := anyName.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return anyName.GetText()
}

// ForeignKeyDefWithAST holds foreign key constraint definition with its AST node for text comparison
type ForeignKeyDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// CheckConstraintDefWithAST holds check constraint definition with its AST node for text comparison
type CheckConstraintDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// IndexDefWithAST holds index/unique constraint definition with its AST node for text comparison
type IndexDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// processForeignKeyChanges analyzes foreign key constraint changes between old and new table definitions
// Following the text-first comparison pattern for performance optimization
func processForeignKeyChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.ForeignKeyDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.ForeignKeyDiff{}
	}

	// Step 1: Extract all foreign key definitions with their AST nodes for text comparison
	oldFKList := extractForeignKeyDefinitionsInOrder(oldTable)
	newFKList := extractForeignKeyDefinitionsInOrder(newTable)

	// Create maps for quick lookup
	oldFKMap := make(map[string]*ForeignKeyDefWithAST)
	for _, def := range oldFKList {
		oldFKMap[def.Name] = def
	}
	newFKMap := make(map[string]*ForeignKeyDefWithAST)
	for _, def := range newFKList {
		newFKMap[def.Name] = def
	}

	var fkDiffs []*schema.ForeignKeyDiff

	// Step 2: Process current foreign keys to find created and modified foreign keys
	for _, newFKDef := range newFKList {
		if oldFKDef, exists := oldFKMap[newFKDef.Name]; exists {
			// FK exists in both - check if modified by comparing text first
			currentText := getForeignKeyText(newFKDef.ASTNode)
			previousText := getForeignKeyText(oldFKDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, newFKDef.Name) {
					continue
				}
				// FK was modified - drop and recreate (PostgreSQL pattern)
				fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldFKDef.ASTNode,
				})
				fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newFKDef.ASTNode,
				})
			}
		} else {
			// New foreign key - store AST node only
			fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newFKDef.ASTNode,
			})
		}
	}

	// Step 3: Process old foreign keys to find dropped ones
	for _, oldFKDef := range oldFKList {
		if _, exists := newFKMap[oldFKDef.Name]; !exists {
			// Foreign key was dropped - store AST node only
			fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldFKDef.ASTNode,
			})
		}
	}

	return fkDiffs
}

// processCheckConstraintChanges analyzes check constraint changes between old and new table definitions
// Following the text-first comparison pattern for performance optimization
func processCheckConstraintChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.CheckConstraintDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.CheckConstraintDiff{}
	}

	// Step 1: Extract all check constraint definitions with their AST nodes for text comparison
	oldCheckList := extractCheckConstraintDefinitionsInOrder(oldTable)
	newCheckList := extractCheckConstraintDefinitionsInOrder(newTable)

	// Create maps for quick lookup
	oldCheckMap := make(map[string]*CheckConstraintDefWithAST)
	for _, def := range oldCheckList {
		oldCheckMap[def.Name] = def
	}
	newCheckMap := make(map[string]*CheckConstraintDefWithAST)
	for _, def := range newCheckList {
		newCheckMap[def.Name] = def
	}

	var checkDiffs []*schema.CheckConstraintDiff

	// Step 2: Process current check constraints to find created and modified check constraints
	for _, newCheckDef := range newCheckList {
		if oldCheckDef, exists := oldCheckMap[newCheckDef.Name]; exists {
			// Check constraint exists in both - check if modified by comparing text first
			currentText := getCheckConstraintText(newCheckDef.ASTNode)
			previousText := getCheckConstraintText(oldCheckDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, newCheckDef.Name) {
					continue
				}
				// Check constraint was modified - drop and recreate (PostgreSQL pattern)
				checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldCheckDef.ASTNode,
				})
				checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newCheckDef.ASTNode,
				})
			}
		} else {
			// New check constraint - store AST node only
			checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newCheckDef.ASTNode,
			})
		}
	}

	// Step 3: Process old check constraints to find dropped ones
	for _, oldCheckDef := range oldCheckList {
		if _, exists := newCheckMap[oldCheckDef.Name]; !exists {
			// Check constraint was dropped - store AST node only
			checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldCheckDef.ASTNode,
			})
		}
	}

	return checkDiffs
}

// processPrimaryKeyChanges analyzes primary key constraint changes between old and new table definitions
func processPrimaryKeyChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.PrimaryKeyDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.PrimaryKeyDiff{}
	}

	// Step 1: Extract all primary key constraint definitions with their AST nodes for text comparison
	oldPKMap := extractPrimaryKeyDefinitionsWithAST(oldTable)
	newPKMap := extractPrimaryKeyDefinitionsWithAST(newTable)

	var pkDiffs []*schema.PrimaryKeyDiff

	// Step 2: Process current primary keys to find created and modified primary keys
	for pkName, newPKDef := range newPKMap {
		if oldPKDef, exists := oldPKMap[pkName]; exists {
			// PK exists in both - check if modified by comparing text first
			currentText := getIndexText(newPKDef.ASTNode)
			previousText := getIndexText(oldPKDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, pkName) {
					continue
				}
				// PK was modified - store AST nodes only
				pkDiffs = append(pkDiffs, &schema.PrimaryKeyDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldPKDef.ASTNode,
				})
				pkDiffs = append(pkDiffs, &schema.PrimaryKeyDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newPKDef.ASTNode,
				})
			}
		} else {
			// New PK - store AST node only
			pkDiffs = append(pkDiffs, &schema.PrimaryKeyDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newPKDef.ASTNode,
			})
		}
	}

	// Step 3: Process old primary keys to find dropped ones
	for pkName, oldPKDef := range oldPKMap {
		if _, exists := newPKMap[pkName]; !exists {
			// PK was dropped - store AST node only
			pkDiffs = append(pkDiffs, &schema.PrimaryKeyDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldPKDef.ASTNode,
			})
		}
	}

	return pkDiffs
}

// processUniqueConstraintChanges analyzes unique constraint changes between old and new table definitions
func processUniqueConstraintChanges(oldTable, newTable *parser.CreatestmtContext, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.UniqueConstraintDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.UniqueConstraintDiff{}
	}

	// Step 1: Extract all unique constraint definitions with their AST nodes for text comparison
	oldUKList := extractUniqueConstraintDefinitionsInOrder(oldTable)
	newUKList := extractUniqueConstraintDefinitionsInOrder(newTable)

	// Create maps for quick lookup
	oldUKMap := make(map[string]*IndexDefWithAST)
	for _, def := range oldUKList {
		oldUKMap[def.Name] = def
	}
	newUKMap := make(map[string]*IndexDefWithAST)
	for _, def := range newUKList {
		newUKMap[def.Name] = def
	}

	var ukDiffs []*schema.UniqueConstraintDiff

	// Step 2: Process current unique constraints to find created and modified unique constraints
	for _, newUKDef := range newUKList {
		if oldUKDef, exists := oldUKMap[newUKDef.Name]; exists {
			// UK exists in both - check if modified by comparing text first
			currentText := getIndexText(newUKDef.ASTNode)
			previousText := getIndexText(oldUKDef.ASTNode)
			if currentText != previousText {
				// Apply constraint-level usability check: skip diff if current constraint matches database metadata SDL
				schemaName, tableName := parseIdentifier(tableIdentifier)
				if currentDBSDLChunks != nil && currentDBSDLChunks.shouldSkipConstraintDiffForUsability(currentText, schemaName, tableName, newUKDef.Name) {
					continue
				}
				// UK was modified - drop and recreate (PostgreSQL pattern)
				ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldUKDef.ASTNode,
				})
				ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newUKDef.ASTNode,
				})
			}
		} else {
			// New UK - store AST node only
			ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newUKDef.ASTNode,
			})
		}
	}

	// Step 3: Process old unique constraints to find dropped ones
	for _, oldUKDef := range oldUKList {
		if _, exists := newUKMap[oldUKDef.Name]; !exists {
			// UK was dropped - store AST node only
			ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldUKDef.ASTNode,
			})
		}
	}

	return ukDiffs
}

// extractUniqueConstraintDefinitionsInOrder extracts unique constraints with their AST nodes in SQL order
func extractUniqueConstraintDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*IndexDefWithAST {
	var ukList []*IndexDefWithAST

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return ukList
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return ukList
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				// Check for UNIQUE constraints (but not PRIMARY KEY)
				isUnique := elem.UNIQUE() != nil && (elem.PRIMARY() == nil || elem.KEY() == nil)

				if isUnique {
					// This is a unique constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					ukList = append(ukList, &IndexDefWithAST{
						Name:    name,
						ASTNode: constraint,
					})
				}
			}
		}
	}

	return ukList
}

// extractForeignKeyDefinitionsInOrder extracts foreign key constraints with their AST nodes in SQL order
func extractForeignKeyDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*ForeignKeyDefWithAST {
	var fkList []*ForeignKeyDefWithAST

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return fkList
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return fkList
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				if elem.FOREIGN() != nil && elem.KEY() != nil {
					// This is a foreign key constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					fkList = append(fkList, &ForeignKeyDefWithAST{
						Name:    name,
						ASTNode: constraint,
					})
				}
			}
		}
	}

	return fkList
}

// extractCheckConstraintDefinitionsInOrder extracts check constraints with their AST nodes in SQL order
func extractCheckConstraintDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*CheckConstraintDefWithAST {
	var checkList []*CheckConstraintDefWithAST

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return checkList
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return checkList
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				if elem.CHECK() != nil {
					// This is a check constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					checkList = append(checkList, &CheckConstraintDefWithAST{
						Name:    name,
						ASTNode: constraint,
					})
				}
			}
		}
	}

	return checkList
}

// extractPrimaryKeyDefinitionsWithAST extracts primary key constraints with their AST nodes
func extractPrimaryKeyDefinitionsWithAST(createStmt *parser.CreatestmtContext) map[string]*IndexDefWithAST {
	pkMap := make(map[string]*IndexDefWithAST)

	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return pkMap
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return pkMap
	}

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() != nil {
			constraint := element.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()
				// Check for PRIMARY KEY constraints
				isPrimary := elem.PRIMARY() != nil && elem.KEY() != nil

				if isPrimary {
					// This is a primary key constraint
					name := ""
					if constraint.Name() != nil {
						name = pgparser.NormalizePostgreSQLName(constraint.Name())
					}
					// Use constraint definition text as fallback key if name is empty
					if name == "" {
						// Get the full original text from tokens
						if parser := constraint.GetParser(); parser != nil {
							if tokenStream := parser.GetTokenStream(); tokenStream != nil {
								name = tokenStream.GetTextFromRuleContext(constraint)
							}
						}
						if name == "" {
							name = constraint.GetText() // Final fallback
						}
					}
					pkMap[name] = &IndexDefWithAST{
						Name:    name,
						ASTNode: constraint,
					}
				}
			}
		}
	}

	return pkMap
}

// getForeignKeyText returns the text representation of a foreign key constraint for comparison
func getForeignKeyText(constraintAST parser.ITableconstraintContext) string {
	if constraintAST == nil {
		return ""
	}

	// Get tokens from the parser for precise text extraction
	if parser := constraintAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := constraintAST.GetStart()
			stop := constraintAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return constraintAST.GetText()
}

// getCheckConstraintText returns the text representation of a check constraint for comparison
func getCheckConstraintText(constraintAST parser.ITableconstraintContext) string {
	if constraintAST == nil {
		return ""
	}

	// Get tokens from the parser for precise text extraction
	if parser := constraintAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := constraintAST.GetStart()
			stop := constraintAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return constraintAST.GetText()
}

// getIndexText returns the text representation of an index/unique constraint for comparison
func getIndexText(constraintAST parser.ITableconstraintContext) string {
	if constraintAST == nil {
		return ""
	}

	// Get tokens from the parser for precise text extraction
	if parser := constraintAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := constraintAST.GetStart()
			stop := constraintAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return constraintAST.GetText()
}

// processStandaloneIndexChanges analyzes standalone CREATE INDEX statement changes
// and adds them to the appropriate table's IndexChanges
func processStandaloneIndexChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	if currentChunks == nil || previousChunks == nil {
		return
	}

	// Initialize map with all existing table diffs for efficient lookups
	affectedTables := make(map[string]*schema.TableDiff, len(diff.TableChanges))
	for _, tableDiff := range diff.TableChanges {
		affectedTables[tableDiff.TableName] = tableDiff
	}

	// Step 1: Process current indexes to find created and modified indexes
	for _, currentChunk := range currentChunks.Indexes {
		tableName := extractTableNameFromIndex(currentChunk.ASTNode)
		if tableName == "" {
			continue // Skip if we can't determine the table name
		}

		if previousChunk, exists := previousChunks.Indexes[currentChunk.Identifier]; exists {
			// Index exists in both - check if modified by comparing text first
			currentText := getStandaloneIndexText(currentChunk.ASTNode)
			previousText := getStandaloneIndexText(previousChunk.ASTNode)
			if currentText != previousText {
				// Apply usability check: skip diff if current chunk matches database metadata
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}
				// Index was modified - use drop and recreate pattern (PostgreSQL standard)
				tableDiff := getOrCreateTableDiff(diff, tableName, affectedTables)
				tableDiff.IndexChanges = append(tableDiff.IndexChanges, &schema.IndexDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: previousChunk.ASTNode,
				})
				tableDiff.IndexChanges = append(tableDiff.IndexChanges, &schema.IndexDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: currentChunk.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New index - store AST node only
			tableDiff := getOrCreateTableDiff(diff, tableName, affectedTables)
			tableDiff.IndexChanges = append(tableDiff.IndexChanges, &schema.IndexDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: currentChunk.ASTNode,
			})
		}
	}

	// Step 2: Process previous indexes to find dropped ones
	for indexName, previousChunk := range previousChunks.Indexes {
		if _, exists := currentChunks.Indexes[indexName]; !exists {
			// Index was dropped - store AST node only
			tableName := extractTableNameFromIndex(previousChunk.ASTNode)
			if tableName == "" {
				continue // Skip if we can't determine the table name
			}

			tableDiff := getOrCreateTableDiff(diff, tableName, affectedTables)
			tableDiff.IndexChanges = append(tableDiff.IndexChanges, &schema.IndexDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: previousChunk.ASTNode,
			})
		}
	}
}

// getStandaloneIndexText returns the text representation of a standalone CREATE INDEX statement for comparison
func getStandaloneIndexText(astNode any) string {
	indexStmt, ok := astNode.(*parser.IndexstmtContext)
	if !ok || indexStmt == nil {
		return ""
	}

	// Get tokens from the parser for precise text extraction with original spacing
	if parser := indexStmt.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := indexStmt.GetStart()
			stop := indexStmt.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return indexStmt.GetText()
}

// extractTableNameFromIndex extracts the table name from a CREATE INDEX statement
func extractTableNameFromIndex(astNode any) string {
	indexStmt, ok := astNode.(*parser.IndexstmtContext)
	if !ok || indexStmt == nil {
		return ""
	}

	// Extract table name from relation_expr in CREATE INDEX ... ON table_name
	if relationExpr := indexStmt.Relation_expr(); relationExpr != nil {
		if qualifiedName := relationExpr.Qualified_name(); qualifiedName != nil {
			// Extract qualified name and return the table name (without schema)
			qualifiedNameParts := pgparser.NormalizePostgreSQLQualifiedName(qualifiedName)
			if len(qualifiedNameParts) > 0 {
				return qualifiedNameParts[len(qualifiedNameParts)-1] // Return the last part (table name)
			}
		}
	}

	return ""
}

// getOrCreateTableDiff finds an existing table diff or creates a new one for the given table
func getOrCreateTableDiff(diff *schema.MetadataDiff, tableName string, affectedTables map[string]*schema.TableDiff) *schema.TableDiff {
	// Check if we already have this table in our map
	if tableDiff, exists := affectedTables[tableName]; exists {
		return tableDiff
	}

	// Create a new table diff for standalone index changes
	// We set Action to ALTER since we're modifying an existing table by adding/removing indexes
	newTableDiff := &schema.TableDiff{
		Action:                  schema.MetadataDiffActionAlter,
		SchemaName:              "public", // Default schema for PostgreSQL
		TableName:               tableName,
		OldTable:                nil, // Will be populated when SDL drift detection is implemented
		NewTable:                nil, // Will be populated when SDL drift detection is implemented
		OldASTNode:              nil, // No table-level AST changes for standalone indexes
		NewASTNode:              nil, // No table-level AST changes for standalone indexes
		ColumnChanges:           []*schema.ColumnDiff{},
		IndexChanges:            []*schema.IndexDiff{},
		PrimaryKeyChanges:       []*schema.PrimaryKeyDiff{},
		UniqueConstraintChanges: []*schema.UniqueConstraintDiff{},
		ForeignKeyChanges:       []*schema.ForeignKeyDiff{},
		CheckConstraintChanges:  []*schema.CheckConstraintDiff{},
	}

	diff.TableChanges = append(diff.TableChanges, newTableDiff)
	affectedTables[tableName] = newTableDiff
	return newTableDiff
}

// processViewChanges analyzes view changes between current and previous chunks
// Following the text-first comparison pattern for performance optimization
func processViewChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current views to find created and modified views
	for _, currentChunk := range currentChunks.Views {
		if previousChunk, exists := previousChunks.Views[currentChunk.Identifier]; exists {
			// View exists in both - check if modified by comparing text first
			currentText := currentChunk.GetText()
			previousText := previousChunk.GetText()
			if currentText != previousText {
				// Apply usability check: skip diff if current chunk matches database metadata SDL
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}
				// View was modified - use drop and recreate pattern (PostgreSQL standard)
				schemaName, viewName := parseIdentifier(currentChunk.Identifier)
				diff.ViewChanges = append(diff.ViewChanges, &schema.ViewDiff{
					Action:     schema.MetadataDiffActionDrop,
					SchemaName: schemaName,
					ViewName:   viewName,
					OldView:    nil, // Will be populated when SDL drift detection is implemented
					NewView:    nil, // Will be populated when SDL drift detection is implemented
					OldASTNode: previousChunk.ASTNode,
					NewASTNode: nil,
				})
				diff.ViewChanges = append(diff.ViewChanges, &schema.ViewDiff{
					Action:     schema.MetadataDiffActionCreate,
					SchemaName: schemaName,
					ViewName:   viewName,
					OldView:    nil, // Will be populated when SDL drift detection is implemented
					NewView:    nil, // Will be populated when SDL drift detection is implemented
					OldASTNode: nil,
					NewASTNode: currentChunk.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New view
			schemaName, viewName := parseIdentifier(currentChunk.Identifier)
			diff.ViewChanges = append(diff.ViewChanges, &schema.ViewDiff{
				Action:     schema.MetadataDiffActionCreate,
				SchemaName: schemaName,
				ViewName:   viewName,
				OldView:    nil,
				NewView:    nil, // Will be populated when SDL drift detection is implemented
				OldASTNode: nil,
				NewASTNode: currentChunk.ASTNode,
			})
		}
	}

	// Process previous views to find dropped ones
	for identifier, previousChunk := range previousChunks.Views {
		if _, exists := currentChunks.Views[identifier]; !exists {
			// View was dropped
			schemaName, viewName := parseIdentifier(identifier)
			diff.ViewChanges = append(diff.ViewChanges, &schema.ViewDiff{
				Action:     schema.MetadataDiffActionDrop,
				SchemaName: schemaName,
				ViewName:   viewName,
				OldView:    nil, // Will be populated when SDL drift detection is implemented
				NewView:    nil,
				OldASTNode: previousChunk.ASTNode,
				NewASTNode: nil,
			})
		}
	}
}

// processFunctionChanges analyzes function changes between current and previous chunks
// Following the text-first comparison pattern for performance optimization
func processFunctionChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current functions to find created and modified functions
	for _, currentChunk := range currentChunks.Functions {
		if previousChunk, exists := previousChunks.Functions[currentChunk.Identifier]; exists {
			// Function exists in both - check if modified by comparing text first
			currentText := currentChunk.GetText()
			previousText := previousChunk.GetText()
			if currentText != previousText {
				// Apply usability check: skip diff if current chunk matches database metadata SDL
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}
				// Function was modified - use CREATE OR REPLACE (AST-only mode)
				schemaName, functionName := parseIdentifier(currentChunk.Identifier)
				diff.FunctionChanges = append(diff.FunctionChanges, &schema.FunctionDiff{
					Action:       schema.MetadataDiffActionAlter,
					SchemaName:   schemaName,
					FunctionName: functionName,
					OldFunction:  nil, // Will be populated when SDL drift detection is implemented
					NewFunction:  nil, // Will be populated when SDL drift detection is implemented
					OldASTNode:   previousChunk.ASTNode,
					NewASTNode:   currentChunk.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New function
			schemaName, functionName := parseIdentifier(currentChunk.Identifier)
			diff.FunctionChanges = append(diff.FunctionChanges, &schema.FunctionDiff{
				Action:       schema.MetadataDiffActionCreate,
				SchemaName:   schemaName,
				FunctionName: functionName,
				OldFunction:  nil,
				NewFunction:  nil,
				OldASTNode:   nil,
				NewASTNode:   currentChunk.ASTNode,
			})
		}
	}

	// Process previous functions to find dropped ones
	for identifier, previousChunk := range previousChunks.Functions {
		if _, exists := currentChunks.Functions[identifier]; !exists {
			// Function was dropped
			schemaName, functionName := parseIdentifier(identifier)
			diff.FunctionChanges = append(diff.FunctionChanges, &schema.FunctionDiff{
				Action:       schema.MetadataDiffActionDrop,
				SchemaName:   schemaName,
				FunctionName: functionName,
				OldFunction:  nil, // Will be populated when SDL drift detection is implemented
				NewFunction:  nil,
				OldASTNode:   previousChunk.ASTNode,
				NewASTNode:   nil,
			})
		}
	}
}

// processSequenceChanges analyzes sequence changes between current and previous chunks
// Following the text-first comparison pattern for performance optimization
func processSequenceChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current sequences to find created and modified sequences
	for _, currentChunk := range currentChunks.Sequences {
		if previousChunk, exists := previousChunks.Sequences[currentChunk.Identifier]; exists {
			// Sequence exists in both - check if modified by comparing text first
			currentText := currentChunk.GetText()
			previousText := previousChunk.GetText()
			if currentText != previousText {
				// Apply usability check: skip diff if current chunk matches database metadata SDL
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}
				// Sequence was modified - use drop and recreate pattern (PostgreSQL standard)
				schemaName, sequenceName := parseIdentifier(currentChunk.Identifier)
				diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
					Action:       schema.MetadataDiffActionDrop,
					SchemaName:   schemaName,
					SequenceName: sequenceName,
					OldSequence:  nil, // Will be populated when SDL drift detection is implemented
					NewSequence:  nil, // Will be populated when SDL drift detection is implemented
					OldASTNode:   previousChunk.ASTNode,
					NewASTNode:   nil,
				})
				diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
					Action:       schema.MetadataDiffActionCreate,
					SchemaName:   schemaName,
					SequenceName: sequenceName,
					OldSequence:  nil, // Will be populated when SDL drift detection is implemented
					NewSequence:  nil, // Will be populated when SDL drift detection is implemented
					OldASTNode:   nil,
					NewASTNode:   currentChunk.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New sequence
			schemaName, sequenceName := parseIdentifier(currentChunk.Identifier)
			diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
				Action:       schema.MetadataDiffActionCreate,
				SchemaName:   schemaName,
				SequenceName: sequenceName,
				OldSequence:  nil,
				NewSequence:  nil,
				OldASTNode:   nil,
				NewASTNode:   currentChunk.ASTNode,
			})
		}
	}

	// Process previous sequences to find dropped ones
	for identifier, previousChunk := range previousChunks.Sequences {
		if _, exists := currentChunks.Sequences[identifier]; !exists {
			// Sequence was dropped
			schemaName, sequenceName := parseIdentifier(identifier)
			diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
				Action:       schema.MetadataDiffActionDrop,
				SchemaName:   schemaName,
				SequenceName: sequenceName,
				OldSequence:  nil, // Will be populated when SDL drift detection is implemented
				NewSequence:  nil,
				OldASTNode:   previousChunk.ASTNode,
				NewASTNode:   nil,
			})
		}
	}
}

// applyMinimalChangesToChunks applies minimal changes to the previous SDL chunks based on schema differences
// This implements the minimal change principle for drift scenarios by directly manipulating chunks
func applyMinimalChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get table differences between schemas
	currentMetadata := currentSchema.GetMetadata()
	previousMetadata := previousSchema.GetMetadata()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Create maps for efficient table lookup using full schema.table identifier
	currentTables := make(map[string]*storepb.TableMetadata)
	previousTables := make(map[string]*storepb.TableMetadata)

	// Build current tables map with proper schema qualification
	for _, schema := range currentMetadata.Schemas {
		schemaName := schema.Name
		if schemaName == "" {
			schemaName = "public" // Default schema for PostgreSQL
		}
		for _, table := range schema.Tables {
			tableKey := schemaName + "." + table.Name
			currentTables[tableKey] = table
		}
	}

	// Build previous tables map with proper schema qualification
	for _, schema := range previousMetadata.Schemas {
		schemaName := schema.Name
		if schemaName == "" {
			schemaName = "public" // Default schema for PostgreSQL
		}
		for _, table := range schema.Tables {
			tableKey := schemaName + "." + table.Name
			previousTables[tableKey] = table
		}
	}

	// Build existing chunk keys mapping for table operations
	existingChunkKeys := make(map[string]string) // schema.table -> chunk key
	for chunkKey, chunk := range previousChunks.Tables {
		if chunk != nil {
			existingChunkKeys[chunk.Identifier] = chunkKey
		}
	}

	// Process table additions: add new tables to chunks
	for tableKey, currentTable := range currentTables {
		if _, exists := previousTables[tableKey]; !exists {
			// Table was added - generate SDL for the new table and parse it to AST
			var buf strings.Builder
			schemaName, _ := parseIdentifier(tableKey)
			err := writeCreateTableSDL(&buf, schemaName, currentTable)
			if err != nil {
				return errors.Wrapf(err, "failed to generate SDL for new table %s", tableKey)
			}
			tableSDL := buf.String()

			// Parse the generated SDL to create AST node
			parseResult, err := pgparser.ParsePostgreSQL(tableSDL)
			if err != nil {
				return errors.Wrapf(err, "failed to parse generated SDL for new table %s", tableKey)
			}

			// Extract the CREATE TABLE AST node
			var createTableNode *parser.CreatestmtContext
			antlr.ParseTreeWalkerDefault.Walk(&createTableExtractor{
				result: &createTableNode,
			}, parseResult.Tree)

			if createTableNode == nil {
				return errors.Errorf("failed to extract CREATE TABLE AST node for new table %s", tableKey)
			}

			// Always use schema.table format for all table identifiers
			chunkKey := tableKey

			// Add the new table chunk with proper AST node
			previousChunks.Tables[chunkKey] = &schema.SDLChunk{
				Identifier: chunkKey,
				ASTNode:    createTableNode,
			}
		}
	}

	// Process table modifications: apply column-level changes using ANTLR rewrite
	for tableKey, currentTable := range currentTables {
		if previousTable, exists := previousTables[tableKey]; exists {
			// Table exists in both schemas - check for column differences
			if existingKey, chunkExists := existingChunkKeys[tableKey]; chunkExists {
				if chunk := previousChunks.Tables[existingKey]; chunk != nil {
					// Apply both column and constraint changes to the existing chunk using a single rewriter
					err := applyTableChangesToChunk(chunk, currentTable, previousTable)
					if err != nil {
						return errors.Wrapf(err, "failed to apply table changes to table %s", tableKey)
					}
				}
			}
		}
	}

	// Process table deletions: remove dropped tables from chunks
	for tableKey := range previousTables {
		if _, exists := currentTables[tableKey]; !exists {
			// Table was dropped - find the corresponding chunk key and remove it
			if existingKey, exists := existingChunkKeys[tableKey]; exists {
				delete(previousChunks.Tables, existingKey)
			}
			// If no mapping exists, the table was not in the original chunks,
			// so there's nothing to delete - this is the expected behavior
		}
	}

	// Process standalone index changes: apply minimal changes to index chunks
	err := applyStandaloneIndexChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply standalone index changes")
	}

	// Process function changes: apply minimal changes to function chunks
	err = applyFunctionChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply function changes")
	}

	// Process sequence changes: apply minimal changes to sequence chunks
	err = applySequenceChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply sequence changes")
	}

	return nil
}

// applyTableChangesToChunk applies minimal column and constraint changes to an existing CREATE TABLE chunk
// by working with the individual chunk's SQL text instead of the full script's tokenStream
func applyTableChangesToChunk(chunk *schema.SDLChunk, currentTable, previousTable *storepb.TableMetadata) error {
	if chunk == nil || chunk.ASTNode == nil || currentTable == nil || previousTable == nil {
		return nil
	}

	// Get the original chunk text
	originalChunkText := chunk.GetText()
	if originalChunkText == "" {
		return errors.New("chunk has no text content")
	}

	// Parse the individual chunk text to get a fresh AST with its own tokenStream
	parseResult, err := pgparser.ParsePostgreSQL(originalChunkText)
	if err != nil {
		return errors.Wrapf(err, "failed to parse original chunk text: %s", originalChunkText)
	}

	// Extract the CREATE TABLE AST node from the fresh parse
	var createStmt *parser.CreatestmtContext
	antlr.ParseTreeWalkerDefault.Walk(&createTableExtractor{
		result: &createStmt,
	}, parseResult.Tree)

	if createStmt == nil {
		return errors.New("failed to extract CREATE TABLE AST node from chunk text")
	}

	// Get the parser and tokenStream from the fresh parse
	ctxParser := createStmt.GetParser()
	if ctxParser == nil {
		return errors.New("parser not available for fresh AST node")
	}

	tokenStream := ctxParser.GetTokenStream()
	if tokenStream == nil {
		return errors.New("token stream not available for fresh parser")
	}

	// Create rewriter for the individual chunk's tokenStream
	rewriter := antlr.NewTokenStreamRewriter(tokenStream)

	// Apply column changes using the rewriter
	err = applyColumnChanges(rewriter, createStmt, currentTable, previousTable)
	if err != nil {
		return errors.Wrapf(err, "failed to apply column changes")
	}

	// Apply constraint changes using the same rewriter
	err = applyConstraintChanges(rewriter, createStmt, currentTable, previousTable)
	if err != nil {
		return errors.Wrapf(err, "failed to apply constraint changes")
	}

	// Get the modified SQL from the rewriter
	modifiedSQL := rewriter.GetTextDefault()

	// Parse the modified SQL to get the final AST
	finalParseResult, err := pgparser.ParsePostgreSQL(modifiedSQL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse modified SQL: %s", modifiedSQL)
	}

	// Extract the final CREATE TABLE AST node
	var newCreateTableNode *parser.CreatestmtContext
	antlr.ParseTreeWalkerDefault.Walk(&createTableExtractor{
		result: &newCreateTableNode,
	}, finalParseResult.Tree)

	if newCreateTableNode == nil {
		return errors.New("failed to extract CREATE TABLE AST node from modified text")
	}

	// Update the chunk with the new AST node
	chunk.ASTNode = newCreateTableNode

	return nil
}

// applyColumnChanges applies column changes using the provided rewriter without parsing SQL
func applyColumnChanges(rewriter *antlr.TokenStreamRewriter, createStmt *parser.CreatestmtContext, currentTable, previousTable *storepb.TableMetadata) error {
	// Create column maps for efficient lookups
	currentColumns := make(map[string]*storepb.ColumnMetadata)
	previousColumns := make(map[string]*storepb.ColumnMetadata)

	for _, col := range currentTable.Columns {
		currentColumns[col.Name] = col
	}
	for _, col := range previousTable.Columns {
		previousColumns[col.Name] = col
	}

	// Extract existing column definitions from AST
	existingColumnDefs := extractColumnDefinitionsWithAST(createStmt)

	// Phase 1: Handle column deletions (process in reverse order to maintain token positions)
	for i := len(existingColumnDefs.Order) - 1; i >= 0; i-- {
		columnName := existingColumnDefs.Order[i]
		if _, exists := currentColumns[columnName]; !exists {
			// Column was deleted
			columnDef := existingColumnDefs.Map[columnName]
			err := deleteColumnFromAST(rewriter, columnDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete column %s", columnName)
			}
		}
	}

	// Phase 2: Handle column modifications
	for _, columnName := range existingColumnDefs.Order {
		if currentCol, currentExists := currentColumns[columnName]; currentExists {
			if previousCol, previousExists := previousColumns[columnName]; previousExists {
				// Column exists in both - check if modified
				if !columnsEqual(currentCol, previousCol) {
					columnDef := existingColumnDefs.Map[columnName]
					err := modifyColumnInAST(rewriter, columnDef.ASTNode, currentCol)
					if err != nil {
						return errors.Wrapf(err, "failed to modify column %s", columnName)
					}
				}
			}
		}
	}

	// Phase 3: Handle column additions (add at the end)
	for _, currentCol := range currentTable.Columns {
		if _, exists := previousColumns[currentCol.Name]; !exists {
			// Column was added
			err := addColumnToAST(rewriter, createStmt, currentCol)
			if err != nil {
				return errors.Wrapf(err, "failed to add column %s", currentCol.Name)
			}
		}
	}

	return nil
}

// applyConstraintChanges applies constraint changes using the provided rewriter without parsing SQL
func applyConstraintChanges(rewriter *antlr.TokenStreamRewriter, createStmt *parser.CreatestmtContext, currentTable, previousTable *storepb.TableMetadata) error {
	// Create constraint maps for efficient lookups
	currentCheckConstraints := make(map[string]*storepb.CheckConstraintMetadata)
	previousCheckConstraints := make(map[string]*storepb.CheckConstraintMetadata)
	currentFKConstraints := make(map[string]*storepb.ForeignKeyMetadata)
	previousFKConstraints := make(map[string]*storepb.ForeignKeyMetadata)
	currentPKConstraints := make(map[string]*storepb.IndexMetadata)
	previousPKConstraints := make(map[string]*storepb.IndexMetadata)
	currentUKConstraints := make(map[string]*storepb.IndexMetadata)
	previousUKConstraints := make(map[string]*storepb.IndexMetadata)

	// Build constraint maps from metadata
	for _, constraint := range currentTable.CheckConstraints {
		currentCheckConstraints[constraint.Name] = constraint
	}
	for _, constraint := range previousTable.CheckConstraints {
		previousCheckConstraints[constraint.Name] = constraint
	}
	for _, constraint := range currentTable.ForeignKeys {
		currentFKConstraints[constraint.Name] = constraint
	}
	for _, constraint := range previousTable.ForeignKeys {
		previousFKConstraints[constraint.Name] = constraint
	}
	// Build primary key constraint maps
	for _, index := range currentTable.Indexes {
		if index.Primary {
			currentPKConstraints[index.Name] = index
		}
	}
	for _, index := range previousTable.Indexes {
		if index.Primary {
			previousPKConstraints[index.Name] = index
		}
	}
	// Build unique key constraint maps (unique constraints, not just indexes)
	for _, index := range currentTable.Indexes {
		if index.Unique && !index.Primary && index.IsConstraint {
			currentUKConstraints[index.Name] = index
		}
	}
	for _, index := range previousTable.Indexes {
		if index.Unique && !index.Primary && index.IsConstraint {
			previousUKConstraints[index.Name] = index
		}
	}

	// Extract constraint definitions with AST nodes for precise manipulation
	currentCheckDefs := extractCheckConstraintDefinitionsWithAST(createStmt)
	currentFKDefs := extractForeignKeyDefinitionsWithAST(createStmt)
	currentPKDefs := extractPrimaryKeyDefinitionsInOrder(createStmt)
	currentUKDefs := extractUniqueKeyDefinitionsInOrder(createStmt)

	// Phase 1: Handle constraint deletions (reverse order for stability)
	// Delete check constraints
	for i := len(currentCheckDefs) - 1; i >= 0; i-- {
		checkDef := currentCheckDefs[i]
		if _, exists := currentCheckConstraints[checkDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, checkDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete check constraint %s", checkDef.Name)
			}
		}
	}

	// Delete foreign key constraints
	for i := len(currentFKDefs) - 1; i >= 0; i-- {
		fkDef := currentFKDefs[i]
		if _, exists := currentFKConstraints[fkDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, fkDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete foreign key constraint %s", fkDef.Name)
			}
		}
	}

	// Delete primary key constraints
	for i := len(currentPKDefs) - 1; i >= 0; i-- {
		pkDef := currentPKDefs[i]
		if _, exists := currentPKConstraints[pkDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, pkDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete primary key constraint %s", pkDef.Name)
			}
		}
	}

	// Delete unique key constraints
	for i := len(currentUKDefs) - 1; i >= 0; i-- {
		ukDef := currentUKDefs[i]
		if _, exists := currentUKConstraints[ukDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, ukDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete unique key constraint %s", ukDef.Name)
			}
		}
	}

	// Phase 2: Handle constraint modifications
	// Modify check constraints
	for _, checkDef := range currentCheckDefs {
		if currentConstraint, exists := currentCheckConstraints[checkDef.Name]; exists {
			if previousConstraint, wasPresent := previousCheckConstraints[checkDef.Name]; wasPresent {
				// Check if constraint was modified by comparing text
				if !constraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, checkDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify check constraint %s", checkDef.Name)
					}
				}
			}
		}
	}

	// Modify foreign key constraints
	for _, fkDef := range currentFKDefs {
		if currentConstraint, exists := currentFKConstraints[fkDef.Name]; exists {
			if previousConstraint, wasPresent := previousFKConstraints[fkDef.Name]; wasPresent {
				// Check if constraint was modified
				if !fkConstraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, fkDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify foreign key constraint %s", fkDef.Name)
					}
				}
			}
		}
	}

	// Modify primary key constraints
	for _, pkDef := range currentPKDefs {
		if currentConstraint, exists := currentPKConstraints[pkDef.Name]; exists {
			if previousConstraint, wasPresent := previousPKConstraints[pkDef.Name]; wasPresent {
				// Check if constraint was modified
				if !pkConstraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, pkDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify primary key constraint %s", pkDef.Name)
					}
				}
			}
		}
	}

	// Modify unique key constraints
	for _, ukDef := range currentUKDefs {
		if currentConstraint, exists := currentUKConstraints[ukDef.Name]; exists {
			if previousConstraint, wasPresent := previousUKConstraints[ukDef.Name]; wasPresent {
				// Check if constraint was modified
				if !ukConstraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, ukDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify unique key constraint %s", ukDef.Name)
					}
				}
			}
		}
	}

	// Phase 3: Handle constraint additions
	// Add new check constraints
	for _, currentConstraint := range currentTable.CheckConstraints {
		if _, existed := previousCheckConstraints[currentConstraint.Name]; !existed {
			// New check constraint
			err := addConstraintToAST(rewriter, createStmt, currentConstraint)
			if err != nil {
				return errors.Wrapf(err, "failed to add check constraint %s", currentConstraint.Name)
			}
		}
	}

	// Add new foreign key constraints
	for _, currentConstraint := range currentTable.ForeignKeys {
		if _, existed := previousFKConstraints[currentConstraint.Name]; !existed {
			// New foreign key constraint
			err := addConstraintToAST(rewriter, createStmt, currentConstraint)
			if err != nil {
				return errors.Wrapf(err, "failed to add foreign key constraint %s", currentConstraint.Name)
			}
		}
	}

	// Add new primary key constraints
	for _, currentIndex := range currentTable.Indexes {
		if currentIndex.Primary {
			if _, existed := previousPKConstraints[currentIndex.Name]; !existed {
				// New primary key constraint
				err := addConstraintToAST(rewriter, createStmt, currentIndex)
				if err != nil {
					return errors.Wrapf(err, "failed to add primary key constraint %s", currentIndex.Name)
				}
			}
		}
	}

	// Add new unique key constraints
	for _, currentIndex := range currentTable.Indexes {
		if currentIndex.Unique && !currentIndex.Primary && currentIndex.IsConstraint {
			if _, existed := previousUKConstraints[currentIndex.Name]; !existed {
				// New unique key constraint
				err := addConstraintToAST(rewriter, createStmt, currentIndex)
				if err != nil {
					return errors.Wrapf(err, "failed to add unique key constraint %s", currentIndex.Name)
				}
			}
		}
	}

	return nil
}

// deleteColumnFromAST removes a column definition from the CREATE TABLE statement using token rewriter
// Improved comma handling: always look for a following comma first, regardless of column position
func deleteColumnFromAST(rewriter *antlr.TokenStreamRewriter, columnDef parser.IColumnDefContext, _ *parser.CreatestmtContext) error {
	if columnDef == nil {
		return errors.New("column definition is nil")
	}

	startToken := columnDef.GetStart()
	stopToken := columnDef.GetStop()
	if startToken == nil || stopToken == nil {
		return errors.New("unable to get column definition tokens")
	}

	// Find the actual deletion range including commas and whitespace
	deleteStartIndex := startToken.GetTokenIndex()
	deleteEndIndex := stopToken.GetTokenIndex()

	// Strategy: Always try to remove the following comma first
	// This handles cases where there are table constraints after columns
	nextCommaIndex := -1
	nextCommaEndIndex := -1
	for i := stopToken.GetTokenIndex() + 1; i < rewriter.GetTokenStream().Size(); i++ {
		token := rewriter.GetTokenStream().Get(i)
		if token.GetTokenType() == parser.PostgreSQLParserCOMMA {
			nextCommaIndex = i
			// Look for whitespace after the comma to include in deletion
			nextCommaEndIndex = i
			for j := i + 1; j < rewriter.GetTokenStream().Size(); j++ {
				nextToken := rewriter.GetTokenStream().Get(j)
				// Include whitespace and newlines after the comma
				if nextToken.GetChannel() != antlr.TokenDefaultChannel {
					nextCommaEndIndex = j
				} else {
					break
				}
			}
			break
		}
		// Skip whitespace and comments, but stop at other meaningful tokens
		if token.GetChannel() == antlr.TokenDefaultChannel {
			// Found a non-comma token on the default channel, stop searching
			break
		}
	}

	if nextCommaIndex != -1 {
		// Found a following comma - remove column, comma, and trailing whitespace
		deleteEndIndex = nextCommaEndIndex
	} else {
		// No following comma found - this might be the last element
		// Try to find a preceding comma to remove
		prevCommaIndex := -1
		prevCommaStartIndex := -1
		for i := startToken.GetTokenIndex() - 1; i >= 0; i-- {
			token := rewriter.GetTokenStream().Get(i)
			if token.GetTokenType() == parser.PostgreSQLParserCOMMA {
				prevCommaIndex = i
				// Look for whitespace before the comma to include in deletion
				prevCommaStartIndex = i
				for j := i - 1; j >= 0; j-- {
					prevToken := rewriter.GetTokenStream().Get(j)
					// Include whitespace and newlines before the comma
					if prevToken.GetChannel() != antlr.TokenDefaultChannel {
						prevCommaStartIndex = j
					} else {
						break
					}
				}
				break
			}
			// Skip whitespace and comments, but stop at other meaningful tokens
			if token.GetChannel() == antlr.TokenDefaultChannel {
				break
			}
		}

		if prevCommaIndex != -1 {
			// Found a preceding comma - remove it along with leading whitespace and the column
			deleteStartIndex = prevCommaStartIndex
		} else {
			// No comma found (single column case) - just remove the column
			// But also clean up any trailing whitespace that might leave empty lines
			for i := stopToken.GetTokenIndex() + 1; i < rewriter.GetTokenStream().Size(); i++ {
				token := rewriter.GetTokenStream().Get(i)
				if token.GetChannel() != antlr.TokenDefaultChannel {
					deleteEndIndex = i
				} else {
					break
				}
			}
		}
	}

	// Perform the deletion with the computed range
	rewriter.DeleteDefault(deleteStartIndex, deleteEndIndex)

	return nil
}

// modifyColumnInAST modifies an existing column definition using token rewriter
func modifyColumnInAST(rewriter *antlr.TokenStreamRewriter, columnDef parser.IColumnDefContext, newColumn *storepb.ColumnMetadata) error {
	if columnDef == nil || newColumn == nil {
		return errors.New("column definition or new column metadata is nil")
	}

	startToken := columnDef.GetStart()
	stopToken := columnDef.GetStop()
	if startToken == nil || stopToken == nil {
		return errors.New("unable to get column definition tokens")
	}

	// Generate new column definition SDL
	newColumnSDL := generateColumnSDL(newColumn)

	// Replace the entire column definition
	rewriter.ReplaceDefault(startToken.GetTokenIndex(), stopToken.GetTokenIndex(), newColumnSDL)

	return nil
}

// addColumnToAST adds a new column definition to the CREATE TABLE statement
func addColumnToAST(rewriter *antlr.TokenStreamRewriter, createStmt *parser.CreatestmtContext, newColumn *storepb.ColumnMetadata) error {
	if createStmt == nil || newColumn == nil {
		return errors.New("create statement or new column metadata is nil")
	}

	// Find the last column definition to insert after it
	columnDefs := extractColumnDefinitionsWithAST(createStmt)

	if len(columnDefs.Order) == 0 {
		// No existing columns - need to add first column to empty table
		// Find the opening parenthesis and insert after it
		if createStmt.Opttableelementlist() != nil {
			// Table has element list structure, find the position to insert
			for i := 0; i < rewriter.GetTokenStream().Size(); i++ {
				token := rewriter.GetTokenStream().Get(i)
				if token.GetTokenType() == parser.PostgreSQLParserOPEN_PAREN {
					// Found opening parenthesis, insert new column after it
					newColumnSDL := "\n    " + generateColumnSDL(newColumn) + "\n"
					rewriter.InsertAfterDefault(i, newColumnSDL)
					return nil
				}
			}
		}
		return errors.New("unable to find position to insert first column")
	}

	// Get the last column definition
	lastColumnName := columnDefs.Order[len(columnDefs.Order)-1]
	lastColumnDef := columnDefs.Map[lastColumnName]

	stopToken := lastColumnDef.ASTNode.GetStop()
	if stopToken == nil {
		return errors.New("unable to get last column stop token")
	}

	// Generate new column definition SDL with leading comma and proper indentation
	newColumnSDL := ",\n    " + generateColumnSDL(newColumn)

	// Insert after the last column
	rewriter.InsertAfterDefault(stopToken.GetTokenIndex(), newColumnSDL)

	return nil
}

// generateColumnSDL generates SDL text for a single column definition using the extracted writeColumnSDL function
func generateColumnSDL(column *storepb.ColumnMetadata) string {
	if column == nil {
		return ""
	}

	var buf strings.Builder
	err := writeColumnSDL(&buf, column)
	if err != nil {
		// If there's an error writing to the buffer, return empty string
		// This should rarely happen since we're writing to a strings.Builder
		return ""
	}

	return buf.String()
}

// columnsEqual compares two column metadata objects for equality
func columnsEqual(a, b *storepb.ColumnMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Name == b.Name &&
		a.Type == b.Type &&
		a.Nullable == b.Nullable &&
		a.Default == b.Default &&
		a.Collation == b.Collation
}

// currentDatabaseSDLChunks stores pre-computed SDL chunks from current database metadata
// for performance optimization during usability checks
type currentDatabaseSDLChunks struct {
	chunks      map[string]string // maps chunk identifier to normalized SDL text from current database metadata
	columns     map[string]string // maps "schema.table.column" to normalized column SDL text
	constraints map[string]string // maps "schema.table.constraint" to normalized constraint SDL text
}

// buildCurrentDatabaseSDLChunks pre-computes SDL chunks from the current database schema
// for usability checks. This avoids repeated expensive calls to convertDatabaseSchemaToSDL
// and ChunkSDLText during diff processing by storing normalized SDL text from current database metadata.
func buildCurrentDatabaseSDLChunks(currentSchema *model.DatabaseSchema) (*currentDatabaseSDLChunks, error) {
	sdlChunks := &currentDatabaseSDLChunks{
		chunks:      make(map[string]string),
		columns:     make(map[string]string),
		constraints: make(map[string]string),
	}

	// Only build SDL chunks if current schema is provided
	if currentSchema == nil {
		return sdlChunks, nil
	}

	// Generate SDL from current database metadata
	currentSDL, err := convertDatabaseSchemaToSDL(currentSchema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert current schema to SDL")
	}

	// Parse the generated SDL to get chunks
	currentSDLChunks, err := ChunkSDLText(currentSDL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk current schema SDL")
	}

	// Populate SDL chunks with normalized chunk texts from current database metadata
	for identifier, chunk := range currentSDLChunks.Tables {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetText())

		// Extract column and constraint SDL texts for fine-grained usability checks
		if err := extractColumnAndConstraintSDLTexts(chunk, identifier, sdlChunks); err != nil {
			// Log error but don't fail the entire operation
			// Fine-grained usability checks will fall back to table-level checks
			continue
		}
	}
	for identifier, chunk := range currentSDLChunks.Views {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetText())
	}
	for identifier, chunk := range currentSDLChunks.Functions {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetText())
	}
	for identifier, chunk := range currentSDLChunks.Sequences {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetText())
	}
	for identifier, chunk := range currentSDLChunks.Indexes {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetText())
	}

	return sdlChunks, nil
}

// extractColumnAndConstraintSDLTexts extracts individual column and constraint SDL texts
// from a table chunk for fine-grained usability checks
func extractColumnAndConstraintSDLTexts(chunk *schema.SDLChunk, tableIdentifier string, sdlChunks *currentDatabaseSDLChunks) error {
	if chunk == nil || chunk.ASTNode == nil {
		return nil
	}

	createStmt, ok := chunk.ASTNode.(*parser.CreatestmtContext)
	if !ok {
		return errors.New("chunk AST node is not a CREATE TABLE statement")
	}

	schemaName, tableName := parseIdentifier(tableIdentifier)

	// Extract column definitions with their SDL texts
	columnDefs := extractColumnDefinitionsWithAST(createStmt)
	for columnName, columnDef := range columnDefs.Map {
		columnSDLText := strings.TrimSpace(getColumnText(columnDef.ASTNode))
		columnKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, columnName)
		sdlChunks.columns[columnKey] = columnSDLText
	}

	// Extract constraint definitions with their SDL texts
	// Primary Key constraints
	pkDefs := extractPrimaryKeyDefinitionsWithAST(createStmt)
	for constraintName, pkDef := range pkDefs {
		constraintSDLText := strings.TrimSpace(getIndexText(pkDef.ASTNode))
		constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, constraintName)
		sdlChunks.constraints[constraintKey] = constraintSDLText
	}

	// Unique constraints
	uniqueDefs := extractUniqueConstraintDefinitionsInOrder(createStmt)
	for _, uniqueDef := range uniqueDefs {
		constraintSDLText := strings.TrimSpace(getIndexText(uniqueDef.ASTNode))
		constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, uniqueDef.Name)
		sdlChunks.constraints[constraintKey] = constraintSDLText
	}

	// Check constraints
	checkDefs := extractCheckConstraintDefinitionsWithAST(createStmt)
	for _, checkDef := range checkDefs {
		constraintSDLText := strings.TrimSpace(getCheckConstraintText(checkDef.ASTNode))
		constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, checkDef.Name)
		sdlChunks.constraints[constraintKey] = constraintSDLText
	}

	// Foreign Key constraints
	fkDefs := extractForeignKeyDefinitionsWithAST(createStmt)
	for _, fkDef := range fkDefs {
		constraintSDLText := strings.TrimSpace(getForeignKeyText(fkDef.ASTNode))
		constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, fkDef.Name)
		sdlChunks.constraints[constraintKey] = constraintSDLText
	}

	return nil
}

// shouldSkipChunkDiffForUsability checks if a chunk should skip diff comparison
// by comparing against the pre-computed SDL chunks from current database metadata
func (sdlChunks *currentDatabaseSDLChunks) shouldSkipChunkDiffForUsability(chunkText string, chunkIdentifier string) bool {
	if sdlChunks == nil || len(sdlChunks.chunks) == 0 {
		return false
	}

	// Get the corresponding SDL text from current database metadata
	currentDatabaseSDLText, exists := sdlChunks.chunks[chunkIdentifier]
	if !exists {
		return false
	}

	// Normalize current chunk text for comparison
	normalizedChunkText := strings.TrimSpace(chunkText)

	// If chunk text matches current database metadata SDL, skip the diff (no actual change needed)
	return normalizedChunkText == currentDatabaseSDLText
}

// shouldSkipColumnDiffForUsability checks if a column should skip diff comparison
// by comparing against the pre-computed column SDL texts from current database metadata
func (sdlChunks *currentDatabaseSDLChunks) shouldSkipColumnDiffForUsability(columnText string, schemaName, tableName, columnName string) bool {
	if sdlChunks == nil || len(sdlChunks.columns) == 0 {
		return false
	}

	// Build column key
	columnKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, columnName)

	// Get the corresponding SDL text from current database metadata
	currentDatabaseColumnSDL, exists := sdlChunks.columns[columnKey]
	if !exists {
		return false
	}

	// Normalize current column text for comparison
	normalizedColumnText := strings.TrimSpace(columnText)

	// If column text matches current database metadata SDL, skip the diff (no actual change needed)
	return normalizedColumnText == currentDatabaseColumnSDL
}

// shouldSkipConstraintDiffForUsability checks if a constraint should skip diff comparison
// by comparing against the pre-computed constraint SDL texts from current database metadata
func (sdlChunks *currentDatabaseSDLChunks) shouldSkipConstraintDiffForUsability(constraintText string, schemaName, tableName, constraintName string) bool {
	if sdlChunks == nil || len(sdlChunks.constraints) == 0 {
		return false
	}

	// Build constraint key
	constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, constraintName)

	// Get the corresponding SDL text from current database metadata
	currentDatabaseConstraintSDL, exists := sdlChunks.constraints[constraintKey]
	if !exists {
		return false
	}

	// Normalize current constraint text for comparison
	normalizedConstraintText := strings.TrimSpace(constraintText)

	// If constraint text matches current database metadata SDL, skip the diff (no actual change needed)
	return normalizedConstraintText == currentDatabaseConstraintSDL
}

// convertDatabaseSchemaToSDL converts a model.DatabaseSchema to SDL format string
// This is used in initialization scenarios where previousUserSDLText is empty
func convertDatabaseSchemaToSDL(dbSchema *model.DatabaseSchema) (string, error) {
	if dbSchema == nil {
		return "", nil
	}

	metadata := dbSchema.GetMetadata()
	if metadata == nil {
		return "", nil
	}

	// Use the existing getSDLFormat function from get_database_definition.go
	return getSDLFormat(metadata)
}

// deleteConstraintFromAST removes a constraint definition from the CREATE TABLE statement using token rewriter
func deleteConstraintFromAST(rewriter *antlr.TokenStreamRewriter, constraintAST parser.ITableconstraintContext, _ *parser.CreatestmtContext) error {
	if constraintAST == nil || rewriter == nil {
		return errors.New("constraint AST or rewriter is nil")
	}

	// Get start and stop tokens for the constraint
	startToken := constraintAST.GetStart()
	stopToken := constraintAST.GetStop()

	if startToken == nil || stopToken == nil {
		return errors.New("unable to get constraint definition tokens")
	}

	// Find the actual deletion range including commas and whitespace
	deleteStartIndex := startToken.GetTokenIndex()
	deleteEndIndex := stopToken.GetTokenIndex()

	// Strategy: Always try to remove the following comma first
	// This handles cases where there are more constraints after this one
	nextCommaIndex := -1
	nextCommaEndIndex := -1
	for i := stopToken.GetTokenIndex() + 1; i < rewriter.GetTokenStream().Size(); i++ {
		token := rewriter.GetTokenStream().Get(i)
		if token.GetTokenType() == parser.PostgreSQLParserCOMMA {
			nextCommaIndex = i
			// Look for whitespace after the comma to include in deletion
			nextCommaEndIndex = i
			for j := i + 1; j < rewriter.GetTokenStream().Size(); j++ {
				nextToken := rewriter.GetTokenStream().Get(j)
				// Include whitespace and newlines after the comma
				if nextToken.GetChannel() != antlr.TokenDefaultChannel {
					nextCommaEndIndex = j
				} else {
					break
				}
			}
			break
		}
		// Skip whitespace and comments, but stop at other meaningful tokens
		if token.GetChannel() == antlr.TokenDefaultChannel {
			// Found a non-comma token on the default channel, stop searching
			break
		}
	}

	if nextCommaIndex != -1 {
		// Found a following comma - remove constraint, comma, and trailing whitespace
		deleteEndIndex = nextCommaEndIndex
	} else {
		// No following comma found - this might be the last element
		// Try to find a preceding comma to remove
		prevCommaIndex := -1
		prevCommaStartIndex := -1
		for i := startToken.GetTokenIndex() - 1; i >= 0; i-- {
			token := rewriter.GetTokenStream().Get(i)
			if token.GetTokenType() == parser.PostgreSQLParserCOMMA {
				prevCommaIndex = i
				// Look for whitespace before the comma to include in deletion
				prevCommaStartIndex = i
				for j := i - 1; j >= 0; j-- {
					prevToken := rewriter.GetTokenStream().Get(j)
					// Include whitespace and newlines before the comma
					if prevToken.GetChannel() != antlr.TokenDefaultChannel {
						prevCommaStartIndex = j
					} else {
						break
					}
				}
				break
			}
			// Skip whitespace and comments, but stop at other meaningful tokens
			if token.GetChannel() == antlr.TokenDefaultChannel {
				break
			}
		}

		if prevCommaIndex != -1 {
			// Found a preceding comma - remove it along with leading whitespace and the constraint
			deleteStartIndex = prevCommaStartIndex
		} else {
			// No comma found (single constraint case) - just remove the constraint
			// But also clean up any trailing whitespace that might leave empty lines
			for i := stopToken.GetTokenIndex() + 1; i < rewriter.GetTokenStream().Size(); i++ {
				token := rewriter.GetTokenStream().Get(i)
				if token.GetChannel() != antlr.TokenDefaultChannel {
					deleteEndIndex = i
				} else {
					break
				}
			}
		}
	}

	// Perform the deletion with the computed range
	rewriter.DeleteDefault(deleteStartIndex, deleteEndIndex)

	return nil
}

// modifyConstraintInAST modifies a constraint definition using token rewriter
func modifyConstraintInAST(rewriter *antlr.TokenStreamRewriter, constraintAST parser.ITableconstraintContext, newConstraint any) error {
	if constraintAST == nil || rewriter == nil || newConstraint == nil {
		return errors.New("constraint AST, rewriter, or new constraint is nil")
	}

	// Get start and stop tokens for the constraint
	startToken := constraintAST.GetStart()
	stopToken := constraintAST.GetStop()

	if startToken == nil || stopToken == nil {
		return errors.New("unable to get constraint definition tokens")
	}

	// Generate the new constraint SDL
	var newConstraintSDL string
	switch constraint := newConstraint.(type) {
	case *storepb.CheckConstraintMetadata:
		newConstraintSDL = generateCheckConstraintSDL(constraint)
	case *storepb.ForeignKeyMetadata:
		newConstraintSDL = generateForeignKeyConstraintSDL(constraint)
	case *storepb.IndexMetadata:
		if constraint.Primary {
			newConstraintSDL = generatePrimaryKeyConstraintSDL(constraint)
		} else if constraint.Unique && constraint.IsConstraint {
			newConstraintSDL = generateUniqueKeyConstraintSDL(constraint)
		} else {
			return errors.New("unsupported index constraint type")
		}
	default:
		return errors.New("unsupported constraint type")
	}

	// Replace the entire constraint definition
	rewriter.ReplaceDefault(startToken.GetTokenIndex(), stopToken.GetTokenIndex(), newConstraintSDL)

	return nil
}

// addConstraintToAST adds a new constraint to the CREATE TABLE statement using token rewriter
func addConstraintToAST(rewriter *antlr.TokenStreamRewriter, createStmt *parser.CreatestmtContext, newConstraint any) error {
	if rewriter == nil || createStmt == nil || newConstraint == nil {
		return errors.New("rewriter, create statement, or new constraint is nil")
	}

	// Generate the new constraint SDL
	var newConstraintSDL string
	switch constraint := newConstraint.(type) {
	case *storepb.CheckConstraintMetadata:
		newConstraintSDL = generateCheckConstraintSDL(constraint)
	case *storepb.ForeignKeyMetadata:
		newConstraintSDL = generateForeignKeyConstraintSDL(constraint)
	case *storepb.IndexMetadata:
		if constraint.Primary {
			newConstraintSDL = generatePrimaryKeyConstraintSDL(constraint)
		} else if constraint.Unique && constraint.IsConstraint {
			newConstraintSDL = generateUniqueKeyConstraintSDL(constraint)
		} else {
			return errors.New("unsupported index constraint type")
		}
	default:
		return errors.New("unsupported constraint type")
	}

	// Find the position to insert the constraint
	// Look for the closing parenthesis of the CREATE TABLE statement
	optTableElementList := createStmt.Opttableelementlist()
	if optTableElementList == nil {
		return errors.New("CREATE TABLE statement has no table element list")
	}

	tableElementList := optTableElementList.Tableelementlist()
	if tableElementList == nil {
		return errors.New("table element list is nil")
	}

	// Get all table elements to find the last one
	tableElements := tableElementList.AllTableelement()
	if len(tableElements) == 0 {
		return errors.New("no table elements found")
	}

	// Get the last element (could be a column or constraint)
	lastElement := tableElements[len(tableElements)-1]
	stopToken := lastElement.GetStop()
	if stopToken == nil {
		return errors.New("unable to get last table element stop token")
	}

	// Generate constraint definition SDL with leading comma and proper indentation
	constraintSDL := ",\n    " + newConstraintSDL

	// Insert after the last element
	rewriter.InsertAfterDefault(stopToken.GetTokenIndex(), constraintSDL)

	return nil
}

// generateCheckConstraintSDL generates SDL text for a check constraint using the existing writeCheckConstraintSDL function
func generateCheckConstraintSDL(constraint *storepb.CheckConstraintMetadata) string {
	if constraint == nil {
		return ""
	}

	var buf strings.Builder
	err := writeCheckConstraintSDL(&buf, constraint)
	if err != nil {
		// If there's an error writing to the buffer, return empty string
		// This should rarely happen since we're writing to a strings.Builder
		return ""
	}

	return buf.String()
}

// generateForeignKeyConstraintSDL generates SDL text for a foreign key constraint using the existing writeForeignKeyConstraintSDL function
func generateForeignKeyConstraintSDL(constraint *storepb.ForeignKeyMetadata) string {
	if constraint == nil {
		return ""
	}

	var buf strings.Builder
	err := writeForeignKeyConstraintSDL(&buf, constraint)
	if err != nil {
		// If there's an error writing to the buffer, return empty string
		// This should rarely happen since we're writing to a strings.Builder
		return ""
	}

	return buf.String()
}

// constraintsEqual compares two check constraint metadata objects for equality
func constraintsEqual(a, b *storepb.CheckConstraintMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Name == b.Name && a.Expression == b.Expression
}

// fkConstraintsEqual compares two foreign key constraint metadata objects for equality
func fkConstraintsEqual(a, b *storepb.ForeignKeyMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Compare basic properties
	if a.Name != b.Name ||
		a.ReferencedSchema != b.ReferencedSchema ||
		a.ReferencedTable != b.ReferencedTable ||
		a.OnDelete != b.OnDelete ||
		a.OnUpdate != b.OnUpdate {
		return false
	}

	// Compare columns arrays
	if len(a.Columns) != len(b.Columns) ||
		len(a.ReferencedColumns) != len(b.ReferencedColumns) {
		return false
	}

	for i := range a.Columns {
		if a.Columns[i] != b.Columns[i] {
			return false
		}
	}

	for i := range a.ReferencedColumns {
		if a.ReferencedColumns[i] != b.ReferencedColumns[i] {
			return false
		}
	}

	return true
}

// extractCheckConstraintDefinitionsWithAST extracts check constraint definitions with their AST nodes
// Note: This is a wrapper around the existing function with a different name for clarity
func extractCheckConstraintDefinitionsWithAST(createStmt *parser.CreatestmtContext) []*CheckConstraintDefWithAST {
	return extractCheckConstraintDefinitionsInOrder(createStmt)
}

// extractForeignKeyDefinitionsWithAST extracts foreign key constraint definitions with their AST nodes
// Note: This is a wrapper around the existing function with a different name for clarity
func extractForeignKeyDefinitionsWithAST(createStmt *parser.CreatestmtContext) []*ForeignKeyDefWithAST {
	return extractForeignKeyDefinitionsInOrder(createStmt)
}

// PrimaryKeyDefWithAST represents a primary key constraint definition with its AST node
type PrimaryKeyDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// UniqueKeyDefWithAST represents a unique key constraint definition with its AST node
type UniqueKeyDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// extractPrimaryKeyDefinitionsInOrder extracts primary key constraint definitions with their AST nodes
func extractPrimaryKeyDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*PrimaryKeyDefWithAST {
	var pkDefs []*PrimaryKeyDefWithAST

	if createStmt == nil {
		return pkDefs
	}

	// Navigate through the CREATE TABLE AST to find table constraints
	optTableElementList := createStmt.Opttableelementlist()
	if optTableElementList == nil {
		return pkDefs
	}

	tableElementList := optTableElementList.(*parser.OpttableelementlistContext).Tableelementlist()
	if tableElementList == nil {
		return pkDefs
	}

	// Iterate through table elements to find constraints
	for _, element := range tableElementList.(*parser.TableelementlistContext).AllTableelement() {
		if tableConstraint := element.(*parser.TableelementContext).Tableconstraint(); tableConstraint != nil {
			constraint, ok := tableConstraint.(*parser.TableconstraintContext)
			if !ok {
				continue
			}

			// Check if it's a primary key constraint
			if constraint.Constraintelem() != nil {
				elem, ok := constraint.Constraintelem().(*parser.ConstraintelemContext)
				if !ok {
					continue
				}
				if elem.PRIMARY() != nil && elem.KEY() != nil {
					// Extract constraint name
					constraintName := ""
					if constraint.Name() != nil {
						constraintName = pgparser.NormalizePostgreSQLName(constraint.Name())
					}

					if constraintName != "" {
						pkDefs = append(pkDefs, &PrimaryKeyDefWithAST{
							Name:    constraintName,
							ASTNode: constraint,
						})
					}
				}
			}
		}
	}

	return pkDefs
}

// extractUniqueKeyDefinitionsInOrder extracts unique key constraint definitions with their AST nodes
func extractUniqueKeyDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*UniqueKeyDefWithAST {
	var ukDefs []*UniqueKeyDefWithAST

	if createStmt == nil {
		return ukDefs
	}

	// Navigate through the CREATE TABLE AST to find table constraints
	optTableElementList := createStmt.Opttableelementlist()
	if optTableElementList == nil {
		return ukDefs
	}

	tableElementList := optTableElementList.(*parser.OpttableelementlistContext).Tableelementlist()
	if tableElementList == nil {
		return ukDefs
	}

	// Iterate through table elements to find constraints
	for _, element := range tableElementList.(*parser.TableelementlistContext).AllTableelement() {
		if tableConstraint := element.(*parser.TableelementContext).Tableconstraint(); tableConstraint != nil {
			constraint, ok := tableConstraint.(*parser.TableconstraintContext)
			if !ok {
				continue
			}

			// Check if it's a unique constraint
			if constraint.Constraintelem() != nil {
				elem, ok := constraint.Constraintelem().(*parser.ConstraintelemContext)
				if !ok {
					continue
				}
				if elem.UNIQUE() != nil {
					// Extract constraint name
					constraintName := ""
					if constraint.Name() != nil {
						constraintName = pgparser.NormalizePostgreSQLName(constraint.Name())
					}

					if constraintName != "" {
						ukDefs = append(ukDefs, &UniqueKeyDefWithAST{
							Name:    constraintName,
							ASTNode: constraint,
						})
					}
				}
			}
		}
	}

	return ukDefs
}

// pkConstraintsEqual checks if two primary key constraints are equal
func pkConstraintsEqual(a, b *storepb.IndexMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Compare basic properties
	if a.Name != b.Name || a.Primary != b.Primary {
		return false
	}

	// Compare expressions (columns)
	if len(a.Expressions) != len(b.Expressions) {
		return false
	}

	for i := range a.Expressions {
		if a.Expressions[i] != b.Expressions[i] {
			return false
		}
	}

	return true
}

// ukConstraintsEqual checks if two unique key constraints are equal
func ukConstraintsEqual(a, b *storepb.IndexMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Compare basic properties
	if a.Name != b.Name || a.Unique != b.Unique || a.IsConstraint != b.IsConstraint {
		return false
	}

	// Compare expressions (columns)
	if len(a.Expressions) != len(b.Expressions) {
		return false
	}

	for i := range a.Expressions {
		if a.Expressions[i] != b.Expressions[i] {
			return false
		}
	}

	return true
}

// generatePrimaryKeyConstraintSDL generates SDL text for a primary key constraint
func generatePrimaryKeyConstraintSDL(constraint *storepb.IndexMetadata) string {
	if constraint == nil || !constraint.Primary {
		return ""
	}

	var buf strings.Builder
	err := writePrimaryKeyConstraintSDL(&buf, constraint)
	if err != nil {
		return ""
	}
	return buf.String()
}

// generateUniqueKeyConstraintSDL generates SDL text for a unique key constraint
func generateUniqueKeyConstraintSDL(constraint *storepb.IndexMetadata) string {
	if constraint == nil || !constraint.Unique || constraint.Primary || !constraint.IsConstraint {
		return ""
	}

	var buf strings.Builder
	err := writeUniqueKeyConstraintSDL(&buf, constraint)
	if err != nil {
		return ""
	}
	return buf.String()
}

// extendedIndexMetadata stores index metadata with table and schema context
type extendedIndexMetadata struct {
	*storepb.IndexMetadata
	SchemaName string
	TableName  string
}

// applyStandaloneIndexChangesToChunks applies minimal changes to standalone CREATE INDEX chunks
// This handles creation, modification, and deletion of independent index statements
func applyStandaloneIndexChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get index differences by comparing schema metadata
	currentMetadata := currentSchema.GetMetadata()
	previousMetadata := previousSchema.GetMetadata()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build index maps for current and previous schemas
	currentIndexes := make(map[string]*extendedIndexMetadata)
	previousIndexes := make(map[string]*extendedIndexMetadata)

	// Collect all standalone indexes from current schema (only non-constraint indexes)
	for _, schema := range currentMetadata.Schemas {
		for _, table := range schema.Tables {
			for _, index := range table.Indexes {
				// Only include standalone indexes (not constraints like PRIMARY KEY, UNIQUE CONSTRAINT)
				if !index.IsConstraint && !index.Primary {
					indexKey := formatIndexKey(schema.Name, index.Name)
					// Store extended index metadata with table/schema context
					extendedIndex := &extendedIndexMetadata{
						IndexMetadata: index,
						SchemaName:    schema.Name,
						TableName:     table.Name,
					}
					currentIndexes[indexKey] = extendedIndex
				}
			}
		}
	}

	// Collect all standalone indexes from previous schema
	for _, schema := range previousMetadata.Schemas {
		for _, table := range schema.Tables {
			for _, index := range table.Indexes {
				// Only include standalone indexes (not constraints)
				if !index.IsConstraint && !index.Primary {
					indexKey := formatIndexKey(schema.Name, index.Name)
					extendedIndex := &extendedIndexMetadata{
						IndexMetadata: index,
						SchemaName:    schema.Name,
						TableName:     table.Name,
					}
					previousIndexes[indexKey] = extendedIndex
				}
			}
		}
	}

	// Process index additions: create new index chunks
	for indexKey, currentIndex := range currentIndexes {
		if _, exists := previousIndexes[indexKey]; !exists {
			// New index - create a chunk for it
			err := createIndexChunk(previousChunks, currentIndex, indexKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create index chunk for %s", indexKey)
			}
		}
	}

	// Process index modifications: update existing chunks
	for indexKey, currentIndex := range currentIndexes {
		if previousIndex, exists := previousIndexes[indexKey]; exists {
			// Index exists in both - check if it needs modification
			err := updateIndexChunkIfNeeded(previousChunks, currentIndex, previousIndex, indexKey)
			if err != nil {
				return errors.Wrapf(err, "failed to update index chunk for %s", indexKey)
			}
		}
	}

	// Process index deletions: remove dropped index chunks
	for indexKey := range previousIndexes {
		if _, exists := currentIndexes[indexKey]; !exists {
			// Index was dropped - remove it from chunks
			deleteIndexChunk(previousChunks, indexKey)
		}
	}

	return nil
}

// formatIndexKey creates a consistent key for index identification
func formatIndexKey(schemaName, indexName string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	return schemaName + "." + indexName
}

// createIndexChunk creates a new CREATE INDEX chunk and adds it to the chunks
func createIndexChunk(chunks *schema.SDLChunks, extIndex *extendedIndexMetadata, indexKey string) error {
	if extIndex == nil || extIndex.IndexMetadata == nil || chunks == nil {
		return nil
	}

	// Generate SDL text for the index
	indexSDL := generateCreateIndexSDL(extIndex)
	if indexSDL == "" {
		return errors.New("failed to generate SDL for index")
	}

	// Parse the SDL to get AST node
	parseResult, err := pgparser.ParsePostgreSQL(indexSDL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse generated index SDL: %s", indexSDL)
	}

	// Extract the CREATE INDEX AST node
	var indexASTNode *parser.IndexstmtContext
	antlr.ParseTreeWalkerDefault.Walk(&indexExtractor{
		result: &indexASTNode,
	}, parseResult.Tree)

	if indexASTNode == nil {
		return errors.New("failed to extract CREATE INDEX AST node")
	}

	// Create and add the chunk
	chunk := &schema.SDLChunk{
		Identifier: indexKey,
		ASTNode:    indexASTNode,
	}

	if chunks.Indexes == nil {
		chunks.Indexes = make(map[string]*schema.SDLChunk)
	}
	chunks.Indexes[indexKey] = chunk

	return nil
}

// updateIndexChunkIfNeeded updates an existing index chunk if the index definition has changed
func updateIndexChunkIfNeeded(chunks *schema.SDLChunks, currentIndex, previousIndex *extendedIndexMetadata, indexKey string) error {
	if currentIndex == nil || previousIndex == nil || chunks == nil {
		return nil
	}

	// Check if the index definition has changed
	if !indexDefinitionsEqual(currentIndex.IndexMetadata, previousIndex.IndexMetadata) {
		// Index has changed - regenerate the chunk
		err := createIndexChunk(chunks, currentIndex, indexKey)
		if err != nil {
			return errors.Wrapf(err, "failed to recreate index chunk for %s", indexKey)
		}
	}

	return nil
}

// deleteIndexChunk removes an index chunk from the chunks
func deleteIndexChunk(chunks *schema.SDLChunks, indexKey string) {
	if chunks != nil && chunks.Indexes != nil {
		delete(chunks.Indexes, indexKey)
	}
}

// indexDefinitionsEqual compares two index definitions to see if they are equivalent
func indexDefinitionsEqual(index1, index2 *storepb.IndexMetadata) bool {
	if index1 == nil || index2 == nil {
		return false
	}

	// Compare basic properties
	if index1.Name != index2.Name ||
		index1.Unique != index2.Unique ||
		index1.Type != index2.Type ||
		len(index1.Expressions) != len(index2.Expressions) ||
		len(index1.Descending) != len(index2.Descending) {
		return false
	}

	// Compare expressions
	for i, expr := range index1.Expressions {
		if i >= len(index2.Expressions) || expr != index2.Expressions[i] {
			return false
		}
	}

	// Compare descending flags
	for i, desc := range index1.Descending {
		if i >= len(index2.Descending) || desc != index2.Descending[i] {
			return false
		}
	}

	// Note: PostgreSQL IndexMetadata doesn't have a Where field in the current schema
	// Index WHERE clauses are handled differently in the system
	// For now, we'll consider indexes equal if all other fields match

	return true
}

// generateCreateIndexSDL generates SDL text for a CREATE INDEX statement using writeIndexSDL
func generateCreateIndexSDL(extIndex *extendedIndexMetadata) string {
	if extIndex == nil || extIndex.IndexMetadata == nil {
		return ""
	}

	var buf strings.Builder
	schemaName := extIndex.SchemaName
	if schemaName == "" {
		schemaName = "public"
	}

	// Use the existing writeIndexSDL function from get_database_definition.go
	if err := writeIndexSDL(&buf, schemaName, extIndex.TableName, extIndex.IndexMetadata); err != nil {
		return ""
	}

	return buf.String()
}

// indexExtractor is a walker to extract CREATE INDEX AST nodes
type indexExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.IndexstmtContext
}

func (e *indexExtractor) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}

// applyFunctionChangesToChunks applies minimal changes to CREATE FUNCTION chunks
// This handles creation, modification, and deletion of function statements
func applyFunctionChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get function differences by comparing schema metadata
	currentMetadata := currentSchema.GetMetadata()
	previousMetadata := previousSchema.GetMetadata()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build function maps for current and previous schemas
	currentFunctions := make(map[string]*storepb.FunctionMetadata)
	previousFunctions := make(map[string]*storepb.FunctionMetadata)

	// Collect all functions from current schema
	for _, schema := range currentMetadata.Schemas {
		for _, function := range schema.Functions {
			functionKey := formatFunctionKey(schema.Name, function)
			currentFunctions[functionKey] = function
		}
	}

	// Collect all functions from previous schema
	for _, schema := range previousMetadata.Schemas {
		for _, function := range schema.Functions {
			functionKey := formatFunctionKey(schema.Name, function)
			previousFunctions[functionKey] = function
		}
	}

	// Process function additions: create new function chunks
	for functionKey, currentFunction := range currentFunctions {
		if _, exists := previousFunctions[functionKey]; !exists {
			// New function - create a chunk for it
			err := createFunctionChunk(previousChunks, currentFunction, functionKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create function chunk for %s", functionKey)
			}
		}
	}

	// Process function modifications: update existing chunks
	for functionKey, currentFunction := range currentFunctions {
		if previousFunction, exists := previousFunctions[functionKey]; exists {
			// Function exists in both - check if it needs modification
			err := updateFunctionChunkIfNeeded(previousChunks, currentFunction, previousFunction, functionKey)
			if err != nil {
				return errors.Wrapf(err, "failed to update function chunk for %s", functionKey)
			}
		}
	}

	// Process function deletions: remove dropped function chunks
	for functionKey := range previousFunctions {
		if _, exists := currentFunctions[functionKey]; !exists {
			// Function was dropped - remove it from chunks
			deleteFunctionChunk(previousChunks, functionKey)
		}
	}

	return nil
}

// formatFunctionKey creates a consistent key for function identification using the full signature
func formatFunctionKey(schemaName string, function *storepb.FunctionMetadata) string {
	if schemaName == "" {
		schemaName = "public"
	}

	// Extract the function signature from the definition
	signature := extractFunctionSignatureFromDefinition(function)
	if signature == "" {
		// Fallback to just the function name if signature extraction fails
		signature = function.Name + "()"
	}

	return schemaName + "." + signature
}

// extractFunctionSignatureFromDefinition extracts the function signature from its definition
func extractFunctionSignatureFromDefinition(function *storepb.FunctionMetadata) string {
	if function == nil || function.Definition == "" {
		return ""
	}

	// Parse the function definition to extract signature
	parseResult, err := pgparser.ParsePostgreSQL(function.Definition)
	if err != nil {
		return ""
	}

	tree := parseResult.Tree

	// Extract the CREATE FUNCTION node using a walker
	var result *parser.CreatefunctionstmtContext
	extractor := &functionExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return ""
	}

	// Use the unified function signature extraction
	return extractFunctionSignatureFromAST(result)
}

// extractFunctionSignatureFromAST extracts function signature from CREATE FUNCTION AST node
// This is the unified function used by both chunk extraction and function comparison
func extractFunctionSignatureFromAST(ctx *parser.CreatefunctionstmtContext) string {
	if ctx == nil {
		return ""
	}

	funcNameCtx := ctx.Func_name()
	if funcNameCtx == nil {
		return ""
	}

	// Extract function name using proper normalization
	funcNameParts := pgparser.NormalizePostgreSQLFuncName(funcNameCtx)
	if len(funcNameParts) == 0 {
		return ""
	}

	// Get the function name (without schema)
	var functionName string
	if len(funcNameParts) == 2 {
		// Schema qualified: schema.function_name
		functionName = funcNameParts[1]
	} else if len(funcNameParts) == 1 {
		// Unqualified: function_name
		functionName = funcNameParts[0]
	} else {
		// Unexpected format
		return ""
	}

	if functionName == "" {
		return ""
	}

	// Use the existing ExtractFunctionSignature function
	return ExtractFunctionSignature(ctx, functionName)
}

// createFunctionChunk creates a new CREATE FUNCTION chunk and adds it to the chunks
func createFunctionChunk(chunks *schema.SDLChunks, function *storepb.FunctionMetadata, functionKey string) error {
	if function == nil || chunks == nil {
		return nil
	}

	// Generate function SDL using the existing writeFunctionSDL function
	sdl := generateCreateFunctionSDL(function)
	if sdl == "" {
		return errors.Errorf("failed to generate SDL for function %s", functionKey)
	}

	// Parse the SDL to get the AST node
	astNode, err := extractFunctionASTFromSDL(sdl)
	if err != nil {
		return errors.Wrapf(err, "failed to extract AST from generated function SDL for %s", functionKey)
	}

	// Create the chunk
	chunk := &schema.SDLChunk{
		Identifier: functionKey,
		ASTNode:    astNode,
	}

	// Ensure Functions map is initialized
	if chunks.Functions == nil {
		chunks.Functions = make(map[string]*schema.SDLChunk)
	}

	chunks.Functions[functionKey] = chunk
	return nil
}

// updateFunctionChunkIfNeeded updates a function chunk if the definition has changed
func updateFunctionChunkIfNeeded(chunks *schema.SDLChunks, currentFunction, previousFunction *storepb.FunctionMetadata, functionKey string) error {
	if currentFunction == nil || previousFunction == nil || chunks == nil {
		return nil
	}

	// Check if function definitions are different
	if !functionDefinitionsEqual(currentFunction, previousFunction) {
		// Function was modified - update the chunk
		err := updateFunctionChunk(chunks, currentFunction, functionKey)
		if err != nil {
			return errors.Wrapf(err, "failed to update function chunk for %s", functionKey)
		}
	}

	return nil
}

// updateFunctionChunk updates an existing function chunk with new definition
func updateFunctionChunk(chunks *schema.SDLChunks, function *storepb.FunctionMetadata, functionKey string) error {
	if function == nil || chunks == nil {
		return nil
	}

	// Generate new function SDL
	sdl := generateCreateFunctionSDL(function)
	if sdl == "" {
		return errors.Errorf("failed to generate SDL for function %s", functionKey)
	}

	// Parse the SDL to get the AST node
	astNode, err := extractFunctionASTFromSDL(sdl)
	if err != nil {
		return errors.Wrapf(err, "failed to extract AST from generated function SDL for %s", functionKey)
	}

	// Update the existing chunk
	if chunk := chunks.Functions[functionKey]; chunk != nil {
		chunk.ASTNode = astNode
	} else {
		// Create new chunk if it doesn't exist
		chunk := &schema.SDLChunk{
			Identifier: functionKey,
			ASTNode:    astNode,
		}
		chunks.Functions[functionKey] = chunk
	}

	return nil
}

// deleteFunctionChunk removes a function chunk from the chunks
func deleteFunctionChunk(chunks *schema.SDLChunks, functionKey string) {
	if chunks != nil && chunks.Functions != nil {
		delete(chunks.Functions, functionKey)
	}
}

// generateCreateFunctionSDL generates SDL for a CREATE FUNCTION statement
func generateCreateFunctionSDL(function *storepb.FunctionMetadata) string {
	if function == nil {
		return ""
	}

	var buf strings.Builder
	// Use the existing writeFunctionSDL function
	if err := writeFunctionSDL(&buf, "", function); err != nil {
		return ""
	}

	return buf.String()
}

// extractFunctionTextFromAST extracts normalized text from function AST node using token stream
func extractFunctionTextFromAST(functionAST *parser.CreatefunctionstmtContext) string {
	if functionAST == nil {
		return ""
	}

	// Get tokens from the parser
	if parser := functionAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := functionAST.GetStart()
			stop := functionAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return functionAST.GetText()
}

// functionDefinitionsEqual compares two function definitions to see if they are equivalent
func functionDefinitionsEqual(function1, function2 *storepb.FunctionMetadata) bool {
	if function1 == nil && function2 == nil {
		return true
	}
	if function1 == nil || function2 == nil {
		return false
	}

	// Compare function names
	if function1.Name != function2.Name {
		return false
	}

	// Compare function definitions (most important comparison)
	definition1 := strings.TrimSpace(function1.Definition)
	definition2 := strings.TrimSpace(function2.Definition)

	// Normalize definitions for comparison
	definition1 = strings.TrimSuffix(definition1, ";")
	definition2 = strings.TrimSuffix(definition2, ";")

	return definition1 == definition2
}

// extractFunctionASTFromSDL parses a function SDL and extracts the CREATE FUNCTION AST node
func extractFunctionASTFromSDL(sdl string) (antlr.ParserRuleContext, error) {
	if sdl == "" {
		return nil, errors.New("empty SDL provided")
	}

	// Parse the SDL to get AST
	parseResult, err := pgparser.ParsePostgreSQL(sdl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse function SDL")
	}

	tree := parseResult.Tree

	// Extract the CREATE FUNCTION node using a walker
	var result *parser.CreatefunctionstmtContext
	extractor := &functionExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return nil, errors.New("no CREATE FUNCTION statement found in SDL")
	}

	return result, nil
}

// functionExtractor is a walker to extract CREATE FUNCTION AST nodes
type functionExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.CreatefunctionstmtContext
}

func (e *functionExtractor) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}

// applySequenceChangesToChunks applies minimal changes to CREATE SEQUENCE chunks
// This handles creation, modification, and deletion of sequence statements
func applySequenceChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get sequence differences by comparing schema metadata
	currentMetadata := currentSchema.GetMetadata()
	previousMetadata := previousSchema.GetMetadata()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build sequence maps for current and previous schemas
	currentSequences := make(map[string]*storepb.SequenceMetadata)
	previousSequences := make(map[string]*storepb.SequenceMetadata)

	// Collect all sequences from current schema
	for _, schema := range currentMetadata.Schemas {
		for _, sequence := range schema.Sequences {
			sequenceKey := formatSequenceKey(schema.Name, sequence.Name)
			currentSequences[sequenceKey] = sequence
		}
	}

	// Collect all sequences from previous schema
	for _, schema := range previousMetadata.Schemas {
		for _, sequence := range schema.Sequences {
			sequenceKey := formatSequenceKey(schema.Name, sequence.Name)
			previousSequences[sequenceKey] = sequence
		}
	}

	// Process sequence additions: create new sequence chunks
	for sequenceKey, currentSequence := range currentSequences {
		if _, exists := previousSequences[sequenceKey]; !exists {
			// New sequence - create a chunk for it
			err := createSequenceChunk(previousChunks, currentSequence, sequenceKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create sequence chunk for %s", sequenceKey)
			}
		}
	}

	// Process sequence modifications: update existing chunks
	for sequenceKey, currentSequence := range currentSequences {
		if previousSequence, exists := previousSequences[sequenceKey]; exists {
			// Sequence exists in both - check if it needs modification
			err := updateSequenceChunkIfNeeded(previousChunks, currentSequence, previousSequence, sequenceKey)
			if err != nil {
				return errors.Wrapf(err, "failed to update sequence chunk for %s", sequenceKey)
			}
		}
	}

	// Process sequence deletions: remove dropped sequence chunks
	for sequenceKey := range previousSequences {
		if _, exists := currentSequences[sequenceKey]; !exists {
			// Sequence was dropped - remove it from chunks
			deleteSequenceChunk(previousChunks, sequenceKey)
		}
	}

	return nil
}

// formatSequenceKey creates a consistent key for sequence identification
func formatSequenceKey(schemaName, sequenceName string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	return schemaName + "." + sequenceName
}

// extractSchemaFromSequenceKey extracts the schema name from a sequence key
func extractSchemaFromSequenceKey(sequenceKey string) string {
	parts := strings.SplitN(sequenceKey, ".", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return "public"
}

// createSequenceChunk creates a new CREATE SEQUENCE chunk and adds it to the chunks
func createSequenceChunk(chunks *schema.SDLChunks, sequence *storepb.SequenceMetadata, sequenceKey string) error {
	if sequence == nil || chunks == nil {
		return nil
	}

	// Generate sequence SDL using the existing writeSequenceSDL function
	schemaName := extractSchemaFromSequenceKey(sequenceKey)
	sdl := generateCreateSequenceSDL(schemaName, sequence)
	if sdl == "" {
		return errors.Errorf("failed to generate SDL for sequence %s", sequenceKey)
	}

	// Parse the SDL to get the AST node
	astNode, err := extractSequenceASTFromSDL(sdl)
	if err != nil {
		return errors.Wrapf(err, "failed to extract AST from generated sequence SDL for %s", sequenceKey)
	}

	// Create the chunk
	chunk := &schema.SDLChunk{
		Identifier: sequenceKey,
		ASTNode:    astNode,
	}

	// Ensure Sequences map is initialized
	if chunks.Sequences == nil {
		chunks.Sequences = make(map[string]*schema.SDLChunk)
	}

	chunks.Sequences[sequenceKey] = chunk
	return nil
}

// updateSequenceChunkIfNeeded updates a sequence chunk if the definition has changed
func updateSequenceChunkIfNeeded(chunks *schema.SDLChunks, currentSequence, previousSequence *storepb.SequenceMetadata, sequenceKey string) error {
	if currentSequence == nil || previousSequence == nil || chunks == nil {
		return nil
	}

	// Check if sequence definitions are different
	if !sequenceDefinitionsEqual(currentSequence, previousSequence) {
		// Sequence was modified - update the chunk
		err := updateSequenceChunk(chunks, currentSequence, sequenceKey)
		if err != nil {
			return errors.Wrapf(err, "failed to update sequence chunk for %s", sequenceKey)
		}
	}

	return nil
}

// updateSequenceChunk updates an existing sequence chunk with new definition
func updateSequenceChunk(chunks *schema.SDLChunks, sequence *storepb.SequenceMetadata, sequenceKey string) error {
	if sequence == nil || chunks == nil {
		return nil
	}

	// Generate new sequence SDL
	schemaName := extractSchemaFromSequenceKey(sequenceKey)
	sdl := generateCreateSequenceSDL(schemaName, sequence)
	if sdl == "" {
		return errors.Errorf("failed to generate SDL for sequence %s", sequenceKey)
	}

	// Parse the SDL to get the AST node
	astNode, err := extractSequenceASTFromSDL(sdl)
	if err != nil {
		return errors.Wrapf(err, "failed to extract AST from generated sequence SDL for %s", sequenceKey)
	}

	// Update the existing chunk
	if chunk := chunks.Sequences[sequenceKey]; chunk != nil {
		chunk.ASTNode = astNode
	} else {
		// Create new chunk if it doesn't exist
		chunk := &schema.SDLChunk{
			Identifier: sequenceKey,
			ASTNode:    astNode,
		}
		chunks.Sequences[sequenceKey] = chunk
	}

	return nil
}

// deleteSequenceChunk removes a sequence chunk from the chunks
func deleteSequenceChunk(chunks *schema.SDLChunks, sequenceKey string) {
	if chunks != nil && chunks.Sequences != nil {
		delete(chunks.Sequences, sequenceKey)
	}
}

// generateCreateSequenceSDL generates SDL for a CREATE SEQUENCE statement
func generateCreateSequenceSDL(schemaName string, sequence *storepb.SequenceMetadata) string {
	if sequence == nil {
		return ""
	}

	if schemaName == "" {
		schemaName = "public"
	}

	var buf strings.Builder
	// Use the existing writeSequenceSDL function
	if err := writeSequenceSDL(&buf, schemaName, sequence); err != nil {
		return ""
	}

	return buf.String()
}

// sequenceDefinitionsEqual compares two sequence definitions to see if they are equivalent
func sequenceDefinitionsEqual(sequence1, sequence2 *storepb.SequenceMetadata) bool {
	if sequence1 == nil && sequence2 == nil {
		return true
	}
	if sequence1 == nil || sequence2 == nil {
		return false
	}

	// Compare sequence names
	if sequence1.Name != sequence2.Name {
		return false
	}

	// Compare sequence parameters
	if sequence1.DataType != sequence2.DataType ||
		sequence1.Start != sequence2.Start ||
		sequence1.Increment != sequence2.Increment ||
		sequence1.MaxValue != sequence2.MaxValue ||
		sequence1.MinValue != sequence2.MinValue ||
		sequence1.CacheSize != sequence2.CacheSize ||
		sequence1.Cycle != sequence2.Cycle ||
		sequence1.OwnerTable != sequence2.OwnerTable ||
		sequence1.OwnerColumn != sequence2.OwnerColumn {
		return false
	}

	return true
}

// extractSequenceASTFromSDL parses a sequence SDL and extracts the CREATE SEQUENCE AST node
func extractSequenceASTFromSDL(sdl string) (antlr.ParserRuleContext, error) {
	if sdl == "" {
		return nil, errors.New("empty SDL provided")
	}

	// Parse the SDL to get AST
	parseResult, err := pgparser.ParsePostgreSQL(sdl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse sequence SDL")
	}

	tree := parseResult.Tree

	// Extract the CREATE SEQUENCE node using a walker
	var result *parser.CreateseqstmtContext
	extractor := &sequenceExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return nil, errors.New("no CREATE SEQUENCE statement found in SDL")
	}

	return result, nil
}

// extractSequenceTextFromAST extracts normalized text from sequence AST node using token stream
func extractSequenceTextFromAST(sequenceAST *parser.CreateseqstmtContext) string {
	if sequenceAST == nil {
		return ""
	}

	// Get tokens from the parser
	if parser := sequenceAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := sequenceAST.GetStart()
			stop := sequenceAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return sequenceAST.GetText()
}

// sequenceExtractor is a walker to extract CREATE SEQUENCE AST nodes
type sequenceExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.CreateseqstmtContext
}

func (e *sequenceExtractor) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}
