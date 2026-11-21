// Package mariadb registers MariaDB's DDL simulation, schema definition/metadata
// conversions, and metadata diff on omni's native MariaDB catalog.
package mariadb

import (
	"context"
	"errors"
	"strings"

	"github.com/bytebase/omni/mariadb/ast"
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
	InspectAST:          inspectMariaDBAST,
	ExtractCatalogError: extractMariaDBCatalogError,
	SplitSQL:            mariadbparser.SplitSQL,
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

func inspectMariaDBAST(unifiedAST base.AST) (omnicatalog.StatementInfo, bool) {
	node, ok := mariadbparser.GetNode(unifiedAST)
	if !ok {
		return omnicatalog.StatementInfo{}, false
	}
	info := omnicatalog.StatementInfo{}
	if source, ok := unifiedAST.(interface{ ASTSource() string }); ok {
		info.Source = strings.TrimSpace(source.ASTSource())
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
	case *ast.TruncateStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		info.TableRefs = toRefs(n.Tables)
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
	case *ast.CreateViewStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		if n.Name != nil {
			info.TableRefs = []omnicatalog.TableRef{toRef(n.Name)}
		}
	case *ast.CreateSequenceStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		if n.Name != nil {
			info.TableRefs = []omnicatalog.TableRef{toRef(n.Name)}
		}
	case *ast.AlterSequenceStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		if n.Name != nil {
			info.TableRefs = []omnicatalog.TableRef{toRef(n.Name)}
		}
	case *ast.DropSequenceStmt:
		info.Kind = omnicatalog.StatementTableRefCheck
		info.TableRefs = toRefs(n.Sequences)
	default:
	}
	return info, true
}
