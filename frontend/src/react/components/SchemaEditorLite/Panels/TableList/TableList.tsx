import { RotateCcw, Trash2 } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { cn } from "@/react/lib/utils";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../../context";

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
  schema,
  tables,
  searchPattern,
  onEditTable,
}: Props) {
  const { t } = useTranslation();
  const { readonly, editStatus } = useSchemaEditorContext();

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
    },
    [getStatus, schema, editStatus, db]
  );

  const handleRestore = useCallback(
    (table: TableMetadata) => {
      editStatus.removeEditStatus(db, { schema, table }, false);
    },
    [editStatus, db, schema]
  );

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
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[40%]">
            {t("schema-editor.column.name")}
          </TableHead>
          <TableHead>{t("schema-editor.column.comment")}</TableHead>
          {!readonly && (
            <TableHead className="w-20 text-right">
              {t("schema-editor.column.operations")}
            </TableHead>
          )}
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
              <TableCell className="font-medium">{table.name}</TableCell>
              <TableCell className="text-control-light">
                {table.comment || "—"}
              </TableCell>
              {!readonly && (
                <TableCell className="text-right">
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
  );
}
