package pg

import (
	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// processCommentChanges processes COMMENT ON statement changes between current and previous chunks
func processCommentChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Build sets of created and dropped objects to avoid generating comment diffs for them
	createdObjects := buildCreatedObjectsSet(diff)
	droppedObjects := buildDroppedObjectsSet(diff)

	// Process object-level comments
	processObjectComments(currentChunks.Tables, previousChunks.Tables, schema.CommentObjectTypeTable, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.Views, previousChunks.Views, schema.CommentObjectTypeView, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.MaterializedViews, previousChunks.MaterializedViews, schema.CommentObjectTypeMaterializedView, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.Functions, previousChunks.Functions, schema.CommentObjectTypeFunction, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.Sequences, previousChunks.Sequences, schema.CommentObjectTypeSequence, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.Extensions, previousChunks.Extensions, schema.CommentObjectTypeExtension, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.EnumTypes, previousChunks.EnumTypes, schema.CommentObjectTypeType, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.Indexes, previousChunks.Indexes, schema.CommentObjectTypeIndex, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.Schemas, previousChunks.Schemas, schema.CommentObjectTypeSchema, createdObjects, droppedObjects, currentDBSDLChunks, diff)
	processObjectComments(currentChunks.EventTriggers, previousChunks.EventTriggers, schema.CommentObjectTypeEventTrigger, createdObjects, droppedObjects, currentDBSDLChunks, diff)

	// Process column comments
	processColumnComments(currentChunks, previousChunks, createdObjects, droppedObjects, diff)
}

// buildCreatedObjectsSet builds a set of object identifiers that were created
func buildCreatedObjectsSet(diff *schema.MetadataDiff) map[string]bool {
	created := make(map[string]bool)

	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			identifier := tableDiff.SchemaName + "." + tableDiff.TableName
			created[identifier] = true
		}
	}

	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			identifier := viewDiff.SchemaName + "." + viewDiff.ViewName
			created[identifier] = true
		}
	}

	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionCreate {
			identifier := mvDiff.SchemaName + "." + mvDiff.MaterializedViewName
			created[identifier] = true
		}
	}

	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate {
			identifier := funcDiff.SchemaName + "." + funcDiff.FunctionName
			created[identifier] = true
		}
	}

	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate {
			identifier := seqDiff.SchemaName + "." + seqDiff.SequenceName
			created[identifier] = true
		}
	}

	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionCreate {
			identifier := enumDiff.SchemaName + "." + enumDiff.EnumTypeName
			created[identifier] = true
		}
	}

	for _, extDiff := range diff.ExtensionChanges {
		if extDiff.Action == schema.MetadataDiffActionCreate {
			// Extensions are database-level, no schema prefix
			created[extDiff.ExtensionName] = true
		}
	}

	for _, etDiff := range diff.EventTriggerChanges {
		if etDiff.Action == schema.MetadataDiffActionCreate {
			// Event triggers are database-level, no schema prefix
			created[etDiff.EventTriggerName] = true
		}
	}

	return created
}

// buildDroppedObjectsSet builds a set of object identifiers that were dropped
func buildDroppedObjectsSet(diff *schema.MetadataDiff) map[string]bool {
	dropped := make(map[string]bool)

	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			identifier := tableDiff.SchemaName + "." + tableDiff.TableName
			dropped[identifier] = true
		}
	}

	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop {
			identifier := viewDiff.SchemaName + "." + viewDiff.ViewName
			dropped[identifier] = true
		}
	}

	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionDrop {
			identifier := mvDiff.SchemaName + "." + mvDiff.MaterializedViewName
			dropped[identifier] = true
		}
	}

	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionDrop {
			identifier := funcDiff.SchemaName + "." + funcDiff.FunctionName
			dropped[identifier] = true
		}
	}

	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionDrop {
			identifier := seqDiff.SchemaName + "." + seqDiff.SequenceName
			dropped[identifier] = true
		}
	}

	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionDrop {
			identifier := enumDiff.SchemaName + "." + enumDiff.EnumTypeName
			dropped[identifier] = true
		}
	}

	for _, extDiff := range diff.ExtensionChanges {
		if extDiff.Action == schema.MetadataDiffActionDrop {
			// Extensions are database-level, no schema prefix
			dropped[extDiff.ExtensionName] = true
		}
	}

	for _, etDiff := range diff.EventTriggerChanges {
		if etDiff.Action == schema.MetadataDiffActionDrop {
			// Event triggers are database-level, no schema prefix
			dropped[etDiff.EventTriggerName] = true
		}
	}

	return dropped
}

