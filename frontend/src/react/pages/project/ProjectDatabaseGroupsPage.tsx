import { Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseGroupTable } from "@/react/components/DatabaseGroupTable";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE,
} from "@/router/dashboard/projectV1";
import { useDBGroupStore, useProjectV1Store } from "@/store";
import {
  getProjectNameAndDatabaseGroupName,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

export function ProjectDatabaseGroupsPage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const dbGroupStore = useDBGroupStore();

  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const [dbGroupList, setDbGroupList] = useState<DatabaseGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleteTarget, setDeleteTarget] = useState<DatabaseGroup | null>(null);

  // Fetch database groups for this project.
  useEffect(() => {
    setLoading(true);
    setDbGroupList([]);
    dbGroupStore
      .fetchDBGroupListByProjectName(projectName, DatabaseGroupView.BASIC)
      .then((list) => {
        setDbGroupList(list);
        setLoading(false);
      });
  }, [projectName, dbGroupStore]);

  const canCreate = useMemo(
    () =>
      project
        ? hasProjectPermissionV2(project, "bb.databaseGroups.create")
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

  const handleCreate = useCallback(() => {
    router.push({ name: PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE });
  }, []);

  const handleRowClick = useCallback(
    (e: React.MouseEvent, group: DatabaseGroup) => {
      const [pid, groupName] = getProjectNameAndDatabaseGroupName(group.name);
      const url = router.resolve({
        name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
        params: { projectId: pid, databaseGroupName: groupName },
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
    if (!deleteTarget) return;
    await dbGroupStore.deleteDatabaseGroup(deleteTarget.name);
    setDbGroupList((prev) => prev.filter((g) => g.name !== deleteTarget.name));
    setDeleteTarget(null);
  }, [deleteTarget, dbGroupStore]);

  return (
    <div className="py-4 flex flex-col gap-y-2">
      <div className="px-4 flex flex-col gap-y-2">
        <FeatureAttention feature={PlanFeature.FEATURE_DATABASE_GROUPS} />
      </div>

      <div className="px-4">
        <DatabaseGroupTable
          projectName={projectName}
          view={DatabaseGroupView.BASIC}
          externalList={dbGroupList}
          externalLoading={loading}
          showActions={canDelete}
          pageSize={20}
          onRowClick={handleRowClick}
          onDelete={setDeleteTarget}
          trailingAction={
            <Button disabled={!canCreate} onClick={handleCreate}>
              <FeatureBadge
                feature={PlanFeature.FEATURE_DATABASE_GROUPS}
                className="text-white"
                fallback={<Plus className="size-4 mr-1" />}
              />
              {t("common.create")}
            </Button>
          }
        />
      </div>

      <Dialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <DialogContent>
          <DialogTitle>
            {deleteTarget
              ? t("database-group.delete-group", { name: deleteTarget.title })
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
