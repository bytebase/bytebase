package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"
)

// parseIdentifier parses a table identifier and returns schema name and object name
func parseIdentifier(identifier string) (schemaName, objectName string) {
	parts := strings.Split(identifier, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", identifier
}

// extractAlterTexts extracts and concatenates text from a list of ALTER statement nodes
func extractAlterTexts(alterNodes []antlr.ParserRuleContext) string {
	if len(alterNodes) == 0 {
		return ""
	}

	var parts []string
	for _, node := range alterNodes {
		text := extractTextFromNode(node)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// extractTextFromNode extracts text from a parser rule context node
func extractTextFromNode(node antlr.ParserRuleContext) string {
	if node == nil {
		return ""
	}

	// Check for interfaces that have the required methods
	type parserContext interface {
		GetParser() antlr.Parser
		GetStart() antlr.Token
		GetStop() antlr.Token
	}

	if ruleContext, ok := node.(parserContext); ok {
		if parser := ruleContext.GetParser(); parser != nil {
			if tokenStream := parser.GetTokenStream(); tokenStream != nil {
				start := ruleContext.GetStart()
				stop := ruleContext.GetStop()
				if start != nil && stop != nil {
					return tokenStream.GetTextFromTokens(start, stop)
				}
			}
		}
	}

	// Fallback to node's GetText method
	return node.GetText()
}

// extractSchemaAndTypeFromTypename extracts schema.type identifier from a Typename context
// Used for parsing COMMENT ON TYPE statements
func extractSchemaAndTypeFromTypename(typenameCtx *parser.TypenameContext) string {
	if typenameCtx == nil {
		return ""
	}

	// Helper function to normalize a PostgreSQL identifier string
	normalizeIdentifier := func(text string) string {
		if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
			// Quoted identifier - preserve case but remove quotes
			return text[1 : len(text)-1]
		}
		// Unquoted identifier - convert to lowercase
		return strings.ToLower(text)
	}

	// Navigate to Simpletypename -> Generictype
	if typenameCtx.GetChildCount() >= 1 {
		if simpleTypeCtx, ok := typenameCtx.GetChild(0).(*parser.SimpletypenameContext); ok {
			if simpleTypeCtx.GetChildCount() >= 1 {
				if genericTypeCtx, ok := simpleTypeCtx.GetChild(0).(*parser.GenerictypeContext); ok {
					// Generictype contains Type_function_name (schema) and Attrs (.typename)
					var schemaName string
					var typeName string

					if genericTypeCtx.GetChildCount() >= 1 {
						if tfnCtx, ok := genericTypeCtx.GetChild(0).(*parser.Type_function_nameContext); ok {
							schemaName = normalizeIdentifier(tfnCtx.GetText())
						}
					}

					if genericTypeCtx.GetChildCount() >= 2 {
						if attrsCtx, ok := genericTypeCtx.GetChild(1).(*parser.AttrsContext); ok {
							// Attrs contains ".typename", need to extract the typename
							attrsText := attrsCtx.GetText()
							// Remove leading dot and normalize
							if len(attrsText) > 1 && attrsText[0] == '.' {
								typeName = normalizeIdentifier(attrsText[1:])
							}
						}
					}

					if schemaName != "" && typeName != "" {
						return schemaName + "." + typeName
					}
					// If only Type_function_name is present (no attrs), treat it as the type name with public schema
					if schemaName != "" && typeName == "" {
						return "public." + schemaName
					}
				}
			}
		}
	}

	return ""
}
