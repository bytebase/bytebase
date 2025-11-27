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

func init() {
	schema.RegisterGetSDLDiff(storepb.Engine_POSTGRES, GetSDLDiff)
	schema.RegisterGetSDLDiff(storepb.Engine_COCKROACHDB, GetSDLDiff)
}

func GetSDLDiff(currentSDLText, previousUserSDLText string, currentSchema, previousSchema *model.DatabaseMetadata) (*schema.MetadataDiff, error) {
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

	// Check for drift scenario: when both schemas are provided, apply minimal changes to previousChunks
	if currentSchema != nil && previousSchema != nil {
		err = applyMinimalChangesToChunks(previousChunks, currentSchema, previousSchema)
		if err != nil {
			return nil, errors.Wrap(err, "failed to apply minimal changes to SDL chunks")
		}
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
	err = processTableChanges(currentChunks, previousChunks, currentSchema, previousSchema, currentDBSDLChunks, diff)
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

// applyMinimalChangesToChunks applies minimal changes to the previous SDL chunks based on schema differences
// This implements the minimal change principle for drift scenarios by directly manipulating chunks
func applyMinimalChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get table differences between schemas
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Create maps for efficient table lookup using full schema.table identifier
	currentTables := make(map[string]*storepb.TableMetadata)
	previousTables := make(map[string]*storepb.TableMetadata)

	// Build current tables map with proper schema qualification
	for _, schema := range currentMetadata.Schemas {
		schemaName := schema.Name
		if schemaName == "" {
			schemaName = "public" // Default schema for PostgreSQL
		}
		for _, table := range schema.Tables {
			tableKey := schemaName + "." + table.Name
			currentTables[tableKey] = table
		}
	}

	// Build previous tables map with proper schema qualification
	for _, schema := range previousMetadata.Schemas {
		schemaName := schema.Name
		if schemaName == "" {
			schemaName = "public" // Default schema for PostgreSQL
		}
		for _, table := range schema.Tables {
			tableKey := schemaName + "." + table.Name
			previousTables[tableKey] = table
		}
	}

	// Build existing chunk keys mapping for table operations
	existingChunkKeys := make(map[string]string) // schema.table -> chunk key
	for chunkKey, chunk := range previousChunks.Tables {
		if chunk != nil {
			existingChunkKeys[chunk.Identifier] = chunkKey
		}
	}

	// Build sequences map: schema.table -> sequences for that table
	tableSequencesMap := make(map[string][]*storepb.SequenceMetadata)
	for _, schema := range currentMetadata.Schemas {
		schemaName := schema.Name
		if schemaName == "" {
			schemaName = "public"
		}
		for _, sequence := range schema.Sequences {
			if sequence.OwnerTable != "" {
				tableKey := schemaName + "." + sequence.OwnerTable
				tableSequencesMap[tableKey] = append(tableSequencesMap[tableKey], sequence)
			}
		}
	}

	// Process table additions: add new tables to chunks
	for tableKey, currentTable := range currentTables {
		if _, exists := previousTables[tableKey]; !exists {
			// Table was added - generate SDL for the new table and parse it to AST
			var buf strings.Builder
			schemaName, _ := parseIdentifier(tableKey)
			// Get sequences for this table
			tableSequences := tableSequencesMap[tableKey]
			err := writeCreateTableSDL(&buf, schemaName, currentTable, tableSequences)
			if err != nil {
				return errors.Wrapf(err, "failed to generate SDL for new table %s", tableKey)
			}
			tableSDL := buf.String()

			// Parse the generated SDL to create AST node
			parseResults, err := pgparser.ParsePostgreSQL(tableSDL)
			if err != nil {
				return errors.Wrapf(err, "failed to parse generated SDL for new table %s", tableKey)
			}

			// Extract the CREATE TABLE AST node
			if len(parseResults) != 1 {
				return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
			}
			parseResult := parseResults[0]
			var createTableNode *parser.CreatestmtContext
			antlr.ParseTreeWalkerDefault.Walk(&createTableExtractor{
				result: &createTableNode,
			}, parseResult.Tree)

			if createTableNode == nil {
				return errors.Errorf("failed to extract CREATE TABLE AST node for new table %s", tableKey)
			}

			// Always use schema.table format for all table identifiers
			chunkKey := tableKey

			// Add the new table chunk with proper AST node
			previousChunks.Tables[chunkKey] = &schema.SDLChunk{
				Identifier: chunkKey,
				ASTNode:    createTableNode,
			}
		}
	}

	// Process table modifications: apply column-level changes using ANTLR rewrite
	for tableKey, currentTable := range currentTables {
		if previousTable, exists := previousTables[tableKey]; exists {
			// Table exists in both schemas - check for column differences
			if existingKey, chunkExists := existingChunkKeys[tableKey]; chunkExists {
				if chunk := previousChunks.Tables[existingKey]; chunk != nil {
					// Get sequences for this table
					tableSequences := tableSequencesMap[tableKey]
					// Apply both column and constraint changes to the existing chunk using a single rewriter
					err := applyTableChangesToChunk(chunk, currentTable, previousTable, tableSequences)
					if err != nil {
						return errors.Wrapf(err, "failed to apply table changes to table %s", tableKey)
					}
				}
			}
		}
	}

	// Process table deletions: remove dropped tables from chunks
	for tableKey := range previousTables {
		if _, exists := currentTables[tableKey]; !exists {
			// Table was dropped - find the corresponding chunk key and remove it
			if existingKey, exists := existingChunkKeys[tableKey]; exists {
				delete(previousChunks.Tables, existingKey)
			}
			// If no mapping exists, the table was not in the original chunks,
			// so there's nothing to delete - this is the expected behavior
		}
	}

	// Process standalone index changes: apply minimal changes to index chunks
	err := applyStandaloneIndexChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply standalone index changes")
	}

	// Process function changes: apply minimal changes to function chunks
	err = applyFunctionChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply function changes")
	}

	// Process sequence changes: apply minimal changes to sequence chunks
	err = applySequenceChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply sequence changes")
	}

	// Process view changes: apply minimal changes to view chunks
	err = applyViewChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply view changes")
	}

	// Process materialized view changes: apply minimal changes to materialized view chunks
	err = applyMaterializedViewChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply materialized view changes")
	}

	// Process enum type changes: apply minimal changes to enum type chunks
	err = applyEnumTypeChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply enum type changes")
	}

	// Process extension changes: apply minimal changes to extension chunks
	// Extensions are database-level objects
	err = applyExtensionChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply extension changes")
	}

	// Process trigger changes: apply minimal changes to trigger chunks
	err = applyTriggerChangesToChunks(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply trigger changes")
	}

	// Process column comment changes: sync column comments based on metadata
	err = applyColumnCommentChanges(previousChunks, currentSchema, previousSchema)
	if err != nil {
		return errors.Wrap(err, "failed to apply column comment changes")
	}

	return nil
}

// applyTableChangesToChunk applies minimal column and constraint changes to an existing CREATE TABLE chunk
// by working with the individual chunk's SQL text instead of the full script's tokenStream
func applyTableChangesToChunk(chunk *schema.SDLChunk, currentTable, previousTable *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) error {
	if chunk == nil || chunk.ASTNode == nil || currentTable == nil || previousTable == nil {
		return nil
	}

	// Get the original chunk text
	originalChunkText := chunk.GetText()
	if originalChunkText == "" {
		return errors.New("chunk has no text content")
	}

	// Parse the individual chunk text to get a fresh AST with its own tokenStream
	parseResults, err := pgparser.ParsePostgreSQL(originalChunkText)
	if err != nil {
		return errors.Wrapf(err, "failed to parse original chunk text: %s", originalChunkText)
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	// Extract the CREATE TABLE AST node from the fresh parse
	var createStmt *parser.CreatestmtContext
	antlr.ParseTreeWalkerDefault.Walk(&createTableExtractor{
		result: &createStmt,
	}, parseResult.Tree)

	if createStmt == nil {
		return errors.New("failed to extract CREATE TABLE AST node from chunk text")
	}

	// Get the parser and tokenStream from the fresh parse
	ctxParser := createStmt.GetParser()
	if ctxParser == nil {
		return errors.New("parser not available for fresh AST node")
	}

	tokenStream := ctxParser.GetTokenStream()
	if tokenStream == nil {
		return errors.New("token stream not available for fresh parser")
	}

	// Create rewriter for the individual chunk's tokenStream
	rewriter := antlr.NewTokenStreamRewriter(tokenStream)

	// Apply column changes using the rewriter
	err = applyColumnChanges(rewriter, createStmt, currentTable, previousTable, sequences)
	if err != nil {
		return errors.Wrapf(err, "failed to apply column changes")
	}

	// Apply constraint changes using the same rewriter
	err = applyConstraintChanges(rewriter, createStmt, currentTable, previousTable)
	if err != nil {
		return errors.Wrapf(err, "failed to apply constraint changes")
	}

	// Get the modified SQL from the rewriter
	modifiedSQL := rewriter.GetTextDefault()

	// Parse the modified SQL to get the final AST
	finalParseResult, err := pgparser.ParsePostgreSQL(modifiedSQL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse modified SQL: %s", modifiedSQL)
	}

	if len(finalParseResult) != 1 {
		return errors.Errorf("expected exactly one statement in modified SQL, got %d", len(finalParseResult))
	}

	// Extract the final CREATE TABLE AST node
	var newCreateTableNode *parser.CreatestmtContext
	antlr.ParseTreeWalkerDefault.Walk(&createTableExtractor{
		result: &newCreateTableNode,
	}, finalParseResult[0].Tree)

	if newCreateTableNode == nil {
		return errors.New("failed to extract CREATE TABLE AST node from modified text")
	}

	// Update the chunk with the new AST node
	chunk.ASTNode = newCreateTableNode

	// Synchronize COMMENT ON TABLE statements only if comment has changed
	if currentTable.Comment != previousTable.Comment {
		schemaName, tableName := parseIdentifier(chunk.Identifier)
		if err := syncObjectCommentStatements(chunk, currentTable.Comment, "TABLE", schemaName, tableName); err != nil {
			return errors.Wrapf(err, "failed to sync COMMENT statements for table %s", chunk.Identifier)
		}
	}

	return nil
}

// applyColumnChanges applies column changes using the provided rewriter without parsing SQL
func applyColumnChanges(rewriter *antlr.TokenStreamRewriter, createStmt *parser.CreatestmtContext, currentTable, previousTable *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) error {
	// Create column maps for efficient lookups
	currentColumns := make(map[string]*storepb.ColumnMetadata)
	previousColumns := make(map[string]*storepb.ColumnMetadata)

	for _, col := range currentTable.Columns {
		currentColumns[col.Name] = col
	}
	for _, col := range previousTable.Columns {
		previousColumns[col.Name] = col
	}

	// Extract existing column definitions from AST
	existingColumnDefs := extractColumnDefinitionsWithAST(createStmt)

	// Phase 1: Handle column deletions (process in reverse order to maintain token positions)
	for i := len(existingColumnDefs.Order) - 1; i >= 0; i-- {
		columnName := existingColumnDefs.Order[i]
		if _, exists := currentColumns[columnName]; !exists {
			// Column was deleted
			columnDef := existingColumnDefs.Map[columnName]
			err := deleteColumnFromAST(rewriter, columnDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete column %s", columnName)
			}
		}
	}

	// Phase 2: Handle column modifications
	for _, columnName := range existingColumnDefs.Order {
		if currentCol, currentExists := currentColumns[columnName]; currentExists {
			if previousCol, previousExists := previousColumns[columnName]; previousExists {
				// Column exists in both - check if modified
				if !columnsEqual(currentCol, previousCol) {
					columnDef := existingColumnDefs.Map[columnName]
					err := modifyColumnInAST(rewriter, columnDef.ASTNode, currentCol, currentTable.Name, sequences)
					if err != nil {
						return errors.Wrapf(err, "failed to modify column %s", columnName)
					}
				}
			}
		}
	}

	// Phase 3: Handle column additions (add at the end)
	for _, currentCol := range currentTable.Columns {
		if _, exists := previousColumns[currentCol.Name]; !exists {
			// Column was added
			err := addColumnToAST(rewriter, createStmt, currentCol, currentTable.Name, sequences)
			if err != nil {
				return errors.Wrapf(err, "failed to add column %s", currentCol.Name)
			}
		}
	}

	return nil
}

// applyConstraintChanges applies constraint changes using the provided rewriter without parsing SQL
func applyConstraintChanges(rewriter *antlr.TokenStreamRewriter, createStmt *parser.CreatestmtContext, currentTable, previousTable *storepb.TableMetadata) error {
	// Create constraint maps for efficient lookups
	currentCheckConstraints := make(map[string]*storepb.CheckConstraintMetadata)
	previousCheckConstraints := make(map[string]*storepb.CheckConstraintMetadata)
	currentFKConstraints := make(map[string]*storepb.ForeignKeyMetadata)
	previousFKConstraints := make(map[string]*storepb.ForeignKeyMetadata)
	currentPKConstraints := make(map[string]*storepb.IndexMetadata)
	previousPKConstraints := make(map[string]*storepb.IndexMetadata)
	currentUKConstraints := make(map[string]*storepb.IndexMetadata)
	previousUKConstraints := make(map[string]*storepb.IndexMetadata)
	currentExcludeConstraints := make(map[string]*storepb.ExcludeConstraintMetadata)
	previousExcludeConstraints := make(map[string]*storepb.ExcludeConstraintMetadata)

	// Build constraint maps from metadata
	for _, constraint := range currentTable.CheckConstraints {
		currentCheckConstraints[constraint.Name] = constraint
	}
	for _, constraint := range previousTable.CheckConstraints {
		previousCheckConstraints[constraint.Name] = constraint
	}
	for _, constraint := range currentTable.ForeignKeys {
		currentFKConstraints[constraint.Name] = constraint
	}
	for _, constraint := range previousTable.ForeignKeys {
		previousFKConstraints[constraint.Name] = constraint
	}
	// Build primary key constraint maps
	for _, index := range currentTable.Indexes {
		if index.Primary {
			currentPKConstraints[index.Name] = index
		}
	}
	for _, index := range previousTable.Indexes {
		if index.Primary {
			previousPKConstraints[index.Name] = index
		}
	}
	// Build unique key constraint maps (unique constraints, not just indexes)
	for _, index := range currentTable.Indexes {
		if index.Unique && !index.Primary && index.IsConstraint {
			currentUKConstraints[index.Name] = index
		}
	}
	for _, index := range previousTable.Indexes {
		if index.Unique && !index.Primary && index.IsConstraint {
			previousUKConstraints[index.Name] = index
		}
	}
	// Build EXCLUDE constraint maps
	for _, constraint := range currentTable.ExcludeConstraints {
		currentExcludeConstraints[constraint.Name] = constraint
	}
	for _, constraint := range previousTable.ExcludeConstraints {
		previousExcludeConstraints[constraint.Name] = constraint
	}

	// Extract constraint definitions with AST nodes for precise manipulation
	currentCheckDefs := extractCheckConstraintDefinitionsWithAST(createStmt)
	currentFKDefs := extractForeignKeyDefinitionsWithAST(createStmt)
	currentPKDefs := extractPrimaryKeyDefinitionsInOrder(createStmt)
	currentUKDefs := extractUniqueKeyDefinitionsInOrder(createStmt)
	currentExcludeDefs := extractExcludeConstraintDefinitionsWithAST(createStmt)

	// Phase 1: Handle constraint deletions (reverse order for stability)
	// Delete check constraints
	for i := len(currentCheckDefs) - 1; i >= 0; i-- {
		checkDef := currentCheckDefs[i]
		if _, exists := currentCheckConstraints[checkDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, checkDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete check constraint %s", checkDef.Name)
			}
		}
	}

	// Delete foreign key constraints
	for i := len(currentFKDefs) - 1; i >= 0; i-- {
		fkDef := currentFKDefs[i]
		if _, exists := currentFKConstraints[fkDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, fkDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete foreign key constraint %s", fkDef.Name)
			}
		}
	}

	// Delete primary key constraints
	for i := len(currentPKDefs) - 1; i >= 0; i-- {
		pkDef := currentPKDefs[i]
		if _, exists := currentPKConstraints[pkDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, pkDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete primary key constraint %s", pkDef.Name)
			}
		}
	}

	// Delete unique key constraints
	for i := len(currentUKDefs) - 1; i >= 0; i-- {
		ukDef := currentUKDefs[i]
		if _, exists := currentUKConstraints[ukDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, ukDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete unique key constraint %s", ukDef.Name)
			}
		}
	}

	// Delete EXCLUDE constraints
	for i := len(currentExcludeDefs) - 1; i >= 0; i-- {
		excludeDef := currentExcludeDefs[i]
		if _, exists := currentExcludeConstraints[excludeDef.Name]; !exists {
			// Constraint was dropped
			err := deleteConstraintFromAST(rewriter, excludeDef.ASTNode, createStmt)
			if err != nil {
				return errors.Wrapf(err, "failed to delete exclude constraint %s", excludeDef.Name)
			}
		}
	}

	// Phase 2: Handle constraint modifications
	// Modify check constraints
	for _, checkDef := range currentCheckDefs {
		if currentConstraint, exists := currentCheckConstraints[checkDef.Name]; exists {
			if previousConstraint, wasPresent := previousCheckConstraints[checkDef.Name]; wasPresent {
				// Check if constraint was modified by comparing text
				if !constraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, checkDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify check constraint %s", checkDef.Name)
					}
				}
			}
		}
	}

	// Modify foreign key constraints
	for _, fkDef := range currentFKDefs {
		if currentConstraint, exists := currentFKConstraints[fkDef.Name]; exists {
			if previousConstraint, wasPresent := previousFKConstraints[fkDef.Name]; wasPresent {
				// Check if constraint was modified
				if !fkConstraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, fkDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify foreign key constraint %s", fkDef.Name)
					}
				}
			}
		}
	}

	// Modify primary key constraints
	for _, pkDef := range currentPKDefs {
		if currentConstraint, exists := currentPKConstraints[pkDef.Name]; exists {
			if previousConstraint, wasPresent := previousPKConstraints[pkDef.Name]; wasPresent {
				// Check if constraint was modified
				if !pkConstraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, pkDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify primary key constraint %s", pkDef.Name)
					}
				}
			}
		}
	}

	// Modify unique key constraints
	for _, ukDef := range currentUKDefs {
		if currentConstraint, exists := currentUKConstraints[ukDef.Name]; exists {
			if previousConstraint, wasPresent := previousUKConstraints[ukDef.Name]; wasPresent {
				// Check if constraint was modified
				if !ukConstraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, ukDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify unique key constraint %s", ukDef.Name)
					}
				}
			}
		}
	}

	// Modify EXCLUDE constraints
	for _, excludeDef := range currentExcludeDefs {
		if currentConstraint, exists := currentExcludeConstraints[excludeDef.Name]; exists {
			if previousConstraint, wasPresent := previousExcludeConstraints[excludeDef.Name]; wasPresent {
				// Check if constraint was modified
				if !excludeConstraintsEqual(currentConstraint, previousConstraint) {
					err := modifyConstraintInAST(rewriter, excludeDef.ASTNode, currentConstraint)
					if err != nil {
						return errors.Wrapf(err, "failed to modify exclude constraint %s", excludeDef.Name)
					}
				}
			}
		}
	}

	// Phase 3: Handle constraint additions
	// Add new check constraints
	for _, currentConstraint := range currentTable.CheckConstraints {
		if _, existed := previousCheckConstraints[currentConstraint.Name]; !existed {
			// New check constraint
			err := addConstraintToAST(rewriter, createStmt, currentConstraint)
			if err != nil {
				return errors.Wrapf(err, "failed to add check constraint %s", currentConstraint.Name)
			}
		}
	}

	// Add new foreign key constraints
	for _, currentConstraint := range currentTable.ForeignKeys {
		if _, existed := previousFKConstraints[currentConstraint.Name]; !existed {
			// New foreign key constraint
			err := addConstraintToAST(rewriter, createStmt, currentConstraint)
			if err != nil {
				return errors.Wrapf(err, "failed to add foreign key constraint %s", currentConstraint.Name)
			}
		}
	}

	// Add new primary key constraints
	for _, currentIndex := range currentTable.Indexes {
		if currentIndex.Primary {
			if _, existed := previousPKConstraints[currentIndex.Name]; !existed {
				// New primary key constraint
				err := addConstraintToAST(rewriter, createStmt, currentIndex)
				if err != nil {
					return errors.Wrapf(err, "failed to add primary key constraint %s", currentIndex.Name)
				}
			}
		}
	}

	// Add new unique key constraints
	for _, currentIndex := range currentTable.Indexes {
		if currentIndex.Unique && !currentIndex.Primary && currentIndex.IsConstraint {
			if _, existed := previousUKConstraints[currentIndex.Name]; !existed {
				// New unique key constraint
				err := addConstraintToAST(rewriter, createStmt, currentIndex)
				if err != nil {
					return errors.Wrapf(err, "failed to add unique key constraint %s", currentIndex.Name)
				}
			}
		}
	}

	// Add new EXCLUDE constraints
	for _, currentConstraint := range currentTable.ExcludeConstraints {
		if _, existed := previousExcludeConstraints[currentConstraint.Name]; !existed {
			// New EXCLUDE constraint
			err := addConstraintToAST(rewriter, createStmt, currentConstraint)
			if err != nil {
				return errors.Wrapf(err, "failed to add exclude constraint %s", currentConstraint.Name)
			}
		}
	}

	return nil
}

