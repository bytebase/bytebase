package v1

import (
	"fmt"

	"github.com/pkg/errors"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func checkDatabaseMetadata(engine storepb.Engine, metadata *storepb.DatabaseSchemaMetadata) error {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_POSTGRES:
	default:
		return errors.Errorf("unsupported engine for check database metadata: %v", engine)
	}

	schemaMap := make(map[string]bool)
	for _, schema := range metadata.GetSchemas() {
		if (engine == storepb.Engine_MYSQL || engine == storepb.Engine_TIDB) && schema.GetName() != "" {
			return errors.Errorf("schema name should be empty for MySQL and TiDB")
		}
		if _, ok := schemaMap[schema.GetName()]; ok {
			return errors.Errorf("duplicate schema name %s", schema.GetName())
		}
		schemaMap[schema.GetName()] = true

		tableNameMap := make(map[string]bool)
		for _, table := range schema.GetTables() {
			if table.GetName() == "" {
				return errors.Errorf("table name should not be empty")
			}
			if _, ok := tableNameMap[table.GetName()]; ok {
				return errors.Errorf("duplicate table name %s", table.GetName())
			}
			tableNameMap[table.GetName()] = true

			columnNameMap := make(map[string]bool)
			for _, column := range table.GetColumns() {
				if column.GetName() == "" {
					return errors.Errorf("column name should not be empty in table %s", table.GetName())
				}
				if _, ok := columnNameMap[column.GetName()]; ok {
					return errors.Errorf("duplicate column name %s in table %s", column.GetName(), table.GetName())
				}
				columnNameMap[column.GetName()] = true

				if column.GetType() == "" {
					return errors.Errorf("column %s type should not be empty in table %s", column.GetName(), table.GetName())
				}
			}

			indexNameMap := make(map[string]bool)
			for _, index := range table.GetIndexes() {
				if index.GetName() == "" {
					return errors.Errorf("index name should not be empty in table %s", table.GetName())
				}
				if _, ok := indexNameMap[index.GetName()]; ok {
					return errors.Errorf("duplicate index name %s in table %s", index.GetName(), table.GetName())
				}
				indexNameMap[index.GetName()] = true
				if index.Primary {
					for _, key := range index.GetExpressions() {
						if _, ok := columnNameMap[key]; !ok {
							return errors.Errorf("primary key column %s not found in table %s", key, table.GetName())
						}
					}
				}
			}
		}
	}
	return nil
}

func checkDatabaseMetadataColumnType(engine storepb.Engine, metadata *storepb.DatabaseSchemaMetadata) error {
	for _, schema := range metadata.GetSchemas() {
		for _, table := range schema.GetTables() {
			for _, column := range table.GetColumns() {
				if !checkColumnType(engine, column.Type) {
					return errors.Errorf("column %s type %s is invalid in table %s", column.Name, column.Type, table.Name)
				}
			}
		}
	}
	return nil
}

func checkColumnType(engine storepb.Engine, tp string) bool {
	switch engine {
	case storepb.Engine_MYSQL:
		return checkMySQLColumnType(tp)
	case storepb.Engine_TIDB:
		return checkTiDBColumnType(tp)
	case storepb.Engine_POSTGRES:
		return checkPostgreSQLColumnType(tp)
	default:
		return false
	}
}

func checkTiDBColumnType(tp string) bool {
	_, err := tidbparser.ParseTiDB(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp), "", "")
	return err == nil
}

func checkMySQLColumnType(tp string) bool {
	_, err := mysqlparser.ParseMySQL(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp))
	return err == nil
}

func checkPostgreSQLColumnType(tp string) bool {
	_, err := pgrawparser.Parse(pgrawparser.ParseContext{}, fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp))
	return err == nil
}
