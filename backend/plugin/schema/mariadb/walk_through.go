// Package mariadb registers MariaDB's DDL simulation.
//
// Unlike the other engine packages here it owns no catalog work. omni's MariaDB
// catalog has New, Exec and LoadSQL but no LoadMetadata, Diff, GenerateMigration
// or version handling — 56 of the 66 catalog entry points schema/mysql uses are
// absent — so there is nothing to install a synced schema into and nothing to
// diff. MariaDB therefore simulates on omni's MySQL catalog, and this package
// exists to hold that decision in a MariaDB file rather than as a special case
// inside schema/mysql. When omni's catalog grows those entry points, this is
// where the real fork goes.
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