// deleteColumnFromAST removes a column definition from the CREATE TABLE statement using token rewriter
// Improved comma handling: always look for a following comma first, regardless of column position
func deleteColumnFromAST(rewriter *antlr.TokenStreamRewriter, columnDef parser.IColumnDefContext, _ *parser.CreatestmtContext) error {
	if columnDef == nil {
		return errors.New("column definition is nil")
	}

	startToken := columnDef.GetStart()
	stopToken := columnDef.GetStop()
	if startToken == nil || stopToken == nil {
		return errors.New("unable to get column definition tokens")
	}

	// Find the actual deletion range including commas and whitespace
	deleteStartIndex := startToken.GetTokenIndex()
	deleteEndIndex := stopToken.GetTokenIndex()

	// Strategy: Always try to remove the following comma first
	// This handles cases where there are table constraints after columns
	nextCommaIndex := -1
	nextCommaEndIndex := -1
	for i := stopToken.GetTokenIndex() + 1; i < rewriter.GetTokenStream().Size(); i++ {
		token := rewriter.GetTokenStream().Get(i)
		if token.GetTokenType() == parser.PostgreSQLParserCOMMA {
			nextCommaIndex = i
			// Look for whitespace after the comma to include in deletion
			nextCommaEndIndex = i
			for j := i + 1; j < rewriter.GetTokenStream().Size(); j++ {
				nextToken := rewriter.GetTokenStream().Get(j)
				// Include whitespace and newlines after the comma
				if nextToken.GetChannel() != antlr.TokenDefaultChannel {
					nextCommaEndIndex = j
				} else {
					break
				}
			}
			break
		}
		// Skip whitespace and comments, but stop at other meaningful tokens
		if token.GetChannel() == antlr.TokenDefaultChannel {
			// Found a non-comma token on the default channel, stop searching
			break
		}
	}

	if nextCommaIndex != -1 {
		// Found a following comma - remove column, comma, and trailing whitespace
		deleteEndIndex = nextCommaEndIndex
	} else {
		// No following comma found - this might be the last element
		// Try to find a preceding comma to remove
		prevCommaIndex := -1
		prevCommaStartIndex := -1
		for i := startToken.GetTokenIndex() - 1; i >= 0; i-- {
			token := rewriter.GetTokenStream().Get(i)
			if token.GetTokenType() == parser.PostgreSQLParserCOMMA {
				prevCommaIndex = i
				// Look for whitespace before the comma to include in deletion
				prevCommaStartIndex = i
				for j := i - 1; j >= 0; j-- {
					prevToken := rewriter.GetTokenStream().Get(j)
					// Include whitespace and newlines before the comma
					if prevToken.GetChannel() != antlr.TokenDefaultChannel {
						prevCommaStartIndex = j
					} else {
						break
					}
				}
				break
			}
			// Skip whitespace and comments, but stop at other meaningful tokens
			if token.GetChannel() == antlr.TokenDefaultChannel {
				break
			}
		}

		if prevCommaIndex != -1 {
			// Found a preceding comma - remove it along with leading whitespace and the column
			deleteStartIndex = prevCommaStartIndex
		} else {
			// No comma found (single column case) - just remove the column
			// But also clean up any trailing whitespace that might leave empty lines
			for i := stopToken.GetTokenIndex() + 1; i < rewriter.GetTokenStream().Size(); i++ {
				token := rewriter.GetTokenStream().Get(i)
				if token.GetChannel() != antlr.TokenDefaultChannel {
					deleteEndIndex = i
				} else {
					break
				}
			}
		}
	}

	// Perform the deletion with the computed range
	rewriter.DeleteDefault(deleteStartIndex, deleteEndIndex)

	return nil
}

// modifyColumnInAST modifies an existing column definition using token rewriter
func modifyColumnInAST(rewriter *antlr.TokenStreamRewriter, columnDef parser.IColumnDefContext, newColumn *storepb.ColumnMetadata, tableName string, sequences []*storepb.SequenceMetadata) error {
	if columnDef == nil || newColumn == nil {
		return errors.New("column definition or new column metadata is nil")
	}

	startToken := columnDef.GetStart()
	stopToken := columnDef.GetStop()
	if startToken == nil || stopToken == nil {
		return errors.New("unable to get column definition tokens")
	}

	// Generate new column definition SDL
	newColumnSDL := generateColumnSDL(newColumn, tableName, sequences)

	// Replace the entire column definition
	rewriter.ReplaceDefault(startToken.GetTokenIndex(), stopToken.GetTokenIndex(), newColumnSDL)

	return nil
}

// addColumnToAST adds a new column definition to the CREATE TABLE statement
func addColumnToAST(rewriter *antlr.TokenStreamRewriter, createStmt *parser.CreatestmtContext, newColumn *storepb.ColumnMetadata, tableName string, sequences []*storepb.SequenceMetadata) error {
	if createStmt == nil || newColumn == nil {
		return errors.New("create statement or new column metadata is nil")
	}

	// Find the last column definition to insert after it
	columnDefs := extractColumnDefinitionsWithAST(createStmt)

	if len(columnDefs.Order) == 0 {
		// No existing columns - need to add first column to empty table
		// Find the opening parenthesis and insert after it
		if createStmt.Opttableelementlist() != nil {
			// Table has element list structure, find the position to insert
			for i := 0; i < rewriter.GetTokenStream().Size(); i++ {
				token := rewriter.GetTokenStream().Get(i)
				if token.GetTokenType() == parser.PostgreSQLParserOPEN_PAREN {
					// Found opening parenthesis, insert new column after it
					newColumnSDL := "\n    " + generateColumnSDL(newColumn, tableName, sequences) + "\n"
					rewriter.InsertAfterDefault(i, newColumnSDL)
					return nil
				}
			}
		}
		return errors.New("unable to find position to insert first column")
	}

	// Get the last column definition
	lastColumnName := columnDefs.Order[len(columnDefs.Order)-1]
	lastColumnDef := columnDefs.Map[lastColumnName]

	stopToken := lastColumnDef.ASTNode.GetStop()
	if stopToken == nil {
		return errors.New("unable to get last column stop token")
	}

	// Generate new column definition SDL with leading comma and proper indentation
	newColumnSDL := ",\n    " + generateColumnSDL(newColumn, tableName, sequences)

	// Insert after the last column
	rewriter.InsertAfterDefault(stopToken.GetTokenIndex(), newColumnSDL)

	return nil
}

// generateColumnSDL generates SDL text for a single column definition using the extracted writeColumnSDL function
func generateColumnSDL(column *storepb.ColumnMetadata, tableName string, sequences []*storepb.SequenceMetadata) string {
	if column == nil {
		return ""
	}

	var buf strings.Builder
	err := writeColumnSDL(&buf, column, tableName, sequences)
	if err != nil {
		// If there's an error writing to the buffer, return empty string
		// This should rarely happen since we're writing to a strings.Builder
		return ""
	}

	return buf.String()
}

