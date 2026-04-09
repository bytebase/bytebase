import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  Database,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  bytesToString,
  getDatabaseEngine,
  hasSchemaProperty,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
} from "@/utils";
import {
  type ObjectSectionRow,
  ObjectSectionTable,
} from "./ObjectSectionTable";
import { TableDetailDialog } from "./TableDetailDialog";

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
  const databaseMetadata = useVueState(() =>
    dbSchemaStore.getDatabaseMetadata(database.name)
  );
  const [selectedTable, setSelectedTable] = useState<TableMetadata>();

  const currentSchema =
    databaseMetadata.schemas.find(
      (schema) => schema.name === selectedSchemaName
    ) || databaseMetadata.schemas[0];

  const tableRows: ObjectSectionRow[] = tableList
    .filter((table) => filterByKeyword(table.name, tableSearchKeyword))
    .map((table) => ({
      key: table.name,
      name: table.name,
      description: `${String(table.rowCount)} rows, ${bytesToString(Number(table.dataSize))}`,
      comment: table.comment,
      onClick: () => setSelectedTable(table),
    }));

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
    key: fn.name,
    name: fn.name,
    description: fn.signature || fn.definition || "-",
    comment: fn.comment,
  }));

  const sequenceRows: ObjectSectionRow[] = (currentSchema?.sequences || []).map(
    (sequence) => ({
      key: sequence.name,
      name: sequence.name,
      description: sequence.dataType || "-",
      comment: sequence.comment,
    })
  );

  const streamRows: ObjectSectionRow[] = (currentSchema?.streams || []).map(
    (stream) => ({
      key: stream.name,
      name: stream.name,
      description: stream.tableName || "-",
      comment: stream.comment,
    })
  );

  const taskRows: ObjectSectionRow[] = (currentSchema?.tasks || []).map(
    (task) => ({
      key: task.name,
      name: task.name,
      description: task.schedule || task.id || "-",
      comment: task.comment,
    })
  );

  const packageRows: ObjectSectionRow[] = (currentSchema?.packages || []).map(
    (pkg) => ({
      key: pkg.name,
      name: pkg.name,
      description: pkg.definition || "-",
    })
  );

  return (
    <div className="space-y-6 pt-6">
      {supportsSchema && (
        <div className="flex flex-wrap items-center gap-x-2 gap-y-2">
          <label
            className="text-lg font-medium text-main"
            htmlFor="schema-select"
          >
            {t("common.schema")}
          </label>
          <select
            id="schema-select"
            className="min-w-48 rounded-xs border border-control-border bg-white px-3 py-2 text-sm text-main"
            disabled={loading}
            value={selectedSchemaName}
            onChange={(event) =>
              onSelectedSchemaNameChange(event.target.value.trim())
            }
          >
            {schemaList.map((schema) => (
              <option key={schema.name} value={schema.name}>
                {schema.name || t("db.schema.default")}
              </option>
            ))}
          </select>
        </div>
      )}

      {databaseEngine !== Engine.REDIS && (
        <>
          <section className="space-y-4">
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
            <ObjectSectionTable rows={tableRows} />
          </section>

          <section className="space-y-4">
            <div className="text-lg font-medium text-main">{t("db.views")}</div>
            <ObjectSectionTable rows={viewRows} />
          </section>

          {(databaseEngine === Engine.POSTGRES ||
            databaseEngine === Engine.HIVE) && (
            <section className="space-y-4">
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
              <ObjectSectionTable rows={externalTableRows} />
            </section>
          )}

          {databaseEngine === Engine.POSTGRES && (
            <section className="space-y-4">
              <div className="text-lg font-medium text-main">
                {t("db.extensions")}
              </div>
              <ObjectSectionTable rows={extensionRows} />
            </section>
          )}

          {(databaseEngine === Engine.POSTGRES ||
            databaseEngine === Engine.MSSQL) && (
            <section className="space-y-4">
              <div className="text-lg font-medium text-main">
                {t("db.functions")}
              </div>
              <ObjectSectionTable rows={functionRows} />
            </section>
          )}

          {instanceV1SupportsSequence(databaseEngine) && (
            <section className="space-y-4">
              <div className="text-lg font-medium text-main">
                {t("db.sequences")}
              </div>
              <ObjectSectionTable rows={sequenceRows} />
            </section>
          )}

          {databaseEngine === Engine.SNOWFLAKE && (
            <>
              <section className="space-y-4">
                <div className="text-lg font-medium text-main">
                  {t("db.streams")}
                </div>
                <ObjectSectionTable rows={streamRows} />
              </section>
              <section className="space-y-4">
                <div className="text-lg font-medium text-main">
                  {t("db.tasks")}
                </div>
                <ObjectSectionTable rows={taskRows} />
              </section>
            </>
          )}

          {instanceV1SupportsPackage(databaseEngine) && (
            <section className="space-y-4">
              <div className="text-lg font-medium text-main">
                {t("db.packages")}
              </div>
              <ObjectSectionTable rows={packageRows} />
            </section>
          )}
        </>
      )}

      <TableDetailDialog
        open={!!selectedTable}
        table={selectedTable}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedTable(undefined);
          }
        }}
      />
    </div>
  );
}
