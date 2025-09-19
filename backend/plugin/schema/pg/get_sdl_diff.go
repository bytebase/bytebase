package pg

import (
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
	currentChunks, err := ChunkSDLText(currentSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk current SDL text")
	}

	previousChunks, err := ChunkSDLText(previousUserSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk previous SDL text")
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
	err = processTableChanges(currentChunks, previousChunks, currentSchema, previousSchema, diff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process table changes")
	}

	// Process index changes (standalone CREATE INDEX statements)
	processStandaloneIndexChanges(currentChunks, previousChunks, diff)

	// Process view changes
	processViewChanges(currentChunks, previousChunks, diff)

	// Process function changes
	processFunctionChanges(currentChunks, previousChunks, diff)

	// Process sequence changes
	processSequenceChanges(currentChunks, previousChunks, diff)

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
			Tokens:    nil,
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
			Tokens:    parseResult.Tokens,
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

	chunk := &schema.SDLChunk{
		Identifier: identifierStr,
		ASTNode:    ctx,
	}

	l.chunks.Tables[identifierStr] = chunk
}

func (l *sdlChunkExtractor) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	chunk := &schema.SDLChunk{
		Identifier: identifierStr,
		ASTNode:    ctx,
	}

	l.chunks.Sequences[identifierStr] = chunk
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

	// Join the parts to create the function name (schema.function_name or function_name)
	funcName := strings.Join(funcNameParts, ".")

	// Extract function signature for proper identification
	funcSignature := ExtractFunctionSignature(ctx, funcName)

	chunk := &schema.SDLChunk{
		Identifier: funcSignature, // Use signature instead of name
		ASTNode:    ctx,
	}

	l.chunks.Functions[funcSignature] = chunk
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

	chunk := &schema.SDLChunk{
		Identifier: indexName,
		ASTNode:    ctx,
	}

	l.chunks.Indexes[indexName] = chunk
}

func (l *sdlChunkExtractor) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	chunk := &schema.SDLChunk{
		Identifier: identifierStr,
		ASTNode:    ctx,
	}

	l.chunks.Views[identifierStr] = chunk
}

