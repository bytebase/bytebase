import { useMemo } from "react";
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
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, hasSchemaProperty } from "@/utils";
import { useViewStateNav } from "../useViewStateNav";

interface DependencyColumnsTableProps {
  db: Database;
  view: ViewMetadata;
  keyword?: string;
}

/**
 * ViewsPanel detail "Dependency Columns" tab. Clicking a row hops the
 * panel into TablesPanel detail focused on that table + column.
 */
export function DependencyColumnsTable({
  db,
  view,
  keyword,
}: DependencyColumnsTableProps) {
  const { t } = useTranslation();
  const { updateViewState } = useViewStateNav();

  const showSchema = hasSchemaProperty(getInstanceResource(db).engine);

  const filtered = useMemo(() => {
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return view.dependencyColumns;
    return view.dependencyColumns.filter(
      (dep) =>
        dep.column.toLowerCase().includes(trimmed) ||
        dep.table.toLowerCase().includes(trimmed) ||
        dep.schema.toLowerCase().includes(trimmed)
    );
  }, [view.dependencyColumns, keyword]);

  return (
    <div className="w-full h-full overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            {showSchema ? <TableHead>{t("common.schema")}</TableHead> : null}
            <TableHead>{t("common.table")}</TableHead>
            <TableHead>{t("database.column")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((dep, i) => (
            <TableRow
              key={`${dep.schema}.${dep.table}.${dep.column}-${i}`}
              className="cursor-pointer"
              onClick={() =>
                updateViewState({
                  view: "TABLES",
                  schema: dep.schema,
                  detail: { table: dep.table, column: dep.column },
                })
              }
            >
              {showSchema ? (
                <TableCell className="truncate">
                  <HighlightLabelText text={dep.schema} keyword={keyword} />
                </TableCell>
              ) : null}
              <TableCell className="truncate">
                <HighlightLabelText text={dep.table} keyword={keyword} />
              </TableCell>
              <TableCell className="truncate">
                <HighlightLabelText text={dep.column} keyword={keyword} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
