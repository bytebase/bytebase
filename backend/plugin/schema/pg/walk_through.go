package pg

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	PublicSchemaName = "public"
)

func init() {
	schema.RegisterWalkThroughWithContext(storepb.Engine_POSTGRES, WalkThroughOmni)
}

// WalkThrough walks through the PostgreSQL DDL and builds catalog metadata.
func WalkThrough(d *model.DatabaseMetadata, ast []base.AST) *storepb.Advice {
	return WalkThroughWithContext(schema.WalkThroughContext{}, d, ast)
}

// WalkThroughWithContext walks through the PostgreSQL DDL and builds catalog metadata.
func WalkThroughWithContext(ctx schema.WalkThroughContext, d *model.DatabaseMetadata, ast []base.AST) *storepb.Advice {
	return WalkThroughOmni(ctx, d, ast)
}
