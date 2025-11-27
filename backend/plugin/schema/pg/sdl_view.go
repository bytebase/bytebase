package pg

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
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

func applyViewChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get view differences by comparing schema metadata
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build view maps for current and previous schemas
	currentViews := make(map[string]*storepb.ViewMetadata)
	previousViews := make(map[string]*storepb.ViewMetadata)

	// Collect all views from current schema
	for _, schema := range currentMetadata.Schemas {
		for _, view := range schema.Views {
			viewKey := formatViewKey(schema.Name, view.Name)
			currentViews[viewKey] = view
		}
	}

	// Collect all views from previous schema
	for _, schema := range previousMetadata.Schemas {
		for _, view := range schema.Views {
			viewKey := formatViewKey(schema.Name, view.Name)
			previousViews[viewKey] = view
		}
	}

	// Process view additions: create new view chunks
	for viewKey, currentView := range currentViews {
		if _, exists := previousViews[viewKey]; !exists {
			// New view - create a chunk for it
			err := createViewChunk(previousChunks, currentView, viewKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create view chunk for %s", viewKey)
			}
		}
	}

	// Process view modifications: update existing chunks
	for viewKey, currentView := range currentViews {
		if previousView, exists := previousViews[viewKey]; exists {
			// View exists in both metadata
			// Only update if chunk exists in SDL (user explicitly defined it)
			// If chunk doesn't exist, skip - we don't force-add database objects that user didn't define
			if _, chunkExists := previousChunks.Views[viewKey]; chunkExists {
				// Chunk exists - update if needed
				err := updateViewChunkIfNeeded(previousChunks, currentView, previousView, viewKey)
				if err != nil {
					return errors.Wrapf(err, "failed to update view chunk for %s", viewKey)
				}
			}
			// If chunk doesn't exist, skip - user didn't define this view in SDL
		}
	}

	// Process view deletions: remove dropped view chunks
	for viewKey := range previousViews {
		if _, exists := currentViews[viewKey]; !exists {
			// View was dropped - remove it from chunks
			deleteViewChunk(previousChunks, viewKey)
		}
	}

	return nil
}

// formatViewKey creates a consistent key for view identification
func formatViewKey(schemaName, viewName string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	return schemaName + "." + viewName
}

