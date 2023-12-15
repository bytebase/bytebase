import { pick } from "lodash-es";
import { Ref, reactive, watch } from "vue";
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
import { EditTarget } from "../types";
import { keyForResource, keyForResourceName } from "./common";

export const useEditConfigs = (targets: Ref<EditTarget[]>) => {
  // Build maps from keys to metadata objects for acceleration
  const buildMaps = (targets: EditTarget[]) => {
    const tableConfig = reactive(
      new Map(
        targets.flatMap((target) => {
          return target.metadata.schemaConfigs.flatMap((schemaConfig) => {
            return schemaConfig.tableConfigs.map((tableConfig) => {
              const key = keyForResourceName(
                target.database.name,
                schemaConfig.name,
                tableConfig.name
              );
              return [key, tableConfig];
            });
          });
        })
      )
    );
    const columnConfig = reactive(
      new Map(
        targets.flatMap((target) => {
          return target.metadata.schemaConfigs.flatMap((schemaConfig) => {
            return schemaConfig.tableConfigs.flatMap((tableConfig) => {
              return tableConfig.columnConfigs.map((columnConfig) => {
                const key = keyForResourceName(
                  target.database.name,
                  schemaConfig.name,
                  tableConfig.name,
                  columnConfig.name
                );
                return [key, columnConfig];
              });
            });
          });
        })
      )
    );
    return { tableConfig, columnConfig };
  };

  const maps = reactive(buildMaps(targets.value));
  watch(
    targets,
    (targets) => {
      Object.assign(maps, buildMaps(targets));
    },
    {
      deep: false,
    }
  );

  // TableConfig
  const getTableConfig = (
    database: ComposedDatabase,
    metadata: {
      schema: SchemaMetadata;
      table: TableMetadata;
    }
  ) => {
    const key = keyForResource(database, pick(metadata, "schema", "table"));
    return maps.tableConfig.get(key);
  };
  const insertTableConfig = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    tableConfig: TableConfig
  ) => {
    let schemaConfig = metadata.database.schemaConfigs.find(
      (sc) => sc.name === metadata.schema.name
    );
    if (!schemaConfig) {
      schemaConfig = SchemaConfig.fromPartial({
        name: metadata.schema.name,
        tableConfigs: [],
      });
      metadata.database.schemaConfigs.push(schemaConfig);
    }
    schemaConfig.tableConfigs.push(tableConfig);

    const key = keyForResource(database, pick(metadata, "schema", "table"));
    maps.tableConfig.set(key, tableConfig);
  };
  const upsertTableConfig = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    update: (config: TableConfig) => void
  ) => {
    let config = getTableConfig(database, metadata);
    if (!config) {
      config = TableConfig.fromPartial({
        name: metadata.table.name,
      });
      insertTableConfig(database, metadata, config);
    }
    update(config);
    // Maintain the columnConfig map
    config.columnConfigs.forEach((columnConfig) => {
      const key = keyForResourceName(
        database.name,
        metadata.schema.name,
        metadata.table.name,
        columnConfig.name
      );
      maps.columnConfig.set(key, columnConfig);
    });
  };

  // ColumnConfig
  const getColumnConfig = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    }
  ) => {
    const key = keyForResource(
      database,
      pick(metadata, "schema", "table", "column")
    );
    return maps.columnConfig.get(key);
  };
  const insertColumnConfig = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    },
    columnConfig: ColumnConfig
  ) => {
    upsertTableConfig(
      database,
      pick(metadata, "database", "schema", "table"),
      (tableConfig) => {
        tableConfig.columnConfigs.push(columnConfig);
      }
    );
    // Need not to maintain columnConfig map here
    // since `upsertTableConfig` did this already
  };
  const upsertColumnConfig = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    },
    update: (config: ColumnConfig) => void
  ) => {
    let config = getColumnConfig(database, metadata);
    if (!config) {
      config = ColumnConfig.fromPartial({
        name: metadata.column.name,
      });
      insertColumnConfig(database, metadata, config);
    }
    update(config);
  };

  return {
    getTableConfig,
    upsertTableConfig,
    getColumnConfig,
    upsertColumnConfig,
  };
};
