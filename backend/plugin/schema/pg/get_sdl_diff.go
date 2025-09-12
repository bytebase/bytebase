package pg

import (
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

func GetSDLDiff(currentSDLText, previousUserSDLText string, currentSchema, previousSchema *model.DatabaseSchema) (*schema.MetadataDiff, error) {
	currentChunks, err := ChunkSDLText(currentSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk current SDL text")
	}

	previousChunks, err := ChunkSDLText(previousUserSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk previous SDL text")
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
	}

	// Process table changes
	err = processTableChanges(currentChunks, previousChunks, currentSchema, previousSchema, diff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process table changes")
	}

	return diff, nil
}

func ChunkSDLText(sdlText string) (*schema.SDLChunks, error) {
	if strings.TrimSpace(sdlText) == "" {
		return &schema.SDLChunks{
			Tables:    make(map[string]*schema.SDLChunk),
			Views:     make(map[string]*schema.SDLChunk),
			Functions: make(map[string]*schema.SDLChunk),
			Indexes:   make(map[string]*schema.SDLChunk),
			Sequences: make(map[string]*schema.SDLChunk),
			Tokens:    nil,
		}, nil
	}

	parseResult, err := pgparser.ParsePostgreSQL(sdlText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse SDL text")
	}

	extractor := &sdlChunkExtractor{
		sdlText: sdlText,
		chunks: &schema.SDLChunks{
			Tables:    make(map[string]*schema.SDLChunk),
			Views:     make(map[string]*schema.SDLChunk),
			Functions: make(map[string]*schema.SDLChunk),
			Indexes:   make(map[string]*schema.SDLChunk),
			Sequences: make(map[string]*schema.SDLChunk),
			Tokens:    parseResult.Tokens,
		},
		tokens: parseResult.Tokens,
	}

	antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)

	return extractor.chunks, nil
}

type sdlChunkExtractor struct {
	*parser.BasePostgreSQLParserListener
	sdlText string
	chunks  *schema.SDLChunks
	tokens  *antlr.CommonTokenStream
}

func (l *sdlChunkExtractor) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if ctx.Qualified_name(0) == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name(0))
	identifierStr := strings.Join(identifier, ".")

	chunk := &schema.SDLChunk{
		Identifier: identifierStr,
		ASTNode:    ctx,
	}

	l.chunks.Tables[identifierStr] = chunk
}

func (l *sdlChunkExtractor) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	chunk := &schema.SDLChunk{
		Identifier: identifierStr,
		ASTNode:    ctx,
	}

	l.chunks.Sequences[identifierStr] = chunk
}

func (l *sdlChunkExtractor) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	funcNameCtx := ctx.Func_name()
	if funcNameCtx == nil {
		return
	}

	// Extract function name directly from the text
	funcName := funcNameCtx.GetText()
	if funcName == "" {
		return
	}

	chunk := &schema.SDLChunk{
		Identifier: funcName,
		ASTNode:    ctx,
	}

	l.chunks.Functions[funcName] = chunk
}

func (l *sdlChunkExtractor) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	// Check if this is CREATE INDEX
	if ctx.CREATE() == nil || ctx.INDEX() == nil {
		return
	}

	// Extract index name
	var indexName string
	if name := ctx.Name(); name != nil {
		indexName = pgparser.NormalizePostgreSQLName(name)
	}

	// If no explicit name, we skip it as PostgreSQL will generate one
	if indexName == "" {
		return
	}

	chunk := &schema.SDLChunk{
		Identifier: indexName,
		ASTNode:    ctx,
	}

	l.chunks.Indexes[indexName] = chunk
}

func (l *sdlChunkExtractor) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	chunk := &schema.SDLChunk{
		Identifier: identifierStr,
		ASTNode:    ctx,
	}

	l.chunks.Views[identifierStr] = chunk
}

