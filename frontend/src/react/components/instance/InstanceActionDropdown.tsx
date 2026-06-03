import { EllipsisVertical } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { router } from "@/react/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

interface InstanceActionDropdownProps {
  instance: Instance;
  onDeleted?: () => void;
}

export function InstanceActionDropdown({
  instance,
  onDeleted,
}: InstanceActionDropdownProps) {
  const { t } = useTranslation();

  const canArchive = hasWorkspacePermissionV2("bb.instances.delete");
  const canRestore = hasWorkspacePermissionV2("bb.instances.undelete");

  const handleArchive = useCallback(async () => {
    const msg = t("instance.archive-instance-instance-name", {
      0: instance.title,
    });
    const forceArchive = window.confirm(
      `${msg}\n\n${t("instance.archived-instances-will-not-be-displayed")}\n\n${t("instance.force-archive-description")}`
    );
    if (!forceArchive) return;

    await useAppStore.getState().archiveInstance(instance, true);
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("instance.successfully-archived-instance", {
        0: instance.title,
      }),
    });
    router.replace({ name: INSTANCE_ROUTE_DASHBOARD });
  }, [instance, t]);

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

  const handleDelete = useCallback(async () => {
    if (
      !window.confirm(
        `${t("common.delete-resource", { type: instance.title })}\n\n${t("common.cannot-undo-this-action")}`
      )
    )
      return;

    await useAppStore.getState().deleteInstance(instance.name);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
    onDeleted?.();
    router.replace({
      name: INSTANCE_ROUTE_DASHBOARD,
      query: { q: "state:DELETED" },
    });
  }, [instance, t, onDeleted]);

  const showArchive = instance.state === State.ACTIVE && canArchive;
  const showRestore = instance.state === State.DELETED && canRestore;
  const showDelete = canArchive || canRestore;

  if (!showArchive && !showRestore && !showDelete) return null;

  return (
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
          <DropdownMenuItem className="text-error" onClick={handleDelete}>
            {t("common.delete")}
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
