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
		if resolver, ok := engine.(omnianalysis.QuerySpanResolver); ok {
			e.QuerySpans = func(ctx context.Context, gCtx GetQuerySpanContext, statements []Statement, database, schema string, ignoreCaseSensitive bool) ([]*QuerySpan, error) {
				return omniQuerySpans(ctx, resolver, gCtx, statements, database, schema, ignoreCaseSensitive)
			}
		}
		if completer, ok := engine.(omnianalysis.Completer); ok {
			e.Complete = func(ctx context.Context, cCtx CompletionContext, statement string, caretLine int, caretOffset int) ([]Candidate, error) {
				return omniComplete(ctx, completer, cCtx, statement, caretLine, caretOffset), nil
			}
		}
	}
	return e
}

// OmniError converts an omni *SyntaxError into the registry's, keeping the
// message of any error that wraps it; any other error is returned as is.
func OmniError(err error) error {
	var syntaxErr *omnianalysis.SyntaxError
	if !errors.As(err, &syntaxErr) {
		return err
	}
	converted := &SyntaxError{
		Position:   omniPosition(syntaxErr.Position),
		Message:    syntaxErr.Message,
		RawMessage: syntaxErr.RawMessage,
	}
	if err == error(syntaxErr) {
		return converted
	}
	return &wrappedError{message: err.Error(), err: converted}
}

// wrappedError is an omni error's message around its converted cause.
type wrappedError struct {
	message string
	err     error
}

func (e *wrappedError) Error() string { return e.message }
func (e *wrappedError) Unwrap() error { return e.err }

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

func omniQuerySpans(ctx context.Context, resolver omnianalysis.QuerySpanResolver, gCtx GetQuerySpanContext, statements []Statement, database, schema string, ignoreCaseSensitive bool) ([]*QuerySpan, error) {
	if len(statements) == 0 {
		return nil, nil
	}
	omniStatements := make([]omnianalysis.QuerySpanStatement, 0, len(statements))
	for _, stmt := range statements {
		start := omnianalysis.Position{Line: 1, Column: 1}
		if stmt.Start != nil {
			start = omnianalysis.Position{Line: int(stmt.Start.Line), Column: int(stmt.Start.Column)}
		}
		omniStatements = append(omniStatements, omnianalysis.QuerySpanStatement{Text: stmt.Text, Start: start})
	}
	spans, err := resolver.QuerySpans(ctx, omnianalysis.QuerySpanRequest{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		Catalog:             omniSpanCatalog(gCtx),
	}, omniStatements)
	if err != nil {
		return nil, omniSpanError(err)
	}
	result := make([]*QuerySpan, 0, len(spans))
	for _, span := range spans {
		result = append(result, OmniResolvedSpan(span))
	}
	return result, nil
}

// omniSpanCatalog gives omni the instance's synced databases for one script,
// each read once. A Bytebase not-found error reaches omni as its own.
func omniSpanCatalog(gCtx GetQuerySpanContext) omnianalysis.Catalog {
	catalog := omniCatalog(gCtx.InstanceID, gCtx.GetDatabaseMetadataFunc, gCtx.ListDatabaseNamesFunc)
	if list := catalog.ListDatabaseNames; list != nil {
		catalog.ListDatabaseNames = func(ctx context.Context) ([]string, error) {
			names, err := list(ctx)
			return names, ToOmniNotFoundError(err)
		}
	}
	if get := catalog.GetDatabase; get != nil {
		databases := map[string]*omnianalysis.Database{}
		catalog.GetDatabase = func(ctx context.Context, name string) (*omnianalysis.Database, error) {
			if database, ok := databases[name]; ok {
				return database, nil
			}
			database, err := get(ctx, name)
			if err != nil {
				return nil, ToOmniNotFoundError(err)
			}
			databases[name] = database
			return database, nil
		}
	}
	if getLinked := gCtx.GetLinkedDatabaseMetadataFunc; getLinked != nil {
		catalog.GetLinkedDatabase = func(ctx context.Context, link, schema string) (*omnianalysis.Database, error) {
			_, name, database, err := getLinked(ctx, gCtx.InstanceID, link, schema)
			if err != nil || database == nil {
				return nil, ToOmniNotFoundError(err)
			}
			return &omnianalysis.Database{
				Name:                name,
				Metadata:            database.GetProto(),
				ObjectCaseSensitive: database.GetIsObjectCaseSensitive(),
			}, nil
		}
	}
	return catalog
}

// omniSpanError converts the errors an omni span resolution returns into the
// registry's.
func omniSpanError(err error) error {
	if mixErr := ConvertOmniMixTablesError(err); mixErr != nil {
		return mixErr
	}
	var notFound *omnianalysis.ResourceNotFoundError
	if errors.As(err, &notFound) {
		return OmniResourceNotFoundError(notFound)
	}
	var typeNotSupported *omnianalysis.TypeNotSupportedError
	if errors.As(err, &typeNotSupported) {
		return OmniTypeNotSupportedError(typeNotSupported)
	}
	return OmniError(err)
}