// processTableChanges processes changes to tables by comparing SDL chunks
// nolint:unparam
func processTableChanges(currentChunks, previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema, diff *schema.MetadataDiff) error {
	// TODO: currentSchema and previousSchema will be used later for SDL drift detection
	_ = currentSchema
	_ = previousSchema

	// Process current table chunks to find created and modified tables
	for identifier, currentChunk := range currentChunks.Tables {
		schemaName, tableName := parseTableIdentifier(currentChunk.Identifier)

		if previousChunk, exists := previousChunks.Tables[identifier]; exists {
			// Table exists in both - check if modified
			currentText := currentChunk.GetText(currentChunks.Tokens)
			previousText := previousChunk.GetText(previousChunks.Tokens)
			if currentText != previousText {
				// Table was modified - process column changes
				oldASTNode, ok := previousChunk.ASTNode.(*parser.CreatestmtContext)
				if !ok {
					return errors.Errorf("expected CreatestmtContext for previous table %s", previousChunk.Identifier)
				}
				newASTNode, ok := currentChunk.ASTNode.(*parser.CreatestmtContext)
				if !ok {
					return errors.Errorf("expected CreatestmtContext for current table %s", currentChunk.Identifier)
				}

				columnChanges := processColumnChanges(oldASTNode, newASTNode)

				tableDiff := &schema.TableDiff{
					Action:        schema.MetadataDiffActionAlter,
					SchemaName:    schemaName,
					TableName:     tableName,
					OldTable:      nil, // Will be populated when SDL drift detection is implemented
					NewTable:      nil, // Will be populated when SDL drift detection is implemented
					OldASTNode:    oldASTNode,
					NewASTNode:    newASTNode,
					ColumnChanges: columnChanges,
				}
				diff.TableChanges = append(diff.TableChanges, tableDiff)
			}
		} else {
			// New table
			newASTNode, ok := currentChunk.ASTNode.(*parser.CreatestmtContext)
			if !ok {
				return errors.Errorf("expected CreatestmtContext for new table %s", currentChunk.Identifier)
			}

			tableDiff := &schema.TableDiff{
				Action:        schema.MetadataDiffActionCreate,
				SchemaName:    schemaName,
				TableName:     tableName,
				OldTable:      nil,
				NewTable:      nil, // Will be populated when SDL drift detection is implemented
				OldASTNode:    nil,
				NewASTNode:    newASTNode,
				ColumnChanges: []*schema.ColumnDiff{}, // Empty - table creation automatically creates all columns
			}
			diff.TableChanges = append(diff.TableChanges, tableDiff)
		}
	}

	// Process previous table chunks to find dropped tables
	for identifier, previousChunk := range previousChunks.Tables {
		if _, exists := currentChunks.Tables[identifier]; !exists {
			// Table was dropped
			schemaName, tableName := parseTableIdentifier(previousChunk.Identifier)
			oldASTNode, ok := previousChunk.ASTNode.(*parser.CreatestmtContext)
			if !ok {
				return errors.Errorf("expected CreatestmtContext for dropped table %s", previousChunk.Identifier)
			}

			tableDiff := &schema.TableDiff{
				Action:        schema.MetadataDiffActionDrop,
				SchemaName:    schemaName,
				TableName:     tableName,
				OldTable:      nil, // Will be populated when SDL drift detection is implemented
				NewTable:      nil,
				OldASTNode:    oldASTNode,
				NewASTNode:    nil,
				ColumnChanges: []*schema.ColumnDiff{}, // Empty - table drop doesn't need column analysis
			}
			diff.TableChanges = append(diff.TableChanges, tableDiff)
		}
	}

	return nil
}

