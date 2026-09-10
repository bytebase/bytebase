import { create } from "@bufbuild/protobuf";
import { Plus, Settings } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { WORKSPACE_ROUTE_GROUPS } from "@/app/router/handles";
import { ComponentPermissionGuard } from "@/components/ComponentPermissionGuard";
import { FeatureBadge } from "@/components/FeatureBadge";
import { UserCell } from "@/components/UserCell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { SearchInput } from "@/components/ui/search-input";
import { listRowStateClassName } from "@/components/ui/styles.stylex";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import { useCurrentUser } from "@/hooks/useAppState";
import { PagedTableFooter, usePagedData } from "@/hooks/usePagedData";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { getUserFullNameByType } from "@/stores/modules/v1/common";
import { AccountType, getAccountTypeByEmail } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { AADSyncSheet } from "./shared/AADSyncSheet";
import { EditUserSheet } from "./users/EditUserSheet";
import { UserFormSheet } from "./users/UserFormSheet";
import { UserRowMenu } from "./users/UserRowMenu";

// ============================================================
// UserTable
// ============================================================

function UserTable({
  users,
  onUserUpdated,
  onUserEdit,
  onGroupSelected,
}: {
  users: User[];
  onUserUpdated: (user: User, replaces?: User) => void;
  onUserEdit?: (user: User) => void;
  onGroupSelected?: (group: Group) => void;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const archiveUser = useAppStore((state) => state.archiveUser);
  const restoreUser = useAppStore((state) => state.restoreUser);
  const batchGetOrFetchGroups = useAppStore(
    (state) => state.batchGetOrFetchGroups
  );
  const deleteServiceAccount = useAppStore(
    (state) => state.deleteServiceAccount
  );
  const undeleteServiceAccount = useAppStore(
    (state) => state.undeleteServiceAccount
  );
  const deleteWorkloadIdentity = useAppStore(
    (state) => state.deleteWorkloadIdentity
  );
  const undeleteWorkloadIdentity = useAppStore(
    (state) => state.undeleteWorkloadIdentity
  );

  // Batch fetch groups when user list changes
  useEffect(() => {
    const allGroupNames = users.flatMap((u) => u.groups);
    if (allGroupNames.length > 0) {
      batchGetOrFetchGroups(allGroupNames);
    }
  }, [users, batchGetOrFetchGroups]);

  const handleDeactivate = async (user: User) => {
    const accountType = getAccountTypeByEmail(user.email);
    const fullName = getUserFullNameByType(user);

    try {
      if (accountType === AccountType.SERVICE_ACCOUNT) {
        await deleteServiceAccount(fullName);
      } else if (accountType === AccountType.WORKLOAD_IDENTITY) {
        await deleteWorkloadIdentity(fullName);
      } else {
        await archiveUser(fullName);
      }

      const updated = create(UserSchema, { ...user, state: State.DELETED });
      onUserUpdated(updated);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      // The store already surfaced the error; rethrow so the confirm dialog
      // stays open instead of dismissing as if the change had landed.
      throw error;
    }
  };

  const handleRestore = async (user: User) => {
    const accountType = getAccountTypeByEmail(user.email);
    const fullName = getUserFullNameByType(user);

    try {
      if (accountType === AccountType.SERVICE_ACCOUNT) {
        await undeleteServiceAccount(fullName);
      } else if (accountType === AccountType.WORKLOAD_IDENTITY) {
        await undeleteWorkloadIdentity(fullName);
      } else {
        await restoreUser(fullName);
      }

      const updated = create(UserSchema, { ...user, state: State.ACTIVE });
      onUserUpdated(updated);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      throw error;
    }
  };

  const getViewPermission = (accountType: AccountType) => {
    switch (accountType) {
      case AccountType.SERVICE_ACCOUNT:
        return "bb.serviceAccounts.get";
      case AccountType.WORKLOAD_IDENTITY:
        return "bb.workloadIdentities.get";
      default:
        return "bb.users.get";
    }
  };

  const getAccountTypeLabel = (accountType: AccountType) => {
    switch (accountType) {
      case AccountType.SERVICE_ACCOUNT:
        return t("settings.members.service-account");
      case AccountType.WORKLOAD_IDENTITY:
        return t("settings.members.workload-identity");
      default:
        return "";
    }
  };

  if (users.length === 0) {
    return (
      <div className="py-8 text-center text-control-light text-sm">
        {t("common.no-data")}
      </div>
    );
  }

  return (
    <div className="border rounded-sm overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead className="whitespace-nowrap">
              {t("settings.members.table.account")}
            </TableHead>
            <TableHead className="whitespace-nowrap">
              {t("settings.members.table.groups")}
            </TableHead>
            <TableHead className="text-right whitespace-nowrap">
              {t("common.operations")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.map((user) => {
            const accountType = getAccountTypeByEmail(user.email);
            const isDeleted = user.state === State.DELETED;
            const isSelf = currentUser.name === user.name;
            const canView = hasWorkspacePermissionV2(
              getViewPermission(accountType)
            );

            return (
              <TableRow key={user.name} className={listRowStateClassName}>
                {/* Account column */}
                <TableCell className="py-2">
                  <UserCell
                    title={user.title}
                    subtitle={user.email}
                    hoverEmail={
                      canView && accountType === AccountType.USER
                        ? user.email
                        : undefined
                    }
                    nameClassName={
                      isDeleted ? "line-through !text-control-light" : undefined
                    }
                    badges={
                      <>
                        {isSelf && (
                          <Badge
                            variant="secondary"
                            className="text-xs px-1.5 py-0"
                          >
                            {t("common.you")}
                          </Badge>
                        )}
                        {accountType !== AccountType.USER && (
                          <Badge className="text-xs px-1.5 py-0">
                            {getAccountTypeLabel(accountType)}
                          </Badge>
                        )}
                        {user.mfaEnabled && (
                          <Badge
                            variant="success"
                            className="text-xs px-1.5 py-0"
                          >
                            {t("two-factor.enabled")}
                          </Badge>
                        )}
                        {user.profile?.source && (
                          <Badge className="text-xs px-1.5 py-0">
                            {user.profile.source}
                          </Badge>
                        )}
                      </>
                    }
                  />
                </TableCell>

                {/* Groups column */}
                <TableCell className="py-2">
                  <UserGroupsCell
                    user={user}
                    onGroupSelected={onGroupSelected}
                  />
                </TableCell>

                {/* Operations column — the overflow menu is the row's only
                    affordance, so a row has exactly one place to act from. */}
                <TableCell className="py-2">
                  <div className="flex justify-end">
                    <UserRowMenu
                      user={user}
                      isSelf={isSelf}
                      onUserUpdated={onUserUpdated}
                      onEdit={(u) => onUserEdit?.(u)}
                      onDeactivate={handleDeactivate}
                      onReactivate={handleRestore}
                    />
                  </div>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

// ============================================================
// UserGroupsCell
// ============================================================

function UserGroupsCell({
  user,
  onGroupSelected,
}: {
  user: User;
  onGroupSelected?: (group: Group) => void;
}) {
  const getGroupByIdentifier = useAppStore(
    (state) => state.getGroupByIdentifier
  );

  if (!user.groups || user.groups.length === 0) {
    return <span className="text-control-light">-</span>;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {user.groups.map((groupName) => {
        const group = getGroupByIdentifier(groupName);
        return (
          <Badge
            key={groupName}
            variant="secondary"
            className="text-xs px-1.5 py-0 cursor-pointer"
            // The parent TableRow is a row-click button — stop propagation so
            // clicking a group badge navigates to the group without also
            // opening the user detail sheet.
            onClick={(e) => {
              e.stopPropagation();
              if (group && onGroupSelected) {
                onGroupSelected(group);
              }
            }}
          >
            {group?.title || groupName}
          </Badge>
        );
      })}
    </div>
  );
}

// ============================================================
// UsersPage (main)
// ============================================================

export function UsersPage() {
  const { t } = useTranslation();
  const listUsers = useAppStore((state) => state.listUsers);

  const isSaaSMode = useAppStore((s) => s.isSaaSMode());

  const hasDirectorySyncFeature = useAppStore((s) =>
    s.hasInstanceFeature(PlanFeature.FEATURE_DIRECTORY_SYNC)
  );
  const canAccessSettings = hasWorkspacePermissionV2("bb.settings.get");

  const [userSearchText, setUserSearchText] = useState("");
  const [showInactiveUsers, setShowInactiveUsers] = useState(false);
  const [inactiveUserSearchText, setInactiveUserSearchText] = useState("");

  // Drawer visibility
  const [showCreateUserDrawer, setShowCreateUserDrawer] = useState(false);
  const [showAadSyncDrawer, setShowAadSyncDrawer] = useState(false);
  const [editingUser, setEditingUser] = useState<User | undefined>(undefined);

  // Active users paged data
  const fetchActiveUsers = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const result = await listUsers({
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        filter: { query: userSearchText },
      });
      return { list: result.users, nextPageToken: result.nextPageToken };
    },
    [listUsers, userSearchText]
  );

  const hasUserListPermission = hasWorkspacePermissionV2("bb.users.list");
  const activeUsers = usePagedData<User>({
    sessionKey: "bb.users.active.page-size",
    fetchList: fetchActiveUsers,
    enabled: !isSaaSMode && hasUserListPermission,
  });

  // Inactive users paged data
  const fetchInactiveUsers = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const result = await listUsers({
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        filter: { query: inactiveUserSearchText, state: State.DELETED },
        showDeleted: true,
      });
      return { list: result.users, nextPageToken: result.nextPageToken };
    },
    [listUsers, inactiveUserSearchText]
  );

  const inactiveUsers = usePagedData<User>({
    sessionKey: "bb.users.inactive.page-size",
    enabled: !isSaaSMode && hasUserListPermission && showInactiveUsers,
    fetchList: fetchInactiveUsers,
  });

  const handleActiveUserUpdated = (user: User, replaces?: User) => {
    if (replaces && replaces.name !== user.name) {
      activeUsers.removeCache(replaces);
    }
    if (user.state === State.DELETED) {
      // Deactivated: remove from active list, add to inactive list
      activeUsers.removeCache(user);
      inactiveUsers.updateCache([user]);
    } else {
      activeUsers.updateCache([user]);
    }
  };

  const handleInactiveUserUpdated = (user: User, replaces?: User) => {
    if (replaces && replaces.name !== user.name) {
      inactiveUsers.removeCache(replaces);
    }
    if (user.state === State.ACTIVE) {
      // Restored: remove from inactive list, add to active list
      inactiveUsers.removeCache(user);
      activeUsers.updateCache([user]);
    } else {
      inactiveUsers.updateCache([user]);
    }
  };

  const handleGroupSelected = (group: Group) => {
    router.push({ name: WORKSPACE_ROUTE_GROUPS, query: { name: group.name } });
  };

  return (
    <WorkspacePageLayout>
      {/* Action bar */}
      <WorkspacePageToolbar>
        <SearchInput
          placeholder={t("common.filter-by-name")}
          value={userSearchText}
          onChange={(e) => setUserSearchText(e.target.value)}
        />
        <div className="flex items-center gap-x-2">
          <Button
            appearance="outline"
            disabled={!hasDirectorySyncFeature || !canAccessSettings}
            onClick={() => setShowAadSyncDrawer(true)}
          >
            <Settings className="h-4 w-4 mr-1" />
            <FeatureBadge
              feature={PlanFeature.FEATURE_DIRECTORY_SYNC}
              clickable={false}
            />
            {t("settings.members.entra-sync.self")}
          </Button>
          <Button
            disabled={!hasWorkspacePermissionV2("bb.users.create")}
            onClick={() => setShowCreateUserDrawer(true)}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("common.create")}
          </Button>
        </div>
      </WorkspacePageToolbar>

      <div className="flex flex-col gap-y-4">
        <ComponentPermissionGuard permissions={["bb.users.list"]}>
          {activeUsers.isLoading && activeUsers.dataList.length === 0 ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
            </div>
          ) : (
            <>
              <UserTable
                users={activeUsers.dataList}
                onUserUpdated={handleActiveUserUpdated}
                onUserEdit={(user) => setEditingUser(user)}
                onGroupSelected={handleGroupSelected}
              />
              <PagedTableFooter
                pageSize={activeUsers.pageSize}
                pageSizeOptions={activeUsers.pageSizeOptions}
                onPageSizeChange={activeUsers.onPageSizeChange}
                hasMore={activeUsers.hasMore}
                isFetchingMore={activeUsers.isFetchingMore}
                onLoadMore={activeUsers.loadMore}
              />
            </>
          )}
        </ComponentPermissionGuard>

        {/* Inactive users toggle (only shown with list permission) */}
        {hasUserListPermission && (
          <label className="flex items-center gap-x-2 text-sm cursor-pointer">
            <Checkbox
              checked={showInactiveUsers}
              onCheckedChange={(checked) => setShowInactiveUsers(checked)}
            />
            {t("settings.members.show-inactive")}
          </label>
        )}

        {showInactiveUsers && (
          <div className="flex flex-col gap-y-4">
            <div className="flex items-center justify-between gap-x-4">
              <h3 className="text-base font-medium">
                {t("settings.members.inactive-users")}
              </h3>
              <SearchInput
                placeholder={t("common.filter-by-name")}
                value={inactiveUserSearchText}
                onChange={(e) => setInactiveUserSearchText(e.target.value)}
              />
            </div>

            {inactiveUsers.isLoading && inactiveUsers.dataList.length === 0 ? (
              <div className="flex items-center justify-center h-32">
                <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
              </div>
            ) : (
              <>
                <UserTable
                  users={inactiveUsers.dataList}
                  onUserUpdated={handleInactiveUserUpdated}
                  onUserEdit={(user) => setEditingUser(user)}
                  onGroupSelected={handleGroupSelected}
                />
                <PagedTableFooter
                  pageSize={inactiveUsers.pageSize}
                  pageSizeOptions={inactiveUsers.pageSizeOptions}
                  onPageSizeChange={inactiveUsers.onPageSizeChange}
                  hasMore={inactiveUsers.hasMore}
                  isFetchingMore={inactiveUsers.isFetchingMore}
                  onLoadMore={inactiveUsers.loadMore}
                />
              </>
            )}
          </div>
        )}
      </div>

      <UserFormSheet
        open={showCreateUserDrawer}
        onClose={() => setShowCreateUserDrawer(false)}
        onCreated={(user) => {
          activeUsers.updateCache([user]);
        }}
      />

      <EditUserSheet
        open={!!editingUser}
        user={editingUser}
        onClose={() => setEditingUser(undefined)}
        onUpdated={(user) => {
          // updateCache appends when the name is absent, so writing to both
          // lists would file an active user into the inactive one as well.
          if (user.state === State.DELETED) {
            inactiveUsers.updateCache([user]);
          } else {
            activeUsers.updateCache([user]);
          }
        }}
      />

      <AADSyncSheet
        open={showAadSyncDrawer}
        onClose={() => setShowAadSyncDrawer(false)}
      />
    </WorkspacePageLayout>
  );
}
