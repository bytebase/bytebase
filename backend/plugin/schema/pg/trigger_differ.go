package pg

import (
	"strings"

	parser "github.com/bytebase/parser/postgresql"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processStandaloneTriggerChanges analyzes standalone CREATE TRIGGER statement changes
// and adds them to the appropriate table's TriggerChanges
func processStandaloneTriggerChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	if currentChunks == nil || previousChunks == nil {
		return
	}

	// Build map of affected tables for efficient lookups
	affectedTables := make(map[string]*schema.TableDiff, len(diff.TableChanges))
	for _, tableDiff := range diff.TableChanges {
		qualifiedTableName := tableDiff.SchemaName + "." + tableDiff.TableName
		affectedTables[qualifiedTableName] = tableDiff
	}

	// Step 1: Process current triggers (CREATE and MODIFY)
	for _, currentChunk := range currentChunks.Triggers {
		targetTableName := extractTableNameFromTriggerChunk(currentChunk.ASTNode)
		if targetTableName == "" {
			// No CREATE TRIGGER statement, might be comment-only chunk
			// Handle comment-only chunks separately
			if len(currentChunk.CommentStatements) > 0 {
				// Extract info from chunk identifier: schema.table.trigger_name
				parts := strings.Split(currentChunk.Identifier, ".")
				if len(parts) == 3 {
					schemaName := parts[0]
					tableName := parts[1]
					triggerName := parts[2]
					targetTableName = schemaName + "." + tableName

					// Add COMMENT changes
					for _, commentNode := range currentChunk.CommentStatements {
						commentText := extractCommentTextFromNode(commentNode)
						diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
							Action:     schema.MetadataDiffActionCreate,
							ObjectType: schema.CommentObjectTypeTrigger,
							SchemaName: schemaName,
							TableName:  targetTableName,
							ObjectName: triggerName,
							OldComment: "",
							NewComment: commentText,
							OldASTNode: nil,
							NewASTNode: commentNode,
						})
					}
				}
			}
			continue
		}

		if previousChunk, exists := previousChunks.Triggers[currentChunk.Identifier]; exists {
			// Trigger exists in both - check if modified
			currentText := getStandaloneTriggerText(currentChunk.ASTNode)
			previousText := getStandaloneTriggerText(previousChunk.ASTNode)

			// Extract trigger info once for both trigger and comment changes
			schemaName, tableName := parseSchemaAndTableFromQualifiedName(targetTableName)
			triggerName := extractTriggerNameFromAST(currentChunk.ASTNode)

			if currentText != previousText {
				// Usability check
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}

				// Trigger was modified - use ALTER action (will be converted to CREATE OR REPLACE)
				tableDiff := getOrCreateTableDiff(diff, targetTableName, affectedTables)

				tableDiff.TriggerChanges = append(tableDiff.TriggerChanges, &schema.TriggerDiff{
					Action:      schema.MetadataDiffActionAlter,
					SchemaName:  schemaName,
					TableName:   tableName,
					TriggerName: triggerName,
					OldASTNode:  previousChunk.ASTNode,
					NewASTNode:  currentChunk.ASTNode,
				})
			}

			// Check for comment changes independently of trigger definition changes
			currentCommentText := ""
			if len(currentChunk.CommentStatements) > 0 {
				currentCommentText = extractCommentTextFromNode(currentChunk.CommentStatements[0])
			}
			previousCommentText := ""
			if len(previousChunk.CommentStatements) > 0 {
				previousCommentText = extractCommentTextFromNode(previousChunk.CommentStatements[0])
			}

			if currentCommentText != previousCommentText {
				if currentCommentText != "" && previousCommentText == "" {
					// Comment added
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeTrigger,
						SchemaName: schemaName,
						TableName:  targetTableName,
						ObjectName: triggerName,
						OldComment: "",
						NewComment: currentCommentText,
						OldASTNode: nil,
						NewASTNode: currentChunk.CommentStatements[0],
					})
				} else if currentCommentText == "" && previousCommentText != "" {
					// Comment removed
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionDrop,
						ObjectType: schema.CommentObjectTypeTrigger,
						SchemaName: schemaName,
						TableName:  targetTableName,
						ObjectName: triggerName,
						OldComment: previousCommentText,
						NewComment: "",
						OldASTNode: previousChunk.CommentStatements[0],
						NewASTNode: nil,
					})
				} else if currentCommentText != "" && previousCommentText != "" {
					// Comment modified
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionAlter,
						ObjectType: schema.CommentObjectTypeTrigger,
						SchemaName: schemaName,
						TableName:  targetTableName,
						ObjectName: triggerName,
						OldComment: previousCommentText,
						NewComment: currentCommentText,
						OldASTNode: previousChunk.CommentStatements[0],
						NewASTNode: currentChunk.CommentStatements[0],
					})
				}
			}
		} else {
			// New trigger
			tableDiff := getOrCreateTableDiff(diff, targetTableName, affectedTables)

			schemaName, tableName := parseSchemaAndTableFromQualifiedName(targetTableName)
			triggerName := extractTriggerNameFromAST(currentChunk.ASTNode)

			tableDiff.TriggerChanges = append(tableDiff.TriggerChanges, &schema.TriggerDiff{
				Action:      schema.MetadataDiffActionCreate,
				SchemaName:  schemaName,
				TableName:   tableName,
				TriggerName: triggerName,
				NewASTNode:  currentChunk.ASTNode,
			})

			// Add comments
			if len(currentChunk.CommentStatements) > 0 {
				for _, commentNode := range currentChunk.CommentStatements {
					commentText := extractCommentTextFromNode(commentNode)
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeTrigger,
						SchemaName: schemaName,
						TableName:  targetTableName,
						ObjectName: triggerName,
						OldComment: "",
						NewComment: commentText,
						OldASTNode: nil,
						NewASTNode: commentNode,
					})
				}
			}
		}
	}

	// Step 2: Process previous triggers (DROP)
	for triggerName, previousChunk := range previousChunks.Triggers {
		if _, exists := currentChunks.Triggers[triggerName]; !exists {
			// Trigger was dropped
			targetTableName := extractTableNameFromTriggerChunk(previousChunk.ASTNode)
			if targetTableName == "" {
				// No CREATE TRIGGER statement, skip (comment-only chunks don't need DROP)
				continue
			}

			tableDiff := getOrCreateTableDiff(diff, targetTableName, affectedTables)

			schemaName, tableName := parseSchemaAndTableFromQualifiedName(targetTableName)
			triggerNameStr := extractTriggerNameFromAST(previousChunk.ASTNode)

			tableDiff.TriggerChanges = append(tableDiff.TriggerChanges, &schema.TriggerDiff{
				Action:      schema.MetadataDiffActionDrop,
				SchemaName:  schemaName,
				TableName:   tableName,
				TriggerName: triggerNameStr,
				OldASTNode:  previousChunk.ASTNode,
			})
		}
	}
}

