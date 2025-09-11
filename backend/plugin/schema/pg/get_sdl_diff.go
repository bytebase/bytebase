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
				// Table was modified
				tableDiff := &schema.TableDiff{
					Action:     schema.MetadataDiffActionAlter,
					SchemaName: schemaName,
					TableName:  tableName,
					OldTable:   nil, // Will be populated when SDL drift detection is implemented
					NewTable:   nil, // Will be populated when SDL drift detection is implemented
					OldASTNode: previousChunk.ASTNode.(*parser.CreatestmtContext),
					NewASTNode: currentChunk.ASTNode.(*parser.CreatestmtContext),
				}
				diff.TableChanges = append(diff.TableChanges, tableDiff)
			}
		} else {
			// New table
			tableDiff := &schema.TableDiff{
				Action:     schema.MetadataDiffActionCreate,
				SchemaName: schemaName,
				TableName:  tableName,
				OldTable:   nil,
				NewTable:   nil, // Will be populated when SDL drift detection is implemented
				OldASTNode: nil,
				NewASTNode: currentChunk.ASTNode.(*parser.CreatestmtContext),
			}
			diff.TableChanges = append(diff.TableChanges, tableDiff)
		}
	}

	// Process previous table chunks to find dropped tables
	for identifier, previousChunk := range previousChunks.Tables {
		if _, exists := currentChunks.Tables[identifier]; !exists {
			// Table was dropped
			schemaName, tableName := parseTableIdentifier(previousChunk.Identifier)
			tableDiff := &schema.TableDiff{
				Action:     schema.MetadataDiffActionDrop,
				SchemaName: schemaName,
				TableName:  tableName,
				OldTable:   nil, // Will be populated when SDL drift detection is implemented
				NewTable:   nil,
				OldASTNode: previousChunk.ASTNode.(*parser.CreatestmtContext),
				NewASTNode: nil,
			}
			diff.TableChanges = append(diff.TableChanges, tableDiff)
		}
	}

	return nil
}

// parseTableIdentifier parses a table identifier and returns schema name and table name
func parseTableIdentifier(identifier string) (schemaName, tableName string) {
	parts := strings.Split(identifier, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", identifier
}