// processTableChanges processes changes to tables by comparing SDL chunks
// nolint:unparam
func processTableChanges(currentChunks, previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema, diff *schema.MetadataDiff) error {
	// TODO: currentSchema and previousSchema will be used later for SDL drift detection
	_ = currentSchema
	_ = previousSchema

	// Process current table chunks to find created and modified tables
	for identifier, currentChunk := range currentChunks.Tables {
		schemaName, tableName := parseIdentifier(currentChunk.Identifier)

		if previousChunk, exists := previousChunks.Tables[identifier]; exists {
			// Table exists in both - check if modified
			currentText := currentChunk.GetText(currentChunks.Tokens)
			previousText := previousChunk.GetText(previousChunks.Tokens)
			if currentText != previousText {
				// Table was modified - process column changes
				oldASTNode, ok := previousChunk.ASTNode.(*parser.CreatestmtContext)
				if !ok {
					return errors.Errorf("expected CreatestmtContext for previous table %s", previousChunk.Identifier)
				}
				newASTNode, ok := currentChunk.ASTNode.(*parser.CreatestmtContext)
				if !ok {
					return errors.Errorf("expected CreatestmtContext for current table %s", currentChunk.Identifier)
				}

				columnChanges := processColumnChanges(oldASTNode, newASTNode, currentSchema, previousSchema)
				foreignKeyChanges := processForeignKeyChanges(oldASTNode, newASTNode)
				checkConstraintChanges := processCheckConstraintChanges(oldASTNode, newASTNode)
				primaryKeyChanges := processPrimaryKeyChanges(oldASTNode, newASTNode)
				uniqueConstraintChanges := processUniqueConstraintChanges(oldASTNode, newASTNode)

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

	// Process previous table chunks to find dropped tables
	for identifier, previousChunk := range previousChunks.Tables {
		if _, exists := currentChunks.Tables[identifier]; !exists {
			// Table was dropped
			schemaName, tableName := parseIdentifier(previousChunk.Identifier)
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
func processColumnChanges(oldTable, newTable *parser.CreatestmtContext, currentSchema, previousSchema *model.DatabaseSchema) []*schema.ColumnDiff {
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

// parseTableIdentifier parses a table identifier and returns schema name and table name
func parseIdentifier(identifier string) (schemaName, objectName string) {
	parts := strings.Split(identifier, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", identifier
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
func processForeignKeyChanges(oldTable, newTable *parser.CreatestmtContext) []*schema.ForeignKeyDiff {
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

	// Step 2: Process old foreign keys in reverse order for DROP operations
	for i := len(oldFKList) - 1; i >= 0; i-- {
		oldFKDef := oldFKList[i]
		if newFKDef, exists := newFKMap[oldFKDef.Name]; exists {
			// FK exists in both - check if modified by comparing text first
			currentText := getForeignKeyText(newFKDef.ASTNode)
			previousText := getForeignKeyText(oldFKDef.ASTNode)
			if currentText != previousText {
				// FK was modified - store AST nodes only
				fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldFKDef.ASTNode,
				})
			}
		} else {
			// Foreign key was dropped - store AST node only
			fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldFKDef.ASTNode,
			})
		}
	}

	// Step 3: Process new foreign keys in order for CREATE operations
	for _, newFKDef := range newFKList {
		if oldFKDef, exists := oldFKMap[newFKDef.Name]; exists {
			// FK exists in both - check if modified by comparing text first
			currentText := getForeignKeyText(newFKDef.ASTNode)
			previousText := getForeignKeyText(oldFKDef.ASTNode)
			if currentText != previousText {
				// FK was modified - store AST nodes only
				fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newFKDef.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New foreign key - store AST node only
			fkDiffs = append(fkDiffs, &schema.ForeignKeyDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newFKDef.ASTNode,
			})
		}
	}

	return fkDiffs
}

// processCheckConstraintChanges analyzes check constraint changes between old and new table definitions
// Following the text-first comparison pattern for performance optimization
func processCheckConstraintChanges(oldTable, newTable *parser.CreatestmtContext) []*schema.CheckConstraintDiff {
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

	// Step 2: Process old constraints in reverse order for DROP operations
	for i := len(oldCheckList) - 1; i >= 0; i-- {
		oldCheckDef := oldCheckList[i]
		if newCheckDef, exists := newCheckMap[oldCheckDef.Name]; exists {
			// Check constraint exists in both - check if modified by comparing text first
			currentText := getCheckConstraintText(newCheckDef.ASTNode)
			previousText := getCheckConstraintText(oldCheckDef.ASTNode)
			if currentText != previousText {
				// Check constraint was modified - store AST nodes only
				// Drop and recreate for modifications (PostgreSQL pattern)
				checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldCheckDef.ASTNode,
				})
			}
		} else {
			// Check constraint was dropped - store AST node only
			checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldCheckDef.ASTNode,
			})
		}
	}

	// Step 3: Process new constraints in order for CREATE operations
	for _, newCheckDef := range newCheckList {
		if oldCheckDef, exists := oldCheckMap[newCheckDef.Name]; exists {
			// Check constraint exists in both - check if modified by comparing text first
			currentText := getCheckConstraintText(newCheckDef.ASTNode)
			previousText := getCheckConstraintText(oldCheckDef.ASTNode)
			if currentText != previousText {
				// Check constraint was modified - store AST nodes only
				checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newCheckDef.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New check constraint - store AST node only
			checkDiffs = append(checkDiffs, &schema.CheckConstraintDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newCheckDef.ASTNode,
			})
		}
	}

	return checkDiffs
}

// processPrimaryKeyChanges analyzes primary key constraint changes between old and new table definitions
func processPrimaryKeyChanges(oldTable, newTable *parser.CreatestmtContext) []*schema.PrimaryKeyDiff {
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
func processUniqueConstraintChanges(oldTable, newTable *parser.CreatestmtContext) []*schema.UniqueConstraintDiff {
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

	// Step 2: Process old constraints in reverse order for DROP operations
	for i := len(oldUKList) - 1; i >= 0; i-- {
		oldUKDef := oldUKList[i]
		if newUKDef, exists := newUKMap[oldUKDef.Name]; exists {
			// UK exists in both - check if modified by comparing text first
			currentText := getIndexText(newUKDef.ASTNode)
			previousText := getIndexText(oldUKDef.ASTNode)
			if currentText != previousText {
				// UK was modified - store AST nodes only
				ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: oldUKDef.ASTNode,
				})
			}
		} else {
			// UK was dropped - store AST node only
			ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldASTNode: oldUKDef.ASTNode,
			})
		}
	}

	// Step 3: Process new constraints in order for CREATE operations
	for _, newUKDef := range newUKList {
		if oldUKDef, exists := oldUKMap[newUKDef.Name]; exists {
			// UK exists in both - check if modified by comparing text first
			currentText := getIndexText(newUKDef.ASTNode)
			previousText := getIndexText(oldUKDef.ASTNode)
			if currentText != previousText {
				// UK was modified - store AST nodes only
				ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: newUKDef.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New UK - store AST node only
			ukDiffs = append(ukDiffs, &schema.UniqueConstraintDiff{
				Action:     schema.MetadataDiffActionCreate,
				NewASTNode: newUKDef.ASTNode,
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
func processStandaloneIndexChanges(currentChunks, previousChunks *schema.SDLChunks, diff *schema.MetadataDiff) {
	if currentChunks == nil || previousChunks == nil {
		return
	}

	// Initialize map with all existing table diffs for efficient lookups
	affectedTables := make(map[string]*schema.TableDiff, len(diff.TableChanges))
	for _, tableDiff := range diff.TableChanges {
		affectedTables[tableDiff.TableName] = tableDiff
	}

	// Step 1: Process current indexes to find created and modified indexes
	for indexName, currentChunk := range currentChunks.Indexes {
		tableName := extractTableNameFromIndex(currentChunk.ASTNode)
		if tableName == "" {
			continue // Skip if we can't determine the table name
		}

		if previousChunk, exists := previousChunks.Indexes[indexName]; exists {
			// Index exists in both - check if modified by comparing text first
			currentText := getStandaloneIndexText(currentChunk.ASTNode)
			previousText := getStandaloneIndexText(previousChunk.ASTNode)
			if currentText != previousText {
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
func processViewChanges(currentChunks, previousChunks *schema.SDLChunks, diff *schema.MetadataDiff) {
	// Process current views to find created and modified views
	for identifier, currentChunk := range currentChunks.Views {
		if previousChunk, exists := previousChunks.Views[identifier]; exists {
			// View exists in both - check if modified by comparing text first
			currentText := currentChunk.GetText(currentChunks.Tokens)
			previousText := previousChunk.GetText(previousChunks.Tokens)
			if currentText != previousText {
				// View was modified - use drop and recreate pattern (PostgreSQL standard)
				schemaName, viewName := parseIdentifier(identifier)
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
			schemaName, viewName := parseIdentifier(identifier)
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
func processFunctionChanges(currentChunks, previousChunks *schema.SDLChunks, diff *schema.MetadataDiff) {
	// Process current functions to find created and modified functions
	for identifier, currentChunk := range currentChunks.Functions {
		if previousChunk, exists := previousChunks.Functions[identifier]; exists {
			// Function exists in both - check if modified by comparing text first
			currentText := currentChunk.GetText(currentChunks.Tokens)
			previousText := previousChunk.GetText(previousChunks.Tokens)
			if currentText != previousText {
				// Function was modified - use CREATE OR REPLACE (AST-only mode)
				schemaName, functionName := parseIdentifier(identifier)
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
			schemaName, functionName := parseIdentifier(identifier)
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
func processSequenceChanges(currentChunks, previousChunks *schema.SDLChunks, diff *schema.MetadataDiff) {
	// Process current sequences to find created and modified sequences
	for identifier, currentChunk := range currentChunks.Sequences {
		if previousChunk, exists := previousChunks.Sequences[identifier]; exists {
			// Sequence exists in both - check if modified by comparing text first
			currentText := currentChunk.GetText(currentChunks.Tokens)
			previousText := previousChunk.GetText(previousChunks.Tokens)
			if currentText != previousText {
				// Sequence was modified - use drop and recreate pattern (PostgreSQL standard)
				schemaName, sequenceName := parseIdentifier(identifier)
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
			schemaName, sequenceName := parseIdentifier(identifier)
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
