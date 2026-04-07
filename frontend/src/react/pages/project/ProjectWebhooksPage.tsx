import { EllipsisVertical, Plus } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { WebhookTypeIcon } from "@/react/components/WebhookTypeIcon";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_WEBHOOK_CREATE,
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useProjectV1Store,
  useProjectWebhookV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { projectWebhookV1ActivityItemList } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Webhook } from "@/types/proto-es/v1/project_service_pb";
import { Activity_Type } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectWebhookID, hasProjectPermissionV2 } from "@/utils";

export function ProjectWebhooksPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const projectWebhookV1Store = useProjectWebhookV1Store();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const [deleteTarget, setDeleteTarget] = useState<Webhook | null>(null);

  const allowEdit = useMemo(() => {
    if (!project) return false;
    if (project.state === State.DELETED) return false;
    return hasProjectPermissionV2(project, "bb.projects.update");
  }, [project]);

  const webhooks = useMemo(() => project?.webhooks ?? [], [project]);

  const handleAdd = useCallback(() => {
    router.push({ name: PROJECT_V1_ROUTE_WEBHOOK_CREATE });
  }, []);

  const handleRowClick = useCallback(
    (e: React.MouseEvent, webhook: Webhook) => {
      const url = router.resolve({
        name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
        params: {
          webhookResourceId: extractProjectWebhookID(webhook.name),
        },
      }).fullPath;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
    []
  );

  const handleDelete = useCallback(async () => {
    if (!deleteTarget || !project) return;
    try {
      const name = deleteTarget.title;
      const updatedProject =
        await projectWebhookV1Store.deleteProjectWebhook(deleteTarget);
      projectStore.updateProjectCache({
        ...project,
        ...updatedProject,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.webhook.success-deleted-prompt", { name }),
      });
    } catch (error: unknown) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: (error as { message?: string })?.message ?? String(error),
      });
    }
    setDeleteTarget(null);
  }, [deleteTarget, project, projectWebhookV1Store, projectStore, t]);

  return (
    <div className="py-4 flex flex-col">
      <div className="px-4 pb-2 flex items-center justify-end">
        <Button disabled={!allowEdit} onClick={handleAdd}>
          <Plus className="w-4 h-4 mr-1" />
          {t("project.webhook.add-a-webhook")}
        </Button>
      </div>

      <WebhookTable
        webhooks={webhooks}
        allowEdit={allowEdit}
        onRowClick={handleRowClick}
        onDelete={setDeleteTarget}
      />

      <Dialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <DialogContent>
          <DialogTitle>
            {deleteTarget
              ? t("project.webhook.deletion.confirm-title", {
                  title: deleteTarget.title,
                })
              : ""}
          </DialogTitle>
          <p className="text-sm text-control-light">
            {t("common.cannot-undo-this-action")}
          </p>
          <div className="flex justify-end gap-x-2 mt-4">
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              {t("common.delete")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function WebhookTable({
  webhooks,
  allowEdit,
  onRowClick,
  onDelete,
}: {
  webhooks: Webhook[];
  allowEdit: boolean;
  onRowClick: (e: React.MouseEvent, webhook: Webhook) => void;
  onDelete: (webhook: Webhook) => void;
}) {
  const { t } = useTranslation();

  if (webhooks.length === 0) {
    return (
      <div className="flex justify-center py-8 text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  const activityItemList = projectWebhookV1ActivityItemList();

  return (
    <div className="px-4">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b text-left text-control-light">
            <th className="py-2 pr-4 font-medium w-60">{t("common.name")}</th>
            <th className="py-2 pr-4 font-medium">URL</th>
            <th className="py-2 pr-4 font-medium">
              {t("project.webhook.triggering-activity")}
            </th>
            {allowEdit && <th className="py-2 font-medium w-12" />}
          </tr>
        </thead>
        <tbody>
          {webhooks.map((webhook) => {
            const activityTitles = webhook.notificationTypes.map((activity) => {
              const item = activityItemList.find(
                (item) => item.activity === activity
              );
              return item
                ? item.title
                : Activity_Type[activity] || `ACTIVITY_${activity}`;
            });

            return (
              <tr
                key={webhook.name}
                className="border-b cursor-pointer hover:bg-gray-50"
                onClick={(e) => onRowClick(e, webhook)}
              >
                <td className="py-2 pr-4">
                  <div className="flex items-center gap-x-2">
                    <WebhookTypeIcon type={webhook.type} className="w-5 h-5" />
                    {webhook.title}
                  </div>
                </td>
                <td className="py-2 pr-4 truncate max-w-xs text-control-light">
                  {webhook.url}
                </td>
                <td className="py-2 pr-4">
                  <div className="flex flex-wrap gap-2">
                    {activityTitles.map((title) => (
                      <span
                        key={title}
                        className="inline-block px-2 py-0.5 text-xs rounded-xs bg-gray-100 text-gray-700"
                      >
                        {title}
                      </span>
                    ))}
                  </div>
                </td>
                {allowEdit && (
                  <td className="py-2">
                    <ActionDropdown webhook={webhook} onDelete={onDelete} />
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function ActionDropdown({
  webhook,
  onDelete,
}: {
  webhook: Webhook;
  onDelete: (webhook: Webhook) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  return (
    <div className="relative flex justify-end">
      <button
        type="button"
        className="p-1 rounded-xs hover:bg-gray-100"
        onClick={(e) => {
          e.stopPropagation();
          setOpen((v) => !v);
        }}
      >
        <EllipsisVertical className="w-4 h-4" />
      </button>
      {open && (
        <>
          <div
            className="fixed inset-0 z-10"
            onClick={(e) => {
              e.stopPropagation();
              setOpen(false);
            }}
          />
          <div className="absolute right-0 top-full z-20 mt-1 bg-white border rounded-sm shadow-md min-w-[100px]">
            <button
              type="button"
              className="w-full text-left px-3 py-1.5 text-sm hover:bg-gray-50 text-error"
              onClick={(e) => {
                e.stopPropagation();
                setOpen(false);
                onDelete(webhook);
              }}
            >
              {t("common.delete")}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