// processColumnChanges analyzes column changes between old and new table definitions
// Following the same pattern as processTableChanges: compare text first, then analyze differences
func processColumnChanges(oldTable, newTable *parser.CreatestmtContext) []*schema.ColumnDiff {
	if oldTable == nil || newTable == nil {
		return []*schema.ColumnDiff{}
	}

	// Step 1: Extract all column definitions with their AST nodes for text comparison
	oldColumnMap := extractColumnDefinitionsWithAST(oldTable)
	newColumnMap := extractColumnDefinitionsWithAST(newTable)

	var columnDiffs []*schema.ColumnDiff

	// Step 2: Process current columns to find created and modified columns
	for columnName, newColumnDef := range newColumnMap {
		if oldColumnDef, exists := oldColumnMap[columnName]; exists {
			// Column exists in both - check if modified by comparing text first
			currentText := getColumnText(newColumnDef.ASTNode)
			previousText := getColumnText(oldColumnDef.ASTNode)
			if currentText != previousText {
				// Column was modified - only now extract detailed metadata
				oldColumn := extractColumnMetadata(oldColumnDef.ASTNode)
				newColumn := extractColumnMetadata(newColumnDef.ASTNode)

				columnDiffs = append(columnDiffs, &schema.ColumnDiff{
					Action:     schema.MetadataDiffActionAlter,
					OldColumn:  oldColumn,
					NewColumn:  newColumn,
					OldASTNode: oldColumnDef.ASTNode,
					NewASTNode: newColumnDef.ASTNode,
				})
			}
			// If text is identical, skip - no changes detected
		} else {
			// New column - extract metadata only for new columns
			newColumn := extractColumnMetadata(newColumnDef.ASTNode)
			columnDiffs = append(columnDiffs, &schema.ColumnDiff{
				Action:     schema.MetadataDiffActionCreate,
				OldColumn:  nil,
				NewColumn:  newColumn,
				OldASTNode: nil,
				NewASTNode: newColumnDef.ASTNode,
			})
		}
	}

	// Step 3: Process previous columns to find dropped columns
	for columnName, oldColumnDef := range oldColumnMap {
		if _, exists := newColumnMap[columnName]; !exists {
			// Column was dropped - extract metadata only for dropped columns
			oldColumn := extractColumnMetadata(oldColumnDef.ASTNode)
			columnDiffs = append(columnDiffs, &schema.ColumnDiff{
				Action:     schema.MetadataDiffActionDrop,
				OldColumn:  oldColumn,
				NewColumn:  nil,
				OldASTNode: oldColumnDef.ASTNode,
				NewASTNode: nil,
			})
		}
	}

	return columnDiffs
}

// ColumnDefWithAST holds a column definition AST node with its name for efficient lookup
type ColumnDefWithAST struct {
	Name    string
	ASTNode parser.IColumnDefContext
}

// extractColumnDefinitionsWithAST extracts only column name and AST node for text comparison
// This is more efficient than full metadata extraction when we only need text comparison
func extractColumnDefinitionsWithAST(createStmt *parser.CreatestmtContext) map[string]*ColumnDefWithAST {
	columns := make(map[string]*ColumnDefWithAST)

	if createStmt == nil {
		return columns
	}

	// Get the optTableElementList which contains column definitions
	if createStmt.Opttableelementlist() != nil && createStmt.Opttableelementlist().Tableelementlist() != nil {
		elementList := createStmt.Opttableelementlist().Tableelementlist()

		for _, element := range elementList.AllTableelement() {
			// Check if this is a columnDef (column definition)
			if element.ColumnDef() != nil {
				columnDef := element.ColumnDef()
				if columnDef.Colid() != nil {
					columnName := pgparser.NormalizePostgreSQLColid(columnDef.Colid())

					columns[columnName] = &ColumnDefWithAST{
						Name:    columnName,
						ASTNode: columnDef,
					}
				}
			}
		}
	}

	return columns
}

// extractColumnMetadata extracts full column metadata from a single column AST node
// This is called only when we actually need the detailed information
func extractColumnMetadata(columnDef parser.IColumnDefContext) *storepb.ColumnMetadata {
	if columnDef == nil {
		return nil
	}

	columnName := pgparser.NormalizePostgreSQLColid(columnDef.Colid())

	return &storepb.ColumnMetadata{
		Name:      columnName,
		Type:      extractColumnType(columnDef.Typename()),
		Nullable:  extractColumnNullable(columnDef),
		Default:   extractColumnDefault(columnDef),
		Comment:   extractColumnComment(columnDef),
		Collation: extractColumnCollation(columnDef),
	}
}

