package pg

import (
	parser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func processTableChanges(currentChunks, previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) error {
	// Process current table chunks to find created and modified tables
	for _, currentChunk := range currentChunks.Tables {
		schemaName, tableName := parseIdentifier(currentChunk.Identifier)

		if previousChunk, exists := previousChunks.Tables[currentChunk.Identifier]; exists {
			// Table exists in both - check if modified (excluding comments)
			currentText := currentChunk.GetTextWithoutComments()
			previousText := previousChunk.GetTextWithoutComments()
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
				excludeConstraintChanges := processExcludeConstraintChanges(oldASTNode, newASTNode, currentDBSDLChunks, currentChunk.Identifier)
				primaryKeyChanges := processPrimaryKeyChanges(oldASTNode, newASTNode, currentDBSDLChunks, currentChunk.Identifier)
				uniqueConstraintChanges := processUniqueConstraintChanges(oldASTNode, newASTNode, currentDBSDLChunks, currentChunk.Identifier)

				tableDiff := &schema.TableDiff{
					Action:                   schema.MetadataDiffActionAlter,
					SchemaName:               schemaName,
					TableName:                tableName,
					OldTable:                 nil, // Will be populated when SDL drift detection is implemented
					NewTable:                 nil, // Will be populated when SDL drift detection is implemented
					OldASTNode:               oldASTNode,
					NewASTNode:               newASTNode,
					ColumnChanges:            columnChanges,
					ForeignKeyChanges:        foreignKeyChanges,
					CheckConstraintChanges:   checkConstraintChanges,
					ExcludeConstraintChanges: excludeConstraintChanges,
					PrimaryKeyChanges:        primaryKeyChanges,
					UniqueConstraintChanges:  uniqueConstraintChanges,
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
				// Add COMMENT ON TABLE diffs if they exist
				if len(currentChunk.CommentStatements) > 0 {
					for _, commentNode := range currentChunk.CommentStatements {
						commentText := extractCommentTextFromNode(commentNode)
						diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
							Action:     schema.MetadataDiffActionCreate,
							ObjectType: schema.CommentObjectTypeTable,
							SchemaName: schemaName,
							ObjectName: tableName,
							OldComment: "",
							NewComment: commentText,
							OldASTNode: nil,
							NewASTNode: commentNode,
						})
					}
				}
				// Add COMMENT ON COLUMN diffs if they exist
				tableIdentifier := currentChunk.Identifier
				if columnComments := currentChunks.ColumnComments[tableIdentifier]; len(columnComments) > 0 {
					for columnName, commentNode := range columnComments {
						commentText := extractCommentTextFromNode(commentNode)
						if commentText != "" {
							diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
								Action:     schema.MetadataDiffActionCreate,
								ObjectType: schema.CommentObjectTypeColumn,
								SchemaName: schemaName,
								ObjectName: tableName,
								ColumnName: columnName,
								OldComment: "",
								NewComment: commentText,
								OldASTNode: nil,
								NewASTNode: commentNode,
							})
						}
					}
				}
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
func processColumnChanges(oldTable, newTable *parser.CreatestmtContext, currentSchema, previousSchema *model.DatabaseMetadata, currentDBSDLChunks *currentDatabaseSDLChunks, tableIdentifier string) []*schema.ColumnDiff {
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
