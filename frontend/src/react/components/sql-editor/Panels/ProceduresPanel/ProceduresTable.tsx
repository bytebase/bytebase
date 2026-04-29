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
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon/utils";
import { useViewStateNav } from "../common/useViewStateNav";

interface ProceduresTableProps {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedures: ProcedureMetadata[];
  keyword?: string;
  onSelect: (selected: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    procedure: ProcedureMetadata;
    position: number;
  }) => void;
}

export function ProceduresTable({
  database,
  schema,
  procedures,
  keyword,
  onSelect,
}: ProceduresTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const rows = useMemo(() => {
    const all = procedures.map((procedure, position) => ({
      procedure,
      position,
    }));
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return all;
    return all.filter(({ procedure }) =>
      procedure.name.toLowerCase().includes(trimmed)
    );
  }, [procedures, keyword]);

  const selectedKey = detail?.procedure;

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
          {rows.map(({ procedure, position }) => {
            const rowKey = keyWithPosition(procedure.name, position);
            return (
              <TableRow
                key={rowKey}
                data-key={rowKey}
                className="cursor-pointer"
                data-state={selectedKey === rowKey ? "selected" : undefined}
                onClick={() =>
                  onSelect({ database, schema, procedure, position })
                }
              >
                <TableCell className="truncate">
                  <HighlightLabelText text={procedure.name} keyword={keyword} />
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
