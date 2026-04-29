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
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon/utils";
import { useViewStateNav } from "../common/useViewStateNav";

interface FunctionsTableProps {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  funcs: FunctionMetadata[];
  keyword?: string;
  onSelect: (selected: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    func: FunctionMetadata;
    position: number;
  }) => void;
}

export function FunctionsTable({
  database,
  schema,
  funcs,
  keyword,
  onSelect,
}: FunctionsTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const rows = useMemo(() => {
    const all = funcs.map((func, position) => ({ func, position }));
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return all;
    return all.filter(({ func }) => func.name.toLowerCase().includes(trimmed));
  }, [funcs, keyword]);

  const selectedKey = detail?.func;

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
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map(({ func, position }) => {
            const rowKey = keyWithPosition(func.name, position);
            return (
              <TableRow
                key={rowKey}
                data-key={rowKey}
                className="cursor-pointer"
                data-state={selectedKey === rowKey ? "selected" : undefined}
                onClick={() => onSelect({ database, schema, func, position })}
              >
                <TableCell className="truncate">
                  <HighlightLabelText text={func.name} keyword={keyword} />
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
