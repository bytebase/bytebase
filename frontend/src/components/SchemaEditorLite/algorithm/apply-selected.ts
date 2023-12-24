import { cloneDeep } from "lodash-es";
import { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  SchemaConfig,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaEditorContext } from "../context";
import { keyForResource, keyForResourceName } from "../context/common";
import { RolloutObject } from "../types";

export const useApplySelectedMetadataEdit = (context: SchemaEditorContext) => {
  const { getTableStatus, getColumnStatus } = context;

  const applySelectedMetadataEdit = (
    db: ComposedDatabase,
    source: DatabaseMetadata,
    target: DatabaseMetadata,
    selectedRolloutObjects: RolloutObject[]
  ) => {
    const selectedObjectKeys = new Set(
      selectedRolloutObjects.map((ro) => keyForResource(ro.db, ro.metadata))
    );
    const sourceTableMap = new Map(
      source.schemas.flatMap((schema) => {
        return schema.tables.map((table) => {
          const key = keyForResource(db, {
            schema,
            table,
          });
          return [key, { schema, table }];
        });
      })
    );
    const sourceTableConfigMap = new Map(
      source.schemaConfigs.flatMap((schemaConfig) => {
        return schemaConfig.tableConfigs.map((tableConfig) => {
          const key = keyForResourceName(
            db.name,
            schemaConfig.name,
            tableConfig.name
          );
          return [key, { schemaConfig, tableConfig }];
        });
      })
    );

    const applyTableEdits = (schema: SchemaMetadata, table: TableMetadata) => {
      // Drop columns
      target.schemas.forEach((schema) => {
        schema.tables.forEach((table) => {
          table.columns = table.columns.filter((column) => {
            const status = getColumnStatus(db, {
              database: target,
              schema,
              table,
              column,
            });
            return status !== "dropped";
          });
        });
      });
    };

    const schemaConfigs: SchemaConfig[] = [];
    for (let i = 0; i < target.schemas.length; i++) {
      const schema = target.schemas[i];
      const schemaConfig = SchemaConfig.fromPartial({ name: schema.name });
      const targetSchemaConfig = target.schemaConfigs.find(
        (sc) => sc.name === schema.name
      );
      const tables: TableMetadata[] = [];
      for (let j = 0; j < schema.tables.length; j++) {
        const table = schema.tables[j];
        const key = keyForResource(db, { schema, table });
        const picked = selectedObjectKeys.has(key);
        if (picked) {
          const status = getTableStatus(db, {
            database: target,
            schema,
            table,
          });
          if (status === "dropped") {
            // Drop table
            // Don't collect the dropped table
            continue;
          } else {
            // Collect the edited table
            tables.push(table);
            // Together with its tableConfig
            const tableConfig = targetSchemaConfig?.tableConfigs.find(
              (tc) => tc.name === table.name
            );
            if (tableConfig) {
              schemaConfig.tableConfigs.push(tableConfig);
            }
            // apply column edits for non-pending-creating (existed and updated) tables
            if (status !== "created") {
              applyTableEdits(schema, table);
            }
          }
        } else {
          const sourceTable = sourceTableMap.get(key);
          if (sourceTable) {
            // Collect the original table
            tables.push(cloneDeep(sourceTable.table));
            // Together with its tableConfig
            const sourceTableConfig =
              sourceTableConfigMap.get(key)?.tableConfig;
            if (sourceTableConfig) {
              schemaConfig.tableConfigs.push(cloneDeep(sourceTableConfig));
            }
          }
        }
      }
      schema.tables = tables;
      if (schemaConfig.tableConfigs.length > 0) {
        schemaConfigs.push(schemaConfig);
      }
    }
    target.schemaConfigs = schemaConfigs;
  };

  return { applySelectedMetadataEdit };
};
