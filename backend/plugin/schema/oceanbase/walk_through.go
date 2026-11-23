package oceanbase

import (
	"context"
	"errors"
	"strings"

	omnianalysis "github.com/bytebase/omni/analysis"
	metadatapb "github.com/bytebase/omni/metadata"
	oceanbaseanalysis "github.com/bytebase/omni/oceanbase/analysis"
	"github.com/bytebase/omni/oceanbase/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterWalkThroughWithContext(storepb.Engine_OCEANBASE, WalkThroughWithContext)
}

var oceanBaseWalkThroughEngine = omnicatalog.WalkThroughEngine[catalog.Catalog]{
	Engine: storepb.Engine_OCEANBASE,
	New:    catalog.New,
	Load: func(ctx context.Context, c *catalog.Catalog, meta *metadatapb.DatabaseSchemaMetadata) error {
		_, err := c.LoadMetadata(ctx, meta)
		return err
	},
	Exec: func(c *catalog.Catalog, sql string) ([]omnicatalog.ExecResult, error) {
		return execCatalog(c, sql, true)
	},
	DatabaseMetadata: func(c *catalog.Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata {
		return c.DatabaseMetadata(dbName)
	},
	InspectAST:          inspectOceanBaseAST,
	ExtractCatalogError: extractOceanBaseCatalogError,
	SplitSQL:            oceanbaseparser.SplitSQL,
	IsCTAS:              oceanbaseanalysis.IsCreateTableAs,
}

// WalkThroughWithContext performs DDL simulation using the omni OceanBase catalog.
func WalkThroughWithContext(ctx context.Context, wtCtx schema.WalkThroughContext, d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
	return omnicatalog.RunWalkThrough(ctx, oceanBaseWalkThroughEngine, wtCtx.RawSQL, d, asts, nil)
}

func execCatalog(c *catalog.Catalog, sql string, continueOnError bool) ([]omnicatalog.ExecResult, error) {
	var opts *catalog.ExecOptions
	if continueOnError {
		opts = &catalog.ExecOptions{ContinueOnError: true}
	}
	results, err := c.Exec(sql, opts)
	if err != nil {
		return nil, err
	}
	out := make([]omnicatalog.ExecResult, len(results))
	for i, r := range results {
		out[i] = omnicatalog.ExecResult{Index: r.Index, Line: r.Line, Error: r.Error}
	}
	return out, nil
}

func extractOceanBaseCatalogError(err error) (int, string, bool) {
	var catErr *catalog.Error
	if errors.As(err, &catErr) {
		return catErr.Code, catErr.Message, true
	}
	return 0, "", false
}

func inspectOceanBaseAST(unifiedAST base.AST) (omnianalysis.WalkThroughStatement, bool) {
	node, ok := oceanbaseparser.GetNode(unifiedAST)
	if !ok {
		return omnianalysis.WalkThroughStatement{}, false
	}
	info := oceanbaseanalysis.ClassifyWalkThroughStatement(node)
	if source, ok := unifiedAST.(interface{ ASTSource() string }); ok {
		info.Source = strings.TrimSpace(source.ASTSource())
	}
	return info, true
}
