import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getColumnDefaultValuePlaceholder } from "@/react/components/SchemaEditorLite/core/columnDefaultValue";
import { Input } from "@/react/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  featureToRef,
  getColumnCatalog,
  getTableCatalog,
  useDatabaseCatalog,
  useDBSchemaV1Store,
  useSettingV1Store,
} from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  Database,
  PackageMetadata,
  SequenceMetadata,
  StreamMetadata,
  TaskMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { TablePartitionMetadata_Type } from "@/types/proto-es/v1/database_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  bytesToString,
  getDatabaseEngine,
  getDatabaseProject,
  hasIndexSizeProperty,
  hasProjectPermissionV2,
  hasSchemaProperty,
  hasTableEngineProperty,
  instanceV1HasCollationAndCharacterSet,
  instanceV1SupportsColumn,
  instanceV1SupportsIndex,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
  instanceV1SupportsTrigger,
} from "@/utils";
import {
  type ObjectSectionRow,
  ObjectSectionTable,
} from "./ObjectSectionTable";
import {
  TableDetailDialog,
  type TableDetailDialogData,
} from "./TableDetailDialog";
import { TableMetadataTable } from "./TableMetadataTable";

function filterByKeyword(name: string, keyword: string) {
  return name.toLowerCase().includes(keyword.trim().toLowerCase());
}

