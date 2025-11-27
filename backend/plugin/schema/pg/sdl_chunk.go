package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// ChunkSDLText parses SDL text and extracts chunks for each database object.
func ChunkSDLText(sdlText string) (*schema.SDLChunks, error) {
	if strings.TrimSpace(sdlText) == "" {
		return &schema.SDLChunks{
			Tables:            make(map[string]*schema.SDLChunk),
			Views:             make(map[string]*schema.SDLChunk),
			MaterializedViews: make(map[string]*schema.SDLChunk),
			Functions:         make(map[string]*schema.SDLChunk),
			Triggers:          make(map[string]*schema.SDLChunk),
			Indexes:           make(map[string]*schema.SDLChunk),
			Sequences:         make(map[string]*schema.SDLChunk),
			Schemas:           make(map[string]*schema.SDLChunk),
			EnumTypes:         make(map[string]*schema.SDLChunk),
			Extensions:        make(map[string]*schema.SDLChunk),
			EventTriggers:     make(map[string]*schema.SDLChunk),
			ColumnComments:    make(map[string]map[string]antlr.ParserRuleContext),
			IndexComments:     make(map[string]map[string]antlr.ParserRuleContext),
		}, nil
	}

	parseResults, err := pgparser.ParsePostgreSQL(sdlText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse SDL text")
	}

	extractor := &sdlChunkExtractor{
		sdlText: sdlText,
		chunks: &schema.SDLChunks{
			Tables:            make(map[string]*schema.SDLChunk),
			Views:             make(map[string]*schema.SDLChunk),
			MaterializedViews: make(map[string]*schema.SDLChunk),
			Functions:         make(map[string]*schema.SDLChunk),
			Triggers:          make(map[string]*schema.SDLChunk),
			Indexes:           make(map[string]*schema.SDLChunk),
			Sequences:         make(map[string]*schema.SDLChunk),
			Schemas:           make(map[string]*schema.SDLChunk),
			EnumTypes:         make(map[string]*schema.SDLChunk),
			Extensions:        make(map[string]*schema.SDLChunk),
			EventTriggers:     make(map[string]*schema.SDLChunk),
			ColumnComments:    make(map[string]map[string]antlr.ParserRuleContext),
			IndexComments:     make(map[string]map[string]antlr.ParserRuleContext),
		},
	}

	// Walk all parsed statements
	for _, parseResult := range parseResults {
		extractor.tokens = parseResult.Tokens
		antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)
	}

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

	// Ensure schema.tableName format
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.Tables[schemaQualifiedName] = chunk
}

func (l *sdlChunkExtractor) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	// Ensure schema.sequenceName format
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	chunk := &schema.SDLChunk{
		Identifier:      schemaQualifiedName,
		ASTNode:         ctx,
		AlterStatements: []antlr.ParserRuleContext{},
	}

	l.chunks.Sequences[schemaQualifiedName] = chunk
}

func (l *sdlChunkExtractor) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	// Extract the sequence name from ALTER SEQUENCE statement
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	// Ensure schema.sequenceName format
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	// Find the corresponding CREATE SEQUENCE chunk and append this ALTER statement
	if chunk, exists := l.chunks.Sequences[schemaQualifiedName]; exists {
		chunk.AlterStatements = append(chunk.AlterStatements, ctx)
	} else {
		// If CREATE SEQUENCE doesn't exist yet, create a placeholder chunk
		// This handles cases where ALTER appears before CREATE in the SDL text
		// (though this is unusual, we handle it for robustness)
		chunk := &schema.SDLChunk{
			Identifier:      schemaQualifiedName,
			ASTNode:         nil, // No CREATE statement yet
			AlterStatements: []antlr.ParserRuleContext{ctx},
		}
		l.chunks.Sequences[schemaQualifiedName] = chunk
	}
}

