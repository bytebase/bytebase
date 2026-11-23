// Package mariadb registers MariaDB's DDL simulation, schema definition/metadata
// conversions, and metadata diff on omni's native MariaDB catalog.
package mariadb

import (
	"context"
	"errors"

	mariadbanalysis "github.com/bytebase/omni/mariadb/analysis"
	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterWalkThroughWithContext(storepb.Engine_MARIADB, WalkThroughWithContext)
}

var mariaDBWalkThroughEngine = omnicatalog.WalkThroughEngine[catalog.Catalog]{
	Engine: storepb.Engine_MARIADB,
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
	InspectAST:          omnicatalog.InspectAST(mariadbparser.GetNode, mariadbanalysis.ClassifyWalkThroughStatement),
	ExtractCatalogError: extractMariaDBCatalogError,
	SplitSQL:            mariadbparser.SplitSQL,
	IsCTAS:              mariadbanalysis.IsCreateTableAs,
}

// WalkThroughWithContext performs DDL simulation using the omni MariaDB catalog.
func WalkThroughWithContext(ctx context.Context, wtCtx schema.WalkThroughContext, d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
	return omnicatalog.RunWalkThrough(ctx, mariaDBWalkThroughEngine, wtCtx.RawSQL, d, asts, nil)
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

func extractMariaDBCatalogError(err error) (int, string, bool) {
	var catErr *catalog.Error
	if errors.As(err, &catErr) {
		return catErr.Code, catErr.Message, true
	}
	return 0, "", false
}
