package v2

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// defaultSchemaName is the default schema name in PostgreSQL like DBMS.
	defaultSchemaName = "public"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_POSTGRES, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_RISINGWAVE, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, statement string, getDatabaseMetadata base.GetDatabaseMetadataFunc) (*base.QuerySpan, error) {
	return nil, nil
}