// EnterDefinestmt handles CREATE TYPE AS ENUM statements
func (l *sdlChunkExtractor) EnterDefinestmt(ctx *parser.DefinestmtContext) {
	// Check if this is CREATE TYPE AS ENUM
	if ctx.CREATE() == nil || ctx.TYPE_P() == nil || ctx.AS() == nil || ctx.ENUM_P() == nil {
		return
	}

	// Extract type name
	typeNames := ctx.AllAny_name()
	if len(typeNames) == 0 {
		return
	}

	// Get the enum type name (first Any_name)
	typeName := typeNames[0]
	identifier := pgparser.NormalizePostgreSQLAnyName(typeName)
	identifierStr := strings.Join(identifier, ".")

	// Ensure schema.enumName format (default to "public" if no schema specified)
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.EnumTypes[schemaQualifiedName] = chunk
}

func (l *sdlChunkExtractor) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	funcNameCtx := ctx.Func_name()
	if funcNameCtx == nil {
		return
	}

	// Extract function name using proper normalization
	funcNameParts := pgparser.NormalizePostgreSQLFuncName(funcNameCtx)
	if len(funcNameParts) == 0 {
		return
	}

	// Parse schema and function name directly from parts
	var schemaName string
	if len(funcNameParts) == 2 {
		// Schema qualified: schema.function_name
		schemaName = funcNameParts[0]
	} else if len(funcNameParts) == 1 {
		// Unqualified: function_name (assume public schema)
		schemaName = "public"
	} else {
		// Unexpected format
		return
	}

	// Use the unified function signature extraction
	signature := extractFunctionSignatureFromAST(ctx)
	if signature == "" {
		return
	}

	schemaQualifiedSignature := schemaName + "." + signature

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedSignature,
		ASTNode:    ctx,
	}

	l.chunks.Functions[schemaQualifiedSignature] = chunk
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

	// Ensure schema.indexName format
	var schemaQualifiedName string
	if strings.Contains(indexName, ".") {
		schemaQualifiedName = indexName
	} else {
		schemaQualifiedName = "public." + indexName
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.Indexes[schemaQualifiedName] = chunk
}

// EnterCreatetrigstmt handles CREATE TRIGGER statements
func (l *sdlChunkExtractor) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	// Check if this is CREATE TRIGGER
	if ctx.CREATE() == nil || ctx.TRIGGER() == nil {
		return
	}

	// Extract trigger name
	if ctx.Name() == nil {
		return
	}
	triggerName := pgparser.NormalizePostgreSQLName(ctx.Name())

	// Extract table name from ON clause
	tableName := extractTableNameFromTrigger(ctx)
	if tableName == "" {
		return
	}

	// Parse schema and table from table name (format: schema.table)
	parts := strings.Split(tableName, ".")
	var schemaName string
	var unqualifiedTableName string
	if len(parts) == 2 {
		schemaName = parts[0]
		unqualifiedTableName = parts[1]
	} else {
		schemaName = "public"
		unqualifiedTableName = tableName
	}

	// Create identifier: schema.table.trigger_name (table-scoped)
	schemaQualifiedName := schemaName + "." + unqualifiedTableName + "." + triggerName

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.Triggers[schemaQualifiedName] = chunk
}

func (l *sdlChunkExtractor) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if ctx.Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	// Ensure schema.viewName format
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.Views[schemaQualifiedName] = chunk
}

// EnterCreatematviewstmt handles CREATE MATERIALIZED VIEW statements
func (l *sdlChunkExtractor) EnterCreatematviewstmt(ctx *parser.CreatematviewstmtContext) {
	// Extract materialized view name from create_mv_target
	if ctx.Create_mv_target() == nil || ctx.Create_mv_target().Qualified_name() == nil {
		return
	}

	identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Create_mv_target().Qualified_name())
	identifierStr := strings.Join(identifier, ".")

	// Ensure schema.materializedViewName format
	var schemaQualifiedName string
	if strings.Contains(identifierStr, ".") {
		schemaQualifiedName = identifierStr
	} else {
		schemaQualifiedName = "public." + identifierStr
	}

	chunk := &schema.SDLChunk{
		Identifier: schemaQualifiedName,
		ASTNode:    ctx,
	}

	l.chunks.MaterializedViews[schemaQualifiedName] = chunk
}

