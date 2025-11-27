package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processMaterializedViewChanges analyzes materialized view changes between current and previous chunks
// Following the same pattern as processViewChanges
// Note: Materialized views use DROP + CREATE pattern (no ALTER support in PostgreSQL)
func processMaterializedViewChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current materialized views to find created and modified materialized views
	for _, currentChunk := range currentChunks.MaterializedViews {
		if previousChunk, exists := previousChunks.MaterializedViews[currentChunk.Identifier]; exists {
			// Materialized view exists in both - check if modified by comparing text first (excluding comments)
			currentText := currentChunk.GetTextWithoutComments()
			previousText := previousChunk.GetTextWithoutComments()
			if currentText != previousText {
				// Apply usability check: skip diff if current chunk matches database metadata SDL
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}
				// Materialized view was modified - use drop and recreate pattern (PostgreSQL doesn't support ALTER MATERIALIZED VIEW definition)
				schemaName, mvName := parseIdentifier(currentChunk.Identifier)
				diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
					Action:               schema.MetadataDiffActionDrop,
					SchemaName:           schemaName,
					MaterializedViewName: mvName,
					OldMaterializedView:  nil, // Will be populated when SDL drift detection is implemented
					NewMaterializedView:  nil,
					OldASTNode:           previousChunk.ASTNode,
					NewASTNode:           nil,
				})
				diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
					Action:               schema.MetadataDiffActionCreate,
					SchemaName:           schemaName,
					MaterializedViewName: mvName,
					OldMaterializedView:  nil, // Will be populated when SDL drift detection is implemented
					NewMaterializedView:  nil, // Will be populated when SDL drift detection is implemented
					OldASTNode:           nil,
					NewASTNode:           currentChunk.ASTNode,
				})
				// Add COMMENT ON MATERIALIZED VIEW diffs if they exist
				if len(currentChunk.CommentStatements) > 0 {
					for _, commentNode := range currentChunk.CommentStatements {
						commentText := extractCommentTextFromNode(commentNode)
						diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
							Action:     schema.MetadataDiffActionCreate,
							ObjectType: schema.CommentObjectTypeMaterializedView,
							SchemaName: schemaName,
							ObjectName: mvName,
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
			// New materialized view
			schemaName, mvName := parseIdentifier(currentChunk.Identifier)
			diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
				Action:               schema.MetadataDiffActionCreate,
				SchemaName:           schemaName,
				MaterializedViewName: mvName,
				OldMaterializedView:  nil,
				NewMaterializedView:  nil, // Will be populated when SDL drift detection is implemented
				OldASTNode:           nil,
				NewASTNode:           currentChunk.ASTNode,
			})
			// Add COMMENT ON MATERIALIZED VIEW diffs if they exist
			if len(currentChunk.CommentStatements) > 0 {
				for _, commentNode := range currentChunk.CommentStatements {
					commentText := extractCommentTextFromNode(commentNode)
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeMaterializedView,
						SchemaName: schemaName,
						ObjectName: mvName,
						OldComment: "",
						NewComment: commentText,
						OldASTNode: nil,
						NewASTNode: commentNode,
					})
				}
			}
		}
	}

	// Process previous materialized views to find dropped ones
	for identifier, previousChunk := range previousChunks.MaterializedViews {
		if _, exists := currentChunks.MaterializedViews[identifier]; !exists {
			// Materialized view was dropped
			schemaName, mvName := parseIdentifier(identifier)
			diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
				Action:               schema.MetadataDiffActionDrop,
				SchemaName:           schemaName,
				MaterializedViewName: mvName,
				OldMaterializedView:  nil, // Will be populated when SDL drift detection is implemented
				NewMaterializedView:  nil,
				OldASTNode:           previousChunk.ASTNode,
				NewASTNode:           nil,
			})
		}
	}
}