export function DatabaseObjectExplorer({
  database,
  loading,
  selectedSchemaName,
  tableSearchKeyword,
  externalTableSearchKeyword,
  onSelectedSchemaNameChange,
  onTableSearchKeywordChange,
  onExternalTableSearchKeywordChange,
}: {
  database: Database;
  loading: boolean;
  selectedSchemaName: string;
  tableSearchKeyword: string;
  externalTableSearchKeyword: string;
  onSelectedSchemaNameChange: (value: string) => void;
  onTableSearchKeywordChange: (value: string) => void;
  onExternalTableSearchKeywordChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  const dbSchemaStore = useDBSchemaV1Store();
  const settingStore = useSettingV1Store();
  const databaseEngine = getDatabaseEngine(database);
  const supportsSchema = hasSchemaProperty(databaseEngine);
  const schemaList = useVueState(() =>
    dbSchemaStore.getSchemaList(database.name)
  );
  const tableList = useVueState(() =>
    dbSchemaStore.getTableList({
      database: database.name,
      schema: selectedSchemaName,
    })
  );
  const viewList = useVueState(() =>
    dbSchemaStore.getViewList({
      database: database.name,
      schema: selectedSchemaName,
    })
  );
  const extensionList = useVueState(() =>
    dbSchemaStore.getExtensionList(database.name)
  );
  const externalTableList = useVueState(() =>
    dbSchemaStore.getExternalTableList({
      database: database.name,
      schema: selectedSchemaName,
    })
  );
  const functionList = useVueState(() =>
    dbSchemaStore.getFunctionList({
      database: database.name,
      schema: selectedSchemaName,
    })
  );
  const routeTable = useVueState(() => {
    const table = router.currentRoute.value.query.table;
    return typeof table === "string" ? table : "";
  });
  const databaseMetadata = useVueState(() =>
    dbSchemaStore.getDatabaseMetadata(database.name)
  );
  const hasSensitiveDataFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_DATA_MASKING).value
  );
  const databaseCatalog = useDatabaseCatalog(database.name, false);
  const catalog = useVueState(() => databaseCatalog.value);
  const project = getDatabaseProject(database);
  const classificationConfig = useVueState(() =>
    settingStore.getProjectClassification(
      project.dataClassificationConfigId ?? ""
    )
  );
  const canUpdateCatalog = hasProjectPermissionV2(
    project,
    "bb.databaseCatalogs.update"
  );
  const [selectedTableName, setSelectedTableName] = useState(routeTable);

  const selectedSchemaMetadata = databaseMetadata.schemas.find(
    (schema) => schema.name === selectedSchemaName
  );
  const sequenceList: SequenceMetadata[] = supportsSchema
    ? (selectedSchemaMetadata?.sequences ?? [])
    : databaseMetadata.schemas.flatMap((schema) => schema.sequences ?? []);
  const streamList: StreamMetadata[] = supportsSchema
    ? (selectedSchemaMetadata?.streams ?? [])
    : databaseMetadata.schemas.flatMap((schema) => schema.streams ?? []);
  const taskList: TaskMetadata[] = supportsSchema
    ? (selectedSchemaMetadata?.tasks ?? [])
    : databaseMetadata.schemas.flatMap((schema) => schema.tasks ?? []);
  const packageList: PackageMetadata[] = supportsSchema
    ? (selectedSchemaMetadata?.packages ?? [])
    : databaseMetadata.schemas.flatMap((schema) => schema.packages ?? []);

  const selectedTable = tableList.find(
    (table) => table.name === selectedTableName
  );
  const selectedTableCatalog = selectedTable
    ? getTableCatalog(catalog, selectedSchemaName, selectedTable.name)
    : undefined;
  const showSemanticTypeColumn =
    hasSensitiveDataFeature &&
    [
      Engine.MYSQL,
      Engine.TIDB,
      Engine.POSTGRES,
      Engine.REDSHIFT,
      Engine.ORACLE,
      Engine.SNOWFLAKE,
      Engine.MSSQL,
      Engine.BIGQUERY,
      Engine.SPANNER,
      Engine.CASSANDRA,
      Engine.TRINO,
    ].includes(databaseEngine);
  const selectedTableDetail: TableDetailDialogData | undefined = selectedTable
    ? {
        database,
        editable: canUpdateCatalog,
        classification: selectedTableCatalog?.classification,
        classificationConfig,
        collation: selectedTable.collation,
        columns: selectedTable.columns.map((column) => {
          const columnCatalog = getColumnCatalog(
            catalog,
            selectedSchemaName,
            selectedTable.name,
            column.name
          );
          return {
            name: column.name,
            semanticType: columnCatalog?.semanticType,
            classification: columnCatalog?.classification,
            type: column.type,
            defaultValue: getColumnDefaultValuePlaceholder(column),
            nullable: column.nullable,
            characterSet: column.characterSet,
            collation: column.collation,
            comment: column.comment,
          };
        }),
        dataSize: bytesToString(Number(selectedTable.dataSize)),
        engine: selectedTable.engine,
        indexes: (selectedTable.indexes ?? []).map((index) => ({
          name: index.name,
          expressions: index.expressions,
          unique: index.unique,
          visible: index.visible,
          comment: index.comment,
        })),
        indexSize: bytesToString(Number(selectedTable.indexSize)),
        partitions: (selectedTable.partitions ?? []).map(
          function mapPartition(
            partition
          ): NonNullable<TableDetailDialogData["partitions"]>[number] {
            return {
              name: partition.name,
              type:
                TablePartitionMetadata_Type[partition.type]
                  ?.replace("TYPE_UNSPECIFIED", "UNKNOWN")
                  .replaceAll("_", " ") || "UNKNOWN",
              expression: partition.expression,
              children: (partition.subpartitions ?? []).map(mapPartition),
            };
          }
        ),
        name:
          supportsSchema && selectedSchemaName
            ? `"${selectedSchemaName}"."${selectedTable.name}"`
            : selectedTable.name,
        rowCount: String(selectedTable.rowCount),
        schema: selectedSchemaName,
        showCharacterSet: databaseEngine !== Engine.POSTGRES,
        showColumnClassification: hasSensitiveDataFeature,
        showColumnCollation:
          databaseEngine !== Engine.CLICKHOUSE &&
          databaseEngine !== Engine.SNOWFLAKE,
        showColumns: instanceV1SupportsColumn(databaseEngine),
        showCollation: instanceV1HasCollationAndCharacterSet(databaseEngine),
        showEngine: hasTableEngineProperty(databaseEngine),
        showIndexComment: databaseEngine !== Engine.MONGODB,
        showIndexes: instanceV1SupportsIndex(databaseEngine),
        showIndexSize: hasIndexSizeProperty(databaseEngine),
        showIndexVisible:
          databaseEngine !== Engine.POSTGRES &&
          databaseEngine !== Engine.MONGODB,
        showPartitionTables:
          databaseEngine === Engine.POSTGRES &&
          (selectedTable.partitions?.length ?? 0) > 0,
        showSemanticType: showSemanticTypeColumn,
        showTriggers:
          instanceV1SupportsTrigger(databaseEngine) &&
          (selectedTable.triggers?.length ?? 0) > 0,
        tableName: selectedTable.name,
        triggers: (selectedTable.triggers ?? []).map((trigger) => ({
          name: trigger.name,
          event: trigger.event,
          timing: trigger.timing,
          body: trigger.body,
          sqlMode: trigger.sqlMode,
        })),
      }
    : undefined;

  useEffect(() => {
    void settingStore.getOrFetchSettingByName(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );
  }, [settingStore]);

  useEffect(() => {
    setSelectedTableName((current) =>
      current === routeTable ? current : routeTable
    );
  }, [routeTable]);

  useEffect(() => {
    if (!selectedTableName || loading || selectedTable) {
      return;
    }

    setSelectedTableName("");
  }, [loading, selectedTable, selectedTableName]);

  useEffect(() => {
    const currentQuery = router.currentRoute.value.query;
    const currentTable =
      typeof currentQuery.table === "string" ? currentQuery.table : "";

    if (currentTable === selectedTableName) {
      return;
    }

    void router.replace({
      query: {
        ...currentQuery,
        table: selectedTableName || undefined,
      },
    });
  }, [selectedTableName]);

  const viewRows: ObjectSectionRow[] = viewList.map((view) => ({
    key: view.name,
    name: view.name,
    description: view.definition || "-",
    comment: view.comment,
  }));

  const externalTableRows: ObjectSectionRow[] = externalTableList
    .filter((table) => filterByKeyword(table.name, externalTableSearchKeyword))
    .map((table) => ({
      key: table.name,
      name: table.name,
      description:
        [table.externalServerName, table.externalDatabaseName]
          .filter(Boolean)
          .join(" / ") || "-",
    }));

  const extensionRows: ObjectSectionRow[] = extensionList.map((extension) => ({
    key: extension.name,
    name: extension.name,
    description: extension.version || "-",
    comment: extension.description,
  }));

  const functionRows: ObjectSectionRow[] = functionList.map((fn) => ({
    key: fn.signature || fn.name,
    name: fn.signature || fn.name,
    description: fn.definition || "-",
    comment: fn.comment,
  }));

  const sequenceRows: ObjectSectionRow[] = sequenceList.map((sequence) => ({
    key: sequence.name,
    name: sequence.name,
    description: sequence.dataType || "-",
    comment: sequence.comment,
  }));

  const streamRows: ObjectSectionRow[] = streamList.map((stream) => ({
    key: stream.name,
    name: stream.name,
    description: stream.tableName || "-",
    comment: stream.comment,
  }));

  const taskRows: ObjectSectionRow[] = taskList.map((task) => ({
    key: task.name,
    name: task.name,
    description: task.schedule || task.id || "-",
    comment: task.comment,
  }));

  const packageRows: ObjectSectionRow[] = packageList.map((pkg) => ({
    key: pkg.name,
    name: pkg.name,
    description: pkg.definition || "-",
  }));

  const selectedSchemaLabel =
    schemaList.find((schema) => schema.name === selectedSchemaName)?.name ||
    (selectedSchemaName === "" ? t("db.schema.default") : selectedSchemaName);

  return (
    <div className="flex flex-col gap-6 pt-6">
      {supportsSchema && (
        <div className="flex flex-wrap items-center gap-x-2 gap-y-2">
          <label
            className="text-lg font-medium text-main"
            htmlFor="schema-select"
          >
            {t("common.schema")}
          </label>
          <Select
            disabled={loading}
            value={selectedSchemaName}
            onValueChange={(value) =>
              onSelectedSchemaNameChange(String(value).trim())
            }
          >
            <SelectTrigger id="schema-select" className="min-w-48">
              <SelectValue>{selectedSchemaLabel}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {schemaList.map((schema) => (
                <SelectItem key={schema.name} value={schema.name}>
                  {schema.name || t("db.schema.default")}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      {databaseEngine !== Engine.REDIS && (
        <>
          <section className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="text-lg font-medium text-main">
                {databaseEngine === Engine.MONGODB
                  ? t("db.collections")
                  : t("db.tables")}
              </div>
              <Input
                className="w-full max-w-sm"
                disabled={loading}
                placeholder={t("common.filter-by-name")}
                value={tableSearchKeyword}
                onChange={(event) =>
                  onTableSearchKeywordChange(event.target.value)
                }
              />
            </div>
            <TableMetadataTable
              database={database}
              loading={loading}
              rows={tableList.filter((table) =>
                filterByKeyword(table.name, tableSearchKeyword)
              )}
              schemaName={selectedSchemaName}
              onRowClick={(table) => setSelectedTableName(table.name)}
            />
          </section>

          <section className="flex flex-col gap-4">
            <div className="text-lg font-medium text-main">{t("db.views")}</div>
            <ObjectSectionTable loading={loading} rows={viewRows} />
          </section>

          {(databaseEngine === Engine.POSTGRES ||
            databaseEngine === Engine.HIVE) && (
            <section className="flex flex-col gap-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="text-lg font-medium text-main">
                  {t("db.external-tables")}
                </div>
                <Input
                  className="w-full max-w-sm"
                  disabled={loading}
                  placeholder={t("common.filter-by-name")}
                  value={externalTableSearchKeyword}
                  onChange={(event) =>
                    onExternalTableSearchKeywordChange(event.target.value)
                  }
                />
              </div>
              <ObjectSectionTable loading={loading} rows={externalTableRows} />
            </section>
          )}

          {databaseEngine === Engine.POSTGRES && (
            <section className="flex flex-col gap-4">
              <div className="text-lg font-medium text-main">
                {t("db.extensions")}
              </div>
              <ObjectSectionTable loading={loading} rows={extensionRows} />
            </section>
          )}

          {(databaseEngine === Engine.POSTGRES ||
            databaseEngine === Engine.MSSQL) && (
            <section className="flex flex-col gap-4">
              <div className="text-lg font-medium text-main">
                {t("db.functions")}
              </div>
              <ObjectSectionTable loading={loading} rows={functionRows} />
            </section>
          )}

          {instanceV1SupportsSequence(databaseEngine) && (
            <section className="flex flex-col gap-4">
              <div className="text-lg font-medium text-main">
                {t("db.sequences")}
              </div>
              <ObjectSectionTable loading={loading} rows={sequenceRows} />
            </section>
          )}

          {databaseEngine === Engine.SNOWFLAKE && (
            <>
              <section className="flex flex-col gap-4">
                <div className="text-lg font-medium text-main">
                  {t("db.streams")}
                </div>
                <ObjectSectionTable loading={loading} rows={streamRows} />
              </section>
              <section className="flex flex-col gap-4">
                <div className="text-lg font-medium text-main">
                  {t("db.tasks")}
                </div>
                <ObjectSectionTable loading={loading} rows={taskRows} />
              </section>
            </>
          )}

          {instanceV1SupportsPackage(databaseEngine) && (
            <section className="flex flex-col gap-4">
              <div className="text-lg font-medium text-main">
                {t("db.packages")}
              </div>
              <ObjectSectionTable loading={loading} rows={packageRows} />
            </section>
          )}
        </>
      )}

      <TableDetailDialog
        open={!!selectedTableName}
        table={selectedTableDetail}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedTableName("");
          }
        }}
      />
    </div>
  );
}
