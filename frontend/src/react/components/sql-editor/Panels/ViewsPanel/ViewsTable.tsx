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
import type {
  DatabaseMetadata,
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { EllipsisCell } from "../common/EllipsisCell";
import { useViewStateNav } from "../common/useViewStateNav";

interface ViewsTableProps {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  views: ViewMetadata[];
  keyword?: string;
  onSelect: (selected: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    view: ViewMetadata;
  }) => void;
}

export function ViewsTable({
  database,
  schema,
  views,
  keyword,
  onSelect,
}: ViewsTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return views;
    return views.filter((view) => view.name.toLowerCase().includes(trimmed));
  }, [views, keyword]);

  const selectedKey = detail?.view;

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
            <TableHead>{t("schema-editor.database.name")}</TableHead>
            <TableHead>{t("schema-editor.database.comment")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((view) => (
            <TableRow
              key={view.name}
              data-key={view.name}
              data-state={selectedKey === view.name ? "selected" : undefined}
              className="cursor-pointer"
              onClick={() => onSelect({ database, schema, view })}
            >
              <TableCell className="truncate max-w-[280px]">
                <HighlightLabelText text={view.name} keyword={keyword} />
              </TableCell>
              <TableCell className="truncate max-w-[400px]">
                <EllipsisCell content={view.comment} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
