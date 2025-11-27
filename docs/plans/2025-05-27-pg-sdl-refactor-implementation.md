# PostgreSQL SDL Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor the 8000+ line `get_sdl_diff.go` into a well-organized `pg/sdl/` subdirectory with files split by object type.

**Architecture:** Create a new `sdl` package under `pg/` containing the SDL diff logic, chunk extraction, and migration generation. Each PostgreSQL object type (table, view, function, etc.) gets its own file containing process, drift, and helper functions.

**Tech Stack:** Go, ANTLR parser, PostgreSQL grammar

---

## Phase 1: Create Directory Structure and Core Files

### Task 1.1: Create sdl directory and diff.go entry point

**Files:**
- Create: `backend/plugin/schema/pg/sdl/diff.go`

**Step 1: Create directory**

```bash
mkdir -p backend/plugin/schema/pg/sdl
```

**Step 2: Create diff.go with package declaration and imports**

Create `backend/plugin/schema/pg/sdl/diff.go`:

```go
package sdl

import (
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterGetSDLDiff(storepb.Engine_POSTGRES, GetDiff)
	schema.RegisterGetSDLDiff(storepb.Engine_COCKROACHDB, GetDiff)
}

// GetDiff computes the difference between current SDL text and previous SDL text,
// taking into account the current and previous database schema metadata.
func GetDiff(currentSDLText, previousUserSDLText string, currentSchema, previousSchema *model.DatabaseMetadata) (*schema.MetadataDiff, error) {
	// This will be populated as we extract functions from get_sdl_diff.go
	// For now, return empty diff to make it compile
	return &schema.MetadataDiff{}, nil
}
```

