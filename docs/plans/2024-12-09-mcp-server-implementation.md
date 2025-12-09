# MCP Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an embedded MCP server to Bytebase that exposes all proto service methods as MCP tools.

**Architecture:** Use official MCP Go SDK with `StreamableHTTPHandler` at `/mcp` route. Tool calls route through internal ConnectRPC client so existing auth/ACL/audit interceptors apply. Proto reflection discovers services at startup; pre-generated JSON schemas provide tool input validation.

**Tech Stack:** `github.com/modelcontextprotocol/go-sdk/mcp`, ConnectRPC, proto reflection, buf protoschema-jsonschema plugin

**Design Doc:** `docs/plans/2024-12-08-mcp-server-design.md`

---

## Task 1: Add JSON Schema Generation Plugin

**Files:**
- Modify: `proto/buf.gen.yaml`

**Step 1: Add protoschema-jsonschema plugin to buf.gen.yaml**

```yaml
# Add after the last plugin entry (before inputs section)
  - remote: buf.build/bufbuild/protoschema-jsonschema
    out: gen/jsonschema
    opt:
      - disallow_additional_properties=true
```

**Step 2: Generate JSON schemas**

Run:
```bash
cd proto && buf generate
```

Expected: New `proto/gen/jsonschema/` directory with JSON schema files for each proto message.

**Step 3: Verify generation**

Run:
```bash
ls proto/gen/jsonschema/ | head -20
```

Expected: Files like `bytebase.v1.GetDatabaseRequest.json`, `bytebase.v1.ListDatabasesRequest.json`, etc.

**Step 4: Commit**

```bash
but commit default -m "feat(mcp): add protoschema-jsonschema plugin for MCP tool schemas"
```

---

## Task 2: Add MCP SDK Dependency

**Files:**
- Modify: `backend/go.mod`

**Step 1: Add MCP SDK dependency**

Run:
```bash
cd /Users/p0ny/bytebase && go get github.com/modelcontextprotocol/go-sdk@latest
```

Expected: `go.mod` updated with new dependency.

**Step 2: Tidy modules**

Run:
```bash
go mod tidy
```

Expected: `go.sum` updated.

**Step 3: Verify import works**

Run:
```bash
go list -m github.com/modelcontextprotocol/go-sdk
```

Expected: Shows version like `github.com/modelcontextprotocol/go-sdk v0.x.x`

**Step 4: Commit**

```bash
but commit default -m "feat(mcp): add official MCP Go SDK dependency"
```

---

## Task 3: Create MCP Server Core

**Files:**
- Create: `backend/api/mcp/server.go`

**Step 1: Create the mcp package directory**

Run:
```bash
mkdir -p backend/api/mcp
```

**Step 2: Write server.go**

```go
// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server and provides HTTP handler integration.
type Server struct {
	mcpServer *mcp.Server
	handler   http.Handler
}

// NewServer creates a new MCP server with all Bytebase tools registered.
func NewServer(registry *Registry) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "bytebase",
		Version: "1.0.0",
	}, nil)

	// Register all tools from the registry
	if err := registry.RegisterTools(mcpServer); err != nil {
		return nil, err
	}

	handler := mcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcp.Server {
			return mcpServer
		},
		nil,
	)

	return &Server{
		mcpServer: mcpServer,
		handler:   handler,
	}, nil
}

// Handler returns the HTTP handler for the MCP server.
func (s *Server) Handler() http.Handler {
	return s.handler
}
```

**Step 3: Verify it compiles**

Run:
```bash
go build ./backend/api/mcp/...
```