func (l *sdlChunkExtractor) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	// Extract schema name
	var schemaName string

	// Schema name can be either from optschemaname or colid
	if ctx.Colid() != nil {
		schemaName = pgparser.NormalizePostgreSQLColid(ctx.Colid())
	} else if ctx.Optschemaname() != nil && ctx.Optschemaname().Colid() != nil {
		schemaName = pgparser.NormalizePostgreSQLColid(ctx.Optschemaname().Colid())
	} else {
		// Skip if we can't determine schema name
		return
	}

	// Skip pg_catalog and information_schema
	if schemaName == "pg_catalog" || schemaName == "information_schema" {
		return
	}

	// Create chunk for this schema
	l.chunks.Schemas[schemaName] = &schema.SDLChunk{
		Identifier: schemaName,
		ASTNode:    ctx,
	}
}

func (l *sdlChunkExtractor) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	// Extract extension name from AST
	if ctx.Name() == nil {
		return
	}

	extensionName := pgparser.NormalizePostgreSQLName(ctx.Name())

	// Extension is database-level, no schema prefix needed
	chunk := &schema.SDLChunk{
		Identifier: extensionName, // Just the extension name
		ASTNode:    ctx,
	}

	l.chunks.Extensions[extensionName] = chunk
}

func (l *sdlChunkExtractor) EnterCreateeventtrigstmt(ctx *parser.CreateeventtrigstmtContext) {
	// Extract event trigger name from AST
	if ctx.Name() == nil {
		return
	}

	eventTriggerName := pgparser.NormalizePostgreSQLName(ctx.Name())

	// Event trigger is database-level, no schema prefix needed
	chunk := &schema.SDLChunk{
		Identifier: eventTriggerName, // Just the event trigger name
		ASTNode:    ctx,
	}

	l.chunks.EventTriggers[eventTriggerName] = chunk
}

