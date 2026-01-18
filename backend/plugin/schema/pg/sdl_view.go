package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func processViewChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current views to find created and modified views
	for _, currentChunk := range currentChunks.Views {
		if previousChunk, exists := previousChunks.Views[currentChunk.Identifier]; exists {
			// View exists in both - check if modified by comparing text first (excluding comments)
			currentText := currentChunk.GetTextWithoutComments()
			previousText := previousChunk.GetTextWithoutComments()
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
				// Add COMMENT ON VIEW diffs if they exist
				if len(currentChunk.CommentStatements) > 0 {
					for _, commentNode := range currentChunk.CommentStatements {
						commentText := extractCommentTextFromNode(commentNode)
						diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
							Action:     schema.MetadataDiffActionCreate,
							ObjectType: schema.CommentObjectTypeView,
							SchemaName: schemaName,
							ObjectName: viewName,
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
			// Add COMMENT ON VIEW diffs if they exist
			if len(currentChunk.CommentStatements) > 0 {
				for _, commentNode := range currentChunk.CommentStatements {
					commentText := extractCommentTextFromNode(commentNode)
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeView,
						SchemaName: schemaName,
						ObjectName: viewName,
						OldComment: "",
						NewComment: commentText,
						OldASTNode: nil,
						NewASTNode: commentNode,
					})
				}
			}
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
