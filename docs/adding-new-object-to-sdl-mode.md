# Adding New Database Object Support to SDL Mode (bb rollout)

## Overview

This guide provides a comprehensive checklist for adding support for new database object types (e.g., Triggers, Policies, Extensions) to Bytebase's SDL mode (`bb rollout`). It is based on the complete implementation of PostgreSQL Materialized Views.

### What is SDL Mode?

SDL (Schema Definition Language) mode, accessed via `bb rollout`, is an AST-only workflow where:
- Schema changes are defined in SDL files
- The parser extracts structure from AST nodes
- No database metadata is available during diff computation
- Dependency resolution must work from AST alone

### Why This Guide?

When adding a new object type, **both metadata mode and AST-only mode must be supported**. Forgetting AST-only mode support leads to:
- ❌ Incorrect migration ordering
- ❌ "Relation does not exist" errors on CREATE
- ❌ "Cannot drop because other objects depend on it" errors on DROP
- ❌ Missing COMMENT statements in migrations

---

## Architecture Overview

### Data Flow in SDL Mode

```
SDL File → Parser → AST Nodes → SDL Chunks → Diff Detection → Topological Sort → Migration SQL
```

Key components:
1. **Parser**: Extracts AST nodes from SDL text
2. **SDL Chunks**: Groups AST nodes by object type
3. **Diff Detection**: Compares current vs previous chunks
4. **Dependency Extraction**: Extracts dependencies from AST (critical!)
5. **Topological Sort**: Orders CREATE/DROP operations
6. **Migration Generation**: Converts AST back to SQL

---

## Implementation Phases

## Phase 1: Schema Definition & Parser

### 1.1 Define Data Structures

**File**: `backend/store/model/database.go` or protobuf definitions

Add metadata structure for the new object type:

```go
message DatabaseSchemaMetadata {
    repeated TableMetadata tables = 1;
    repeated ViewMetadata views = 2;
    repeated MaterializedViewMetadata materialized_views = 3;  // ✅ Add new object
    repeated FunctionMetadata functions = 4;
    // ...
}

message MaterializedViewMetadata {
    string name = 1;
    string definition = 2;
    string comment = 3;
    repeated DependencyColumn dependency_columns = 4;  // ⚠️ Required for dependency resolution
}
```

**Key Requirements**:
- Include `DependencyColumn` field if the object can depend on other objects
- Include `comment` field for COMMENT support

---

### 1.2 Register in SDLChunks

**File**: `backend/plugin/schema/schema.go`

Add the new object type to the SDL chunks structure:

```go
type SDLChunks struct {
    Tables            map[string]*SDLChunk
    Views             map[string]*SDLChunk
    MaterializedViews map[string]*SDLChunk  // ✅ Add new object
    Functions         map[string]*SDLChunk
    Indexes           map[string]*SDLChunk
    Sequences         map[string]*SDLChunk
    Schemas           map[string]*SDLChunk
    ColumnComments    map[string]map[string]antlr.ParserRuleContext
    IndexComments     map[string]map[string]antlr.ParserRuleContext
}
```

---

### 1.3 Add AST Listener

**File**: `backend/plugin/schema/pg/get_sdl_diff.go`

Implement a listener method for the object's CREATE statement:

```go
// EnterCreatematviewstmt handles CREATE MATERIALIZED VIEW statements
func (l *sdlChunkExtractor) EnterCreatematviewstmt(ctx *parser.CreatematviewstmtContext) {
    // Extract object name from AST context
    if ctx.Create_mv_target() == nil || ctx.Create_mv_target().Qualified_name() == nil {
        return
    }

    identifier := pgparser.NormalizePostgreSQLQualifiedName(ctx.Create_mv_target().Qualified_name())
    identifierStr := strings.Join(identifier, ".")

    // Ensure schema.objectName format (default to "public" if no schema specified)
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
```

**Important**:
- Always normalize identifiers to `schema.object` format
- Store the raw AST node for later SQL extraction
- Use appropriate default schema (usually "public" for PostgreSQL)

---

### 1.4 Handle COMMENT Statements

**File**: `backend/plugin/schema/pg/get_sdl_diff.go`

Add a case in `EnterCommentstmt` to associate COMMENT statements with objects:

```go
func (l *sdlChunkExtractor) EnterCommentstmt(ctx *parser.CommentstmtContext) {
    // ... extract object type and identifier ...

    switch objectType {
    case "TABLE":
        // ... handle table comments ...
    case "VIEW":
        // ... handle view comments ...
    case "MATERIALIZED VIEW", "MATERIALIZEDVIEW":  // ⚠️ Handle both variants
        if chunk, exists := l.chunks.MaterializedViews[identifier]; exists {
            // Object already exists - append comment
            chunk.CommentStatements = append(chunk.CommentStatements, ctx)
        } else {
            // Object doesn't exist yet - create chunk with only comment
            chunk := &schema.SDLChunk{
                Identifier:        identifier,
                ASTNode:           nil,  // No CREATE statement yet
                CommentStatements: []antlr.ParserRuleContext{ctx},
            }
            l.chunks.MaterializedViews[identifier] = chunk
        }
    }
}
```

**Why handle variants?**
Different PostgreSQL grammar implementations may return "MATERIALIZED VIEW" (with space) or "MATERIALIZEDVIEW" (no space).

---

## Phase 2: SDL Diff Processing

### 2.1 Initialize Maps

**File**: `backend/plugin/schema/pg/get_sdl_diff.go` - `ChunkSDLText` function

Initialize the new object map when creating empty SDL chunks:

```go
func ChunkSDLText(sdlText string) (*schema.SDLChunks, error) {
    if strings.TrimSpace(sdlText) == "" {
        return &schema.SDLChunks{
            Tables:            make(map[string]*SDLChunk),
            Views:             make(map[string]*SDLChunk),
            MaterializedViews: make(map[string]*SDLChunk),  // ✅ Initialize
            Functions:         make(map[string]*SDLChunk),
            Indexes:           make(map[string]*SDLChunk),
            Sequences:         make(map[string]*SDLChunk),
            Schemas:           make(map[string]*SDLChunk),
            ColumnComments:    make(map[string]map[string]antlr.ParserRuleContext),
            IndexComments:     make(map[string]map[string]antlr.ParserRuleContext),
        }, nil
    }

    // ... same initialization in parser setup ...
}
```

---

### 2.2 Call Object Change Processor

**File**: `backend/plugin/schema/pg/get_sdl_diff.go` - `GetSDLDiff` function

Add a call to process the new object type:

```go
func GetSDLDiff(currentSDLText, previousUserSDLText string, currentSchema, previousSchema *model.DatabaseSchema) (*schema.MetadataDiff, error) {
    // ... parse SDL ...

    // Process table changes
    err = processTableChanges(currentChunks, previousChunks, currentSchema, previousSchema, currentDBSDLChunks, diff)
    if err != nil {
        return nil, errors.Wrap(err, "failed to process table changes")
    }

    // Process view changes
    processViewChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

    // Process materialized view changes
    processMaterializedViewChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)  // ✅ Add call

    // Process function changes
    processFunctionChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

    // ... other object types ...

    // Process comment changes (must be after all object changes)
    processCommentChanges(currentChunks, previousChunks, currentDBSDLChunks, diff)

    return diff, nil
}
```

**Order matters**: Comment processing must come after all object processing.

---

### 2.3 Implement Object Change Detection

**File**: `backend/plugin/schema/pg/get_sdl_diff.go`

Create a function to detect CREATE/DROP/MODIFY operations:

