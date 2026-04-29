import { Diamond, Key, Pencil } from "lucide-react";
import { useCallback, useMemo } from "react";
import { hashCode } from "@/bbkit/BBUtil";
import { cn } from "@/react/lib/utils";
import type {
  ColumnMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useGeometry, useSchemaDiagramContext } from "../common/context";
import { FocusButton } from "../common/FocusButton";
import { isIndex, isPrimaryKey } from "../common/schema";
import { isFocusedForeignTable } from "./libs/isFocusedFKTable";

interface TableNodeProps {
  schema: SchemaMetadata;
  table: TableMetadata;
}

const COLOR_LIST = [
  "#64748B",
  "#EF4444",
  "#F97316",
  "#EAB308",
  "#84CC16",
  "#22C55E",
  "#10B981",
  "#06B6D4",
  "#0EA5E9",
  "#3B82F6",
  "#6366F1",
  "#8B5CF6",
  "#A855F7",
  "#D946EF",
  "#EC4899",
  "#F43F5E",
];

/**
 * React port of `ER/TableNode.vue`. One table card with title bar, FK
 * focus button, optional edit pencil, and a column list with PK / index
 * glyphs and edit-on-click affordances when `editable` is on.
 */
export function TableNode({ schema, table }: TableNodeProps) {
  const ctx = useSchemaDiagramContext();
  const {
    dummy,
    editable,
    focusedTables,
    foreignKeys,
    idOfTable,
    rectOfTable,
    schemaStatus,
    tableStatus,
    columnStatus,
    panning,
    events,
  } = ctx;

  const tableColor = useMemo(() => {
    const index = (hashCode(table.name) & 0xfffffff) % COLOR_LIST.length;
    return COLOR_LIST[index];
  }, [table.name]);

  const isTableDropped = useMemo(
    () =>
      schemaStatus(schema) === "dropped" || tableStatus(table) === "dropped",
    [schema, table, schemaStatus, tableStatus]
  );

  const isTableCreated = useMemo(
    () =>
      schemaStatus(schema) === "created" || tableStatus(table) === "created",
    [schema, table, schemaStatus, tableStatus]
  );

  const isTableChanged = useMemo(
    () => tableStatus(table) === "changed",
    [table, tableStatus]
  );

  const tableStatusText = isTableCreated
    ? "Created"
    : isTableDropped
      ? "Dropped"
      : isTableChanged
        ? "Changed"
        : "";

  const tableClasses = useMemo(() => {
    if (focusedTables.size === 0) return "";
    if (focusedTables.has(table)) return "opacity-100";
    if (isFocusedForeignTable(table, focusedTables, foreignKeys)) {
      return "opacity-100";
    }
    return "opacity-20 hover:opacity-100";
  }, [table, focusedTables, foreignKeys]);

  const columnClasses = useCallback(
    (column: ColumnMetadata) => {
      const classes: string[] = [];
      if (editable) classes.push("cursor-pointer");
      const status = columnStatus(column);
      if (status === "changed") classes.push("text-yellow-700 bg-yellow-50");
      else if (status === "created") classes.push("text-green-700 bg-green-50");
      else if (status === "dropped")
        classes.push("text-red-700 bg-red-50 line-through");
      return classes.join(" ");
    },
    [editable, columnStatus]
  );

  const handleClickColumn = useCallback(
    (column: ColumnMetadata, target: "name" | "type") => {
      if (!editable) return;
      if (panning) return;
      void events.emit("edit-column", { schema, table, column, target });
    },
    [editable, panning, events, schema, table]
  );

  const id = idOfTable(table);
  const rect = rectOfTable(table);

  // Register the rect so the canvas's fit-view bbox includes this table.
  useGeometry(rect);

  return (
    <div
      className={cn(
        "absolute overflow-hidden rounded-md shadow-lg border-b border-control-border bg-background w-[16rem] divide-y divide-control-border z-10 transition-opacity",
        tableClasses
      )}
      data-bb-node-type="table"
      data-bb-node-id={dummy ? `dummy-${id}` : id}
      data-bb-status={tableStatus(table)}
      style={{ left: `${rect.x}px`, top: `${rect.y}px` }}
    >
      <h3
        className={cn(
          "group font-medium leading-6 text-white px-2 py-2 rounded-t-md gap-x-1 relative text-center whitespace-pre-wrap break-words"
        )}
        style={{ backgroundColor: tableColor }}
      >
        <FocusButton
          table={table}
          setCenter={false}
          className="invisible group-hover:visible !absolute top-[50%] -mt-[9px] left-1 text-control group-hover:!bg-white/70 group-hover:!text-control"
          focusedClass="!text-white"
        />

        {schema.name !== "" && (
          <>
            <span className={cn(isTableDropped && "line-through")}>
              {schema.name}
            </span>
            <span>.</span>
          </>
        )}
        <span className={cn(isTableDropped && "line-through")}>
          {table.name}
        </span>
        {tableStatusText && (
          <span className="ml-1 text-sm">({tableStatusText})</span>
        )}

        {editable && (
          <button
            type="button"
            className="invisible group-hover:visible absolute top-[50%] -mt-[9px] right-1 text-control bg-white/70 hover:bg-control-bg p-0.5 rounded-sm"
            onClick={() => events.emit("edit-table", { schema, table })}
          >
            <Pencil className="size-4" />
          </button>
        )}
      </h3>

      <table className="w-full text-sm table-fixed">
        <tbody>
          {table.columns.map((column, i) => (
            <tr
              key={i}
              data-bb-column-name={dummy ? `dummy-${column.name}` : column.name}
              data-bb-status={columnStatus(column)}
              className={columnClasses(column)}
            >
              <td className="w-5 py-1.5 relative">
                {isPrimaryKey(table, column) ? (
                  <Key className="size-3 mx-auto text-warning" />
                ) : isIndex(table, column) ? (
                  <Diamond className="size-3 mx-auto text-control-light" />
                ) : null}
              </td>
              <td className="w-auto text-xs py-1.5">
                <div
                  className={cn(
                    "whitespace-pre-wrap break-words pr-1.5",
                    editable && "hover:!text-accent"
                  )}
                  onClick={() => handleClickColumn(column, "name")}
                >
                  {column.name}
                  {columnStatus(column) !== "normal" && (
                    <span className="inline-block rounded-full ml-0.5 h-1.5 w-1.5 bg-accent opacity-75 -translate-y-px" />
                  )}
                </div>
              </td>
              <td className="w-24 text-xs text-control-light py-1.5 text-right">
                <div
                  className={cn(
                    "truncate pr-1.5",
                    editable && "hover:!text-accent"
                  )}
                  onClick={() => handleClickColumn(column, "type")}
                >
                  {column.type}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
