import { RotateCcw, Trash2 } from "lucide-react";
import { useCallback, useLayoutEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
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
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../../context";
import {
  removeColumnPrimaryKey,
  upsertColumnPrimaryKey,
} from "../../core/edit";
import { refreshTableEditStatus } from "../../core/refreshEditStatus";
import { INLINE_EDIT_INPUT_CLASS, markUUID } from "../common";
import { DataTypeCell } from "./DataTypeCell";
import { DefaultValueCell } from "./DefaultValueCell";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  engine: Engine;
  readonly: boolean;
  disableChangeTable: boolean;
  allowChangePrimaryKeys: boolean;
  searchPattern?: string;
}

export function TableColumnEditor({
  db,
  database: _database,
  schema,
  table,
  engine,
  readonly: isReadonly,
  disableChangeTable,
  allowChangePrimaryKeys,
  searchPattern,
}: Props) {
  const { t } = useTranslation();
  const { editStatus, targets } = useSchemaEditorContext();

  const baselineMetadata = useMemo(
    () =>
      targets.find((target) => target.database.name === db.name)
        ?.baselineMetadata,
    [targets, db.name]
  );

  // Recompute this table's edit status by diffing against the baseline, so
  // reverting a field back to its original value clears the dirty marker.
  const refreshStatus = useCallback(() => {
    if (!baselineMetadata) return;
    refreshTableEditStatus(editStatus, db, baselineMetadata, schema, table);
  }, [editStatus, db, baselineMetadata, schema, table]);

  const primaryKey = useMemo(() => {
    return table.indexes.find((idx) => idx.primary);
  }, [table.indexes]);

  const isColumnPrimaryKey = useCallback(
    (column: ColumnMetadata) => {
      return primaryKey?.expressions.includes(column.name) ?? false;
    },
    [primaryKey]
  );

  const columns = useMemo(() => {
    if (!searchPattern) return table.columns;
    const pattern = searchPattern.toLowerCase();
    return table.columns.filter((c) => c.name.toLowerCase().includes(pattern));
  }, [table.columns, searchPattern]);

  const getColumnKey = useCallback((column: ColumnMetadata) => {
    return markUUID(column);
  }, []);

  const getStatus = useCallback(
    (column: ColumnMetadata) => {
      return editStatus.getColumnStatus(db, { schema, table, column });
    },
    [editStatus, db, schema, table]
  );

  const handleColumnNameChange = useCallback(
    (column: ColumnMetadata, newName: string) => {
      const oldName = column.name;
      if (newName === oldName) return;
      // Status is keyed by column name, so capture it before renaming and
      // clear the old-name key (avoids leaving a stale dirty entry behind).
      const status = editStatus.getColumnStatus(db, { schema, table, column });
      editStatus.removeEditStatus(db, { schema, table, column }, false);
      column.name = newName;
      // Sync PK name if column is in PK
      if (primaryKey && primaryKey.expressions.includes(oldName)) {
        const idx = primaryKey.expressions.indexOf(oldName);
        if (idx >= 0) {
          primaryKey.expressions[idx] = newName;
        }
      }
      // Re-keying "created" onto a name that already belongs to another column
      // would make that existing column read as "created" too (shared name key),
      // so its trash action would splice the real column out. Only preserve
      // "created" when the new name is unique within the table.
      const nameCollides = table.columns.some(
        (c) => c !== column && c.name === newName
      );
      if (status === "created" && !nameCollides) {
        // A freshly-added column being named for the first time must stay
        // "created" (re-keyed to the new name) — otherwise refreshStatus would
        // diff it against a non-existent baseline and downgrade it to
        // "updated", and the trash action would then drop it instead of
        // splicing the unsaved column out.
        editStatus.markEditStatus(db, { schema, table, column }, "created");
      } else {
        refreshStatus();
      }
    },
    [primaryKey, editStatus, db, schema, table, refreshStatus]
  );

  const handleColumnTypeChange = useCallback(
    (column: ColumnMetadata, newType: string) => {
      column.type = newType;
      refreshStatus();
    },
    [refreshStatus]
  );

  const handleDefaultChange = useCallback(
    (
      column: ColumnMetadata,
      value: { hasDefault: boolean; default: string }
    ) => {
      column.hasDefault = value.hasDefault;
      column.default = value.default;
      refreshStatus();
    },
    [refreshStatus]
  );

  const handleNullableChange = useCallback(
    (column: ColumnMetadata, nullable: boolean) => {
      column.nullable = nullable;
      refreshStatus();
    },
    [refreshStatus]
  );

  const handlePrimaryKeyChange = useCallback(
    (column: ColumnMetadata, isPK: boolean) => {
      if (isPK) {
        column.nullable = false;
        upsertColumnPrimaryKey(engine, table, column.name);
      } else {
        removeColumnPrimaryKey(table, column.name);
      }
      refreshStatus();
    },
    [engine, table, refreshStatus]
  );

  const handleCommentChange = useCallback(
    (column: ColumnMetadata, comment: string) => {
      column.comment = comment; // ColumnMetadata.comment
      refreshStatus();
    },
    [refreshStatus]
  );

  const handleDropColumn = useCallback(
    (column: ColumnMetadata) => {
      const status = getStatus(column);
      if (status === "created") {
        const idx = table.columns.indexOf(column);
        if (idx >= 0) table.columns.splice(idx, 1);
        editStatus.removeEditStatus(db, { schema, table, column }, true);
      } else {
        editStatus.markEditStatus(db, { schema, table, column }, "dropped");
      }
    },
    [getStatus, table, editStatus, db, schema]
  );

  const handleRestoreColumn = useCallback(
    (column: ColumnMetadata) => {
      editStatus.removeEditStatus(db, { schema, table, column }, false);
    },
    [editStatus, db, schema, table]
  );

  // Compact rows: trim the shared Table's default vertical padding (py-3 cells,
  // h-10 headers) but keep py-2 so inline editors have breathing room around
  // them. Kept in sync with TableList.
  const cellClass = "py-2";
  const headClass = "h-8 py-1 whitespace-nowrap";

  // Resizable column layout. Name/type/default/comment can be dragged wider;
  // the fixed-content columns (not-null/primary/operations) keep their width.
  // `widths[i]` is positional, so the body cells below must stay in this order.
  const columnDefs = useMemo(
    () => [
      {
        key: "name",
        title: t("schema-editor.column.name"),
        defaultWidth: 160,
        minWidth: 80,
        resizable: true,
        align: "",
      },
      {
        key: "type",
        title: t("schema-editor.column.type"),
        defaultWidth: 180,
        minWidth: 80,
        resizable: true,
        align: "",
      },
      {
        key: "default",
        title: t("schema-editor.column.default"),
        defaultWidth: 200,
        minWidth: 80,
        resizable: true,
        align: "",
      },
      {
        key: "comment",
        title: t("schema-editor.column.comment"),
        defaultWidth: 240,
        minWidth: 80,
        resizable: true,
        align: "",
      },
      {
        key: "not-null",
        title: t("schema-editor.column.not-null"),
        defaultWidth: 72,
        resizable: false,
        align: "text-center",
      },
      {
        key: "primary",
        title: t("schema-editor.column.primary"),
        defaultWidth: 72,
        resizable: false,
        align: "text-center",
      },
      ...(isReadonly
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
    [t, isReadonly]
  );

  const { widths, totalWidth, onResizeStart, setWidths } =
    useColumnWidths(columnDefs);

  // Fit the columns to the available width on first layout so the grid fills
  // the panel instead of overflowing at the sum of the default widths. Runs
  // once; later panel resizes leave any user-dragged widths untouched.
  const containerRef = useRef<HTMLDivElement>(null);
  const didFitRef = useRef(false);
  useLayoutEffect(() => {
    if (didFitRef.current) return;
    const width = containerRef.current?.clientWidth ?? 0;
    if (width <= 0) return;
    didFitRef.current = true;
    setWidths(distributeColumnWidths(columnDefs, width));
  }, [columnDefs, setWidths]);

  return (
    <div ref={containerRef} className="size-full overflow-auto">
      {/* Exact-width (not w-full) so rendered widths match state and dragging
          stays precise; the container scrolls when columns exceed it. */}
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
                className={cn(headClass, col.align)}
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
          {columns.map((column) => {
            const key = getColumnKey(column);
            const status = getStatus(column);
            const isPK = isColumnPrimaryKey(column);
            const disabled =
              isReadonly || disableChangeTable || status === "dropped";

            return (
              <TableRow
                key={key}
                className={cn(
                  // Soft per-status background tint instead of coloring every
                  // input's text — far easier to read at a glance. Line-through
                  // on dropped survives so the row reads as "going away."
                  status === "created" && "bg-success/5",
                  status === "updated" && "bg-warning/5",
                  status === "dropped" && "bg-error/5 line-through",
                  // Hover only on rows with no status tint so we don't fight
                  // the colored backgrounds above.
                  status === "normal" && "hover:bg-control-bg-hover"
                )}
              >
                <TableCell className={cellClass}>
                  <Input
                    // Newly-created, unnamed rows autofocus on mount so the
                    // user can type immediately. `autoFocus` fires only on
                    // the initial mount of a row, so clearing the name on
                    // an existing column won't re-steal focus.
                    autoFocus={status === "created" && column.name === ""}
                    placeholder={t("schema-editor.column.name-placeholder")}
                    value={column.name}
                    disabled={disabled}
                    size="xs"
                    className={INLINE_EDIT_INPUT_CLASS}
                    onChange={(e) =>
                      handleColumnNameChange(column, e.target.value)
                    }
                  />
                </TableCell>
                <TableCell className={cellClass}>
                  <DataTypeCell
                    column={column}
                    engine={engine}
                    readonly={disabled}
                    onUpdateValue={(val) => handleColumnTypeChange(column, val)}
                  />
                </TableCell>
                <TableCell className={cellClass}>
                  <DefaultValueCell
                    column={column}
                    disabled={disabled}
                    onUpdate={(val) => handleDefaultChange(column, val)}
                  />
                </TableCell>
                <TableCell className={cellClass}>
                  <Input
                    value={column.comment}
                    disabled={disabled}
                    size="xs"
                    className={INLINE_EDIT_INPUT_CLASS}
                    onChange={(e) =>
                      handleCommentChange(column, e.target.value)
                    }
                  />
                </TableCell>
                <TableCell className={cn(cellClass, "text-center")}>
                  <Checkbox
                    checked={!column.nullable}
                    disabled={disabled || isPK}
                    onCheckedChange={(checked) =>
                      handleNullableChange(column, !checked)
                    }
                  />
                </TableCell>
                <TableCell className={cn(cellClass, "text-center")}>
                  <Checkbox
                    checked={isPK}
                    disabled={disabled || !allowChangePrimaryKeys}
                    onCheckedChange={(checked) =>
                      handlePrimaryKeyChange(column, checked)
                    }
                  />
                </TableCell>
                {!isReadonly && (
                  <TableCell className={cn(cellClass, "text-right")}>
                    {status === "dropped" ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="size-7 p-0"
                        onClick={() => handleRestoreColumn(column)}
                      >
                        <RotateCcw className="size-3.5" />
                      </Button>
                    ) : (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="size-7 p-0 text-control-light hover:bg-error/10 hover:text-error"
                        disabled={disableChangeTable}
                        onClick={() => handleDropColumn(column)}
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