```go
func processMaterializedViewChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
    // Iterate through current objects to find CREATE and MODIFY
    for _, currentChunk := range currentChunks.MaterializedViews {
        if previousChunk, exists := previousChunks.MaterializedViews[currentChunk.Identifier]; exists {
            // Object exists in both - check if modified by comparing text (excluding comments)
            currentText := currentChunk.GetTextWithoutComments()
            previousText := previousChunk.GetTextWithoutComments()

            if currentText != previousText {
                // Apply usability check: skip diff if current chunk matches database metadata SDL
                if currentDBSDLChunks.shouldSkipChunkDiffForUsability(currentText, currentChunk.Identifier) {
                    continue
                }

                // Object was modified
                schemaName, objName := parseIdentifier(currentChunk.Identifier)

                // For objects that don't support ALTER, use DROP + CREATE pattern
                diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
                    Action:               schema.MetadataDiffActionDrop,
                    SchemaName:           schemaName,
                    MaterializedViewName: objName,
                    OldMaterializedView:  nil,
                    NewMaterializedView:  nil,
                    OldASTNode:           previousChunk.ASTNode,
                    NewASTNode:           nil,
                })
                diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
                    Action:               schema.MetadataDiffActionCreate,
                    SchemaName:           schemaName,
                    MaterializedViewName: objName,
                    OldMaterializedView:  nil,
                    NewMaterializedView:  nil,
                    OldASTNode:           nil,
                    NewASTNode:           currentChunk.ASTNode,
                })

                // Add COMMENT ON diffs if they exist
                if len(currentChunk.CommentStatements) > 0 {
                    for _, commentNode := range currentChunk.CommentStatements {
                        commentText := extractCommentTextFromNode(commentNode)
                        diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
                            Action:     schema.MetadataDiffActionCreate,
                            ObjectType: schema.CommentObjectTypeMaterializedView,
                            SchemaName: schemaName,
                            ObjectName: objName,
                            OldComment: "",
                            NewComment: commentText,
                            OldASTNode: nil,
                            NewASTNode: commentNode,
                        })
                    }
                }
            }
            // If text is identical, skip - comment-only changes handled in processCommentChanges
        } else {
            // New object - CREATE
            schemaName, objName := parseIdentifier(currentChunk.Identifier)
            diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
                Action:               schema.MetadataDiffActionCreate,
                SchemaName:           schemaName,
                MaterializedViewName: objName,
                OldMaterializedView:  nil,
                NewMaterializedView:  nil,
                OldASTNode:           nil,
                NewASTNode:           currentChunk.ASTNode,
            })

            // Add comments if present
            if len(currentChunk.CommentStatements) > 0 {
                for _, commentNode := range currentChunk.CommentStatements {
                    commentText := extractCommentTextFromNode(commentNode)
                    diff.CommentChanges = append(diff.CommentChanges, &schema.CommentDiff{
                        Action:     schema.MetadataDiffActionCreate,
                        ObjectType: schema.CommentObjectTypeMaterializedView,
                        SchemaName: schemaName,
                        ObjectName: objName,
                        OldComment: "",
                        NewComment: commentText,
                        OldASTNode: nil,
                        NewASTNode: commentNode,
                    })
                }
            }
        }
    }

    // Iterate through previous objects to find DROP
    for identifier, previousChunk := range previousChunks.MaterializedViews {
        if _, exists := currentChunks.MaterializedViews[identifier]; !exists {
            // Object was dropped
            schemaName, objName := parseIdentifier(identifier)
            diff.MaterializedViewChanges = append(diff.MaterializedViewChanges, &schema.MaterializedViewDiff{
                Action:               schema.MetadataDiffActionDrop,
                SchemaName:           schemaName,
                MaterializedViewName: objName,
                OldMaterializedView:  nil,
                NewMaterializedView:  nil,
                OldASTNode:           previousChunk.ASTNode,
                NewASTNode:           nil,
            })
        }
    }
}
```

**Key Points**:
- Compare text **without comments** to detect structural changes
- Use `shouldSkipChunkDiffForUsability` to avoid false positives
- For objects without ALTER support, use DROP + CREATE pattern
- Store both `OldASTNode` and `NewASTNode` for migration generation

---

### 2.4 Register in Comment-Only Changes System

Comment-only changes (where object definition is unchanged but comment is added/removed/updated) must be handled separately.

#### 2.4.1 Add to `processCommentChanges`

**File**: `backend/plugin/schema/pg/get_sdl_diff.go`

```go
func processCommentChanges(currentChunks, previousChunks *schema.SDLChunks, currentDBSDLChunks *currentDatabaseSDLChunks, diff *schema.MetadataDiff) {
    // Build sets of created and dropped objects to avoid generating comment diffs for them
    createdObjects := buildCreatedObjectsSet(diff)
    droppedObjects := buildDroppedObjectsSet(diff)

    // Process object-level comments
    processObjectComments(currentChunks.Tables, previousChunks.Tables, schema.CommentObjectTypeTable, createdObjects, droppedObjects, currentDBSDLChunks, diff)
    processObjectComments(currentChunks.Views, previousChunks.Views, schema.CommentObjectTypeView, createdObjects, droppedObjects, currentDBSDLChunks, diff)
    processObjectComments(currentChunks.MaterializedViews, previousChunks.MaterializedViews, schema.CommentObjectTypeMaterializedView, createdObjects, droppedObjects, currentDBSDLChunks, diff)  // ✅ Add
    processObjectComments(currentChunks.Functions, previousChunks.Functions, schema.CommentObjectTypeFunction, createdObjects, droppedObjects, currentDBSDLChunks, diff)
    // ...

    // Process column comments
    processColumnComments(currentChunks, previousChunks, createdObjects, droppedObjects, diff)
}
```

#### 2.4.2 Add to `buildCreatedObjectsSet`

```go
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

    // ✅ Add materialized views
    for _, mvDiff := range diff.MaterializedViewChanges {
        if mvDiff.Action == schema.MetadataDiffActionCreate {
            identifier := mvDiff.SchemaName + "." + mvDiff.MaterializedViewName
            created[identifier] = true
        }
    }

    // ... other object types ...

    return created
}
```

#### 2.4.3 Add to `buildDroppedObjectsSet`

```go
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

    // ✅ Add materialized views
    for _, mvDiff := range diff.MaterializedViewChanges {
        if mvDiff.Action == schema.MetadataDiffActionDrop {
            identifier := mvDiff.SchemaName + "." + mvDiff.MaterializedViewName
            dropped[identifier] = true
        }
    }

    // ... other object types ...

    return dropped
}
```

**Why is this needed?**
The comment processing system needs to know which objects are being created or dropped to avoid:
- Generating comment changes for newly created objects (comments are already in CREATE statement)
- Generating comment changes for dropped objects (comments are automatically removed with the object)

---

### 2.5 Implement Drift Handling (REQUIRED)

Drift handling synchronizes SDL chunks with actual database state when they differ. **This is mandatory for all object types.**

This is needed when:
- User's SDL file doesn't match database state
- Auto-sync is enabled
- Database schema differs from SDL (drift detection)

**Without this implementation, the object type will not work correctly in drift scenarios.**

**File**: `backend/plugin/schema/pg/get_sdl_diff.go`

```go
func applyMaterializedViewChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema) error {
    // Build maps of current and previous materialized views from database metadata
    currentMVs := make(map[string]*storepb.MaterializedViewMetadata)
    for _, schema := range currentSchema.Schemas {
        for _, mv := range schema.MaterializedViews {
            key := schema.Name + "." + mv.Name
            currentMVs[key] = mv
        }
    }

    previousMVs := make(map[string]*storepb.MaterializedViewMetadata)
    for _, schema := range previousSchema.Schemas {
        for _, mv := range schema.MaterializedViews {
            key := schema.Name + "." + mv.Name
            previousMVs[key] = mv
        }
    }

    // Handle additions: objects exist in DB but not in SDL chunks
    for key, mv := range currentMVs {
        if _, exists := previousMVs[key]; !exists {
            if err := createMaterializedViewChunk(previousChunks, mv); err != nil {
                return err
            }
        }
    }

    // Handle updates: objects exist in both but differ
    for key, currentMV := range currentMVs {
        if previousMV, exists := previousMVs[key]; exists {
            if err := updateMaterializedViewChunkIfNeeded(previousChunks, currentMV, previousMV); err != nil {
                return err
            }
        }
    }

    // Handle deletions: objects exist in SDL but not in DB
    for key := range previousMVs {
        if _, exists := currentMVs[key]; !exists {
            deleteMaterializedViewChunk(previousChunks, key)
        }
    }

    return nil
}
```

Then call it in `applyMinimalChangesToChunks`:

```go
func applyMinimalChangesToChunks(previousChunks *schema.SDLChunks, currentSchema, previousSchema *model.DatabaseSchema) error {
    // Process table changes
    if err := applyTableChangesToChunks(previousChunks, currentSchema, previousSchema); err != nil {
        return errors.Wrap(err, "failed to apply table changes")
    }

    // Process view changes
    if err := applyViewChangesToChunks(previousChunks, currentSchema, previousSchema); err != nil {
        return errors.Wrap(err, "failed to apply view changes")
    }

    // Process materialized view changes
    if err := applyMaterializedViewChangesToChunks(previousChunks, currentSchema, previousSchema); err != nil {  // ✅ Add
        return errors.Wrap(err, "failed to apply materialized view changes")
    }

    // ... other object types ...

    return nil
}
```

---

## Phase 3: Migration Generation

This is the **most critical phase** where dependency ordering happens.

### 3.1 Register Comment Type

**File**: `backend/plugin/schema/differ.go`

```go
type CommentObjectType string

const (
    CommentObjectTypeTable              CommentObjectType = "TABLE"
    CommentObjectTypeColumn             CommentObjectType = "COLUMN"
    CommentObjectTypeView               CommentObjectType = "VIEW"
    CommentObjectTypeMaterializedView   CommentObjectType = "MATERIALIZED VIEW"  // ✅ Add
    CommentObjectTypeFunction           CommentObjectType = "FUNCTION"
    CommentObjectTypeSequence           CommentObjectType = "SEQUENCE"
    CommentObjectTypeIndex              CommentObjectType = "INDEX"
    CommentObjectTypeSchema             CommentObjectType = "SCHEMA"
)
```

---

### 3.2 Add Objects to CREATE Topological Sort

