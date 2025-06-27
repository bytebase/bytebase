import { create } from "@bufbuild/protobuf";
import type { Ref } from "vue";
import { reactive, watch } from "vue";
import type {
  TableCatalog,
  ColumnCatalog,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  DatabaseCatalogSchema,
  SchemaCatalogSchema,
  TableCatalogSchema,
  TableCatalog_ColumnsSchema,
  ColumnCatalogSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import type { EditTarget } from "../types";
import { keyForResourceName } from "./common";

export const useEditCatalogs = (targets: Ref<EditTarget[]>) => {
  // Build maps from keys to metadata objects for acceleration
  const buildMaps = (targets: EditTarget[]) => {
    const databaseCatalog = reactive(
      new Map(
        targets.map((target) => {
          const key = keyForResourceName({
            database: target.database.name,
          });
          return [key, target.catalog];
        })
      )
    );
    const tableCatalog = reactive(
      new Map(
        targets.flatMap((target) => {
          return target.catalog.schemas.flatMap((schemaCatalog) => {
            return schemaCatalog.tables.map((tableCatalog) => {
              const key = keyForResourceName({
                database: target.database.name,
                schema: schemaCatalog.name,
                table: tableCatalog.name,
              });
              return [key, tableCatalog];
            });
          });
        })
      )
    );
    const columnCatalog = reactive(
      new Map(
        targets.flatMap((target) => {
          return target.catalog.schemas.flatMap((schemaCatalog) => {
            return schemaCatalog.tables.flatMap((tableCatalog) => {
              return (
                tableCatalog.kind?.case === "columns"
                  ? tableCatalog.kind.value.columns
                  : []
              ).map((columnCatalog) => {
                const key = keyForResourceName({
                  database: target.database.name,
                  schema: schemaCatalog.name,
                  table: tableCatalog.name,
                  column: columnCatalog.name,
                });
                return [key, columnCatalog];
              });
            });
          });
        })
      )
    );
    return { databaseCatalog, tableCatalog, columnCatalog };
  };

  const maps = reactive(buildMaps(targets.value));
  watch(
    targets,
    (targets) => {
      Object.assign(maps, buildMaps(targets));
    },
    {
      deep: true,
    }
  );

  const getDatabaseCatalog = (database: string) => {
    const databaseKey = keyForResourceName({
      database,
    });
    let databaseCatalog = maps.databaseCatalog.get(databaseKey);
    if (!databaseCatalog) {
      databaseCatalog = create(DatabaseCatalogSchema, {
        name: database,
      });
      maps.databaseCatalog.set(databaseKey, databaseCatalog);
    }
    return databaseCatalog;
  };

  // Table catalog
  const getTableCatalog = ({
    database,
    schema,
    table,
  }: {
    database: string;
    schema: string;
    table: string;
  }) => {
    const key = keyForResourceName({
      database,
      schema,
      table,
    });
    return maps.tableCatalog.get(key);
  };
  const insertTableCatalog = ({
    database,
    schema,
    table,
  }: {
    database: string;
    schema: string;
    table: TableCatalog;
  }) => {
    const databaseCatalog = getDatabaseCatalog(database);
    let schemaCatalog = databaseCatalog.schemas.find(
      (sc) => sc.name === schema
    );
    if (!schemaCatalog) {
      schemaCatalog = create(SchemaCatalogSchema, {
        name: schema,
        tables: [],
      });
      databaseCatalog.schemas.push(schemaCatalog);
    }
    schemaCatalog.tables.push(table);

    const key = keyForResourceName({
      database,
      schema,
      table: table.name,
    });
    maps.tableCatalog.set(key, table);
  };
  const removeTableCatalog = ({
    database,
    schema,
    table,
  }: {
    database: string;
    schema: string;
    table: string;
  }) => {
    const databaseCatalog = getDatabaseCatalog(database);
    const schemaCatalog = databaseCatalog.schemas.find(
      (sc) => sc.name === schema
    );
    if (schemaCatalog) {
      schemaCatalog.tables = schemaCatalog.tables.filter(
        (tableCatalog) => tableCatalog.name !== table
      );
    }
  };
  const upsertTableCatalog = (
    {
      database,
      schema,
      table,
    }: {
      database: string;
      schema: string;
      table: string;
    },
    update: (catalog: TableCatalog) => void
  ) => {
    let tableCatalog = getTableCatalog({
      database,
      schema,
      table,
    });
    if (!tableCatalog) {
      tableCatalog = create(TableCatalogSchema, {
        name: table,
        kind: {
          case: "columns",
          value: create(TableCatalog_ColumnsSchema, {}),
        },
      });
      insertTableCatalog({
        database,
        schema,
        table: tableCatalog,
      });
    }
    if (tableCatalog.kind?.case !== "columns") {
      tableCatalog.kind = {
        case: "columns",
        value: create(TableCatalog_ColumnsSchema, {}),
      };
    }
    update(tableCatalog);
  };

  // Column catalog
  const getColumnCatalog = ({
    database,
    schema,
    table,
    column,
  }: {
    database: string;
    schema: string;
    table: string;
    column: string;
  }) => {
    const key = keyForResourceName({
      database,
      schema,
      table,
      column,
    });
    return maps.columnCatalog.get(key);
  };
  const insertColumnCatalog = ({
    database,
    schema,
    table,
    column,
  }: {
    database: string;
    schema: string;
    table: string;
    column: ColumnCatalog;
  }) => {
    upsertTableCatalog(
      {
        database,
        schema,
        table,
      },
      (tableCatalog) => {
        if (tableCatalog.kind?.case === "columns") {
          tableCatalog.kind.value.columns.push(column);
        }
      }
    );
    // Need not to maintain column catalog map here
    // since `upsertTableCatalog` did this already
  };
  const removeColumnCatalog = ({
    database,
    schema,
    table,
    column,
  }: {
    database: string;
    schema: string;
    table: string;
    column: string;
  }) => {
    const tableCatalog = getTableCatalog({
      database,
      schema,
      table,
    });
    if (!tableCatalog) {
      return;
    }
    if (tableCatalog.kind?.case !== "columns") {
      tableCatalog.kind = {
        case: "columns",
        value: create(TableCatalog_ColumnsSchema, {}),
      };
    }
    tableCatalog.kind.value.columns = tableCatalog.kind.value.columns.filter(
      (columnCatalog) => columnCatalog.name !== column
    );
  };
  const upsertColumnCatalog = (
    {
      database,
      schema,
      table,
      column,
    }: {
      database: string;
      schema: string;
      table: string;
      column: string;
    },
    update: (catalog: ColumnCatalog) => void
  ) => {
    let columnCatalog = getColumnCatalog({
      database,
      schema,
      table,
      column,
    });
    if (!columnCatalog) {
      columnCatalog = create(ColumnCatalogSchema, {
        name: column,
      });
      insertColumnCatalog({
        database,
        schema,
        table,
        column: columnCatalog,
      });
    }
    update(columnCatalog);
  };

  return {
    getDatabaseCatalog,
    getTableCatalog,
    upsertTableCatalog,
    removeTableCatalog,
    getColumnCatalog,
    removeColumnCatalog,
    upsertColumnCatalog,
  };
};
