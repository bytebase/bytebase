package util

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func extractSensitiveField(dbType db.Type, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}

	switch dbType {
	case db.MySQL, db.MariaDB, db.OceanBase:
		for _, database := range schemaInfo.DatabaseList {
			if len(database.SchemaList) == 0 {
				continue
			}
			if len(database.SchemaList) > 1 {
				return nil, errors.Errorf("MySQL schema info should only have one schema per database, but got %d, %v", len(database.SchemaList), database.SchemaList)
			}
			if database.SchemaList[0].Name != "" {
				return nil, errors.Errorf("MySQL schema info should have empty schema name, but got %s", database.SchemaList[0].Name)
			}
		}
		extractor := &mysqlparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractMySQLSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.TiDB:
		for _, database := range schemaInfo.DatabaseList {
			if len(database.SchemaList) == 0 {
				continue
			}
			if len(database.SchemaList) > 1 {
				return nil, errors.Errorf("TiDB schema info should only have one schema per database, but got %d, %v", len(database.SchemaList), database.SchemaList)
			}
			if database.SchemaList[0].Name != "" {
				return nil, errors.Errorf("TiDB schema info should have empty schema name, but got %s", database.SchemaList[0].Name)
			}
		}
		extractor := &tidbparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractTiDBSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.Postgres, db.Redshift, db.RisingWave:
		extractor := &pgparser.SensitiveFieldExtractor{
			SchemaInfo: schemaInfo,
		}
		return extractor.ExtractPostgreSQLSensitiveField(statement)
	case db.Oracle, db.DM:
		for _, database := range schemaInfo.DatabaseList {
			if len(database.SchemaList) == 0 {
				continue
			}
			if len(database.SchemaList) > 1 {
				return nil, errors.Errorf("Oracle schema info should only have one schema per database, but got %d, %v", len(database.SchemaList), database.SchemaList)
			}
			if database.SchemaList[0].Name != database.Name {
				return nil, errors.Errorf("Oracle schema info should have the same database name and schema name, but got %s and %s", database.Name, database.SchemaList[0].Name)
			}
		}
		extractor := &plsqlparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractOracleSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.Snowflake:
		extractor := &snowparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractSnowsqlSensitiveFields(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.MSSQL:
		extractor := &tsqlparser.SensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractTSqlSensitiveFields(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, nil
	}
}
