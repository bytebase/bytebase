import { cloneDeep } from "lodash-es";
import { t } from "@/plugins/i18n";
import { pushNotification, useDBSchemaV1Store, useDatabaseCatalogV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnConfig,
  TableConfig,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import {
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
  config,
}: {
  database: string;
  schema: string;
  table: string;
  column: string;
  config: Partial<ColumnConfig>;
}) => {
  const dbSchemaV1Store = useDBSchemaV1Store();
  const databaseMetadata = dbSchemaV1Store.getDatabaseMetadata(
    database,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );

  const tableConfig = dbSchemaV1Store.getTableConfig(database, schema, table);
  const index = tableConfig.columnConfigs.findIndex(
    (config) => config.name === column
  );

  const pendingUpdateTableConfig = cloneDeep(tableConfig);
  if (index < 0) {
    pendingUpdateTableConfig.columnConfigs.push(
      ColumnConfig.fromPartial({
        name: column,
        ...config,
      })
    );
  } else {
    pendingUpdateTableConfig.columnConfigs[index] = {
      ...pendingUpdateTableConfig.columnConfigs[index],
      ...config,
    };
  }

  const schemaConfig = dbSchemaV1Store.getSchemaConfig(database, schema);
  const pendingUpdateSchemaConfig = cloneDeep(schemaConfig);
  const tableIndex = pendingUpdateSchemaConfig.tableConfigs.findIndex(
    (config) => config.name === pendingUpdateTableConfig.name
  );
  if (tableIndex < 0) {
    pendingUpdateSchemaConfig.tableConfigs.push(pendingUpdateTableConfig);
  } else {
    pendingUpdateSchemaConfig.tableConfigs[tableIndex] =
      pendingUpdateTableConfig;
  }

  const pendingUpdateDatabaseConfig = cloneDeep(databaseMetadata);
  const schemaIndex = pendingUpdateDatabaseConfig.schemaConfigs.findIndex(
    (config) => config.name === pendingUpdateSchemaConfig.name
  );
  if (schemaIndex < 0) {
    pendingUpdateDatabaseConfig.schemaConfigs.push(pendingUpdateSchemaConfig);
  } else {
    pendingUpdateDatabaseConfig.schemaConfigs[schemaIndex] =
      pendingUpdateSchemaConfig;
  }

  await dbSchemaV1Store.updateDatabaseSchemaConfigs(
    pendingUpdateDatabaseConfig
  );
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
  config: Partial<TableCatalog>
) => {
  const dbSchemaV1Store = useDBSchemaV1Store();
  const dbCatalogStore = useDatabaseCatalogV1Store();
  const databaseMetadata = dbSchemaV1Store.getDatabaseMetadata(
    database,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
  const catalog = dbCatalogStore.getOrFetchDatabaseCatalog({
    database,
    skipCache: true,
    silent: true,
  });
  console.debug("catalog: ", catalog);

  const schemaConfig = dbSchemaV1Store.getSchemaConfig(database, schema);
  const pendingUpdateSchemaConfig = cloneDeep(schemaConfig);
  const tableIndex = pendingUpdateSchemaConfig.tableConfigs.findIndex(
    (config) => config.name === table
  );
  if (tableIndex < 0) {
    pendingUpdateSchemaConfig.tableConfigs.push(
      TableConfig.fromPartial({
        name: table,
        ...config,
      })
    );
  } else {
    pendingUpdateSchemaConfig.tableConfigs[tableIndex] = {
      ...pendingUpdateSchemaConfig.tableConfigs[tableIndex],
      ...config,
    };
  }

  const pendingUpdateDatabaseConfig = cloneDeep(databaseMetadata);
  const schemaIndex = pendingUpdateDatabaseConfig.schemaConfigs.findIndex(
    (config) => config.name === pendingUpdateSchemaConfig.name
  );
  if (schemaIndex < 0) {
    pendingUpdateDatabaseConfig.schemaConfigs.push(pendingUpdateSchemaConfig);
  } else {
    pendingUpdateDatabaseConfig.schemaConfigs[schemaIndex] =
      pendingUpdateSchemaConfig;
  }

  await dbSchemaV1Store.updateDatabaseSchemaConfigs(
    pendingUpdateDatabaseConfig
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