// createViewChunk creates a new CREATE VIEW chunk and adds it to the chunks
func createViewChunk(chunks *schema.SDLChunks, view *storepb.ViewMetadata, viewKey string) error {
	if view == nil || chunks == nil {
		return nil
	}

	// Generate SDL text for the view
	schemaName, _ := parseIdentifier(viewKey)
	viewSDL := generateCreateViewSDL(schemaName, view)
	if viewSDL == "" {
		return errors.New("failed to generate SDL for view")
	}

	// Parse the SDL to get AST node
	parseResults, err := pgparser.ParsePostgreSQL(viewSDL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse generated view SDL: %s", viewSDL)
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	// Extract the CREATE VIEW AST node
	var viewASTNode *parser.ViewstmtContext
	antlr.ParseTreeWalkerDefault.Walk(&viewExtractor{
		result: &viewASTNode,
	}, parseResult.Tree)

	if viewASTNode == nil {
		return errors.New("failed to extract CREATE VIEW AST node")
	}

	// Create and add the chunk
	chunk := &schema.SDLChunk{
		Identifier: viewKey,
		ASTNode:    viewASTNode,
	}

	// Add comment if the view has one
	if view.Comment != "" {
		commentSQL := generateCommentOnViewSQL(schemaName, view.Name, view.Comment)
		commentParseResult, err := pgparser.ParsePostgreSQL(commentSQL)
		if err == nil && len(commentParseResult) > 0 && commentParseResult[0].Tree != nil {
			// Extract COMMENT ON VIEW AST node
			var commentASTNode *parser.CommentstmtContext
			antlr.ParseTreeWalkerDefault.Walk(&commentExtractor{
				result: &commentASTNode,
			}, commentParseResult[0].Tree)

			if commentASTNode != nil {
				chunk.CommentStatements = []antlr.ParserRuleContext{commentASTNode}
			}
		}
	}

	if chunks.Views == nil {
		chunks.Views = make(map[string]*schema.SDLChunk)
	}
	chunks.Views[viewKey] = chunk

	return nil
}

// updateViewChunkIfNeeded updates an existing view chunk if the view definition has changed
func updateViewChunkIfNeeded(chunks *schema.SDLChunks, currentView, previousView *storepb.ViewMetadata, viewKey string) error {
	if currentView == nil || previousView == nil || chunks == nil {
		return nil
	}

	// Get the existing chunk
	chunk, exists := chunks.Views[viewKey]
	if !exists {
		return errors.Errorf("view chunk not found for key %s", viewKey)
	}

	// Check if the CREATE VIEW definition has changed (excluding comment)
	definitionChanged := !viewDefinitionsEqualExcludingComment(currentView, previousView)

	if definitionChanged {
		// View definition has changed - regenerate the CREATE VIEW chunk
		schemaName, _ := parseIdentifier(viewKey)
		viewSDL := generateCreateViewSDL(schemaName, currentView)
		if viewSDL == "" {
			return errors.New("failed to generate SDL for view")
		}

		// Parse the SDL to get AST node
		parseResults, err := pgparser.ParsePostgreSQL(viewSDL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse generated view SDL: %s", viewSDL)
		}

		// Expect single statement
		if len(parseResults) != 1 {
			return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
		}
		parseResult := parseResults[0]

		// Extract the CREATE VIEW AST node
		var viewASTNode *parser.ViewstmtContext
		antlr.ParseTreeWalkerDefault.Walk(&viewExtractor{
			result: &viewASTNode,
		}, parseResult.Tree)

		if viewASTNode == nil {
			return errors.New("failed to extract CREATE VIEW AST node")
		}

		// Update the CREATE VIEW AST node
		chunk.ASTNode = viewASTNode
	}

	// Synchronize COMMENT ON VIEW statements only if comment has changed
	if currentView.Comment != previousView.Comment {
		schemaName, _ := parseIdentifier(viewKey)
		if err := syncObjectCommentStatements(chunk, currentView.Comment, "VIEW", schemaName, currentView.Name); err != nil {
			return errors.Wrapf(err, "failed to sync COMMENT statements for view %s", viewKey)
		}
	}

	return nil
}

// deleteViewChunk removes a view chunk from the chunks
func deleteViewChunk(chunks *schema.SDLChunks, viewKey string) {
	if chunks != nil && chunks.Views != nil {
		delete(chunks.Views, viewKey)
	}
}

// viewDefinitionsEqualExcludingComment compares two view definitions excluding comments
func viewDefinitionsEqualExcludingComment(view1, view2 *storepb.ViewMetadata) bool {
	if view1 == nil || view2 == nil {
		return false
	}

	// Compare name and definition (excluding comment)
	if view1.Name != view2.Name ||
		view1.Definition != view2.Definition {
		return false
	}

	return true
}

// generateCreateViewSDL generates SDL text for a CREATE VIEW statement
func generateCreateViewSDL(schemaName string, view *storepb.ViewMetadata) string {
	if view == nil {
		return ""
	}

	var buf strings.Builder
	if err := writeViewSDL(&buf, schemaName, view); err != nil {
		return ""
	}

	return buf.String()
}

// generateCommentOnViewSQL generates a COMMENT ON VIEW statement
func generateCommentOnViewSQL(schemaName, viewName, comment string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	// Escape single quotes in comment
	escapedComment := strings.ReplaceAll(comment, "'", "''")
	return fmt.Sprintf("COMMENT ON VIEW \"%s\".\"%s\" IS '%s';", schemaName, viewName, escapedComment)
}

// viewExtractor is a walker to extract CREATE VIEW AST nodes
type viewExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.ViewstmtContext
}

func (e *viewExtractor) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}
