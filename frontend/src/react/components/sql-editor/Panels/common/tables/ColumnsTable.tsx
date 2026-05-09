import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { engineSupportsMultiSchema } from "@/react/components/SchemaEditorLite/core/spec";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource } from "@/utils";
import { EllipsisCell } from "../EllipsisCell";
import { useViewStateNav } from "../useViewStateNav";

export type ColumnsTableFlavor = "table" | "view" | "external-table";

interface ColumnsTableProps {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  /** Required when `flavor === "table"` so primary-key + FK columns can resolve. */
  table?: TableMetadata;
  columns: ColumnMetadata[];
  flavor: ColumnsTableFlavor;
  keyword?: string;
}

/**
 * Generic columns table covering TablesPanel, ViewsPanel, and
 * ExternalTablesPanel detail surfaces. The `flavor` prop selects the
 * column header set.
 */
export function ColumnsTable({
  db,
  database,
  schema,
  table,
  columns,
  flavor,
  keyword,
}: ColumnsTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return columns;
    return columns.filter((column) =>
      column.name.toLowerCase().includes(trimmed)
    );
  }, [columns, keyword]);

  const engine = getInstanceResource(db).engine;
  const showOnUpdate =
    flavor === "table" && (engine === Engine.MYSQL || engine === Engine.TIDB);
  const showPrimary = flavor === "table";
  const showForeignKey = flavor === "table";
  const primaryKey = useMemo(() => {
    return flavor === "table"
      ? table?.indexes.find((idx) => idx.primary)
      : undefined;
  }, [flavor, table]);

  const selectedKey = detail?.column;

  useEffect(() => {
    if (!selectedKey) return;
    const el = containerRef.current?.querySelector<HTMLElement>(
      `[data-key="${CSS.escape(selectedKey)}"]`
    );
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedKey]);

  return (
    <div ref={containerRef} className="w-full h-full overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("schema-editor.column.name")}</TableHead>
            <TableHead>{t("schema-editor.column.type")}</TableHead>
            <TableHead>{t("schema-editor.column.default")}</TableHead>
            {showOnUpdate ? (
              <TableHead>{t("schema-editor.column.on-update")}</TableHead>
            ) : null}
            <TableHead>{t("schema-editor.column.comment")}</TableHead>
            <TableHead className="w-20">
              {t("schema-editor.column.not-null")}
            </TableHead>
            {showPrimary ? (
              <TableHead className="w-20">
                {t("schema-editor.column.primary")}
              </TableHead>
            ) : null}
            {showForeignKey ? (
              <TableHead>{t("schema-editor.column.foreign-key")}</TableHead>
            ) : null}
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((column) => (
            <TableRow
              key={column.name}
              data-key={column.name}
              data-state={selectedKey === column.name ? "selected" : undefined}
            >
              <TableCell className="truncate max-w-[200px]">
                <HighlightLabelText text={column.name} keyword={keyword} />
              </TableCell>
              <TableCell className="truncate max-w-[320px]">
                {column.type}
              </TableCell>
              <TableCell className="truncate max-w-[320px]">
                {column.default ? (
                  <span className="text-control">{column.default}</span>
                ) : (
                  <span className="italic text-control-placeholder">
                    {t("schema-editor.default.placeholder")}
                  </span>
                )}
              </TableCell>
              {showOnUpdate ? (
                <TableCell className="truncate max-w-[320px]">
                  <EllipsisCell content={column.onUpdate} />
                </TableCell>
              ) : null}
              <TableCell className="truncate max-w-[320px]">
                <EllipsisCell content={column.comment} />
              </TableCell>
              <TableCell className="w-20">
                <Checkbox checked={!column.nullable} disabled />
              </TableCell>
              {showPrimary ? (
                <TableCell className="w-20">
                  <Checkbox
                    checked={!!primaryKey?.expressions.includes(column.name)}
                    disabled
                  />
                </TableCell>
              ) : null}
              {showForeignKey ? (
                <TableCell>
                  <ForeignKeyDisplay
                    db={db}
                    database={database}
                    schema={schema}
                    table={table}
                    column={column}
                  />
                </TableCell>
              ) : null}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function ForeignKeyDisplay({
  db,
  database,
  table,
  column,
}: {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
  column: ColumnMetadata;
}) {
  const fks = useMemo(() => {
    if (!table) return [];
    return table.foreignKeys.filter((fk) => fk.columns.includes(column.name));
  }, [table, column.name]);

  if (fks.length === 0) {
    return <span className="italic text-control-placeholder">EMPTY</span>;
  }

  const multiSchema = engineSupportsMultiSchema(getInstanceResource(db).engine);

  return (
    <div className="flex flex-col gap-1">
      {fks.map((fk) => {
        const position = fk.columns.indexOf(column.name);
        if (position < 0) return null;

        const referencedSchema = database.schemas.find(
          (s) => s.name === fk.referencedSchema
        );
        const referencedTable = referencedSchema?.tables.find(
          (t) => t.name === fk.referencedTable
        );
        const referencedColumn = referencedTable?.columns.find(
          (c) => c.name === fk.referencedColumns[position]
        );
        const label = multiSchema
          ? `${referencedSchema?.name ?? ""}.${referencedTable?.name ?? ""}(${referencedColumn?.name ?? ""})`
          : `${referencedTable?.name ?? ""}(${referencedColumn?.name ?? ""})`;
        return (
          <span key={fk.name} className="break-all">
            {label}
          </span>
        );
      })}
    </div>
  );
}
