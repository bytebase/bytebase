package partiql

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/partiql"
	"github.com/bytebase/omni/partiql/analysis"
	"github.com/bytebase/omni/partiql/ast"
	"github.com/bytebase/omni/partiql/catalog"
	"github.com/bytebase/omni/partiql/completion"
	"github.com/bytebase/omni/partiql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:         SplitSQL,
		Parse:         parsePartiQLStatements,
		Complete:      Completion,
		ValidateQuery: validateQuery,
	}, storepb.Engine_DYNAMODB)
}

// astNode wraps an omni AST node to implement the base.AST interface.
type astNode = base.NodeAST[ast.Node]

// SplitSQL splits the input into multiple SQL statements using the omni splitter.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, partiql.Split(statement), func(s partiql.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, false), nil
}

// parsePartiQLStatements is the ParseStatementsFunc for PartiQL (DynamoDB).
// Returns []ParsedStatement with both text and AST populated.
func parsePartiQLStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty {
			if strings.TrimSpace(stmt.Text) == "" {
				ps.Empty = true
			} else {
				// Parse the segment as-is so that ParseError byte offsets
				// align with stmt.Text; convertParseError then offsets them
				// by stmt.Start to refer back to the original script.
				list, err := parser.Parse(stmt.Text)
				if err != nil {
					return nil, convertParseError(stmt.Text, err, stmt.Start)
				}
				var node ast.Node
				if len(list.Items) > 0 {
					node = list.Items[0]
				}
				ps.AST = &astNode{
					Node:          node,
					StartPosition: stmt.Start,
				}
			}
		}
		result = append(result, ps)
	}

	return result, nil
}

func validateQuery(statement string) (bool, bool, error) {
	err := analysis.ValidateQuery(statement)
	if err == nil {
		return true, true, nil
	}
	// Distinguish syntax errors (*parser.ParseError -> *base.SyntaxError) from
	// validation rejections (syntactically valid DML/DDL/EXEC that is not read-only).
	var parseErr *parser.ParseError
	if errors.As(err, &parseErr) {
		return false, false, convertParseError(statement, err, nil)
	}
	return false, false, nil
}

// Completion provides auto-complete candidates for PartiQL statements,
// delegating keyword, table, and column resolution to omni's PartiQL completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	cat := catalog.New()
	var protoMeta *metadatapb.DatabaseSchemaMetadata
	if cCtx.Metadata != nil && cCtx.DefaultDatabase != "" {
		_, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, cCtx.DefaultDatabase)
		if err == nil && dbMeta != nil {
			protoMeta = dbMeta.GetProto()
			cat.LoadMetadata(protoMeta)
		}
	}

	pos := lineOffsetToBytePos(statement, caretLine, caretOffset)
	candidates := completion.CompleteWithMetadata(statement, pos, cat, protoMeta)

	return base.DeduplicateCandidates(candidates, func(c completion.Candidate) (base.Candidate, bool) {
		typ := base.OmniCandidateType(c.Type)
		if typ == base.CandidateTypeKeyword && cCtx.Scene == base.SceneTypeQuery && isDMLKeyword(c.Text) {
			return base.Candidate{}, false
		}
		return base.Candidate{
			Type: typ,
			Text: c.Text,
		}, true
	}, base.CompareCandidateByTypeText), nil
}

// lineOffsetToBytePos converts a 1-based line number and 0-based
// character (rune) offset to a 0-based byte position in s.
func lineOffsetToBytePos(s string, line, offset int) int {
	pos := 0
	currentLine := 1
	for pos < len(s) && currentLine < line {
		if s[pos] == '\n' {
			currentLine++
		}
		pos++
	}
	for i := 0; i < offset && pos < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[pos:])
		pos += size
	}
	if pos > len(s) {
		return len(s)
	}
	return pos
}

func isDMLKeyword(w string) bool {
	switch strings.ToUpper(w) {
	case "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "REPLACE",
		"UPSERT", "REMOVE", "SET":
		return true
	}
	return false
}

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	return base.ConvertParseError(statement, err, basePos, func(pe *parser.ParseError) (int, string, string) {
		return pe.Position, pe.Error(), pe.Message
	})
}
