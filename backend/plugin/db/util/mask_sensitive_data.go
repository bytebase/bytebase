package util

import (
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

func extractSensitiveField(dbType db.Type, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}

	switch dbType {
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
		extractor := &TiDBSensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractTiDBSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
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
		extractor := &MySQLSensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractMySQLSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.Postgres, db.Redshift, db.RisingWave:
		extractor := &PGSensitiveFieldExtractor{
			schemaInfo: schemaInfo,
		}
		result, err := extractor.ExtractPostgreSQLSensitiveField(statement)
		if err != nil {
			tableNotFound := regexp.MustCompile("^Table \"(.*)\\.(.*)\" not found$")
			content := tableNotFound.FindStringSubmatch(err.Error())
			if len(content) == 3 && (isPostgreSQLSystemSchema(content[1]) || dbType == db.RisingWave && isRisingWaveSystemSchema(content[1])) {
				// skip for system schema
				return nil, nil
			}
			return nil, err
		}
		return result, nil
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
		extractor := &snowparser.SnowSensitiveFieldExtractor{
			CurrentDatabase: currentDatabase,
			SchemaInfo:      schemaInfo,
		}
		result, err := extractor.ExtractSnowsqlSensitiveFields(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.MSSQL:
		extractor := &TSQLSensitiveFieldExtractor{
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
