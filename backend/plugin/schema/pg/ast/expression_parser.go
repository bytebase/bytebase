package ast

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/parser/postgresql"
)

// ExpressionParser parses PostgreSQL expressions into AST
type ExpressionParser struct {
	// Configuration options
	IgnoreSchemaPrefix bool // ignore public schema prefix
	CaseSensitive      bool // case sensitive comparison
}

// NewExpressionParser creates a new expression parser
func NewExpressionParser() *ExpressionParser {
	return &ExpressionParser{
		IgnoreSchemaPrefix: true,  // default to ignore public schema
		CaseSensitive:      false, // default case insensitive
	}
}

// ParseExpression parses a PostgreSQL expression into AST
func (p *ExpressionParser) ParseExpression(expr string) (ExpressionAST, error) {
	if strings.TrimSpace(expr) == "" {
		return nil, nil
	}

	// Use ANTLR to parse the expression
	inputStream := antlr.NewInputStream(expr)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	tokens := stream.GetAllTokens()

	// Build AST from tokens using a simple token-based approach
	// This is more practical than full grammar parsing for expressions
	return p.buildASTFromTokens(tokens)
}

// buildASTFromTokens builds AST from ANTLR tokens
func (p *ExpressionParser) buildASTFromTokens(tokens []antlr.Token) (ExpressionAST, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	// Filter out whitespace and EOF tokens
	var filteredTokens []antlr.Token
	for _, token := range tokens {
		if !p.isWhitespaceToken(token.GetTokenType()) && token.GetText() != "<EOF>" {
			filteredTokens = append(filteredTokens, token)
		}
	}

	if len(filteredTokens) == 0 {
		return nil, nil
	}

	// Parse expression using recursive descent approach
	return p.parseExpressionTokens(filteredTokens, 0, len(filteredTokens))
}

// parseExpressionTokens parses expression tokens recursively
func (p *ExpressionParser) parseExpressionTokens(tokens []antlr.Token, start, end int) (ExpressionAST, error) {
	if start >= end {
		return nil, nil
	}

	// Handle parentheses - only if they truly wrap the entire expression
	if tokens[start].GetText() == "(" && tokens[end-1].GetText() == ")" {
		// Check if the opening parenthesis has a matching closing parenthesis at the end
		// AND there are no other top-level operators outside these parentheses
		if p.isBalancedParentheses(tokens, start, end) && p.parenthesesWrapEntireExpression(tokens, start, end) {
			inner, err := p.parseExpressionTokens(tokens, start+1, end-1)
			if err != nil {
				return nil, err
			}
			return &ParenthesesExpr{Inner: inner}, nil
		}
	}

	// Check for expression lists (comma-separated expressions)
	// Only parse as list if there are actually commas at the top level
	if listExpr := p.parseExpressionList(tokens, start, end); listExpr != nil {
		return listExpr, nil
	}

	// Find binary operators (precedence handling)
	if binaryExpr := p.parseBinaryOperation(tokens, start, end); binaryExpr != nil {
		return binaryExpr, nil
	}

	// Find unary operators
	if unaryExpr := p.parseUnaryOperation(tokens, start, end); unaryExpr != nil {
		return unaryExpr, nil
	}

	// Parse function calls
	if funcExpr := p.parseFunctionCall(tokens, start, end); funcExpr != nil {
		return funcExpr, nil
	}

	// Parse single token (identifier or literal)
	if end-start == 1 {
		return p.parseSingleToken(tokens[start])
	}

	// Parse qualified identifier (schema.table.column)
	if qualifiedExpr := p.parseQualifiedIdentifier(tokens, start, end); qualifiedExpr != nil {
		return qualifiedExpr, nil
	}

	// Fallback: create a literal expression with the concatenated text
	var parts []string
	for i := start; i < end; i++ {
		parts = append(parts, tokens[i].GetText())
	}
	return &LiteralExpr{
		Value:     strings.Join(parts, " "),
		ValueType: "unknown",
	}, nil
}

// parseMultiKeywordOperators parses multi-keyword operators like IS NULL, BETWEEN...AND, etc.
func (p *ExpressionParser) parseMultiKeywordOperators(tokens []antlr.Token, start, end int) ExpressionAST {
	// Look for CASE expressions first (they are most complex)
	if caseExpr := p.parseCaseExpression(tokens, start, end); caseExpr != nil {
		return caseExpr
	}

	// Look for IS NULL / IS NOT NULL patterns
	if nullExpr := p.parseIsNullOperator(tokens, start, end); nullExpr != nil {
		return nullExpr
	}

	// Look for BETWEEN...AND patterns
	return p.parseBetweenOperator(tokens, start, end)
}

