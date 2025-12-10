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
	"gopkg.in/yaml.v3"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Registry holds MCP tool definitions derived from proto services.
type Registry struct {
	openAPIDoc *OpenAPIDoc
	invoker    *Invoker
}

// OpenAPIDoc matches the structure of the generated openapi.yaml
type OpenAPIDoc struct {
	Components struct {
		Schemas map[string]any `yaml:"schemas"`
	} `yaml:"components"`
}

// NewRegistry creates a new tool registry.
func NewRegistry(invoker *Invoker) (*Registry, error) {
	r := &Registry{
		invoker: invoker,
	}

	if err := r.loadOpenAPISchemas(); err != nil {
		return nil, errors.Wrap(err, "failed to load OpenAPI schemas")
	}

	return r, nil
}

// loadOpenAPISchemas parses the embedded OpenAPI spec.
func (r *Registry) loadOpenAPISchemas() error {
	data, err := embeddedSchemas.ReadFile("spec/openapi.yaml")
	if err != nil {
		return err
	}

	var doc OpenAPIDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	r.openAPIDoc = &doc
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
			// Proto full name is like "bytebase.v1.BatchGetIamPolicyRequest"
			// OpenAPI schemas are flat, like "BatchGetIamPolicyRequest"
			// We strip the package prefix to find the matching schema.
			schemaName := strings.TrimPrefix(inputName, "bytebase.v1.")

			schema, err := r.resolveSchema(schemaName)
			if err != nil {
				// Warn but continue? For now, we assume schema must exist if it's in v1.
				// However, some empty requests might not be in OpenAPI if not used in HTTP body.
				// But gnostic-openapi usually generates them.
				// Let's log error or skip?
				// Just skip if not found.
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

// resolveSchema returns a self-contained JSON schema for the given definition name.
// It crawls dependencies in the OpenAPI doc and includes them in $defs.
func (r *Registry) resolveSchema(rootDef string) (json.RawMessage, error) {
	// Verify root schema exists
	if _, ok := r.openAPIDoc.Components.Schemas[rootDef]; !ok {
		return nil, errors.Errorf("schema %s not found", rootDef)
	}

	// Collected definitions to include
	defs := make(map[string]any)
	visited := make(map[string]bool)

	var crawl func(name string) error
	crawl = func(name string) error {
		if visited[name] {
			return nil
		}
		visited[name] = true

		s, ok := r.openAPIDoc.Components.Schemas[name]
		if !ok {
			return errors.Errorf("dependent schema %s not found", name)
		}

		// Deep copy schema to avoid modifying the global doc in place
		// We use JSON roundtrip for simplicity and to ensure standard map[string]any types
		sJSON, err := json.Marshal(s)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal schema %s", name)
		}
		var clone any
		if err := json.Unmarshal(sJSON, &clone); err != nil {
			return errors.Wrapf(err, "failed to unmarshal schema clone %s", name)
		}

		defs[name] = clone

		// Find $refs in this schema and recurse
		return r.walkRefs(clone, func(refName string) error {
			return crawl(refName)
		})
	}

	if err := crawl(rootDef); err != nil {
		return nil, err
	}

	// Construct the final schema
	// We want to inline the root schema properties so that MCP clients can see them at the top level.
	// We copy the root schema fields to finalSchema, but use the collected defs.

	// Convert root clone to map
	rootMap, ok := defs[rootDef].(map[string]any)
	if !ok {
		// Should not happen as we unmarshaled into interface{}
		return nil, errors.Errorf("root definition %s is not a map", rootDef)
	}

	// Remove rootDef from defs since we are inlining it
	delete(defs, rootDef)

	finalSchema := make(map[string]any)
	// Copy all fields from root definition
	for k, v := range rootMap {
		finalSchema[k] = v
	}

	// Ensure $schema version matches (override if present or add)
	finalSchema["$schema"] = "http://json-schema.org/draft-07/schema#"
	// Ensure type is object (though it should be from root)
	finalSchema["type"] = "object"
	// Add other definitions
	if len(defs) > 0 {
		finalSchema["$defs"] = defs
	}

	return json.Marshal(finalSchema)
}

// walkRefs validates and walks all $ref occurrences in the schema.
func (r *Registry) walkRefs(v any, fn func(refName string) error) error {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if k == "$ref" {
				if refStr, ok := val.(string); ok {
					// OpenAPI refs are '#/components/schemas/Name'
					// We need to extract 'Name' and change ref to '#/$defs/Name'
					name := strings.TrimPrefix(refStr, "#/components/schemas/")
					// Update the ref in place to point to $defs
					t[k] = "#/$defs/" + name
					if err := fn(name); err != nil {
						return err
					}
				}
			} else {
				if err := r.walkRefs(val, fn); err != nil {
					return err
				}
			}
		}
	case []any:
		for _, val := range t {
			if err := r.walkRefs(val, fn); err != nil {
				return err
			}
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
