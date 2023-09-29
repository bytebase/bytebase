package util

import (
	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func extractSensitiveField(dbType db.Type, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	switch dbType {
	case db.MySQL, db.MariaDB, db.OceanBase:
		extractor := &mysqlparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.TiDB:
		extractor := &tidbparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.Postgres, db.Redshift, db.RisingWave:
		extractor := &pgparser.SensitiveFieldExtractor{
			SchemaInfo: schemaInfo,
		}
		return extractor.ExtractSensitiveField(statement)
	case db.Oracle, db.DM:
		extractor := &plsqlparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.Snowflake:
		extractor := &snowparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractSensitiveFields(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.MSSQL:
		extractor := &tsqlparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractSensitiveFields(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, nil
	}
}