**File**: `backend/plugin/schema/pg/generate_migration.go` - `createObjectsInOrder` function

#### 3.2.1 Add Nodes to Graph

```go
func createObjectsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) error {
    // ... create schemas, enums, sequences ...

    // Build dependency graph for all objects being created or altered
    graph := base.NewGraph()

    // Build temporary metadata for AST-only mode dependency extraction
    tempMetadata := buildTempMetadataForCreate(diff)

    // Maps to store different object types
    viewMap := make(map[string]*schema.ViewDiff)
    materializedViewMap := make(map[string]*schema.MaterializedViewDiff)
    tableMap := make(map[string]*schema.TableDiff)
    functionMap := make(map[string]*schema.FunctionDiff)

    // Track all object IDs for dependency resolution
    allObjects := make(map[string]bool)

    // Add tables to graph
    for _, tableDiff := range diff.TableChanges {
        if tableDiff.Action == schema.MetadataDiffActionCreate || tableDiff.Action == schema.MetadataDiffActionAlter {
            tableID := getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)
            graph.AddNode(tableID)
            tableMap[tableID] = tableDiff
            allObjects[tableID] = true
        }
    }

    // Add views to graph
    for _, viewDiff := range diff.ViewChanges {
        if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
            viewID := getMigrationObjectID(viewDiff.SchemaName, viewDiff.ViewName)
            graph.AddNode(viewID)
            viewMap[viewID] = viewDiff
            allObjects[viewID] = true
        }
    }

    // ✅ Add materialized views to graph
    for _, mvDiff := range diff.MaterializedViewChanges {
        if mvDiff.Action == schema.MetadataDiffActionCreate || mvDiff.Action == schema.MetadataDiffActionAlter {
            mvID := getMigrationObjectID(mvDiff.SchemaName, mvDiff.MaterializedViewName)
            graph.AddNode(mvID)
            materializedViewMap[mvID] = mvDiff
            allObjects[mvID] = true
        }
    }

    // Add functions to graph
    for _, funcDiff := range diff.FunctionChanges {
        if funcDiff.Action == schema.MetadataDiffActionCreate || funcDiff.Action == schema.MetadataDiffActionAlter {
            funcID := getMigrationObjectID(funcDiff.SchemaName, funcDiff.FunctionName)
            graph.AddNode(funcID)
            functionMap[funcID] = funcDiff
            allObjects[funcID] = true
        }
    }

    // ... continue to add dependency edges ...
}
```

#### 3.2.2 Add Dependency Edges (CRITICAL!)

This is where dependency ordering happens. **Both metadata mode and AST-only mode must be supported.**

```go
    // For tables with foreign keys depending on other tables
    for tableID, tableDiff := range tableMap {
        if tableDiff.Action == schema.MetadataDiffActionCreate {
            var foreignKeys []*storepb.ForeignKeyMetadata

            if tableDiff.NewTable != nil {
                // Metadata mode: use ForeignKeys from metadata
                foreignKeys = tableDiff.NewTable.ForeignKeys
            } else if tableDiff.NewASTNode != nil {
                // AST-only mode: extract foreign keys from AST node
                foreignKeys = extractForeignKeysFromAST(tableDiff.NewASTNode, tableDiff.SchemaName)
            }

            for _, fk := range foreignKeys {
                depID := getMigrationObjectID(fk.ReferencedSchema, fk.ReferencedTable)
                if depID != tableID {
                    // Edge from dependency to dependent (referenced table to table with FK)
                    graph.AddEdge(depID, tableID)
                }
            }
        }
    }

    // For views depending on tables/views
    for viewID, viewDiff := range viewMap {
        var dependencies []*storepb.DependencyColumn

        if viewDiff.NewView != nil {
            // Metadata mode: use metadata
            dependencies = viewDiff.NewView.DependencyColumns
        } else if viewDiff.NewASTNode != nil {
            // AST-only mode: extract dependencies from AST node
            dependencies = getViewDependenciesFromAST(viewDiff.NewASTNode, viewDiff.SchemaName, tempMetadata)
        }

        for _, dep := range dependencies {
            depID := getMigrationObjectID(dep.Schema, dep.Table)
            if allObjects[depID] {
                // Edge from dependency to dependent (table/view to view)
                graph.AddEdge(depID, viewID)
            }
        }
    }

    // ✅ For materialized views depending on tables/views
    // ⚠️ THIS IS THE MOST CRITICAL PART - both metadata and AST mode required!
    for mvID, mvDiff := range materializedViewMap {
        var dependencies []*storepb.DependencyColumn

        if mvDiff.NewMaterializedView != nil {
            // Metadata mode: use metadata
            dependencies = mvDiff.NewMaterializedView.DependencyColumns
        } else if mvDiff.NewASTNode != nil {
            // ⚠️ AST-only mode: MUST extract dependencies from AST
            // Without this, dependency ordering will be wrong in SDL mode!
            dependencies = getMaterializedViewDependenciesFromAST(mvDiff.NewASTNode, mvDiff.SchemaName, tempMetadata)
        }

        for _, dep := range dependencies {
            depID := getMigrationObjectID(dep.Schema, dep.Table)
            if allObjects[depID] {
                // Edge from dependency to dependent (table/view to materialized view)
                graph.AddEdge(depID, mvID)
            }
        }
    }

    // For functions depending on tables
    for funcID, funcDiff := range functionMap {
        if funcDiff.NewFunction != nil {
            for _, dep := range funcDiff.NewFunction.DependencyTables {
                depID := getMigrationObjectID(dep.Schema, dep.Table)
                // Edge from table to function
                graph.AddEdge(depID, funcID)
            }
        }
    }
```

**Edge Direction for CREATE**:
- `graph.AddEdge(A, B)` means "A must be created before B"
- For a view depending on a table: `graph.AddEdge(tableID, viewID)`
- For a materialized view depending on a view: `graph.AddEdge(viewID, mvID)`

#### 3.2.3 Implement AST Dependency Extraction Function

This function extracts dependencies from AST nodes in SDL mode:

```go
// getMaterializedViewDependenciesFromAST extracts table/view dependencies from a materialized view's AST node
func getMaterializedViewDependenciesFromAST(astNode any, schemaName string, _ *storepb.DatabaseSchemaMetadata) []*storepb.DependencyColumn {
    if astNode == nil {
        return []*storepb.DependencyColumn{}
    }

    var selectStatement string

    // Extract SELECT statement from the CREATE MATERIALIZED VIEW AST node
    if ctx, ok := astNode.(*pgparser.CreatematviewstmtContext); ok {
        if ctx.Selectstmt() != nil {
            // Try to get text using token stream first (most reliable)
            if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
                start := ctx.Selectstmt().GetStart()
                stop := ctx.Selectstmt().GetStop()
                if start != nil && stop != nil {
                    selectStatement = tokenStream.GetTextFromTokens(start, stop)
                }
            }

            // Fallback to token-based approach if token stream failed
            if selectStatement == "" {
                selectStatement = getTextFromAST(ctx.Selectstmt())
            }
        }
    }

    if selectStatement == "" {
        return []*storepb.DependencyColumn{}
    }

    queryStatement := strings.TrimSpace(selectStatement)

    // Use ExtractAccessTables to parse dependencies from the SELECT statement
    accessTables, err := pgpluginparser.ExtractAccessTables(queryStatement, pgpluginparser.ExtractAccessTablesOption{
        DefaultDatabase:        "",
        DefaultSchema:          schemaName,
        SkipMetadataValidation: true,  // Important: we don't have full metadata in SDL mode
    })
    if err != nil {
        return []*storepb.DependencyColumn{}
    }

    // Build dependency list
    dependencyMap := make(map[string]*storepb.DependencyColumn)
    for _, resource := range accessTables {
        // Skip system catalogs
        if resource.Schema == "pg_catalog" || resource.Schema == "information_schema" {
            continue
        }

        resourceSchema := resource.Schema
        if resourceSchema == "" {
            resourceSchema = schemaName  // Use default schema if not specified
        }

        key := fmt.Sprintf("%s.%s", resourceSchema, resource.Table)
        if _, exists := dependencyMap[key]; !exists {
            dependencyMap[key] = &storepb.DependencyColumn{
                Schema: resourceSchema,
                Table:  resource.Table,
                Column: "*",  // Table-level dependencies
            }
        }
    }

    var dependencies []*storepb.DependencyColumn
    for _, dep := range dependencyMap {
        dependencies = append(dependencies, dep)
    }

    return dependencies
}
```

**Key Points**:
- Use token stream when available (most reliable)
- Fall back to `getTextFromAST` if token stream unavailable
- Use `ExtractAccessTables` to parse SQL and find referenced tables/views
- Filter out system catalogs (`pg_catalog`, `information_schema`)
- Return table-level dependencies (column-level not needed for ordering)

#### 3.2.4 Generate SQL in Topological Order

After topological sort, iterate through objects and generate SQL:

```go
    // Get topological order
    orderedList, err := graph.TopologicalSort()
    if err != nil {
        // If there's a cycle, fall back to safe order
        // ... fallback logic ...
        return errors.Wrap(err, "failed to topologically sort objects")
    }

    // Iterate through topologically sorted objects and generate SQL
    for _, objectID := range orderedList {
        // Handle tables
        if tableDiff, exists := tableMap[objectID]; exists {
            if tableDiff.Action == schema.MetadataDiffActionCreate {
                if tableDiff.NewTable != nil {
                    // Metadata mode
                    createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable, false)
                    if err != nil {
                        return err
                    }
                    _, _ = buf.WriteString(createTableSQL)
                    // ... add comments ...
                } else if tableDiff.NewASTNode != nil {
                    // AST-only mode
                    if err := writeMigrationTableFromAST(buf, tableDiff.NewASTNode); err != nil {
                        return err
                    }
                }
            }
        }

        // Handle views
        if viewDiff, exists := viewMap[objectID]; exists {
            switch viewDiff.Action {
            case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
                if viewDiff.NewView != nil {
                    // Metadata mode
                    if err := writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView); err != nil {
                        return err
                    }
                } else if viewDiff.NewASTNode != nil {
                    // AST-only mode
                    if err := writeMigrationViewFromAST(buf, viewDiff.NewASTNode); err != nil {
                        return err
                    }
                }

                // Add comment for newly created views
                if viewDiff.NewView != nil && viewDiff.NewView.Comment != "" {
                    writeCommentOnView(buf, viewDiff.SchemaName, viewDiff.ViewName, viewDiff.NewView.Comment)
                }
            }
        }

        // ✅ Handle materialized views
        if mvDiff, exists := materializedViewMap[objectID]; exists {
            switch mvDiff.Action {
            case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
                if mvDiff.NewMaterializedView != nil {
                    // Metadata mode
                    if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
                        return err
                    }
                } else if mvDiff.NewASTNode != nil {
                    // AST-only mode: extract SQL from AST
                    if err := writeMigrationMaterializedViewFromAST(buf, mvDiff.NewASTNode); err != nil {
                        return err
                    }
                } else {
                    return errors.Errorf("materialized view diff for %s.%s has neither metadata nor AST node", mvDiff.SchemaName, mvDiff.MaterializedViewName)
                }

                // Add comment for newly created materialized views
                if mvDiff.NewMaterializedView != nil && mvDiff.NewMaterializedView.Comment != "" {
                    writeCommentOnMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName, mvDiff.NewMaterializedView.Comment)
                }
            }
        }

        // Handle functions
        if funcDiff, exists := functionMap[objectID]; exists {
            // ... similar pattern ...
        }
    }

    return nil
}
```

---

### 3.3 Add Objects to DROP Topological Sort (CRITICAL!)

DROP operations must happen in **reverse dependency order**.

**File**: `backend/plugin/schema/pg/generate_migration.go` - `dropObjectsInOrder` function

#### 3.3.1 Add Nodes to Graph

```go
func dropObjectsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) {
    // ... drop triggers first ...

    // Build dependency graph
    graph := base.NewGraph()

    // Build temporary metadata for AST-only mode dependency extraction
    tempMetadata := buildTempMetadataForDrop(diff)

    // Maps to store different object types
    viewMap := make(map[string]*schema.ViewDiff)
    materializedViewMap := make(map[string]*schema.MaterializedViewDiff)
    functionMap := make(map[string]*schema.FunctionDiff)
    tableMap := make(map[string]*schema.TableDiff)

    // Track all object IDs
    allObjects := make(map[string]bool)

    // Add views to graph
    for _, viewDiff := range diff.ViewChanges {
        if viewDiff.Action == schema.MetadataDiffActionDrop || viewDiff.Action == schema.MetadataDiffActionAlter {
            viewID := getMigrationObjectID(viewDiff.SchemaName, viewDiff.ViewName)
            graph.AddNode(viewID)
            viewMap[viewID] = viewDiff
            allObjects[viewID] = true
        }
    }

    // ✅ Add materialized views to graph
    for _, mvDiff := range diff.MaterializedViewChanges {
        if mvDiff.Action == schema.MetadataDiffActionDrop || mvDiff.Action == schema.MetadataDiffActionAlter {
            mvID := getMigrationObjectID(mvDiff.SchemaName, mvDiff.MaterializedViewName)
            graph.AddNode(mvID)
            materializedViewMap[mvID] = mvDiff
            allObjects[mvID] = true
        }
    }

    // Add functions to graph
    for _, funcDiff := range diff.FunctionChanges {
        if funcDiff.Action == schema.MetadataDiffActionDrop {
            funcID := getMigrationObjectID(funcDiff.SchemaName, funcDiff.FunctionName)
            graph.AddNode(funcID)
            functionMap[funcID] = funcDiff
            allObjects[funcID] = true
        }
    }

    // Add tables to graph
    for _, tableDiff := range diff.TableChanges {
        if tableDiff.Action == schema.MetadataDiffActionDrop {
            tableID := getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)
            graph.AddNode(tableID)
            tableMap[tableID] = tableDiff
            allObjects[tableID] = true
        }
    }

    // ... continue to add dependency edges ...
}
```

#### 3.3.2 Add Dependency Edges (REVERSE Direction!)

**Edge Direction for DROP**:
- `graph.AddEdge(A, B)` means "A must be dropped before B"
- For a view depending on a table: `graph.AddEdge(viewID, tableID)` (reverse!)
- For a materialized view depending on a view: `graph.AddEdge(mvID, viewID)` (reverse!)

```go
    // For views depending on tables/views
    for viewID, viewDiff := range viewMap {
        var dependencies []*storepb.DependencyColumn

        if viewDiff.OldView != nil {
            // Metadata mode: use metadata
            dependencies = viewDiff.OldView.DependencyColumns
        } else if viewDiff.OldASTNode != nil {
            // AST-only mode: extract dependencies from AST node
            dependencies = getViewDependenciesFromAST(viewDiff.OldASTNode, viewDiff.SchemaName, tempMetadata)
        }

        for _, dep := range dependencies {
            depID := getMigrationObjectID(dep.Schema, dep.Table)
            if allObjects[depID] {
                // ⚠️ Edge from dependent to dependency (view depends on table/view)
                // For DROP: view -> table means view should be dropped before table
                graph.AddEdge(viewID, depID)
            }
        }
    }

    // ✅ For materialized views depending on tables/views
    // ⚠️ CRITICAL: Must support both metadata and AST mode for DROP too!
    for mvID, mvDiff := range materializedViewMap {
        var dependencies []*storepb.DependencyColumn

        if mvDiff.OldMaterializedView != nil {
            // Metadata mode: use metadata
            dependencies = mvDiff.OldMaterializedView.DependencyColumns
        } else if mvDiff.OldASTNode != nil {
            // ⚠️ AST-only mode: extract dependencies from AST
            // Without this, "cannot drop because other objects depend on it" error!
            dependencies = getMaterializedViewDependenciesFromAST(mvDiff.OldASTNode, mvDiff.SchemaName, tempMetadata)
        }

        for _, dep := range dependencies {
            depID := getMigrationObjectID(dep.Schema, dep.Table)
            if allObjects[depID] {
                // ⚠️ Edge from dependent to dependency (reverse for DROP!)
                // mv -> view/table means mv should be dropped before view/table
                graph.AddEdge(mvID, depID)
            }
        }
    }

    // For functions depending on tables
    for funcID, funcDiff := range functionMap {
        if funcDiff.OldFunction != nil {
            for _, dep := range funcDiff.OldFunction.DependencyTables {
                depID := getMigrationObjectID(dep.Schema, dep.Table)
                if allObjects[depID] {
                    // Edge from function to table (function dropped before table)
                    graph.AddEdge(funcID, depID)
                }
            }
        }
    }

    // For tables with foreign keys
    for tableID, tableDiff := range tableMap {
        var foreignKeys []*storepb.ForeignKeyMetadata

        if tableDiff.OldTable != nil {
            // Metadata mode: use ForeignKeys from metadata
            foreignKeys = tableDiff.OldTable.ForeignKeys
        } else if tableDiff.OldASTNode != nil {
            // AST-only mode: extract foreign keys from AST node
            foreignKeys = extractForeignKeysFromAST(tableDiff.OldASTNode, tableDiff.SchemaName)
        }

        for _, fk := range foreignKeys {
            depID := getMigrationObjectID(fk.ReferencedSchema, fk.ReferencedTable)
            if allObjects[depID] && depID != tableID {
                // Edge from table with FK to referenced table
                // For DROP: table1 (with FK) -> table2 (referenced)
                // This ensures table1 is dropped before table2
                graph.AddEdge(tableID, depID)
            }
        }
    }
```

---

### 3.4 Implement AST to SQL Conversion

**File**: `backend/plugin/schema/pg/generate_migration.go`