// extractTableNameFromTriggerChunk extracts the table name from a trigger chunk's AST node
// Returns empty string if the AST node is not a CREATE TRIGGER statement or if extraction fails
func extractTableNameFromTriggerChunk(astNode any) string {
	if astNode == nil {
		return ""
	}
	triggerCtx, ok := astNode.(*parser.CreatetrigstmtContext)
	if !ok {
		return ""
	}
	return extractTableNameFromTrigger(triggerCtx)
}

// getStandaloneTriggerText returns the text representation of a CREATE TRIGGER statement
func getStandaloneTriggerText(astNode any) string {
	triggerStmt, ok := astNode.(*parser.CreatetrigstmtContext)
	if !ok || triggerStmt == nil {
		return ""
	}

	// Get text from token stream
	if parser := triggerStmt.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := triggerStmt.GetStart()
			stop := triggerStmt.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	return triggerStmt.GetText()
}

// extractTriggerNameFromAST extracts trigger name from CREATE TRIGGER AST
func extractTriggerNameFromAST(astNode any) string {
	triggerStmt, ok := astNode.(*parser.CreatetrigstmtContext)
	if !ok || triggerStmt == nil || triggerStmt.Name() == nil {
		return ""
	}
	return pgparser.NormalizePostgreSQLName(triggerStmt.Name())
}

// parseSchemaAndTableFromQualifiedName parses "schema.table" into separate parts
func parseSchemaAndTableFromQualifiedName(qualifiedName string) (schemaName, tableName string) {
	parts := strings.Split(qualifiedName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", qualifiedName
}