Expected: Build succeeds (may have unused import warnings, that's OK for now).

**Step 4: Commit**

```bash
but commit default -m "feat(mcp): add MCP server core with SDK integration"
```

---

## Task 4: Create Tool Registry with Proto Reflection

**Files:**
- Create: `backend/api/mcp/registry.go`

**Step 1: Write registry.go**

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Registry holds MCP tool definitions derived from proto services.
type Registry struct {
	schemaDir string
	schemas   map[string]json.RawMessage // proto message name -> JSON schema
	invoker   *Invoker
}

// NewRegistry creates a new tool registry.
func NewRegistry(schemaDir string, invoker *Invoker) (*Registry, error) {
	r := &Registry{
		schemaDir: schemaDir,
		schemas:   make(map[string]json.RawMessage),
		invoker:   invoker,
	}

	if err := r.loadSchemas(); err != nil {
		return nil, fmt.Errorf("failed to load JSON schemas: %w", err)
	}

	return r, nil
}

// loadSchemas loads all JSON schemas from the schema directory.
func (r *Registry) loadSchemas() error {
	entries, err := os.ReadDir(r.schemaDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(r.schemaDir, entry.Name()))
		if err != nil {
			return err
		}

		// Schema filename format: bytebase.v1.MessageName.json
		name := strings.TrimSuffix(entry.Name(), ".json")
		r.schemas[name] = json.RawMessage(data)
	}

	return nil
}

// getServices returns all v1 service descriptors to register as tools.
func (r *Registry) getServices() []protoreflect.ServiceDescriptor {
	var services []protoreflect.ServiceDescriptor

	// Get file descriptors from the generated v1 package
	// Each service is registered in the global registry
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		pkg := string(fd.Package())
		if pkg != "bytebase.v1" {
			return true
		}

		for i := 0; i < fd.Services().Len(); i++ {
			services = append(services, fd.Services().Get(i))
		}
		return true
	})

	return services
}

// RegisterTools registers all proto service methods as MCP tools.
func (r *Registry) RegisterTools(server *mcp.Server) error {
	services := r.getServices()

	for _, svc := range services {
		svcName := string(svc.Name())

		for i := 0; i < svc.Methods().Len(); i++ {
			method := svc.Methods().Get(i)
			methodName := string(method.Name())
			toolName := fmt.Sprintf("%s_%s", svcName, methodName)

			inputName := string(method.Input().FullName())
			schema, ok := r.schemas[inputName]
			if !ok {
				// Skip methods without schemas (shouldn't happen)
				continue
			}

			// Extract description from proto comments if available
			description := r.extractDescription(method)

			server.AddTool(&mcp.Tool{
				Name:        toolName,
				Description: description,
				InputSchema: schema,
			}, r.createHandler(svc, method))
		}
	}

	return nil
}

// extractDescription extracts the method description from proto comments.
func (r *Registry) extractDescription(method protoreflect.MethodDescriptor) string {
	// Proto comments are available via source info, but for simplicity
	// we'll use the method's full name as description for now
	return fmt.Sprintf("Call %s.%s", method.Parent().Name(), method.Name())
}

// createHandler creates an MCP tool handler for a proto method.
func (r *Registry) createHandler(svc protoreflect.ServiceDescriptor, method protoreflect.MethodDescriptor) mcp.ToolHandlerFunc {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Invoke the gRPC method via the invoker
		result, err := r.invoker.Invoke(ctx, svc, method, req.Arguments)
		if err != nil {
			return nil, err
		}

		// Marshal result to JSON for MCP response
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultJSON),
				},
			},
		}, nil
	}
}

