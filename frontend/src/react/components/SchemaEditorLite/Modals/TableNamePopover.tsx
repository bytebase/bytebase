import { create } from "@bufbuild/protobuf";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { Popover, PopoverContent } from "@/react/components/ui/popover";
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
  // Screen-space coordinates of the click that triggered the popover. We
  // anchor to a virtual 0×0 rect at this point so we don't depend on a DOM
  // element that may have unmounted (e.g. the closed context menu item).
  anchorPoint: { x: number; y: number };
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
}

const TABLE_NAME_REGEX = /^\S[\S ]*\S?$/;

export function TableNamePopover({
  open,
  onClose,
  anchorPoint,
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
    schema.tables.some((tt) => tt.name === tableName);

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

  // Virtual anchor at the click point. Base UI's Positioner accepts an
  // object exposing getBoundingClientRect(); a 0×0 rect at (x, y) lets the
  // popover float right next to where the user clicked.
  const anchor = {
    getBoundingClientRect: () =>
      ({
        width: 0,
        height: 0,
        x: anchorPoint.x,
        y: anchorPoint.y,
        top: anchorPoint.y,
        left: anchorPoint.x,
        right: anchorPoint.x,
        bottom: anchorPoint.y,
        toJSON() {
          return this;
        },
      }) as DOMRect,
  };

  return (
    <Popover open={open} onOpenChange={(next) => !next && onClose()}>
      <PopoverContent
        anchor={anchor}
        side="bottom"
        align="start"
        className="w-72 p-3"
      >
        <div className="flex flex-col gap-y-3">
          <div className="text-sm font-medium text-control">
            {isCreateMode
              ? t("schema-editor.actions.create-table")
              : t("schema-editor.actions.rename-table")}
          </div>
          <Input
            value={tableName}
            placeholder={t("schema-editor.table.name-placeholder")}
            autoFocus
            onChange={(e) => setTableName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleConfirm();
              if (e.key === "Escape") onClose();
            }}
          />
          {isDuplicate && (
            <p className="text-xs text-error">
              {t("schema-editor.table.duplicate-name")}
            </p>
          )}
          <div className="flex items-center justify-end gap-x-2">
            <Button variant="outline" size="sm" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button size="sm" disabled={!isValid} onClick={handleConfirm}>
              {isCreateMode ? t("common.create") : t("common.save")}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
