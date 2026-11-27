package base

import (
	"context"
	"errors"

	lsp "github.com/bytebase/lsp-protocol"
	omnianalysis "github.com/bytebase/omni/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// OmniEngine converts the capabilities of omni engine values (an engine's
// analysis Engine, its completion Engine) into the registry's functions. Each
// capability none of them implements stays nil.
func OmniEngine(engines ...any) Engine {
	var e Engine
	for _, engine := range engines {
		if splitter, ok := engine.(omnianalysis.Splitter); ok {
			e.Split = func(script string) ([]Statement, error) {
				return omniSplit(script, splitter), nil
			}
		}
		if parser, ok := engine.(omnianalysis.Parser); ok {
			e.Parse = func(script string) ([]ParsedStatement, error) {
				return omniParse(script, parser)
			}
		}
		if classifier, ok := engine.(omnianalysis.StatementClassifier); ok {
			e.StatementTypes = func(asts []AST) ([]storepb.StatementType, error) {
				typed, err := OmniStatementTypesWithPosition(classifier, asts)
				if err != nil {
					return nil, err
				}
				return ExtractTypesFromPositions(typed), nil
			}
		}
		if ranger, ok := engine.(omnianalysis.StatementRanger); ok {
			e.StatementRanges = func(_ context.Context, _ StatementRangeContext, script string) ([]Range, error) {
				return omniLSPRanges(ranger.StatementRanges(script)), nil
			}
		}
		if diagnoser, ok := engine.(omnianalysis.Diagnoser); ok {
			e.Diagnose = func(_ context.Context, _ DiagnoseContext, script string) ([]Diagnostic, error) {
				return omniDiagnostics(script, diagnoser.Diagnose(script)), nil
			}
		}
		if validator, ok := engine.(omnianalysis.QueryValidator); ok {
			e.ValidateQuery = func(script string) (bool, bool, error) {
				readOnly, returnsData, err := validator.ValidateQuery(script)
				return readOnly, returnsData, OmniError(err)
			}
		}
		if explainer, ok := engine.(omnianalysis.Explainer); ok {
			e.Explain = func(statement string, format ExplainFormat) (string, error) {
				explain, err := explainer.Explain(statement, omnianalysis.ExplainFormat(format))
				return ExplainStatementResult(explain, err, OmniError)
			}
		}
		if limiter, ok := engine.(omnianalysis.ResultLimiter); ok {
			e.ResultLimit = limiter.ResultLimit
		}
		if completer, ok := engine.(omnianalysis.Completer); ok {
			e.Complete = func(ctx context.Context, cCtx CompletionContext, statement string, caretLine int, caretOffset int) ([]Candidate, error) {
				return omniComplete(ctx, completer, cCtx, statement, caretLine, caretOffset), nil
			}
		}
	}
	return e
}

// OmniError converts an omni *SyntaxError into the registry's; any other error
// is returned as is.
func OmniError(err error) error {
	var syntaxErr *omnianalysis.SyntaxError
	if !errors.As(err, &syntaxErr) {
		return err
	}
	return &SyntaxError{
		Position:   omniPosition(syntaxErr.Position),
		Message:    syntaxErr.Message,
		RawMessage: syntaxErr.RawMessage,
	}
}

// OmniStatementTypesWithPosition classifies each parsed statement, keeping the
// line it starts on and its text.
func OmniStatementTypesWithPosition(classifier omnianalysis.StatementClassifier, asts []AST) ([]StatementTypeWithPosition, error) {
	return CollectStatementTypesWithPosition(asts, func(a AST) ([]storepb.StatementType, string, bool, error) {
		parsed, ok := a.(*NodeAST[any])
		if !ok {
			return nil, "", false, errors.New("expected an AST from an omni engine")
		}
		types := classifier.StatementTypes(omnianalysis.ParsedStatement{
			Statement: omnianalysis.Statement{Text: parsed.Text},
			Node:      parsed.Node,
		})
		if len(types) == 0 {
			return nil, "", false, nil
		}
		return OmniStatementTypes(types), parsed.Text, true, nil
	})
}

func omniSplit(script string, splitter omnianalysis.Splitter) []Statement {
	return BuildStatements(script, splitter.Split(script), func(s omnianalysis.Statement) (int, int, bool) {
		return s.Start, s.End, s.Empty
	}, false)
}

func omniParse(script string, parser omnianalysis.Parser) ([]ParsedStatement, error) {
	parsed, err := parser.Parse(script)
	if err != nil {
		return nil, OmniError(err)
	}
	if parsed == nil {
		return nil, nil
	}
	mapper := NewByteOffsetPositionMapper(script)
	result := make([]ParsedStatement, 0, len(parsed))
	for _, p := range parsed {
		stmt := NewExactStatement(script, mapper, p.Start, p.End, p.Empty)
		ps := ParsedStatement{Statement: stmt}
		if !p.Empty {
			ps.AST = &NodeAST[any]{Node: p.Node, Text: stmt.Text, StartPosition: stmt.Start}
		}
		result = append(result, ps)
	}
	return result, nil
}

func omniLSPRanges(ranges []omnianalysis.LSPRange) []Range {
	if ranges == nil {
		return nil
	}
	result := make([]Range, 0, len(ranges))
	for _, r := range ranges {
		result = append(result, lsp.Range{
			Start: lsp.Position{Line: r.Start.Line, Character: r.Start.Character},
			End:   lsp.Position{Line: r.End.Line, Character: r.End.Character},
		})
	}
	return result
}

func omniDiagnostics(script string, diagnostics []omnianalysis.Diagnostic) []Diagnostic {
	if diagnostics == nil {
		return nil
	}
	result := make([]Diagnostic, 0, len(diagnostics))
	for _, d := range diagnostics {
		start := *convertPositionToUTF16Position(omniPosition(d.Start), script)
		end := start
		end.Character++
		if d.End != (omnianalysis.Position{}) {
			end = *convertPositionToUTF16Position(omniPosition(d.End), script)
		}
		result = append(result, Diagnostic{
			Range:    lsp.Range{Start: start, End: end},
			Severity: lsp.DiagnosticSeverity(d.Severity),
			Source:   "Syntax check",
			Message:  d.Message,
		})
	}
	return result
}

func omniPosition(p omnianalysis.Position) *storepb.Position {
	return &storepb.Position{Line: safeIntToInt32(p.Line), Column: safeIntToInt32(p.Column)}
}

func omniComplete(ctx context.Context, completer omnianalysis.Completer, cCtx CompletionContext, statement string, caretLine, caretOffset int) []Candidate {
	candidates := completer.Complete(ctx, omnianalysis.CompletionRequest{
		ReadOnly:        cCtx.Scene == SceneTypeQuery,
		DefaultDatabase: cCtx.DefaultDatabase,
		DefaultSchema:   cCtx.DefaultSchema,
		Catalog:         omniCatalog(cCtx.InstanceID, cCtx.Metadata, cCtx.ListDatabaseNames),
	}, statement, caretLine, caretOffset)
	return ConvertOmniCandidates(candidates, func(c omnianalysis.Candidate) (string, omnianalysis.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, CompareCandidate)
}

// omniCatalog gives omni the instance's synced databases.
func omniCatalog(instanceID string, getDatabase GetDatabaseMetadataFunc, listDatabaseNames ListDatabaseNamesFunc) omnianalysis.Catalog {
	var catalog omnianalysis.Catalog
	if listDatabaseNames != nil {
		catalog.ListDatabaseNames = func(ctx context.Context) ([]string, error) {
			return listDatabaseNames(ctx, instanceID)
		}
	}
	if getDatabase != nil {
		catalog.GetDatabase = func(ctx context.Context, name string) (*omnianalysis.Database, error) {
			resolved, database, err := getDatabase(ctx, instanceID, name)
			if err != nil || database == nil {
				return nil, err
			}
			return &omnianalysis.Database{
				Name:                resolved,
				Metadata:            database.GetProto(),
				ObjectCaseSensitive: database.GetIsObjectCaseSensitive(),
			}, nil
		}
	}
	return catalog
}