// processObjectComments processes comment changes for a specific object type
// droppedObjects is intentionally unused because we only process objects in currentMap,
// and dropped objects won't appear there.
func processObjectComments(currentMap, previousMap map[string]*schema.SDLChunk, objectType schema.CommentObjectType, createdObjects, _ map[string]bool, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
	// Process all objects in current chunks
	for identifier, currentChunk := range currentMap {
		// Skip if object was created (comment will be in CREATE statement)
		if createdObjects[identifier] {
			continue
		}

		previousChunk := previousMap[identifier]
		if previousChunk == nil {
			// Object doesn't exist in previous - this shouldn't happen as it should be in createdObjects
			continue
		}

		// Extract comment text from both chunks
		currentCommentText := extractCommentTextFromChunk(currentChunk)
		previousCommentText := extractCommentTextFromChunk(previousChunk)

		// If comments are different, check usability before generating a CommentDiff
		if currentCommentText != previousCommentText {
			// Apply usability check: skip comment diff if current comment matches database metadata
			if currentDBSDLChunks != nil && shouldSkipCommentDiff(currentCommentText, identifier, currentDBSDLChunks) {
				continue
			}
			var schemaName, objectName string

			// For SCHEMA, EXTENSION, and EVENT TRIGGER objects, identifier is just the object name (database-level)
			// For other objects, identifier is "schema.object"
			switch objectType {
			case schema.CommentObjectTypeSchema:
				schemaName = identifier
				objectName = identifier // For schemas, objectName is also the schema name
			case schema.CommentObjectTypeExtension, schema.CommentObjectTypeEventTrigger:
				schemaName = "" // Extensions and event triggers are database-level, no schema
				objectName = identifier
			default:
				schemaName, objectName = parseIdentifier(identifier)
			}

			action := schema.MetadataDiffActionAlter
			if previousCommentText == "" {
				action = schema.MetadataDiffActionCreate
			}

			diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
				Action:     action,
				ObjectType: objectType,
				SchemaName: schemaName,
				ObjectName: objectName,
				OldComment: previousCommentText,
				NewComment: currentCommentText,
				OldASTNode: getFirstCommentNode(previousChunk.CommentStatements),
				NewASTNode: getFirstCommentNode(currentChunk.CommentStatements),
			})
		}
	}
}

// processColumnComments processes column comment changes
func processColumnComments(currentChunks, previousChunks *schema.SDLChunks, createdObjects, droppedObjects map[string]bool, diff *schema.MetadataDiff) {
	// Process all tables in current chunks
	allTableIdentifiers := make(map[string]bool)
	for identifier := range currentChunks.ColumnComments {
		allTableIdentifiers[identifier] = true
	}
	for identifier := range previousChunks.ColumnComments {
		allTableIdentifiers[identifier] = true
	}

	for tableIdentifier := range allTableIdentifiers {
		// Skip if table was created or dropped
		if createdObjects[tableIdentifier] || droppedObjects[tableIdentifier] {
			continue
		}

		currentColumns := currentChunks.ColumnComments[tableIdentifier]
		previousColumns := previousChunks.ColumnComments[tableIdentifier]

		// Find all column names
		allColumnNames := make(map[string]bool)
		for columnName := range currentColumns {
			allColumnNames[columnName] = true
		}
		for columnName := range previousColumns {
			allColumnNames[columnName] = true
		}

		// Compare each column's comment
		for columnName := range allColumnNames {
			currentNode := currentColumns[columnName]
			previousNode := previousColumns[columnName]

			currentCommentText := extractCommentTextFromNode(currentNode)
			previousCommentText := extractCommentTextFromNode(previousNode)

			if currentCommentText != previousCommentText {
				schemaName, tableName := parseIdentifier(tableIdentifier)
				action := schema.MetadataDiffActionAlter
				if previousCommentText == "" {
					action = schema.MetadataDiffActionCreate
				}

				diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
					Action:     action,
					ObjectType: schema.CommentObjectTypeColumn,
					SchemaName: schemaName,
					ObjectName: tableName,
					ColumnName: columnName,
					OldComment: previousCommentText,
					NewComment: currentCommentText,
					OldASTNode: previousNode,
					NewASTNode: currentNode,
				})
			}
		}
	}
}

// extractCommentTextFromChunk extracts the comment text from a chunk's comment statements
func extractCommentTextFromChunk(chunk *schema.SDLChunk) string {
	if chunk == nil || len(chunk.CommentStatements) == 0 {
		return ""
	}

	// Get the first comment statement (there should typically only be one)
	return extractCommentTextFromNode(chunk.CommentStatements[0])
}

// extractCommentTextFromNode extracts comment text from a COMMENT ON statement AST node
func extractCommentTextFromNode(node antlr.ParserRuleContext) string {
	if node == nil {
		return ""
	}

	// Try to cast to CommentstmtContext
	commentStmt, ok := node.(*parser.CommentstmtContext)
	if !ok {
		return ""
	}

	// Get the comment_text
	if commentStmt.Comment_text() == nil {
		return ""
	}

	commentTextCtx := commentStmt.Comment_text()

	// Check if it's NULL_P
	if commentTextCtx.NULL_P() != nil {
		return "" // NULL comment means no comment
	}

	// Get the sconst (string constant)
	if commentTextCtx.Sconst() != nil {
		text := commentTextCtx.Sconst().GetText()
		// Remove surrounding quotes
		if len(text) >= 2 && text[0] == '\'' && text[len(text)-1] == '\'' {
			return text[1 : len(text)-1]
		}
		return text
	}

	return ""
}

// getFirstCommentNode returns the first comment AST node from a list, or nil if empty
func getFirstCommentNode(nodes []antlr.ParserRuleContext) antlr.ParserRuleContext {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}
