package mcp

import (
	_ "embed"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/pb33f/libopenapi"
	v3base "github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

//go:embed gen/openapi.yaml
var openAPISpec []byte

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

// EndpointInfo is the indexed representation of an API endpoint.
type EndpointInfo struct {
	// OperationID is the unique identifier for the operation (e.g., "bytebase.v1.SQLService.Query").
	OperationID string `json:"operationId"`
	// Path is the HTTP path for the endpoint (e.g., "/bytebase.v1.SQLService/Query").
	Path string `json:"path"`
	// Service is the service name (e.g., "SQLService").
	Service string `json:"service"`
	// Method is the RPC method name (e.g., "Query").
	Method string `json:"method"`
	// Summary is a brief description of the endpoint.
	Summary string `json:"summary"`
	// Description is a detailed description of the endpoint.
	Description string `json:"description"`
	// Permissions is the list of permissions required to call this endpoint.
	Permissions []string `json:"permissions,omitempty"`
	// RequestSchemaRef is the reference to the request schema (e.g., "#/components/schemas/bytebase.v1.QueryRequest").
	RequestSchemaRef string `json:"requestSchemaRef,omitempty"`
	// ResponseSchemaRef is the reference to the response schema.
	ResponseSchemaRef string `json:"responseSchemaRef,omitempty"`
}

// PropertyInfo describes a property in a schema.
type PropertyInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// OpenAPIIndex is an indexed representation of the OpenAPI spec for fast lookup.
type OpenAPIIndex struct {
	doc         *libopenapi.DocumentModel[v3high.Document]
	endpoints   []EndpointInfo
	byOperation map[string]*EndpointInfo
	byService   map[string][]*EndpointInfo
	services    []string
	keywords    map[string][]*EndpointInfo
}

// NewOpenAPIIndex creates a new OpenAPI index from the embedded spec.
func NewOpenAPIIndex() (*OpenAPIIndex, error) {
	document, err := libopenapi.NewDocument(openAPISpec)
	if err != nil {
		return nil, err
	}

	model, err := document.BuildV3Model()
	if err != nil {
		return nil, err
	}

	idx := &OpenAPIIndex{
		doc:         model,
		byOperation: make(map[string]*EndpointInfo),
		byService:   make(map[string][]*EndpointInfo),
		keywords:    make(map[string][]*EndpointInfo),
	}

	idx.parseSpec()
	return idx, nil
}

var permissionRegex = regexp.MustCompile(`Permissions required:\s*([^\n]+)`)

func (idx *OpenAPIIndex) parseSpec() {
	serviceSet := make(map[string]bool)

	if idx.doc.Model.Paths == nil {
		return
	}

	for pair := idx.doc.Model.Paths.PathItems.First(); pair != nil; pair = pair.Next() {
		path := pair.Key()
		pathItem := pair.Value()

		// Connect RPC uses POST for all endpoints
		op := pathItem.Post
		if op == nil {
			continue
		}

		// Extract service and method from path
		// /bytebase.v1.SQLService/Query -> service=SQLService, method=Query
		trimmed := strings.TrimPrefix(path, "/bytebase.v1.")
		parts := strings.Split(trimmed, "/")
		if len(parts) != 2 {
			continue
		}

		service := parts[0]
		method := parts[1]

		endpoint := EndpointInfo{
			OperationID: op.OperationId,
			Path:        path,
			Service:     service,
			Method:      method,
			Summary:     op.Summary,
			Description: op.Description,
			Permissions: extractPermissions(op.Description),
		}

		// Extract request schema reference
		if op.RequestBody != nil {
			content := op.RequestBody.Content
			if content != nil {
				if jsonContent, ok := content.Get("application/json"); ok && jsonContent.Schema != nil {
					endpoint.RequestSchemaRef = jsonContent.Schema.GetReference()
				}
			}
		}

		// Extract response schema reference (from 200 response)
		if op.Responses != nil {
			if resp, ok := op.Responses.Codes.Get("200"); ok && resp != nil {
				if resp.Content != nil {
					if jsonContent, ok := resp.Content.Get("application/json"); ok && jsonContent.Schema != nil {
						endpoint.ResponseSchemaRef = jsonContent.Schema.GetReference()
					}
				}
			}
		}

		idx.endpoints = append(idx.endpoints, endpoint)
		ptr := &idx.endpoints[len(idx.endpoints)-1]

		idx.byOperation[endpoint.OperationID] = ptr
		idx.byService[service] = append(idx.byService[service], ptr)
		serviceSet[service] = true

		// Index keywords for search
		idx.indexKeywords(ptr)
	}

	// Sort services alphabetically
	for s := range serviceSet {
		idx.services = append(idx.services, s)
	}
	slices.Sort(idx.services)
}

func extractPermissions(description string) []string {
	matches := permissionRegex.FindStringSubmatch(description)
	if len(matches) < 2 {
		return nil
	}

	permStr := strings.TrimSpace(matches[1])
	if permStr == "None" || permStr == "" {
		return nil
	}

	// Remove parenthetical notes like "(for project parent)"
	if idx := strings.Index(permStr, "("); idx > 0 {
		permStr = strings.TrimSpace(permStr[:idx])
	}

	// Split by comma
	perms := strings.Split(permStr, ",")

	var result []string
	for _, p := range perms {
		p = strings.TrimSpace(p)
		if p != "" && p != "None" {
			result = append(result, p)
		}
	}
	return result
}

func (idx *OpenAPIIndex) indexKeywords(ep *EndpointInfo) {
	// Extract keywords from service, method, summary
	keywords := extractKeywords(ep.Service, ep.Method, ep.Summary)
	for _, kw := range keywords {
		idx.keywords[kw] = append(idx.keywords[kw], ep)
	}
}

func extractKeywords(texts ...string) []string {
	keywordSet := make(map[string]bool)

	for _, text := range texts {
		// Split camelCase and PascalCase
		words := splitCamelCase(text)
		for _, word := range words {
			word = strings.ToLower(word)
			if len(word) >= 2 && !isStopWord(word) {
				keywordSet[word] = true
			}
		}

		// Also split by spaces and punctuation
		for _, word := range strings.FieldsFunc(text, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}) {
			word = strings.ToLower(word)
			if len(word) >= 2 && !isStopWord(word) {
				keywordSet[word] = true
			}
		}
	}

	var result []string
	for kw := range keywordSet {
		result = append(result, kw)
	}
	return result
}

