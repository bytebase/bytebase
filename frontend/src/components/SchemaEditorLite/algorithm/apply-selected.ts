import { cloneDeep } from "lodash-es";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnConfig,
  ColumnMetadata,
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaConfig, TableConfig } from "@/types/proto/v1/database_service";
import { filterColumnMetadata, filterTableMetadata } from "@/utils";
import type { SchemaEditorContext } from "../context";
import { keyForResource, keyForResourceName } from "../context/common";
import type { RolloutObject } from "../types";
import {
  buildColumnConfigMap,
  buildColumnMap,
  cleanupUnusedConfigs,
} from "./utils";

export const useApplySelectedMetadataEdit = (context: SchemaEditorContext) => {
  const {
    getTableStatus,
    getColumnStatus,
    getViewStatus,
    getProcedureStatus,
    getFunctionStatus,
  } = context;

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
    const sourceViewMap = new Map(
      source.schemas.flatMap((schema) => {
        return schema.views.map((view) => {
          const key = keyForResource(db, {
            schema,
            view,
          });
          return [key, { schema, view }];
        });
      })
    );
    const sourceProcedureMap = new Map(
      source.schemas.flatMap((schema) => {
        return schema.procedures.map((procedure) => {
          const key = keyForResource(db, {
            schema,
            procedure,
          });
          return [key, { schema, procedure }];
        });
      })
    );
    const sourceFunctionMap = new Map(
      source.schemas.flatMap((schema) => {
        return schema.functions.map((func) => {
          const key = keyForResource(db, {
            schema,
            function: func,
          });
          return [key, { schema, function: func }];
        });
      })
    );
    const sourceTableConfigMap = new Map(
      source.schemaConfigs.flatMap((schemaConfig) => {
        return schemaConfig.tableConfigs.map((tableConfig) => {
          const key = keyForResourceName({
            database: db.name,
            schema: schemaConfig.name,
            table: tableConfig.name,
          });
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
      tableConfig: TableConfig
    ) => {
      const targetTableConfigMap = new Map(
        tableConfig.columnConfigs.map((cc) => [cc.name, cc])
      );
      const pickedColumns: ColumnMetadata[] = [];
      const pickedColumnConfigs: ColumnConfig[] = [];
      for (let i = 0; i < table.columns.length; i++) {
        const column = table.columns[i];
        const key = keyForResourceName({
          database: db.name,
          schema: schema.name,
          table: table.name,
          column: column.name,
        });
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
          ...tableConfig,
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
            const tableConfig =
              targetSchemaConfig?.tableConfigs.find(
                (tc) => tc.name === table.name
              ) ?? TableConfig.fromPartial({ name: table.name });
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

      const views: ViewMetadata[] = [];
      for (let j = 0; j < schema.views.length; j++) {
        const view = schema.views[j];
        const key = keyForResource(db, { schema, view });
        const picked = selectedObjectKeys.has(key);
        if (picked) {
          const status = getViewStatus(db, {
            database: target,
            schema,
            view,
          });
          if (status === "dropped") {
            // Drop view
            // Don't collect the dropped view
            continue;
          } else {
            // Collect the edited view
            views.push(view);
          }
        } else {
          const sourceView = sourceViewMap.get(key);
          if (sourceView) {
            // Collect the original view
            views.push(cloneDeep(sourceView.view));
          }
        }
      }
      schema.views = views;

      const procedures: ProcedureMetadata[] = [];
      for (let j = 0; j < schema.procedures.length; j++) {
        const procedure = schema.procedures[j];
        const key = keyForResource(db, { schema, procedure });
        const picked = selectedObjectKeys.has(key);
        if (picked) {
          const status = getProcedureStatus(db, {
            database: target,
            schema,
            procedure,
          });
          if (status === "dropped") {
            // Drop procedure
            // Don't collect the dropped procedure
            continue;
          } else {
            // Collect the edited procedure
            procedures.push(procedure);
          }
        } else {
          const sourceProcedure = sourceProcedureMap.get(key);
          if (sourceProcedure) {
            // Collect the original procedure
            procedures.push(cloneDeep(sourceProcedure.procedure));
          }
        }
      }
      schema.procedures = procedures;

      const functions: FunctionMetadata[] = [];
      for (let j = 0; j < schema.functions.length; j++) {
        const func = schema.functions[j];
        const key = keyForResource(db, { schema, function: func });
        const picked = selectedObjectKeys.has(key);
        if (picked) {
          const status = getFunctionStatus(db, {
            database: target,
            schema,
            function: func,
          });
          if (status === "dropped") {
            // Drop function
            // Don't collect the dropped function
            continue;
          } else {
            // Collect the edited function
            functions.push(func);
          }
        } else {
          const sourceFunction = sourceFunctionMap.get(key);
          if (sourceFunction) {
            // Collect the original function
            functions.push(cloneDeep(sourceFunction.function));
          }
        }
      }
      schema.functions = functions;
    }

    target.schemaConfigs = schemaConfigs;

    cleanupUnusedConfigs(target);
  };

  return { applySelectedMetadataEdit };
};