// columnsEqual compares two column metadata objects for equality
func columnsEqual(a, b *storepb.ColumnMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Name == b.Name &&
		a.Type == b.Type &&
		a.Nullable == b.Nullable &&
		a.Default == b.Default &&
		a.Collation == b.Collation
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

// deleteConstraintFromAST removes a constraint definition from the CREATE TABLE statement using token rewriter
func deleteConstraintFromAST(rewriter *antlr.TokenStreamRewriter, constraintAST parser.ITableconstraintContext, _ *parser.CreatestmtContext) error {
	if constraintAST == nil || rewriter == nil {
		return errors.New("constraint AST or rewriter is nil")
	}

	// Get start and stop tokens for the constraint
	startToken := constraintAST.GetStart()
	stopToken := constraintAST.GetStop()

	if startToken == nil || stopToken == nil {
		return errors.New("unable to get constraint definition tokens")
	}

	// Find the actual deletion range including commas and whitespace
	deleteStartIndex := startToken.GetTokenIndex()
	deleteEndIndex := stopToken.GetTokenIndex()

	// Strategy: Always try to remove the following comma first
	// This handles cases where there are more constraints after this one
	nextCommaIndex := -1
	nextCommaEndIndex := -1
	for i := stopToken.GetTokenIndex() + 1; i < rewriter.GetTokenStream().Size(); i++ {
		token := rewriter.GetTokenStream().Get(i)
		if token.GetTokenType() == parser.PostgreSQLParserCOMMA {
			nextCommaIndex = i
			// Look for whitespace after the comma to include in deletion
			nextCommaEndIndex = i
			for j := i + 1; j < rewriter.GetTokenStream().Size(); j++ {
				nextToken := rewriter.GetTokenStream().Get(j)
				// Include whitespace and newlines after the comma
				if nextToken.GetChannel() != antlr.TokenDefaultChannel {
					nextCommaEndIndex = j
				} else {
					break
				}
			}
			break
		}
		// Skip whitespace and comments, but stop at other meaningful tokens
		if token.GetChannel() == antlr.TokenDefaultChannel {
			// Found a non-comma token on the default channel, stop searching
			break
		}
	}

	if nextCommaIndex != -1 {
		// Found a following comma - remove constraint, comma, and trailing whitespace
		deleteEndIndex = nextCommaEndIndex
	} else {
		// No following comma found - this might be the last element
		// Try to find a preceding comma to remove
		prevCommaIndex := -1
		prevCommaStartIndex := -1
		for i := startToken.GetTokenIndex() - 1; i >= 0; i-- {
			token := rewriter.GetTokenStream().Get(i)
			if token.GetTokenType() == parser.PostgreSQLParserCOMMA {
				prevCommaIndex = i
				// Look for whitespace before the comma to include in deletion
				prevCommaStartIndex = i
				for j := i - 1; j >= 0; j-- {
					prevToken := rewriter.GetTokenStream().Get(j)
					// Include whitespace and newlines before the comma
					if prevToken.GetChannel() != antlr.TokenDefaultChannel {
						prevCommaStartIndex = j
					} else {
						break
					}
				}
				break
			}
			// Skip whitespace and comments, but stop at other meaningful tokens
			if token.GetChannel() == antlr.TokenDefaultChannel {
				break
			}
		}

		if prevCommaIndex != -1 {
			// Found a preceding comma - remove it along with leading whitespace and the constraint
			deleteStartIndex = prevCommaStartIndex
		} else {
			// No comma found (single constraint case) - just remove the constraint
			// But also clean up any trailing whitespace that might leave empty lines
			for i := stopToken.GetTokenIndex() + 1; i < rewriter.GetTokenStream().Size(); i++ {
				token := rewriter.GetTokenStream().Get(i)
				if token.GetChannel() != antlr.TokenDefaultChannel {
					deleteEndIndex = i
				} else {
					break
				}
			}
		}
	}

	// Perform the deletion with the computed range
	rewriter.DeleteDefault(deleteStartIndex, deleteEndIndex)

	return nil
}

// modifyConstraintInAST modifies a constraint definition using token rewriter
func modifyConstraintInAST(rewriter *antlr.TokenStreamRewriter, constraintAST parser.ITableconstraintContext, newConstraint any) error {
	if constraintAST == nil || rewriter == nil || newConstraint == nil {
		return errors.New("constraint AST, rewriter, or new constraint is nil")
	}

	// Get start and stop tokens for the constraint
	startToken := constraintAST.GetStart()
	stopToken := constraintAST.GetStop()

	if startToken == nil || stopToken == nil {
		return errors.New("unable to get constraint definition tokens")
	}

	// Generate the new constraint SDL
	var newConstraintSDL string
	switch constraint := newConstraint.(type) {
	case *storepb.CheckConstraintMetadata:
		newConstraintSDL = generateCheckConstraintSDL(constraint)
	case *storepb.ForeignKeyMetadata:
		newConstraintSDL = generateForeignKeyConstraintSDL(constraint)
	case *storepb.IndexMetadata:
		if constraint.Primary {
			newConstraintSDL = generatePrimaryKeyConstraintSDL(constraint)
		} else if constraint.Unique && constraint.IsConstraint {
			newConstraintSDL = generateUniqueKeyConstraintSDL(constraint)
		} else {
			return errors.New("unsupported index constraint type")
		}
	case *storepb.ExcludeConstraintMetadata:
		newConstraintSDL = generateExcludeConstraintSDL(constraint)
	default:
		return errors.New("unsupported constraint type")
	}

	// Replace the entire constraint definition
	rewriter.ReplaceDefault(startToken.GetTokenIndex(), stopToken.GetTokenIndex(), newConstraintSDL)

	return nil
}

// addConstraintToAST adds a new constraint to the CREATE TABLE statement using token rewriter
func addConstraintToAST(rewriter *antlr.TokenStreamRewriter, createStmt *parser.CreatestmtContext, newConstraint any) error {
	if rewriter == nil || createStmt == nil || newConstraint == nil {
		return errors.New("rewriter, create statement, or new constraint is nil")
	}

	// Generate the new constraint SDL
	var newConstraintSDL string
	switch constraint := newConstraint.(type) {
	case *storepb.CheckConstraintMetadata:
		newConstraintSDL = generateCheckConstraintSDL(constraint)
	case *storepb.ForeignKeyMetadata:
		newConstraintSDL = generateForeignKeyConstraintSDL(constraint)
	case *storepb.IndexMetadata:
		if constraint.Primary {
			newConstraintSDL = generatePrimaryKeyConstraintSDL(constraint)
		} else if constraint.Unique && constraint.IsConstraint {
			newConstraintSDL = generateUniqueKeyConstraintSDL(constraint)
		} else {
			return errors.New("unsupported index constraint type")
		}
	case *storepb.ExcludeConstraintMetadata:
		newConstraintSDL = generateExcludeConstraintSDL(constraint)
	default:
		return errors.New("unsupported constraint type")
	}

	// Find the position to insert the constraint
	// Look for the closing parenthesis of the CREATE TABLE statement
	optTableElementList := createStmt.Opttableelementlist()
	if optTableElementList == nil {
		return errors.New("CREATE TABLE statement has no table element list")
	}

	tableElementList := optTableElementList.Tableelementlist()
	if tableElementList == nil {
		return errors.New("table element list is nil")
	}

	// Get all table elements to find the last one
	tableElements := tableElementList.AllTableelement()
	if len(tableElements) == 0 {
		return errors.New("no table elements found")
	}

	// Get the last element (could be a column or constraint)
	lastElement := tableElements[len(tableElements)-1]
	stopToken := lastElement.GetStop()
	if stopToken == nil {
		return errors.New("unable to get last table element stop token")
	}

	// Generate constraint definition SDL with leading comma and proper indentation
	constraintSDL := ",\n    " + newConstraintSDL

	// Insert after the last element
	rewriter.InsertAfterDefault(stopToken.GetTokenIndex(), constraintSDL)

	return nil
}

// generateCheckConstraintSDL generates SDL text for a check constraint using the existing writeCheckConstraintSDL function
func generateCheckConstraintSDL(constraint *storepb.CheckConstraintMetadata) string {
	if constraint == nil {
		return ""
	}

	var buf strings.Builder
	err := writeCheckConstraintSDL(&buf, constraint)
	if err != nil {
		// If there's an error writing to the buffer, return empty string
		// This should rarely happen since we're writing to a strings.Builder
		return ""
	}

	return buf.String()
}

// generateForeignKeyConstraintSDL generates SDL text for a foreign key constraint using the existing writeForeignKeyConstraintSDL function
func generateForeignKeyConstraintSDL(constraint *storepb.ForeignKeyMetadata) string {
	if constraint == nil {
		return ""
	}

	var buf strings.Builder
	err := writeForeignKeyConstraintSDL(&buf, constraint)
	if err != nil {
		// If there's an error writing to the buffer, return empty string
		// This should rarely happen since we're writing to a strings.Builder
		return ""
	}

	return buf.String()
}

// generateExcludeConstraintSDL generates SDL text for an EXCLUDE constraint using the existing writeExcludeConstraintSDL function
func generateExcludeConstraintSDL(constraint *storepb.ExcludeConstraintMetadata) string {
	if constraint == nil {
		return ""
	}

	var buf strings.Builder
	err := writeExcludeConstraintSDL(&buf, constraint)
	if err != nil {
		// If there's an error writing to the buffer, return empty string
		// This should rarely happen since we're writing to a strings.Builder
		return ""
	}

	return buf.String()
}

// constraintsEqual compares two check constraint metadata objects for equality
func constraintsEqual(a, b *storepb.CheckConstraintMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Name == b.Name && a.Expression == b.Expression
}

// fkConstraintsEqual compares two foreign key constraint metadata objects for equality
func fkConstraintsEqual(a, b *storepb.ForeignKeyMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Compare basic properties
	if a.Name != b.Name ||
		a.ReferencedSchema != b.ReferencedSchema ||
		a.ReferencedTable != b.ReferencedTable ||
		a.OnDelete != b.OnDelete ||
		a.OnUpdate != b.OnUpdate {
		return false
	}

	// Compare columns arrays
	if len(a.Columns) != len(b.Columns) ||
		len(a.ReferencedColumns) != len(b.ReferencedColumns) {
		return false
	}

	for i := range a.Columns {
		if a.Columns[i] != b.Columns[i] {
			return false
		}
	}

	for i := range a.ReferencedColumns {
		if a.ReferencedColumns[i] != b.ReferencedColumns[i] {
			return false
		}
	}

	return true
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

// extractExcludeConstraintDefinitionsWithAST extracts EXCLUDE constraint definitions with their AST nodes
// Note: This is a wrapper around the existing function with a different name for clarity
func extractExcludeConstraintDefinitionsWithAST(createStmt *parser.CreatestmtContext) []*ExcludeConstraintDefWithAST {
	return extractExcludeConstraintDefinitionsInOrder(createStmt)
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

// extractPrimaryKeyDefinitionsInOrder extracts primary key constraint definitions with their AST nodes
func extractPrimaryKeyDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*PrimaryKeyDefWithAST {
	var pkDefs []*PrimaryKeyDefWithAST

	if createStmt == nil {
		return pkDefs
	}

	// Navigate through the CREATE TABLE AST to find table constraints
	optTableElementList := createStmt.Opttableelementlist()
	if optTableElementList == nil {
		return pkDefs
	}

	tableElementList := optTableElementList.(*parser.OpttableelementlistContext).Tableelementlist()
	if tableElementList == nil {
		return pkDefs
	}

	// Iterate through table elements to find constraints
	for _, element := range tableElementList.(*parser.TableelementlistContext).AllTableelement() {
		if tableConstraint := element.(*parser.TableelementContext).Tableconstraint(); tableConstraint != nil {
			constraint, ok := tableConstraint.(*parser.TableconstraintContext)
			if !ok {
				continue
			}

			// Check if it's a primary key constraint
			if constraint.Constraintelem() != nil {
				elem, ok := constraint.Constraintelem().(*parser.ConstraintelemContext)
				if !ok {
					continue
				}
				if elem.PRIMARY() != nil && elem.KEY() != nil {
					// Extract constraint name
					constraintName := ""
					if constraint.Name() != nil {
						constraintName = pgparser.NormalizePostgreSQLName(constraint.Name())
					}

					if constraintName != "" {
						pkDefs = append(pkDefs, &PrimaryKeyDefWithAST{
							Name:    constraintName,
							ASTNode: constraint,
						})
					}
				}
			}
		}
	}

	return pkDefs
}

// extractUniqueKeyDefinitionsInOrder extracts unique key constraint definitions with their AST nodes
func extractUniqueKeyDefinitionsInOrder(createStmt *parser.CreatestmtContext) []*UniqueKeyDefWithAST {
	var ukDefs []*UniqueKeyDefWithAST

	if createStmt == nil {
		return ukDefs
	}

	// Navigate through the CREATE TABLE AST to find table constraints
	optTableElementList := createStmt.Opttableelementlist()
	if optTableElementList == nil {
		return ukDefs
	}

	tableElementList := optTableElementList.(*parser.OpttableelementlistContext).Tableelementlist()
	if tableElementList == nil {
		return ukDefs
	}

	// Iterate through table elements to find constraints
	for _, element := range tableElementList.(*parser.TableelementlistContext).AllTableelement() {
		if tableConstraint := element.(*parser.TableelementContext).Tableconstraint(); tableConstraint != nil {
			constraint, ok := tableConstraint.(*parser.TableconstraintContext)
			if !ok {
				continue
			}

			// Check if it's a unique constraint
			if constraint.Constraintelem() != nil {
				elem, ok := constraint.Constraintelem().(*parser.ConstraintelemContext)
				if !ok {
					continue
				}
				if elem.UNIQUE() != nil {
					// Extract constraint name
					constraintName := ""
					if constraint.Name() != nil {
						constraintName = pgparser.NormalizePostgreSQLName(constraint.Name())
					}

					if constraintName != "" {
						ukDefs = append(ukDefs, &UniqueKeyDefWithAST{
							Name:    constraintName,
							ASTNode: constraint,
						})
					}
				}
			}
		}
	}

	return ukDefs
}

// pkConstraintsEqual checks if two primary key constraints are equal
func pkConstraintsEqual(a, b *storepb.IndexMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Compare basic properties
	if a.Name != b.Name || a.Primary != b.Primary {
		return false
	}

	// Compare expressions (columns)
	if len(a.Expressions) != len(b.Expressions) {
		return false
	}

	for i := range a.Expressions {
		if a.Expressions[i] != b.Expressions[i] {
			return false
		}
	}

	return true
}

// ukConstraintsEqual checks if two unique key constraints are equal
func ukConstraintsEqual(a, b *storepb.IndexMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Compare basic properties
	if a.Name != b.Name || a.Unique != b.Unique || a.IsConstraint != b.IsConstraint {
		return false
	}

	// Compare expressions (columns)
	if len(a.Expressions) != len(b.Expressions) {
		return false
	}

	for i := range a.Expressions {
		if a.Expressions[i] != b.Expressions[i] {
			return false
		}
	}

	return true
}

// excludeConstraintsEqual compares two EXCLUDE constraint metadata objects for equality
func excludeConstraintsEqual(a, b *storepb.ExcludeConstraintMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Name == b.Name && a.Expression == b.Expression
}

// generatePrimaryKeyConstraintSDL generates SDL text for a primary key constraint
func generatePrimaryKeyConstraintSDL(constraint *storepb.IndexMetadata) string {
	if constraint == nil || !constraint.Primary {
		return ""
	}

	var buf strings.Builder
	err := writePrimaryKeyConstraintSDL(&buf, constraint)
	if err != nil {
		return ""
	}
	return buf.String()
}

// generateUniqueKeyConstraintSDL generates SDL text for a unique key constraint
func generateUniqueKeyConstraintSDL(constraint *storepb.IndexMetadata) string {
	if constraint == nil || !constraint.Unique || constraint.Primary || !constraint.IsConstraint {
		return ""
	}

	var buf strings.Builder
	err := writeUniqueKeyConstraintSDL(&buf, constraint)
	if err != nil {
		return ""
	}
	return buf.String()
}

// extendedIndexMetadata stores index metadata with table and schema context
type extendedIndexMetadata struct {
	*storepb.IndexMetadata
	SchemaName string
	TableName  string // Table name or MaterializedView name
	TargetType string // "table" or "materialized_view"
}

// applyStandaloneIndexChangesToChunks applies minimal changes to standalone CREATE INDEX chunks
// This handles creation, modification, and deletion of independent index statements
func applyStandaloneIndexChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get index differences by comparing schema metadata
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build index maps for current and previous schemas
	currentIndexes := make(map[string]*extendedIndexMetadata)
	previousIndexes := make(map[string]*extendedIndexMetadata)

	// Collect all standalone indexes from current schema (only non-constraint indexes)
	for _, schema := range currentMetadata.Schemas {
		// Collect indexes from tables
		for _, table := range schema.Tables {
			for _, index := range table.Indexes {
				// Only include standalone indexes (not constraints like PRIMARY KEY, UNIQUE CONSTRAINT)
				if !index.IsConstraint && !index.Primary {
					indexKey := formatIndexKey(schema.Name, index.Name)
					// Store extended index metadata with table/schema context
					extendedIndex := &extendedIndexMetadata{
						IndexMetadata: index,
						SchemaName:    schema.Name,
						TableName:     table.Name,
						TargetType:    "table",
					}
					currentIndexes[indexKey] = extendedIndex
				}
			}
		}

		// Collect indexes from materialized views
		for _, mv := range schema.MaterializedViews {
			for _, index := range mv.Indexes {
				// Only include standalone indexes (not constraints)
				if !index.IsConstraint && !index.Primary {
					indexKey := formatIndexKey(schema.Name, index.Name)
					// Store extended index metadata with materialized view/schema context
					extendedIndex := &extendedIndexMetadata{
						IndexMetadata: index,
						SchemaName:    schema.Name,
						TableName:     mv.Name,
						TargetType:    "materialized_view",
					}
					currentIndexes[indexKey] = extendedIndex
				}
			}
		}
	}

	// Collect all standalone indexes from previous schema
	for _, schema := range previousMetadata.Schemas {
		// Collect indexes from tables
		for _, table := range schema.Tables {
			for _, index := range table.Indexes {
				// Only include standalone indexes (not constraints)
				if !index.IsConstraint && !index.Primary {
					indexKey := formatIndexKey(schema.Name, index.Name)
					extendedIndex := &extendedIndexMetadata{
						IndexMetadata: index,
						SchemaName:    schema.Name,
						TableName:     table.Name,
						TargetType:    "table",
					}
					previousIndexes[indexKey] = extendedIndex
				}
			}
		}

		// Collect indexes from materialized views
		for _, mv := range schema.MaterializedViews {
			for _, index := range mv.Indexes {
				// Only include standalone indexes (not constraints)
				if !index.IsConstraint && !index.Primary {
					indexKey := formatIndexKey(schema.Name, index.Name)
					extendedIndex := &extendedIndexMetadata{
						IndexMetadata: index,
						SchemaName:    schema.Name,
						TableName:     mv.Name,
						TargetType:    "materialized_view",
					}
					previousIndexes[indexKey] = extendedIndex
				}
			}
		}
	}

	// Process index additions: create new index chunks
	for indexKey, currentIndex := range currentIndexes {
		if _, exists := previousIndexes[indexKey]; !exists {
			// New index - create a chunk for it
			err := createIndexChunk(previousChunks, currentIndex, indexKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create index chunk for %s", indexKey)
			}
		}
	}

	// Process index modifications: update existing chunks
	for indexKey, currentIndex := range currentIndexes {
		if previousIndex, exists := previousIndexes[indexKey]; exists {
			// Index exists in both metadata
			// Only update if chunk exists in SDL (user explicitly defined it)
			// If chunk doesn't exist, skip - we don't force-add database objects that user didn't define
			if _, chunkExists := previousChunks.Indexes[indexKey]; chunkExists {
				// Chunk exists - update if needed
				err := updateIndexChunkIfNeeded(previousChunks, currentIndex, previousIndex, indexKey)
				if err != nil {
					return errors.Wrapf(err, "failed to update index chunk for %s", indexKey)
				}
			}
			// If chunk doesn't exist, skip - user didn't define this index in SDL
		}
	}

	// Process index deletions: remove dropped index chunks
	for indexKey := range previousIndexes {
		if _, exists := currentIndexes[indexKey]; !exists {
			// Index was dropped - remove it from chunks
			deleteIndexChunk(previousChunks, indexKey)
		}
	}

	return nil
}

// formatIndexKey creates a consistent key for index identification
func formatIndexKey(schemaName, indexName string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	return schemaName + "." + indexName
}

// createIndexChunk creates a new CREATE INDEX chunk and adds it to the chunks
func createIndexChunk(chunks *schema.SDLChunks, extIndex *extendedIndexMetadata, indexKey string) error {
	if extIndex == nil || extIndex.IndexMetadata == nil || chunks == nil {
		return nil
	}

	// Generate SDL text for the index
	indexSDL := generateCreateIndexSDL(extIndex)
	if indexSDL == "" {
		return errors.New("failed to generate SDL for index")
	}

	// Parse the SDL to get AST node
	parseResults, err := pgparser.ParsePostgreSQL(indexSDL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse generated index SDL: %s", indexSDL)
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	// Extract the CREATE INDEX AST node
	var indexASTNode *parser.IndexstmtContext
	antlr.ParseTreeWalkerDefault.Walk(&indexExtractor{
		result: &indexASTNode,
	}, parseResult.Tree)

	if indexASTNode == nil {
		return errors.New("failed to extract CREATE INDEX AST node")
	}

	// Create and add the chunk
	chunk := &schema.SDLChunk{
		Identifier: indexKey,
		ASTNode:    indexASTNode,
	}

	// Add COMMENT ON INDEX if the index has a comment
	if extIndex.Comment != "" {
		schemaName := extIndex.SchemaName
		if schemaName == "" {
			schemaName = "public"
		}
		if err := syncObjectCommentStatements(chunk, extIndex.Comment, "INDEX", schemaName, extIndex.Name); err != nil {
			return errors.Wrapf(err, "failed to add COMMENT statements for index %s", indexKey)
		}
	}

	if chunks.Indexes == nil {
		chunks.Indexes = make(map[string]*schema.SDLChunk)
	}
	chunks.Indexes[indexKey] = chunk

	return nil
}

// updateIndexChunkIfNeeded updates an existing index chunk if the index definition has changed
func updateIndexChunkIfNeeded(chunks *schema.SDLChunks, currentIndex, previousIndex *extendedIndexMetadata, indexKey string) error {
	if currentIndex == nil || previousIndex == nil || chunks == nil {
		return nil
	}

	// Get the existing chunk
	chunk, exists := chunks.Indexes[indexKey]
	if !exists {
		return errors.Errorf("index chunk not found for key %s", indexKey)
	}

	// Check if the CREATE INDEX definition has changed (excluding comment)
	definitionChanged := !indexDefinitionsEqualExcludingComment(currentIndex.IndexMetadata, previousIndex.IndexMetadata)

	if definitionChanged {
		// Index definition has changed - regenerate the CREATE INDEX chunk
		indexSDL := generateCreateIndexSDL(currentIndex)
		if indexSDL == "" {
			return errors.New("failed to generate SDL for index")
		}

		// Parse the SDL to get AST node
		parseResults, err := pgparser.ParsePostgreSQL(indexSDL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse generated index SDL: %s", indexSDL)
		}

		// Expect single statement
		if len(parseResults) != 1 {
			return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
		}
		parseResult := parseResults[0]

		// Extract the CREATE INDEX AST node
		var indexASTNode *parser.IndexstmtContext
		antlr.ParseTreeWalkerDefault.Walk(&indexExtractor{
			result: &indexASTNode,
		}, parseResult.Tree)

		if indexASTNode == nil {
			return errors.New("failed to extract CREATE INDEX AST node")
		}

		// Update the CREATE INDEX AST node
		chunk.ASTNode = indexASTNode
	}

	// Synchronize COMMENT ON INDEX statements only if comment has changed
	if currentIndex.Comment != previousIndex.Comment {
		schemaName := currentIndex.SchemaName
		if schemaName == "" {
			schemaName = "public"
		}
		if err := syncObjectCommentStatements(chunk, currentIndex.Comment, "INDEX", schemaName, currentIndex.Name); err != nil {
			return errors.Wrapf(err, "failed to sync COMMENT statements for index %s", indexKey)
		}
	}

	return nil
}

// indexDefinitionsEqualExcludingComment compares two index definitions excluding comments
func indexDefinitionsEqualExcludingComment(index1, index2 *storepb.IndexMetadata) bool {
	if index1 == nil || index2 == nil {
		return false
	}

	// Compare basic properties (excluding comment)
	if index1.Name != index2.Name ||
		index1.Unique != index2.Unique ||
		index1.Type != index2.Type ||
		len(index1.Expressions) != len(index2.Expressions) ||
		len(index1.Descending) != len(index2.Descending) {
		return false
	}

	// Compare expressions
	for i, expr := range index1.Expressions {
		if i >= len(index2.Expressions) || expr != index2.Expressions[i] {
			return false
		}
	}

	// Compare descending flags
	for i, desc := range index1.Descending {
		if i >= len(index2.Descending) || desc != index2.Descending[i] {
			return false
		}
	}

	return true
}

// deleteIndexChunk removes an index chunk from the chunks
func deleteIndexChunk(chunks *schema.SDLChunks, indexKey string) {
	if chunks != nil && chunks.Indexes != nil {
		delete(chunks.Indexes, indexKey)
	}
}

// indexDefinitionsEqual compares two index definitions to see if they are equivalent
func indexDefinitionsEqual(index1, index2 *storepb.IndexMetadata) bool {
	// First check everything except comment
	if !indexDefinitionsEqualExcludingComment(index1, index2) {
		return false
	}

	// Then check comment
	if index1.Comment != index2.Comment {
		return false
	}

	return true
}

// generateCreateIndexSDL generates SDL text for a CREATE INDEX statement using writeIndexSDL
func generateCreateIndexSDL(extIndex *extendedIndexMetadata) string {
	if extIndex == nil || extIndex.IndexMetadata == nil {
		return ""
	}

	var buf strings.Builder
	schemaName := extIndex.SchemaName
	if schemaName == "" {
		schemaName = "public"
	}

	// Use the existing writeIndexSDL function from get_database_definition.go
	if err := writeIndexSDL(&buf, schemaName, extIndex.TableName, extIndex.IndexMetadata); err != nil {
		return ""
	}

	return buf.String()
}

// indexExtractor is a walker to extract CREATE INDEX AST nodes
type indexExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.IndexstmtContext
}

func (e *indexExtractor) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}

// applyFunctionChangesToChunks applies minimal changes to CREATE FUNCTION chunks
// This handles creation, modification, and deletion of function statements
func applyFunctionChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get function differences by comparing schema metadata
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build function maps for current and previous schemas
	currentFunctions := make(map[string]*storepb.FunctionMetadata)
	previousFunctions := make(map[string]*storepb.FunctionMetadata)

	// Collect all functions from current schema
	for _, schema := range currentMetadata.Schemas {
		for _, function := range schema.Functions {
			functionKey := formatFunctionKey(schema.Name, function)
			currentFunctions[functionKey] = function
		}
	}

	// Collect all functions from previous schema
	for _, schema := range previousMetadata.Schemas {
		for _, function := range schema.Functions {
			functionKey := formatFunctionKey(schema.Name, function)
			previousFunctions[functionKey] = function
		}
	}

	// Process function additions: create new function chunks
	for functionKey, currentFunction := range currentFunctions {
		if _, exists := previousFunctions[functionKey]; !exists {
			// New function - create a chunk for it
			err := createFunctionChunk(previousChunks, currentFunction, functionKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create function chunk for %s", functionKey)
			}
		}
	}

	// Process function modifications: update existing chunks
	for functionKey, currentFunction := range currentFunctions {
		if previousFunction, exists := previousFunctions[functionKey]; exists {
			// Function exists in both metadata
			// Only update if chunk exists in SDL (user explicitly defined it)
			// If chunk doesn't exist, skip - we don't force-add database objects that user didn't define
			if _, chunkExists := previousChunks.Functions[functionKey]; chunkExists {
				// Chunk exists - update if needed
				err := updateFunctionChunkIfNeeded(previousChunks, currentFunction, previousFunction, functionKey)
				if err != nil {
					return errors.Wrapf(err, "failed to update function chunk for %s", functionKey)
				}
			}
			// If chunk doesn't exist, skip - user didn't define this function in SDL
		}
	}

	// Process function deletions: remove dropped function chunks
	for functionKey := range previousFunctions {
		if _, exists := currentFunctions[functionKey]; !exists {
			// Function was dropped - remove it from chunks
			deleteFunctionChunk(previousChunks, functionKey)
		}
	}

	return nil
}

