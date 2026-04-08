import { EllipsisVertical, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
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

  const [searchText, setSearchText] = useState("");
  const [dbGroupList, setDbGroupList] = useState<DatabaseGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleteTarget, setDeleteTarget] = useState<DatabaseGroup | null>(null);

  // Fetch database groups
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

  const filteredList = useMemo(() => {
    const filter = searchText.trim().toLowerCase();
    if (!filter) return dbGroupList;
    return dbGroupList.filter(
      (group) =>
        group.name.toLowerCase().includes(filter) ||
        group.title.toLowerCase().includes(filter)
    );
  }, [dbGroupList, searchText]);

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
    <div className="py-4 flex flex-col">
      <div className="px-4 flex flex-col gap-y-2 pb-2">
        <FeatureAttention feature={PlanFeature.FEATURE_DATABASE_GROUPS} />
        <div className="flex flex-row items-center justify-end gap-x-2">
          <Input
            className="max-w-full"
            placeholder={t("common.filter-by-name")}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
          />
          <Button disabled={!canCreate} onClick={handleCreate}>
            <FeatureBadge
              feature={PlanFeature.FEATURE_DATABASE_GROUPS}
              className="text-white"
            />
            <Plus className="w-4 h-4 mr-1" />
            {t("common.create")}
          </Button>
        </div>
      </div>

      <DatabaseGroupTable
        databaseGroupList={filteredList}
        loading={loading}
        showActions={canDelete}
        onRowClick={handleRowClick}
        onDelete={setDeleteTarget}
      />

      {/* Delete confirmation dialog */}
      <Dialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <DialogContent>
          <DialogTitle>
            {deleteTarget
              ? t("database-group.delete-group", {
                  name: deleteTarget.title,
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

// ---------------------------------------------------------------------------
// DatabaseGroupTable
// ---------------------------------------------------------------------------
function DatabaseGroupTable({
  databaseGroupList,
  loading,
  showActions,
  onRowClick,
  onDelete,
}: {
  databaseGroupList: DatabaseGroup[];
  loading: boolean;
  showActions: boolean;
  onRowClick: (e: React.MouseEvent, group: DatabaseGroup) => void;
  onDelete: (group: DatabaseGroup) => void;
}) {
  const { t } = useTranslation();
  const PAGE_SIZE = 20;
  const [page, setPage] = useState(0);

  // Reset page when list changes
  useEffect(() => {
    setPage(0);
  }, [databaseGroupList.length]);

  const totalPages = Math.ceil(databaseGroupList.length / PAGE_SIZE);
  const pagedList = databaseGroupList.slice(
    page * PAGE_SIZE,
    (page + 1) * PAGE_SIZE
  );

  if (loading) {
    return (
      <div className="flex justify-center py-8 text-control-light">
        {t("common.loading")}
      </div>
    );
  }

  if (databaseGroupList.length === 0) {
    return (
      <div className="flex justify-center py-8 text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  return (
    <div className="px-4">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b text-left text-control-light">
            <th className="py-2 pr-4 font-medium w-64">{t("common.name")}</th>
            <th className="py-2 pr-4 font-medium">
              {t("database.expression")}
            </th>
            {showActions && <th className="py-2 font-medium w-12" />}
          </tr>
        </thead>
        <tbody>
          {pagedList.map((group) => (
            <tr
              key={group.name}
              className="border-b cursor-pointer hover:bg-gray-50"
              onClick={(e) => onRowClick(e, group)}
            >
              <td className="py-2 pr-4 truncate max-w-64">{group.title}</td>
              <td className="py-2 pr-4 truncate text-control-light">
                {group.databaseExpr?.expression || (
                  <span className="italic">{t("common.empty")}</span>
                )}
              </td>
              {showActions && (
                <td className="py-2">
                  <ActionDropdown group={group} onDelete={onDelete} />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>

      {totalPages > 1 && (
        <div className="flex justify-end items-center gap-x-2 mt-3">
          <Button
            variant="outline"
            size="sm"
            disabled={page === 0}
            onClick={() => setPage((p) => p - 1)}
          >
            {t("common.previous")}
          </Button>
          <span className="text-sm text-control-light">
            {page + 1} / {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages - 1}
            onClick={() => setPage((p) => p + 1)}
          >
            {t("common.next")}
          </Button>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// ActionDropdown
// ---------------------------------------------------------------------------
function ActionDropdown({
  group,
  onDelete,
}: {
  group: DatabaseGroup;
  onDelete: (group: DatabaseGroup) => void;
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
          {/* backdrop */}
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full z-20 mt-1 bg-white border rounded-sm shadow-md min-w-[100px]">
            <button
              type="button"
              className="w-full text-left px-3 py-1.5 text-sm hover:bg-gray-50 text-error"
              onClick={(e) => {
                e.stopPropagation();
                setOpen(false);
                onDelete(group);
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
