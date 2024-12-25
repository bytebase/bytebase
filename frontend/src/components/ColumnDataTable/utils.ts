import { cloneDeep } from "lodash-es";
import { t } from "@/plugins/i18n";
import { pushNotification, useDatabaseCatalogV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnCatalog,
  TableCatalog,
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

export const updateColumnConfig = async ({
  database,
  schema,
  table,
  column,
  columnCatalog,
}: {
  database: string;
  schema: string;
  table: string;
  column: string;
  columnCatalog: Partial<ColumnCatalog>;
}) => {
  const dbCatalogStore = useDatabaseCatalogV1Store();
  const catalog = await dbCatalogStore.getOrFetchDatabaseCatalog({database});

  const pendingUpdateCatalog = cloneDeep(catalog);
  let targetSchema = pendingUpdateCatalog.schemas.find((s) => s.name === schema);
  if (!targetSchema) {
    targetSchema = {name: schema, tables: []};
    pendingUpdateCatalog.schemas.push(targetSchema);
  }

  let targetTable = targetSchema.tables.find((t) => t.name === table);
  if (!targetTable) {
    targetTable = TableCatalog.fromPartial({ name: table });
    targetSchema.tables.push(targetTable);
  }

  const columns = targetTable.columns?.columns || [];
  const columnIndex = columns.findIndex((c) => c.name === column);
  if (columnIndex < 0) {
    columns.push(ColumnCatalog.fromPartial({name: column, ...columnCatalog}));
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
    title: t("common.updated"),
  });
};

export const updateTableConfig = async (
  database: string,
  schema: string,
  table: string,
  tableCatalog: Partial<TableCatalog>
) => {
  const dbCatalogStore = useDatabaseCatalogV1Store();
  const catalog = await dbCatalogStore.getOrFetchDatabaseCatalog({database});

  const pendingUpdateCatalog = cloneDeep(catalog);
  let targetSchema = pendingUpdateCatalog.schemas.find((s) => s.name === schema);
  if (!targetSchema) {
    targetSchema = {name: schema, tables: []};
    pendingUpdateCatalog.schemas.push(targetSchema);
  }

  const tableIndex = targetSchema.tables.findIndex((t) => t.name === table);
  if (tableIndex < 0) {
    targetSchema.tables.push(TableCatalog.fromPartial({
      name: table,
      ...tableCatalog,
    }));
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
    title: t("common.updated"),
  });
};