```go
func writeMigrationMaterializedViewFromAST(buf *strings.Builder, astNode any) error {
    if astNode == nil {
        return errors.Errorf("AST node is nil")
    }

    ctx, ok := astNode.(*pgparser.CreatematviewstmtContext)
    if !ok {
        return errors.Errorf("unexpected AST node type: %T", astNode)
    }

    // Get the full SQL text from token stream
    var sqlText string
    if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
        start := ctx.GetStart()
        stop := ctx.GetStop()
        if start != nil && stop != nil {
            sqlText = tokenStream.GetTextFromTokens(start, stop)
        }
    }

    // Fallback to text extraction if token stream unavailable
    if sqlText == "" {
        sqlText = getTextFromAST(ctx)
    }

    if sqlText == "" {
        return errors.Errorf("failed to extract SQL text from AST")
    }

    // Ensure it ends with semicolon
    sqlText = strings.TrimSpace(sqlText)
    if !strings.HasSuffix(sqlText, ";") {
        sqlText += ";"
    }

    _, err := buf.WriteString(sqlText)
    if err != nil {
        return err
    }
    _, err = buf.WriteString("\n")
    return err
}
```

---

### 3.5 Add Comment Generation

**File**: `backend/plugin/schema/pg/generate_migration.go` - `generateCommentChangesFromSDL` function

Add a case for the new object type:

```go
func generateCommentChangesFromSDL(buf *strings.Builder, diff *schema.MetadataDiff) error {
    if len(diff.CommentChanges) == 0 {
        return nil
    }

    // ... build sets of tables/columns being dropped ...

    for _, commentDiff := range diff.CommentChanges {
        // Extract the new comment text from the AST node
        newComment := extractCommentFromDiff(commentDiff)

        switch commentDiff.ObjectType {
        case schema.CommentObjectTypeSchema:
            writeCommentOnSchema(buf, commentDiff.SchemaName, newComment)

        case schema.CommentObjectTypeTable:
            // ... handle table comments ...
            writeCommentOnTable(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

        case schema.CommentObjectTypeColumn:
            // ... handle column comments ...
            writeCommentOnColumn(buf, commentDiff.SchemaName, commentDiff.ObjectName, commentDiff.ColumnName, newComment)

        case schema.CommentObjectTypeView:
            writeCommentOnView(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

        case schema.CommentObjectTypeMaterializedView:  // ✅ Add case
            writeCommentOnMaterializedView(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

        case schema.CommentObjectTypeFunction:
            // ... handle function comments ...
            writeCommentOnFunction(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment, functionASTNode, functionDefinition)

        case schema.CommentObjectTypeSequence:
            writeCommentOnSequence(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

        case schema.CommentObjectTypeIndex:
            // ... handle index comments ...
        }
    }

    return nil
}
```

---

### 3.6 Implement Comment Write Function

```go
func writeCommentOnMaterializedView(buf *strings.Builder, schemaName, mvName, comment string) {
    if comment == "" {
        // Remove comment with IS NULL
        _, _ = buf.WriteString(fmt.Sprintf("COMMENT ON MATERIALIZED VIEW \"%s\".\"%s\" IS NULL;\n", schemaName, mvName))
    } else {
        // Add/update comment
        escapedComment := strings.ReplaceAll(comment, "'", "''")
        _, _ = buf.WriteString(fmt.Sprintf("COMMENT ON MATERIALIZED VIEW \"%s\".\"%s\" IS '%s';\n", schemaName, mvName, escapedComment))
    }
}
```

**Important**:
- Properly escape single quotes in comments (`'` → `''`)
- Use `IS NULL` to remove comments
- Use quoted identifiers for schema and object names

---

## Phase 4: Database Definition (SDL Output)

### 4.1 Single-file SDL Format

**File**: `backend/plugin/schema/pg/get_database_definition.go` - `getSDLFormat` function

Add the new object type to single-file SDL generation:

```go
func getSDLFormat(dbSchema *model.DatabaseSchema, schemaVersion string) (string, error) {
    var buf bytes.Buffer

    // ... write schemas ...

    for _, schema := range dbSchema.Schemas {
        // ... write tables ...

        // Write views
        for _, view := range schema.Views {
            if view.SkipDump {
                continue
            }

            if err := writeViewSDL(&buf, schema.Name, view); err != nil {
                return "", err
            }

            if _, err := buf.WriteString(";\n\n"); err != nil {
                return "", err
            }

            // Write view comment if present
            if len(view.Comment) > 0 {
                if err := writeViewCommentSDL(&buf, schema.Name, view); err != nil {
                    return "", err
                }
            }
        }

        // ✅ Write materialized views after views
        for _, materializedView := range schema.MaterializedViews {
            if materializedView.SkipDump {
                continue
            }

            if err := writeMaterializedViewSDL(&buf, schema.Name, materializedView); err != nil {
                return "", err
            }

            if _, err := buf.WriteString(";\n\n"); err != nil {
                return "", err
            }

            // Write materialized view comment if present
            if len(materializedView.Comment) > 0 {
                if err := writeMaterializedViewCommentSDL(&buf, schema.Name, materializedView); err != nil {
                    return "", err
                }
            }
        }

        // ... write functions, sequences, etc ...
    }

    return buf.String(), nil
}
```

---

### 4.2 Multi-file SDL Format

**File**: `backend/plugin/schema/pg/get_database_definition.go` - `GetMultiFileDatabaseDefinition` function

```go
func GetMultiFileDatabaseDefinition(dbSchema *model.DatabaseSchema, schemaVersion string) ([]schema.File, error) {
    var files []schema.File

    for _, schemaMetadata := range dbSchema.Schemas {
        schemaName := schemaMetadata.Name

        // ... generate table files ...

        // ... generate view files ...

        // ✅ Generate materialized view files
        for _, materializedView := range schemaMetadata.MaterializedViews {
            if materializedView.SkipDump {
                continue
            }

            var buf strings.Builder
            if err := writeMaterializedViewSDL(&buf, schemaName, materializedView); err != nil {
                return nil, errors.Wrapf(err, "failed to generate materialized view SDL for %s.%s", schemaName, materializedView.Name)
            }
            buf.WriteString(";\n")

            // Write materialized view comment if present
            if len(materializedView.Comment) > 0 {
                buf.WriteString("\n")
                if err := writeMaterializedViewCommentSDL(&buf, schemaName, materializedView); err != nil {
                    return nil, errors.Wrapf(err, "failed to generate materialized view comment for %s.%s", schemaName, materializedView.Name)
                }
            }

            files = append(files, schema.File{
                Name:    fmt.Sprintf("schemas/%s/materialized_views/%s.sql", schemaName, materializedView.Name),
                Content: buf.String(),
            })
        }

        // ... generate function files, sequence files, etc ...
    }

    return files, nil
}
```

**File naming convention**:
- Tables: `schemas/{schema}/tables/{name}.sql`
- Views: `schemas/{schema}/views/{name}.sql`
- Materialized Views: `schemas/{schema}/materialized_views/{name}.sql`
- Functions: `schemas/{schema}/functions/{name}.sql`

---

### 4.3 Implement SDL Write Functions

**File**: `backend/plugin/schema/pg/get_database_definition.go`

#### Write Object SDL

```go
// writeMaterializedViewSDL writes the SDL (simple) version
// No indexes, no comments, no WITH NO DATA - just the core CREATE statement
func writeMaterializedViewSDL(out io.Writer, schemaName string, mv *storepb.MaterializedViewMetadata) error {
    if _, err := io.WriteString(out, `CREATE MATERIALIZED VIEW "`); err != nil {
        return err
    }
    if _, err := io.WriteString(out, schemaName); err != nil {
        return err
    }
    if _, err := io.WriteString(out, `"."`); err != nil {
        return err
    }
    if _, err := io.WriteString(out, mv.Name); err != nil {
        return err
    }
    if _, err := io.WriteString(out, `" AS`); err != nil {
        return err
    }
    if _, err := io.WriteString(out, "\n"); err != nil {
        return err
    }

    definition := strings.TrimSpace(mv.Definition)
    // Remove trailing semicolon if present (will be added by caller)
    definition = strings.TrimSuffix(definition, ";")

    if _, err := io.WriteString(out, definition); err != nil {
        return err
    }

    return nil
}
```

**Key Points**:
- SDL version is **simple** - only core CREATE statement
- No extra clauses (e.g., no `WITH NO DATA` for materialized views)
- No indexes, no comments in the body
- Trailing semicolon is added by caller

#### Write Comment SDL

```go
// nolint:unused
func writeMaterializedViewCommentSDL(out io.Writer, schemaName string, mv *storepb.MaterializedViewMetadata) error {
    if len(mv.Comment) == 0 {
        return nil
    }

    escapedComment := strings.ReplaceAll(mv.Comment, "'", "''")
    if _, err := io.WriteString(out, fmt.Sprintf("COMMENT ON MATERIALIZED VIEW \"%s\".\"%s\" IS '%s';\n", schemaName, mv.Name, escapedComment)); err != nil {
        return err
    }

    return nil
}
```

**Note**: The `// nolint:unused` comment is needed if the function is only called conditionally (when comment is not empty).

