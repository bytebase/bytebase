# MCP Schema Lookup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add schema parameter to search_api for looking up message types, and truncate verbose protobuf descriptions.

**Architecture:** Extend search_api with a new `schema` parameter that looks up component schemas from the OpenAPI spec. Add a map of known protobuf types with short descriptions to replace verbose documentation.

**Tech Stack:** Go, libopenapi, MCP SDK

---

## Task 1: Add GetSchema method to OpenAPIIndex

**Files:**
- Modify: `backend/api/mcp/openapi_index.go:465` (end of file)
- Test: `backend/api/mcp/openapi_index_test.go` (create if needed)

**Step 1: Write the failing test**

Add to `backend/api/mcp/tool_search_test.go`:

```go
func TestSearchAPISchemaLookup(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test schema lookup with full name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "bytebase.v1.Instance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "bytebase.v1.Instance")
	require.Contains(t, text, "name:")
	require.Contains(t, text, "engine:")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v github.com/bytebase/bytebase/backend/api/mcp -run ^TestSearchAPISchemaLookup$`
Expected: FAIL - SearchInput has no Schema field

**Step 3: Add Schema field to SearchInput**

In `backend/api/mcp/tool_search.go`, add after line 23:

```go
// SearchInput is the input for the search_api tool.
type SearchInput struct {
	// OperationID gets detailed schema for a specific endpoint.
	// Use this after finding the endpoint you need.
	OperationID string `json:"operationId,omitempty"`
	// Schema gets the definition of a message type.
	// Examples: "bytebase.v1.Instance", "Instance", "Engine"
	Schema string `json:"schema,omitempty"`
	// Query is a free-text search query to find relevant API endpoints.
	// Examples: "create database", "execute sql", "list projects"
	Query string `json:"query,omitempty"`
	// Service filters results to a specific service.
	// Examples: "SQLService", "DatabaseService", "ProjectService"
	Service string `json:"service,omitempty"`
	// Limit is the maximum number of results to return (default: 5, max: 50).
	Limit int `json:"limit,omitempty"`
}
```

**Step 4: Add GetSchema method to OpenAPIIndex**

In `backend/api/mcp/openapi_index.go`, add at end of file:

```go
// GetSchema returns the schema properties for a component schema by name.
// Supports both full name (bytebase.v1.Instance) and short name (Instance).
func (idx *OpenAPIIndex) GetSchema(name string) ([]PropertyInfo, bool) {
	if idx.doc.Model.Components == nil || idx.doc.Model.Components.Schemas == nil {
		return nil, false
	}

	// Try exact name first
	if props := idx.getSchemaByName(name); props != nil {
		return props, true
	}

	// Try with bytebase.v1. prefix
	if !strings.HasPrefix(name, "bytebase.v1.") {
		fullName := "bytebase.v1." + name
		if props := idx.getSchemaByName(fullName); props != nil {
			return props, true
		}
	}

	return nil, false
}

func (idx *OpenAPIIndex) getSchemaByName(name string) []PropertyInfo {
	schemaProxy, ok := idx.doc.Model.Components.Schemas.Get(name)
	if !ok || schemaProxy == nil {
		return nil
	}

	schema := schemaProxy.Schema()
	if schema == nil {
		return nil
	}

	// Handle enum types
	if len(schema.Enum) > 0 {
		var enumValues []string
		for _, v := range schema.Enum {
			if s, ok := v.Value.(string); ok {
				enumValues = append(enumValues, s)
			}
		}
		return []PropertyInfo{{
			Name:        "enum",
			Type:        "string",
			Description: strings.Join(enumValues, ", "),
		}}
	}

	if schema.Properties == nil {
		return nil
	}

	var props []PropertyInfo
	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
		propName := pair.Key()
		prop := pair.Value()

		propType := "object"
		if prop != nil {
			propSchema := prop.Schema()
			if propSchema != nil && len(propSchema.Type) > 0 {
				propType = propSchema.Type[0]
			}
			if prop.GetReference() != "" {
				refParts := strings.Split(prop.GetReference(), "/")
				propType = refParts[len(refParts)-1]
			}
		}

		desc := ""
		if prop != nil && prop.Schema() != nil {
			desc = prop.Schema().Description
		}

		props = append(props, PropertyInfo{
			Name:        propName,
			Type:        propType,
			Description: desc,
			Required:    requiredSet[propName],
		})
	}

	slices.SortFunc(props, func(a, b PropertyInfo) int {
		return strings.Compare(a.Name, b.Name)
	})

	return props
}
```