// formatFunctionKey creates a consistent key for function identification using the full signature
func formatFunctionKey(schemaName string, function *storepb.FunctionMetadata) string {
	if schemaName == "" {
		schemaName = "public"
	}

	// Extract the function signature from the definition
	signature := extractFunctionSignatureFromDefinition(function)
	if signature == "" {
		// Fallback to just the function name if signature extraction fails
		signature = function.Name + "()"
	}

	return schemaName + "." + signature
}

// extractFunctionSignatureFromDefinition extracts the function signature from its definition
func extractFunctionSignatureFromDefinition(function *storepb.FunctionMetadata) string {
	if function == nil || function.Definition == "" {
		return ""
	}

	// Parse the function definition to extract signature
	parseResults, err := pgparser.ParsePostgreSQL(function.Definition)
	if err != nil {
		return ""
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return ""
	}
	parseResult := parseResults[0]

	tree := parseResult.Tree

	// Extract the CREATE FUNCTION node using a walker
	var result *parser.CreatefunctionstmtContext
	extractor := &functionExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return ""
	}

	// Use the unified function signature extraction
	return extractFunctionSignatureFromAST(result)
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

// createFunctionChunk creates a new CREATE FUNCTION chunk and adds it to the chunks
func createFunctionChunk(chunks *schema.SDLChunks, function *storepb.FunctionMetadata, functionKey string) error {
	if function == nil || chunks == nil {
		return nil
	}

	// Generate function SDL using the existing writeFunctionSDL function
	sdl := generateCreateFunctionSDL(function)
	if sdl == "" {
		return errors.Errorf("failed to generate SDL for function %s", functionKey)
	}

	// Parse the SDL to get the AST node
	astNode, err := extractFunctionASTFromSDL(sdl)
	if err != nil {
		return errors.Wrapf(err, "failed to extract AST from generated function SDL for %s", functionKey)
	}

	// Create the chunk
	chunk := &schema.SDLChunk{
		Identifier: functionKey,
		ASTNode:    astNode,
	}

	// Add COMMENT ON FUNCTION/PROCEDURE if the function has a comment
	if function.Comment != "" {
		// Determine if this is a PROCEDURE or FUNCTION by checking the definition
		objectType := "FUNCTION"
		if isDefinitionProcedure(function.Definition) {
			objectType = "PROCEDURE"
		}
		schemaName, functionName := parseIdentifier(functionKey)
		if err := syncObjectCommentStatements(chunk, function.Comment, objectType, schemaName, functionName); err != nil {
			return errors.Wrapf(err, "failed to add COMMENT statements for function %s", functionKey)
		}
	}

	// Ensure Functions map is initialized
	if chunks.Functions == nil {
		chunks.Functions = make(map[string]*schema.SDLChunk)
	}

	chunks.Functions[functionKey] = chunk
	return nil
}

