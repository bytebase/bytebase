import { create } from "@bufbuild/protobuf";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { revisionServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { pushNotification, useSheetV1Store } from "@/store";
import {
  BatchCreateRevisionsRequestSchema,
  CreateRevisionRequestSchema,
  type Revision,
  Revision_Type,
} from "@/types/proto-es/v1/revision_service_pb";

export function CreateRevisionDialog({
  databaseName,
  existingVersions,
  open,
  projectName,
  onCreated,
  onOpenChange,
}: {
  databaseName: string;
  existingVersions: string[];
  open: boolean;
  projectName: string;
  onCreated: (revisions: Revision[]) => void;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();
  const sheetStore = useSheetV1Store();
  const [version, setVersion] = useState("");
  const [statement, setStatement] = useState("");
  const [type, setType] = useState(Revision_Type.VERSIONED);
  const [isCreating, setIsCreating] = useState(false);
  const normalizedVersion = version.trim();
  const isDuplicateVersion = existingVersions.includes(normalizedVersion);
  const isValidVersion = /^(\d+)(\.(\d+))*$/.test(normalizedVersion);

  const reset = () => {
    setVersion("");
    setStatement("");
    setType(Revision_Type.VERSIONED);
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      reset();
    }
    onOpenChange(nextOpen);
  };

  const canCreate =
    normalizedVersion.length > 0 &&
    statement.trim().length > 0 &&
    isValidVersion &&
    !isDuplicateVersion;

  const handleCreate = async () => {
    if (!canCreate || isCreating) {
      return;
    }

    setIsCreating(true);
    try {
      const sheet = await sheetStore.createSheet(projectName, {
        content: new TextEncoder().encode(statement),
      });
      const response = await revisionServiceClientConnect.batchCreateRevisions(
        create(BatchCreateRevisionsRequestSchema, {
          parent: databaseName,
          requests: [
            create(CreateRevisionRequestSchema, {
              parent: databaseName,
              revision: {
                sheet: sheet.name,
                type,
                version: normalizedVersion,
              },
            }),
          ],
        })
      );
      onCreated(response.revisions);
      handleOpenChange(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-3xl p-6">
        <DialogTitle>{t("database.revision.import-revision")}</DialogTitle>
        <div className="mt-4 space-y-4">
          <div className="space-y-2">
            <label
              className="block text-sm font-medium text-control"
              htmlFor="revision-version"
            >
              {t("common.version")}
            </label>
            <input
              id="revision-version"
              className="w-full rounded-xs border border-control-border px-3 py-2 text-sm text-main"
              name="version"
              value={version}
              onChange={(event) => setVersion(event.target.value)}
            />
            {normalizedVersion.length > 0 && !isValidVersion && (
              <p className="text-sm text-error">
                {t("database.revision.invalid-version-format")}
              </p>
            )}
            {isDuplicateVersion && (
              <p className="text-sm text-error">
                {t("database.revision.version-already-exists")}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <label
              className="block text-sm font-medium text-control"
              htmlFor="revision-type"
            >
              {t("common.type")}
            </label>
            <select
              id="revision-type"
              className="w-full rounded-xs border border-control-border px-3 py-2 text-sm text-main"
              value={String(type)}
              onChange={(event) =>
                setType(Number(event.target.value) as Revision_Type)
              }
            >
              <option value={String(Revision_Type.VERSIONED)}>
                {t("database.revision.type-versioned")}
              </option>
              <option value={String(Revision_Type.DECLARATIVE)}>
                {t("database.revision.type-declarative")}
              </option>
            </select>
          </div>
          <div className="space-y-2">
            <label
              className="block text-sm font-medium text-control"
              htmlFor="revision-statement"
            >
              {t("common.statement")}
            </label>
            <textarea
              id="revision-statement"
              className="min-h-48 w-full rounded-xs border border-control-border px-3 py-2 text-sm text-main"
              name="statement"
              value={statement}
              onChange={(event) => setStatement(event.target.value)}
            />
          </div>
          <div className="flex items-center justify-end gap-x-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => handleOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              disabled={!canCreate || isCreating}
              type="button"
              onClick={() => void handleCreate()}
            >
              {isCreating ? t("common.loading") : t("common.create")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