**Step 5: Add schema case to handleSearchAPI**

In `backend/api/mcp/tool_search.go`, modify `handleSearchAPI` to add schema case after operationID case (around line 52):

```go
func (s *Server) handleSearchAPI(_ context.Context, _ *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, any, error) {
	var text string

	switch {
	case input.OperationID != "":
		// Detail mode: get full schema for a specific endpoint
		text = s.formatEndpointDetail(input.OperationID)

	case input.Schema != "":
		// Schema lookup mode: get properties of a message type
		text = s.formatSchemaDetail(input.Schema)

	case input.Query == "" && input.Service == "":
		// ... rest unchanged
```

**Step 6: Add formatSchemaDetail method**

In `backend/api/mcp/tool_search.go`, add after `formatEndpointDetail`:

```go
func (s *Server) formatSchemaDetail(schemaName string) string {
	props, ok := s.openAPIIndex.GetSchema(schemaName)
	if !ok {
		return fmt.Sprintf("Unknown schema: %s\n\nUse search_api(operationId=\"...\") to see schemas in request/response bodies.", schemaName)
	}

	var sb strings.Builder

	// Normalize name for display
	displayName := schemaName
	if !strings.HasPrefix(schemaName, "bytebase.v1.") {
		displayName = "bytebase.v1." + schemaName
	}

	sb.WriteString(fmt.Sprintf("## %s\n\n", displayName))

	// Check if it's an enum
	if len(props) == 1 && props[0].Name == "enum" {
		sb.WriteString("**Enum values:** ")
		sb.WriteString(props[0].Description)
		sb.WriteString("\n")
		return sb.String()
	}

	for _, prop := range props {
		s.formatProperty(&sb, prop)
	}

	return sb.String()
}
```

**Step 7: Run test to verify it passes**

Run: `go test -v github.com/bytebase/bytebase/backend/api/mcp -run ^TestSearchAPISchemaLookup$`
Expected: PASS

**Step 8: Commit**

```bash
but commit <branch> -m "feat(mcp): add schema lookup to search_api"
```

---

## Task 2: Add short name schema lookup test

**Files:**
- Modify: `backend/api/mcp/tool_search_test.go`

**Step 1: Write the test**

Add to `backend/api/mcp/tool_search_test.go`:

```go
func TestSearchAPISchemaLookupShortName(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test schema lookup with short name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "Instance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "bytebase.v1.Instance")
	require.Contains(t, text, "name:")
}

func TestSearchAPISchemaLookupNotFound(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test schema lookup with unknown name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "NonExistentSchema",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Unknown schema")
}

func TestSearchAPISchemaLookupEnum(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test enum schema lookup
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "Engine",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Enum values:")
}
```

**Step 2: Run tests**

Run: `go test -v github.com/bytebase/bytebase/backend/api/mcp -run ^TestSearchAPISchemaLookup`
Expected: PASS

**Step 3: Commit**

```bash
but commit <branch> -m "test(mcp): add schema lookup tests"
```

---

## Task 3: Add protobuf type description truncation

**Files:**
- Modify: `backend/api/mcp/openapi_index.go`
- Modify: `backend/api/mcp/tool_search.go:184-203` (formatProperty)

**Step 1: Write the failing test**

Add to `backend/api/mcp/tool_search_test.go`:

```go
func TestSearchAPIProtobufDescriptionTruncation(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test that protobuf types have short descriptions
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "InstanceService/CreateInstance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	text := result.Content[0].(*mcpsdk.TextContent).Text

	// Should NOT contain verbose protobuf documentation
	require.NotContains(t, text, "A Timestamp represents a point in time")
	require.NotContains(t, text, "A Duration represents a signed")

	// Should contain short description
	if strings.Contains(text, "google.protobuf.Timestamp") {
		require.Contains(t, text, "ISO 8601")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v github.com/bytebase/bytebase/backend/api/mcp -run ^TestSearchAPIProtobufDescriptionTruncation$`
