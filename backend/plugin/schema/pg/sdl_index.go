package pg

import (
	"strings"

	parser "github.com/bytebase/parser/postgresql"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processStandaloneIndexChanges analyzes standalone CREATE INDEX statement changes
// and adds them to the appropriate table's or materialized view's IndexChanges
func processStandaloneIndexChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	if currentChunks == nil || previousChunks == nil {
		return
	}

	// Initialize map with all existing table diffs for efficient lookups
	// Use schema.table format as key for consistency with extractTableNameFromIndex
	affectedTables := make(map[string]*schema.TableDiff, len(diff.TableChanges))
	for _, tableDiff := range diff.TableChanges {
		qualifiedTableName := tableDiff.SchemaName + "." + tableDiff.TableName
		affectedTables[qualifiedTableName] = tableDiff
	}

	// Initialize map with all existing materialized view diffs
	affectedMaterializedViews := make(map[string]*schema.MaterializedViewDiff, len(diff.MaterializedViewChanges))
	for _, mvDiff := range diff.MaterializedViewChanges {
		qualifiedMVName := mvDiff.SchemaName + "." + mvDiff.MaterializedViewName
		affectedMaterializedViews[qualifiedMVName] = mvDiff
	}

	// Step 1: Process current indexes to find created and modified indexes
	for _, currentChunk := range currentChunks.Indexes {
		targetObjectName := extractTableNameFromIndex(currentChunk.ASTNode)
		if targetObjectName == "" {
			continue // Skip if we can't determine the target object name
		}

		// Determine if this index is on a table or materialized view
		isOnMaterializedView := isIndexOnMaterializedView(targetObjectName, currentChunks, previousChunks)

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
				if isOnMaterializedView {
					// Index is on materialized view
					mvDiff := getOrCreateMaterializedViewDiff(diff, targetObjectName, affectedMaterializedViews)
					mvDiff.IndexChanges = append(mvDiff.IndexChanges, &schema.IndexDiff{
						Action:     schema.MetadataDiffActionDrop,
						OldASTNode: previousChunk.ASTNode,
					})
					mvDiff.IndexChanges = append(mvDiff.IndexChanges, &schema.IndexDiff{
						Action:     schema.MetadataDiffActionCreate,
						NewASTNode: currentChunk.ASTNode,
					})
				} else {
					// Index is on table
					tableDiff := getOrCreateTableDiff(diff, targetObjectName, affectedTables)
					tableDiff.IndexChanges = append(tableDiff.IndexChanges, &schema.IndexDiff{
						Action:     schema.MetadataDiffActionDrop,
						OldASTNode: previousChunk.ASTNode,
					})
					tableDiff.IndexChanges = append(tableDiff.IndexChanges, &schema.IndexDiff{
						Action:     schema.MetadataDiffActionCreate,
						NewASTNode: currentChunk.ASTNode,
					})
				}
				// Add COMMENT ON INDEX diffs if they exist in the new version
				if len(currentChunk.CommentStatements) > 0 {
					schemaName, indexName := parseIdentifier(currentChunk.Identifier)
					for _, commentNode := range currentChunk.CommentStatements {
						commentText := extractCommentTextFromNode(commentNode)
						diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
							Action:     schema.MetadataDiffActionCreate,
							ObjectType: schema.CommentObjectTypeIndex,
							SchemaName: schemaName,
							ObjectName: indexName,
							OldComment: "",
							NewComment: commentText,
							OldASTNode: nil,
							NewASTNode: commentNode,
						})
					}
				}
			}
			// If text is identical, skip - no changes detected
		} else {
			// New index - store AST node only
			if isOnMaterializedView {
				mvDiff := getOrCreateMaterializedViewDiff(diff, targetObjectName, affectedMaterializedViews)
				mvDiff.IndexChanges = append(mvDiff.IndexChanges, &schema.IndexDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: currentChunk.ASTNode,
				})
			} else {
				tableDiff := getOrCreateTableDiff(diff, targetObjectName, affectedTables)
				tableDiff.IndexChanges = append(tableDiff.IndexChanges, &schema.IndexDiff{
					Action:     schema.MetadataDiffActionCreate,
					NewASTNode: currentChunk.ASTNode,
				})
			}
			// Add COMMENT ON INDEX diffs if they exist
			if len(currentChunk.CommentStatements) > 0 {
				schemaName, indexName := parseIdentifier(currentChunk.Identifier)
				for _, commentNode := range currentChunk.CommentStatements {
					commentText := extractCommentTextFromNode(commentNode)
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeIndex,
						SchemaName: schemaName,
						ObjectName: indexName,
						OldComment: "",
						NewComment: commentText,
						OldASTNode: nil,
						NewASTNode: commentNode,
					})
				}
			}
		}
	}

	// Step 2: Process previous indexes to find dropped ones
	for indexName, previousChunk := range previousChunks.Indexes {
		if _, exists := currentChunks.Indexes[indexName]; !exists {
			// Index was dropped - store AST node only
			targetObjectName := extractTableNameFromIndex(previousChunk.ASTNode)
			if targetObjectName == "" {
				continue // Skip if we can't determine the target object name
			}

			// Determine if this index was on a table or materialized view
			isOnMaterializedView := isIndexOnMaterializedView(targetObjectName, currentChunks, previousChunks)

			if isOnMaterializedView {
				mvDiff := getOrCreateMaterializedViewDiff(diff, targetObjectName, affectedMaterializedViews)
				mvDiff.IndexChanges = append(mvDiff.IndexChanges, &schema.IndexDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: previousChunk.ASTNode,
				})
			} else {
				tableDiff := getOrCreateTableDiff(diff, targetObjectName, affectedTables)
				tableDiff.IndexChanges = append(tableDiff.IndexChanges, &schema.IndexDiff{
					Action:     schema.MetadataDiffActionDrop,
					OldASTNode: previousChunk.ASTNode,
				})
			}
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

// extractTableNameFromIndex extracts the fully qualified table name from a CREATE INDEX statement
// Returns schema.table format, defaulting to public schema if not specified
func extractTableNameFromIndex(astNode any) string {
	indexStmt, ok := astNode.(*parser.IndexstmtContext)
	if !ok || indexStmt == nil {
		return ""
	}

	// Extract table name from relation_expr in CREATE INDEX ... ON table_name
	if relationExpr := indexStmt.Relation_expr(); relationExpr != nil {
		if qualifiedName := relationExpr.Qualified_name(); qualifiedName != nil {
			// Extract qualified name parts
			qualifiedNameParts := pgparser.NormalizePostgreSQLQualifiedName(qualifiedName)
			if len(qualifiedNameParts) == 0 {
				return ""
			}

			// Return fully qualified name (schema.table)
			if len(qualifiedNameParts) == 1 {
				// No schema specified, default to public
				return "public." + qualifiedNameParts[0]
			}
			// Schema is specified
			return strings.Join(qualifiedNameParts, ".")
		}
	}

	return ""
}

// getOrCreateTableDiff finds an existing table diff or creates a new one for the given table
// tableName should be in schema.table format
func getOrCreateTableDiff(diff *schema.MetadataDiff, tableName string, affectedTables map[string]*schema.TableDiff) *schema.TableDiff {
	// Check if we already have this table in our map
	if tableDiff, exists := affectedTables[tableName]; exists {
		return tableDiff
	}

	// Parse schema and table name from tableName (format: schema.table)
	schemaName, tableNameOnly := parseIdentifier(tableName)

	// Create a new table diff for standalone index changes
	// We set Action to ALTER since we're modifying an existing table by adding/removing indexes
	newTableDiff := &schema.TableDiff{
		Action:                   schema.MetadataDiffActionAlter,
		SchemaName:               schemaName,
		TableName:                tableNameOnly,
		OldTable:                 nil, // Will be populated when SDL drift detection is implemented
		NewTable:                 nil, // Will be populated when SDL drift detection is implemented
		OldASTNode:               nil, // No table-level AST changes for standalone indexes
		NewASTNode:               nil, // No table-level AST changes for standalone indexes
		ColumnChanges:            []*schema.ColumnDiff{},
		IndexChanges:             []*schema.IndexDiff{},
		PrimaryKeyChanges:        []*schema.PrimaryKeyDiff{},
		UniqueConstraintChanges:  []*schema.UniqueConstraintDiff{},
		ForeignKeyChanges:        []*schema.ForeignKeyDiff{},
		CheckConstraintChanges:   []*schema.CheckConstraintDiff{},
		ExcludeConstraintChanges: []*schema.ExcludeConstraintDiff{},
	}

	diff.TableChanges = append(diff.TableChanges, newTableDiff)
	affectedTables[tableName] = newTableDiff
	return newTableDiff
}

// getOrCreateMaterializedViewDiff finds an existing materialized view diff or creates a new one
// mvName should be in schema.mv_name format
func getOrCreateMaterializedViewDiff(diff *schema.MetadataDiff, mvName string, affectedMaterializedViews map[string]*schema.MaterializedViewDiff) *schema.MaterializedViewDiff {
	// Check if we already have this materialized view in our map
	if mvDiff, exists := affectedMaterializedViews[mvName]; exists {
		return mvDiff
	}

	// Parse schema and MV name from mvName (format: schema.mv_name)
	schemaName, mvNameOnly := parseIdentifier(mvName)

	// Create a new materialized view diff for standalone index changes
	// We set Action to ALTER since we're modifying an existing MV by adding/removing indexes
	newMVDiff := &schema.MaterializedViewDiff{
		Action:               schema.MetadataDiffActionAlter,
		SchemaName:           schemaName,
		MaterializedViewName: mvNameOnly,
		OldMaterializedView:  nil,
		NewMaterializedView:  nil,
		OldASTNode:           nil,
		NewASTNode:           nil,
		IndexChanges:         []*schema.IndexDiff{},
	}

	diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, newMVDiff)
	affectedMaterializedViews[mvName] = newMVDiff
	return newMVDiff
}

// isIndexOnMaterializedView checks if the given object name refers to a materialized view
// rather than a table, by checking if it exists in the materialized view chunks
func isIndexOnMaterializedView(objectName string, currentChunks, previousChunks *schema.SDLChunks) bool {
	// Check if the object exists as a materialized view in current or previous chunks
	if currentChunks != nil && currentChunks.MaterializedViews != nil {
		if _, exists := currentChunks.MaterializedViews[objectName]; exists {
			return true
		}
	}
	if previousChunks != nil && previousChunks.MaterializedViews != nil {
		if _, exists := previousChunks.MaterializedViews[objectName]; exists {
			return true
		}
	}
	return false
}
