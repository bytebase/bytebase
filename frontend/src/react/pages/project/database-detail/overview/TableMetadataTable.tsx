import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { useVueState } from "@/react/hooks/useVueState";
import { updateTableCatalog } from "@/react/lib/column-data-table/utils";
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
    <div className="overflow-hidden rounded border border-block-border">
      <Table className="min-w-full">
        <TableHeader className="bg-control-bg">
          <TableRow className="hover:bg-control-bg">
            {columns.map((column) => (
              <TableHead key={column.key}>{column.label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody className="bg-background">
          {rows.map((table) => {
            const tableCatalog = catalog
              ? getTableCatalog(catalog, schemaName, table.name)
              : undefined;
            const classification =
              tableCatalog?.classification?.trim() || undefined;

            return (
              <TableRow
                key={`${database.name}.${schemaName}.${table.name}`}
                className={onRowClick ? "cursor-pointer" : undefined}
                role={onRowClick ? "button" : undefined}
                tabIndex={onRowClick ? 0 : undefined}
                onClick={onRowClick ? () => onRowClick(table) : undefined}
                onKeyDown={
                  onRowClick
                    ? (event) => {
                        if (event.target !== event.currentTarget) {
                          return;
                        }
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          onRowClick(table);
                        }
                      }
                    : undefined
                }
              >
                {showSchemaColumn && (
                  <TableCell className="text-main">
                    {schemaName || t("db.schema.default")}
                  </TableCell>
                )}
                <TableCell className="text-main">{table.name}</TableCell>
                {showClassificationColumn && (
                  <TableCell>
                    <div
                      data-testid={`table-row-classification-${table.name}-action`}
                      className="inline-flex"
                      onClick={(event) => event.stopPropagation()}
                      onMouseDown={(event) => event.stopPropagation()}
                      onPointerDown={(event) => event.stopPropagation()}
                    >
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
                    </div>
                  </TableCell>
                )}
                {showEngineColumn && (
                  <TableCell>{table.engine || "-"}</TableCell>
                )}
                {showPartitionedColumn && (
                  <TableCell>
                    {(table.partitions?.length ?? 0) > 0 ? "True" : ""}
                  </TableCell>
                )}
                <TableCell>{String(table.rowCount)}</TableCell>
                <TableCell>{bytesToString(Number(table.dataSize))}</TableCell>
                <TableCell>{bytesToString(Number(table.indexSize))}</TableCell>
                <TableCell>{table.comment || "-"}</TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
