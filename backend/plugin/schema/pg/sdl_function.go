package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processFunctionChanges analyzes function changes between current and previous chunks
// Following the text-first comparison pattern for performance optimization
func processFunctionChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current functions to find created and modified functions
	for _, currentChunk := range currentChunks.Functions {
		if previousChunk, exists := previousChunks.Functions[currentChunk.Identifier]; exists {
			// Function exists in both - check if modified by comparing text first (excluding comments)
			currentText := currentChunk.GetTextWithoutComments()
			previousText := previousChunk.GetTextWithoutComments()
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
			// Add COMMENT ON FUNCTION diffs if they exist
			if len(currentChunk.CommentStatements) > 0 {
				for _, commentNode := range currentChunk.CommentStatements {
					commentText := extractCommentTextFromNode(commentNode)
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeFunction,
						SchemaName: schemaName,
						ObjectName: functionName,
						OldComment: "",
						NewComment: commentText,
						OldASTNode: nil,
						NewASTNode: commentNode,
					})
				}
			}
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
