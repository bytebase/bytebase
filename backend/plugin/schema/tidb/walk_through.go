package tidb

import (
	"context"
	"strings"

	omnianalysis "github.com/bytebase/omni/analysis"
	metadatapb "github.com/bytebase/omni/metadata"
	tidbanalysis "github.com/bytebase/omni/tidb/analysis"
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterWalkThrough(storepb.Engine_TIDB, WalkThrough)
}

var tidbWalkThroughEngine = omnicatalog.WalkThroughEngine[catalog.Catalog]{
	Engine: storepb.Engine_TIDB,
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
	InspectAST: inspectTiDBAST,
	ExtractCatalogError: omnicatalog.ExtractError(func(e *catalog.Error) (int, string) {
		return e.Code, e.Message
	}),
	TrackBatchTables: true,
	TiDBErrorDialect: true,
	CaseSensitiveResult: func(d *model.DatabaseMetadata) bool {
		return d.GetIsObjectCaseSensitive()
	},
}

// WalkThrough simulates the DDL in asts against the database state d using the
// omni TiDB catalog, then replaces d with the resulting schema so downstream
// advisor rules see the post-DDL metadata.
func WalkThrough(d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
	rawSQL, execASTs := tidbBuildRawSQL(asts)
	return omnicatalog.RunWalkThrough(context.Background(), tidbWalkThroughEngine, rawSQL, d, asts, execASTs)
}

func execCatalog(c *catalog.Catalog, sql string, continueOnError bool) ([]omnicatalog.ExecResult, error) {
	var opts *catalog.ExecOptions
	if continueOnError {
		opts = &catalog.ExecOptions{ContinueOnError: true}
	}
	results, err := c.Exec(sql, opts)
	return omnicatalog.MapExecResults(results, err, func(r catalog.ExecResult) omnicatalog.ExecResult {
		return omnicatalog.ExecResult{Index: r.Index, Line: r.Line, Error: r.Error}
	})
}

// tidbBuildRawSQL rebuilds the original SQL from the omni ASTs. Each statement
// keeps its own text; they are re-joined with a terminator so catalog.Exec
// re-parses them as a statement list. The returned slice is parallel to that
// list: execASTs[i] is the base.AST that produced the i-th executed statement,
// so a simulation error's index maps back to the submitted statement's
// position.
func tidbBuildRawSQL(asts []base.AST) (string, []base.AST) {
	var b strings.Builder
	execASTs := make([]base.AST, 0, len(asts))
	for _, a := range asts {
		wrapped, ok := a.(*tidbparser.AST)
		if !ok {
			continue
		}
		text := strings.TrimSpace(wrapped.Text)
		if text == "" {
			continue
		}
		b.WriteString(strings.TrimRight(text, ";"))
		b.WriteString(";\n")
		execASTs = append(execASTs, a)
	}
	return b.String(), execASTs
}

func inspectTiDBAST(unifiedAST base.AST) (omnianalysis.WalkThroughStatement, bool) {
	node, ok := tidbparser.GetNode(unifiedAST)
	if !ok {
		return omnianalysis.WalkThroughStatement{}, false
	}
	info := tidbanalysis.ClassifyWalkThroughStatement(node)
	if wrapped, ok := unifiedAST.(*tidbparser.AST); ok {
		info.Source = strings.TrimSpace(wrapped.Text)
	}
	return info, true
}
