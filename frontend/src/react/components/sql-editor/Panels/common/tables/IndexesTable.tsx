import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import type { TableMetadata } from "@/types/proto-es/v1/database_service_pb";
import { EllipsisCell } from "../EllipsisCell";
import { useViewStateNav } from "../useViewStateNav";

interface IndexesTableProps {
  table: TableMetadata;
  keyword?: string;
}

export function IndexesTable({ table, keyword }: IndexesTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return table.indexes;
    return table.indexes.filter(
      (index) =>
        index.name.toLowerCase().includes(trimmed) ||
        index.expressions.some((column) =>
          column.toLowerCase().includes(trimmed)
        )
    );
  }, [table.indexes, keyword]);

  const selectedKey = detail?.index;

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
            <TableHead>{t("schema-editor.column.comment")}</TableHead>
            <TableHead className="w-20">
              {t("schema-editor.column.primary")}
            </TableHead>
            <TableHead className="w-20">
              {t("schema-editor.index.unique")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((index) => (
            <TableRow
              key={index.name}
              data-key={index.name}
              data-state={selectedKey === index.name ? "selected" : undefined}
            >
              <TableCell className="truncate max-w-[200px]">
                <HighlightLabelText text={index.name} keyword={keyword} />
              </TableCell>
              <TableCell className="truncate">
                {index.expressions.map((column, i) => (
                  <span key={`${column}-${i}`}>
                    {i > 0 ? ", " : null}
                    <HighlightLabelText text={column} keyword={keyword} />
                  </span>
                ))}
              </TableCell>
              <TableCell className="truncate max-w-[320px]">
                <EllipsisCell content={index.comment} />
              </TableCell>
              <TableCell className="w-20">
                <Checkbox checked={index.primary} disabled />
              </TableCell>
              <TableCell className="w-20">
                <Checkbox checked={index.unique} disabled />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
