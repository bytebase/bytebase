import { Eye, Pencil, Plus, Search, Settings, Trash2, Undo2 } from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/react/components/ui/alert";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useGroupStore,
  useServiceAccountStore,
  useSubscriptionV1Store,
  useUserStore,
  useWorkloadIdentityStore,
} from "@/store";
import { getUserFullNameByType } from "@/store/modules/v1/common";
import { AccountType, getAccountTypeByEmail } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { getDefaultPagination } from "@/utils/pagination";
import { hasWorkspacePermissionV2 } from "@/utils";

// ============================================================
// usePagedData hook
// ============================================================

function usePagedData<T extends { name: string }>({
  sessionKey,
  fetchList,
  debounce = 500,
}: {
  sessionKey: string;
  fetchList: (params: {
    pageSize: number;
    pageToken: string;
  }) => Promise<{ list: T[]; nextPageToken?: string }>;
  debounce?: number;
}) {
  const pageSizeOptions = [getDefaultPagination(), 50, 100, 200, 500].filter(
    (v, i, a) => a.indexOf(v) === i
  );

  const getStoredPageSize = () => {
    try {
      const stored = localStorage.getItem(sessionKey);
      if (stored) {
        const n = parseInt(stored, 10);
        if (n > 0) return n;
      }
    } catch {
      // ignore
    }
    return pageSizeOptions[0];
  };

  const [pageSize, setPageSize] = useState(getStoredPageSize);
  const [dataList, setDataList] = useState<T[]>([]);
  const [nextPageToken, setNextPageToken] = useState<string | undefined>(
    undefined
  );
  const [isLoading, setIsLoading] = useState(true);
  const [hasFetched, setHasFetched] = useState(false);

  const abortRef = useRef<AbortController | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const hasMore = !!nextPageToken;

  const doFetch = useCallback(
    async (token: string, append: boolean) => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      setIsLoading(true);
      try {
        const result = await fetchList({ pageSize, pageToken: token });
        if (controller.signal.aborted) return;
        setDataList((prev) => (append ? [...prev, ...result.list] : result.list));
        setNextPageToken(result.nextPageToken || undefined);
        setHasFetched(true);
      } finally {
        if (!controller.signal.aborted) {
          setIsLoading(false);
        }
      }
    },
    [fetchList, pageSize]
  );

  const loadMore = useCallback(() => {
    if (nextPageToken) {
      doFetch(nextPageToken, true);
    }
  }, [nextPageToken, doFetch]);

  const refresh = useCallback(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      setDataList([]);
      setNextPageToken(undefined);
      doFetch("", false);
    }, debounce);
  }, [doFetch, debounce]);

  const updateCache = useCallback((items: T[]) => {
    setDataList((prev) => {
      const next = [...prev];
      for (const item of items) {
        const idx = next.findIndex((d) => d.name === item.name);
        if (idx >= 0) {
          next[idx] = item;
        } else {
          next.push(item);
        }
      }
      return next;
    });
  }, []);

  const removeCache = useCallback((item: T) => {
    setDataList((prev) => prev.filter((d) => d.name !== item.name));
  }, []);

  const onPageSizeChange = useCallback(
    (size: number) => {
      setPageSize(size);
      localStorage.setItem(sessionKey, String(size));
    },
    [sessionKey]
  );

  // Initial fetch
  useEffect(() => {
    doFetch("", false);
    return () => {
      abortRef.current?.abort();
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [doFetch]);

  return {
    dataList,
    isLoading: isLoading || !hasFetched,
    hasMore,
    loadMore,
    refresh,
    updateCache,
    removeCache,
    pageSize,
    pageSizeOptions,
    onPageSizeChange,
  };
}

// ============================================================
// PagedTableFooter
// ============================================================

function PagedTableFooter({
  pageSize,
  pageSizeOptions,
  onPageSizeChange,
  hasMore,
  isLoading,
  onLoadMore,
}: {
  pageSize: number;
  pageSizeOptions: number[];
  onPageSizeChange: (size: number) => void;
  hasMore: boolean;
  isLoading: boolean;
  onLoadMore: () => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-between px-4 py-2 border-t text-sm">
      <div className="flex items-center gap-x-2">
        <span className="text-control-light">
          {t("common.rows-per-page")}
        </span>
        <select
          value={pageSize}
          onChange={(e) => onPageSizeChange(Number(e.target.value))}
          className="h-8 rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm"
        >
          {pageSizeOptions.map((opt) => (
            <option key={opt} value={opt}>
              {opt}
            </option>
          ))}
        </select>
      </div>
      {hasMore && (
        <Button
          variant="outline"
          size="sm"
          disabled={isLoading}
          onClick={onLoadMore}
        >
          {t("common.load-more")}
        </Button>
      )}
    </div>
  );
}

// ============================================================
// UserTable
// ============================================================

function UserTable({
  users,
  onUserUpdated,
  onUserRemoved,
}: {
  users: User[];
  onUserUpdated: (user: User) => void;
  onUserRemoved: (user: User) => void;
}) {
  const { t } = useTranslation();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const groupStore = useGroupStore();
  const userStore = useUserStore();
  const serviceAccountStore = useServiceAccountStore();
  const workloadIdentityStore = useWorkloadIdentityStore();

  // Batch fetch groups when user list changes
  useEffect(() => {
    const allGroupNames = users.flatMap((u) => u.groups);
    if (allGroupNames.length > 0) {
      groupStore.batchGetOrFetchGroups(allGroupNames);
    }
  }, [users, groupStore]);

  const handleDeactivate = async (user: User) => {
    const accountType = getAccountTypeByEmail(user.email);
    const fullName = getUserFullNameByType(user);

    const confirmed = window.confirm(
      t("settings.members.deactivate-confirm", { name: user.title })
    );
    if (!confirmed) return;

    try {
      if (accountType === AccountType.SERVICE_ACCOUNT) {
        await serviceAccountStore.deleteServiceAccount(fullName);
      } else if (accountType === AccountType.WORKLOAD_IDENTITY) {
        await workloadIdentityStore.deleteWorkloadIdentity(fullName);
      } else {
        await userStore.archiveUser(fullName);
      }

      const updated = { ...user, state: State.DELETED } as User;
      onUserUpdated(updated);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("settings.members.deactivated"),
      });
    } catch {
      // error already shown by store
    }
  };

  const handleRestore = async (user: User) => {
    const accountType = getAccountTypeByEmail(user.email);
    const fullName = getUserFullNameByType(user);

    try {
      if (accountType === AccountType.SERVICE_ACCOUNT) {
        await serviceAccountStore.undeleteServiceAccount(fullName);
      } else if (accountType === AccountType.WORKLOAD_IDENTITY) {
        await workloadIdentityStore.undeleteWorkloadIdentity(fullName);
      } else {
        await userStore.restoreUser(fullName);
      }

      const updated = { ...user, state: State.ACTIVE } as User;
      onUserRemoved(updated);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("settings.members.restored"),
      });
    } catch {
      // error already shown by store
    }
  };

  const handleView = (user: User) => {
    window.location.href = `/users/${user.email}`;
  };

  const getDeletePermission = (accountType: AccountType) => {
    switch (accountType) {
      case AccountType.SERVICE_ACCOUNT:
        return "bb.serviceAccounts.delete";
      case AccountType.WORKLOAD_IDENTITY:
        return "bb.workloadIdentities.delete";
      default:
        return "bb.users.delete";
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

  const getUndeletePermission = (accountType: AccountType) => {
    switch (accountType) {
      case AccountType.SERVICE_ACCOUNT:
        return "bb.serviceAccounts.undelete";
      case AccountType.WORKLOAD_IDENTITY:
        return "bb.workloadIdentities.undelete";
      default:
        return "bb.users.undelete";
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
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-control-bg">
            <th className="px-4 py-2 text-left font-medium whitespace-nowrap">
              {t("settings.members.table.account")}
            </th>
            <th className="px-4 py-2 text-left font-medium whitespace-nowrap">
              {t("settings.members.table.groups")}
            </th>
            <th className="px-4 py-2 text-right font-medium whitespace-nowrap">
              {t("common.operations")}
            </th>
          </tr>
        </thead>
        <tbody>
          {users.map((user, i) => {
            const accountType = getAccountTypeByEmail(user.email);
            const isDeleted = user.state === State.DELETED;
            const isSelf = currentUser.name === user.name;

            return (
              <tr
                key={user.name}
                className={`border-b last:border-b-0 ${i % 2 === 1 ? "bg-gray-50" : ""}`}
              >
                {/* Account column */}
                <td className="px-4 py-2">
                  <div className="flex items-center gap-x-2">
                    <div className="flex flex-col">
                      <div className="flex items-center gap-x-1.5">
                        <span
                          className={
                            isDeleted ? "line-through text-control-light" : ""
                          }
                        >
                          {user.title}
                        </span>
                        {isSelf && (
                          <Badge variant="secondary" className="text-xs px-1.5 py-0">
                            {t("common.you")}
                          </Badge>
                        )}
                        {accountType !== AccountType.USER && (
                          <Badge className="text-xs px-1.5 py-0">
                            {getAccountTypeLabel(accountType)}
                          </Badge>
                        )}
                        {user.mfaEnabled && (
                          <Badge variant="success" className="text-xs px-1.5 py-0">
                            MFA
                          </Badge>
                        )}
                        {user.profile?.source && (
                          <Badge className="text-xs px-1.5 py-0">
                            {user.profile.source}
                          </Badge>
                        )}
                      </div>
                      <span className="textinfolabel text-xs">
                        {user.email}
                      </span>
                    </div>
                  </div>
                </td>

                {/* Groups column */}
                <td className="px-4 py-2">
                  <UserGroupsCell user={user} />
                </td>

                {/* Operations column */}
                <td className="px-4 py-2">
                  <div className="flex justify-end gap-x-1">
                    {!isDeleted && (
                      <>
                        {hasWorkspacePermissionV2(
                          getDeletePermission(accountType)
                        ) &&
                          !isSelf && (
                            <Tooltip content={t("settings.members.deactivate")}>
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
                        {hasWorkspacePermissionV2(
                          getViewPermission(accountType)
                        ) && (
                          <Tooltip
                            content={
                              accountType === AccountType.USER
                                ? t("common.view")
                                : t("common.edit")
                            }
                          >
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7"
                              onClick={() => handleView(user)}
                            >
                              {accountType === AccountType.USER ? (
                                <Eye className="h-4 w-4" />
                              ) : (
                                <Pencil className="h-4 w-4" />
                              )}
                            </Button>
                          </Tooltip>
                        )}
                      </>
                    )}
                    {isDeleted &&
                      hasWorkspacePermissionV2(
                        getUndeletePermission(accountType)
                      ) && (
                        <Tooltip content={t("settings.members.restore")}>
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
          })}
        </tbody>
      </table>
    </div>
  );
}

// ============================================================
// UserGroupsCell
// ============================================================

function UserGroupsCell({ user }: { user: User }) {
  const groupStore = useGroupStore();

  if (!user.groups || user.groups.length === 0) {
    return <span className="text-control-light">-</span>;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {user.groups.map((groupName) => {
        const group = groupStore.getGroupByIdentifier(groupName);
        return (
          <Badge key={groupName} variant="secondary" className="text-xs px-1.5 py-0 cursor-pointer">
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

type TabValue = "USERS" | "MEMBERS" | "GROUPS";

function getInitialTab(isSaaSMode: boolean): TabValue {
  const hash = window.location.hash.replace("#", "").toUpperCase();
  if (hash === "USERS" && !isSaaSMode) return "USERS";
  if (hash === "MEMBERS" && isSaaSMode) return "MEMBERS";
  if (hash === "GROUPS") return "GROUPS";
  return isSaaSMode ? "MEMBERS" : "USERS";
}

export function UsersPage() {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const userStore = useUserStore();

  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const activeUserCount = useVueState(() => actuatorStore.activeUserCount);
  const userCountLimit = useVueState(() => subscriptionStore.userCountLimit);

  const [tab, setTab] = useState<TabValue>(() => getInitialTab(isSaaSMode));
  const [userSearchText, setUserSearchText] = useState("");
  const [groupSearchText, setGroupSearchText] = useState("");
  const [showInactiveUsers, setShowInactiveUsers] = useState(false);
  const [inactiveUserSearchText, setInactiveUserSearchText] = useState("");

  // Drawer visibility
  const [showCreateUserDrawer, setShowCreateUserDrawer] = useState(false);
  const [_showCreateGroupDrawer, _setShowCreateGroupDrawer] = useState(false);
  const [_showAadSyncDrawer, _setShowAadSyncDrawer] = useState(false);

  // Suppress unused warnings for drawers not yet implemented
  void showCreateUserDrawer;

  const remainingUserCount = useMemo(
    () => Math.max(0, userCountLimit - activeUserCount),
    [userCountLimit, activeUserCount]
  );

  // Active users paged data
  const fetchActiveUsers = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const result = await userStore.fetchUserList({
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        filter: { query: userSearchText },
      });
      return { list: result.users, nextPageToken: result.nextPageToken };
    },
    [userStore, userSearchText]
  );

  const activeUsers = usePagedData<User>({
    sessionKey: "bb.users.active.page-size",
    fetchList: fetchActiveUsers,
  });

  // Refresh active users when search text changes
  useEffect(() => {
    activeUsers.refresh();
  }, [userSearchText]); // eslint-disable-line react-hooks/exhaustive-deps

  // Inactive users paged data
  const fetchInactiveUsers = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const result = await userStore.fetchUserList({
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        filter: { query: inactiveUserSearchText, state: State.DELETED },
        showDeleted: true,
      });
      return { list: result.users, nextPageToken: result.nextPageToken };
    },
    [userStore, inactiveUserSearchText]
  );

  const inactiveUsers = usePagedData<User>({
    sessionKey: "bb.users.inactive.page-size",
    fetchList: fetchInactiveUsers,
  });

  // Refresh inactive users when search text changes
  useEffect(() => {
    if (showInactiveUsers) {
      inactiveUsers.refresh();
    }
  }, [inactiveUserSearchText, showInactiveUsers]); // eslint-disable-line react-hooks/exhaustive-deps

  // Sync tab to URL hash
  useEffect(() => {
    window.location.hash = tab;
  }, [tab]);

  const handleTabChange = (value: TabValue) => {
    if (value) setTab(value);
  };

  const handleActiveUserUpdated = (user: User) => {
    activeUsers.updateCache([user]);
  };

  const handleActiveUserRemoved = (user: User) => {
    activeUsers.removeCache(user);
  };

  const handleInactiveUserUpdated = (user: User) => {
    inactiveUsers.updateCache([user]);
  };

  const handleInactiveUserRemoved = (user: User) => {
    inactiveUsers.removeCache(user);
  };

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {!isSaaSMode && remainingUserCount <= 3 && (
        <Alert variant="warning" className="mb-2">
          <AlertTitle>
            {t("subscription.usage.user-count.title")}
          </AlertTitle>
          <AlertDescription>
            {t("subscription.usage.user-count.description", {
              total: userCountLimit,
              remaining: remainingUserCount,
            })}{" "}
            {t("subscription.usage.upgrade-prompt")}
          </AlertDescription>
        </Alert>
      )}

      <Tabs
        value={tab}
        onValueChange={(val) => handleTabChange(val as TabValue)}
      >
        <div className="flex items-center justify-between">
          <TabsList>
            {!isSaaSMode && (
              <TabsTrigger value="USERS">
                {t("common.users")}
                <span className="ml-1 font-normal text-control-light">
                  ({activeUserCount})
                </span>
              </TabsTrigger>
            )}
            {isSaaSMode && (
              <TabsTrigger value="MEMBERS">
                {t("common.members", { count: 2 })}
              </TabsTrigger>
            )}
            <TabsTrigger value="GROUPS">
              {t("settings.members.groups.self")}
            </TabsTrigger>
          </TabsList>

          <div className="flex items-center gap-x-2">
            {tab === "USERS" && (
              <>
                <div className="relative">
                  <Input
                    placeholder={t("common.filter-by-name")}
                    value={userSearchText}
                    onChange={(e) => setUserSearchText(e.target.value)}
                    className="h-8 text-sm pr-8"
                  />
                  <Search className="absolute right-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
                </div>
                <Button variant="outline">
                  <Settings className="h-4 w-4 mr-1" />
                  <FeatureBadge
                    feature={PlanFeature.FEATURE_DIRECTORY_SYNC}
                    clickable={false}
                  />
                  {t("settings.members.entra-sync.self")}
                </Button>
                <Button onClick={() => setShowCreateUserDrawer(true)}>
                  <Plus className="h-4 w-4 mr-1" />
                  {t("settings.members.add-user")}
                </Button>
              </>
            )}
            {tab === "GROUPS" && (
              <>
                <div className="relative">
                  <Input
                    placeholder={t("common.filter-by-name")}
                    value={groupSearchText}
                    onChange={(e) => setGroupSearchText(e.target.value)}
                    className="h-8 text-sm pr-8"
                  />
                  <Search className="absolute right-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
                </div>
                <Button variant="outline">
                  <Settings className="h-4 w-4 mr-1" />
                  <FeatureBadge
                    feature={PlanFeature.FEATURE_DIRECTORY_SYNC}
                    clickable={false}
                  />
                  {t("settings.members.entra-sync.self")}
                </Button>
                <Button>
                  <Plus className="h-4 w-4 mr-1" />
                  {t("settings.members.groups.add-group")}
                </Button>
              </>
            )}
            {tab === "MEMBERS" && (
              <>{/* Placeholder for members tab actions */}</>
            )}
          </div>
        </div>

        {!isSaaSMode && (
          <TabsPanel value="USERS">
            <div className="py-4 flex flex-col gap-y-4">
              {activeUsers.isLoading && activeUsers.dataList.length === 0 ? (
                <div className="flex items-center justify-center h-32">
                  <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
                </div>
              ) : (
                <>
                  <UserTable
                    users={activeUsers.dataList}
                    onUserUpdated={handleActiveUserUpdated}
                    onUserRemoved={handleActiveUserRemoved}
                  />
                  <PagedTableFooter
                    pageSize={activeUsers.pageSize}
                    pageSizeOptions={activeUsers.pageSizeOptions}
                    onPageSizeChange={activeUsers.onPageSizeChange}
                    hasMore={activeUsers.hasMore}
                    isLoading={activeUsers.isLoading}
                    onLoadMore={activeUsers.loadMore}
                  />
                </>
              )}

              {/* Inactive users toggle */}
              <label className="flex items-center gap-x-2 text-sm cursor-pointer">
                <input
                  type="checkbox"
                  checked={showInactiveUsers}
                  onChange={(e) => setShowInactiveUsers(e.target.checked)}
                />
                {t("settings.members.show-inactive")}
              </label>

              {showInactiveUsers && (
                <div className="flex flex-col gap-y-3">
                  <div className="flex items-center justify-between">
                    <h3 className="text-base font-medium">
                      {t("settings.members.inactive")}
                    </h3>
                    <div className="relative">
                      <Input
                        placeholder={t("common.filter-by-name")}
                        value={inactiveUserSearchText}
                        onChange={(e) =>
                          setInactiveUserSearchText(e.target.value)
                        }
                        className="h-8 text-sm pr-8"
                      />
                      <Search className="absolute right-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
                    </div>
                  </div>

                  {inactiveUsers.isLoading &&
                  inactiveUsers.dataList.length === 0 ? (
                    <div className="flex items-center justify-center h-32">
                      <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
                    </div>
                  ) : (
                    <>
                      <UserTable
                        users={inactiveUsers.dataList}
                        onUserUpdated={handleInactiveUserUpdated}
                        onUserRemoved={handleInactiveUserRemoved}
                      />
                      <PagedTableFooter
                        pageSize={inactiveUsers.pageSize}
                        pageSizeOptions={inactiveUsers.pageSizeOptions}
                        onPageSizeChange={inactiveUsers.onPageSizeChange}
                        hasMore={inactiveUsers.hasMore}
                        isLoading={inactiveUsers.isLoading}
                        onLoadMore={inactiveUsers.loadMore}
                      />
                    </>
                  )}
                </div>
              )}
            </div>
          </TabsPanel>
        )}
        {isSaaSMode && (
          <TabsPanel value="MEMBERS">
            <div className="py-4 text-control-light">
              Members content here
            </div>
          </TabsPanel>
        )}
        <TabsPanel value="GROUPS">
          <div className="py-4 text-control-light">
            Groups content here
          </div>
        </TabsPanel>
      </Tabs>
    </div>
  );
}
