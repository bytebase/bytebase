import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { updateTableCatalog } from "@/components/ColumnDataTable/utils";
import { Button } from "@/react/components/ui/button";
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
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  bytesToString,
  getDatabaseEngine,
  getDatabaseProject,
  hasProjectPermissionV2,
  hasSchemaProperty,
} from "@/utils";
import { EditableClassificationCell } from "./TableDetailDialog";

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
  const editable = hasProjectPermissionV2(
    project,
    "bb.databaseCatalogs.update"
  );

  const showEngineColumn = databaseEngine !== Engine.POSTGRES;
  const showPartitionedColumn = databaseEngine === Engine.POSTGRES;

  useEffect(() => {
    void settingStore.getOrFetchSettingByName(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );
  }, [settingStore]);

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
        onRowClick
          ? { key: "operations", label: t("common.operations") }
          : undefined,
      ].filter((column) => column !== undefined),
    [
      onRowClick,
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
                className="hover:bg-control-bg"
              >
                {showSchemaColumn && (
                  <td className="px-4 py-3 text-sm text-main">
                    {schemaName || t("db.schema.default")}
                  </td>
                )}
                <td className="px-4 py-3 text-sm text-main">{table.name}</td>
                {showClassificationColumn && (
                  <td className="px-4 py-3 text-sm text-control">
                    <EditableClassificationCell
                      classification={classification}
                      classificationConfig={classificationConfig}
                      readonly={!editable}
                      testIdPrefix={`table-row-classification-${table.name}`}
                      onApply={async (classificationId) => {
                        await updateTableCatalog({
                          database: database.name,
                          schema: schemaName,
                          table: table.name,
                          tableCatalog: {
                            classification: classificationId,
                          },
                        });
                      }}
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
                {onRowClick && (
                  <td className="px-4 py-3 text-sm text-control">
                    <div className="flex items-center gap-x-2">
                      <Button
                        size="sm"
                        variant="ghost"
                        data-testid={`table-row-view-${table.name}`}
                        onClick={() => onRowClick(table)}
                      >
                        {t("common.view-details")}
                      </Button>
                    </div>
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