// updateFunctionChunkIfNeeded updates a function chunk if the definition has changed
func updateFunctionChunkIfNeeded(chunks *schema.SDLChunks, currentFunction, previousFunction *storepb.FunctionMetadata, functionKey string) error {
	if currentFunction == nil || previousFunction == nil || chunks == nil {
		return nil
	}

	// Get the existing chunk
	chunk, exists := chunks.Functions[functionKey]
	if !exists {
		return errors.Errorf("function chunk not found for key %s", functionKey)
	}

	// Check if the CREATE FUNCTION definition has changed (excluding comment)
	definitionChanged := !functionDefinitionsEqualExcludingComment(currentFunction, previousFunction)

	if definitionChanged {
		// Function definition has changed - regenerate the CREATE FUNCTION chunk
		sdl := generateCreateFunctionSDL(currentFunction)
		if sdl == "" {
			return errors.Errorf("failed to generate SDL for function %s", functionKey)
		}

		// Parse the SDL to get the AST node
		astNode, err := extractFunctionASTFromSDL(sdl)
		if err != nil {
			return errors.Wrapf(err, "failed to extract AST from generated function SDL for %s", functionKey)
		}

		// Update the CREATE FUNCTION AST node
		chunk.ASTNode = astNode
	}

	// Synchronize COMMENT ON FUNCTION/PROCEDURE statements only if comment has changed
	if currentFunction.Comment != previousFunction.Comment {
		// Determine if this is a PROCEDURE or FUNCTION by checking the definition
		objectType := "FUNCTION"
		if isDefinitionProcedure(currentFunction.Definition) {
			objectType = "PROCEDURE"
		}
		schemaName, functionName := parseIdentifier(functionKey)
		if err := syncObjectCommentStatements(chunk, currentFunction.Comment, objectType, schemaName, functionName); err != nil {
			return errors.Wrapf(err, "failed to sync COMMENT statements for function %s", functionKey)
		}
	}

	return nil
}

// updateFunctionChunk updates an existing function chunk with new definition
// This function synchronizes the CREATE FUNCTION and COMMENT ON FUNCTION statements
// deleteFunctionChunk removes a function chunk from the chunks
func deleteFunctionChunk(chunks *schema.SDLChunks, functionKey string) {
	if chunks != nil && chunks.Functions != nil {
		delete(chunks.Functions, functionKey)
	}
}

// generateCreateFunctionSDL generates SDL for a CREATE FUNCTION statement
func generateCreateFunctionSDL(function *storepb.FunctionMetadata) string {
	if function == nil {
		return ""
	}

	var buf strings.Builder
	// Use the existing writeFunctionSDL function
	if err := writeFunctionSDL(&buf, "", function); err != nil {
		return ""
	}

	return buf.String()
}

// extractFunctionTextFromAST extracts normalized text from function AST node using token stream
func extractFunctionTextFromAST(functionAST *parser.CreatefunctionstmtContext) string {
	if functionAST == nil {
		return ""
	}

	// Get tokens from the parser
	if parser := functionAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := functionAST.GetStart()
			stop := functionAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return functionAST.GetText()
}

// functionDefinitionsEqual compares two function definitions to see if they are equivalent
func functionDefinitionsEqual(function1, function2 *storepb.FunctionMetadata) bool {
	// First check everything except comment
	if !functionDefinitionsEqualExcludingComment(function1, function2) {
		return false
	}

	// Then check comment
	if function1.Comment != function2.Comment {
		return false
	}

	return true
}

// functionDefinitionsEqualExcludingComment compares two function definitions excluding comments
func functionDefinitionsEqualExcludingComment(function1, function2 *storepb.FunctionMetadata) bool {
	if function1 == nil && function2 == nil {
		return true
	}
	if function1 == nil || function2 == nil {
		return false
	}

	// Compare function names
	if function1.Name != function2.Name {
		return false
	}

	// Compare function definitions (most important comparison)
	definition1 := strings.TrimSpace(function1.Definition)
	definition2 := strings.TrimSpace(function2.Definition)

	// Normalize definitions for comparison
	definition1 = strings.TrimSuffix(definition1, ";")
	definition2 = strings.TrimSuffix(definition2, ";")

	return definition1 == definition2
}

// extractFunctionASTFromSDL parses a function SDL and extracts the CREATE FUNCTION AST node
func extractFunctionASTFromSDL(sdl string) (antlr.ParserRuleContext, error) {
	if sdl == "" {
		return nil, errors.New("empty SDL provided")
	}

	// Parse the SDL to get AST
	parseResults, err := pgparser.ParsePostgreSQL(sdl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse function SDL")
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	tree := parseResult.Tree

	// Extract the CREATE FUNCTION node using a walker
	var result *parser.CreatefunctionstmtContext
	extractor := &functionExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return nil, errors.New("no CREATE FUNCTION statement found in SDL")
	}

	return result, nil
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

// applySequenceChangesToChunks applies minimal changes to CREATE SEQUENCE chunks
// This handles creation, modification, and deletion of sequence statements
func applySequenceChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get sequence differences by comparing schema metadata
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build sequence maps for current and previous schemas
	currentSequences := make(map[string]*storepb.SequenceMetadata)
	previousSequences := make(map[string]*storepb.SequenceMetadata)

	// Collect all sequences from current schema
	for _, schema := range currentMetadata.Schemas {
		for _, sequence := range schema.Sequences {
			sequenceKey := formatSequenceKey(schema.Name, sequence.Name)
			currentSequences[sequenceKey] = sequence
		}
	}

	// Collect all sequences from previous schema
	for _, schema := range previousMetadata.Schemas {
		for _, sequence := range schema.Sequences {
			sequenceKey := formatSequenceKey(schema.Name, sequence.Name)
			previousSequences[sequenceKey] = sequence
		}
	}

	// Process sequence additions: create new sequence chunks
	for sequenceKey, currentSequence := range currentSequences {
		if _, exists := previousSequences[sequenceKey]; !exists {
			// Skip sequences owned by tables (e.g., SERIAL columns) - they should be managed by their owner tables
			if currentSequence.OwnerTable != "" {
				continue
			}
			// New sequence - create a chunk for it
			err := createSequenceChunk(previousChunks, currentSequence, sequenceKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create sequence chunk for %s", sequenceKey)
			}
		}
	}

	// Process sequence modifications: update existing chunks
	for sequenceKey, currentSequence := range currentSequences {
		if previousSequence, exists := previousSequences[sequenceKey]; exists {
			// Sequence exists in both metadata
			// Only update if chunk exists in SDL (user explicitly defined it)
			// If chunk doesn't exist, skip - we don't force-add database objects that user didn't define
			if _, chunkExists := previousChunks.Sequences[sequenceKey]; chunkExists {
				// Chunk exists - update if needed
				err := updateSequenceChunkIfNeeded(previousChunks, currentSequence, previousSequence, sequenceKey)
				if err != nil {
					return errors.Wrapf(err, "failed to update sequence chunk for %s", sequenceKey)
				}
			}
			// If chunk doesn't exist, skip - user didn't define this sequence in SDL
		}
	}

	// Process sequence deletions: remove dropped sequence chunks
	for sequenceKey := range previousSequences {
		if _, exists := currentSequences[sequenceKey]; !exists {
			// Sequence was dropped - remove it from chunks
			deleteSequenceChunk(previousChunks, sequenceKey)
		}
	}

	return nil
}

// formatSequenceKey creates a consistent key for sequence identification
func formatSequenceKey(schemaName, sequenceName string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	return schemaName + "." + sequenceName
}

// extractSchemaFromSequenceKey extracts the schema name from a sequence key
func extractSchemaFromSequenceKey(sequenceKey string) string {
	parts := strings.SplitN(sequenceKey, ".", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return "public"
}

// createSequenceChunk creates a new CREATE SEQUENCE chunk and adds it to the chunks
func createSequenceChunk(chunks *schema.SDLChunks, sequence *storepb.SequenceMetadata, sequenceKey string) error {
	if sequence == nil || chunks == nil {
		return nil
	}

	// Generate sequence SDL using the existing writeSequenceSDL function
	schemaName := extractSchemaFromSequenceKey(sequenceKey)
	sdl := generateCreateSequenceSDL(schemaName, sequence)
	if sdl == "" {
		return errors.Errorf("failed to generate SDL for sequence %s", sequenceKey)
	}

	// Parse the SDL to get the AST node
	astNode, err := extractSequenceASTFromSDL(sdl)
	if err != nil {
		return errors.Wrapf(err, "failed to extract AST from generated sequence SDL for %s", sequenceKey)
	}

	// Create the chunk
	chunk := &schema.SDLChunk{
		Identifier: sequenceKey,
		ASTNode:    astNode,
	}

	// Add COMMENT ON SEQUENCE if the sequence has a comment
	if sequence.Comment != "" {
		if err := syncCommentStatements(chunk, sequence, schemaName); err != nil {
			return errors.Wrapf(err, "failed to add COMMENT statements for sequence %s", sequenceKey)
		}
	}

	// Ensure Sequences map is initialized
	if chunks.Sequences == nil {
		chunks.Sequences = make(map[string]*schema.SDLChunk)
	}

	chunks.Sequences[sequenceKey] = chunk
	return nil
}

// updateSequenceChunkIfNeeded updates a sequence chunk if the definition has changed
func updateSequenceChunkIfNeeded(chunks *schema.SDLChunks, currentSequence, previousSequence *storepb.SequenceMetadata, sequenceKey string) error {
	if currentSequence == nil || previousSequence == nil || chunks == nil {
		return nil
	}

	// Get the existing chunk
	chunk, exists := chunks.Sequences[sequenceKey]
	if !exists {
		return errors.Errorf("sequence chunk not found for key %s", sequenceKey)
	}

	// Check if the CREATE SEQUENCE definition has changed (excluding comment and owner)
	definitionChanged := !sequenceDefinitionsEqualExcludingCommentAndOwner(currentSequence, previousSequence)

	if definitionChanged {
		// Sequence definition has changed - regenerate the CREATE SEQUENCE chunk
		schemaName := extractSchemaFromSequenceKey(sequenceKey)
		sdl := generateCreateSequenceSDL(schemaName, currentSequence)
		if sdl == "" {
			return errors.Errorf("failed to generate SDL for sequence %s", sequenceKey)
		}

		// Parse the SDL to get the AST node for CREATE SEQUENCE
		astNode, err := extractSequenceASTFromSDL(sdl)
		if err != nil {
			return errors.Wrapf(err, "failed to extract AST from generated sequence SDL for %s", sequenceKey)
		}

		// Update the CREATE SEQUENCE AST node
		chunk.ASTNode = astNode
	}

	schemaName := extractSchemaFromSequenceKey(sequenceKey)

	// Synchronize ALTER SEQUENCE OWNED BY statements only if owner has changed
	ownerChanged := currentSequence.OwnerTable != previousSequence.OwnerTable ||
		currentSequence.OwnerColumn != previousSequence.OwnerColumn
	if ownerChanged {
		if err := syncAlterSequenceStatements(chunk, currentSequence, schemaName); err != nil {
			return errors.Wrapf(err, "failed to sync ALTER statements for sequence %s", sequenceKey)
		}
	}

	// Synchronize COMMENT ON SEQUENCE statements only if comment has changed
	if currentSequence.Comment != previousSequence.Comment {
		if err := syncCommentStatements(chunk, currentSequence, schemaName); err != nil {
			return errors.Wrapf(err, "failed to sync COMMENT statements for sequence %s", sequenceKey)
		}
	}

	return nil
}

// updateSequenceChunk updates an existing sequence chunk with new definition
// This function synchronizes the CREATE SEQUENCE, ALTER SEQUENCE, and COMMENT ON SEQUENCE statements
// based on the current sequence metadata from the database
// syncAlterSequenceStatements synchronizes ALTER SEQUENCE OWNED BY statements in the chunk
// based on the current OwnerTable and OwnerColumn from sequence metadata
func syncAlterSequenceStatements(chunk *schema.SDLChunk, sequence *storepb.SequenceMetadata, schemaName string) error {
	if chunk == nil || sequence == nil {
		return nil
	}

	// Check if sequence has an owner
	hasOwner := sequence.OwnerTable != "" && sequence.OwnerColumn != ""

	if hasOwner {
		// Generate ALTER SEQUENCE OWNED BY statement
		alterSDL := fmt.Sprintf("ALTER SEQUENCE \"%s\".\"%s\" OWNED BY \"%s\".\"%s\";",
			schemaName, sequence.Name, sequence.OwnerTable, sequence.OwnerColumn)

		// Parse to get AST node
		alterNode, err := extractAlterSequenceASTFromSDL(alterSDL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse ALTER SEQUENCE statement")
		}

		// Replace all existing ALTER statements with the new one
		// (There should only be one ALTER SEQUENCE OWNED BY statement per sequence)
		chunk.AlterStatements = []antlr.ParserRuleContext{alterNode}
	} else {
		// No owner - remove all ALTER SEQUENCE statements
		chunk.AlterStatements = nil
	}

	return nil
}

// syncCommentStatements synchronizes COMMENT ON SEQUENCE statements in the chunk
// based on the current Comment field from sequence metadata
func syncCommentStatements(chunk *schema.SDLChunk, sequence *storepb.SequenceMetadata, schemaName string) error {
	if chunk == nil || sequence == nil {
		return nil
	}

	// Use the generic comment sync function
	return syncObjectCommentStatements(chunk, sequence.Comment, "SEQUENCE", schemaName, sequence.Name)
}

// syncObjectCommentStatements is a generic function to synchronize COMMENT statements for any object type
// objectType should be "SEQUENCE", "TABLE", "VIEW", "FUNCTION", "INDEX", etc.
func syncObjectCommentStatements(chunk *schema.SDLChunk, comment, objectType, schemaName, objectName string) error {
	if chunk == nil {
		return nil
	}

	// Check if object has a comment
	hasComment := comment != ""

	if hasComment {
		// Generate COMMENT ON <objectType> statement
		// Escape single quotes in comment text
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		commentSDL := fmt.Sprintf("COMMENT ON %s \"%s\".\"%s\" IS '%s';",
			objectType, schemaName, objectName, escapedComment)

		// Parse to get AST node
		commentNode, err := extractCommentASTFromSDL(commentSDL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse COMMENT ON %s statement", objectType)
		}

		// Replace all existing COMMENT statements with the new one
		// (There should only be one COMMENT ON statement per object)
		chunk.CommentStatements = []antlr.ParserRuleContext{commentNode}
	} else {
		// No comment - remove all COMMENT statements
		chunk.CommentStatements = nil
	}

	return nil
}

// extractAlterSequenceASTFromSDL parses an ALTER SEQUENCE SDL and extracts the AST node
func extractAlterSequenceASTFromSDL(sdl string) (antlr.ParserRuleContext, error) {
	if sdl == "" {
		return nil, errors.New("empty ALTER SEQUENCE SDL provided")
	}

	parseResults, err := pgparser.ParsePostgreSQL(sdl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ALTER SEQUENCE SDL")
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	tree := parseResult.Tree
	if tree == nil {
		return nil, errors.New("parse result tree is nil")
	}

	// Extract the ALTER SEQUENCE node
	var result *parser.AlterseqstmtContext
	extractor := &alterSequenceExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return nil, errors.New("failed to extract ALTER SEQUENCE AST node from SDL")
	}

	return result, nil
}

// extractCommentASTFromSDL parses a COMMENT ON SDL and extracts the AST node
func extractCommentASTFromSDL(sdl string) (antlr.ParserRuleContext, error) {
	if sdl == "" {
		return nil, errors.New("empty COMMENT SDL provided")
	}

	parseResults, err := pgparser.ParsePostgreSQL(sdl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse COMMENT SDL")
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	tree := parseResult.Tree
	if tree == nil {
		return nil, errors.New("parse result tree is nil")
	}

	// Extract the COMMENT node
	var result *parser.CommentstmtContext
	extractor := &commentExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return nil, errors.New("failed to extract COMMENT AST node from SDL")
	}

	return result, nil
}

// alterSequenceExtractor extracts ALTER SEQUENCE statement from AST
type alterSequenceExtractor struct {
	*parser.BasePostgreSQLParserListener
	result **parser.AlterseqstmtContext
}

func (e *alterSequenceExtractor) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if *e.result == nil {
		*e.result = ctx
	}
}

