package pg

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterGetSDLDiff(storepb.Engine_POSTGRES, GetSDLDiff)
	schema.RegisterGetSDLDiff(storepb.Engine_COCKROACHDB, GetSDLDiff)
}

func GetSDLDiff(currentSDLText, previousUserSDLText string, currentSchema *model.DatabaseMetadata) (*schema.MetadataDiff, error) {
	generatedSDL, err := convertDatabaseSchemaToSDL(currentSchema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert current schema to SDL format for initialization")
	}

	// Check for initialization scenario: previousUserSDLText is empty
	if strings.TrimSpace(previousUserSDLText) == "" && currentSchema != nil {
		// Initialization scenario: convert currentSchema to SDL format as baseline
		previousUserSDLText = generatedSDL
	}

	// Only skip processing if both current SDL and generated SDL match
	// AND there is actually a current schema to compare against.
	// If currentSchema is nil, we must process the diff to detect drops from previous SDL.
	if currentSchema != nil && strings.TrimSpace(currentSDLText) == strings.TrimSpace(generatedSDL) {
		// No changes detected between current SDL and database schema
		return &schema.MetadataDiff{}, nil
	}

	currentChunks, err := ChunkSDLText(currentSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk current SDL text")
	}

	previousChunks, err := ChunkSDLText(previousUserSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk previous SDL text")
	}

	// Pre-compute SDL chunks from current database metadata for performance optimization
	// This avoids repeated calls to convertDatabaseSchemaToSDL and ChunkSDLText during usability checks
	currentDBSDLChunks, err := buildCurrentDatabaseSDLChunks(currentSchema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build current database SDL chunks")
	}

	// Initialize MetadataDiff
	diff := &schema.MetadataDiff{
		DatabaseName:            "",
		SchemaChanges:           []*schema.SchemaDiff{},
		TableChanges:            []*schema.TableDiff{},
		ViewChanges:             []*schema.ViewDiff{},
		MaterializedViewChanges: []*schema.MaterializedViewDiff{},
		FunctionChanges:         []*schema.FunctionDiff{},
		ProcedureChanges:        []*schema.ProcedureDiff{},
		SequenceChanges:         []*schema.SequenceDiff{},
		EnumTypeChanges:         []*schema.EnumTypeDiff{},
		CommentChanges:          []*schema.CommentDiff{},
	}

	// Process table changes
	err = processTableChanges(currentChunks, previousChunks, currentSchema, currentDBSDLChunks, diff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process table changes")
	}

	// Process index changes (standalone CREATE INDEX statements)
	processStandaloneIndexChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process trigger changes (standalone CREATE TRIGGER statements)
	processStandaloneTriggerChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process view changes
	processViewChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process materialized view changes
	processMaterializedViewChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process function changes
	processFunctionChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process sequence changes
	processSequenceChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process enum type changes
	processEnumTypeChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process extension changes
	processExtensionChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process event trigger changes
	processEventTriggerChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Process explicit schema changes (CREATE SCHEMA statements)
	processSchemaChanges(currentChunks, previousChunks, currentSchema, diff)

	// Process comment changes (must be after all object changes are processed)
	processCommentChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

	// Add implicit schema creation for schemas referenced by new objects
	// This handles the case where users define objects in a new schema without explicit CREATE SCHEMA
	addImplicitSchemaCreation(diff, currentSchema)

	return diff, nil
}

// currentDatabaseSDLChunks stores pre-computed SDL chunks from current database metadata
// for performance optimization during usability checks
type currentDatabaseSDLChunks struct {
	chunks      map[string]string   // maps chunk identifier to normalized SDL text (without comments) from current database metadata
	comments    map[string][]string // maps chunk identifier to normalized comment texts
	columns     map[string]string   // maps "schema.table.column" to normalized column SDL text
	constraints map[string]string   // maps "schema.table.constraint" to normalized constraint SDL text
}