func (l *sdlChunkExtractor) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	// Extract the comment text (can be sconst or NULL)
	// We store the entire AST node, not just the comment text

	// Check for COLUMN comment: COMMENT ON COLUMN any_name IS comment_text
	if ctx.COLUMN() != nil {
		if ctx.Any_name() != nil {
			// Extract schema.table.column from any_name
			anyName := pgparser.NormalizePostgreSQLAnyName(ctx.Any_name())
			if len(anyName) >= 3 {
				// Format: schema.table.column
				schemaName := anyName[0]
				tableName := anyName[1]
				columnName := anyName[2]
				tableIdentifier := schemaName + "." + tableName

				// Ensure the map for this table exists
				if l.chunks.ColumnComments[tableIdentifier] == nil {
					l.chunks.ColumnComments[tableIdentifier] = make(map[string]antlr.ParserRuleContext)
				}
				l.chunks.ColumnComments[tableIdentifier][columnName] = ctx
			} else if len(anyName) == 2 {
				// Format: table.column (assume public schema)
				tableName := anyName[0]
				columnName := anyName[1]
				tableIdentifier := "public." + tableName

				if l.chunks.ColumnComments[tableIdentifier] == nil {
					l.chunks.ColumnComments[tableIdentifier] = make(map[string]antlr.ParserRuleContext)
				}
				l.chunks.ColumnComments[tableIdentifier][columnName] = ctx
			}
		}
		return
	}

	// Check for FUNCTION/PROCEDURE comment
	// Note: ctx.FUNCTION() is true for both COMMENT ON FUNCTION and COMMENT ON PROCEDURE
	// We need to check ctx.PROCEDURE() to distinguish between them
	if ctx.FUNCTION() != nil || ctx.PROCEDURE() != nil {
		if ctx.Function_with_argtypes() != nil {
			funcWithArgsCtx := ctx.Function_with_argtypes()

			// Extract function name from func_name
			var funcNameParts []string
			if funcWithArgsCtx.Func_name() != nil {
				funcNameParts = pgparser.NormalizePostgreSQLFuncName(funcWithArgsCtx.Func_name())
			} else if funcWithArgsCtx.Colid() != nil {
				// Single identifier case
				funcNameParts = []string{pgparser.NormalizePostgreSQLColid(funcWithArgsCtx.Colid())}
			} else {
				// Fallback: use the whole text
				funcNameParts = []string{funcWithArgsCtx.GetText()}
			}

			// Determine schema name
			var schemaName string
			var funcName string
			if len(funcNameParts) >= 2 {
				schemaName = funcNameParts[0]
				funcName = funcNameParts[1]
			} else if len(funcNameParts) == 1 {
				schemaName = "public"
				funcName = funcNameParts[0]
			} else {
				return
			}

			// Try to find matching function by name prefix (schema.funcname)
			// This is necessary because CREATE FUNCTION uses parameter names and normalized types,
			// while COMMENT ON FUNCTION may use different type representations
			funcNamePrefix := schemaName + "." + funcName
			var matchedChunk *schema.SDLChunk
			for identifier, chunk := range l.chunks.Functions {
				if strings.HasPrefix(identifier, funcNamePrefix+"(") || identifier == funcNamePrefix {
					matchedChunk = chunk
					break
				}
			}

			if matchedChunk != nil {
				// Add comment to the existing function chunk
				matchedChunk.CommentStatements = append(matchedChunk.CommentStatements, ctx)
			}
			// If no match found, we don't create a new chunk because the function
			// should already exist from CREATE FUNCTION statement
		}
		return
	}

	// Check for TYPE comment: COMMENT ON TYPE typename IS comment_text
	if ctx.TYPE_P() != nil {
		// Extract typename from the third child (TYPE_P is second, typename is third)
		if ctx.GetChildCount() >= 4 {
			child := ctx.GetChild(3)
			if typenameCtx, ok := child.(*parser.TypenameContext); ok {
				// Extract schema.type from typename
				identifier := extractSchemaAndTypeFromTypename(typenameCtx)
				if identifier != "" {
					if chunk, exists := l.chunks.EnumTypes[identifier]; exists {
						chunk.CommentStatements = append(chunk.CommentStatements, ctx)
					} else {
						// Create a new chunk for the comment (type may not have CREATE TYPE statement)
						chunk := &schema.SDLChunk{
							Identifier:        identifier,
							ASTNode:           nil,
							CommentStatements: []antlr.ParserRuleContext{ctx},
						}
						l.chunks.EnumTypes[identifier] = chunk
					}
				}
			}
		}
		return
	}

	// Check for object_type_any_name: TABLE, SEQUENCE, VIEW, INDEX, etc.
	if ctx.Object_type_any_name() != nil && ctx.Any_name() != nil {
		objectType := ctx.Object_type_any_name().GetText()
		anyName := pgparser.NormalizePostgreSQLAnyName(ctx.Any_name())

		// Debug: log the object type to understand what we're getting
		// This will help us identify the issue with MATERIALIZED VIEW comments
		_ = objectType // Prevent unused variable error in production

		var identifier string
		if len(anyName) >= 2 {
			// Format: schema.object
			identifier = strings.Join(anyName, ".")
		} else if len(anyName) == 1 {
			// Format: object (assume public schema)
			identifier = "public." + anyName[0]
		} else {
			return
		}

		// Route to appropriate chunk map based on object type
		// Note: For "MATERIALIZED VIEW", the parser may return "MATERIALIZEDVIEW" (no space)
		objectTypeUpper := strings.ToUpper(objectType)
		switch objectTypeUpper {
		case "TABLE":
			if chunk, exists := l.chunks.Tables[identifier]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        identifier,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.Tables[identifier] = chunk
			}
		case "VIEW":
			if chunk, exists := l.chunks.Views[identifier]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        identifier,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.Views[identifier] = chunk
			}
		case "MATERIALIZED VIEW", "MATERIALIZEDVIEW":
			// Handle both "MATERIALIZED VIEW" (with space) and "MATERIALIZEDVIEW" (no space)
			// The parser may return either depending on the grammar definition
			if chunk, exists := l.chunks.MaterializedViews[identifier]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        identifier,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.MaterializedViews[identifier] = chunk
			}
		case "SEQUENCE":
			if chunk, exists := l.chunks.Sequences[identifier]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        identifier,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.Sequences[identifier] = chunk
			}
		case "INDEX":
			if chunk, exists := l.chunks.Indexes[identifier]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        identifier,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.Indexes[identifier] = chunk
			}
		case "TYPE":
			// Handle COMMENT ON TYPE (for enum types)
			if chunk, exists := l.chunks.EnumTypes[identifier]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        identifier,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.EnumTypes[identifier] = chunk
			}
		default:
			// Unsupported object type for comment tracking (e.g., FOREIGN TABLE)
			// We skip these for now
		}
		return
	}

	// Check for object_type_name: SCHEMA, EXTENSION, DATABASE, etc.
	if ctx.Object_type_name() != nil && ctx.Name() != nil {
		objectType := ctx.Object_type_name().GetText()
		name := pgparser.NormalizePostgreSQLName(ctx.Name())

		objectTypeUpper := strings.ToUpper(objectType)
		switch objectTypeUpper {
		case "SCHEMA":
			if chunk, exists := l.chunks.Schemas[name]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        name,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.Schemas[name] = chunk
			}
		case "EXTENSION":
			// Extensions are database-level, no schema prefix
			if chunk, exists := l.chunks.Extensions[name]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        name,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.Extensions[name] = chunk
			}
		case "EVENTTRIGGER", "EVENT TRIGGER":
			// Event triggers are database-level, no schema prefix
			if chunk, exists := l.chunks.EventTriggers[name]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        name,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.EventTriggers[name] = chunk
			}
		default:
			// Other object types are handled elsewhere
		}
		return
	}

	// Handle: COMMENT ON TRIGGER trigger_name ON table_name IS 'comment'
	if ctx.Object_type_name_on_any_name() != nil {
		objectType := ctx.Object_type_name_on_any_name().GetText()
		if strings.ToUpper(objectType) == "TRIGGER" {
			// Extract trigger name
			if ctx.Name() == nil {
				return
			}
			triggerName := pgparser.NormalizePostgreSQLName(ctx.Name())

			// Extract table name from Any_name
			if ctx.Any_name() == nil {
				return
			}
			anyName := pgparser.NormalizePostgreSQLAnyName(ctx.Any_name())
			if len(anyName) == 0 {
				return
			}

			// Build fully qualified table name
			var tableName string
			var unqualifiedTableName string
			if len(anyName) >= 2 {
				// schema.table
				tableName = strings.Join(anyName, ".")
				unqualifiedTableName = anyName[1]
			} else {
				// table (default to public schema)
				tableName = "public." + anyName[0]
				unqualifiedTableName = anyName[0]
			}

			// Parse schema from table name
			parts := strings.Split(tableName, ".")
			var schemaName string
			if len(parts) == 2 {
				schemaName = parts[0]
			} else {
				schemaName = "public"
			}

			// Use table-scoped identifier: schema.table.trigger_name
			// This matches the identifier format in EnterCreatetrigstmt
			identifier := schemaName + "." + unqualifiedTableName + "." + triggerName

			if chunk, exists := l.chunks.Triggers[identifier]; exists {
				chunk.CommentStatements = append(chunk.CommentStatements, ctx)
			} else {
				chunk := &schema.SDLChunk{
					Identifier:        identifier,
					ASTNode:           nil,
					CommentStatements: []antlr.ParserRuleContext{ctx},
				}
				l.chunks.Triggers[identifier] = chunk
			}
		}
	}

	// Check for object_type_name_on_any_name: TRIGGER, RULE, POLICY (we don't track these currently)
	// These are table-level objects that we may want to support in the future
}
