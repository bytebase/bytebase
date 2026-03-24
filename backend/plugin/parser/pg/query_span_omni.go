package pg

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// omniQuerySpanExtractor extracts query span information using the omni parser
// and catalog, replacing the ANTLR-based querySpanExtractor.
type omniQuerySpanExtractor struct {
	ctx             context.Context
	gCtx            base.GetQuerySpanContext
	defaultDatabase string
	searchPath      []string
	// metaCache is a lazy-load cache for database metadata.
	// Use getDatabaseMetadata() instead of accessing directly.
	metaCache map[string]*model.DatabaseMetadata
	cat       *catalog.Catalog
}

// newOmniQuerySpanExtractor creates a new omni-based query span extractor.
func newOmniQuerySpanExtractor(defaultDatabase string, searchPath []string, gCtx base.GetQuerySpanContext) *omniQuerySpanExtractor {
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}
	return &omniQuerySpanExtractor{
		defaultDatabase: defaultDatabase,
		searchPath:      searchPath,
		gCtx:            gCtx,
		metaCache:       make(map[string]*model.DatabaseMetadata),
	}
}

// getDatabaseMetadata returns cached database metadata, fetching it if not yet cached.
func (e *omniQuerySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if meta, ok := e.metaCache[database]; ok {
		return meta, nil
	}
	_, meta, err := e.gCtx.GetDatabaseMetadataFunc(e.ctx, e.gCtx.InstanceID, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}
	e.metaCache[database] = meta
	return meta, nil
}

// initCatalog initializes the omni catalog from the database metadata.
// It generates DDL from the metadata, creates a new catalog, sets the search path,
// and loads the schema.
func (e *omniQuerySpanExtractor) initCatalog() error {
	meta, err := e.getDatabaseMetadata(e.defaultDatabase)
	if err != nil {
		return errors.Wrapf(err, "failed to get database metadata for catalog init")
	}
	if meta == nil {
		// No metadata available; create an empty catalog.
		e.cat = catalog.New()
		e.cat.SetSearchPath(e.searchPath)
		return nil
	}

	schemaDDL, err := schema.GetDatabaseDefinition(
		storepb.Engine_POSTGRES,
		schema.GetDefinitionContext{},
		meta.GetProto(),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to generate schema DDL")
	}

	e.cat = catalog.New()
	e.cat.SetSearchPath(e.searchPath)

	if schemaDDL != "" {
		if _, err := e.cat.Exec(schemaDDL, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
			return errors.Wrapf(err, "failed to load schema DDL into catalog")
		}
	}

	return nil
}

// getQuerySpan extracts the query span for the given SQL statement.
func (e *omniQuerySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	e.ctx = ctx

	// Step 1: Parse with omni.
	omniStmts, err := ParsePg(stmt)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement")
	}
	if len(omniStmts) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(omniStmts))
	}

	// Step 2: Extract access tables.
	accessTables, err := ExtractAccessTables(stmt, ExtractAccessTablesOption{
		DefaultDatabase:        e.defaultDatabase,
		DefaultSchema:          e.searchPath[0],
		GetDatabaseMetadata:    e.gCtx.GetDatabaseMetadataFunc,
		Ctx:                    ctx,
		InstanceID:             e.gCtx.InstanceID,
		SkipMetadataValidation: false,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract access tables")
	}

	// Build access map.
	accessesMap := make(base.SourceColumnSet)
	for _, resource := range accessTables {
		accessesMap[resource] = true
	}

	// Step 3: Check for mixed system/user tables.
	allSystems, mixed := isMixedQuery(accessesMap)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	// Step 4: Classify query type.
	queryType, isExplainAnalyze := classifyQueryType(omniStmts[0].AST, allSystems)

	// Step 5: Return early for non-SELECT queries.
	if queryType != base.Select {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// For EXPLAIN ANALYZE SELECT, return with source columns but no results.
	if isExplainAnalyze {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Step 6: Cast to SelectStmt, init catalog, analyze.
	selStmt, ok := omniStmts[0].AST.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("expected SelectStmt for SELECT query, got %T", omniStmts[0].AST)
	}

	if err := e.initCatalog(); err != nil {
		return nil, errors.Wrapf(err, "failed to initialize catalog")
	}

	query, err := e.cat.AnalyzeSelectStmt(selStmt)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to analyze select statement")
	}

	// Step 7: Extract lineage and source columns (stubs for now).
	results := e.extractLineage(query)
	sourceColumns := e.extractAllSourceColumns(query)

	// Merge extracted source columns with access tables.
	for col := range sourceColumns {
		accessesMap[col] = true
	}

	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: accessesMap,
		Results:       results,
	}, nil
}

// extractLineage extracts the result column lineage from an analyzed query.
// This is a stub that will be implemented in a later task.
func (*omniQuerySpanExtractor) extractLineage(_ *catalog.Query) []base.QuerySpanResult {
	return []base.QuerySpanResult{}
}

// extractAllSourceColumns extracts all source columns referenced by the query.
// This is a stub that will be implemented in a later task.
func (*omniQuerySpanExtractor) extractAllSourceColumns(_ *catalog.Query) base.SourceColumnSet {
	return base.SourceColumnSet{}
}