// commentExtractor extracts COMMENT statement from AST
type commentExtractor struct {
	*parser.BasePostgreSQLParserListener
	result **parser.CommentstmtContext
}

func (e *commentExtractor) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if *e.result == nil {
		*e.result = ctx
	}
}

// deleteSequenceChunk removes a sequence chunk from the chunks
func deleteSequenceChunk(chunks *schema.SDLChunks, sequenceKey string) {
	if chunks != nil && chunks.Sequences != nil {
		delete(chunks.Sequences, sequenceKey)
	}
}

// generateCreateSequenceSDL generates SDL for a CREATE SEQUENCE statement
func generateCreateSequenceSDL(schemaName string, sequence *storepb.SequenceMetadata) string {
	if sequence == nil {
		return ""
	}

	if schemaName == "" {
		schemaName = "public"
	}

	var buf strings.Builder
	// Use the existing writeSequenceSDL function
	if err := writeSequenceSDL(&buf, schemaName, sequence); err != nil {
		return ""
	}

	return buf.String()
}

// sequenceDefinitionsEqual compares two sequence definitions to see if they are equivalent
func sequenceDefinitionsEqual(sequence1, sequence2 *storepb.SequenceMetadata) bool {
	// First check everything except comment and owner
	if !sequenceDefinitionsEqualExcludingCommentAndOwner(sequence1, sequence2) {
		return false
	}

	// Then check owner
	if sequence1.OwnerTable != sequence2.OwnerTable ||
		sequence1.OwnerColumn != sequence2.OwnerColumn {
		return false
	}

	// Finally check comment
	if sequence1.Comment != sequence2.Comment {
		return false
	}

	return true
}

// sequenceDefinitionsEqualExcludingCommentAndOwner compares two sequence definitions excluding comments and owner
func sequenceDefinitionsEqualExcludingCommentAndOwner(sequence1, sequence2 *storepb.SequenceMetadata) bool {
	if sequence1 == nil && sequence2 == nil {
		return true
	}
	if sequence1 == nil || sequence2 == nil {
		return false
	}

	// Compare sequence names
	if sequence1.Name != sequence2.Name {
		return false
	}

	// Compare sequence parameters (excluding OwnerTable, OwnerColumn, and Comment)
	if sequence1.DataType != sequence2.DataType ||
		sequence1.Start != sequence2.Start ||
		sequence1.Increment != sequence2.Increment ||
		sequence1.MaxValue != sequence2.MaxValue ||
		sequence1.MinValue != sequence2.MinValue ||
		sequence1.CacheSize != sequence2.CacheSize ||
		sequence1.Cycle != sequence2.Cycle {
		return false
	}

	return true
}

// extractSequenceASTFromSDL parses a sequence SDL and extracts the CREATE SEQUENCE AST node
func extractSequenceASTFromSDL(sdl string) (antlr.ParserRuleContext, error) {
	if sdl == "" {
		return nil, errors.New("empty SDL provided")
	}

	// Parse the SDL to get AST
	parseResults, err := pgparser.ParsePostgreSQL(sdl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse sequence SDL")
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	tree := parseResult.Tree

	// Extract the CREATE SEQUENCE node using a walker
	var result *parser.CreateseqstmtContext
	extractor := &sequenceExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return nil, errors.New("no CREATE SEQUENCE statement found in SDL")
	}

	return result, nil
}

// extractSequenceTextFromAST extracts normalized text from sequence AST node using token stream
func extractSequenceTextFromAST(sequenceAST *parser.CreateseqstmtContext) string {
	if sequenceAST == nil {
		return ""
	}

	// Get tokens from the parser
	if parser := sequenceAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := sequenceAST.GetStart()
			stop := sequenceAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return sequenceAST.GetText()
}

// sequenceExtractor is a walker to extract CREATE SEQUENCE AST nodes
type sequenceExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.CreateseqstmtContext
}

func (e *sequenceExtractor) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}

// processCommentChanges processes comment changes for all database objects
// It must be called after all object changes have been processed to determine
// which objects were created or dropped (those should not generate comment diffs)

// applyViewChangesToChunks applies minimal changes to CREATE VIEW chunks
// This handles creation, modification, and deletion of view statements

// applyMaterializedViewChangesToChunks applies minimal changes to materialized view chunks based on schema metadata
func applyMaterializedViewChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get materialized view differences by comparing schema metadata
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build materialized view maps for current and previous schemas
	currentMaterializedViews := make(map[string]*storepb.MaterializedViewMetadata)
	previousMaterializedViews := make(map[string]*storepb.MaterializedViewMetadata)

	// Collect all materialized views from current schema
	for _, schema := range currentMetadata.Schemas {
		for _, mv := range schema.MaterializedViews {
			mvKey := formatViewKey(schema.Name, mv.Name)
			currentMaterializedViews[mvKey] = mv
		}
	}

	// Collect all materialized views from previous schema
	for _, schema := range previousMetadata.Schemas {
		for _, mv := range schema.MaterializedViews {
			mvKey := formatViewKey(schema.Name, mv.Name)
			previousMaterializedViews[mvKey] = mv
		}
	}

	// Process materialized view additions: create new materialized view chunks
	for mvKey, currentMV := range currentMaterializedViews {
		if _, exists := previousMaterializedViews[mvKey]; !exists {
			// New materialized view - create a chunk for it
			err := createMaterializedViewChunk(previousChunks, currentMV, mvKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create materialized view chunk for %s", mvKey)
			}
		}
	}

	// Process materialized view modifications: update existing chunks
	for mvKey, currentMV := range currentMaterializedViews {
		if previousMV, exists := previousMaterializedViews[mvKey]; exists {
			// Materialized view exists in both metadata
			// Only update if chunk exists in SDL (user explicitly defined it)
			if _, chunkExists := previousChunks.MaterializedViews[mvKey]; chunkExists {
				// Chunk exists - update if needed
				err := updateMaterializedViewChunkIfNeeded(previousChunks, currentMV, previousMV, mvKey)
				if err != nil {
					return errors.Wrapf(err, "failed to update materialized view chunk for %s", mvKey)
				}
			}
			// If chunk doesn't exist, skip - user didn't define this materialized view in SDL
		}
	}

	// Process materialized view deletions: remove dropped materialized view chunks
	for mvKey := range previousMaterializedViews {
		if _, exists := currentMaterializedViews[mvKey]; !exists {
			// Materialized view was dropped - remove it from chunks
			deleteMaterializedViewChunk(previousChunks, mvKey)
		}
	}

	return nil
}

// createMaterializedViewChunk creates a new CREATE MATERIALIZED VIEW chunk and adds it to the chunks
func createMaterializedViewChunk(chunks *schema.SDLChunks, mv *storepb.MaterializedViewMetadata, mvKey string) error {
	if mv == nil || chunks == nil {
		return nil
	}

	// Generate SDL text for the materialized view
	schemaName, _ := parseIdentifier(mvKey)
	mvSDL := generateCreateMaterializedViewSDL(schemaName, mv)
	if mvSDL == "" {
		return errors.New("failed to generate SDL for materialized view")
	}

	// Parse the SDL to get AST node
	parseResults, err := pgparser.ParsePostgreSQL(mvSDL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse generated materialized view SDL: %s", mvSDL)
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	// Extract the CREATE MATERIALIZED VIEW AST node
	var mvASTNode *parser.CreatematviewstmtContext
	antlr.ParseTreeWalkerDefault.Walk(&materializedViewExtractor{
		result: &mvASTNode,
	}, parseResult.Tree)

	if mvASTNode == nil {
		return errors.New("failed to extract CREATE MATERIALIZED VIEW AST node")
	}

	// Create and add the chunk
	chunk := &schema.SDLChunk{
		Identifier: mvKey,
		ASTNode:    mvASTNode,
	}

	// Add comment if the materialized view has one
	if mv.Comment != "" {
		commentSQL := generateCommentOnMaterializedViewSQL(schemaName, mv.Name, mv.Comment)
		commentParseResult, err := pgparser.ParsePostgreSQL(commentSQL)
		if err == nil && len(commentParseResult) > 0 && commentParseResult[0].Tree != nil {
			// Extract COMMENT ON MATERIALIZED VIEW AST node
			var commentASTNode *parser.CommentstmtContext
			antlr.ParseTreeWalkerDefault.Walk(&commentExtractor{
				result: &commentASTNode,
			}, commentParseResult[0].Tree)

			if commentASTNode != nil {
				chunk.CommentStatements = []antlr.ParserRuleContext{commentASTNode}
			}
		}
	}

	if chunks.MaterializedViews == nil {
		chunks.MaterializedViews = make(map[string]*schema.SDLChunk)
	}
	chunks.MaterializedViews[mvKey] = chunk

	return nil
}

// updateMaterializedViewChunkIfNeeded updates an existing materialized view chunk if the definition has changed
func updateMaterializedViewChunkIfNeeded(chunks *schema.SDLChunks, currentMV, previousMV *storepb.MaterializedViewMetadata, mvKey string) error {
	if currentMV == nil || previousMV == nil || chunks == nil {
		return nil
	}

	// Get the existing chunk
	chunk, exists := chunks.MaterializedViews[mvKey]
	if !exists {
		return errors.Errorf("materialized view chunk not found for key %s", mvKey)
	}

	// Check if the materialized view definition has changed
	if currentMV.Definition != previousMV.Definition {
		// Materialized view definition changed - regenerate the chunk
		schemaName, _ := parseIdentifier(mvKey)
		mvSDL := generateCreateMaterializedViewSDL(schemaName, currentMV)
		if mvSDL == "" {
			return errors.New("failed to generate SDL for materialized view")
		}

		// Parse the new SDL to get a fresh AST node
		parseResults, err := pgparser.ParsePostgreSQL(mvSDL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse generated materialized view SDL: %s", mvSDL)
		}

		// Expect single statement
		if len(parseResults) != 1 {
			return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
		}
		parseResult := parseResults[0]

		// Extract the CREATE MATERIALIZED VIEW AST node
		var mvASTNode *parser.CreatematviewstmtContext
		antlr.ParseTreeWalkerDefault.Walk(&materializedViewExtractor{
			result: &mvASTNode,
		}, parseResult.Tree)

		if mvASTNode == nil {
			return errors.New("failed to extract CREATE MATERIALIZED VIEW AST node")
		}

		// Update the chunk's AST node
		chunk.ASTNode = mvASTNode
	}

	// Handle comment changes independently of definition changes
	if currentMV.Comment != previousMV.Comment {
		schemaName, _ := parseIdentifier(mvKey)
		if currentMV.Comment != "" {
			// New or updated comment
			commentSQL := generateCommentOnMaterializedViewSQL(schemaName, currentMV.Name, currentMV.Comment)
			commentParseResult, err := pgparser.ParsePostgreSQL(commentSQL)
			if err == nil && len(commentParseResult) > 0 && commentParseResult[0].Tree != nil {
				// Extract COMMENT ON MATERIALIZED VIEW AST node
				var commentASTNode *parser.CommentstmtContext
				antlr.ParseTreeWalkerDefault.Walk(&commentExtractor{
					result: &commentASTNode,
				}, commentParseResult[0].Tree)

				if commentASTNode != nil {
					chunk.CommentStatements = []antlr.ParserRuleContext{commentASTNode}
				}
			}
		} else {
			// Comment was removed
			chunk.CommentStatements = nil
		}
	}

	return nil
}

// deleteMaterializedViewChunk removes a materialized view chunk from the chunks
func deleteMaterializedViewChunk(chunks *schema.SDLChunks, mvKey string) {
	if chunks != nil && chunks.MaterializedViews != nil {
		delete(chunks.MaterializedViews, mvKey)
	}
}

// generateCreateMaterializedViewSDL generates the SDL text for a CREATE MATERIALIZED VIEW statement
func generateCreateMaterializedViewSDL(schemaName string, mv *storepb.MaterializedViewMetadata) string {
	if mv == nil {
		return ""
	}

	var buf strings.Builder
	if err := writeMaterializedViewSDL(&buf, schemaName, mv); err != nil {
		return ""
	}

	return buf.String()
}

// generateCommentOnMaterializedViewSQL generates a COMMENT ON MATERIALIZED VIEW statement
func generateCommentOnMaterializedViewSQL(schemaName, mvName, comment string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	// Escape single quotes in comment
	escapedComment := strings.ReplaceAll(comment, "'", "''")
	return fmt.Sprintf("COMMENT ON MATERIALIZED VIEW \"%s\".\"%s\" IS '%s';", schemaName, mvName, escapedComment)
}

// applyEnumTypeChangesToChunks applies minimal changes to enum type chunks based on schema metadata
func applyEnumTypeChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get enum type differences by comparing schema metadata
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build enum type maps for current and previous schemas
	currentEnumTypes := make(map[string]*storepb.EnumTypeMetadata)
	previousEnumTypes := make(map[string]*storepb.EnumTypeMetadata)

	// Collect all enum types from current schema
	for _, schema := range currentMetadata.Schemas {
		for _, enumType := range schema.EnumTypes {
			enumKey := schema.Name + "." + enumType.Name
			currentEnumTypes[enumKey] = enumType
		}
	}

	// Collect all enum types from previous schema
	for _, schema := range previousMetadata.Schemas {
		for _, enumType := range schema.EnumTypes {
			enumKey := schema.Name + "." + enumType.Name
			previousEnumTypes[enumKey] = enumType
		}
	}

	// Process enum type additions: create new enum type chunks
	for enumKey, currentEnum := range currentEnumTypes {
		if _, exists := previousEnumTypes[enumKey]; !exists {
			// New enum type - create a chunk for it
			err := createEnumTypeChunk(previousChunks, currentEnum, enumKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create enum type chunk for %s", enumKey)
			}
		}
	}

	// Process enum type modifications: update existing chunks
	for enumKey, currentEnum := range currentEnumTypes {
		if previousEnum, exists := previousEnumTypes[enumKey]; exists {
			// Enum type exists in both metadata
			// Only update if chunk exists in SDL (user explicitly defined it)
			if _, chunkExists := previousChunks.EnumTypes[enumKey]; chunkExists {
				// Chunk exists - update if needed
				err := updateEnumTypeChunkIfNeeded(previousChunks, currentEnum, previousEnum, enumKey)
				if err != nil {
					return errors.Wrapf(err, "failed to update enum type chunk for %s", enumKey)
				}
			}
			// If chunk doesn't exist, skip - user didn't define this enum type in SDL
		}
	}

	// Process enum type deletions: remove dropped enum type chunks
	for enumKey := range previousEnumTypes {
		if _, exists := currentEnumTypes[enumKey]; !exists {
			// Enum type was dropped - remove it from chunks
			deleteEnumTypeChunk(previousChunks, enumKey)
		}
	}

	return nil
}

