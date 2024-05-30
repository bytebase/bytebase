package utils

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// ConvertDatabaseToParserEngineType converts a database type to a parser engine type.
func ConvertDatabaseToParserEngineType(engine storepb.Engine) (storepb.Engine, error) {
	switch engine {
	case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
		return storepb.Engine_ORACLE, nil
	case storepb.Engine_MSSQL:
		return storepb.Engine_MSSQL, nil
	case storepb.Engine_POSTGRES:
		return storepb.Engine_POSTGRES, nil
	case storepb.Engine_REDSHIFT:
		return storepb.Engine_REDSHIFT, nil
	case storepb.Engine_MYSQL:
		return storepb.Engine_MYSQL, nil
	case storepb.Engine_TIDB:
		return storepb.Engine_TIDB, nil
	case storepb.Engine_MARIADB:
		return storepb.Engine_MARIADB, nil
	case storepb.Engine_OCEANBASE:
		return storepb.Engine_OCEANBASE, nil
	}
	return storepb.Engine_ENGINE_UNSPECIFIED, errors.Errorf("unsupported engine type %q", engine)
}
