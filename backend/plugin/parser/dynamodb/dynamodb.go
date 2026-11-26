package dynamodb

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/bytebase/omni/dynamodb/analysis"
	"github.com/bytebase/omni/dynamodb/ast"
	"github.com/bytebase/omni/dynamodb/catalog"
	"github.com/bytebase/omni/dynamodb/completion"
	"github.com/bytebase/omni/dynamodb/parser"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine)
	engine.Parse = parsePartiQLStatements
	engine.Complete = Completion
	base.Register(engine, storepb.Engine_DYNAMODB)
}

// astNode wraps an omni AST node to implement the base.AST interface.
type astNode = base.NodeAST[ast.Node]

// parsePartiQLStatements is the ParseStatementsFunc for PartiQL (DynamoDB).
// Returns []ParsedStatement with both text and AST populated.
func parsePartiQLStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_DYNAMODB, statement)
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
