package parser

import (
	"github.com/pkg/errors"

	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// ExtractDatabaseList extracts all databases from statement.
func ExtractDatabaseList(engineType storepb.Engine, statement string, currentDatabase string) ([]string, error) {
	switch engineType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		// TODO(d): use mysql parser.
		return tidbparser.ExtractDatabaseList(statement, currentDatabase)
	case storepb.Engine_TIDB:
		return tidbparser.ExtractDatabaseList(statement, currentDatabase)
	case storepb.Engine_SNOWFLAKE:
		return snowparser.ExtractDatabaseList(statement, currentDatabase)
	case storepb.Engine_MSSQL:
		return tsqlparser.ExtractDatabaseList(statement, currentDatabase)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}
