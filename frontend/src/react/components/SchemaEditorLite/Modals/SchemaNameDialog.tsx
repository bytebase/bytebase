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
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { SchemaMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";

interface Props {
  open: boolean;
  onClose: () => void;
  db: Database;
  database: DatabaseMetadata;
  schema?: SchemaMetadata;
}

export function SchemaNameDialog({
  open,
  onClose,
  db,
  database,
  schema,
}: Props) {
  const { t } = useTranslation();
  const { editStatus, rebuildTree } = useSchemaEditorContext();

  const [schemaName, setSchemaName] = useState(schema?.name ?? "");
  const isCreateMode = !schema;

  const isDuplicate =
    schemaName !== schema?.name &&
    database.schemas.some((s) => s.name === schemaName);

  const isValid = schemaName.length > 0 && !isDuplicate;

  const handleConfirm = useCallback(() => {
    if (!isValid) return;

    if (isCreateMode) {
      const newSchema = create(SchemaMetadataSchema, {
        name: schemaName,
        tables: [],
        views: [],
        procedures: [],
        functions: [],
      });
      database.schemas.push(newSchema);
      editStatus.markEditStatus(db, { schema: newSchema }, "created");
      rebuildTree(false);
    }

    onClose();
  }, [
    isValid,
    isCreateMode,
    schemaName,
    database,
    editStatus,
    db,
    rebuildTree,
    onClose,
  ]);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogTitle>{t("schema-editor.actions.create-schema")}</DialogTitle>
        <div className="mt-4 flex flex-col gap-y-4">
          <Input
            value={schemaName}
            placeholder={t("schema-editor.schema.name-placeholder")}
            autoFocus
            onChange={(e) => setSchemaName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleConfirm();
            }}
          />
          {isDuplicate && (
            <p className="text-xs text-error">
              {t("schema-editor.schema.duplicate-name")}
            </p>
          )}
          <div className="flex items-center justify-end gap-x-2">
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button disabled={!isValid} onClick={handleConfirm}>
              {t("common.create")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
