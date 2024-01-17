import { cloneDeep } from "lodash-es";
import { ComposedDatabase } from "@/types";
import {
  ColumnConfig,
  ColumnMetadata,
  DatabaseMetadata,
  SchemaConfig,
  SchemaMetadata,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { filterColumnMetadata, filterTableMetadata } from "@/utils";
import { SchemaEditorContext } from "../context";
import { keyForResource, keyForResourceName } from "../context/common";
import { RolloutObject } from "../types";
import {
  buildColumnConfigMap,
  buildColumnMap,
  cleanupUnusedConfigs,
} from "./utils";

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
    const sourceColumnMap = buildColumnMap(db, source);
    const sourceColumnConfigMap = buildColumnConfigMap(db, source);

    const applyTableEdits = (
      schema: SchemaMetadata,
      table: TableMetadata,
      schemaConfig: SchemaConfig,
      tableConfig: TableConfig | undefined
    ) => {
      const targetTableConfigMap = new Map(
        tableConfig?.columnConfigs.map((cc) => [cc.name, cc])
      );
      const pickedColumns: ColumnMetadata[] = [];
      const pickedColumnConfigs: ColumnConfig[] = [];
      for (let i = 0; i < table.columns.length; i++) {
        const column = table.columns[i];
        const key = keyForResourceName(
          db.name,
          schema.name,
          table.name,
          column.name
        );
        const picked = selectedObjectKeys.has(key);
        if (picked) {
          const status = getColumnStatus(db, {
            database: target,
            schema,
            table,
            column,
          });
          if (status === "dropped") {
            // Don't collect dropped columns
            continue;
          } else {
            // collect the column
            // maybe it's clean, updated or created
            pickedColumns.push(column);
            // Together with its column config, if found
            const columnConfig = targetTableConfigMap.get(column.name);
            if (columnConfig) {
              pickedColumnConfigs.push(columnConfig);
            }
          }
        } else {
          // for non-picked columns, collect the original version of the column
          const sourceColumn = sourceColumnMap.get(key);
          if (sourceColumn) {
            // collect the original column
            pickedColumns.push(
              filterColumnMetadata(cloneDeep(sourceColumn.column))
            );
            // together with its original columnConfig
            const columnConfig = sourceColumnConfigMap.get(key)?.columnConfig;
            if (columnConfig) {
              pickedColumnConfigs.push(cloneDeep(columnConfig));
            }
          }
        }
      }
      table.columns = pickedColumns;
      schemaConfig.tableConfigs.push(
        TableConfig.fromPartial({
          name: table.name,
          columnConfigs: pickedColumnConfigs,
        })
      );
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
            const tableConfig = targetSchemaConfig?.tableConfigs.find(
              (tc) => tc.name === table.name
            );
            // apply column edits for picked table
            // Together with its tableConfig and columnConfigs
            applyTableEdits(schema, table, schemaConfig, tableConfig);
          }
        } else {
          const sourceTable = sourceTableMap.get(key);
          if (sourceTable) {
            // Collect the original table
            tables.push(filterTableMetadata(cloneDeep(sourceTable.table)));
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

    cleanupUnusedConfigs(target);
  };

  return { applySelectedMetadataEdit };
};
