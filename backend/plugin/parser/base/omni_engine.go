package base

import (
	"context"
	"errors"

	lsp "github.com/bytebase/lsp-protocol"
	omnianalysis "github.com/bytebase/omni/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
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
		if extractor, ok := engine.(omnianalysis.ChangeExtractor); ok {
			e.ExtractChanged = func(database, schema string, dbMetadata *model.DatabaseMetadata, asts []AST, script string) (*ChangeSummary, error) {
				return omniChanges(extractor, database, schema, dbMetadata, asts, script)
			}
		}
		if generator, ok := engine.(omnianalysis.BackupGenerator); ok {
			e.TransformDMLToSelect = func(ctx context.Context, tCtx TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]BackupStatement, error) {
				return omniBackups(ctx, generator, tCtx, statement, sourceDatabase, targetDatabase, tablePrefix)
			}
		}
		if generator, ok := engine.(omnianalysis.RestoreGenerator); ok {
			e.GenerateRestoreSQL = func(ctx context.Context, rCtx RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
				return omniRestoreSQL(ctx, generator, rCtx, statement, backupItem)
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
		Instance:            gCtx.InstanceID,
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
	if getDatabase := gCtx.GetDatabaseMetadataFunc; getDatabase != nil {
		catalog.GetInstanceDatabase = func(ctx context.Context, instance, name string) (*omnianalysis.Database, error) {
			resolved, database, err := getDatabase(ctx, instance, name)
			if err != nil || database == nil {
				return nil, ToOmniNotFoundError(err)
			}
			return &omnianalysis.Database{
				Name:                resolved,
				Metadata:            database.GetProto(),
				ObjectCaseSensitive: database.GetIsObjectCaseSensitive(),
			}, nil
		}
	}
	if getLinked := gCtx.GetLinkedDatabaseMetadataFunc; getLinked != nil {
		catalog.GetLinkedDatabase = func(ctx context.Context, instance, link, schema string) (*omnianalysis.Database, error) {
			linkedInstance, name, database, err := getLinked(ctx, instance, link, schema)
			if err != nil || database == nil {
				return nil, ToOmniNotFoundError(err)
			}
			return &omnianalysis.Database{
				Name:                name,
				Metadata:            database.GetProto(),
				ObjectCaseSensitive: database.GetIsObjectCaseSensitive(),
				Instance:            linkedInstance,
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

// OmniParsedStatements converts parsed ASTs into the statements an omni engine
// reads. An AST that wraps no omni node reaches the engine as is, which rejects
// it as a foreign node.
func OmniParsedStatements(asts []AST) []omnianalysis.ParsedStatement {
	statements := make([]omnianalysis.ParsedStatement, 0, len(asts))
	for _, a := range asts {
		var node any = a
		if parsed, ok := a.(interface{ ASTNode() any }); ok {
			node = parsed.ASTNode()
		}
		statements = append(statements, omnianalysis.ParsedStatement{Statement: omnianalysis.Statement{Text: ASTSource(a)}, Node: node})
	}
	return statements
}

func omniChanges(extractor omnianalysis.ChangeExtractor, database, schema string, dbMetadata *model.DatabaseMetadata, asts []AST, script string) (*ChangeSummary, error) {
	statements := OmniParsedStatements(asts)
	request := omnianalysis.ChangeRequest{DefaultDatabase: database, DefaultSchema: schema, Script: script}
	if dbMetadata != nil {
		request.Database = &omnianalysis.Database{
			Name:                dbMetadata.DatabaseName(),
			Metadata:            dbMetadata.GetProto(),
			ObjectCaseSensitive: dbMetadata.GetIsObjectCaseSensitive(),
		}
	}
	changes, err := extractor.ExtractChanges(request, statements)
	if err != nil {
		return nil, err
	}
	summary := &ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
		DMLCount:         changes.DMLCount,
		InsertCount:      changes.InsertCount,
	}
	for _, resource := range changes.Resources {
		if resource.DatabaseOnly {
			summary.ChangedResources.AddDatabase(resource.Database)
			continue
		}
		summary.ChangedResources.AddTable(resource.Database, resource.Schema, &storepb.ChangedResourceTable{Name: resource.Table}, resource.AffectsData)
	}
	for _, dml := range changes.DMLStatements {
		text := dml.Text
		if dml.SearchPath != nil {
			text = WithSearchPath(text, dml.SearchPath)
		}
		summary.DMLStatements = append(summary.DMLStatements, text)
	}
	return summary, nil
}

func omniBackups(ctx context.Context, generator omnianalysis.BackupGenerator, tCtx TransformContext, statement, sourceDatabase, targetDatabase, tablePrefix string) ([]BackupStatement, error) {
	backups, err := generator.Backups(ctx, omnianalysis.BackupRequest{
		DefaultDatabase:      tCtx.DatabaseName,
		SourceDatabase:       sourceDatabase,
		BackupDatabase:       targetDatabase,
		TablePrefix:          tablePrefix,
		CaseSensitive:        tCtx.IsCaseSensitive,
		Version:              tCtx.Version,
		Catalog:              omniCatalog(tCtx.InstanceID, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc),
		ReferencingDatabases: tCtx.ReferencingDatabases(ctx),
	}, statement)
	if err != nil || len(backups) == 0 {
		return nil, err
	}
	result := make([]BackupStatement, 0, len(backups))
	for _, backup := range backups {
		result = append(result, BackupStatement{
			Statement:       backup.Statement,
			SourceSchema:    backup.SourceSchema,
			SourceTableName: backup.SourceTable,
			TargetTableName: backup.BackupTable,
			StartPosition:   omniBackupPosition(backup.Start),
			EndPosition:     omniBackupPosition(backup.End),
		})
	}
	return result, nil
}

// omniBackupPosition is a backup's recorded position, nil when the engine could
// not place the statement.
func omniBackupPosition(p omnianalysis.Position) *storepb.Position {
	if p == (omnianalysis.Position{}) {
		return nil
	}
	return omniPosition(p)
}

func omniRestoreSQL(ctx context.Context, generator omnianalysis.RestoreGenerator, rCtx RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if err := ValidateBackupItem(backupItem); err != nil {
		return "", err
	}
	sourceDatabase, backupDatabase, err := ParseBackupItemDatabases(backupItem)
	if err != nil {
		return "", err
	}
	return generator.RestoreSQL(ctx, omnianalysis.RestoreRequest{
		Script:         statement,
		SourceDatabase: sourceDatabase,
		SourceSchema:   backupItem.SourceTable.Schema,
		SourceTable:    backupItem.SourceTable.Table,
		BackupDatabase: backupDatabase,
		BackupSchema:   backupItem.TargetTable.Schema,
		BackupTable:    backupItem.TargetTable.Table,
		Start:          omnianalysis.Position{Line: int(backupItem.StartPosition.GetLine()), Column: int(backupItem.StartPosition.GetColumn())},
		End:            omnianalysis.Position{Line: int(backupItem.EndPosition.GetLine()), Column: int(backupItem.EndPosition.GetColumn())},
		CaseSensitive:  rCtx.IsCaseSensitive,
		Catalog:        omniCatalog(rCtx.InstanceID, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc),
	})
}
