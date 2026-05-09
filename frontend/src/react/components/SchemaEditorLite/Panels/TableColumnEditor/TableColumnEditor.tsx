import { RotateCcw, Trash2 } from "lucide-react";
import { useCallback, useMemo } from "react";
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
import { markUUID } from "../common";
import { DataTypeCell, DataTypeSuggestionsDatalist } from "./DataTypeCell";
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
  onMarkTableStatus: (status: "updated") => void;
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
  onMarkTableStatus,
}: Props) {
  const { t } = useTranslation();
  const { editStatus } = useSchemaEditorContext();

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
      column.name = newName;
      // Sync PK name if column is in PK
      if (primaryKey && primaryKey.expressions.includes(oldName)) {
        const idx = primaryKey.expressions.indexOf(oldName);
        if (idx >= 0) {
          primaryKey.expressions[idx] = newName;
        }
      }
      const status = getStatus(column);
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, table, column }, "updated");
      }
      onMarkTableStatus("updated");
    },
    [primaryKey, getStatus, editStatus, db, schema, table, onMarkTableStatus]
  );

  const handleColumnTypeChange = useCallback(
    (column: ColumnMetadata, newType: string) => {
      column.type = newType;
      const status = getStatus(column);
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, table, column }, "updated");
      }
      onMarkTableStatus("updated");
    },
    [getStatus, editStatus, db, schema, table, onMarkTableStatus]
  );

  const handleDefaultChange = useCallback(
    (
      column: ColumnMetadata,
      value: { hasDefault: boolean; default: string }
    ) => {
      column.hasDefault = value.hasDefault;
      column.default = value.default;
      const status = getStatus(column);
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, table, column }, "updated");
      }
      onMarkTableStatus("updated");
    },
    [getStatus, editStatus, db, schema, table, onMarkTableStatus]
  );

  const handleNullableChange = useCallback(
    (column: ColumnMetadata, nullable: boolean) => {
      column.nullable = nullable;
      const status = getStatus(column);
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, table, column }, "updated");
      }
      onMarkTableStatus("updated");
    },
    [getStatus, editStatus, db, schema, table, onMarkTableStatus]
  );

  const handlePrimaryKeyChange = useCallback(
    (column: ColumnMetadata, isPK: boolean) => {
      if (isPK) {
        column.nullable = false;
        upsertColumnPrimaryKey(engine, table, column.name);
      } else {
        removeColumnPrimaryKey(table, column.name);
      }
      const status = getStatus(column);
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, table, column }, "updated");
      }
      onMarkTableStatus("updated");
    },
    [engine, table, getStatus, editStatus, db, schema, onMarkTableStatus]
  );

  const handleCommentChange = useCallback(
    (column: ColumnMetadata, comment: string) => {
      column.comment = comment; // ColumnMetadata.comment
      const status = getStatus(column);
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, table, column }, "updated");
      }
      onMarkTableStatus("updated");
    },
    [getStatus, editStatus, db, schema, table, onMarkTableStatus]
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
      onMarkTableStatus("updated");
    },
    [getStatus, table, editStatus, db, schema, onMarkTableStatus]
  );

  const handleRestoreColumn = useCallback(
    (column: ColumnMetadata) => {
      editStatus.removeEditStatus(db, { schema, table, column }, false);
    },
    [editStatus, db, schema, table]
  );

  const datalistId = `schema-editor-types-${engine}`;
  // Compact override for table cells: reduce default px-4 py-3 padding so each
  // row hugs the inline editor heights instead of doubling them.
  const cellClass = "px-2 py-1";
  const headClass = "h-8 px-2 py-1 whitespace-nowrap";

  return (
    <div className="size-full overflow-auto">
      <DataTypeSuggestionsDatalist id={datalistId} engine={engine} />
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className={cn(headClass, "w-[160px]")}>
              {t("schema-editor.column.name")}
            </TableHead>
            <TableHead className={cn(headClass, "w-[180px]")}>
              {t("schema-editor.column.type")}
            </TableHead>
            <TableHead className={cn(headClass, "w-[200px]")}>
              {t("schema-editor.column.default")}
            </TableHead>
            <TableHead className={cn(headClass)}>
              {t("schema-editor.column.comment")}
            </TableHead>
            <TableHead className={cn(headClass, "w-[72px] text-center")}>
              {t("schema-editor.column.not-null")}
            </TableHead>
            <TableHead className={cn(headClass, "w-[72px] text-center")}>
              {t("schema-editor.column.primary")}
            </TableHead>
            {!isReadonly && (
              <TableHead className={cn(headClass, "w-[60px] text-right")}>
                {t("schema-editor.column.operations")}
              </TableHead>
            )}
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
                  status === "created" && "text-success",
                  status === "updated" && "text-warning",
                  status === "dropped" && "text-error line-through"
                )}
              >
                <TableCell className={cellClass}>
                  <Input
                    value={column.name}
                    disabled={disabled}
                    size="xs"
                    className="border-none bg-transparent shadow-none focus-visible:ring-1"
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
                    datalistId={datalistId}
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
                    className="border-none bg-transparent shadow-none focus-visible:ring-1"
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
                        className="size-7 p-0 text-error hover:text-error"
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
