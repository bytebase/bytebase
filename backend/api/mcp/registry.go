package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Registry holds MCP tool definitions derived from proto services.
type Registry struct {
	schemas map[string]json.RawMessage // proto message name -> JSON schema
	invoker *Invoker
}

// NewRegistry creates a new tool registry.
func NewRegistry(invoker *Invoker) (*Registry, error) {
	r := &Registry{
		schemas: make(map[string]json.RawMessage),
		invoker: invoker,
	}

	if err := r.loadSchemas(); err != nil {
		return nil, errors.Wrap(err, "failed to load JSON schemas")
	}

	return r, nil
}

// loadSchemas loads all JSON schemas from embedded files.
func (r *Registry) loadSchemas() error {
	entries, err := embeddedSchemas.ReadDir("schemas")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonschema.json") {
			continue
		}

		data, err := embeddedSchemas.ReadFile("schemas/" + entry.Name())
		if err != nil {
			return err
		}

		// Schema filename format: bytebase.v1.MessageName.jsonschema.json
		// Extract just bytebase.v1.MessageName part
		name := strings.TrimSuffix(entry.Name(), ".jsonschema.json")
		r.schemas[name] = json.RawMessage(data)
	}

	return nil
}

// getServices returns all v1 service descriptors to register as tools.
func (*Registry) getServices() []protoreflect.ServiceDescriptor {
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
func (*Registry) extractDescription(method protoreflect.MethodDescriptor) string {
	// Proto comments are available via source info, but for simplicity
	// we'll use the method's full name as description for now
	return fmt.Sprintf("Call %s.%s", method.Parent().Name(), method.Name())
}

// createHandler creates an MCP tool handler for a proto method.
func (r *Registry) createHandler(svc protoreflect.ServiceDescriptor, method protoreflect.MethodDescriptor) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Invoke the gRPC method via the invoker
		result, err := r.invoker.Invoke(ctx, svc, method, req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		// Marshal result to JSON for MCP response
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal result")
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: string(resultJSON),
				},
			},
		}, nil
	}
}

// Ensure v1pb is imported (triggers proto registration).
var _ = v1pb.File_v1_database_service_proto
