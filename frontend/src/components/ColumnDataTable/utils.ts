import { cloneDeep } from "lodash-es";
import { t } from "@/plugins/i18n";
import { pushNotification, useDatabaseCatalogV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  SchemaCatalog,
  ColumnCatalog,
  TableCatalog,
  TableCatalog_Columns,
} from "@/types/proto/v1/database_catalog_service";

export const supportClassificationFromCommentFeature = (engine: Engine) => {
  return engine === Engine.MYSQL || engine === Engine.POSTGRES;
};

export const supportSetClassificationFromComment = (
  engine: Engine,
  classificationFromConfig: boolean
) => {
  if (!supportClassificationFromCommentFeature(engine)) {
    // Only support get classification from comment for MYSQL and PG.
    return false;
  }

  return !classificationFromConfig;
};

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
    targetSchema = SchemaCatalog.fromPartial({ name: schema, tables: [] });
    pendingUpdateCatalog.schemas.push(targetSchema);
  }

  let targetTable = targetSchema.tables.find((t) => t.name === table);
  if (!targetTable) {
    targetTable = TableCatalog.fromPartial({
      name: table,
      columns: TableCatalog_Columns.fromPartial({}),
    });
    targetSchema.tables.push(targetTable);
  }
  if (!targetTable.columns) {
    targetTable.columns = TableCatalog_Columns.fromPartial({});
  }

  const columns = targetTable.columns?.columns || [];
  const columnIndex = columns.findIndex((c) => c.name === column);
  if (columnIndex < 0) {
    columns.push(ColumnCatalog.fromPartial({ name: column, ...columnCatalog }));
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
    targetSchema = { name: schema, tables: [] };
    pendingUpdateCatalog.schemas.push(targetSchema);
  }

  const tableIndex = targetSchema.tables.findIndex((t) => t.name === table);
  if (tableIndex < 0) {
    targetSchema.tables.push(
      TableCatalog.fromPartial({
        name: table,
        ...tableCatalog,
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