// parseCaseExpression parses CASE WHEN...THEN...ELSE...END expressions
func (p *ExpressionParser) parseCaseExpression(tokens []antlr.Token, start, end int) ExpressionAST {
	// Look for CASE...END pattern
	if start < end && strings.ToUpper(tokens[start].GetText()) == "CASE" &&
		strings.ToUpper(tokens[end-1].GetText()) == "END" {
		// Parse the CASE expression content between CASE and END
		var caseArgs []ExpressionAST

		// Add a marker to identify this as a CASE expression
		caseArgs = append(caseArgs, &LiteralExpr{Value: "CASE", ValueType: "keyword"})

		// Parse WHEN...THEN pairs and optional ELSE
		i := start + 1
		for i < end-1 {
			tokenText := strings.ToUpper(tokens[i].GetText())

			if tokenText == "WHEN" {
				// Find the corresponding THEN
				thenPos := -1
				parenLevel := 0
				for j := i + 1; j < end-1; j++ {
					if tokens[j].GetText() == "(" {
						parenLevel++
					} else if tokens[j].GetText() == ")" {
						parenLevel--
					} else if parenLevel == 0 && strings.ToUpper(tokens[j].GetText()) == "THEN" {
						thenPos = j
						break
					}
				}

				if thenPos != -1 {
					// Parse condition between WHEN and THEN
					condition, _ := p.parseExpressionTokens(tokens, i+1, thenPos)
					if condition != nil {
						caseArgs = append(caseArgs, condition)
					}

					// Find the next WHEN, ELSE, or END to determine the THEN value range
					nextPos := end - 1 // default to END
					parenLevel = 0
					for j := thenPos + 1; j < end-1; j++ {
						if tokens[j].GetText() == "(" {
							parenLevel++
						} else if tokens[j].GetText() == ")" {
							parenLevel--
						} else if parenLevel == 0 {
							nextToken := strings.ToUpper(tokens[j].GetText())
							if nextToken == "WHEN" || nextToken == "ELSE" {
								nextPos = j
								break
							}
						}
					}

					// Parse value between THEN and next keyword
					value, _ := p.parseExpressionTokens(tokens, thenPos+1, nextPos)
					if value != nil {
						caseArgs = append(caseArgs, value)
					}

					i = nextPos
				} else {
					i++
				}
			} else if tokenText == "ELSE" {
				// Parse ELSE value (everything from ELSE to END)
				elseValue, _ := p.parseExpressionTokens(tokens, i+1, end-1)
				if elseValue != nil {
					caseArgs = append(caseArgs, elseValue)
				}
				break
			} else {
				i++
			}
		}

		return &FunctionExpr{
			Name: "CASE",
			Args: caseArgs,
		}
	}

	return nil
}

