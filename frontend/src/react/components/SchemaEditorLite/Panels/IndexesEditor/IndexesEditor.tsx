import { create } from "@bufbuild/protobuf";
import { Plus, Trash2 } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Combobox } from "@/react/components/ui/combobox";
import { Input } from "@/react/components/ui/input";
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
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { IndexMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../../context";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  readonly: boolean;
}

export function IndexesEditor({
  db,
  database: _database,
  schema,
  table,
  readonly: isReadonly,
}: Props) {
  const { t } = useTranslation();
  const { editStatus } = useSchemaEditorContext();

  const columnOptions = table.columns.map((col) => ({
    label: col.name,
    value: col.name,
  }));

  const handleAddIndex = useCallback(() => {
    const index = create(IndexMetadataSchema, {
      name: `idx_${table.name}_${Date.now()}`,
      expressions: [],
      primary: false,
      unique: false,
      comment: "",
    });
    table.indexes.push(index);
    editStatus.markEditStatus(db, { schema, table }, "updated");
  }, [table, editStatus, db, schema]);

  const handleDropIndex = useCallback(
    (index: IndexMetadata) => {
      const idx = table.indexes.indexOf(index);
      if (idx >= 0) table.indexes.splice(idx, 1);
      editStatus.markEditStatus(db, { schema, table }, "updated");
    },
    [table, editStatus, db, schema]
  );

  const handleIndexChange = useCallback(() => {
    editStatus.markEditStatus(db, { schema, table }, "updated");
  }, [editStatus, db, schema, table]);

  return (
    <div className="flex size-full flex-col gap-y-2 overflow-auto">
      {!isReadonly && (
        <div>
          <Button variant="outline" size="sm" onClick={handleAddIndex}>
            <Plus className="mr-1 size-4" />
            {t("schema-editor.actions.add-index")}
          </Button>
        </div>
      )}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[180px]">
              {t("schema-editor.column.name")}
            </TableHead>
            <TableHead className="w-[200px]">
              {t("schema-editor.index.columns")}
            </TableHead>
            <TableHead className="w-[100px]">
              {t("schema-editor.column.comment")}
            </TableHead>
            <TableHead className="w-16 text-center">
              {t("schema-editor.index.unique")}
            </TableHead>
            <TableHead className="w-16 text-center">
              {t("schema-editor.index.primary")}
            </TableHead>
            {!isReadonly && (
              <TableHead className="w-16 text-right">
                {t("schema-editor.column.operations")}
              </TableHead>
            )}
          </TableRow>
        </TableHeader>
        <TableBody>
          {table.indexes.map((index, i) => (
            <TableRow key={`${index.name}-${i}`}>
              <TableCell>
                <Input
                  value={index.name}
                  disabled={isReadonly || index.primary}
                  size="sm"
                  className="border-none bg-transparent shadow-none focus-visible:ring-1"
                  onChange={(e) => {
                    index.name = e.target.value;
                    handleIndexChange();
                  }}
                />
              </TableCell>
              <TableCell>
                <Combobox
                  multiple
                  value={[...index.expressions]}
                  onChange={(val) => {
                    index.expressions = val as string[];
                    handleIndexChange();
                  }}
                  options={columnOptions}
                  disabled={isReadonly}
                  className="h-7"
                />
              </TableCell>
              <TableCell>
                <Input
                  value={index.comment}
                  disabled={isReadonly}
                  size="sm"
                  className="border-none bg-transparent shadow-none focus-visible:ring-1"
                  onChange={(e) => {
                    index.comment = e.target.value;
                    handleIndexChange();
                  }}
                />
              </TableCell>
              <TableCell className="text-center">
                <input
                  type="checkbox"
                  checked={index.unique}
                  disabled={isReadonly || index.primary}
                  onChange={(e) => {
                    index.unique = e.target.checked;
                    handleIndexChange();
                  }}
                />
              </TableCell>
              <TableCell className="text-center">
                <input type="checkbox" checked={index.primary} disabled />
              </TableCell>
              {!isReadonly && (
                <TableCell className="text-right">
                  {!index.primary && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="size-7 p-0 text-error hover:text-error"
                      onClick={() => handleDropIndex(index)}
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  )}
                </TableCell>
              )}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