// buildCurrentDatabaseSDLChunks pre-computes SDL chunks from the current database schema
// for usability checks. This avoids repeated expensive calls to convertDatabaseSchemaToSDL
// and ChunkSDLText during diff processing by storing normalized SDL text from current database metadata.
func buildCurrentDatabaseSDLChunks(currentSchema *model.DatabaseMetadata) (*currentDatabaseSDLChunks, error) {
	sdlChunks := &currentDatabaseSDLChunks{
		chunks:      make(map[string]string),
		comments:    make(map[string][]string),
		columns:     make(map[string]string),
		constraints: make(map[string]string),
	}

	// Only build SDL chunks if current schema is provided
	if currentSchema == nil {
		return sdlChunks, nil
	}

	// Generate SDL from current database metadata
	currentSDL, err := convertDatabaseSchemaToSDL(currentSchema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert current schema to SDL")
	}

	// Parse the generated SDL to get chunks
	currentSDLChunks, err := ChunkSDLText(currentSDL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk current schema SDL")
	}

	// Populate SDL chunks with normalized chunk texts from current database metadata
	// Use GetTextWithoutComments() to focus on structural changes only, not comment formatting differences
	for identifier, chunk := range currentSDLChunks.Tables {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[identifier] = []string{commentText}
		}

		// Extract column and constraint SDL texts for fine-grained usability checks
		if err := extractColumnAndConstraintSDLTexts(chunk, identifier, sdlChunks); err != nil {
			// Log error but don't fail the entire operation
			// Fine-grained usability checks will fall back to table-level checks
			continue
		}
	}
	for identifier, chunk := range currentSDLChunks.Views {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[identifier] = []string{commentText}
		}
	}
	for identifier, chunk := range currentSDLChunks.MaterializedViews {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[identifier] = []string{commentText}
		}
	}
	for identifier, chunk := range currentSDLChunks.Functions {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[identifier] = []string{commentText}
		}
	}
	for identifier, chunk := range currentSDLChunks.Sequences {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[identifier] = []string{commentText}
		}
	}
	for identifier, chunk := range currentSDLChunks.Indexes {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[identifier] = []string{commentText}
		}
	}
	for identifier, chunk := range currentSDLChunks.EnumTypes {
		sdlChunks.chunks[identifier] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[identifier] = []string{commentText}
		}
	}
	for extensionName, chunk := range currentSDLChunks.Extensions {
		sdlChunks.chunks[extensionName] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[extensionName] = []string{commentText}
		}
	}
	for eventTriggerName, chunk := range currentSDLChunks.EventTriggers {
		sdlChunks.chunks[eventTriggerName] = strings.TrimSpace(chunk.GetTextWithoutComments())
		// Store comment text for usability check
		commentText := extractCommentTextFromChunk(chunk)
		if commentText != "" {
			sdlChunks.comments[eventTriggerName] = []string{commentText}
		}
	}

	return sdlChunks, nil
}

// extractColumnAndConstraintSDLTexts extracts individual column and constraint SDL texts
// from a table chunk for fine-grained usability checks
func extractColumnAndConstraintSDLTexts(chunk *schema.SDLChunk, tableIdentifier string, sdlChunks *currentDatabaseSDLChunks) error {
	if chunk == nil || chunk.ASTNode == nil {
		return nil
	}

	createStmt, ok := chunk.ASTNode.(*parser.CreatestmtContext)
	if !ok {
		return errors.New("chunk AST node is not a CREATE TABLE statement")
	}

	schemaName, tableName := parseIdentifier(tableIdentifier)

	// Extract column definitions with their SDL texts
	columnDefs := extractColumnDefinitionsWithAST(createStmt)
	for columnName, columnDef := range columnDefs.Map {
		columnSDLText := strings.TrimSpace(getColumnText(columnDef.ASTNode))
		columnKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, columnName)
		sdlChunks.columns[columnKey] = columnSDLText
	}

	// Extract constraint definitions with their SDL texts
	// Primary Key constraints
	pkDefs := extractPrimaryKeyDefinitionsWithAST(createStmt)
	for constraintName, pkDef := range pkDefs {
		constraintSDLText := strings.TrimSpace(getIndexText(pkDef.ASTNode))
		constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, constraintName)
		sdlChunks.constraints[constraintKey] = constraintSDLText
	}

	// Unique constraints
	uniqueDefs := extractUniqueConstraintDefinitionsInOrder(createStmt)
	for _, uniqueDef := range uniqueDefs {
		constraintSDLText := strings.TrimSpace(getIndexText(uniqueDef.ASTNode))
		constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, uniqueDef.Name)
		sdlChunks.constraints[constraintKey] = constraintSDLText
	}

	// Check constraints
	checkDefs := extractCheckConstraintDefinitionsWithAST(createStmt)
	for _, checkDef := range checkDefs {
		constraintSDLText := strings.TrimSpace(getCheckConstraintText(checkDef.ASTNode))
		constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, checkDef.Name)
		sdlChunks.constraints[constraintKey] = constraintSDLText
	}

	// Foreign Key constraints
	fkDefs := extractForeignKeyDefinitionsWithAST(createStmt)
	for _, fkDef := range fkDefs {
		constraintSDLText := strings.TrimSpace(getForeignKeyText(fkDef.ASTNode))
		constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, fkDef.Name)
		sdlChunks.constraints[constraintKey] = constraintSDLText
	}

	return nil
}

