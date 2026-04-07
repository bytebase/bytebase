import { EllipsisVertical } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useInstanceV1Store } from "@/store";
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
  const instanceStore = useInstanceV1Store();
  const [open, setOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

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

    await instanceStore.archiveInstance(instance, true);
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("instance.successfully-archived-instance", {
        0: instance.title,
      }),
    });
    router.replace({ name: INSTANCE_ROUTE_DASHBOARD });
  }, [instance, instanceStore, t]);

  const handleRestore = useCallback(async () => {
    if (
      !window.confirm(
        t("instance.restore-instance-instance-name-to-normal-state", {
          0: instance.title,
        })
      )
    )
      return;

    await instanceStore.restoreInstance(instance);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-restored-instance", {
        0: instance.title,
      }),
    });
  }, [instance, instanceStore, t]);

  const handleDelete = useCallback(async () => {
    if (
      !window.confirm(
        `${t("common.delete-resource", { resource: instance.title })}\n\n${t("common.cannot-undo-this-action")}`
      )
    )
      return;

    await instanceStore.deleteInstance(instance.name);
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
  }, [instance, instanceStore, t, onDeleted]);

  const options: { key: string; label: string; handler: () => void }[] = [];

  if (instance.state === State.ACTIVE && canArchive) {
    options.push({
      key: "archive",
      label: t("common.archive"),
      handler: handleArchive,
    });
  } else if (instance.state === State.DELETED && canRestore) {
    options.push({
      key: "restore",
      label: t("common.restore"),
      handler: handleRestore,
    });
  }

  if (canArchive || canRestore) {
    options.push({
      key: "delete",
      label: t("common.delete"),
      handler: handleDelete,
    });
  }

  if (options.length === 0) return null;

  return (
    <div className="relative" ref={dropdownRef}>
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8"
        onClick={() => setOpen(!open)}
      >
        <EllipsisVertical className="w-4 h-4" />
      </Button>
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full mt-1 z-50 min-w-[120px] rounded-md border bg-white shadow-md py-1">
            {options.map((opt) => (
              <button
                key={opt.key}
                type="button"
                className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-100 ${
                  opt.key === "delete" ? "text-red-600" : ""
                }`}
                onClick={() => {
                  setOpen(false);
                  opt.handler();
                }}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
