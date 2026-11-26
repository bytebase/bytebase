package mongodb

import (
	"context"
	"strings"

	"github.com/bytebase/omni/mongo"
	"github.com/bytebase/omni/mongo/analysis"
	"github.com/bytebase/omni/mongo/ast"
	omnicompletion "github.com/bytebase/omni/mongo/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine)
	engine.Split = SplitSQL
	engine.Parse = parseMongoShell
	engine.Complete = Completion
	engine.QuerySpan = GetQuerySpan
	base.Register(engine, storepb.Engine_MONGODB)
}

// AST wraps an omni MongoDB AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits the input into multiple MongoDB statements.
// Unlike SQL-based engines that split on semicolons, MongoDB statements can be
// separated by newlines without semicolons, so we use the parser to identify
// statement boundaries.
func SplitSQL(statement string) ([]base.Statement, error) {
	stmts, err := parseMongoShell(statement)
	if err != nil {
		return nil, err
	}

	if len(stmts) == 0 {
		if len(strings.TrimSpace(statement)) == 0 {
			return nil, nil
		}
		return []base.Statement{{Text: statement}}, nil
	}

	var result []base.Statement
	for _, ps := range stmts {
		result = append(result, ps.Statement)
	}
	return result, nil
}

// parseMongoShell parses a MongoDB shell script strictly so a parse error in any
// statement fails the whole input instead of silently dropping statements (BYT-9950).
func parseMongoShell(statement string) ([]base.ParsedStatement, error) {
	stmts, err := mongo.Parse(statement)
	if err != nil {
		return nil, err
	}
	return convertStatements(statement, stmts), nil
}

func convertStatements(input string, stmts []mongo.Statement) []base.ParsedStatement {
	var result []base.ParsedStatement
	positionMapper := base.NewByteOffsetPositionMapper(input)
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		baseStmt := base.NewExactStatement(input, positionMapper, stmt.ByteStart, stmt.ByteEnd, false)
		result = append(result, base.ParsedStatement{
			Statement: baseStmt,
			AST: &AST{
				Node:          stmt.AST,
				StartPosition: baseStmt.Start,
			},
		})
	}
	return result
}

// Completion is the entry point for MongoDB code completion.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDatabase, _ := base.CompletionMetadataLoaders(cCtx)

	candidates := omnicompletion.CompleteSession(ctx, omnicompletion.SessionConfig{
		DefaultDatabase: cCtx.DefaultDatabase,
		LoadDatabase:    loadDatabase,
	}, statement, caretLine, caretOffset)

	return base.DeduplicateCandidates(candidates, func(c omnicompletion.Candidate) (base.Candidate, bool) {
		typ := base.OmniCandidateType(c.Type)
		if typ == base.CandidateTypeNone {
			typ = base.CandidateTypeFunction
		}
		return base.Candidate{
			Type:       typ,
			Text:       c.Text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}, true
	}, base.CompareCandidate), nil
}

// GetQuerySpan returns a QuerySpan for a MongoDB shell statement, classifying
// the statement type and populating PredicatePaths from masking analysis.
func GetQuerySpan(_ context.Context, _ base.GetQuerySpanContext, stmt base.Statement, _ string, _ string, _ bool) (*base.QuerySpan, error) {
	span := &base.QuerySpan{
		Type: getQueryType(stmt.Text),
	}

	masking, err := analyzeMaskingStatement(stmt.Text)
	if err != nil {
		return span, err
	}
	if masking == nil {
		return span, nil
	}

	span.MongoDBAnalysis = masking

	if len(masking.PredicateFields) > 0 {
		span.PredicatePaths = make(map[string]*base.PathAST, len(masking.PredicateFields))
		for _, field := range masking.PredicateFields {
			span.PredicatePaths[field] = dotPathToPathAST(field)
		}
	}

	return span, nil
}

// getQueryType parses a MongoDB shell statement and returns its query type.
func getQueryType(statement string) base.QueryType {
	qt, _ := analysis.ClassifyStatement(statement)
	return base.OmniQueryType(qt)
}

// dotPathToPathAST converts a dot-delimited field path (e.g. "contact.phone")
// into a PathAST with ItemSelector nodes.
func dotPathToPathAST(dotPath string) *base.PathAST {
	parts := strings.Split(dotPath, ".")
	nodes := make([]base.SelectorNode, len(parts))
	for i, part := range parts {
		nodes[i] = base.NewItemSelector(part)
	}
	return base.BuildPathAST(nodes)
}
