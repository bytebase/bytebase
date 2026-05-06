import { EllipsisVertical, Plus } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
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
        <PermissionGuard permissions={["bb.projects.update"]} project={project}>
          <Button disabled={!allowEdit} onClick={handleAdd}>
            <Plus className="size-4 mr-1" />
            {t("common.create")}
          </Button>
        </PermissionGuard>
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

  const activityItemList = projectWebhookV1ActivityItemList();

  return (
    <div className="px-4">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-60">{t("common.name")}</TableHead>
            <TableHead>URL</TableHead>
            <TableHead>{t("project.webhook.triggering-activity")}</TableHead>
            {allowEdit && <TableHead className="w-12" />}
          </TableRow>
        </TableHeader>
        <TableBody>
          {webhooks.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={allowEdit ? 4 : 3}
                className="py-8 text-center text-control-light"
              >
                {t("common.no-data")}
              </TableCell>
            </TableRow>
          ) : (
            webhooks.map((webhook) => {
              const activityTitles = webhook.notificationTypes.map(
                (activity) => {
                  const item = activityItemList.find(
                    (item) => item.activity === activity
                  );
                  return item
                    ? item.title
                    : Activity_Type[activity] || `ACTIVITY_${activity}`;
                }
              );

              return (
                <TableRow
                  key={webhook.name}
                  className="cursor-pointer"
                  onClick={(e) => onRowClick(e, webhook)}
                >
                  <TableCell>
                    <div className="flex items-center gap-x-2">
                      <WebhookTypeIcon type={webhook.type} className="size-5" />
                      {webhook.title}
                    </div>
                  </TableCell>
                  <TableCell className="truncate max-w-xs text-control-light">
                    {webhook.url}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-2">
                      {activityTitles.map((title) => (
                        <span
                          key={title}
                          className="inline-block px-2 py-0.5 text-xs rounded-xs bg-control-bg text-control"
                        >
                          {title}
                        </span>
                      ))}
                    </div>
                  </TableCell>
                  {allowEdit && (
                    <TableCell>
                      <ActionDropdown webhook={webhook} onDelete={onDelete} />
                    </TableCell>
                  )}
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>
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

  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger
          className="p-1 rounded-xs hover:bg-control-bg outline-hidden"
          onClick={(e) => e.stopPropagation()}
        >
          <EllipsisVertical className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem
            className="text-error"
            onClick={(e) => {
              e.stopPropagation();
              onDelete(webhook);
            }}
          >
            {t("common.delete")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