// parseIsNullOperator parses IS NULL and IS NOT NULL operators
func (p *ExpressionParser) parseIsNullOperator(tokens []antlr.Token, start, end int) ExpressionAST {
	// Look for IS NULL pattern (minimum 3 tokens: operand IS NULL)
	for i := start + 1; i < end-1; i++ {
		if p.getParenthesesLevel(tokens, start, i) == 0 {
			isToken := strings.ToUpper(tokens[i].GetText())

			if isToken == "IS" {
				// Check for IS NOT NULL (4 tokens after IS)
				if i+2 < end {
					notToken := strings.ToUpper(tokens[i+1].GetText())
					nullToken := strings.ToUpper(tokens[i+2].GetText())

					if notToken == "NOT" && nullToken == "NULL" {
						// Parse: operand IS NOT NULL
						left, _ := p.parseExpressionTokens(tokens, start, i)
						if left != nil {
							return &BinaryOpExpr{
								Left:     left,
								Operator: "IS NOT NULL",
								Right:    &LiteralExpr{Value: "NULL", ValueType: "null"},
							}
						}
					}
				}

				// Check for IS NULL (2 tokens after IS)
				if i+1 < end {
					nullToken := strings.ToUpper(tokens[i+1].GetText())

					if nullToken == "NULL" {
						// Parse: operand IS NULL
						left, _ := p.parseExpressionTokens(tokens, start, i)
						if left != nil {
							return &BinaryOpExpr{
								Left:     left,
								Operator: "IS NULL",
								Right:    &LiteralExpr{Value: "NULL", ValueType: "null"},
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// parseBetweenOperator parses BETWEEN...AND operators
func (p *ExpressionParser) parseBetweenOperator(tokens []antlr.Token, start, end int) ExpressionAST {
	// Look for BETWEEN pattern (minimum 5 tokens: operand BETWEEN value1 AND value2)
	for i := start + 1; i < end-3; i++ {
		if p.getParenthesesLevel(tokens, start, i) == 0 {
			betweenToken := strings.ToUpper(tokens[i].GetText())

			if betweenToken == "BETWEEN" {
				// Find the AND keyword
				for j := i + 2; j < end-1; j++ {
					if p.getParenthesesLevel(tokens, i+1, j) == 0 {
						andToken := strings.ToUpper(tokens[j].GetText())

						if andToken == "AND" {
							// Parse: operand BETWEEN value1 AND value2
							left, _ := p.parseExpressionTokens(tokens, start, i)
							value1, _ := p.parseExpressionTokens(tokens, i+1, j)
							value2, _ := p.parseExpressionTokens(tokens, j+1, end)

							if left != nil && value1 != nil && value2 != nil {
								// Create a special function-like expression for BETWEEN
								return &FunctionExpr{
									Name: "BETWEEN",
									Args: []ExpressionAST{left, value1, value2},
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// parseExpressionList parses comma-separated expression lists
func (p *ExpressionParser) parseExpressionList(tokens []antlr.Token, start, end int) ExpressionAST {
	// Look for comma separators at the top level (not inside parentheses)
	var commaPositions []int
	parenLevel := 0

	for i := start; i < end; i++ {
		tokenText := tokens[i].GetText()
		if tokenText == "(" {
			parenLevel++
		} else if tokenText == ")" {
			parenLevel--
		} else if tokenText == "," && parenLevel == 0 {
			commaPositions = append(commaPositions, i)
		}
	}

	// If we found commas at the top level, parse as expression list
	if len(commaPositions) > 0 {
		var elements []ExpressionAST

		// Parse elements between commas, but use a non-recursive approach for list elements
		prevPos := start
		for _, commaPos := range commaPositions {
			if commaPos > prevPos {
				element := p.parseNonListExpression(tokens, prevPos, commaPos)
				if element != nil {
					elements = append(elements, element)
				}
			}
			prevPos = commaPos + 1
		}

		// Parse last element
		if prevPos < end {
			element := p.parseNonListExpression(tokens, prevPos, end)
			if element != nil {
				elements = append(elements, element)
			}
		}

		if len(elements) > 1 {
			return &ListExpr{Elements: elements}
		}
	}

	return nil
}

// parseNonListExpression parses expressions but excludes list parsing to avoid infinite recursion
func (p *ExpressionParser) parseNonListExpression(tokens []antlr.Token, start, end int) ExpressionAST {
	if start >= end {
		return nil
	}

	// Handle parentheses
	if tokens[start].GetText() == "(" && tokens[end-1].GetText() == ")" {
		// Check if parentheses are balanced and wrap the entire expression
		if p.isBalancedParentheses(tokens, start, end) {
			inner := p.parseNonListExpression(tokens, start+1, end-1)
			if inner != nil {
				return &ParenthesesExpr{Inner: inner}
			}
		}
	}

	// Find binary operators (precedence handling)
	if binaryExpr := p.parseBinaryOperation(tokens, start, end); binaryExpr != nil {
		return binaryExpr
	}

	// Find unary operators
	if unaryExpr := p.parseUnaryOperation(tokens, start, end); unaryExpr != nil {
		return unaryExpr
	}

	// Parse function calls
	if funcExpr := p.parseFunctionCall(tokens, start, end); funcExpr != nil {
		return funcExpr
	}

	// Parse single token (identifier or literal)
	if end-start == 1 {
		result, _ := p.parseSingleToken(tokens[start])
		return result
	}

	// Parse qualified identifier (schema.table.column)
	if qualifiedExpr := p.parseQualifiedIdentifier(tokens, start, end); qualifiedExpr != nil {
		return qualifiedExpr
	}

	// Fallback: create a literal expression with the concatenated text
	var parts []string
	for i := start; i < end; i++ {
		parts = append(parts, tokens[i].GetText())
	}
	return &LiteralExpr{
		Value:     strings.Join(parts, " "),
		ValueType: "unknown",
	}
}

// parseBinaryOperation parses binary operations with precedence
func (p *ExpressionParser) parseBinaryOperation(tokens []antlr.Token, start, end int) ExpressionAST {
	// Check for multi-keyword operators first
	if multiKeywordExpr := p.parseMultiKeywordOperators(tokens, start, end); multiKeywordExpr != nil {
		return multiKeywordExpr
	}

	// Precedence levels (lowest to highest)
	precedenceLevels := [][]string{
		{"OR", "||"}, // lowest precedence
		{"AND", "&&"},
		{"=", "<>", "!=", "<", ">", "<=", ">=", "LIKE", "ILIKE"},
		{"+", "-"},
		{"*", "/", "%"},
		{"::"}, // type cast has high precedence
	}

	// Find operators from lowest to highest precedence
	for _, operators := range precedenceLevels {
		for i := end - 2; i > start; i-- { // scan right to left for left associativity
			if p.containsOperator(operators, tokens[i].GetText()) {
				// Make sure this operator is not inside parentheses
				if p.getParenthesesLevel(tokens, start, i) == 0 {
					left, _ := p.parseExpressionTokens(tokens, start, i)
					right, _ := p.parseExpressionTokens(tokens, i+1, end)
					if left != nil && right != nil {
						return &BinaryOpExpr{
							Left:     left,
							Operator: strings.ToUpper(tokens[i].GetText()),
							Right:    right,
						}
					}
				}
			}
		}
	}

	return nil
}

// parseUnaryOperation parses unary operations
func (p *ExpressionParser) parseUnaryOperation(tokens []antlr.Token, start, end int) ExpressionAST {
	if start >= end {
		return nil
	}

	unaryOps := []string{"NOT", "-", "+"}
	firstToken := strings.ToUpper(tokens[start].GetText())

	for _, op := range unaryOps {
		if firstToken == op {
			operand, _ := p.parseExpressionTokens(tokens, start+1, end)
			if operand != nil {
				return &UnaryOpExpr{
					Operator: op,
					Operand:  operand,
				}
			}
		}
	}

	return nil
}

// parseFunctionCall parses function calls
func (p *ExpressionParser) parseFunctionCall(tokens []antlr.Token, start, end int) ExpressionAST {
	// Check for special EXTRACT function first
	if extractExpr := p.parseExtractFunction(tokens, start, end); extractExpr != nil {
		return extractExpr
	}

	// Look for pattern: identifier ( args )
	for i := start; i < end-1; i++ {
		if tokens[i+1].GetText() == "(" {
			// Find matching closing parenthesis
			parenLevel := 0
			closeParenPos := -1
			for j := i + 1; j < end; j++ {
				if tokens[j].GetText() == "(" {
					parenLevel++
				} else if tokens[j].GetText() == ")" {
					parenLevel--
					if parenLevel == 0 {
						closeParenPos = j
						break
					}
				}
			}

			if closeParenPos == end-1 { // function call spans the entire range
				// Parse function name (may be qualified)
				funcName := p.parseQualifiedName(tokens, start, i+1)

				// Parse arguments
				var args []ExpressionAST
				if closeParenPos > i+2 { // has arguments
					args = p.parseArgumentList(tokens, i+2, closeParenPos)
				}

				return &FunctionExpr{
					Schema: funcName.Schema,
					Name:   funcName.Name,
					Args:   args,
				}
			}
		}
	}

	return nil
}

// parseExtractFunction parses EXTRACT(field FROM source) function calls
func (p *ExpressionParser) parseExtractFunction(tokens []antlr.Token, start, end int) ExpressionAST {
	// Look for EXTRACT( pattern
	if start < end-1 &&
		strings.ToUpper(tokens[start].GetText()) == "EXTRACT" &&
		tokens[start+1].GetText() == "(" {
		// Find matching closing parenthesis
		parenLevel := 0
		closeParenPos := -1
		for j := start + 1; j < end; j++ {
			if tokens[j].GetText() == "(" {
				parenLevel++
			} else if tokens[j].GetText() == ")" {
				parenLevel--
				if parenLevel == 0 {
					closeParenPos = j
					break
				}
			}
		}

		if closeParenPos == end-1 { // EXTRACT call spans the entire range
			// Look for FROM keyword inside parentheses
			for k := start + 2; k < closeParenPos-1; k++ {
				if strings.ToUpper(tokens[k].GetText()) == "FROM" {
					// Parse: EXTRACT(field FROM source)
					field, _ := p.parseExpressionTokens(tokens, start+2, k)
					source, _ := p.parseExpressionTokens(tokens, k+1, closeParenPos)

					if field != nil && source != nil {
						return &FunctionExpr{
							Name: "EXTRACT",
							Args: []ExpressionAST{field, source},
						}
					}
				}
			}
		}
	}

	return nil
}

// parseQualifiedIdentifier parses qualified identifiers like schema.table.column
func (p *ExpressionParser) parseQualifiedIdentifier(tokens []antlr.Token, start, end int) ExpressionAST {
	if end-start < 1 {
		return nil
	}

	// Check if all tokens form a qualified identifier (identifier.identifier.identifier)
	parts := []string{}
	expectIdentifier := true

	for i := start; i < end; i++ {
		token := tokens[i]
		tokenText := token.GetText()

		if expectIdentifier {
			if p.isIdentifierToken(token) {
				parts = append(parts, tokenText)
				expectIdentifier = false
			} else {
				return nil // not a qualified identifier
			}
		} else {
			if tokenText == "." {
				expectIdentifier = true
			} else {
				return nil // not a qualified identifier
			}
		}
	}

	if len(parts) == 0 || expectIdentifier {
		return nil // incomplete qualified identifier
	}

	// Create identifier expression based on number of parts
	switch len(parts) {
	case 1:
		return &IdentifierExpr{Name: parts[0]}
	case 2:
		// Could be schema.table or table.column
		return &IdentifierExpr{Table: parts[0], Name: parts[1]}
	case 3:
		return &IdentifierExpr{Schema: parts[0], Table: parts[1], Name: parts[2]}
	default:
		// More than 3 parts, treat as complex identifier
		return &IdentifierExpr{Name: strings.Join(parts, ".")}
	}
}

// parseSingleToken parses a single token into an expression
func (p *ExpressionParser) parseSingleToken(token antlr.Token) (ExpressionAST, error) {
	tokenText := token.GetText()

	// Check if it's a quoted identifier (PostgreSQL uses double quotes for identifiers)
	if p.isQuotedIdentifier(tokenText) {
		return &IdentifierExpr{Name: tokenText}, nil
	}

	// Check if it's a literal
	if p.isStringLiteral(tokenText) {
		return &LiteralExpr{
			Value:     tokenText,
			ValueType: "string",
		}, nil
	}

	if p.isNumericLiteral(tokenText) {
		return &LiteralExpr{
			Value:     tokenText,
			ValueType: "number",
		}, nil
	}

	if p.isBooleanLiteral(tokenText) {
		return &LiteralExpr{
			Value:     strings.ToUpper(tokenText),
			ValueType: "boolean",
		}, nil
	}

	if strings.ToUpper(tokenText) == "NULL" {
		return &LiteralExpr{
			Value:     "NULL",
			ValueType: "null",
		}, nil
	}

	// Default to identifier
	if p.isIdentifierToken(token) {
		return &IdentifierExpr{Name: tokenText}, nil
	}

	// Fallback to literal
	return &LiteralExpr{
		Value:     tokenText,
		ValueType: "unknown",
	}, nil
}

// Helper functions

// isWhitespaceToken checks if token is whitespace
func (*ExpressionParser) isWhitespaceToken(tokenType int) bool {
	return tokenType == pgparser.PostgreSQLLexerWhitespace ||
		tokenType == pgparser.PostgreSQLLexerNewline
}

// containsOperator checks if operators list contains the given operator
func (*ExpressionParser) containsOperator(operators []string, op string) bool {
	opUpper := strings.ToUpper(op)
	for _, operator := range operators {
		if strings.ToUpper(operator) == opUpper {
			return true
		}
	}
	return false
}

// getParenthesesLevel returns the parentheses nesting level at given position
func (*ExpressionParser) getParenthesesLevel(tokens []antlr.Token, start, pos int) int {
	level := 0
	for i := start; i < pos; i++ {
		if tokens[i].GetText() == "(" {
			level++
		} else if tokens[i].GetText() == ")" {
			level--
		}
	}
	return level
}

// isBalancedParentheses checks if parentheses are balanced in the range
func (*ExpressionParser) isBalancedParentheses(tokens []antlr.Token, start, end int) bool {
	level := 0
	for i := start; i < end; i++ {
		if tokens[i].GetText() == "(" {
			level++
		} else if tokens[i].GetText() == ")" {
			level--
		}
		if level < 0 {
			return false
		}
	}
	return level == 0
}

// parenthesesWrapEntireExpression checks if parentheses at start/end truly wrap the entire expression
func (*ExpressionParser) parenthesesWrapEntireExpression(tokens []antlr.Token, start, end int) bool {
	if start >= end-1 || tokens[start].GetText() != "(" || tokens[end-1].GetText() != ")" {
		return false
	}

	// Find the matching closing parenthesis for the opening parenthesis at start
	level := 0
	matchingClosePos := -1

	for i := start; i < end; i++ {
		if tokens[i].GetText() == "(" {
			level++
		} else if tokens[i].GetText() == ")" {
			level--
			if level == 0 {
				matchingClosePos = i
				break
			}
		}
	}

	// If the matching closing parenthesis is at the end, then parentheses wrap entire expression
	return matchingClosePos == end-1
}

// isIdentifierToken checks if token is an identifier
func (*ExpressionParser) isIdentifierToken(token antlr.Token) bool {
	tokenText := token.GetText()
	tokenType := token.GetTokenType()

	// Check for quoted identifiers
	if tokenType == pgparser.PostgreSQLLexerQuotedIdentifier ||
		tokenType == pgparser.PostgreSQLLexerUnicodeQuotedIdentifier {
		return true
	}

	// Check for unquoted identifiers (simple pattern matching)
	if len(tokenText) > 0 {
		firstChar := tokenText[0]
		if (firstChar >= 'a' && firstChar <= 'z') ||
			(firstChar >= 'A' && firstChar <= 'Z') ||
			firstChar == '_' {
			return true
		}
	}

	return false
}

// isStringLiteral checks if token is a string literal (single quotes in PostgreSQL)
func (*ExpressionParser) isStringLiteral(tokenText string) bool {
	return len(tokenText) >= 2 &&
		tokenText[0] == '\'' && tokenText[len(tokenText)-1] == '\''
}

// isQuotedIdentifier checks if token is a quoted identifier (double quotes in PostgreSQL)
func (*ExpressionParser) isQuotedIdentifier(tokenText string) bool {
	return len(tokenText) >= 2 &&
		tokenText[0] == '"' && tokenText[len(tokenText)-1] == '"'
}

// isNumericLiteral checks if token is numeric
func (*ExpressionParser) isNumericLiteral(tokenText string) bool {
	if len(tokenText) == 0 {
		return false
	}

	for i, char := range tokenText {
		if i == 0 && (char == '+' || char == '-') {
			continue
		}
		if char == '.' {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// isBooleanLiteral checks if token is boolean
func (*ExpressionParser) isBooleanLiteral(tokenText string) bool {
	upper := strings.ToUpper(tokenText)
	return upper == "TRUE" || upper == "FALSE"
}

// parseQualifiedName parses qualified name from tokens
func (*ExpressionParser) parseQualifiedName(tokens []antlr.Token, start, end int) *IdentifierExpr {
	var parts []string

	for i := start; i < end; i++ {
		token := tokens[i]
		if token.GetText() != "." {
			parts = append(parts, token.GetText())
		}
	}

	switch len(parts) {
	case 1:
		return &IdentifierExpr{Name: parts[0]}
	case 2:
		return &IdentifierExpr{Schema: parts[0], Name: parts[1]}
	default:
		return &IdentifierExpr{Name: strings.Join(parts, ".")}
	}
}

// parseArgumentList parses function argument list
func (p *ExpressionParser) parseArgumentList(tokens []antlr.Token, start, end int) []ExpressionAST {
	if start >= end {
		return nil
	}

	var args []ExpressionAST
	argStart := start
	parenLevel := 0

	for i := start; i <= end; i++ {
		var tokenText string
		if i < end {
			tokenText = tokens[i].GetText()
		}

		if tokenText == "(" {
			parenLevel++
		} else if tokenText == ")" {
			parenLevel--
		} else if tokenText == "," && parenLevel == 0 || i == end {
			// Found argument boundary
			if argStart < i {
				if arg, _ := p.parseExpressionTokens(tokens, argStart, i); arg != nil {
					args = append(args, arg)
				}
			}
			argStart = i + 1
		}
	}

	return args
}
