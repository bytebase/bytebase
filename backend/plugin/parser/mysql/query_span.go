package mysql

import (
	"context"
	"errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MYSQL, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_MARIADB, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_OCEANBASE, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_STARROCKS, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_DORIS, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, statement, database string, getDatabaseMetadata base.GetDatabaseMetadataFunc) (*base.QuerySpan, error) {
	return nil, errors.New("not implemented")
}
