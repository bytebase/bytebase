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
  ExternalTableMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useViewStateNav } from "../common/useViewStateNav";

interface ExternalTablesTableProps {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  externalTables: ExternalTableMetadata[];
  keyword?: string;
  onSelect: (selected: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    externalTable: ExternalTableMetadata;
  }) => void;
}

export function ExternalTablesTable({
  database,
  schema,
  externalTables,
  keyword,
  onSelect,
}: ExternalTablesTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return externalTables;
    return externalTables.filter((externalTable) =>
      externalTable.name.toLowerCase().includes(trimmed)
    );
  }, [externalTables, keyword]);

  const selectedKey = detail?.externalTable;

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
            <TableHead>{t("database.external-server-name")}</TableHead>
            <TableHead>{t("database.external-database-name")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((externalTable) => (
            <TableRow
              key={externalTable.name}
              data-key={externalTable.name}
              data-state={
                selectedKey === externalTable.name ? "selected" : undefined
              }
              className="cursor-pointer"
              onClick={() => onSelect({ database, schema, externalTable })}
            >
              <TableCell className="truncate">
                <HighlightLabelText
                  text={externalTable.name}
                  keyword={keyword}
                />
              </TableCell>
              <TableCell className="truncate">
                {externalTable.externalServerName}
              </TableCell>
              <TableCell className="truncate">
                {externalTable.externalDatabaseName}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