Expected: FAIL - contains verbose protobuf docs

**Step 3: Add typeDescriptions map**

In `backend/api/mcp/openapi_index.go`, add after imports (around line 14):

```go
// typeDescriptions provides concise descriptions for known types.
// These replace verbose protobuf documentation.
var typeDescriptions = map[string]string{
	"google.protobuf.Timestamp": `ISO 8601 format, e.g. "2024-01-15T01:30:15Z"`,
	"google.protobuf.Duration":  `e.g. "3.5s" or "1h30m"`,
	"google.protobuf.FieldMask": `e.g. "title,engine"`,
	"google.protobuf.Empty":     "empty message",
	"google.protobuf.Any":       "any JSON value",
	"google.protobuf.Struct":    "JSON object",
	"google.protobuf.Value":     "any JSON value",
}

// GetTypeDescription returns a concise description for known types.
func GetTypeDescription(typeName string) (string, bool) {
	desc, ok := typeDescriptions[typeName]
	return desc, ok
}
```

**Step 4: Update formatProperty to use short descriptions**

In `backend/api/mcp/tool_search.go`, replace `formatProperty` method:

```go
func (*Server) formatProperty(sb *strings.Builder, prop PropertyInfo) {
	required := ""
	if prop.Required {
		required = " (required)"
	}

	desc := ""
	// Check if type has a known short description
	if shortDesc, ok := GetTypeDescription(prop.Type); ok {
		desc = fmt.Sprintf(" // %s", shortDesc)
	} else if prop.Description != "" {
		// Remove newlines and truncate long descriptions
		cleanDesc := strings.ReplaceAll(prop.Description, "\n", " ")
		cleanDesc = strings.ReplaceAll(cleanDesc, "\r", "")
		// Truncate at 100 chars
		if len(cleanDesc) > 100 {
			cleanDesc = cleanDesc[:97] + "..."
		}
		desc = fmt.Sprintf(" // %s", cleanDesc)
	}

	sb.WriteString("  \"")
	sb.WriteString(prop.Name)
	sb.WriteString("\": ")
	sb.WriteString(prop.Type)
	sb.WriteString(required)
	sb.WriteString(desc)
	sb.WriteString("\n")
}
```

**Step 5: Run test to verify it passes**

Run: `go test -v github.com/bytebase/bytebase/backend/api/mcp -run ^TestSearchAPIProtobufDescriptionTruncation$`
Expected: PASS

**Step 6: Run all tests**

Run: `go test -v github.com/bytebase/bytebase/backend/api/mcp`
Expected: All PASS

**Step 7: Commit**

```bash
but commit <branch> -m "feat(mcp): truncate verbose protobuf descriptions"
```

---

## Task 4: Update tool description

**Files:**
- Modify: `backend/api/mcp/tool_search.go:27-37`

**Step 1: Update searchAPIDescription**

Replace the `searchAPIDescription` constant:

```go
const searchAPIDescription = `Discover Bytebase API endpoints. **Always call before call_api - never guess schemas.**

| Mode | Parameters | Result |
|------|------------|--------|
| List | (none) | All services |
| Browse | service="SQLService" | All endpoints in service |
| Search | query="database" | Matching endpoints |
| Filter | service+query | Search within service |
| Details | operationId="SQLService/Query" | Request/response schema |
| Schema | schema="Instance" | Message type definition |

**Workflow:** search_api() → search_api(operationId="...") → call_api(...)`
```

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners ./backend/api/mcp/...`
Expected: No errors

**Step 3: Commit**

```bash
but commit <branch> -m "docs(mcp): update search_api description with schema mode"
```

---

## Task 5: Manual verification

**Step 1: Build and test manually**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

**Step 2: Test via MCP**

Start Bytebase and test:
- `search_api(schema="Instance")` - should show Instance fields
- `search_api(schema="bytebase.v1.Instance")` - same result
- `search_api(schema="Engine")` - should show enum values
- `search_api(operationId="InstanceService/CreateInstance")` - should have short protobuf descriptions

**Step 3: Final commit if needed**

```bash
but commit <branch> -m "feat(mcp): schema lookup and description truncation"
```

---

Plan complete and saved to `docs/plans/2025-12-15-mcp-schema-lookup.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
