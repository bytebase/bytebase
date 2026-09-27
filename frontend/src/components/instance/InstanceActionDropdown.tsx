import { EllipsisVertical } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/app/router/handles";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { InstanceDeleteDialog } from "./InstanceDeleteDialog";
import { hasInstancePermission } from "./permission";

interface InstanceActionDropdownProps {
  instance: Instance;
  project?: Project;
  onArchived?: () => void;
  onDeleted?: () => void;
}

export function InstanceActionDropdown({
  instance,
  project,
  onArchived,
  onDeleted,
}: InstanceActionDropdownProps) {
  const { t } = useTranslation();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const canArchive = hasInstancePermission(project, "bb.instances.delete");
  const canRestore = hasInstancePermission(project, "bb.instances.undelete");

  const handleArchive = useCallback(async () => {
    const msg = t("instance.archive-instance-instance-name", {
      0: instance.title,
    });
    const confirmed = window.confirm(
      project
        ? `${msg}\n\n${t("instance.archived-instances-will-not-be-displayed")}`
        : `${msg}\n\n${t("instance.archived-instances-will-not-be-displayed")}\n\n${t("instance.force-archive-description")}`
    );
    if (!confirmed) return;

    await useAppStore.getState().archiveInstance(instance, !project);
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("instance.successfully-archived-instance", {
        0: instance.title,
      }),
    });
    if (onArchived) {
      onArchived();
    } else {
      router.replace({ name: INSTANCE_ROUTE_DASHBOARD });
    }
  }, [instance, onArchived, project, t]);

  const handleRestore = useCallback(async () => {
    if (
      !window.confirm(
        t("instance.restore-instance-instance-name-to-normal-state", {
          0: instance.title,
        })
      )
    )
      return;

    await useAppStore.getState().restoreInstance(instance);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-restored-instance", {
        0: instance.title,
      }),
    });
  }, [instance, t]);

  const showArchive = instance.state === State.ACTIVE && canArchive;
  const showRestore = instance.state === State.DELETED && canRestore;
  const showDelete = canArchive;

  if (!showArchive && !showRestore && !showDelete) return null;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger className="inline-flex items-center justify-center size-8 rounded-xs text-control hover:bg-control-bg cursor-pointer outline-hidden focus-visible:ring-2 focus-visible:ring-accent">
          <EllipsisVertical className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          {showArchive && (
            <DropdownMenuItem onClick={handleArchive}>
              {t("common.archive")}
            </DropdownMenuItem>
          )}
          {showRestore && (
            <DropdownMenuItem onClick={handleRestore}>
              {t("common.restore")}
            </DropdownMenuItem>
          )}
          {showDelete && (
            <DropdownMenuItem
              className="text-error"
              onClick={() => setShowDeleteConfirm(true)}
            >
              {t("common.delete")}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
      <InstanceDeleteDialog
        open={showDeleteConfirm}
        instance={instance}
        forceArchive={!project}
        onOpenChange={setShowDeleteConfirm}
        onDeleted={() => {
          if (onDeleted) {
            onDeleted();
          } else {
            router.replace({ name: INSTANCE_ROUTE_DASHBOARD });
          }
        }}
      />
    </>
  );
}
