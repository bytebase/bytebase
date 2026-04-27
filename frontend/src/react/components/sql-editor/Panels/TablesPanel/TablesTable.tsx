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
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { bytesToString, getInstanceResource } from "@/utils";
import {
  hasIndexSizeProperty,
  hasTableEngineProperty,
  instanceV1HasCollationAndCharacterSet,
} from "@/utils/v1/instance";
import { EllipsisCell } from "../common/EllipsisCell";
import { useViewStateNav } from "../common/useViewStateNav";

interface TablesTableProps {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  tables: TableMetadata[];
  keyword?: string;
  onSelect: (selected: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
  }) => void;
}

export function TablesTable({
  db,
  database,
  schema,
  tables,
  keyword,
  onSelect,
}: TablesTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return tables;
    return tables.filter((table) => table.name.toLowerCase().includes(trimmed));
  }, [tables, keyword]);

  const instance = useMemo(() => getInstanceResource(db), [db]);
  const showEngine = hasTableEngineProperty(instance);
  const showCollation = instanceV1HasCollationAndCharacterSet(instance);
  const showIndexSize = hasIndexSizeProperty(instance);

  const selectedKey = detail?.table;

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
            {showEngine ? (
              <TableHead>{t("schema-editor.database.engine")}</TableHead>
            ) : null}
            {showCollation ? (
              <TableHead>{t("schema-editor.database.collation")}</TableHead>
            ) : null}
            <TableHead>{t("database.row-count-est")}</TableHead>
            <TableHead>{t("database.data-size")}</TableHead>
            {showIndexSize ? (
              <TableHead>{t("database.index-size")}</TableHead>
            ) : null}
            <TableHead>{t("schema-editor.database.comment")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((table) => (
            <TableRow
              key={table.name}
              data-key={table.name}
              data-state={selectedKey === table.name ? "selected" : undefined}
              className="cursor-pointer"
              onClick={() => onSelect({ database, schema, table })}
            >
              <TableCell className="truncate max-w-[280px]">
                <HighlightLabelText text={table.name} keyword={keyword} />
              </TableCell>
              {showEngine ? (
                <TableCell className="truncate">{table.engine}</TableCell>
              ) : null}
              {showCollation ? (
                <TableCell className="truncate">{table.collation}</TableCell>
              ) : null}
              <TableCell>{String(table.rowCount)}</TableCell>
              <TableCell>{bytesToString(Number(table.dataSize))}</TableCell>
              {showIndexSize ? (
                <TableCell>{bytesToString(Number(table.indexSize))}</TableCell>
              ) : null}
              <TableCell className="truncate max-w-[320px]">
                <EllipsisCell content={table.comment} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