**Step 3: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add backend/plugin/schema/pg/sdl/
git commit -m "refactor(pg/sdl): create sdl package with entry point"
```

---

### Task 1.2: Create chunk.go with ChunkText function

**Files:**
- Create: `backend/plugin/schema/pg/sdl/chunk.go`

**Step 1: Create chunk.go**

Extract from `get_sdl_diff.go` lines 128-856 (ChunkSDLText and sdlChunkExtractor).

Create `backend/plugin/schema/pg/sdl/chunk.go`:

```go
package sdl

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// ChunkText parses SDL text and extracts chunks for each database object.
func ChunkText(sdlText string) (*schema.SDLChunks, error) {
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

	extractor := &chunkExtractor{
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

	for _, parseResult := range parseResults {
		extractor.tokens = parseResult.Tokens
		antlr.ParseTreeWalkerDefault.Walk(extractor, parseResult.Tree)
	}

	return extractor.chunks, nil
}

type chunkExtractor struct {
	*parser.BasePostgreSQLParserListener
	sdlText string
	chunks  *schema.SDLChunks
	tokens  *antlr.CommonTokenStream
}

// Copy all Enter* methods from get_sdl_diff.go lines 187-856
// Rename sdlChunkExtractor -> chunkExtractor throughout
```

**Step 2: Copy all Enter* methods from original file**

Copy these methods from `get_sdl_diff.go` to `chunk.go`, changing receiver type from `sdlChunkExtractor` to `chunkExtractor`:
- `EnterCreatestmt` (line 187)
- `EnterCreateseqstmt` (line 211)
- `EnterAlterseqstmt` (line 236)
- `EnterDefinestmt` (line 270)
- `EnterCreatefunctionstmt` (line 303)
- `EnterIndexstmt` (line 344)
- `EnterCreatetrigstmt` (line 378)
- `EnterViewstmt` (line 419)
- `EnterCreatematviewstmt` (line 444)
- `EnterCreateschemastmt` (line 469)
- `EnterCreateextensionstmt` (line 495)
- `EnterCreateeventtrigstmt` (line 512)
- `EnterCommentstmt` (line 529)

Also copy helper functions used by these methods:
- `extractFunctionSignatureFromAST`
- `extractTableNameFromTrigger`

**Step 3: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 4: Commit**

```bash
git add backend/plugin/schema/pg/sdl/chunk.go
git commit -m "refactor(pg/sdl): extract chunk extraction to chunk.go"
```

---

### Task 1.3: Create common.go with shared utilities

**Files:**
- Create: `backend/plugin/schema/pg/sdl/common.go`

**Step 1: Create common.go with shared types and functions**

Extract commonly used functions from `get_sdl_diff.go`:

```go
package sdl

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// currentDatabaseSDLChunks holds pre-computed SDL chunks from the current database state
// for performance optimization during usability checks.
type currentDatabaseSDLChunks struct {
	chunks                   *schema.SDLChunks
	columnSDLTexts           map[string]map[string]string // tableKey -> columnName -> SDL text
	constraintSDLTexts       map[string]map[string]string // tableKey -> constraintName -> SDL text
	standaloneCommentSDLTexts map[string]string           // objectIdentifier -> comment SDL text
}

// parseIdentifier splits a qualified identifier into schema and object name.
func parseIdentifier(identifier string) (schemaName, objectName string) {
	parts := strings.Split(identifier, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", identifier
}

// extractTextFromNode extracts the original SQL text from an AST node.
func extractTextFromNode(node antlr.ParserRuleContext) string {
	// Copy implementation from get_sdl_diff.go line 1251
}

// extractAlterTexts combines multiple ALTER statement AST nodes into text.
func extractAlterTexts(alterNodes []antlr.ParserRuleContext) string {
	// Copy implementation from get_sdl_diff.go line 1235
}

// Copy other shared utility functions:
// - extractSchemaAndTypeFromTypename (line 1178)
// - extractColumnType (line 1311)
// - extractColumnNullable (line 1332)
// - extractColumnDefault (line 1354)
// - extractColumnCollation (line 1389)
// - extractAnyName (line 1406)
// - getColumnText (line 1290)
// - buildCurrentDatabaseSDLChunks (line 3719)
// - convertDatabaseSchemaToSDL (line 3981)
// - shouldSkipChunkDiffForUsability (line 3889)
// - shouldSkipCommentDiff (line 3909)
// - shouldSkipColumnDiffForUsability (line 3935)
// - shouldSkipConstraintDiffForUsability (line 3958)
```

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/common.go
git commit -m "refactor(pg/sdl): extract shared utilities to common.go"
```

---

## Phase 2: Extract Object Type Files

### Task 2.1: Create table.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/table.go`

**Step 1: Create table.go**

Extract from `get_sdl_diff.go`:
- `processTableChanges` (line 857-1015)
- `applyTableChangesToChunk` (line 3093-3189)
- `getOrCreateTableDiff` (line 2310-2344)
- Related helper functions

```go
package sdl

import (
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// processTableChanges detects changes between current and previous table definitions.
func processTableChanges(currentChunks, previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) error {
	// Copy implementation from get_sdl_diff.go lines 857-1015
}

// applyTableChangesToChunk applies drift changes to a table chunk.
func applyTableChangesToChunk(chunk *schema.SDLChunk, currentTable, previousTable *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) error {
	// Copy implementation from get_sdl_diff.go lines 3093-3189
}

// Copy helper functions used by table processing
```

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/table.go
git commit -m "refactor(pg/sdl): extract table processing to table.go"
```

---

### Task 2.2: Create column.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/column.go`

**Step 1: Create column.go**

Extract from `get_sdl_diff.go`:
- `processColumnChanges` (line 1016-1112)
- `applyColumnChanges` (line 3190-3248)
- `extractColumnDefinitionsWithAST` (line 1113-1149)
- `extractColumnMetadata` (line 1150-1167)
- `deleteColumnFromAST` (line 3514-3611)
- `modifyColumnInAST` (line 3612-3632)
- `addColumnToAST` (line 3633-3677)
- `generateColumnSDL` (line 3678-3694)
- `columnsEqual` (line 3695-3718)
- `ColumnDefWithASTOrdered` type and related

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/column.go
git commit -m "refactor(pg/sdl): extract column processing to column.go"
```

---

### Task 2.3: Create constraint.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/constraint.go`

**Step 1: Create constraint.go**

Extract from `get_sdl_diff.go`:
- `processForeignKeyChanges` (line 1452)
- `processCheckConstraintChanges` (line 1520)
- `processExcludeConstraintChanges` (line 1588)
- `processPrimaryKeyChanges` (line 1655)
- `processUniqueConstraintChanges` (line 1712)
- `extractUniqueConstraintDefinitionsInOrder` (line 1779)
- `extractForeignKeyDefinitionsInOrder` (line 1830)
- `extractCheckConstraintDefinitionsInOrder` (line 1878)
- `extractExcludeConstraintDefinitionsInOrder` (line 1926)
- `extractPrimaryKeyDefinitionsWithAST` (line 1974)
- `applyConstraintChanges` (line 3249)
- `deleteConstraintFromAST` (line 3996)
- `modifyConstraintInAST` (line 4096)
- `addConstraintToAST` (line 4137)
- `generateCheckConstraintSDL` (line 4198)
- `generateForeignKeyConstraintSDL` (line 4215)
- `generateExcludeConstraintSDL` (line 4232)
- `constraintsEqual` (line 4249)
- `fkConstraintsEqual` (line 4258)
- Helper functions: `getForeignKeyText`, `getCheckConstraintText`, `getExcludeConstraintText`, `getIndexText`
- Types: `ForeignKeyDefWithAST`, `CheckConstraintDefWithAST`, `ExcludeConstraintDefWithAST`, `IndexDefWithAST`

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/constraint.go
git commit -m "refactor(pg/sdl): extract constraint processing to constraint.go"
```

---

### Task 2.4: Create index.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/index.go`

**Step 1: Create index.go**

Extract from `get_sdl_diff.go`:
- `processStandaloneIndexChanges` (line 2110-2256)
- `applyStandaloneIndexChangesToChunks` (line 4528)
- `getStandaloneIndexText` (line 2257)
- `extractTableNameFromIndex` (line 2280)
- `isIndexOnMaterializedView` (line 2374)
- `getOrCreateMaterializedViewDiff` (line 2345)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/index.go
git commit -m "refactor(pg/sdl): extract index processing to index.go"
```

---

### Task 2.5: Create view.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/view.go`

**Step 1: Create view.go**

Extract from `get_sdl_diff.go`:
- `processViewChanges` (line 2391-2492)
- `applyViewChangesToChunks` (line 6087-6158)
- `formatViewKey` (line 6159)
- `createViewChunk` (line 6167)
- `updateViewChunkIfNeeded` (line 6233)
- `deleteViewChunk` (line 6293)
- `viewDefinitionsEqualExcludingComment` (line 6300)
- `generateCreateViewSDL` (line 6315)
- `generateCommentOnViewSQL` (line 6329)
- `viewExtractor` type and `EnterViewstmt` (line 6344)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/view.go
git commit -m "refactor(pg/sdl): extract view processing to view.go"
```

---

### Task 2.6: Create materialized_view.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/materialized_view.go`

**Step 1: Create materialized_view.go**

Extract from `get_sdl_diff.go`:
- `processMaterializedViewChanges` (line 2493-2593)
- `applyMaterializedViewChangesToChunks` (line 6351-6421)
- `createMaterializedViewChunk` (line 6422)
- `updateMaterializedViewChunkIfNeeded` (line 6488)
- `deleteMaterializedViewChunk` (line 6562)
- `generateCreateMaterializedViewSDL` (line 6569)
- `generateCommentOnMaterializedViewSQL` (line 6583)
- `materializedViewExtractor` type (line 6859)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/materialized_view.go
git commit -m "refactor(pg/sdl): extract materialized view processing to materialized_view.go"
```

---

### Task 2.7: Create function.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/function.go`

**Step 1: Create function.go**

Extract from `get_sdl_diff.go`:
- `processFunctionChanges` (line 2594-2670)
- `applyFunctionChangesToChunks` (line 4876)
- `generateCreateFunctionSDL` (line 5139)
- `extractFunctionTextFromAST` (line 5154)
- `functionDefinitionsEqual` (line 5175)
- `functionDefinitionsEqualExcludingComment` (line 5190)
- `extractFunctionASTFromSDL` (line 5215)
- `functionExtractor` type (line 5252)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/function.go
git commit -m "refactor(pg/sdl): extract function processing to function.go"
```

---

### Task 2.8: Create procedure.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/procedure.go`

**Step 1: Create procedure.go**

Procedure processing is similar to function. Extract any procedure-specific functions. If procedures share most logic with functions, this file may be small or merged.

Check for procedure-specific code in `get_sdl_diff.go` and extract accordingly.

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/procedure.go
git commit -m "refactor(pg/sdl): extract procedure processing to procedure.go"
```

---

### Task 2.9: Create sequence.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/sequence.go`

**Step 1: Create sequence.go**

Extract from `get_sdl_diff.go`:
- `processSequenceChanges` (line 2671-2894)
- `applySequenceChangesToChunks` (line 5260)
- `formatSequenceKey` (line 5336)
- `extractSchemaFromSequenceKey` (line 5344)
- `createSequenceChunk` (line 5353)
- `updateSequenceChunkIfNeeded` (line 5394)
- `syncAlterSequenceStatements` (line 5452)
- `syncCommentStatements` (line 5484)
- `syncObjectCommentStatements` (line 5495)
- `extractAlterSequenceASTFromSDL` (line 5528)
- `extractCommentASTFromSDL` (line 5562)
- `alterSequenceExtractor` (line 5601)
- `commentExtractor` (line 5613)
- `deleteSequenceChunk` (line 5620)
- `generateCreateSequenceSDL` (line 5627)
- `sequenceDefinitionsEqual` (line 5646)
- `sequenceDefinitionsEqualExcludingCommentAndOwner` (line 5667)
- `extractSequenceASTFromSDL` (line 5695)
- `extractSequenceTextFromAST` (line 5727)
- `sequenceExtractor` (line 5753)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/sequence.go
git commit -m "refactor(pg/sdl): extract sequence processing to sequence.go"
```

---

### Task 2.10: Create trigger.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/trigger.go`

**Step 1: Create trigger.go**

Extract from `get_sdl_diff.go`:
- `processStandaloneTriggerChanges` (called from main diff function)
- `applyTriggerChangesToChunks` (line 7721)
- `createTriggerChunk` (line 7823)
- `updateTriggerChunkIfNeeded` (line 7891)
- `deleteTriggerChunk` (line 7968)
- `generateCreateTriggerSDL` (line 7975)
- `generateCommentOnTriggerSQL` (line 7989)
- `triggerExtractor` (line 8004)
- `extractTableNameFromTrigger` (line 7694)
- `triggerWithContext` type

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/trigger.go
git commit -m "refactor(pg/sdl): extract trigger processing to trigger.go"
```

---

### Task 2.11: Create enum_type.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/enum_type.go`

**Step 1: Create enum_type.go**

Extract from `get_sdl_diff.go`:
- `processEnumTypeChanges` (line 7012-7113)
- `applyEnumTypeChangesToChunks` (line 6593)
- `createEnumTypeChunk` (line 6664)
- `updateEnumTypeChunkIfNeeded` (line 6724)
- `deleteEnumTypeChunk` (line 6783)
- `enumTypesEqual` (line 6790)
- `generateCreateEnumTypeSDL` (line 6814)
- `generateCommentOnTypeSQL` (line 6829)
- `enumTypeExtractor` (line 6844)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/enum_type.go
git commit -m "refactor(pg/sdl): extract enum type processing to enum_type.go"
```

---

### Task 2.12: Create extension.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/extension.go`

**Step 1: Create extension.go**

Extract from `get_sdl_diff.go`:
- `processExtensionChanges` (line 7114-7205)
- `applyExtensionChangesToChunks` (line 7422)
- `createExtensionChunk` (line 7489)
- `updateExtensionChunkIfNeeded` (line 7548)
- `deleteExtensionChunk` (line 7621)
- `generateCreateExtensionSQL` (line 7629)
- `generateCommentOnExtensionSQL` (line 7659)
- `extensionExtractor` (line 7671)
- `extensionsEqual` (line 7678)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/extension.go
git commit -m "refactor(pg/sdl): extract extension processing to extension.go"
```

---

### Task 2.13: Create schema.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/schema.go`

**Step 1: Create schema.go**

Extract from `get_sdl_diff.go`:
- `processSchemaChanges` (line 7285-7330)
- `addImplicitSchemaCreation` (line 7331-7421)
- `processEventTriggerChanges` (line 7206-7284)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/schema.go
git commit -m "refactor(pg/sdl): extract schema processing to schema.go"
```

---

### Task 2.14: Create comment.go

**Files:**
- Create: `backend/plugin/schema/pg/sdl/comment.go`

**Step 1: Create comment.go**

Extract from `get_sdl_diff.go`:
- `processCommentChanges` (line 5762-5783)
- `buildCreatedObjectsSet` (line 5784)
- `buildDroppedObjectsSet` (line 5847)
- `processObjectComments` (line 5912)
- `processColumnComments` (line 5971)
- `extractCommentTextFromChunk` (line 6031)
- `extractCommentTextFromNode` (line 6041)
- `getFirstCommentNode` (line 6078)
- `applyColumnCommentChanges` (line 6867)
- `syncColumnComment` (line 6966)

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/comment.go
git commit -m "refactor(pg/sdl): extract comment processing to comment.go"
```

---

## Phase 3: Complete diff.go and Move Migration

### Task 3.1: Complete diff.go with full GetDiff implementation

**Files:**
- Modify: `backend/plugin/schema/pg/sdl/diff.go`

**Step 1: Update diff.go with complete implementation**

Now that all object-type functions are extracted, complete the `GetDiff` function:

```go
package sdl

import (
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterGetSDLDiff(storepb.Engine_POSTGRES, GetDiff)
	schema.RegisterGetSDLDiff(storepb.Engine_COCKROACHDB, GetDiff)
}

// GetDiff computes the difference between current SDL text and previous SDL text.
func GetDiff(currentSDLText, previousUserSDLText string, currentSchema, previousSchema *model.DatabaseMetadata) (*schema.MetadataDiff, error) {
	generatedSDL, err := convertDatabaseSchemaToSDL(currentSchema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert current schema to SDL format for initialization")
	}

	if strings.TrimSpace(previousUserSDLText) == "" && currentSchema != nil {
		previousUserSDLText = generatedSDL
	}

	if currentSchema != nil && strings.TrimSpace(currentSDLText) == strings.TrimSpace(generatedSDL) {
		return &schema.MetadataDiff{}, nil
	}

	currentChunks, err := ChunkText(currentSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk current SDL text")
	}

	previousChunks, err := ChunkText(previousUserSDLText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to chunk previous SDL text")
	}

	if currentSchema != nil && previousSchema != nil {
		err = applyMinimalChangesToChunks(previousChunks, currentSchema, previousSchema)
		if err != nil {
			return nil, errors.Wrap(err, "failed to apply minimal changes to SDL chunks")
		}
	}

	currentDBSDLChunks, err := buildCurrentDatabaseSDLChunks(currentSchema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build current database SDL chunks")
	}

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

	if err := processTableChanges(currentChunks, previousChunks, currentSchema, previousSchema, currentDBSDLChunks, diff); err != nil {
		return nil, errors.Wrap(err, "failed to process table changes")
	}

	processStandaloneIndexChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processStandaloneTriggerChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processViewChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processMaterializedViewChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processFunctionChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processSequenceChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processEnumTypeChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processExtensionChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processEventTriggerChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	processSchemaChanges(currentChunks, previousChunks, currentSchema, diff)
	processCommentChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)
	addImplicitSchemaCreation(diff, currentSchema)

	return diff, nil
}

// applyMinimalChangesToChunks synchronizes previousChunks with database drift.
func applyMinimalChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseMetadata) error {
	// Copy implementation from get_sdl_diff.go line 2895
	// This calls into various applyXxxChangesToChunks functions
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 3: Commit**

```bash
git add backend/plugin/schema/pg/sdl/diff.go
git commit -m "refactor(pg/sdl): complete GetDiff implementation"
```

---

### Task 3.2: Move generate_migration.go to sdl/migration.go

**Files:**
- Move: `backend/plugin/schema/pg/generate_migration.go` -> `backend/plugin/schema/pg/sdl/migration.go`

**Step 1: Copy file and update package**

```bash
cp backend/plugin/schema/pg/generate_migration.go backend/plugin/schema/pg/sdl/migration.go
```

**Step 2: Update package declaration**

Change first line from `package pg` to `package sdl`.

**Step 3: Update imports and internal references**

- Update any internal function calls that now need qualification
- Ensure all dependencies are imported correctly

**Step 4: Verify it compiles**

```bash
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 5: Commit**

```bash
git add backend/plugin/schema/pg/sdl/migration.go
git commit -m "refactor(pg/sdl): move migration generation to sdl/migration.go"
```

---

## Phase 4: Move Tests

### Task 4.1: Move SDL-related test files to sdl/

**Files:**
- Move multiple test files from `backend/plugin/schema/pg/` to `backend/plugin/schema/pg/sdl/`

**Step 1: Move test files**

```bash
cd backend/plugin/schema/pg

# Move SDL diff tests
mv get_sdl_diff_test.go sdl/diff_test.go
mv get_sdl_diff_comment_test.go sdl/comment_test.go
mv chunk_extraction_test.go sdl/chunk_test.go

# Move object-specific tests
mv column_sdl_diff_test.go sdl/column_test.go
mv view_sdl_diff_test.go sdl/view_test.go
mv materialized_view_sdl_diff_test.go sdl/materialized_view_test.go
mv function_sdl_diff_test.go sdl/function_test.go
mv procedure_sdl_diff_test.go sdl/procedure_test.go
mv sequence_sdl_diff_test.go sdl/sequence_test.go
mv trigger_sdl_diff_test.go sdl/trigger_test.go

# Move integration tests
mv column_migration_integration_test.go sdl/
mv table_migration_integration_test.go sdl/
mv view_migration_integration_test.go sdl/
mv function_migration_integration_test.go sdl/
mv sequence_migration_integration_test.go sdl/
mv comment_migration_test.go sdl/

# Move other SDL-related tests
mv enum_type_test.go sdl/
mv extension_test.go sdl/
mv standalone_index_test.go sdl/
mv table_constraints_test.go sdl/
mv usability_check_test.go sdl/
mv sdl_diff_table_test.go sdl/
mv schema_implicit_creation_test.go sdl/
mv trigger_drift_test.go sdl/
mv trigger_drop_dependency_test.go sdl/
mv exclude_index_test.go sdl/
mv commented_enum_test.go sdl/
mv enum_special_chars_test.go sdl/
mv materialized_view_index_test.go sdl/
mv procedure_comment_roundtrip_test.go sdl/
mv function_rewrite_test.go sdl/
mv sequence_rewrite_test.go sdl/
mv index_rewrite_test.go sdl/
```

**Step 2: Update package declarations in all moved test files**

Change `package pg` to `package sdl` in each file.

**Step 3: Update function references**

- Change `GetSDLDiff` to `GetDiff`
- Change `ChunkSDLText` to `ChunkText`
- Update any other renamed functions

**Step 4: Verify tests compile**

```bash
go test -c ./backend/plugin/schema/pg/sdl/...
```

**Step 5: Commit**

```bash
git add backend/plugin/schema/pg/sdl/*_test.go
git add backend/plugin/schema/pg/
git commit -m "refactor(pg/sdl): move SDL test files to sdl/"
```

---

### Task 4.2: Move testdata directory

**Files:**
- Move: `backend/plugin/schema/pg/sdl_testdata/` -> `backend/plugin/schema/pg/sdl/testdata/`

**Step 1: Move testdata**

```bash
mv backend/plugin/schema/pg/sdl_testdata backend/plugin/schema/pg/sdl/testdata
```

**Step 2: Update test file paths**

Search for `sdl_testdata` in test files and replace with `testdata`:

```bash
cd backend/plugin/schema/pg/sdl
sed -i '' 's/sdl_testdata/testdata/g' *_test.go
```

**Step 3: Verify tests pass**

```bash
go test ./backend/plugin/schema/pg/sdl/... -v -count=1 -run TestChunk
```

**Step 4: Commit**

```bash
git add backend/plugin/schema/pg/sdl/testdata/
git add backend/plugin/schema/pg/sdl/*_test.go
git commit -m "refactor(pg/sdl): move testdata to sdl/testdata/"
```

---

## Phase 5: Cleanup and Verification

### Task 5.1: Remove old get_sdl_diff.go

**Files:**
- Delete: `backend/plugin/schema/pg/get_sdl_diff.go`

**Step 1: Verify all functions have been moved**

```bash
# Should show nothing or only comments
grep "^func " backend/plugin/schema/pg/get_sdl_diff.go
```

**Step 2: Delete the file**

```bash
rm backend/plugin/schema/pg/get_sdl_diff.go
```

**Step 3: Verify build**

```bash
go build ./backend/plugin/schema/pg/...
go build ./backend/plugin/schema/pg/sdl/...
```

**Step 4: Commit**

```bash
git add backend/plugin/schema/pg/get_sdl_diff.go
git commit -m "refactor(pg/sdl): remove old get_sdl_diff.go"
```

---

### Task 5.2: Remove old generate_migration.go

**Files:**
- Delete: `backend/plugin/schema/pg/generate_migration.go`

**Step 1: Delete the file**

```bash
rm backend/plugin/schema/pg/generate_migration.go
```

**Step 2: Update any imports**

Search for imports of the old location and update to use sdl package:

```bash
grep -r "plugin/schema/pg\"" backend/ --include="*.go" | grep -v "_test.go" | head -20
```

For files that call `WriteMigrationSQL`, update to import sdl package and call `sdl.WriteMigrationSQL`.

**Step 3: Verify build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add backend/plugin/schema/pg/generate_migration.go
git add backend/  # any updated imports
git commit -m "refactor(pg/sdl): remove old generate_migration.go and update imports"
```

---

### Task 5.3: Run full test suite

**Step 1: Run all SDL tests**

```bash
go test -v -count=1 ./backend/plugin/schema/pg/sdl/...
```

Expected: All tests pass

**Step 2: Run all pg schema tests**

```bash
go test -v -count=1 ./backend/plugin/schema/pg/...
```

Expected: All tests pass

**Step 3: Run linter**

```bash
golangci-lint run --allow-parallel-runners ./backend/plugin/schema/pg/...
```

Expected: No errors

**Step 4: Commit any fixes**

```bash
git add .
git commit -m "fix(pg/sdl): fix issues found during testing"
```

---

### Task 5.4: Final verification and cleanup

**Step 1: Verify directory structure**

```bash
ls -la backend/plugin/schema/pg/sdl/
```

Expected structure:
```
sdl/
├── diff.go
├── chunk.go
├── common.go
├── table.go
├── column.go
├── constraint.go
├── index.go
├── view.go
├── materialized_view.go
├── function.go
├── procedure.go
├── sequence.go
├── trigger.go
├── enum_type.go
├── extension.go
├── schema.go
├── comment.go
├── migration.go
├── *_test.go
└── testdata/
```

**Step 2: Verify file sizes are reasonable**

```bash
wc -l backend/plugin/schema/pg/sdl/*.go | sort -n
```

Expected: Most files < 1000 lines (except migration.go)

**Step 3: Full build**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: Build succeeds

**Step 4: Final commit**

```bash
git add .
git commit -m "refactor(pg/sdl): complete SDL refactoring to pg/sdl package"
```

---

## Summary

This plan extracts the 8000+ line `get_sdl_diff.go` into a well-organized `pg/sdl/` package with:

- **17 source files** organized by object type
- **~30 test files** moved alongside their code
- **Clear separation** of concerns
- **Extensibility** - adding new object types follows the same pattern

### File Count Summary

| Category | Files |
|----------|-------|
| Core (diff, chunk, common) | 3 |
| Object types | 12 |
| Migration | 1 |
| Tests | ~30 |
| Testdata | 1 directory |

### Estimated Total Lines by File

| File | ~Lines |
|------|--------|
| diff.go | 300 |
| chunk.go | 500 |
| common.go | 400 |
| table.go | 800 |
| column.go | 600 |
| constraint.go | 800 |
| index.go | 300 |
| view.go | 400 |
| materialized_view.go | 400 |
| function.go | 400 |
| procedure.go | 200 |
| sequence.go | 600 |
| trigger.go | 400 |
| enum_type.go | 300 |
| extension.go | 300 |
| schema.go | 200 |
| comment.go | 400 |
| migration.go | 5000 |
