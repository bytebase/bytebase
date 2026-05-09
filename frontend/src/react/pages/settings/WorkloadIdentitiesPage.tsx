import { Plus, Trash2, Undo2 } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CreateWorkloadIdentitySheet } from "@/react/components/CreateWorkloadIdentitySheet";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { UserCell } from "@/react/components/UserCell";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
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
      <Table>
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead className="whitespace-nowrap">
              {t("settings.members.table.account")}
            </TableHead>
            <TableHead className="text-right whitespace-nowrap">
              {t("common.operations")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={2}
                className="py-8 text-center text-control-light"
              >
                {t("common.no-data")}
              </TableCell>
            </TableRow>
          ) : (
            users.map((user) => {
              const isDeleted = user.state === State.DELETED;
              const canOpenDetail =
                !!onUserSelected &&
                (project
                  ? hasProjectPermissionV2(project, "bb.workloadIdentities.get")
                  : hasWorkspacePermissionV2("bb.workloadIdentities.get"));
              const canDelete = project
                ? hasProjectPermissionV2(
                    project,
                    "bb.workloadIdentities.delete"
                  )
                : hasWorkspacePermissionV2("bb.workloadIdentities.delete");
              const canRestore = project
                ? hasProjectPermissionV2(
                    project,
                    "bb.workloadIdentities.undelete"
                  )
                : hasWorkspacePermissionV2("bb.workloadIdentities.undelete");

              return (
                <TableRow
                  key={user.name}
                  className={
                    canOpenDetail
                      ? "cursor-pointer hover:bg-control-bg focus-visible:outline-none focus-visible:bg-control-bg"
                      : undefined
                  }
                  tabIndex={canOpenDetail ? 0 : undefined}
                  role={canOpenDetail ? "button" : undefined}
                  aria-label={
                    canOpenDetail ? user.title || user.email : undefined
                  }
                  onClick={
                    canOpenDetail ? () => onUserSelected(user) : undefined
                  }
                  onKeyDown={
                    canOpenDetail
                      ? (e) => {
                          if (e.key === "Enter" || e.key === " ") {
                            e.preventDefault();
                            onUserSelected(user);
                          }
                        }
                      : undefined
                  }
                >
                  {/* Account column */}
                  <TableCell>
                    <UserCell
                      title={user.title}
                      subtitle={user.email}
                      nameClassName={
                        isDeleted
                          ? "line-through !text-control-light"
                          : undefined
                      }
                    />
                  </TableCell>

                  {/* Operations column — destructive/secondary actions only.
                      The row itself is clickable to open the detail sheet. */}
                  <TableCell>
                    <div className="flex justify-end gap-x-1">
                      {!isDeleted && canDelete && (
                        <Tooltip
                          content={t(
                            "settings.members.action.deactivate-confirm-title"
                          )}
                        >
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-error hover:text-error"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeactivate(user);
                            }}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </Tooltip>
                      )}
                      {isDeleted && canRestore && (
                        <Tooltip
                          content={t(
                            "settings.members.action.reactivate-confirm-title"
                          )}
                        >
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleRestore(user);
                            }}
                          >
                            <Undo2 className="h-4 w-4" />
                          </Button>
                        </Tooltip>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>
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

  // Sequence guard for async row-selection. Each click increments the
  // counter; only the response from the latest click is allowed to update
  // `editingWI`. Without this, fast double-clicks on different rows can
  // resolve out of order and open the sheet with the wrong identity.
  const selectSeqRef = useRef(0);
  const handleUserSelected = async (user: User) => {
    // Use getOrFetch so the full WorkloadIdentity (including
    // workloadIdentityConfig with subjectPattern) is loaded before we open
    // the edit Sheet. `getWorkloadIdentity` returns a stub with only
    // name/email/title if the cache is empty — which it is on first click
    // of a row, since listWorkloadIdentities doesn't populate the cache.
    // Without this, the edit Sheet would see no config and leave
    // Organization/Repository/Branch fields blank.
    const seq = ++selectSeqRef.current;
    const wi = await workloadIdentityStore.getOrFetchWorkloadIdentity(
      user.email,
      true
    );
    // Drop stale response — a later click superseded this request.
    if (seq !== selectSeqRef.current) return;
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
        <PermissionGuard
          permissions={["bb.workloadIdentities.create"]}
          project={project}
        >
          <Button
            disabled={
              project
                ? project.state !== State.ACTIVE ||
                  !hasProjectPermissionV2(
                    project,
                    "bb.workloadIdentities.create"
                  )
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
        </PermissionGuard>
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
          <Checkbox
            checked={showInactive}
            onCheckedChange={(checked) => setShowInactive(checked)}
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

      <CreateWorkloadIdentitySheet
        open={showDrawer}
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
    </div>
  );
}
