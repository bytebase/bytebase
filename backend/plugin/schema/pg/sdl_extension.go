package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processExtensionChanges processes extension changes between current and previous chunks
func processExtensionChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current extensions to find created and modified ones
	for extensionName, currentChunk := range currentChunks.Extensions {
		if previousChunk, exists := previousChunks.Extensions[extensionName]; exists {
			// Extension exists in both - check if modified by comparing text (excluding comments)
			currentText := currentChunk.GetTextWithoutComments()
			previousText := previousChunk.GetTextWithoutComments()
			if currentText != previousText {
				// Apply usability check
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, extensionName) {
					continue
				}
				// Extension was modified - use DROP + CREATE pattern
				// (PostgreSQL doesn't support ALTER EXTENSION for most changes)
				diff.ExtensionChanges = append(diff.ExtensionChanges, &schema.ExtensionDiff{
					Action:        schema.MetadataDiffActionDrop,
					ExtensionName: extensionName,
					OldExtension:  nil,
					NewExtension:  nil,
					OldASTNode:    previousChunk.ASTNode,
					NewASTNode:    nil,
				})
				diff.ExtensionChanges = append(diff.ExtensionChanges, &schema.ExtensionDiff{
					Action:        schema.MetadataDiffActionCreate,
					ExtensionName: extensionName,
					OldExtension:  nil,
					NewExtension:  nil,
					OldASTNode:    nil,
					NewASTNode:    currentChunk.ASTNode,
				})
				// Add COMMENT ON EXTENSION diffs if they exist in the new version
				if len(currentChunk.CommentStatements) > 0 {
					for _, commentNode := range currentChunk.CommentStatements {
						commentText := extractCommentTextFromNode(commentNode)
						diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
							Action:     schema.MetadataDiffActionCreate,
							ObjectType: schema.CommentObjectTypeExtension,
							ObjectName: extensionName,
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
			// New extension
			diff.ExtensionChanges = append(diff.ExtensionChanges, &schema.ExtensionDiff{
				Action:        schema.MetadataDiffActionCreate,
				ExtensionName: extensionName,
				OldExtension:  nil,
				NewExtension:  nil,
				OldASTNode:    nil,
				NewASTNode:    currentChunk.ASTNode,
			})
			// Add COMMENT ON EXTENSION diffs if they exist
			if len(currentChunk.CommentStatements) > 0 {
				for _, commentNode := range currentChunk.CommentStatements {
					commentText := extractCommentTextFromNode(commentNode)
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeExtension,
						ObjectName: extensionName,
						OldComment: "",
						NewComment: commentText,
						OldASTNode: nil,
						NewASTNode: commentNode,
					})
				}
			}
		}
	}

	// Process previous extensions to find dropped ones
	for extensionName, previousChunk := range previousChunks.Extensions {
		if _, exists := currentChunks.Extensions[extensionName]; !exists {
			// Extension was dropped
			diff.ExtensionChanges = append(diff.ExtensionChanges, &schema.ExtensionDiff{
				Action:        schema.MetadataDiffActionDrop,
				ExtensionName: extensionName,
				OldExtension:  nil,
				NewExtension:  nil,
				OldASTNode:    previousChunk.ASTNode,
				NewASTNode:    nil,
			})
		}
	}
}
