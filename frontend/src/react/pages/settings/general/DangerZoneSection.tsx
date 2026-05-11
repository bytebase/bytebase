import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { workspaceServiceClientConnect } from "@/connect";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useCurrentUser, useWorkspace } from "@/react/hooks/useAppState";
import { useVueState } from "@/react/hooks/useVueState";
import { pushNotification, useWorkspaceV1Store } from "@/store";
import { PresetRoleType } from "@/types/iam/role";
import { userBindingPrefix } from "@/types/v1/user";
import { hasWorkspacePermissionV2 } from "@/utils";

export function DangerZoneSection() {
  const { t } = useTranslation();
  const workspace = useWorkspace();
  const currentUser = useCurrentUser()!;
  const workspaceStore = useWorkspaceV1Store();
  const workspacePolicy = useVueState(() => workspaceStore.workspaceIamPolicy);

  const canDelete = hasWorkspacePermissionV2("bb.workspaces.delete");

  // Check if there are other admins in the workspace besides the current user.
  // Group bindings (group:...) are treated as "other admin" since they may
  // contain other users — the backend does the definitive check.
  const hasOtherAdmin = useMemo(() => {
    const currentBinding = `${userBindingPrefix}${currentUser.email}`;
    for (const binding of workspacePolicy.bindings) {
      if (binding.role !== PresetRoleType.WORKSPACE_ADMIN) continue;
      for (const member of binding.members) {
        if (member === currentBinding) continue;
        // A group binding counts as "other admin" — it may contain
        // other users who are admins through the group.
        if (member.startsWith("group:") || member.startsWith("groups/")) {
          return true;
        }
        // A different direct user binding.
        return true;
      }
    }
    return false;
  }, [workspacePolicy, currentUser.email]);

  // --- Delete workspace ---
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [confirmText, setConfirmText] = useState("");
  const [deleting, setDeleting] = useState(false);
  const workspaceTitle = workspace?.title ?? "";
  const canConfirmDelete = confirmText === workspaceTitle && !deleting;

  const handleDelete = useCallback(async () => {
    if (!workspace?.name) return;
    setDeleting(true);
    try {
      await workspaceServiceClientConnect.deleteWorkspace({
        name: workspace.name,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("settings.general.workspace.danger-zone.deleted"),
      });
      // The backend switches to the next workspace (new auth cookies set)
      // or returns empty if no workspace remains (login will provision one).
      window.location.href = "/";
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("settings.general.workspace.danger-zone.delete-failed"),
      });
      setDeleting(false);
    }
  }, [workspace?.name, t]);

  // --- Exit workspace ---
  const [exitOpen, setExitOpen] = useState(false);
  const [exiting, setExiting] = useState(false);

  const handleExit = useCallback(async () => {
    if (!workspace?.name) return;
    setExiting(true);
    try {
      // The backend removes the user from the workspace IAM, switches to
      // the next available workspace, and sets new auth cookies.
      await workspaceServiceClientConnect.leaveWorkspace({
        name: workspace.name,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("settings.general.workspace.danger-zone.exited"),
      });
      window.location.href = "/";
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("settings.general.workspace.danger-zone.exit-failed"),
      });
      setExiting(false);
    }
  }, [workspace?.name, t]);

  return (
    <div className="py-6 lg:flex">
      <div className="text-left lg:w-1/4">
        <h1 className="text-2xl font-bold">{t("common.danger-zone")}</h1>
      </div>
      <div className="flex-1 mt-4 lg:px-4 lg:mt-0">
        <div className="border border-error-alpha bg-error-alpha rounded-sm divide-y divide-error-alpha">
          {/* Exit workspace */}
          {hasOtherAdmin && (
            <div className="p-6 flex items-start justify-between gap-x-6">
              <div className="flex-1">
                <h4 className="font-medium text-error">
                  {t("settings.general.workspace.danger-zone.exit-workspace")}
                </h4>
                <p className="text-sm text-control-light mt-1">
                  {t("settings.general.workspace.danger-zone.exit-description")}
                </p>
              </div>
              <Button
                variant="destructive"
                disabled={exiting}
                onClick={() => setExitOpen(true)}
              >
                {t("settings.general.workspace.danger-zone.exit-workspace")}
              </Button>
            </div>
          )}

          {/* Delete workspace */}
          {canDelete && (
            <div className="p-6 flex items-start justify-between gap-x-6">
              <div className="flex-1">
                <h4 className="font-medium text-error">
                  {t("settings.general.workspace.danger-zone.delete-workspace")}
                </h4>
                <p className="text-sm text-control-light mt-1">
                  {t("settings.general.workspace.danger-zone.description")}
                </p>
              </div>
              <Button
                variant="destructive"
                disabled={deleting}
                onClick={() => setDeleteOpen(true)}
              >
                {t("settings.general.workspace.danger-zone.delete-workspace")}
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Exit workspace dialog */}
      <AlertDialog open={exitOpen}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogTitle>
            {t("settings.general.workspace.danger-zone.exit-confirm-title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "settings.general.workspace.danger-zone.exit-confirm-description",
              { workspace: workspaceTitle }
            )}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              variant="outline"
              onClick={() => setExitOpen(false)}
              disabled={exiting}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleExit()}
              disabled={exiting}
            >
              {exiting
                ? t("settings.general.workspace.danger-zone.exiting")
                : t("settings.general.workspace.danger-zone.exit-workspace")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete workspace dialog */}
      <AlertDialog open={deleteOpen}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogTitle>
            {t("settings.general.workspace.danger-zone.confirm-title")}
          </AlertDialogTitle>
          <AlertDialogDescription className="flex flex-col gap-y-3">
            <span>
              {t("settings.general.workspace.danger-zone.confirm-description", {
                workspace: workspaceTitle,
              })}
            </span>
            <Input
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder={workspaceTitle}
            />
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setDeleteOpen(false);
                setConfirmText("");
              }}
              disabled={deleting}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleDelete()}
              disabled={!canConfirmDelete}
            >
              {deleting
                ? t("settings.general.workspace.danger-zone.deleting")
                : t("settings.general.workspace.danger-zone.delete-workspace")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
