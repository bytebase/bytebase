package util

import (
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	defaultMaskingLevel storepb.MaskingLevel = storepb.MaskingLevel_NONE
	maxMaskingLevel     storepb.MaskingLevel = storepb.MaskingLevel_FULL
)

type fieldInfo struct {
	name         string
	table        string
	schema       string
	database     string
	maskingLevel storepb.MaskingLevel
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
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		result, err := extractor.extractTiDBSensitiveField(statement)
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
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		result, err := extractor.extractMySQLSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
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
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		result, err := extractor.extractOracleSensitiveField(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.Snowflake:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		result, err := extractor.extractSnowsqlSensitiveFields(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	case db.MSSQL:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		result, err := extractor.extractTSqlSensitiveFields(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, nil
	}
}