// Ensure v1pb is imported (triggers proto registration).
var _ = v1pb.File_v1_database_service_proto
```

**Step 2: Verify it compiles**

Run:
```bash
go build ./backend/api/mcp/...
```

Expected: Build succeeds.

**Step 3: Commit**

```bash
but commit default -m "feat(mcp): add tool registry with proto reflection"
```

---

## Task 5: Create gRPC Invoker

**Files:**
- Create: `backend/api/mcp/invoker.go`

**Step 1: Write invoker.go**

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Invoker handles invoking gRPC methods via the internal ConnectRPC client.
type Invoker struct {
	httpClient  *http.Client
	baseURL     string
	authHeader  string // Will be set per-request from MCP context
}

// NewInvoker creates a new gRPC method invoker.
func NewInvoker(baseURL string) *Invoker {
	return &Invoker{
		httpClient: &http.Client{},
		baseURL:    baseURL,
	}
}

// Invoke calls a gRPC method with the given arguments.
func (i *Invoker) Invoke(
	ctx context.Context,
	svc protoreflect.ServiceDescriptor,
	method protoreflect.MethodDescriptor,
	args json.RawMessage,
) (proto.Message, error) {
	// Create request message dynamically
	reqType, err := protoregistry.GlobalTypes.FindMessageByName(method.Input().FullName())
	if err != nil {
		return nil, fmt.Errorf("failed to find request type %s: %w", method.Input().FullName(), err)
	}

	reqMsg := reqType.New().Interface()

	// Unmarshal JSON args into proto message
	if len(args) > 0 {
		if err := protojson.Unmarshal(args, reqMsg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %w", err)
		}
	}

	// Build the ConnectRPC endpoint URL
	// Format: /bytebase.v1.ServiceName/MethodName
	procedure := fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())
	url := i.baseURL + procedure

	// Create response message dynamically
	respType, err := protoregistry.GlobalTypes.FindMessageByName(method.Output().FullName())
	if err != nil {
		return nil, fmt.Errorf("failed to find response type %s: %w", method.Output().FullName(), err)
	}

	respMsg := respType.New().Interface()

	// Make the ConnectRPC call
	if err := i.doConnect(ctx, url, reqMsg, respMsg); err != nil {
		return nil, err
	}

	return respMsg, nil
}

// doConnect performs a ConnectRPC unary call.
func (i *Invoker) doConnect(ctx context.Context, url string, req, resp proto.Message) error {
	// Serialize request
	reqBody, err := protojson.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Forward auth header if present in context
	if auth := ctx.Value(authHeaderKey{}); auth != nil {
		httpReq.Header.Set("Authorization", auth.(string))
	}

	// Make request
	httpResp, err := i.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Check for errors
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d", httpResp.StatusCode)
	}

	// Deserialize response
	var buf strings.Builder
	if _, err := buf.ReadFrom(httpResp.Body); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := protojson.Unmarshal([]byte(buf.String()), resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// authHeaderKey is the context key for the auth header.
type authHeaderKey struct{}

// WithAuthHeader returns a context with the auth header set.
func WithAuthHeader(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, authHeaderKey{}, auth)
}
```

**Step 2: Verify it compiles**

Run:
```bash
go build ./backend/api/mcp/...
```

Expected: Build succeeds.

**Step 3: Commit**

```bash
but commit default -m "feat(mcp): add gRPC invoker for tool execution"
```

---

## Task 6: Create MCP HTTP Handler Wrapper

**Files:**
- Modify: `backend/api/mcp/server.go`

**Step 1: Update server.go to extract auth from request**

Replace the entire `server.go` with:

```go
// Package mcp provides an MCP (Model Context Protocol) server for Bytebase.
package mcp

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server and provides HTTP handler integration.
type Server struct {
	mcpServer *mcp.Server
	registry  *Registry
}

// NewServer creates a new MCP server with all Bytebase tools registered.
func NewServer(registry *Registry) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "bytebase",
		Version: "1.0.0",
	}, nil)

	// Register all tools from the registry
	if err := registry.RegisterTools(mcpServer); err != nil {
		return nil, err
	}

	return &Server{
		mcpServer: mcpServer,
		registry:  registry,
	}, nil
}

// Handler returns the HTTP handler for the MCP server.
// It wraps the SDK handler to inject auth context.
func (s *Server) Handler() http.Handler {
	sdkHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			// Extract auth header and store in invoker context
			auth := r.Header.Get("Authorization")
			if auth != "" {
				// Store in request context for invoker to use
				ctx := WithAuthHeader(r.Context(), auth)
				*r = *r.WithContext(ctx)
			}
			return s.mcpServer
		},
		nil,
	)

	return sdkHandler
}
```

**Step 2: Verify it compiles**

Run:
```bash
go build ./backend/api/mcp/...
```

Expected: Build succeeds.

**Step 3: Commit**

```bash
but commit default -m "feat(mcp): add auth header extraction to MCP handler"
```

---

## Task 7: Register MCP Route in Echo

**Files:**
- Modify: `backend/server/echo_routes.go`

**Step 1: Read current echo_routes.go to find the right place**

Run:
```bash
head -100 backend/server/echo_routes.go
```

**Step 2: Add MCP server initialization and route**

Add import:
```go
mcpserver "github.com/bytebase/bytebase/backend/api/mcp"
```

