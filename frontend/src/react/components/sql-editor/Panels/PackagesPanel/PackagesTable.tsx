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
  PackageMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon/utils";
import { useViewStateNav } from "../common/useViewStateNav";

interface PackagesTableProps {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  packages: PackageMetadata[];
  keyword?: string;
  onSelect: (selected: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    pack: PackageMetadata;
    position: number;
  }) => void;
}

export function PackagesTable({
  database,
  schema,
  packages,
  keyword,
  onSelect,
}: PackagesTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const rows = useMemo(() => {
    const all = packages.map((pack, position) => ({ pack, position }));
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return all;
    return all.filter(({ pack }) => pack.name.toLowerCase().includes(trimmed));
  }, [packages, keyword]);

  const selectedKey = detail?.package;

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
          {rows.map(({ pack, position }) => {
            const rowKey = keyWithPosition(pack.name, position);
            return (
              <TableRow
                key={rowKey}
                data-key={rowKey}
                className="cursor-pointer"
                data-state={selectedKey === rowKey ? "selected" : undefined}
                onClick={() => onSelect({ database, schema, pack, position })}
              >
                <TableCell className="truncate">
                  <HighlightLabelText text={pack.name} keyword={keyword} />
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
