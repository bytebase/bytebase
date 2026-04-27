import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
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
  Database,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource } from "@/utils";
import { useViewStateNav } from "../useViewStateNav";

interface ForeignKeysTableProps {
  db: Database;
  table: TableMetadata;
  keyword?: string;
}

export function ForeignKeysTable({
  db,
  table,
  keyword,
}: ForeignKeysTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return table.foreignKeys;
    return table.foreignKeys.filter(
      (fk) =>
        fk.name.toLowerCase().includes(trimmed) ||
        fk.columns.some((c) => c.toLowerCase().includes(trimmed)) ||
        fk.referencedSchema.toLowerCase().includes(trimmed) ||
        fk.referencedTable.toLowerCase().includes(trimmed) ||
        fk.referencedColumns.some((c) => c.toLowerCase().includes(trimmed))
    );
  }, [table.foreignKeys, keyword]);

  const showOnDelete = table.foreignKeys.some((fk) => fk.onDelete);
  const showOnUpdate = table.foreignKeys.some((fk) => fk.onUpdate);
  const showMatchType = getInstanceResource(db).engine === Engine.POSTGRES;

  const selectedKey = detail?.foreignKey;

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
            <TableHead>{t("schema-editor.columns")}</TableHead>
            <TableHead>{t("database.foreign-key.reference")}</TableHead>
            {showOnDelete ? <TableHead>ON DELETE</TableHead> : null}
            {showOnUpdate ? <TableHead>ON UPDATE</TableHead> : null}
            {showMatchType ? <TableHead>Match type</TableHead> : null}
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((fk) => (
            <TableRow
              key={fk.name}
              data-key={fk.name}
              data-state={selectedKey === fk.name ? "selected" : undefined}
            >
              <TableCell className="truncate max-w-[200px]">
                <HighlightLabelText text={fk.name} keyword={keyword} />
              </TableCell>
              <TableCell>
                <div className="flex flex-col gap-1">
                  {fk.columns.map((column, i) => (
                    <span key={`${column}-${i}`}>
                      <HighlightLabelText text={column} keyword={keyword} />
                    </span>
                  ))}
                </div>
              </TableCell>
              <TableCell>
                <div className="flex flex-col gap-1">
                  {fk.referencedColumns.map((column, i) => {
                    const parts: string[] = [];
                    if (fk.referencedSchema) parts.push(fk.referencedSchema);
                    parts.push(fk.referencedTable);
                    parts.push(column);
                    return (
                      <span key={`${column}-${i}`}>
                        <HighlightLabelText
                          text={parts.join(".")}
                          keyword={keyword}
                        />
                      </span>
                    );
                  })}
                </div>
              </TableCell>
              {showOnDelete ? <TableCell>{fk.onDelete}</TableCell> : null}
              {showOnUpdate ? <TableCell>{fk.onUpdate}</TableCell> : null}
              {showMatchType ? <TableCell>{fk.matchType}</TableCell> : null}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
