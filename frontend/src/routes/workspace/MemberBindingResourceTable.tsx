import { useCallback, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { TableCellContent } from "@/components/ui/table-cell-content";
import {
  distributeColumnWidths,
  fillableWidth,
  useColumnWidths,
} from "@/hooks/useColumnWidths";
import type { DatabaseResource } from "@/types";
import { MemberDatabaseResourceName } from "./MemberDatabaseResourceName";

const COLUMNS = [
  { key: "database", defaultWidth: 280, minWidth: 160 },
  { key: "schema", defaultWidth: 200, minWidth: 120 },
  { key: "table", defaultWidth: 240, minWidth: 120 },
] as const;

// One role binding's databases, schemas and tables. An empty list is a binding
// with no resource condition, which reaches every database.
export function MemberBindingResourceTable({
  resources,
}: {
  resources: DatabaseResource[];
}) {
  const { t } = useTranslation();
  const titles = [t("common.database"), t("common.schema"), t("common.table")];
  const columns = useMemo(() => [...COLUMNS], []);
  const { widths, totalWidth, onResizeStart, setWidths } =
    useColumnWidths(columns);

  // Fitted once, to this card's scroller, so every card opens with the same
  // columns; a later drag can widen this card past its edge.
  const didFitRef = useRef(false);
  const fitToCard = useCallback(
    (node: HTMLDivElement | null) => {
      if (!node || didFitRef.current) return;
      const width = fillableWidth(node);
      if (width <= 0) return;
      didFitRef.current = true;
      setWidths(distributeColumnWidths(columns, width));
    },
    [columns, setWidths]
  );

  const rows: (DatabaseResource | undefined)[] =
    resources.length > 0 ? resources : [undefined];

  return (
    <div ref={fitToCard} className="overflow-x-auto">
      <Table
        className="w-auto table-fixed"
        style={{ width: `${totalWidth}px` }}
      >
        <colgroup>
          {widths.map((w, i) => (
            <col key={columns[i].key} style={{ width: `${w}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow>
            {columns.map((col, i) => (
              <TableHead
                key={col.key}
                resizable
                onResizeStart={(e) => onResizeStart(i, e)}
              >
                {titles[i]}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((resource, i) => (
            <TableRow key={i}>
              <TableCell>
                <TableCellContent>
                  <MemberDatabaseResourceName resource={resource} />
                </TableCellContent>
              </TableCell>
              <TableCell>
                <EllipsisText text={resource?.schema ?? "*"} />
              </TableCell>
              <TableCell>
                <EllipsisText text={resource?.table ?? "*"} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
