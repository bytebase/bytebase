import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { featureToRef, getTableCatalog, useDatabaseCatalog } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  Database,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { bytesToString, getDatabaseEngine, hasSchemaProperty } from "@/utils";

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
  const databaseEngine = getDatabaseEngine(database);
  const showSchemaColumn = hasSchemaProperty(databaseEngine);
  const showClassificationColumn = useVueState(
    () => featureToRef(PlanFeature.FEATURE_DATA_MASKING).value
  );
  const databaseCatalog = useDatabaseCatalog(database.name, false);
  const catalog = useVueState(() => databaseCatalog.value);

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
    [showClassificationColumn, showPartitionedColumn, showSchemaColumn, t]
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
                    {classification || "-"}
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