// shouldSkipChunkDiffForUsability checks if a chunk should skip diff comparison
// by comparing against the pre-computed SDL chunks from current database metadata
func (sdlChunks *currentDatabaseSDLChunks) shouldSkipChunkDiffForUsability(chunkText string, chunkIdentifier string) bool {
	if sdlChunks == nil || len(sdlChunks.chunks) == 0 {
		return false
	}

	// Get the corresponding SDL text from current database metadata
	currentDatabaseSDLText, exists := sdlChunks.chunks[chunkIdentifier]
	if !exists {
		return false
	}

	// Normalize current chunk text for comparison
	normalizedChunkText := strings.TrimSpace(chunkText)

	// If chunk text matches current database metadata SDL, skip the diff (no actual change needed)
	return normalizedChunkText == currentDatabaseSDLText
}

// shouldSkipCommentDiff checks if a comment should skip diff comparison
// by comparing against the pre-computed comment texts from current database metadata
func shouldSkipCommentDiff(commentText string, objectIdentifier string, sdlChunks *currentDatabaseSDLChunks) bool {
	if sdlChunks == nil || len(sdlChunks.comments) == 0 {
		return false
	}

	// Get the corresponding comment text from current database metadata
	currentDatabaseComments, exists := sdlChunks.comments[objectIdentifier]
	if !exists || len(currentDatabaseComments) == 0 {
		return false
	}

	// Normalize current comment text for comparison
	normalizedCommentText := strings.TrimSpace(commentText)

	// Check if comment matches any of the database comments (typically just one)
	for _, dbComment := range currentDatabaseComments {
		if normalizedCommentText == strings.TrimSpace(dbComment) {
			return true
		}
	}

	return false
}

// shouldSkipColumnDiffForUsability checks if a column should skip diff comparison
// by comparing against the pre-computed column SDL texts from current database metadata
func (sdlChunks *currentDatabaseSDLChunks) shouldSkipColumnDiffForUsability(columnText string, schemaName, tableName, columnName string) bool {
	if sdlChunks == nil || len(sdlChunks.columns) == 0 {
		return false
	}

	// Build column key
	columnKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, columnName)

	// Get the corresponding SDL text from current database metadata
	currentDatabaseColumnSDL, exists := sdlChunks.columns[columnKey]
	if !exists {
		return false
	}

	// Normalize current column text for comparison
	normalizedColumnText := strings.TrimSpace(columnText)

	// If column text matches current database metadata SDL, skip the diff (no actual change needed)
	return normalizedColumnText == currentDatabaseColumnSDL
}

// shouldSkipConstraintDiffForUsability checks if a constraint should skip diff comparison
// by comparing against the pre-computed constraint SDL texts from current database metadata
func (sdlChunks *currentDatabaseSDLChunks) shouldSkipConstraintDiffForUsability(constraintText string, schemaName, tableName, constraintName string) bool {
	if sdlChunks == nil || len(sdlChunks.constraints) == 0 {
		return false
	}

	// Build constraint key
	constraintKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, constraintName)

	// Get the corresponding SDL text from current database metadata
	currentDatabaseConstraintSDL, exists := sdlChunks.constraints[constraintKey]
	if !exists {
		return false
	}

	// Normalize current constraint text for comparison
	normalizedConstraintText := strings.TrimSpace(constraintText)

	// If constraint text matches current database metadata SDL, skip the diff (no actual change needed)
	return normalizedConstraintText == currentDatabaseConstraintSDL
}

