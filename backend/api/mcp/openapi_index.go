package mcp

import (
	_ "embed"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/pb33f/libopenapi"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

//go:embed gen/openapi.yaml
var openAPISpec []byte

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

	// Split by comma or space
	perms := strings.FieldsFunc(permStr, func(r rune) bool {
		return r == ',' || r == ' '
	})

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

// Services returns the list of all services.
func (idx *OpenAPIIndex) Services() []string {
	return idx.services
}

// GetEndpoint returns the endpoint info for the given operation ID.
func (idx *OpenAPIIndex) GetEndpoint(operationID string) (*EndpointInfo, bool) {
	ep, ok := idx.byOperation[operationID]
	return ep, ok
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

	// Also do substring matching on operationID and summary
	queryLower := strings.ToLower(query)
	for i := range idx.endpoints {
		ep := &idx.endpoints[i]
		if strings.Contains(strings.ToLower(ep.OperationID), queryLower) {
			scores[ep] += 3
		}
		if strings.Contains(strings.ToLower(ep.Summary), queryLower) {
			scores[ep] += 2
		}
		if strings.Contains(strings.ToLower(ep.Method), queryLower) {
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

		propType := "object"
		if prop != nil {
			propSchema := prop.Schema()
			if propSchema != nil && len(propSchema.Type) > 0 {
				propType = propSchema.Type[0]
			}
			if prop.GetReference() != "" {
				// It's a reference to another type
				refParts := strings.Split(prop.GetReference(), "/")
				propType = refParts[len(refParts)-1]
			}
		}

		desc := ""
		if prop != nil && prop.Schema() != nil {
			desc = prop.Schema().Description
		}

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