func splitCamelCase(s string) []string {
	var words []string
	var current strings.Builder

	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(r)
		} else if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			// Non-alphanumeric, flush current word
			words = append(words, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "with": true, "by": true, "from": true,
		"service": true, "request": true, "response": true,
	}
	return stopWords[word]
}

// isPrimaryCRUD returns true if the method is a primary CRUD operation without prefixes.
func isPrimaryCRUD(method string) bool {
	primaryOps := map[string]bool{
		"List": true, "Get": true, "Create": true, "Update": true, "Delete": true,
		"Search": true, "Query": true, "Execute": true,
	}
	// Check if method starts with a primary operation (e.g., "ListDatabases" -> "List")
	for op := range primaryOps {
		if strings.HasPrefix(method, op) {
			// Ensure it's at the beginning (not "BatchList" or "PreviewList")
			return true
		}
	}
	return false
}

// Services returns the list of all services.
func (idx *OpenAPIIndex) Services() []string {
	return idx.services
}

// GetEndpoint returns the endpoint info for the given operation ID.
// Supports both full format (bytebase.v1.SQLService.Query) and short format (SQLService/Query).
func (idx *OpenAPIIndex) GetEndpoint(operationID string) (*EndpointInfo, bool) {
	// Try full format first
	if ep, ok := idx.byOperation[operationID]; ok {
		return ep, true
	}

	// Try short format: Service/Method -> bytebase.v1.Service.Method
	if parts := strings.Split(operationID, "/"); len(parts) == 2 {
		fullID := fmt.Sprintf("bytebase.v1.%s.%s", parts[0], parts[1])
		if ep, ok := idx.byOperation[fullID]; ok {
			return ep, true
		}
	}

	return nil, false
}

// GetServiceEndpoints returns all endpoints for the given service.
func (idx *OpenAPIIndex) GetServiceEndpoints(service string) []*EndpointInfo {
	return idx.byService[service]
}

