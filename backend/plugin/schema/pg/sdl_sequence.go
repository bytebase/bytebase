package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processSequenceChanges analyzes sequence changes between current and previous chunks
// Following the text-first comparison pattern for performance optimization
// Supports fine-grained diff: if only ALTER SEQUENCE OWNED BY changed, generate ALTER instead of DROP+CREATE
func processSequenceChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current sequences to find created and modified sequences
	for _, currentChunk := range currentChunks.Sequences {
		if previousChunk, exists := previousChunks.Sequences[currentChunk.Identifier]; exists {
			// Sequence exists in both - check if modified (excluding comments)
			currentText := currentChunk.GetTextWithoutComments()
			previousText := previousChunk.GetTextWithoutComments()
			if currentText != previousText {
				// Apply usability check: skip diff if current chunk matches database metadata SDL
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
					continue
				}

				// Fine-grained comparison: check if only ALTER statements changed
				currentCreateText := extractTextFromNode(currentChunk.ASTNode)
				previousCreateText := extractTextFromNode(previousChunk.ASTNode)
				currentAlterTexts := extractAlterTexts(currentChunk.AlterStatements)
				previousAlterTexts := extractAlterTexts(previousChunk.AlterStatements)

				createChanged := currentCreateText != previousCreateText
				alterChanged := currentAlterTexts != previousAlterTexts

				schemaName, sequenceName := parseIdentifier(currentChunk.Identifier)

				if createChanged && alterChanged {
					// Both CREATE and ALTER changed - use drop and recreate
					diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
						Action:       schema.MetadataDiffActionDrop,
						SchemaName:   schemaName,
						SequenceName: sequenceName,
						OldSequence:  nil,
						NewSequence:  nil,
						OldASTNode:   previousChunk.ASTNode,
						NewASTNode:   nil,
					})
					diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
						Action:       schema.MetadataDiffActionCreate,
						SchemaName:   schemaName,
						SequenceName: sequenceName,
						OldSequence:  nil,
						NewSequence:  nil,
						OldASTNode:   nil,
						NewASTNode:   currentChunk.ASTNode,
					})
					// Also need to add ALTER if current has ALTER statements
					if len(currentChunk.AlterStatements) > 0 {
						for _, alterNode := range currentChunk.AlterStatements {
							diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
								Action:       schema.MetadataDiffActionAlter,
								SchemaName:   schemaName,
								SequenceName: sequenceName,
								OldSequence:  nil,
								NewSequence:  nil,
								OldASTNode:   nil,
								NewASTNode:   alterNode,
							})
						}
					}
					// Add COMMENT ON SEQUENCE diffs if they exist in the new version
					if len(currentChunk.CommentStatements) > 0 {
						for _, commentNode := range currentChunk.CommentStatements {
							commentText := extractCommentTextFromNode(commentNode)
							diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
								Action:     schema.MetadataDiffActionCreate,
								ObjectType: schema.CommentObjectTypeSequence,
								SchemaName: schemaName,
								ObjectName: sequenceName,
								OldComment: "",
								NewComment: commentText,
								OldASTNode: nil,
								NewASTNode: commentNode,
							})
						}
					}
				} else if createChanged {
					// Only CREATE changed - use drop and recreate
					diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
						Action:       schema.MetadataDiffActionDrop,
						SchemaName:   schemaName,
						SequenceName: sequenceName,
						OldSequence:  nil,
						NewSequence:  nil,
						OldASTNode:   previousChunk.ASTNode,
						NewASTNode:   nil,
					})
					diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
						Action:       schema.MetadataDiffActionCreate,
						SchemaName:   schemaName,
						SequenceName: sequenceName,
						OldSequence:  nil,
						NewSequence:  nil,
						OldASTNode:   nil,
						NewASTNode:   currentChunk.ASTNode,
					})
					// Preserve ALTER if it exists in current
					if len(currentChunk.AlterStatements) > 0 {
						for _, alterNode := range currentChunk.AlterStatements {
							diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
								Action:       schema.MetadataDiffActionAlter,
								SchemaName:   schemaName,
								SequenceName: sequenceName,
								OldSequence:  nil,
								NewSequence:  nil,
								OldASTNode:   nil,
								NewASTNode:   alterNode,
							})
						}
					}
					// Add COMMENT ON SEQUENCE diffs if they exist in the new version
					if len(currentChunk.CommentStatements) > 0 {
						for _, commentNode := range currentChunk.CommentStatements {
							commentText := extractCommentTextFromNode(commentNode)
							diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
								Action:     schema.MetadataDiffActionCreate,
								ObjectType: schema.CommentObjectTypeSequence,
								SchemaName: schemaName,
								ObjectName: sequenceName,
								OldComment: "",
								NewComment: commentText,
								OldASTNode: nil,
								NewASTNode: commentNode,
							})
						}
					}
				} else if alterChanged {
					// Only ALTER changed - generate ALTER statements
					// This handles ownership changes without recreating the sequence
					if len(currentChunk.AlterStatements) > 0 {
						// Adding or modifying ALTER statements
						for _, alterNode := range currentChunk.AlterStatements {
							diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
								Action:       schema.MetadataDiffActionAlter,
								SchemaName:   schemaName,
								SequenceName: sequenceName,
								OldSequence:  nil,
								NewSequence:  nil,
								OldASTNode:   nil,
								NewASTNode:   alterNode,
							})
						}
					} else if len(previousChunk.AlterStatements) > 0 {
						// Removing ALTER statements - use the previous ALTER node to represent the removal
						// The migration generator should interpret this as removing the ownership
						for _, alterNode := range previousChunk.AlterStatements {
							diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
								Action:       schema.MetadataDiffActionAlter,
								SchemaName:   schemaName,
								SequenceName: sequenceName,
								OldSequence:  nil,
								NewSequence:  nil,
								OldASTNode:   alterNode,
								NewASTNode:   nil,
							})
						}
					}
				}
			}
			// If text is identical, skip - no changes detected
		} else {
			// New sequence
			schemaName, sequenceName := parseIdentifier(currentChunk.Identifier)
			// Add CREATE SEQUENCE diff
			diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
				Action:       schema.MetadataDiffActionCreate,
				SchemaName:   schemaName,
				SequenceName: sequenceName,
				OldSequence:  nil,
				NewSequence:  nil,
				OldASTNode:   nil,
				NewASTNode:   currentChunk.ASTNode,
			})
			// Add ALTER SEQUENCE OWNED BY diffs if they exist
			if len(currentChunk.AlterStatements) > 0 {
				for _, alterNode := range currentChunk.AlterStatements {
					diff.SequenceChanges = append(diff.SequenceChanges, &schema.SequenceDiff{
						Action:       schema.MetadataDiffActionAlter,
						SchemaName:   schemaName,
						SequenceName: sequenceName,
						OldSequence:  nil,
						NewSequence:  nil,
						OldASTNode:   nil,
						NewASTNode:   alterNode,
					})
				}
			}
			// Add COMMENT ON SEQUENCE diffs if they exist
			if len(currentChunk.CommentStatements) > 0 {
				for _, commentNode := range currentChunk.CommentStatements {
					commentText := extractCommentTextFromNode(commentNode)
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeSequence,
						SchemaName: schemaName,
						ObjectName: sequenceName,
						OldComment: "",
						NewComment: commentText,
						OldASTNode: nil,
						NewASTNode: commentNode,
					})
				}
			}
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
