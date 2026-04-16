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
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { ProcedureMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";

interface Props {
  open: boolean;
  onClose: () => void;
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure?: ProcedureMetadata;
}

export function ProcedureNameDialog({
  open,
  onClose,
  db,
  database,
  schema,
  procedure,
}: Props) {
  const { t } = useTranslation();
  const { tabs, editStatus, rebuildTree } = useSchemaEditorContext();

  const [name, setName] = useState(procedure?.name ?? "");
  const isCreateMode = !procedure;

  const isDuplicate =
    name !== procedure?.name && schema.procedures.some((p) => p.name === name);
  const isValid = name.length > 0 && !isDuplicate;

  const handleConfirm = useCallback(() => {
    if (!isValid) return;
    if (isCreateMode) {
      const newProc = create(ProcedureMetadataSchema, {
        name,
        definition: "",
      }) as ProcedureMetadata;
      schema.procedures.push(newProc);
      editStatus.markEditStatus(db, { schema, procedure: newProc }, "created");
      tabs.addTab({
        type: "procedure",
        database: db,
        metadata: { database, schema, procedure: newProc },
      });
      rebuildTree(false);
    } else {
      procedure.name = name;
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
    procedure,
    onClose,
  ]);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogTitle>
          {isCreateMode
            ? t("schema-editor.actions.create-procedure")
            : t("schema-editor.actions.rename")}
        </DialogTitle>
        <div className="mt-4 flex flex-col gap-y-4">
          <Input
            value={name}
            placeholder="Procedure name"
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
