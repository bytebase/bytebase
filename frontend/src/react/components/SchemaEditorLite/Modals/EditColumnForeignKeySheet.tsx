import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  removeColumnFromForeignKey,
  upsertColumnFromForeignKey,
} from "@/components/SchemaEditorLite/edit";
import { Button } from "@/react/components/ui/button";
import { Combobox } from "@/react/components/ui/combobox";
import { Input } from "@/react/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseEngine, hasSchemaProperty } from "@/utils";
import { useSchemaEditorContext } from "../context";

interface Props {
  open: boolean;
  onClose: () => void;
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  column: ColumnMetadata;
  foreignKey?: ForeignKeyMetadata;
}

export function EditColumnForeignKeySheet({
  open,
  onClose,
  db,
  database,
  schema,
  table,
  column,
  foreignKey,
}: Props) {
  const { t } = useTranslation();
  const { editStatus } = useSchemaEditorContext();
  const engine = getDatabaseEngine(db);
  const showSchemaSelector = hasSchemaProperty(engine);

  const [fkName, setFkName] = useState("");
  const [refSchemaName, setRefSchemaName] = useState("");
  const [refTableName, setRefTableName] = useState<string | null>(null);
  const [refColumnName, setRefColumnName] = useState<string | null>(null);

  // Initialize from existing FK
  useEffect(() => {
    if (!foreignKey) {
      setFkName(`fk_${table.name}_${column.name}`);
      setRefSchemaName(schema.name);
      return;
    }
    setFkName(foreignKey.name);
    setRefSchemaName(foreignKey.referencedSchema || schema.name);
    setRefTableName(foreignKey.referencedTable || null);

    const position = foreignKey.columns.indexOf(column.name);
    if (position >= 0 && foreignKey.referencedColumns[position]) {
      setRefColumnName(foreignKey.referencedColumns[position]);
    }
  }, [foreignKey, table.name, column.name, schema.name]);

  // Cascading options
  const schemaOptions = useMemo(
    () => database.schemas.map((s) => ({ label: s.name, value: s.name })),
    [database.schemas]
  );

  const refSchema = useMemo(
    () => database.schemas.find((s) => s.name === refSchemaName),
    [database.schemas, refSchemaName]
  );

  const tableOptions = useMemo(
    () =>
      refSchema?.tables.map((t) => ({ label: t.name, value: t.name })) ?? [],
    [refSchema]
  );

  const refTable = useMemo(
    () => refSchema?.tables.find((t) => t.name === refTableName),
    [refSchema, refTableName]
  );

  const columnOptions = useMemo(
    () =>
      refTable?.columns.map((c) => ({ label: c.name, value: c.name })) ?? [],
    [refTable]
  );

  const allowConfirm = fkName.length > 0 && refColumnName !== null;

  const handleSchemaChange = useCallback((val: string) => {
    setRefSchemaName(val);
    setRefTableName(null);
    setRefColumnName(null);
  }, []);

  const handleTableChange = useCallback((val: string | null) => {
    setRefTableName(val);
    setRefColumnName(null);
  }, []);

  const handleConfirm = useCallback(() => {
    if (!allowConfirm || !refColumnName) return;

    if (!foreignKey) {
      // Create new FK
      const fk = {
        name: fkName,
        columns: [column.name],
        referencedSchema: refSchemaName,
        referencedTable: refTableName!,
        referencedColumns: [refColumnName],
      } as ForeignKeyMetadata;
      table.foreignKeys.push(fk);
    } else {
      // Update existing FK
      foreignKey.name = fkName;
      if (
        foreignKey.referencedSchema !== refSchemaName ||
        foreignKey.referencedTable !== refTableName
      ) {
        foreignKey.referencedSchema = refSchemaName;
        foreignKey.referencedTable = refTableName!;
        foreignKey.columns = [];
        foreignKey.referencedColumns = [];
      }
      upsertColumnFromForeignKey(foreignKey, column.name, refColumnName);
    }

    const status = editStatus.getColumnStatus(db, {
      schema,
      table,
      column,
    });
    if (status === "normal") {
      editStatus.markEditStatus(db, { schema, table, column }, "updated");
    }

    onClose();
  }, [
    allowConfirm,
    refColumnName,
    foreignKey,
    fkName,
    refSchemaName,
    refTableName,
    column,
    table,
    editStatus,
    db,
    schema,
    onClose,
  ]);

  const handleDelete = useCallback(() => {
    if (!foreignKey) return;
    removeColumnFromForeignKey(table, foreignKey, column.name);
    // Remove FK entirely if no columns left
    if (foreignKey.columns.length === 0) {
      const idx = table.foreignKeys.indexOf(foreignKey);
      if (idx >= 0) table.foreignKeys.splice(idx, 1);
    }
    const status = editStatus.getColumnStatus(db, {
      schema,
      table,
      column,
    });
    if (status === "normal") {
      editStatus.markEditStatus(db, { schema, table, column }, "updated");
    }
    onClose();
  }, [foreignKey, table, column, editStatus, db, schema, onClose]);

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="wide">
        <SheetHeader>
          <SheetTitle>
            {foreignKey
              ? t("schema-editor.edit-foreign-key")
              : t("schema-editor.create-foreign-key")}
          </SheetTitle>
        </SheetHeader>
        <div className="flex flex-col gap-y-4 py-4">
          <div>
            <label className="mb-1 block text-sm font-medium">
              {t("schema-editor.foreign-key.name")}
            </label>
            <Input value={fkName} onChange={(e) => setFkName(e.target.value)} />
          </div>
          {showSchemaSelector && (
            <div>
              <label className="mb-1 block text-sm font-medium">
                {t("schema-editor.foreign-key.referenced-schema")}
              </label>
              <Combobox
                value={refSchemaName}
                onChange={(val) => handleSchemaChange(val as string)}
                options={schemaOptions}
                portal
              />
            </div>
          )}
          <div>
            <label className="mb-1 block text-sm font-medium">
              {t("schema-editor.foreign-key.referenced-table")}
            </label>
            <Combobox
              value={refTableName ?? ""}
              onChange={(val) => handleTableChange((val as string) || null)}
              options={tableOptions}
              portal
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">
              {t("schema-editor.foreign-key.referenced-column")}
            </label>
            <Combobox
              value={refColumnName ?? ""}
              onChange={(val) => setRefColumnName((val as string) || null)}
              options={columnOptions}
              portal
            />
          </div>
        </div>
        <SheetFooter>
          <div className="flex w-full items-center justify-between">
            <div>
              {foreignKey && (
                <Button variant="destructive" onClick={handleDelete}>
                  {t("common.delete")}
                </Button>
              )}
            </div>
            <div className="flex items-center gap-x-2">
              <Button variant="outline" onClick={onClose}>
                {t("common.cancel")}
              </Button>
              <Button disabled={!allowConfirm} onClick={handleConfirm}>
                {t("common.save")}
              </Button>
            </div>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