---

## Phase 5: Testing

### 5.1 SDL Diff Basic Tests

**File**: `backend/plugin/schema/pg/{object}_sdl_diff_test.go`

```go
func TestMaterializedViewSDLDiff(t *testing.T) {
    tests := []struct {
        name                            string
        previousSDL                     string
        currentSDL                      string
        expectedMaterializedViewChanges int
        expectedActions                 []schema.MetadataDiffAction
    }{
        {
            name:        "Create new materialized view",
            previousSDL: ``,
            currentSDL: `
                CREATE TABLE users (id INT, name TEXT);
                CREATE MATERIALIZED VIEW user_mv AS SELECT * FROM users;
            `,
            expectedMaterializedViewChanges: 1,
            expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
        },
        {
            name: "Drop materialized view",
            previousSDL: `
                CREATE TABLE users (id INT, name TEXT);
                CREATE MATERIALIZED VIEW user_mv AS SELECT * FROM users;
            `,
            currentSDL:                      `CREATE TABLE users (id INT, name TEXT);`,
            expectedMaterializedViewChanges: 1,
            expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionDrop},
        },
        {
            name: "Modify materialized view (drop and recreate)",
            previousSDL: `
                CREATE TABLE users (id INT, name TEXT);
                CREATE MATERIALIZED VIEW user_mv AS SELECT id FROM users;
            `,
            currentSDL: `
                CREATE TABLE users (id INT, name TEXT);
                CREATE MATERIALIZED VIEW user_mv AS SELECT id, name FROM users;
            `,
            expectedMaterializedViewChanges: 2, // Drop + Create
            expectedActions:                 []schema.MetadataDiffAction{schema.MetadataDiffActionDrop, schema.MetadataDiffActionCreate},
        },
        {
            name: "No changes to materialized view",
            previousSDL: `
                CREATE TABLE users (id INT, name TEXT);
                CREATE MATERIALIZED VIEW user_mv AS SELECT * FROM users;
            `,
            currentSDL: `
                CREATE TABLE users (id INT, name TEXT);
                CREATE MATERIALIZED VIEW user_mv AS SELECT * FROM users;
            `,
            expectedMaterializedViewChanges: 0,
            expectedActions:                 []schema.MetadataDiffAction{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
            require.NoError(t, err)
            require.NotNil(t, diff)

            assert.Equal(t, tt.expectedMaterializedViewChanges, len(diff.MaterializedViewChanges),
                "Expected %d materialized view changes, got %d", tt.expectedMaterializedViewChanges, len(diff.MaterializedViewChanges))

            // Check that the actions match expectations
            var actualActions []schema.MetadataDiffAction
            for _, mvDiff := range diff.MaterializedViewChanges {
                actualActions = append(actualActions, mvDiff.Action)
            }

            assert.ElementsMatch(t, tt.expectedActions, actualActions,
                "Expected actions %v, got %v", tt.expectedActions, actualActions)

            // Verify AST nodes are properly set
            for i, mvDiff := range diff.MaterializedViewChanges {
                switch mvDiff.Action {
                case schema.MetadataDiffActionCreate:
                    assert.NotNil(t, mvDiff.NewASTNode,
                        "Materialized view diff %d should have NewASTNode for CREATE action", i)
                    assert.Nil(t, mvDiff.OldASTNode,
                        "Materialized view diff %d should not have OldASTNode for CREATE action", i)
                case schema.MetadataDiffActionDrop:
                    assert.NotNil(t, mvDiff.OldASTNode,
                        "Materialized view diff %d should have OldASTNode for DROP action", i)
                    assert.Nil(t, mvDiff.NewASTNode,
                        "Materialized view diff %d should not have NewASTNode for DROP action", i)
                default:
                    t.Errorf("Unexpected action %v for materialized view diff %d", mvDiff.Action, i)
                }
            }
        })
    }
}
```

---

### 5.2 Comment Parsing Tests

```go
func TestMaterializedViewWithCommentParsing(t *testing.T) {
    tests := []struct {
        name               string
        sdl                string
        expectedMVCount    int
        expectedComment    string
        expectCommentInSDL bool
    }{
        {
            name: "Create materialized view with comment",
            sdl: `
                CREATE TABLE public.users (
                    id SERIAL PRIMARY KEY,
                    name VARCHAR(255) NOT NULL
                );

                CREATE MATERIALIZED VIEW public.user_summary_mv AS
                SELECT id, name FROM users;

                COMMENT ON MATERIALIZED VIEW public.user_summary_mv IS 'Summary of all users';
            `,
            expectedMVCount:    1,
            expectedComment:    "Summary of all users",
            expectCommentInSDL: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Parse the SDL to get chunks
            chunks, err := ChunkSDLText(tt.sdl)
            require.NoError(t, err)
            require.NotNil(t, chunks)

            // Verify we parsed the correct number of materialized views
            assert.Equal(t, tt.expectedMVCount, len(chunks.MaterializedViews),
                "Expected %d materialized view(s), got %d", tt.expectedMVCount, len(chunks.MaterializedViews))

            // Get the materialized view chunk
            var mvChunk *schema.SDLChunk
            for _, chunk := range chunks.MaterializedViews {
                mvChunk = chunk
                break
            }
            require.NotNil(t, mvChunk, "Materialized view chunk should exist")

            // Get the full text including comments
            fullText := mvChunk.GetText()
            t.Logf("Full materialized view text:\n%s", fullText)

            // Verify comment is included in the SDL
            if tt.expectCommentInSDL {
                assert.Contains(t, fullText, "COMMENT ON MATERIALIZED VIEW",
                    "Full text should contain COMMENT ON MATERIALIZED VIEW statement")
                assert.Contains(t, fullText, tt.expectedComment,
                    "Comment should contain the expected text: %s", tt.expectedComment)

                // Verify comment statements count
                assert.Greater(t, len(mvChunk.CommentStatements), 0,
                    "Materialized view should have comment statements")
            }

            // Verify CREATE MATERIALIZED VIEW is present
            assert.Contains(t, fullText, "CREATE MATERIALIZED VIEW",
                "Full text should contain CREATE MATERIALIZED VIEW statement")
        })
    }
}
```

---

### 5.3 Comment Migration Generation Tests

```go
func TestMaterializedViewCommentMigrationGeneration(t *testing.T) {
    tests := []struct {
        name        string
        previousSDL string
        currentSDL  string
        wantComment bool
    }{
        {
            name:        "Create materialized view with comment generates COMMENT statement",
            previousSDL: ``,
            currentSDL: `
                CREATE TABLE products (id SERIAL PRIMARY KEY, name VARCHAR(255));
                CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
                COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
            `,
            wantComment: true,
        },
        {
            name: "Add comment to existing materialized view",
            previousSDL: `
                CREATE TABLE products (id SERIAL PRIMARY KEY, name VARCHAR(255));
                CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
            `,
            currentSDL: `
                CREATE TABLE products (id SERIAL PRIMARY KEY, name VARCHAR(255));
                CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
                COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
            `,
            wantComment: true,
        },
        {
            name: "Remove comment from materialized view",
            previousSDL: `
                CREATE TABLE products (id SERIAL PRIMARY KEY, name VARCHAR(255));
                CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
                COMMENT ON MATERIALIZED VIEW product_mv IS 'Product summary view';
            `,
            currentSDL: `
                CREATE TABLE products (id SERIAL PRIMARY KEY, name VARCHAR(255));
                CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
            `,
            wantComment: true, // Should generate COMMENT statement with IS NULL
        },
        {
            name: "Update comment on materialized view",
            previousSDL: `
                CREATE TABLE products (id SERIAL PRIMARY KEY, name VARCHAR(255));
                CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
                COMMENT ON MATERIALIZED VIEW product_mv IS 'Old comment';
            `,
            currentSDL: `
                CREATE TABLE products (id SERIAL PRIMARY KEY, name VARCHAR(255));
                CREATE MATERIALIZED VIEW product_mv AS SELECT id, name FROM products;
                COMMENT ON MATERIALIZED VIEW product_mv IS 'New comment';
            `,
            wantComment: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
            require.NoError(t, err)
            require.NotNil(t, diff)

            migration, err := generateMigration(diff)
            require.NoError(t, err)

            t.Logf("Generated migration:\n%s", migration)

            if tt.wantComment {
                assert.Contains(t, migration, "COMMENT ON MATERIALIZED VIEW",
                    "Expected migration to contain COMMENT ON MATERIALIZED VIEW statement")
            }
        })
    }
}
```

---

### 5.4 Dependency Order Tests (MOST IMPORTANT!)

This is the **most critical test** - it verifies that topological sort works correctly in SDL mode.