// createEnumTypeChunk creates a new CREATE TYPE AS ENUM chunk and adds it to the chunks
func createEnumTypeChunk(chunks *schema.SDLChunks, enumType *storepb.EnumTypeMetadata, enumKey string) error {
	if enumType == nil || chunks == nil {
		return nil
	}

	// Generate SDL text for the enum type
	schemaName, _ := parseIdentifier(enumKey)
	enumSDL := generateCreateEnumTypeSDL(schemaName, enumType)
	if enumSDL == "" {
		return errors.New("failed to generate SDL for enum type")
	}

	// Parse the SDL to get AST node
	parseResults, err := pgparser.ParsePostgreSQL(enumSDL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse generated enum type SDL: %s", enumSDL)
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	// Extract the CREATE TYPE AS ENUM AST node
	var enumASTNode *parser.DefinestmtContext
	antlr.ParseTreeWalkerDefault.Walk(&enumTypeExtractor{
		result: &enumASTNode,
	}, parseResult.Tree)

	if enumASTNode == nil {
		return errors.New("failed to extract CREATE TYPE AS ENUM AST node")
	}

	// Create and add the chunk
	chunk := &schema.SDLChunk{
		Identifier: enumKey,
		ASTNode:    enumASTNode,
	}

	// Handle comment if present
	if len(enumType.Comment) > 0 {
		commentSQL := generateCommentOnTypeSQL(schemaName, enumType.Name, enumType.Comment)
		commentParseResult, err := pgparser.ParsePostgreSQL(commentSQL)
		if err == nil {
			var commentNode *parser.CommentstmtContext
			antlr.ParseTreeWalkerDefault.Walk(&commentExtractor{
				result: &commentNode,
			}, commentParseResult[0].Tree)
			if commentNode != nil {
				chunk.CommentStatements = []antlr.ParserRuleContext{commentNode}
			}
		}
	}

	chunks.EnumTypes[enumKey] = chunk
	return nil
}

// updateEnumTypeChunkIfNeeded updates an enum type chunk if the definition or comment changed
func updateEnumTypeChunkIfNeeded(chunks *schema.SDLChunks, currentEnum, previousEnum *storepb.EnumTypeMetadata, enumKey string) error {
	if currentEnum == nil || previousEnum == nil {
		return nil
	}

	chunk, exists := chunks.EnumTypes[enumKey]
	if !exists {
		return errors.Errorf("enum type chunk not found for key %s", enumKey)
	}

	// Check if the enum definition has changed (values changed)
	definitionChanged := !enumTypesEqual(currentEnum, previousEnum)

	if definitionChanged {
		// Enum definition has changed - regenerate the CREATE TYPE chunk
		schemaName, _ := parseIdentifier(enumKey)
		enumSDL := generateCreateEnumTypeSDL(schemaName, currentEnum)
		if enumSDL == "" {
			return errors.New("failed to generate SDL for enum type")
		}

		// Parse the SDL to get AST node
		parseResults, err := pgparser.ParsePostgreSQL(enumSDL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse generated enum type SDL: %s", enumSDL)
		}

		// Expect single statement
		if len(parseResults) != 1 {
			return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
		}
		parseResult := parseResults[0]

		// Extract the CREATE TYPE AS ENUM AST node
		var enumASTNode *parser.DefinestmtContext
		antlr.ParseTreeWalkerDefault.Walk(&enumTypeExtractor{
			result: &enumASTNode,
		}, parseResult.Tree)

		if enumASTNode == nil {
			return errors.New("failed to extract CREATE TYPE AS ENUM AST node")
		}

		// Update the CREATE TYPE AST node
		chunk.ASTNode = enumASTNode
	}

	// Synchronize COMMENT ON TYPE statements only if comment has changed
	if currentEnum.Comment != previousEnum.Comment {
		schemaName, _ := parseIdentifier(enumKey)
		if err := syncObjectCommentStatements(chunk, currentEnum.Comment, "TYPE", schemaName, currentEnum.Name); err != nil {
			return errors.Wrapf(err, "failed to sync COMMENT statements for enum type %s", enumKey)
		}
	}

	return nil
}

// deleteEnumTypeChunk removes an enum type chunk from the chunks
func deleteEnumTypeChunk(chunks *schema.SDLChunks, enumKey string) {
	if chunks != nil && chunks.EnumTypes != nil {
		delete(chunks.EnumTypes, enumKey)
	}
}

// enumTypesEqual compares two enum type definitions excluding comments
func enumTypesEqual(enum1, enum2 *storepb.EnumTypeMetadata) bool {
	if enum1 == nil || enum2 == nil {
		return false
	}

	// Compare name
	if enum1.Name != enum2.Name {
		return false
	}

	// Compare values
	if len(enum1.Values) != len(enum2.Values) {
		return false
	}
	for i, v1 := range enum1.Values {
		if v1 != enum2.Values[i] {
			return false
		}
	}

	return true
}

// generateCreateEnumTypeSDL generates the SDL text for a CREATE TYPE AS ENUM statement
func generateCreateEnumTypeSDL(schemaName string, enumType *storepb.EnumTypeMetadata) string {
	if enumType == nil {
		return ""
	}

	var buf strings.Builder
	if err := writeEnum(&buf, schemaName, enumType); err != nil {
		return ""
	}
	buf.WriteString(";")

	return buf.String()
}

// generateCommentOnTypeSQL generates a COMMENT ON TYPE statement
func generateCommentOnTypeSQL(schemaName, typeName, comment string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	// Escape single quotes in comment
	escapedComment := strings.ReplaceAll(comment, "'", "''")
	return fmt.Sprintf("COMMENT ON TYPE \"%s\".\"%s\" IS '%s';", schemaName, typeName, escapedComment)
}

// enumTypeExtractor is a walker to extract CREATE TYPE AS ENUM AST nodes
type enumTypeExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.DefinestmtContext
}

func (e *enumTypeExtractor) EnterDefinestmt(ctx *parser.DefinestmtContext) {
	// Only extract CREATE TYPE AS ENUM statements
	if ctx.CREATE() != nil && ctx.TYPE_P() != nil && ctx.AS() != nil && ctx.ENUM_P() != nil {
		if e.result != nil && *e.result == nil {
			*e.result = ctx
		}
	}
}

// materializedViewExtractor is a walker to extract CREATE MATERIALIZED VIEW AST nodes
type materializedViewExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.CreatematviewstmtContext
}

func (e *materializedViewExtractor) EnterCreatematviewstmt(ctx *parser.CreatematviewstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}

// applyColumnCommentChanges applies minimal changes to column comments based on schema metadata
// This function only updates COMMENT ON COLUMN statements without modifying CREATE TABLE statements
func applyColumnCommentChanges(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get table metadata from schemas
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build table maps for current and previous schemas
	currentTables := make(map[string]*storepb.TableMetadata)
	previousTables := make(map[string]*storepb.TableMetadata)

	// Collect all tables from current schema
	for _, schema := range currentMetadata.Schemas {
		schemaName := schema.Name
		if schemaName == "" {
			schemaName = "public"
		}
		for _, table := range schema.Tables {
			tableKey := schemaName + "." + table.Name
			currentTables[tableKey] = table
		}
	}

	// Collect all tables from previous schema
	for _, schema := range previousMetadata.Schemas {
		schemaName := schema.Name
		if schemaName == "" {
			schemaName = "public"
		}
		for _, table := range schema.Tables {
			tableKey := schemaName + "." + table.Name
			previousTables[tableKey] = table
		}
	}

	// Initialize ColumnComments map if needed
	if previousChunks.ColumnComments == nil {
		previousChunks.ColumnComments = make(map[string]map[string]antlr.ParserRuleContext)
	}

	// Process tables that exist in both schemas
	for tableKey, currentTable := range currentTables {
		previousTable, exists := previousTables[tableKey]
		if !exists {
			// Table is new, skip (will be handled by table creation)
			continue
		}

		// Build column maps
		currentColumns := make(map[string]*storepb.ColumnMetadata)
		previousColumns := make(map[string]*storepb.ColumnMetadata)

		for _, col := range currentTable.Columns {
			currentColumns[col.Name] = col
		}
		for _, col := range previousTable.Columns {
			previousColumns[col.Name] = col
		}

		// Process columns that exist in both versions
		for columnName, currentColumn := range currentColumns {
			previousColumn, colExists := previousColumns[columnName]
			if !colExists {
				// Column is new, skip (will be handled by column addition)
				continue
			}

			// Check if comment has changed
			if currentColumn.Comment != previousColumn.Comment {
				err := syncColumnComment(previousChunks, tableKey, columnName, currentColumn.Comment)
				if err != nil {
					return errors.Wrapf(err, "failed to sync comment for column %s.%s", tableKey, columnName)
				}
			}
		}

		// Process columns that were dropped - remove their comments
		for columnName := range previousColumns {
			if _, exists := currentColumns[columnName]; !exists {
				// Column was dropped - remove its comment
				if previousChunks.ColumnComments[tableKey] != nil {
					delete(previousChunks.ColumnComments[tableKey], columnName)
					if len(previousChunks.ColumnComments[tableKey]) == 0 {
						delete(previousChunks.ColumnComments, tableKey)
					}
				}
			}
		}
	}

	return nil
}

// syncColumnComment synchronizes a single column comment in the chunks
func syncColumnComment(chunks *schema.SDLChunks, tableKey, columnName, comment string) error {
	if chunks == nil {
		return nil
	}

	// Parse table key to get schema and table names
	schemaName, tableName := parseIdentifier(tableKey)

	// If comment is empty, remove the comment statement
	if comment == "" {
		if chunks.ColumnComments[tableKey] != nil {
			delete(chunks.ColumnComments[tableKey], columnName)
			if len(chunks.ColumnComments[tableKey]) == 0 {
				delete(chunks.ColumnComments, tableKey)
			}
		}
		return nil
	}

	// Generate COMMENT ON COLUMN statement
	escapedComment := strings.ReplaceAll(comment, "'", "''")
	commentSDL := fmt.Sprintf("COMMENT ON COLUMN \"%s\".\"%s\".\"%s\" IS '%s';",
		schemaName, tableName, columnName, escapedComment)

	// Parse the SDL to get AST node
	commentNode, err := extractCommentASTFromSDL(commentSDL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse column comment SDL: %s", commentSDL)
	}

	// Initialize maps if needed
	if chunks.ColumnComments == nil {
		chunks.ColumnComments = make(map[string]map[string]antlr.ParserRuleContext)
	}
	if chunks.ColumnComments[tableKey] == nil {
		chunks.ColumnComments[tableKey] = make(map[string]antlr.ParserRuleContext)
	}

	// Update the comment node
	chunks.ColumnComments[tableKey][columnName] = commentNode

	return nil
}

// processEnumTypeChanges analyzes enum type changes between current and previous chunks
// Enum types use DROP + CREATE pattern for modifications (PostgreSQL doesn't support ALTER TYPE ... RENAME VALUE)

// processExtensionChanges processes extension changes between current and previous chunks

// processEventTriggerChanges processes event trigger changes between current and previous chunks

// processSchemaChanges processes explicit CREATE SCHEMA statements in the SDL

// applyExtensionChangesToChunks applies minimal changes to extension chunks based on schema differences
// Extensions are database-level objects (not schema-scoped)
func applyExtensionChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get extension differences by comparing schema metadata
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build extension maps for current and previous schemas
	// Extensions are database-level, stored directly in DatabaseSchemaMetadata
	currentExtensions := make(map[string]*storepb.ExtensionMetadata)
	previousExtensions := make(map[string]*storepb.ExtensionMetadata)

	// Collect all extensions from current schema
	for _, extension := range currentMetadata.Extensions {
		// Extension key is just the name (database-level, no schema prefix)
		currentExtensions[extension.Name] = extension
	}

	// Collect all extensions from previous schema
	for _, extension := range previousMetadata.Extensions {
		previousExtensions[extension.Name] = extension
	}

	// Process extension additions: create new extension chunks
	for extensionName, currentExtension := range currentExtensions {
		if _, exists := previousExtensions[extensionName]; !exists {
			// New extension - create a chunk for it
			err := createExtensionChunk(previousChunks, currentExtension)
			if err != nil {
				return errors.Wrapf(err, "failed to create extension chunk for %s", extensionName)
			}
		}
	}

	// Process extension modifications: update existing chunks
	for extensionName, currentExtension := range currentExtensions {
		if previousExtension, exists := previousExtensions[extensionName]; exists {
			// Extension exists in both metadata
			// Only update if chunk exists in SDL (user explicitly defined it)
			if _, chunkExists := previousChunks.Extensions[extensionName]; chunkExists {
				// Chunk exists - update if needed
				err := updateExtensionChunkIfNeeded(previousChunks, currentExtension, previousExtension)
				if err != nil {
					return errors.Wrapf(err, "failed to update extension chunk for %s", extensionName)
				}
			}
			// If chunk doesn't exist, skip - user didn't define this extension in SDL
		}
	}

	// Process extension deletions: remove dropped extension chunks
	for extensionName := range previousExtensions {
		if _, exists := currentExtensions[extensionName]; !exists {
			// Extension was dropped - remove it from chunks
			deleteExtensionChunk(previousChunks, extensionName)
		}
	}

	return nil
}

