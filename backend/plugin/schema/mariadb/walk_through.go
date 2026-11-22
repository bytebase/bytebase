// Package mariadb registers MariaDB's DDL simulation and its metadata diff.
//
// The metadata diff (metadata_migration.go) runs on omni's MariaDB catalog, which
// is the only catalog that installs MariaDB-only definitions such as WITH SYSTEM
// VERSIONING.
//
// The walk-through still simulates on omni's MySQL catalog, re-reading the script
// with omni's MySQL parser and discarding a script that parser rejects. Nothing in
// the MariaDB catalog blocks moving it; that is a change of its own.
package mariadb

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	mysqlschema "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterWalkThroughWithContext(storepb.Engine_MARIADB, walkThrough)
}

// walkThrough simulates MariaDB DDL on omni's MySQL catalog.
//
// The MariaDB parser's nodes cannot be fed to that catalog, so the script is
// re-read with omni's MySQL parser. A statement that parser rejects — one only
// MariaDB accepts — leaves the script unsimulated, because simulating the rest
// would check it against a schema missing that statement's effect.
func walkThrough(ctx context.Context, wtCtx schema.WalkThroughContext, d *model.DatabaseMetadata, _ []base.AST) *storepb.Advice {
	parsed, err := base.ParseStatements(storepb.Engine_MYSQL, wtCtx.RawSQL)
	if err != nil {
		return nil
	}
	return mysqlschema.WalkThroughWithContext(ctx, wtCtx, d, base.ExtractASTs(parsed))
}