// Search searches for endpoints matching the query.
func (idx *OpenAPIIndex) Search(query string) []*EndpointInfo {
	queryKeywords := extractKeywords(query)
	if len(queryKeywords) == 0 {
		return nil
	}

	// Score each endpoint by number of matching keywords
	scores := make(map[*EndpointInfo]int)
	for _, kw := range queryKeywords {
		for _, ep := range idx.keywords[kw] {
			scores[ep]++
		}
	}

	// Also do substring matching on operationID, summary, method, and service
	queryLower := strings.ToLower(query)
	for i := range idx.endpoints {
		ep := &idx.endpoints[i]
		methodLower := strings.ToLower(ep.Method)
		serviceLower := strings.ToLower(ep.Service)
		matched := false

		// Exact method name match gets highest boost
		if methodLower == queryLower {
			scores[ep] += 10
			matched = true
		} else if strings.Contains(methodLower, queryLower) {
			scores[ep] += 5
			matched = true
		}

		// Service name match (e.g., "sql" matches "SQLService")
		if strings.Contains(serviceLower, queryLower) {
			scores[ep] += 4
			matched = true
		}

		// Substring match in operationID
		if strings.Contains(strings.ToLower(ep.OperationID), queryLower) {
			scores[ep] += 3
			matched = true
		}

		// Substring match in summary
		if strings.Contains(strings.ToLower(ep.Summary), queryLower) {
			scores[ep] += 2
			matched = true
		}

		// Boost primary CRUD operations only if there's already a match
		if matched && isPrimaryCRUD(ep.Method) {
			scores[ep] += 2
		}
	}

	// Sort by score descending
	type scored struct {
		ep    *EndpointInfo
		score int
	}
	var results []scored
	for ep, score := range scores {
		results = append(results, scored{ep, score})
	}
	slices.SortFunc(results, func(a, b scored) int {
		return b.score - a.score // descending
	})

	var endpoints []*EndpointInfo
	for _, r := range results {
		endpoints = append(endpoints, r.ep)
	}
	return endpoints
}

// SearchInService searches for endpoints matching the query within a specific service.
func (idx *OpenAPIIndex) SearchInService(query, service string) []*EndpointInfo {
	allResults := idx.Search(query)
	var filtered []*EndpointInfo
	for _, ep := range allResults {
		if ep.Service == service {
			filtered = append(filtered, ep)
		}
	}
	return filtered
}

// GetRequestSchema returns the simplified schema properties for a request type.
func (idx *OpenAPIIndex) GetRequestSchema(schemaRef string) []PropertyInfo {
	if schemaRef == "" || idx.doc.Model.Components == nil || idx.doc.Model.Components.Schemas == nil {
		return nil
	}

	// Extract schema name from ref: "#/components/schemas/bytebase.v1.QueryRequest"
	parts := strings.Split(schemaRef, "/")
	schemaName := parts[len(parts)-1]

	schemaProxy, ok := idx.doc.Model.Components.Schemas.Get(schemaName)
	if !ok || schemaProxy == nil {
		return nil
	}

	schema := schemaProxy.Schema()
	if schema == nil {
		return nil
	}

	var props []PropertyInfo
	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	if schema.Properties == nil {
		return nil
	}

	for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
		name := pair.Key()
		prop := pair.Value()

		propType, desc := extractPropertyTypeAndDesc(prop)

		props = append(props, PropertyInfo{
			Name:        name,
			Type:        propType,
			Description: desc,
			Required:    requiredSet[name],
		})
	}

	// Sort by name for consistent output
	slices.SortFunc(props, func(a, b PropertyInfo) int {
		return strings.Compare(a.Name, b.Name)
	})

	return props
}

// extractPropertyTypeAndDesc extracts the type string and description from a schema property.
// For arrays, it includes the item type (e.g., "array<string>").
func extractPropertyTypeAndDesc(prop *v3base.SchemaProxy) (string, string) {
	if prop == nil {
		return "object", ""
	}

	propSchema := prop.Schema()
	propType := "object"

	if propSchema != nil && len(propSchema.Type) > 0 {
		propType = propSchema.Type[0]

		// For arrays, get the item type
		if propType == "array" && propSchema.Items != nil && propSchema.Items.A != nil {
			// Check for $ref first (before resolving schema)
			if propSchema.Items.A.GetReference() != "" {
				refParts := strings.Split(propSchema.Items.A.GetReference(), "/")
				propType = fmt.Sprintf("array<%s>", refParts[len(refParts)-1])
			} else if itemSchema := propSchema.Items.A.Schema(); itemSchema != nil && len(itemSchema.Type) > 0 {
				propType = fmt.Sprintf("array<%s>", itemSchema.Type[0])
			}
		}
	}

	if prop.GetReference() != "" {
		// It's a reference to another type
		refParts := strings.Split(prop.GetReference(), "/")
		propType = refParts[len(refParts)-1]
	}

	desc := ""
	if propSchema != nil {
		desc = propSchema.Description
	}

	return propType, desc
}

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
			if v != nil && v.Value != "" {
				enumValues = append(enumValues, v.Value)
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

		propType, desc := extractPropertyTypeAndDesc(prop)

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
