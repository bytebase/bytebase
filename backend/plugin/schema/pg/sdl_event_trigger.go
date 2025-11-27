package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processEventTriggerChanges processes event trigger changes between current and previous chunks
func processEventTriggerChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process current event triggers to find created and modified ones
	for eventTriggerName, currentChunk := range currentChunks.EventTriggers {
		if previousChunk, exists := previousChunks.EventTriggers[eventTriggerName]; exists {
			// Event trigger exists in both - check if modified by comparing text (excluding comments)
			currentText := currentChunk.GetTextWithoutComments()
			previousText := previousChunk.GetTextWithoutComments()
			if currentText != previousText {
				// Apply usability check
				if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, eventTriggerName) {
					continue
				}
				// Event trigger was modified - use DROP + CREATE pattern
				// (PostgreSQL doesn't support CREATE OR REPLACE for event triggers)
				diff.EventTriggerChanges = append(diff.EventTriggerChanges, &schema.EventTriggerDiff{
					Action:           schema.MetadataDiffActionDrop,
					EventTriggerName: eventTriggerName,
					OldEventTrigger:  nil,
					NewEventTrigger:  nil,
					OldASTNode:       previousChunk.ASTNode,
					NewASTNode:       nil,
				})
				diff.EventTriggerChanges = append(diff.EventTriggerChanges, &schema.EventTriggerDiff{
					Action:           schema.MetadataDiffActionCreate,
					EventTriggerName: eventTriggerName,
					OldEventTrigger:  nil,
					NewEventTrigger:  nil,
					OldASTNode:       nil,
					NewASTNode:       currentChunk.ASTNode,
				})
			}
			// Note: Comment-only changes are handled by processCommentChanges
		} else {
			// Event trigger is new in current SDL
			currentText := currentChunk.GetTextWithoutComments()
			// Apply usability check
			if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, eventTriggerName) {
				continue
			}
			diff.EventTriggerChanges = append(diff.EventTriggerChanges, &schema.EventTriggerDiff{
				Action:           schema.MetadataDiffActionCreate,
				EventTriggerName: eventTriggerName,
				OldEventTrigger:  nil,
				NewEventTrigger:  nil,
				OldASTNode:       nil,
				NewASTNode:       currentChunk.ASTNode,
			})
			// Handle comments for new event triggers
			if len(currentChunk.CommentStatements) > 0 {
				for _, commentNode := range currentChunk.CommentStatements {
					diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
						Action:     schema.MetadataDiffActionCreate,
						ObjectType: schema.CommentObjectTypeEventTrigger,
						ObjectName: eventTriggerName,
						NewComment: extractTextFromNode(commentNode),
						NewASTNode: commentNode,
					})
				}
			}
		}
	}

	// Process previous event triggers to find dropped ones
	for eventTriggerName, previousChunk := range previousChunks.EventTriggers {
		if _, exists := currentChunks.EventTriggers[eventTriggerName]; !exists {
			// Event trigger was dropped
			diff.EventTriggerChanges = append(diff.EventTriggerChanges, &schema.EventTriggerDiff{
				Action:           schema.MetadataDiffActionDrop,
				EventTriggerName: eventTriggerName,
				OldEventTrigger:  nil,
				NewEventTrigger:  nil,
				OldASTNode:       previousChunk.ASTNode,
				NewASTNode:       nil,
			})
		}
	}
}
