import { RotateCcw, Trash2 } from "lucide-react";
import { useCallback, useLayoutEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import {
  distributeColumnWidths,
  useColumnWidths,
} from "@/react/hooks/useColumnWidths";
import { cn } from "@/react/lib/utils";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../../context";
import { refreshTableEditStatus } from "../../core/refreshEditStatus";
import { INLINE_EDIT_INPUT_CLASS } from "../common";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  tables: TableMetadata[];
  searchPattern?: string;
  onEditTable: (table: { name: string }) => void;
}

export function TableList({
  db,
  database,
  schema,
  tables,
  searchPattern,
  onEditTable,
}: Props) {
  const { t } = useTranslation();
  const { readonly, editStatus, targets, tabs } = useSchemaEditorContext();

  const baselineMetadata = useMemo(
    () =>
      targets.find((target) => target.database.name === db.name)
        ?.baselineMetadata,
    [targets, db.name]
  );

  const filteredTables = useMemo(() => {
    if (!searchPattern) return tables;
    const pattern = searchPattern.toLowerCase();
    return tables.filter((t) => t.name.toLowerCase().includes(pattern));
  }, [tables, searchPattern]);

  const getStatus = useCallback(
    (table: TableMetadata) => {
      return editStatus.getTableStatus(db, { schema, table });
    },
    [editStatus, db, schema]
  );

  const handleDrop = useCallback(
    (table: TableMetadata) => {
      const status = getStatus(table);
      if (status === "created") {
        // Remove from array entirely
        const idx = schema.tables.indexOf(table);
        if (idx >= 0) schema.tables.splice(idx, 1);
        editStatus.removeEditStatus(db, { schema, table }, true);
      } else {
        editStatus.markEditStatus(db, { schema, table }, "dropped");
      }
      // Close the table's open tab — its editor is no longer relevant once the
      // table is dropped (and points at a removed object for created tables).
      const tab = tabs.findTab({
        type: "table",
        database: db,
        metadata: { database, schema, table },
      });
      if (tab) tabs.closeTab(tab.id);
    },
    [getStatus, schema, editStatus, db, database, tabs]
  );

  const handleRestore = useCallback(
    (table: TableMetadata) => {
      editStatus.removeEditStatus(db, { schema, table }, false);
    },
    [editStatus, db, schema]
  );

  const handleCommentChange = useCallback(
    (table: TableMetadata, comment: string) => {
      table.comment = comment;
      // Recompute against the baseline so reverting the comment clears dirty.
      if (baselineMetadata) {
        refreshTableEditStatus(editStatus, db, baselineMetadata, schema, table);
      }
    },
    [editStatus, db, baselineMetadata, schema]
  );

  // Resizable column layout. Name/comment can be dragged wider; operations
  // keeps a fixed width. `widths[i]` is positional with `columnDefs`.
  const columnDefs = useMemo(
    () => [
      {
        key: "name",
        title: t("schema-editor.column.name"),
        defaultWidth: 240,
        minWidth: 100,
        resizable: true,
        align: "",
      },
      {
        key: "comment",
        title: t("schema-editor.column.comment"),
        defaultWidth: 360,
        minWidth: 100,
        resizable: true,
        align: "",
      },
      ...(readonly
        ? []
        : [
            {
              key: "operations",
              title: t("schema-editor.column.operations"),
              defaultWidth: 96,
              resizable: false,
              align: "text-right",
            },
          ]),
    ],
    [t, readonly]
  );

  const { widths, totalWidth, onResizeStart, setWidths } =
    useColumnWidths(columnDefs);

  // Fit columns to the available width on first layout (see TableColumnEditor).
  const containerRef = useRef<HTMLDivElement>(null);
  const didFitRef = useRef(false);
  useLayoutEffect(() => {
    if (didFitRef.current) return;
    const width = containerRef.current?.clientWidth ?? 0;
    if (width <= 0) return;
    didFitRef.current = true;
    setWidths(distributeColumnWidths(columnDefs, width));
  }, [columnDefs, setWidths]);

  if (filteredTables.length === 0) {
    return (
      <div className="flex size-full items-center justify-center py-8 text-sm text-control-light">
        {searchPattern
          ? t("schema-editor.table.no-match")
          : t("schema-editor.table.no-tables")}
      </div>
    );
  }

  return (
    <div ref={containerRef} className="w-full overflow-x-auto">
      <Table
        className="w-auto table-fixed"
        style={{ width: `${totalWidth}px` }}
      >
        <colgroup>
          {widths.map((w, i) => (
            <col key={columnDefs[i].key} style={{ width: `${w}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow>
            {columnDefs.map((col, colIdx) => (
              <TableHead
                key={col.key}
                className={cn("h-8 py-1 whitespace-nowrap", col.align)}
                resizable={col.resizable}
                onResizeStart={
                  col.resizable ? (e) => onResizeStart(colIdx, e) : undefined
                }
              >
                {col.title}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {filteredTables.map((table) => {
            const status = getStatus(table);
            return (
              <TableRow
                key={table.name}
                className={cn(
                  "cursor-pointer",
                  status === "created" && "text-success",
                  status === "updated" && "text-warning",
                  status === "dropped" && "text-error line-through"
                )}
                onClick={() => onEditTable(table)}
              >
                <TableCell className="py-2 font-medium">{table.name}</TableCell>
                <TableCell
                  className="py-2 text-control-light"
                  onClick={(e) => e.stopPropagation()}
                >
                  {readonly ? (
                    table.comment || "—"
                  ) : (
                    <Input
                      value={table.comment}
                      disabled={status === "dropped"}
                      size="xs"
                      className={INLINE_EDIT_INPUT_CLASS}
                      onChange={(e) =>
                        handleCommentChange(table, e.target.value)
                      }
                    />
                  )}
                </TableCell>
                {!readonly && (
                  <TableCell className="py-2 text-right">
                    {status === "dropped" ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="size-7 p-0"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleRestore(table);
                        }}
                      >
                        <RotateCcw className="size-3.5" />
                      </Button>
                    ) : (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="size-7 p-0 text-error hover:text-error"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDrop(table);
                        }}
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    )}
                  </TableCell>
                )}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
