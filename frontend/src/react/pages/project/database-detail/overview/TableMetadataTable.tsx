import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import {
  featureToRef,
  getTableCatalog,
  useDatabaseCatalog,
  useSettingV1Store,
} from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  Database,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  bytesToString,
  getDatabaseEngine,
  getDatabaseProject,
  hasSchemaProperty,
} from "@/utils";

const BG_COLORS = [
  "bg-green-200",
  "bg-yellow-200",
  "bg-orange-300",
  "bg-amber-500",
  "bg-red-500",
];

function ClassificationBadge({
  classificationId,
  classificationConfig,
}: {
  classificationId: string | undefined;
  classificationConfig:
    | DataClassificationSetting_DataClassificationConfig
    | undefined;
}) {
  if (!classificationId || !classificationConfig) {
    return <span>-</span>;
  }
  const entry = classificationConfig.classification[classificationId];
  if (!entry) {
    return <span>{classificationId}</span>;
  }
  const level = (classificationConfig.levels ?? []).find(
    (l) => l.level === entry.level
  );
  const levelColor = BG_COLORS[(entry.level ?? 0) - 1] ?? "bg-gray-200";
  return (
    <span className="inline-flex items-center gap-x-1">
      <span className="truncate">{entry.title}</span>
      {level && (
        <span className={`shrink-0 rounded px-1 py-0.5 text-xs ${levelColor}`}>
          {level.title}
        </span>
      )}
    </span>
  );
}

export function TableMetadataTable({
  database,
  schemaName,
  rows,
  loading = false,
  onRowClick,
}: {
  database: Database;
  schemaName: string;
  rows: TableMetadata[];
  loading?: boolean;
  onRowClick?: (table: TableMetadata) => void;
}) {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();
  const databaseEngine = getDatabaseEngine(database);
  const showSchemaColumn = hasSchemaProperty(databaseEngine);
  const showClassificationColumn = useVueState(
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

  const showEngineColumn = databaseEngine !== Engine.POSTGRES;
  const showPartitionedColumn = databaseEngine === Engine.POSTGRES;
  const columns = useMemo(
    () =>
      [
        showSchemaColumn
          ? { key: "schema", label: t("common.schema") }
          : undefined,
        { key: "name", label: t("common.name") },
        showClassificationColumn
          ? {
              key: "classification",
              label: t("database.classification.self"),
            }
          : undefined,
        showEngineColumn
          ? {
              key: "engine",
              label: t("database.engine"),
            }
          : undefined,
        showPartitionedColumn
          ? {
              key: "partitioned",
              label: t("database.partitioned"),
            }
          : undefined,
        { key: "rowCount", label: t("database.row-count-est") },
        { key: "dataSize", label: t("database.data-size") },
        { key: "indexSize", label: t("database.index-size") },
        { key: "comment", label: t("common.comment") },
      ].filter((column) => column !== undefined),
    [
      showClassificationColumn,
      showEngineColumn,
      showPartitionedColumn,
      showSchemaColumn,
      t,
    ]
  );

  if (loading) {
    return (
      <div className="rounded-lg border border-dashed border-block-border px-4 py-6 text-sm text-control-light">
        {t("common.loading")}
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-block-border px-4 py-6 text-sm text-control-light">
        -
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border border-block-border">
      <table className="min-w-full divide-y divide-block-border">
        <thead className="bg-control-bg">
          <tr className="text-left text-sm text-control-light">
            {columns.map((column) => (
              <th key={column.key} className="px-4 py-2 font-medium">
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-block-border bg-white">
          {rows.map((table) => {
            const tableCatalog = catalog
              ? getTableCatalog(catalog, schemaName, table.name)
              : undefined;
            const classification =
              tableCatalog?.classification?.trim() || undefined;

            return (
              <tr
                key={`${database.name}.${schemaName}.${table.name}`}
                className={
                  onRowClick ? "cursor-pointer hover:bg-control-bg" : ""
                }
                role={onRowClick ? "button" : undefined}
                tabIndex={onRowClick ? 0 : undefined}
                onClick={() => onRowClick?.(table)}
                onKeyDown={
                  onRowClick
                    ? (event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          onRowClick(table);
                        }
                      }
                    : undefined
                }
              >
                {showSchemaColumn && (
                  <td className="px-4 py-3 text-sm text-main">
                    {schemaName || t("db.schema.default")}
                  </td>
                )}
                <td className="px-4 py-3 text-sm text-main">{table.name}</td>
                {showClassificationColumn && (
                  <td className="px-4 py-3 text-sm text-control">
                    <ClassificationBadge
                      classificationId={classification}
                      classificationConfig={classificationConfig}
                    />
                  </td>
                )}
                {showEngineColumn && (
                  <td className="px-4 py-3 text-sm text-control">
                    {table.engine || "-"}
                  </td>
                )}
                {showPartitionedColumn && (
                  <td className="px-4 py-3 text-sm text-control">
                    {(table.partitions?.length ?? 0) > 0 ? "True" : ""}
                  </td>
                )}
                <td className="px-4 py-3 text-sm text-control">
                  {String(table.rowCount)}
                </td>
                <td className="px-4 py-3 text-sm text-control">
                  {bytesToString(Number(table.dataSize))}
                </td>
                <td className="px-4 py-3 text-sm text-control">
                  {bytesToString(Number(table.indexSize))}
                </td>
                <td className="px-4 py-3 text-sm text-control">
                  {table.comment || "-"}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