```go
func TestMaterializedViewDependencyOrder(t *testing.T) {
    tests := []struct {
        name        string
        previousSDL string
        currentSDL  string
        description string
    }{
        {
            name:        "CREATE with dependencies: tables -> view -> materialized view",
            previousSDL: ``,
            currentSDL: `
                CREATE TABLE customers (
                    customer_id SERIAL PRIMARY KEY,
                    name VARCHAR(255) NOT NULL,
                    email VARCHAR(255),
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                );

                CREATE TABLE orders (
                    order_id SERIAL PRIMARY KEY,
                    customer_id INTEGER REFERENCES customers(customer_id),
                    amount DECIMAL(10,2),
                    order_date DATE
                );

                -- View depends on tables
                CREATE VIEW customer_stats_view AS
                SELECT
                    c.customer_id,
                    c.name,
                    c.email,
                    COUNT(o.order_id) as order_count,
                    SUM(o.amount) as total_spent
                FROM customers c
                LEFT JOIN orders o ON c.customer_id = o.customer_id
                GROUP BY c.customer_id, c.name, c.email;

                -- Materialized view depends on the view above
                CREATE MATERIALIZED VIEW customer_segmentation_mv AS
                SELECT
                    csv.customer_id,
                    csv.name,
                    csv.total_spent,
                    CASE
                        WHEN csv.total_spent >= 1000 THEN 'Premium'
                        WHEN csv.total_spent >= 500 THEN 'Standard'
                        ELSE 'Basic'
                    END as segment
                FROM customer_stats_view csv;
            `,
            description: "Tests that objects are created in correct dependency order: tables -> view -> materialized view",
        },
        {
            name: "DROP with dependencies: materialized view -> view -> tables",
            previousSDL: `
                CREATE TABLE customers (
                    customer_id SERIAL PRIMARY KEY,
                    name VARCHAR(255) NOT NULL,
                    email VARCHAR(255),
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                );

                CREATE TABLE orders (
                    order_id SERIAL PRIMARY KEY,
                    customer_id INTEGER REFERENCES customers(customer_id),
                    amount DECIMAL(10,2),
                    order_date DATE
                );

                CREATE VIEW customer_stats_view AS
                SELECT
                    c.customer_id,
                    c.name,
                    c.email,
                    COUNT(o.order_id) as order_count,
                    SUM(o.amount) as total_spent
                FROM customers c
                LEFT JOIN orders o ON c.customer_id = o.customer_id
                GROUP BY c.customer_id, c.name, c.email;

                CREATE MATERIALIZED VIEW customer_segmentation_mv AS
                SELECT
                    csv.customer_id,
                    csv.name,
                    csv.total_spent,
                    CASE
                        WHEN csv.total_spent >= 1000 THEN 'Premium'
                        WHEN csv.total_spent >= 500 THEN 'Standard'
                        ELSE 'Basic'
                    END as segment
                FROM customer_stats_view csv;
            `,
            currentSDL:  ``,
            description: "Tests that objects are dropped in correct dependency order: materialized view -> view -> tables",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
            require.NoError(t, err)
            require.NotNil(t, diff)

            migration, err := generateMigration(diff)
            require.NoError(t, err)

            t.Logf("Generated migration:\n%s", migration)

            // Check if this is a CREATE or DROP test
            if tt.currentSDL != "" {
                // CREATE test: verify correct order for creation
                // Tables should come before views, and views should come before materialized views
                customersIdx := strings.Index(migration, "CREATE TABLE customers")
                ordersIdx := strings.Index(migration, "CREATE TABLE orders")
                viewIdx := strings.Index(migration, "CREATE VIEW customer_stats_view")
                mvIdx := strings.Index(migration, "CREATE MATERIALIZED VIEW customer_segmentation_mv")

                assert.NotEqual(t, -1, customersIdx, "customers table should be created")
                assert.NotEqual(t, -1, ordersIdx, "orders table should be created")
                assert.NotEqual(t, -1, viewIdx, "customer_stats_view should be created")
                assert.NotEqual(t, -1, mvIdx, "customer_segmentation_mv should be created")

                // Verify correct order for CREATE
                if customersIdx != -1 && viewIdx != -1 {
                    assert.Less(t, customersIdx, viewIdx,
                        "customers table must be created before customer_stats_view")
                }
                if ordersIdx != -1 && viewIdx != -1 {
                    assert.Less(t, ordersIdx, viewIdx,
                        "orders table must be created before customer_stats_view")
                }
                if viewIdx != -1 && mvIdx != -1 {
                    assert.Less(t, viewIdx, mvIdx,
                        "customer_stats_view must be created before customer_segmentation_mv")
                }
            } else {
                // DROP test: verify correct order for dropping
                // Materialized views should be dropped before views, and views before tables
                mvIdx := strings.Index(migration, "DROP MATERIALIZED VIEW")
                viewIdx := strings.Index(migration, "DROP VIEW")
                customersIdx := strings.Index(migration, "DROP TABLE")

                assert.NotEqual(t, -1, mvIdx, "customer_segmentation_mv should be dropped")
                assert.NotEqual(t, -1, viewIdx, "customer_stats_view should be dropped")
                assert.NotEqual(t, -1, customersIdx, "tables should be dropped")

                // Verify correct order for DROP (reverse of CREATE)
                if mvIdx != -1 && viewIdx != -1 {
                    assert.Less(t, mvIdx, viewIdx,
                        "customer_segmentation_mv must be dropped before customer_stats_view")
                }
                if viewIdx != -1 && customersIdx != -1 {
                    assert.Less(t, viewIdx, customersIdx,
                        "customer_stats_view must be dropped before tables")
                }
            }
        })
    }
}
```

**What this test verifies**:
- ✅ CREATE order: dependencies created first (table → view → materialized view)
- ✅ DROP order: dependents dropped first (materialized view → view → table)
- ✅ AST dependency extraction works correctly
- ✅ No "relation does not exist" errors
- ✅ No "cannot drop because other objects depend on it" errors

---

## Phase 6: Integration Points (CRITICAL - Often Forgotten!)

This phase covers integration points that are **not** in the main implementation flow but are **critical** for production usage.

### 6.1 Archive Schema Filter (CRITICAL!)

**File**: `backend/plugin/schema/differ.go` - `FilterPostgresArchiveSchema` function

The `FilterPostgresArchiveSchema` function filters out objects from the `bbdataarchive` schema. This is called in **every** `bb rollout` execution.

**⚠️ CRITICAL**: If you forget to add your object type here, it will be correctly detected in diff but **filtered out during execution**!

#### For Database-level Objects (Extensions, Events)

Database-level objects are stored directly in `DatabaseSchemaMetadata` (not in `SchemaMetadata`). They don't have a schema name, so they should be **copied directly** without filtering:

```go
// Extensions and Events are database-level objects, not schema-specific, so copy them all
filtered.ExtensionChanges = diff.ExtensionChanges
filtered.EventChanges = diff.EventChanges
```

#### For Schema-scoped Objects (Tables, Views, Functions, etc.)

Schema-scoped objects must be **filtered by schema name**:

```go
// Filter table changes
for _, tableChange := range diff.TableChanges {
    if tableChange.SchemaName != archiveSchemaName {
        filtered.TableChanges = append(filtered.TableChanges, tableChange)
    }
}
```

#### Example Implementation

**Before** (missing ExtensionChanges):
```go
func FilterPostgresArchiveSchema(diff *MetadataDiff) *MetadataDiff {
    // ... filter schema-scoped objects ...

    // Events are database-level objects, not schema-specific, so copy them all
    filtered.EventChanges = diff.EventChanges

    return filtered  // ❌ ExtensionChanges are lost!
}
```

**After** (correct):
```go
func FilterPostgresArchiveSchema(diff *MetadataDiff) *MetadataDiff {
    // ... filter schema-scoped objects ...

    // Extensions and Events are database-level objects, not schema-specific, so copy them all
    filtered.ExtensionChanges = diff.ExtensionChanges  // ✅ Added
    filtered.EventChanges = diff.EventChanges

    return filtered
}
```

#### Symptoms of Forgetting This

- ✅ SDL parsing works correctly
- ✅ Diff calculation detects changes correctly
- ✅ All unit tests pass
- ❌ **bb rollout execution silently drops the object**
- ❌ Dependent objects fail with "relation does not exist" errors

**Real-world example**: Extension implementation forgot this, causing:
```
ERROR: type "citext" does not exist (SQLSTATE 42704)
```
Even though the SDL file contained `CREATE EXTENSION "citext"` and the diff correctly detected it!

#### Test for This

Always add a filter test to catch this issue:

```go
func TestXXXNotFilteredByArchiveSchemaFilter(t *testing.T) {
    diff := &schema.MetadataDiff{
        ExtensionChanges: []*schema.ExtensionDiff{
            {
                Action:        schema.MetadataDiffActionCreate,
                ExtensionName: "test_extension",
            },
        },
        TableChanges: []*schema.TableDiff{
            {
                Action:     schema.MetadataDiffActionCreate,
                SchemaName: "bbdataarchive",  // Should be filtered
                TableName:  "archive_table",
            },
        },
    }

    filtered := schema.FilterPostgresArchiveSchema(diff)

    // Extension should be preserved (database-level)
    require.Equal(t, 1, len(filtered.ExtensionChanges))

    // Archive schema table should be filtered out
    require.Equal(t, 0, len(filtered.TableChanges))
}
```

