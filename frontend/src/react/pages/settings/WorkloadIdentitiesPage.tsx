import { Pencil, Plus, Trash2, Undo2 } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { CreateWorkloadIdentityDrawer } from "@/react/components/CreateWorkloadIdentityDrawer";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useVueState } from "@/react/hooks/useVueState";
import {
  ensureWorkloadIdentityFullName,
  pushNotification,
  useActuatorV1Store,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store/modules/workloadIdentity";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

// ============================================================
// WorkloadIdentityTable
// ============================================================

function WorkloadIdentityTable({
  users,
  project,
  onUserUpdated,
  onUserSelected,
}: {
  users: User[];
  project?: Project;
  onUserUpdated: (user: User) => void;
  onUserSelected?: (user: User) => void;
}) {
  const { t } = useTranslation();
  const workloadIdentityStore = useWorkloadIdentityStore();

  const handleDeactivate = async (user: User) => {
    const confirmed = window.confirm(
      t("settings.members.action.deactivate-confirm-title")
    );
    if (!confirmed) return;

    try {
      const fullName = ensureWorkloadIdentityFullName(user.email);
      await workloadIdentityStore.deleteWorkloadIdentity(fullName);
      const updated = { ...user, state: State.DELETED };
      onUserUpdated(updated as User);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    }
  };

  const handleRestore = async (user: User) => {
    try {
      const fullName = ensureWorkloadIdentityFullName(user.email);
      await workloadIdentityStore.undeleteWorkloadIdentity(fullName);
      const updated = { ...user, state: State.ACTIVE };
      onUserUpdated(updated as User);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    }
  };

  return (
    <div className="border rounded-sm overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-control-bg">
            <th className="px-4 py-2 text-left font-medium whitespace-nowrap">
              {t("settings.members.table.account")}
            </th>
            <th className="px-4 py-2 text-right font-medium whitespace-nowrap">
              {t("common.operations")}
            </th>
          </tr>
        </thead>
        <tbody>
          {users.length === 0 ? (
            <tr>
              <td
                colSpan={2}
                className="py-8 text-center text-control-light text-sm"
              >
                {t("common.no-data")}
              </td>
            </tr>
          ) : (
            users.map((user, i) => {
              const isDeleted = user.state === State.DELETED;

              return (
                <tr
                  key={user.name}
                  className={`border-b last:border-b-0 ${i % 2 === 1 ? "bg-gray-50" : ""}`}
                >
                  {/* Account column */}
                  <td className="px-4 py-2">
                    <div className="flex items-center gap-x-3">
                      <UserAvatar title={user.title || user.email} />
                      <div className="flex flex-col">
                        <span
                          className={
                            isDeleted
                              ? "line-through text-control-light font-medium"
                              : "font-medium text-accent"
                          }
                        >
                          {user.title || user.email}
                        </span>
                        <span className="textinfolabel text-xs">
                          {user.email}
                        </span>
                      </div>
                    </div>
                  </td>

                  {/* Operations column */}
                  <td className="px-4 py-2">
                    <div className="flex justify-end gap-x-1">
                      {!isDeleted && (
                        <>
                          {(project
                            ? hasProjectPermissionV2(
                                project,
                                "bb.workloadIdentities.delete"
                              )
                            : hasWorkspacePermissionV2(
                                "bb.workloadIdentities.delete"
                              )) && (
                            <Tooltip
                              content={t(
                                "settings.members.action.deactivate-confirm-title"
                              )}
                            >
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 text-error hover:text-error"
                                onClick={() => handleDeactivate(user)}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </Tooltip>
                          )}
                          {(project
                            ? hasProjectPermissionV2(
                                project,
                                "bb.workloadIdentities.get"
                              )
                            : hasWorkspacePermissionV2(
                                "bb.workloadIdentities.get"
                              )) &&
                            onUserSelected && (
                              <Tooltip content={t("common.edit")}>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-7 w-7"
                                  onClick={() => onUserSelected(user)}
                                >
                                  <Pencil className="h-4 w-4" />
                                </Button>
                              </Tooltip>
                            )}
                        </>
                      )}
                      {isDeleted &&
                        (project
                          ? hasProjectPermissionV2(
                              project,
                              "bb.workloadIdentities.undelete"
                            )
                          : hasWorkspacePermissionV2(
                              "bb.workloadIdentities.undelete"
                            )) && (
                          <Tooltip
                            content={t(
                              "settings.members.action.reactivate-confirm-title"
                            )}
                          >
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7"
                              onClick={() => handleRestore(user)}
                            >
                              <Undo2 className="h-4 w-4" />
                            </Button>
                          </Tooltip>
                        )}
                    </div>
                  </td>
                </tr>
              );
            })
          )}
        </tbody>
      </table>
    </div>
  );
}

// ============================================================
// WorkloadIdentitiesPage (main)
// ============================================================

export function WorkloadIdentitiesPage({ projectId }: { projectId?: string }) {
  const { t } = useTranslation();
  const workloadIdentityStore = useWorkloadIdentityStore();
  const actuatorStore = useActuatorV1Store();
  const projectStore = useProjectV1Store();

  const projectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : undefined;
  const project = useVueState(() =>
    projectName ? projectStore.getProjectByName(projectName) : undefined
  );

  const parent = useVueState(
    () => projectName ?? actuatorStore.workspaceResourceName
  );

  const [showInactive, setShowInactive] = useState(false);
  const [showDrawer, setShowDrawer] = useState(false);
  const [editingWI, setEditingWI] = useState<WorkloadIdentity | undefined>(
    undefined
  );

  const fetchActive = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const response = await workloadIdentityStore.listWorkloadIdentities({
        parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        showDeleted: false,
      });
      return {
        list: response.workloadIdentities.map(workloadIdentityToUser),
        nextPageToken: response.nextPageToken,
      };
    },
    [workloadIdentityStore, parent]
  );

  const fetchInactive = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const response = await workloadIdentityStore.listWorkloadIdentities({
        parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        showDeleted: true,
        filter: { state: State.DELETED },
      });
      return {
        list: response.workloadIdentities.map(workloadIdentityToUser),
        nextPageToken: response.nextPageToken,
      };
    },
    [workloadIdentityStore, parent]
  );

  const activeData = usePagedData<User>({
    sessionKey: `bb.paged-workload-identity-table${projectName ? `.${projectName}` : ""}.active`,
    fetchList: fetchActive,
  });

  const inactiveData = usePagedData<User>({
    sessionKey: `bb.paged-workload-identity-table${projectName ? `.${projectName}` : ""}.deleted`,
    fetchList: fetchInactive,
    enabled: showInactive,
  });

  const handleActiveUpdated = (user: User) => {
    if (user.state === State.DELETED) {
      activeData.removeCache(user);
      inactiveData.updateCache([user]);
    } else {
      activeData.updateCache([user]);
    }
  };

  const handleInactiveUpdated = (user: User) => {
    if (user.state === State.ACTIVE) {
      inactiveData.removeCache(user);
      activeData.refresh();
    } else {
      inactiveData.updateCache([user]);
    }
  };

  const handleUserSelected = (user: User) => {
    const wi = workloadIdentityStore.getWorkloadIdentity(user.email);
    setEditingWI(wi);
    setShowDrawer(true);
  };

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <p className="text-lg font-medium leading-7 text-main">
          {t("settings.members.workload-identities")}
        </p>
        <Button
          disabled={
            project
              ? project.state !== State.ACTIVE ||
                !hasProjectPermissionV2(project, "bb.workloadIdentities.create")
              : !hasWorkspacePermissionV2("bb.workloadIdentities.create")
          }
          onClick={() => {
            setEditingWI(undefined);
            setShowDrawer(true);
          }}
        >
          <Plus className="h-4 w-4 mr-1" />
          {t("common.create")}
        </Button>
      </div>

      <div className="flex flex-col gap-y-4">
        {activeData.isLoading && activeData.dataList.length === 0 ? (
          <div className="flex items-center justify-center h-32">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        ) : (
          <>
            <WorkloadIdentityTable
              users={activeData.dataList}
              project={project}
              onUserUpdated={handleActiveUpdated}
              onUserSelected={handleUserSelected}
            />
            <PagedTableFooter
              pageSize={activeData.pageSize}
              pageSizeOptions={activeData.pageSizeOptions}
              onPageSizeChange={activeData.onPageSizeChange}
              hasMore={activeData.hasMore}
              isFetchingMore={activeData.isFetchingMore}
              onLoadMore={activeData.loadMore}
            />
          </>
        )}

        {/* Show inactive toggle */}
        <label className="flex items-center gap-x-2 text-sm cursor-pointer">
          <input
            type="checkbox"
            checked={showInactive}
            onChange={(e) => setShowInactive(e.target.checked)}
          />
          <span className="textinfolabel">
            {t("settings.members.show-inactive")}
          </span>
        </label>

        {showInactive && (
          <div className="flex flex-col gap-y-4">
            <p className="text-lg font-medium leading-7">
              {t("settings.members.inactive-workload-identities")}
            </p>

            {inactiveData.isLoading && inactiveData.dataList.length === 0 ? (
              <div className="flex items-center justify-center h-32">
                <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
              </div>
            ) : (
              <>
                <WorkloadIdentityTable
                  users={inactiveData.dataList}
                  project={project}
                  onUserUpdated={handleInactiveUpdated}
                />
                <PagedTableFooter
                  pageSize={inactiveData.pageSize}
                  pageSizeOptions={inactiveData.pageSizeOptions}
                  onPageSizeChange={inactiveData.onPageSizeChange}
                  hasMore={inactiveData.hasMore}
                  isFetchingMore={inactiveData.isFetchingMore}
                  onLoadMore={inactiveData.loadMore}
                />
              </>
            )}
          </div>
        )}
      </div>

      {showDrawer && (
        <CreateWorkloadIdentityDrawer
          workloadIdentity={editingWI}
          project={projectName}
          onClose={() => {
            setShowDrawer(false);
            setEditingWI(undefined);
          }}
          onCreated={(wi) => {
            activeData.updateCache([workloadIdentityToUser(wi)]);
          }}
          onUpdated={(wi) => {
            activeData.updateCache([workloadIdentityToUser(wi)]);
          }}
        />
      )}
    </div>
  );
}
