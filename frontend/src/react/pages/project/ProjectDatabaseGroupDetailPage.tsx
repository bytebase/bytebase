import { Edit, EllipsisVertical } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseGroupForm } from "@/react/components/DatabaseGroupForm";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { preCreateIssue } from "@/react/lib/plan/issue";
import { router } from "@/react/router";
import { PROJECT_V1_ROUTE_DATABASE_GROUPS } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { isValidDatabaseGroupName, unknownDatabaseGroup } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  hasProjectPermissionV2,
  PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE,
} from "@/utils";

export function ProjectDatabaseGroupDetailPage({
  projectId,
  databaseGroupName,
}: {
  projectId: string;
  databaseGroupName: string;
}) {
  const { t } = useTranslation();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useProjectByName(projectName);

  const resourceName = `${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`;

  // Subscribe to the cached entry directly (stable ref) and derive the
  // unknown fallback outside the selector — returning `unknownDatabaseGroup()`
  // from the selector would yield a fresh object each call and loop forever.
  const cachedGroup = useAppStore((s) => s.dbGroupsByName[resourceName]);
  const databaseGroup = useMemo(
    () => cachedGroup ?? unknownDatabaseGroup(),
    [cachedGroup]
  );

  const [editing, setEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  // Fetch the database group on mount
  useEffect(() => {
    useAppStore.getState().getOrFetchDBGroupByName(resourceName, {
      skipCache: true,
      view: DatabaseGroupView.FULL,
    });
  }, [resourceName]);

  const hasDatabaseGroupFeature = useAppStore((s) =>
    s.hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS)
  );

  const hasMatchedDatabases = useMemo(
    () => (databaseGroup?.matchedDatabases.length ?? 0) > 0,
    [databaseGroup]
  );

  const canUpdate = useMemo(
    () =>
      project
        ? hasProjectPermissionV2(project, "bb.databaseGroups.update")
        : false,
    [project]
  );

  const canDelete = useMemo(
    () =>
      project
        ? hasProjectPermissionV2(project, "bb.databaseGroups.delete")
        : false,
    [project]
  );

  const canChangeDatabase = useMemo(
    () =>
      project
        ? PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE.every((p) =>
            hasProjectPermissionV2(project, p)
          )
        : false,
    [project]
  );

  const handleDelete = useCallback(async () => {
    await useAppStore.getState().deleteDatabaseGroup(resourceName);
    router.push({ name: PROJECT_V1_ROUTE_DATABASE_GROUPS });
    setShowDeleteDialog(false);
  }, [resourceName]);

  if (!databaseGroup || !project) return null;
  if (!isValidDatabaseGroupName(databaseGroup.name)) return null;

  return (
    <div className="min-h-full flex-1 relative flex flex-col gap-y-4 px-4 pt-4">
      <FeatureAttention feature={PlanFeature.FEATURE_DATABASE_GROUPS} />

      {hasDatabaseGroupFeature && !editing && (
        <div className="flex flex-row justify-end items-center flex-wrap shrink gap-x-2 gap-y-2">
          <PermissionGuard
            permissions={["bb.plans.create", "bb.sheets.create"]}
            project={project}
          >
            <Button
              variant="outline"
              disabled={!canChangeDatabase || !hasMatchedDatabases}
              onClick={() => preCreateIssue(project.name, [resourceName])}
              title={
                !hasMatchedDatabases
                  ? t("database-group.no-matched-databases")
                  : undefined
              }
            >
              {t("database.change-database")}
            </Button>
          </PermissionGuard>

          <PermissionGuard
            permissions={["bb.databaseGroups.update"]}
            project={project}
          >
            <Button disabled={!canUpdate} onClick={() => setEditing(true)}>
              <Edit className="size-4 mr-1" />
              {t("common.configure")}
            </Button>
          </PermissionGuard>

          {canDelete && (
            <DropdownMenu>
              <DropdownMenuTrigger className="inline-flex items-center justify-center h-8 px-1 rounded-xs text-sm text-control hover:bg-control-bg cursor-pointer outline-hidden focus-visible:ring-2 focus-visible:ring-accent">
                <EllipsisVertical className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem
                  className="text-error"
                  onClick={() => setShowDeleteDialog(true)}
                >
                  {t("common.delete")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      )}

      <DatabaseGroupForm
        readonly={!editing}
        project={project}
        databaseGroup={databaseGroup}
        onDismiss={() => setEditing(false)}
      />

      <AlertDialog
        open={showDeleteDialog}
        onOpenChange={(open) => {
          if (!open) setShowDeleteDialog(false);
        }}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("database-group.delete-group", { name: databaseGroup.title })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("common.cannot-undo-this-action")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