---

## Critical Points & Common Pitfalls

### 🚨 Most Common Mistakes

#### 1. Forgetting AST-only Mode for Dependencies

**❌ Wrong** (only supports metadata mode):
```go
for mvID, mvDiff := range materializedViewMap {
    if mvDiff.NewMaterializedView != nil {
        for _, dep := range mvDiff.NewMaterializedView.DependencyColumns {
            depID := getMigrationObjectID(dep.Schema, dep.Table)
            graph.AddEdge(depID, mvID)
        }
    }
}
```

**✅ Correct** (supports both modes):
```go
for mvID, mvDiff := range materializedViewMap {
    var dependencies []*storepb.DependencyColumn

    if mvDiff.NewMaterializedView != nil {
        dependencies = mvDiff.NewMaterializedView.DependencyColumns
    } else if mvDiff.NewASTNode != nil {
        // ⚠️ MUST support AST mode for SDL!
        dependencies = getMaterializedViewDependenciesFromAST(mvDiff.NewASTNode, mvDiff.SchemaName, tempMetadata)
    }

    for _, dep := range dependencies {
        depID := getMigrationObjectID(dep.Schema, dep.Table)
        if allObjects[depID] {
            graph.AddEdge(depID, mvID)
        }
    }
}
```

#### 2. Forgetting DROP Dependency Extraction

CREATE and DROP **both** need AST-only mode support. Many implementations forget DROP.

**Symptom**: CREATE works fine, but DROP fails with "cannot drop because other objects depend on it"

**Fix**: Add AST-only mode support in `dropObjectsInOrder` just like in `createObjectsInOrder`

#### 3. Wrong Edge Direction

**For CREATE**: `graph.AddEdge(dependency, dependent)`
- Example: `graph.AddEdge(tableID, viewID)` means "create table before view"

**For DROP**: `graph.AddEdge(dependent, dependency)` (reversed!)
- Example: `graph.AddEdge(viewID, tableID)` means "drop view before table"

#### 4. Not Registering in Comment System

Forgetting to add the object to:
- `processCommentChanges`
- `buildCreatedObjectsSet`
- `buildDroppedObjectsSet`

**Symptom**: Comment-only changes are not detected

#### 5. Not Handling COMMENT Statement Variants

PostgreSQL parser may return "MATERIALIZED VIEW" or "MATERIALIZEDVIEW" (no space).

**Fix**: Always handle both variants:
```go
case "MATERIALIZED VIEW", "MATERIALIZEDVIEW":
    // handle comment
```

#### 6. Forgetting to Update Archive Schema Filter (CRITICAL!)

**The #1 mistake that breaks production but passes all tests!**

**Problem**: Adding the object to all processing functions but forgetting `FilterPostgresArchiveSchema`.

**Symptom**:
- ✅ All tests pass
- ✅ SDL parsing works
- ✅ Diff detection works
- ❌ **bb rollout silently drops the object during execution**

**Why it happens**: The filter function is **not** in the normal implementation flow. It's an integration point that only gets called during actual `bb rollout` execution.

**Fix**: Always check and update both patterns:

**For database-level objects** (Extensions, Events):
```go
// In FilterPostgresArchiveSchema:
filtered.ExtensionChanges = diff.ExtensionChanges  // Direct copy
filtered.EventChanges = diff.EventChanges
```

**For schema-scoped objects** (Tables, Views, etc.):
```go
// In FilterPostgresArchiveSchema:
for _, tableChange := range diff.TableChanges {
    if tableChange.SchemaName != archiveSchemaName {
        filtered.TableChanges = append(filtered.TableChanges, tableChange)
    }
}
```

**Prevention**: Always add a `TestXXXNotFilteredByArchiveSchemaFilter` test (see Phase 6.1).

---

### ✅ Verification Checklist

Use this checklist to ensure complete implementation:

#### Parser & Schema
- [ ] Object metadata structure defined with `DependencyColumns` field
- [ ] Object registered in `SDLChunks` structure
- [ ] AST listener implemented (`EnterXXXstmt`)
- [ ] COMMENT listener case added (with variant handling)
- [ ] Maps initialized in `ChunkSDLText`

#### SDL Diff
- [ ] Object change processor implemented (`processXXXChanges`)
- [ ] Change processor called in `GetSDLDiff`
- [ ] Detects CREATE/DROP/MODIFY correctly
- [ ] Comment diffs added for object changes
- [ ] Registered in `processCommentChanges`
- [ ] Registered in `buildCreatedObjectsSet`
- [ ] Registered in `buildDroppedObjectsSet`
- [ ] **Drift handling implemented (`applyXXXChangesToChunks`) - REQUIRED**
- [ ] **Drift handler called in `applyMinimalChangesToChunks` - REQUIRED**

#### Migration Generation - CREATE
- [ ] Comment type registered in `differ.go`
- [ ] Objects added to graph in `createObjectsInOrder`
- [ ] Dependency edges added **with AST-only mode support**
- [ ] Dependency extraction function implemented (`getXXXDependenciesFromAST`)
- [ ] SQL generation in topological order (both metadata and AST modes)
- [ ] Comment generation for new objects
- [ ] AST to SQL conversion function implemented

#### Migration Generation - DROP
- [ ] Objects added to graph in `dropObjectsInOrder`
- [ ] Dependency edges added **with AST-only mode support** (reversed!)
- [ ] Uses same dependency extraction function as CREATE
- [ ] DROP SQL generation in topological order

#### Migration Generation - Comments
- [ ] Case added in `generateCommentChangesFromSDL`
- [ ] Comment write function implemented
- [ ] Handles both add and remove (IS NULL)
- [ ] Properly escapes quotes

#### SDL Output
- [ ] Object write function implemented (`writeXXXSDL`)
- [ ] Comment write function implemented (`writeXXXCommentSDL`)
- [ ] Integrated into single-file format (`getSDLFormat`)
- [ ] Integrated into multi-file format (`GetMultiFileDatabaseDefinition`)
- [ ] Correct file path for multi-file format

#### Testing
- [ ] Basic SDL diff test (CREATE/DROP/MODIFY/NO CHANGE)
- [ ] Comment parsing test
- [ ] Comment migration generation test (add/remove/update)
- [ ] **Dependency order test (CREATE)**
- [ ] **Dependency order test (DROP)**
- [ ] All tests pass
- [ ] Linter passes

#### Integration Points (CRITICAL!)
- [ ] **Added to `FilterPostgresArchiveSchema` (database-level: direct copy; schema-scoped: filter by schema)**
- [ ] **Filter test added (`TestXXXNotFilteredByArchiveSchemaFilter`)**

---

## Summary

Adding a new database object to SDL mode requires changes in **6 major phases**:

1. **Parser**: Extract AST nodes and associate COMMENT statements
2. **SDL Diff**: Detect CREATE/DROP/MODIFY and comment-only changes + **Drift handling (REQUIRED)**
3. **Migration Generation**: Add to topological sort with **AST-only dependency extraction** (both CREATE and DROP!)
4. **SDL Output**: Generate single-file and multi-file SDL formats
5. **Testing**: Comprehensive tests including dependency ordering
6. **Integration Points**: Update `FilterPostgresArchiveSchema` and add filter test

The **most critical** and **most often forgotten** parts are:

### 1. AST-only mode dependency extraction (CRITICAL!)
> **AST-only mode dependency extraction for both CREATE and DROP operations**

Without this, migrations will have wrong ordering in `bb rollout` (SDL mode), leading to:
- ❌ "Relation does not exist" errors on CREATE
- ❌ "Cannot drop because other objects depend on it" errors on DROP

Always implement and test dependency ordering with the `TestXXXDependencyOrder` test!

### 2. Drift handling implementation (REQUIRED!)
> **`applyXXXChangesToChunks` function and call in `applyMinimalChangesToChunks`**

Without this, the object type will not work correctly when:
- User's SDL file doesn't match database state
- Auto-sync is enabled
- Database schema differs from SDL (drift detection)

All other object types (tables, views, functions, sequences, materialized views) implement drift handling. Your new object type must too.

### 3. Archive schema filter (MOST CRITICAL - Breaks production!)
> **Add object to `FilterPostgresArchiveSchema` in `differ.go`**

This is **the #1 mistake** that:
- ✅ Passes all unit tests
- ✅ Works correctly in SDL parsing and diff calculation
- ❌ **Silently fails in production `bb rollout`**

Without this:
- The object will be correctly detected in diff
- But **filtered out** before migration execution
- Causing "relation does not exist" errors for dependent objects

**Prevention**:
- For database-level objects: `filtered.ExtensionChanges = diff.ExtensionChanges`
- For schema-scoped objects: Filter by `schemaName != archiveSchemaName`
- Always add `TestXXXNotFilteredByArchiveSchemaFilter` test

See **Phase 6.1** for detailed explanation and real-world example.