// convertDatabaseSchemaToSDL converts a model.DatabaseMetadata to SDL format string
// This is used in initialization scenarios where previousUserSDLText is empty
func convertDatabaseSchemaToSDL(dbMetadata *model.DatabaseMetadata) (string, error) {
	if dbMetadata == nil {
		return "", nil
	}

	metadata := dbMetadata.GetProto()
	if metadata == nil {
		return "", nil
	}

	// Use the existing getSDLFormat function from get_database_definition.go
	return getSDLFormat(metadata)
}

// extractCheckConstraintDefinitionsWithAST extracts check constraint definitions with their AST nodes
// Note: This is a wrapper around the existing function with a different name for clarity
func extractCheckConstraintDefinitionsWithAST(createStmt *parser.CreatestmtContext) []*CheckConstraintDefWithAST {
	return extractCheckConstraintDefinitionsInOrder(createStmt)
}

// extractForeignKeyDefinitionsWithAST extracts foreign key constraint definitions with their AST nodes
// Note: This is a wrapper around the existing function with a different name for clarity
func extractForeignKeyDefinitionsWithAST(createStmt *parser.CreatestmtContext) []*ForeignKeyDefWithAST {
	return extractForeignKeyDefinitionsInOrder(createStmt)
}

// PrimaryKeyDefWithAST represents a primary key constraint definition with its AST node
type PrimaryKeyDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// UniqueKeyDefWithAST represents a unique key constraint definition with its AST node
type UniqueKeyDefWithAST struct {
	Name    string
	ASTNode parser.ITableconstraintContext
}

// extractFunctionSignatureFromAST extracts function signature from CREATE FUNCTION AST node
// This is the unified function used by both chunk extraction and function comparison
func extractFunctionSignatureFromAST(ctx *parser.CreatefunctionstmtContext) string {
	if ctx == nil {
		return ""
	}

	funcNameCtx := ctx.Func_name()
	if funcNameCtx == nil {
		return ""
	}

	// Extract function name using proper normalization
	funcNameParts := pgparser.NormalizePostgreSQLFuncName(funcNameCtx)
	if len(funcNameParts) == 0 {
		return ""
	}

	// Get the function name (without schema)
	var functionName string
	if len(funcNameParts) == 2 {
		// Schema qualified: schema.function_name
		functionName = funcNameParts[1]
	} else if len(funcNameParts) == 1 {
		// Unqualified: function_name
		functionName = funcNameParts[0]
	} else {
		// Unexpected format
		return ""
	}

	if functionName == "" {
		return ""
	}

	// Use the existing ExtractFunctionSignature function
	return ExtractFunctionSignature(ctx, functionName)
}

// functionExtractor is a walker to extract CREATE FUNCTION AST nodes
type functionExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.CreatefunctionstmtContext
}

func (e *functionExtractor) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}

// processCommentChanges processes comment changes for all database objects
// It must be called after all object changes have been processed to determine
// which objects were created or dropped (those should not generate comment diffs)

// processEnumTypeChanges analyzes enum type changes between current and previous chunks
// Enum types use DROP + CREATE pattern for modifications (PostgreSQL doesn't support ALTER TYPE ... RENAME VALUE)

// processExtensionChanges processes extension changes between current and previous chunks

// processEventTriggerChanges processes event trigger changes between current and previous chunks

// extractTableNameFromTrigger extracts the fully qualified table name from CREATE TRIGGER ... ON table
// Returns schema.table format, defaulting to public schema if not specified
func extractTableNameFromTrigger(ctx *parser.CreatetrigstmtContext) string {
	if ctx == nil {
		return ""
	}

	// Find ON clause - the table name comes after ON keyword
	// In PostgreSQL grammar: CREATE TRIGGER name ... ON qualified_name
	if ctx.Qualified_name() == nil {
		return ""
	}

	// Extract table name
	qualifiedNameParts := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	if len(qualifiedNameParts) == 0 {
		return ""
	}

	// Return fully qualified name (schema.table)
	if len(qualifiedNameParts) == 1 {
		// No schema specified, default to public
		return "public." + qualifiedNameParts[0]
	}
	// Schema is specified
	return strings.Join(qualifiedNameParts, ".")
}
