package util

import (
	"regexp"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

type fieldInfo struct {
	name      string
	table     string
	schema    string
	database  string
	sensitive bool
}

type sensitiveFieldExtractor struct {
	// For Oracle, we need to know the current database to determine if the table is in the current schema.
	currentDatabase    string
	schemaInfo         *db.SensitiveSchemaInfo
	outerSchemaInfo    []fieldInfo
	cteOuterSchemaInfo []db.TableSchema

	// SELECT statement specific field.
	fromFieldList []fieldInfo
}

func extractSensitiveField(dbType db.Type, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}

	switch dbType {
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractMySQLSensitiveField(statement)
	case db.Postgres, db.Redshift, db.RisingWave:
		extractor := &sensitiveFieldExtractor{
			schemaInfo: schemaInfo,
		}
		result, err := extractor.extractPostgreSQLSensitiveField(statement)
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
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractOracleSensitiveField(statement)
	case db.Snowflake:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractSnowsqlSensitiveFields(statement)
	case db.MSSQL:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractTSqlSensitiveFields(statement)
	default:
		return nil, nil
	}
}
