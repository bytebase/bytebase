package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processEnumTypeChanges analyzes enum type changes between current and previous chunks
// Enum types use DROP + CREATE pattern for modifications (PostgreSQL doesn't support ALTER TYPE ... RENAME VALUE)
func processEnumTypeChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current enum types to find created and modified ones
	for _, currentChunk := range currentChunks.EnumTypes {
		if previousChunk, exists := previousChunks.EnumTypes[currentChunk.Identifier]; exists {
			// Enum type exists in both - check if modified by comparing text (excluding comments)
			currentText := currentChunk.GetTextWithoutComments()
			previousText := previousChunk.GetTextWithoutComments()
			if currentText != previousText {
				// Apply usability check
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}
				// Enum type was modified - use DROP + CREATE pattern
				schemaName, enumName := parseIdentifier(currentChunk.Identifier)
				// Add DROP diff
				diff.EnumTypeChanges = append(diff.EnumTypeChanges, &schema.EnumTypeDiff{
					Action:       schema.MetadataDiffActionDrop,
					SchemaName:   schemaName,
					EnumTypeName: enumName,
					OldEnumType:  nil,
					NewEnumType:  nil,
					OldASTNode:   previousChunk.ASTNode,
					NewASTNode:   nil,
				})
				// Add CREATE diff
				diff.EnumTypeChanges = append(diff.EnumTypeChanges, &schema.EnumTypeDiff{
					Action:       schema.MetadataDiffActionCreate,
					SchemaName:   schemaName,
					EnumTypeName: enumName,
					OldEnumType:  nil,
					NewEnumType:  nil,
					OldASTNode:   nil,
					NewASTNode:   currentChunk.ASTNode,
				})
				// Add COMMENT ON TYPE diffs if they exist in the new version
				if len(currentChunk.CommentStatements) > 0 {
					for _, commentNode := range currentChunk.CommentStatements {
						commentText := extractCommentTextFromNode(commentNode)
						diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
							Action:     schema.MetadataDiffActionCreate,
							ObjectType: schema.CommentObjectTypeType,
							SchemaName: schemaName,
							ObjectName: enumName,
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
			// New enum type
			schemaName, enumName := parseIdentifier(currentChunk.Identifier)
			diff.EnumTypeChanges = append(diff.EnumTypeChanges, &schema.EnumTypeDiff{
				Action:       schema.MetadataDiffActionCreate,
				SchemaName:   schemaName,
				EnumTypeName: enumName,
				OldEnumType:  nil,
				NewEnumType:  nil,
				OldASTNode:   nil,
				NewASTNode:   currentChunk.ASTNode,
			})
			// Add COMMENT ON TYPE diffs if they exist
			if len(currentChunk.CommentStatements) > 0 {
				for _, commentNode := range currentChunk.CommentStatements {
					commentText := extractCommentTextFromNode(commentNode)
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeType,
						SchemaName: schemaName,
						ObjectName: enumName,
						OldComment: "",
						NewComment: commentText,
						OldASTNode: nil,
						NewASTNode: commentNode,
					})
				}
			}
		}
	}

	// Process previous enum types to find dropped ones
	for identifier, previousChunk := range previousChunks.EnumTypes {
		if _, exists := currentChunks.EnumTypes[identifier]; !exists {
			// Enum type was dropped
			schemaName, enumName := parseIdentifier(identifier)
			diff.EnumTypeChanges = append(diff.EnumTypeChanges, &schema.EnumTypeDiff{
				Action:       schema.MetadataDiffActionDrop,
				SchemaName:   schemaName,
				EnumTypeName: enumName,
				OldEnumType:  nil,
				NewEnumType:  nil,
				OldASTNode:   previousChunk.ASTNode,
				NewASTNode:   nil,
			})
		}
	}
}
