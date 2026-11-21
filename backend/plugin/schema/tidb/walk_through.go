package tidb

import (
	"context"
	"errors"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/tidb/ast"
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
	InspectAST:          inspectTiDBAST,
	ExtractCatalogError: extractTiDBCatalogError,
	TrackBatchTables:    true,
	TiDBErrorDialect:    true,
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
	if err != nil {
		return nil, err
	}
	out := make([]omnicatalog.ExecResult, len(results))
	for i, r := range results {
		out[i] = omnicatalog.ExecResult{Index: r.Index, Line: r.Line, Error: r.Error}
	}
	return out, nil
}

func extractTiDBCatalogError(err error) (int, string, bool) {
	var catErr *catalog.Error
	if errors.As(err, &catErr) {
		return catErr.Code, catErr.Message, true
	}
	return 0, "", false
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

func inspectTiDBAST(unifiedAST base.AST) (omnicatalog.StatementInfo, bool) {
	node, ok := tidbparser.GetNode(unifiedAST)
	if !ok {
		return omnicatalog.StatementInfo{}, false
	}
	info := omnicatalog.StatementInfo{}
	if wrapped, ok := unifiedAST.(*tidbparser.AST); ok {
		info.Source = strings.TrimSpace(wrapped.Text)
	}
	toRef := func(t *ast.TableRef) omnicatalog.TableRef {
		if t == nil {
			return omnicatalog.TableRef{}
		}
		return omnicatalog.TableRef{Schema: t.Schema, Name: t.Name}
	}
	toRefs := func(tables []*ast.TableRef) []omnicatalog.TableRef {
		refs := make([]omnicatalog.TableRef, 0, len(tables))
		for _, t := range tables {
			if t != nil {
				refs = append(refs, toRef(t))
			}
		}
		return refs
	}

	switch n := node.(type) {
	case *ast.CreateDatabaseStmt:
		info.Kind = omnicatalog.StatementCreateDatabase
		info.Database = n.Name
	case *ast.AlterDatabaseStmt:
		info.Kind = omnicatalog.StatementAlterDatabase
		info.Database = n.Name
	case *ast.DropDatabaseStmt:
		info.Kind = omnicatalog.StatementDropDatabase
		info.Database = n.Name
	case *ast.CreateTableStmt:
		info.Kind = omnicatalog.StatementCreateTable
		info.IfNotExists = n.IfNotExists
		info.IsCTAS = n.Select != nil
		if n.Table != nil {
			info.TableRefs = []omnicatalog.TableRef{toRef(n.Table)}
		}
	case *ast.AlterTableStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		if n.Table != nil {
			info.TableRefs = []omnicatalog.TableRef{toRef(n.Table)}
		}
		for _, cmd := range n.Commands {
			if cmd != nil && cmd.Type == ast.ATRenameTable && n.Table != nil && cmd.NewTable != nil {
				info.RenamePairs = append(info.RenamePairs, omnicatalog.RenamePair{
					Old: toRef(n.Table),
					New: toRef(cmd.NewTable),
				})
			}
		}
	case *ast.DropTableStmt:
		info.Kind = omnicatalog.StatementDropTable
		info.TableRefs = toRefs(n.Tables)
	case *ast.CreateIndexStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		if n.Table != nil {
			info.TableRefs = []omnicatalog.TableRef{toRef(n.Table)}
		}
	case *ast.DropIndexStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		if n.Table != nil {
			info.TableRefs = []omnicatalog.TableRef{toRef(n.Table)}
		}
	case *ast.CreateViewStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		if n.Name != nil {
			info.TableRefs = []omnicatalog.TableRef{toRef(n.Name)}
		}
	case *ast.RenameTableStmt:
		info.Kind = omnicatalog.StatementRenameTable
		for _, pair := range n.Pairs {
			if pair == nil {
				continue
			}
			oldRef := toRef(pair.Old)
			if pair.Old != nil {
				info.TableRefs = append(info.TableRefs, oldRef)
			}
			if pair.Old != nil && pair.New != nil {
				info.RenamePairs = append(info.RenamePairs, omnicatalog.RenamePair{
					Old: oldRef,
					New: toRef(pair.New),
				})
			}
		}
	default:
	}
	return info, true
}