Add to `configureEchoRouters` function parameters:
```go
mcpServer *mcpserver.Server,
```

Add route after the `/lsp` route:
```go
// MCP server endpoint.
e.Any("/mcp", echo.WrapHandler(mcpServer.Handler()))
e.Any("/mcp/*", echo.WrapHandler(mcpServer.Handler()))
```

**Step 3: Verify it compiles**

Run:
```bash
go build ./backend/server/...
```

Expected: Build fails because we need to update server.go to pass mcpServer. That's expected.

**Step 4: Commit partial progress**

```bash
but commit default -m "feat(mcp): add /mcp route to Echo server"
```

---

## Task 8: Initialize MCP Server in Main Server

**Files:**
- Modify: `backend/server/server.go`

**Step 1: Read current server initialization**

Run:
```bash
grep -n "lspServer" backend/server/server.go | head -20
```

**Step 2: Add MCP server initialization**

Add import:
```go
mcpserver "github.com/bytebase/bytebase/backend/api/mcp"
```

Add MCP server creation after other server initializations:
```go
// Initialize MCP server.
mcpInvoker := mcpserver.NewInvoker(fmt.Sprintf("http://localhost:%d", profile.Port))
mcpRegistry, err := mcpserver.NewRegistry("proto/gen/jsonschema", mcpInvoker)
if err != nil {
    return nil, fmt.Errorf("failed to create MCP registry: %w", err)
}
mcpServer, err := mcpserver.NewServer(mcpRegistry)
if err != nil {
    return nil, fmt.Errorf("failed to create MCP server: %w", err)
}
```

Update `configureEchoRouters` call to pass `mcpServer`.

**Step 3: Verify it compiles**

Run:
```bash
go build ./backend/server/...
```

Expected: Build succeeds.

**Step 4: Run linter**

Run:
```bash
golangci-lint run --allow-parallel-runners ./backend/api/mcp/... ./backend/server/...
```

Expected: No errors (or fix any that appear).

**Step 5: Commit**

```bash
but commit default -m "feat(mcp): initialize MCP server in main server"
```

---

## Task 9: Add Basic Tests

**Files:**
- Create: `backend/api/mcp/server_test.go`

**Step 1: Write basic tests**

```go
package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewInvoker(t *testing.T) {
	invoker := NewInvoker("http://localhost:8080")
	require.NotNil(t, invoker)
	require.Equal(t, "http://localhost:8080", invoker.baseURL)
}

func TestWithAuthHeader(t *testing.T) {
	ctx := context.Background()
	ctx = WithAuthHeader(ctx, "Bearer test-token")

	auth := ctx.Value(authHeaderKey{})
	require.Equal(t, "Bearer test-token", auth)
}
```

**Step 2: Run tests**

Run:
```bash
go test -v ./backend/api/mcp/...
```

Expected: Tests pass.

**Step 3: Commit**

```bash
but commit default -m "test(mcp): add basic unit tests for MCP server"
```

---

## Task 10: Build and Manual Test

**Step 1: Build the full binary**

Run:
```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: Build succeeds.

**Step 2: Start the server**

Run:
```bash
PG_URL=postgresql://bbdev@localhost/bbdev ./bytebase-build/bytebase --port 8080 --data . --debug
```

**Step 3: Test MCP endpoint**

Run (in another terminal):
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

Expected: JSON response with server info and capabilities.

**Step 4: Test tools/list**

Run:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

Expected: JSON response listing all available tools (DatabaseService_ListDatabases, etc.).

**Step 5: Commit final integration**

```bash
but commit default -m "feat(mcp): complete MCP server integration"
```

---

## Summary

After completing all tasks:

1. **JSON schemas generated** from proto definitions via buf plugin
2. **MCP SDK integrated** with StreamableHTTPHandler
3. **Tool registry** discovers all proto services via reflection
4. **Invoker** routes tool calls through internal ConnectRPC client
5. **Auth flows** through to existing interceptors
6. **/mcp route** registered in Echo server
7. **Tests** verify basic functionality

The MCP server exposes all Bytebase API methods as tools, with auth/ACL/audit handled by existing infrastructure.
