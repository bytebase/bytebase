import { create } from "@bufbuild/protobuf";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  ColumnMetadataSchema,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseEngine } from "@/utils";
import { useSchemaEditorContext } from "../context";
import { upsertColumnPrimaryKey } from "../core/edit";
import { markUUID } from "../Panels/common";

interface Props {
  open: boolean;
  onClose: () => void;
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
}

const TABLE_NAME_REGEX = /^\S[\S ]*\S?$/;

export function TableNameDialog({
  open,
  onClose,
  db,
  database,
  schema,
  table,
}: Props) {
  const { t } = useTranslation();
  const { tabs, editStatus, rebuildTree, rebuildEditStatus, scrollStatus } =
    useSchemaEditorContext();
  const engine = getDatabaseEngine(db);

  const [tableName, setTableName] = useState(table?.name ?? "");
  const isCreateMode = !table;

  const isDuplicate =
    tableName !== table?.name &&
    schema.tables.some((t) => t.name === tableName);

  const isValid =
    tableName.length > 0 && TABLE_NAME_REGEX.test(tableName) && !isDuplicate;

  const handleConfirm = useCallback(() => {
    if (!isValid) return;

    if (isCreateMode) {
      const newTable = create(TableMetadataSchema, {
        name: tableName,
        columns: [],
        indexes: [],
        foreignKeys: [],
        partitions: [],
        comment: "",
      });
      schema.tables.push(newTable);
      editStatus.markEditStatus(db, { schema, table: newTable }, "created");

      const defaultType = engine === Engine.POSTGRES ? "integer" : "int";
      const idColumn = create(ColumnMetadataSchema, {
        name: "id",
        type: defaultType,
        nullable: false,
        hasDefault: false,
        default: "",
        comment: "",
      });
      markUUID(idColumn);
      newTable.columns.push(idColumn);
      editStatus.markEditStatus(
        db,
        { schema, table: newTable, column: idColumn },
        "created"
      );
      upsertColumnPrimaryKey(engine, newTable, "id");

      tabs.addTab({
        type: "table",
        database: db,
        metadata: { database, schema, table: newTable },
      });
      scrollStatus.queuePendingScrollToTable({
        db,
        metadata: { database, schema, table: newTable },
      });
      rebuildTree(false);
    } else {
      table.name = tableName;
      rebuildEditStatus(["tree"]);
    }

    onClose();
  }, [
    isValid,
    isCreateMode,
    tableName,
    schema,
    editStatus,
    db,
    database,
    engine,
    tabs,
    scrollStatus,
    rebuildTree,
    rebuildEditStatus,
    table,
    onClose,
  ]);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogTitle>
          {isCreateMode
            ? t("schema-editor.actions.create-table")
            : t("schema-editor.actions.rename-table")}
        </DialogTitle>
        <div className="mt-4 flex flex-col gap-y-4">
          <Input
            value={tableName}
            placeholder={t("schema-editor.table.name-placeholder")}
            autoFocus
            onChange={(e) => setTableName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleConfirm();
            }}
          />
          {isDuplicate && (
            <p className="text-xs text-error">
              {t("schema-editor.table.duplicate-name")}
            </p>
          )}
          <div className="flex items-center justify-end gap-x-2">
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button disabled={!isValid} onClick={handleConfirm}>
              {isCreateMode ? t("common.create") : t("common.save")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
