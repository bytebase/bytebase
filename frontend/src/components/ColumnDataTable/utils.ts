import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { t } from "@/plugins/i18n";
import { pushNotification, useDatabaseCatalogV1Store } from "@/store";
import type {
  ColumnCatalog,
  TableCatalog,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  ColumnCatalogSchema,
  SchemaCatalogSchema,
  TableCatalog_ColumnsSchema,
  TableCatalogSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";

export const updateColumnCatalog = async ({
  database,
  schema,
  table,
  column,
  columnCatalog,
  notification = "common.updated",
}: {
  database: string;
  schema: string;
  table: string;
  column: string;
  columnCatalog: Partial<ColumnCatalog>;
  notification?: string;
}) => {
  const dbCatalogStore = useDatabaseCatalogV1Store();
  const catalog = await dbCatalogStore.getOrFetchDatabaseCatalog({ database });

  const pendingUpdateCatalog = cloneDeep(catalog);
  let targetSchema = pendingUpdateCatalog.schemas.find(
    (s) => s.name === schema
  );
  if (!targetSchema) {
    targetSchema = create(SchemaCatalogSchema, { name: schema, tables: [] });
    pendingUpdateCatalog.schemas.push(targetSchema);
  }

  let targetTable = targetSchema.tables.find((t) => t.name === table);
  if (!targetTable) {
    targetTable = create(TableCatalogSchema, {
      name: table,
      kind: {
        case: "columns",
        value: create(TableCatalog_ColumnsSchema, {}),
      },
    });
    targetSchema.tables.push(targetTable);
  }
  if (!targetTable.kind || targetTable.kind.case !== "columns") {
    targetTable.kind = {
      case: "columns",
      value: create(TableCatalog_ColumnsSchema, {}),
    };
  }

  const columns = targetTable.kind.value.columns || [];
  const columnIndex = columns.findIndex((c) => c.name === column);
  if (columnIndex < 0) {
    columns.push(
      create(ColumnCatalogSchema, {
        name: column,
        semanticType: columnCatalog.semanticType,
        labels: columnCatalog.labels,
        classification: columnCatalog.classification,
        objectSchema: columnCatalog.objectSchema,
      })
    );
  } else {
    columns[columnIndex] = {
      ...columns[columnIndex],
      ...columnCatalog,
    };
  }
  await dbCatalogStore.updateDatabaseCatalog(pendingUpdateCatalog);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(notification),
  });
};

export const updateTableCatalog = async ({
  database,
  schema,
  table,
  tableCatalog,
  notification = "common.updated",
}: {
  database: string;
  schema: string;
  table: string;
  tableCatalog: Partial<TableCatalog>;
  notification?: string;
}) => {
  const dbCatalogStore = useDatabaseCatalogV1Store();
  const catalog = await dbCatalogStore.getOrFetchDatabaseCatalog({ database });

  const pendingUpdateCatalog = cloneDeep(catalog);
  let targetSchema = pendingUpdateCatalog.schemas.find(
    (s) => s.name === schema
  );
  if (!targetSchema) {
    targetSchema = create(SchemaCatalogSchema, { name: schema, tables: [] });
    pendingUpdateCatalog.schemas.push(targetSchema);
  }

  const tableIndex = targetSchema.tables.findIndex((t) => t.name === table);
  if (tableIndex < 0) {
    targetSchema.tables.push(
      create(TableCatalogSchema, {
        name: table,
        kind: tableCatalog.kind,
        classification: tableCatalog.classification,
      })
    );
  } else {
    targetSchema.tables[tableIndex] = {
      ...targetSchema.tables[tableIndex],
      ...tableCatalog,
    };
  }

  await dbCatalogStore.updateDatabaseCatalog(pendingUpdateCatalog);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(notification),
  });
};
