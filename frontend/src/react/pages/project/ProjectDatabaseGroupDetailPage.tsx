import { Edit, EllipsisVertical } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseGroupForm } from "@/react/components/DatabaseGroupForm";
import { FeatureAttention } from "@/react/components/FeatureAttention";
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
import { useVueState } from "@/react/hooks/useVueState";
import { preCreateIssue } from "@/react/lib/plan/issue";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DATABASE_GROUPS } from "@/router/dashboard/projectV1";
import { hasFeature, useDBGroupStore, useProjectV1Store } from "@/store";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { isValidDatabaseGroupName } from "@/types";
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
  const projectStore = useProjectV1Store();
  const dbGroupStore = useDBGroupStore();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const resourceName = `${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`;

  const databaseGroup = useVueState(() =>
    dbGroupStore.getDBGroupByName(resourceName)
  );

  const [editing, setEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  // Fetch the database group on mount
  useEffect(() => {
    dbGroupStore.getOrFetchDBGroupByName(resourceName, {
      skipCache: true,
      view: DatabaseGroupView.FULL,
    });
  }, [resourceName, dbGroupStore]);

  const hasDatabaseGroupFeature = useVueState(() =>
    hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS)
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
    await dbGroupStore.deleteDatabaseGroup(resourceName);
    router.push({ name: PROJECT_V1_ROUTE_DATABASE_GROUPS });
    setShowDeleteDialog(false);
  }, [resourceName, dbGroupStore]);

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

      <Dialog
        open={showDeleteDialog}
        onOpenChange={(open) => {
          if (!open) setShowDeleteDialog(false);
        }}
      >
        <DialogContent>
          <DialogTitle>
            {t("database-group.delete-group", { name: databaseGroup.title })}
          </DialogTitle>
          <p className="text-sm text-control-light">
            {t("common.cannot-undo-this-action")}
          </p>
          <div className="flex justify-end gap-x-2 mt-4">
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
            >
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
