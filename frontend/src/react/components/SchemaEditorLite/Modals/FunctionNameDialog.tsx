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
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { FunctionMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";

interface Props {
  open: boolean;
  onClose: () => void;
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  func?: FunctionMetadata;
}

export function FunctionNameDialog({
  open,
  onClose,
  db,
  database,
  schema,
  func,
}: Props) {
  const { t } = useTranslation();
  const { tabs, editStatus, rebuildTree } = useSchemaEditorContext();

  const [name, setName] = useState(func?.name ?? "");
  const isCreateMode = !func;

  const isDuplicate =
    name !== func?.name && schema.functions.some((f) => f.name === name);
  const isValid = name.length > 0 && !isDuplicate;

  const handleConfirm = useCallback(() => {
    if (!isValid) return;
    if (isCreateMode) {
      const newFunc = create(FunctionMetadataSchema, {
        name,
        definition: "",
      }) as FunctionMetadata;
      schema.functions.push(newFunc);
      editStatus.markEditStatus(db, { schema, function: newFunc }, "created");
      tabs.addTab({
        type: "function",
        database: db,
        metadata: { database, schema, function: newFunc },
      });
      rebuildTree(false);
    } else {
      func.name = name;
    }
    onClose();
  }, [
    isValid,
    isCreateMode,
    name,
    schema,
    editStatus,
    db,
    database,
    tabs,
    rebuildTree,
    func,
    onClose,
  ]);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogTitle>
          {isCreateMode
            ? t("schema-editor.actions.create-function")
            : t("schema-editor.actions.rename")}
        </DialogTitle>
        <div className="mt-4 flex flex-col gap-y-4">
          <Input
            value={name}
            placeholder="Function name"
            autoFocus
            onChange={(e) => setName(e.target.value)}
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