// parseTableIdentifier parses a table identifier and returns schema name and table name
func parseTableIdentifier(identifier string) (schemaName, tableName string) {
	parts := strings.Split(identifier, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", identifier
}

// getColumnText extracts the text representation of a column definition
func getColumnText(columnAST parser.IColumnDefContext) string {
	if columnAST == nil {
		return ""
	}

	// Get tokens from the parser
	if parser := columnAST.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := columnAST.GetStart()
			stop := columnAST.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return columnAST.GetText()
}

// extractColumnType extracts column type information from the typename AST node
func extractColumnType(typename parser.ITypenameContext) string {
	if typename == nil {
		return "unknown"
	}

	// Get tokens from the parser
	if parser := typename.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := typename.GetStart()
			stop := typename.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return typename.GetText()
}

// extractColumnNullable extracts whether a column is nullable from column constraints
func extractColumnNullable(columnDef parser.IColumnDefContext) bool {
	if columnDef == nil || columnDef.Colquallist() == nil {
		return true // Default is nullable
	}

	// Check all column constraints
	for _, constraint := range columnDef.Colquallist().AllColconstraint() {
		if constraint.Colconstraintelem() != nil {
			elem := constraint.Colconstraintelem()
			if elem.NOT() != nil && elem.NULL_P() != nil {
				return false // NOT NULL
			}
			if elem.PRIMARY() != nil && elem.KEY() != nil {
				return false // PRIMARY KEY implies NOT NULL
			}
		}
	}

	return true // Default is nullable
}

// extractColumnDefault extracts the default value from column constraints
func extractColumnDefault(columnDef parser.IColumnDefContext) string {
	if columnDef == nil || columnDef.Colquallist() == nil {
		return ""
	}

	// Check all column constraints for DEFAULT
	for _, constraint := range columnDef.Colquallist().AllColconstraint() {
		if constraint.Colconstraintelem() != nil {
			elem := constraint.Colconstraintelem()
			if elem.DEFAULT() != nil && elem.B_expr() != nil {
				// Extract default value text
				if parser := elem.GetParser(); parser != nil {
					if tokenStream := parser.GetTokenStream(); tokenStream != nil {
						start := elem.B_expr().GetStart()
						stop := elem.B_expr().GetStop()
						if start != nil && stop != nil {
							return tokenStream.GetTextFromTokens(start, stop)
						}
					}
				}
			}
		}
	}

	return ""
}

// extractColumnComment extracts column comment (PostgreSQL doesn't store comments in CREATE TABLE syntax)
func extractColumnComment(_ parser.IColumnDefContext) string {
	// PostgreSQL column comments are set using COMMENT ON COLUMN, not in CREATE TABLE
	// This information is not available in the CREATE TABLE AST
	return ""
}

// extractColumnCollation extracts collation from column constraints
func extractColumnCollation(columnDef parser.IColumnDefContext) string {
	if columnDef == nil || columnDef.Colquallist() == nil {
		return ""
	}

	// Check all column constraints for COLLATE
	for _, constraint := range columnDef.Colquallist().AllColconstraint() {
		if constraint.COLLATE() != nil && constraint.Any_name() != nil {
			// Extract collation name
			return extractAnyName(constraint.Any_name())
		}
	}

	return ""
}

// extractAnyName extracts the string representation from an Any_name context
func extractAnyName(anyName parser.IAny_nameContext) string {
	if anyName == nil {
		return ""
	}

	// Get tokens from the parser
	if parser := anyName.GetParser(); parser != nil {
		if tokenStream := parser.GetTokenStream(); tokenStream != nil {
			start := anyName.GetStart()
			stop := anyName.GetStop()
			if start != nil && stop != nil {
				return tokenStream.GetTextFromTokens(start, stop)
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return anyName.GetText()
}
