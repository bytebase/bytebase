import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { extractInstanceResourceName } from "@/utils";

interface InstanceDeleteDialogProps {
  open: boolean;
  instance: Instance;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}

export function InstanceDeleteDialog({
  open,
  instance,
  onOpenChange,
  onDeleted,
}: InstanceDeleteDialogProps) {
  const { t } = useTranslation();
  const [confirmText, setConfirmText] = useState("");
  const [deleting, setDeleting] = useState(false);
  const instanceId = useMemo(
    () => extractInstanceResourceName(instance.name),
    [instance.name]
  );
  const isArchived = instance.state === State.DELETED;
  const canConfirm = confirmText === instanceId && !deleting;
  const description = isArchived
    ? t("instance.delete-confirmation", {
        id: instanceId,
        name: instance.title,
      })
    : t("instance.delete-active-confirmation", {
        id: instanceId,
        name: instance.title,
      });

  useEffect(() => {
    if (open) {
      setConfirmText("");
      setDeleting(false);
    }
  }, [open, instanceId]);

  const handleDelete = async () => {
    if (!canConfirm) return;

    setDeleting(true);
    try {
      const store = useAppStore.getState();
      if (!isArchived) {
        await store.archiveInstance(instance, true);
      }
      await store.deleteInstance(instance.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
      onOpenChange(false);
      onDeleted?.();
    } catch (error: unknown) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.delete"),
        description: (error as { message?: string }).message,
      });
    } finally {
      setDeleting(false);
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className="max-w-lg">
        <AlertDialogTitle>
          {t("common.delete-resource", { type: instance.title })}
        </AlertDialogTitle>
        <AlertDialogDescription className="mt-2">
          {description}
        </AlertDialogDescription>
        <div className="mt-4">
          <Input
            value={confirmText}
            onChange={(event) => setConfirmText(event.target.value)}
            placeholder={instanceId}
            autoFocus
            autoComplete="off"
          />
        </div>
        <div className="mt-6 flex justify-end gap-x-2">
          <Button
            appearance="outline"
            disabled={deleting}
            onClick={() => onOpenChange(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button
            variant="destructive"
            disabled={!canConfirm}
            onClick={() => void handleDelete()}
          >
            {t("common.delete")}
          </Button>
        </div>
      </AlertDialogContent>
    </AlertDialog>
  );
}