// createExtensionChunk creates a new CREATE EXTENSION chunk and adds it to the chunks
func createExtensionChunk(chunks *schema.SDLChunks, extension *storepb.ExtensionMetadata) error {
	if extension == nil || chunks == nil {
		return nil
	}

	// Generate SDL text for the extension
	extensionSDL := generateCreateExtensionSQL(extension)
	if extensionSDL == "" {
		return errors.New("failed to generate SDL for extension")
	}

	// Parse the SDL to get AST node
	parseResults, err := pgparser.ParsePostgreSQL(extensionSDL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse generated extension SDL: %s", extensionSDL)
	}

	// Expect single statement
	if len(parseResults) != 1 {
		return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	// Extract the CREATE EXTENSION AST node
	var extensionASTNode *parser.CreateextensionstmtContext
	antlr.ParseTreeWalkerDefault.Walk(&extensionExtractor{
		result: &extensionASTNode,
	}, parseResult.Tree)

	if extensionASTNode == nil {
		return errors.New("failed to extract CREATE EXTENSION AST node")
	}

	// Create and add the chunk
	chunk := &schema.SDLChunk{
		Identifier: extension.Name, // Extensions are database-level, no schema prefix
		ASTNode:    extensionASTNode,
	}

	// Handle comment if present (description field is the comment)
	if len(extension.Description) > 0 {
		commentSQL := generateCommentOnExtensionSQL(extension.Name, extension.Description)
		commentParseResult, err := pgparser.ParsePostgreSQL(commentSQL)
		if err == nil {
			var commentNode *parser.CommentstmtContext
			antlr.ParseTreeWalkerDefault.Walk(&commentExtractor{
				result: &commentNode,
			}, commentParseResult[0].Tree)
			if commentNode != nil {
				chunk.CommentStatements = []antlr.ParserRuleContext{commentNode}
			}
		}
	}

	chunks.Extensions[extension.Name] = chunk
	return nil
}

// updateExtensionChunkIfNeeded updates an extension chunk if the definition or comment changed
func updateExtensionChunkIfNeeded(chunks *schema.SDLChunks, currentExtension, previousExtension *storepb.ExtensionMetadata) error {
	if currentExtension == nil || previousExtension == nil {
		return nil
	}

	chunk, exists := chunks.Extensions[currentExtension.Name]
	if !exists {
		return errors.Errorf("extension chunk not found for %s", currentExtension.Name)
	}

	// Check if the extension definition has changed (schema, version, or description)
	definitionChanged := !extensionsEqual(currentExtension, previousExtension)

	if definitionChanged {
		// Extension definition has changed - regenerate the CREATE EXTENSION chunk
		extensionSDL := generateCreateExtensionSQL(currentExtension)
		if extensionSDL == "" {
			return errors.New("failed to generate SDL for extension")
		}

		// Parse the SDL to get AST node
		parseResults, err := pgparser.ParsePostgreSQL(extensionSDL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse generated extension SDL: %s", extensionSDL)
		}

		// Expect single statement
		if len(parseResults) != 1 {
			return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
		}
		parseResult := parseResults[0]

		// Extract the CREATE EXTENSION AST node
		var extensionASTNode *parser.CreateextensionstmtContext
		antlr.ParseTreeWalkerDefault.Walk(&extensionExtractor{
			result: &extensionASTNode,
		}, parseResult.Tree)

		if extensionASTNode == nil {
			return errors.New("failed to extract CREATE EXTENSION AST node")
		}

		// Update the AST node in the chunk
		chunk.ASTNode = extensionASTNode
	}

	// Handle comment changes (description field)
	commentChanged := currentExtension.Description != previousExtension.Description

	if commentChanged {
		// Remove old comment statements
		chunk.CommentStatements = nil

		// Add new comment if present
		if len(currentExtension.Description) > 0 {
			commentSQL := generateCommentOnExtensionSQL(currentExtension.Name, currentExtension.Description)
			commentParseResult, err := pgparser.ParsePostgreSQL(commentSQL)
			if err == nil {
				var commentNode *parser.CommentstmtContext
				antlr.ParseTreeWalkerDefault.Walk(&commentExtractor{
					result: &commentNode,
				}, commentParseResult[0].Tree)
				if commentNode != nil {
					chunk.CommentStatements = []antlr.ParserRuleContext{commentNode}
				}
			}
		}
	}

	return nil
}

// deleteExtensionChunk removes an extension chunk from the chunks map
func deleteExtensionChunk(chunks *schema.SDLChunks, extensionName string) {
	if chunks == nil {
		return
	}
	delete(chunks.Extensions, extensionName)
}

// generateCreateExtensionSQL generates a CREATE EXTENSION IF NOT EXISTS statement
func generateCreateExtensionSQL(extension *storepb.ExtensionMetadata) string {
	if extension == nil {
		return ""
	}

	var buf strings.Builder
	buf.WriteString(`CREATE EXTENSION IF NOT EXISTS "`)
	buf.WriteString(extension.Name)
	buf.WriteString(`"`)

	// Add WITH SCHEMA clause if schema is specified
	if extension.Schema != "" {
		buf.WriteString(` WITH SCHEMA "`)
		buf.WriteString(extension.Schema)
		buf.WriteString(`"`)
	}

	// Add VERSION clause if version is specified
	if extension.Version != "" {
		buf.WriteString(` VERSION '`)
		buf.WriteString(extension.Version)
		buf.WriteString(`'`)
	}

	buf.WriteString(`;`)

	return buf.String()
}

// generateCommentOnExtensionSQL generates a COMMENT ON EXTENSION statement
func generateCommentOnExtensionSQL(extensionName, comment string) string {
	// Escape single quotes in comment
	escapedComment := strings.ReplaceAll(comment, "'", "''")
	return fmt.Sprintf("COMMENT ON EXTENSION \"%s\" IS '%s';", extensionName, escapedComment)
}

// extensionExtractor is a walker to extract CREATE EXTENSION AST nodes
type extensionExtractor struct {
	parser.BasePostgreSQLParserListener
	result **parser.CreateextensionstmtContext
}

func (e *extensionExtractor) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if e.result != nil {
		*e.result = ctx
	}
}

// extensionsEqual compares two extension metadata for equality (excluding comments)
func extensionsEqual(a, b *storepb.ExtensionMetadata) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.Name == b.Name &&
		a.Schema == b.Schema &&
		a.Version == b.Version &&
		a.Description == b.Description
}

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

// applyTriggerChangesToChunks applies minimal changes to trigger chunks based on schema metadata
func applyTriggerChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	if currentSchema == nil || previousSchema == nil || previousChunks == nil {
		return nil
	}

	// Get trigger differences by comparing schema metadata
	currentMetadata := currentSchema.GetProto()
	previousMetadata := previousSchema.GetProto()
	if currentMetadata == nil || previousMetadata == nil {
		return nil
	}

	// Build trigger maps for current and previous schemas
	// Key format: schema.table.trigger_name (table-scoped)
	currentTriggers := make(map[string]*triggerWithContext)
	previousTriggers := make(map[string]*triggerWithContext)

	// Collect all triggers from current schema
	for _, schemaObj := range currentMetadata.Schemas {
		schemaName := schemaObj.Name
		if schemaName == "" {
			schemaName = "public"
		}
		for _, table := range schemaObj.Tables {
			for _, trigger := range table.Triggers {
				// Use table-scoped identifier: schema.table.trigger_name
				triggerKey := schemaName + "." + table.Name + "." + trigger.Name
				currentTriggers[triggerKey] = &triggerWithContext{
					trigger:    trigger,
					schemaName: schemaName,
					tableName:  table.Name,
				}
			}
		}
	}

	// Collect all triggers from previous schema
	for _, schemaObj := range previousMetadata.Schemas {
		schemaName := schemaObj.Name
		if schemaName == "" {
			schemaName = "public"
		}
		for _, table := range schemaObj.Tables {
			for _, trigger := range table.Triggers {
				// Use table-scoped identifier: schema.table.trigger_name
				triggerKey := schemaName + "." + table.Name + "." + trigger.Name
				previousTriggers[triggerKey] = &triggerWithContext{
					trigger:    trigger,
					schemaName: schemaName,
					tableName:  table.Name,
				}
			}
		}
	}

	// Process trigger additions: create new trigger chunks
	for triggerKey, currentTrigger := range currentTriggers {
		if _, exists := previousTriggers[triggerKey]; !exists {
			// New trigger - create a chunk for it
			err := createTriggerChunk(previousChunks, currentTrigger, triggerKey)
			if err != nil {
				return errors.Wrapf(err, "failed to create trigger chunk for %s", triggerKey)
			}
		}
	}

	// Process trigger modifications: update existing chunks
	for triggerKey, currentTrigger := range currentTriggers {
		if previousTrigger, exists := previousTriggers[triggerKey]; exists {
			// Trigger exists in both metadata
			// Only update if chunk exists in SDL (user explicitly defined it)
			// If chunk doesn't exist, skip - we don't force-add database objects that user didn't define
			if _, chunkExists := previousChunks.Triggers[triggerKey]; chunkExists {
				// Chunk exists - update if needed
				err := updateTriggerChunkIfNeeded(previousChunks, currentTrigger, previousTrigger, triggerKey)
				if err != nil {
					return errors.Wrapf(err, "failed to update trigger chunk for %s", triggerKey)
				}
			}
			// If chunk doesn't exist, skip - user didn't define this trigger in SDL
		}
	}

	// Process trigger deletions: remove dropped trigger chunks
	for triggerKey := range previousTriggers {
		if _, exists := currentTriggers[triggerKey]; !exists {
			// Trigger was dropped - remove it from chunks
			deleteTriggerChunk(previousChunks, triggerKey)
		}
	}

	return nil
}

// triggerWithContext holds trigger metadata with its schema and table context
type triggerWithContext struct {
	trigger    *storepb.TriggerMetadata
	schemaName string
	tableName  string
}

// createTriggerChunk creates a new CREATE TRIGGER chunk and adds it to the chunks
func createTriggerChunk(chunks *schema.SDLChunks, triggerCtx *triggerWithContext, triggerKey string) error {
	if triggerCtx == nil || triggerCtx.trigger == nil || chunks == nil {
		return nil
	}

	trigger := triggerCtx.trigger
	schemaName := triggerCtx.schemaName
	tableName := triggerCtx.tableName

	// Generate SDL text for the trigger
	triggerSDL := generateCreateTriggerSDL(schemaName, tableName, trigger)
	if triggerSDL == "" {
		return errors.New("failed to generate SDL for trigger")
	}

	// Parse the SDL to get AST node
	parseResults, err := pgparser.ParsePostgreSQL(triggerSDL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse generated trigger SDL: %s", triggerSDL)
	}
	if len(parseResults) != 1 {
		return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
	}
	parseResult := parseResults[0]

	// Extract the CREATE TRIGGER AST node
	var triggerASTNode *parser.CreatetrigstmtContext
	antlr.ParseTreeWalkerDefault.Walk(&triggerExtractor{
		result: &triggerASTNode,
	}, parseResult.Tree)

	if triggerASTNode == nil {
		return errors.New("failed to extract CREATE TRIGGER AST node")
	}

	// Create and add the chunk
	chunk := &schema.SDLChunk{
		Identifier: triggerKey,
		ASTNode:    triggerASTNode,
	}

	// Add comment if the trigger has one
	if trigger.Comment != "" {
		commentSQL := generateCommentOnTriggerSQL(schemaName, tableName, trigger.Name, trigger.Comment)
		commentParseResults, err := pgparser.ParsePostgreSQL(commentSQL)
		if err == nil && len(commentParseResults) > 0 && commentParseResults[0].Tree != nil {
			commentParseResult := commentParseResults[0]
			// Extract COMMENT ON TRIGGER AST node
			var commentASTNode *parser.CommentstmtContext
			antlr.ParseTreeWalkerDefault.Walk(&commentExtractor{
				result: &commentASTNode,
			}, commentParseResult.Tree)

			if commentASTNode != nil {
				chunk.CommentStatements = []antlr.ParserRuleContext{commentASTNode}
			}
		}
	}

	if chunks.Triggers == nil {
		chunks.Triggers = make(map[string]*schema.SDLChunk)
	}
	chunks.Triggers[triggerKey] = chunk

	return nil
}

// updateTriggerChunkIfNeeded updates an existing trigger chunk if the definition has changed
func updateTriggerChunkIfNeeded(chunks *schema.SDLChunks, currentTriggerCtx, previousTriggerCtx *triggerWithContext, triggerKey string) error {
	if currentTriggerCtx == nil || previousTriggerCtx == nil || chunks == nil {
		return nil
	}

	currentTrigger := currentTriggerCtx.trigger
	previousTrigger := previousTriggerCtx.trigger
	schemaName := currentTriggerCtx.schemaName
	tableName := currentTriggerCtx.tableName

	// Get the existing chunk
	chunk, exists := chunks.Triggers[triggerKey]
	if !exists {
		return errors.Errorf("trigger chunk not found for key %s", triggerKey)
	}

	// Check if the trigger definition has changed
	// Trigger.Body contains the complete CREATE TRIGGER statement
	if currentTrigger.Body != previousTrigger.Body {
		// Trigger definition changed - regenerate the chunk
		triggerSDL := generateCreateTriggerSDL(schemaName, tableName, currentTrigger)
		if triggerSDL == "" {
			return errors.New("failed to generate SDL for trigger")
		}

		// Parse the new SDL to get a fresh AST node
		parseResults, err := pgparser.ParsePostgreSQL(triggerSDL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse generated trigger SDL: %s", triggerSDL)
		}
		if len(parseResults) != 1 {
			return errors.Errorf("expected exactly one statement, got %d", len(parseResults))
		}
		parseResult := parseResults[0]

		// Extract the CREATE TRIGGER AST node
		var triggerASTNode *parser.CreatetrigstmtContext
		antlr.ParseTreeWalkerDefault.Walk(&triggerExtractor{
			result: &triggerASTNode,
		}, parseResult.Tree)

		if triggerASTNode == nil {
			return errors.New("failed to extract CREATE TRIGGER AST node")
		}

		// Update the chunk's AST node
		chunk.ASTNode = triggerASTNode
	}

	// Handle comment changes independently of definition changes
	if currentTrigger.Comment != previousTrigger.Comment {
		if currentTrigger.Comment != "" {
			// New or updated comment
			commentSQL := generateCommentOnTriggerSQL(schemaName, tableName, currentTrigger.Name, currentTrigger.Comment)
			commentParseResults, err := pgparser.ParsePostgreSQL(commentSQL)
			if err == nil && len(commentParseResults) > 0 && commentParseResults[0].Tree != nil {
				commentParseResult := commentParseResults[0]
				// Extract COMMENT ON TRIGGER AST node
				var commentASTNode *parser.CommentstmtContext
				antlr.ParseTreeWalkerDefault.Walk(&commentExtractor{
					result: &commentASTNode,
				}, commentParseResult.Tree)

				if commentASTNode != nil {
					chunk.CommentStatements = []antlr.ParserRuleContext{commentASTNode}
				}
			}
		} else {
			// Comment was removed
			chunk.CommentStatements = nil
		}
	}

	return nil
}

// deleteTriggerChunk removes a trigger chunk from the chunks
func deleteTriggerChunk(chunks *schema.SDLChunks, triggerKey string) {
	if chunks != nil && chunks.Triggers != nil {
		delete(chunks.Triggers, triggerKey)
	}
}

// generateCreateTriggerSDL generates the SDL text for a CREATE TRIGGER statement
func generateCreateTriggerSDL(schemaName, tableName string, trigger *storepb.TriggerMetadata) string {
	if trigger == nil {
		return ""
	}

	var buf strings.Builder
	if err := writeTriggerSDL(&buf, schemaName, tableName, trigger); err != nil {
		return ""
	}

	return buf.String()
}

// generateCommentOnTriggerSQL generates a COMMENT ON TRIGGER statement
func generateCommentOnTriggerSQL(schemaName, tableName, triggerName, comment string) string {
	if schemaName == "" {
		schemaName = "public"
	}
	// Escape single quotes in comment
	escapedComment := strings.ReplaceAll(comment, "'", "''")
	return fmt.Sprintf("COMMENT ON TRIGGER \"%s\" ON \"%s\".\"%s\" IS '%s';", triggerName, schemaName, tableName, escapedComment)
}

// triggerExtractor extracts CREATE TRIGGER AST nodes
type triggerExtractor struct {
	*parser.BasePostgreSQLParserListener
	result **parser.CreatetrigstmtContext
}

func (e *triggerExtractor) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if e.result != nil && *e.result == nil {
		*e.result = ctx
	}
}
