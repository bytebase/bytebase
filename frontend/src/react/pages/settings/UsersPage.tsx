import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import {
  Building2,
  Check,
  ChevronDown,
  ChevronRight,
  CircleAlert,
  CircleCheck,
  Copy,
  Eye,
  EyeOff,
  Pencil,
  Plus,
  Search,
  Settings,
  Trash2,
  Undo2,
  Users,
  X,
} from "lucide-react";
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import type { MemberBinding } from "@/components/Member/types";
import { getMemberBindings } from "@/components/Member/utils";
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
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
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useGroupStore,
  useRoleStore,
  useServiceAccountStore,
  useSettingV1Store,
  useSubscriptionV1Store,
  useUserStore,
  useWorkloadIdentityStore,
  useWorkspaceV1Store,
} from "@/store";
import {
  extractUserEmail,
  getUserFullNameByType,
  groupNamePrefix,
} from "@/store/modules/v1/common";
import {
  AccountType,
  ALL_USERS_USER_EMAIL,
  getAccountTypeByEmail,
  getUserEmailInBinding,
  userBindingPrefix,
} from "@/types";
import {
  PRESET_PROJECT_ROLES,
  PRESET_ROLES,
  PRESET_WORKSPACE_ROLES,
  PresetRoleType,
} from "@/types/iam";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Group, GroupMember } from "@/types/proto-es/v1/group_service_pb";
import {
  GroupMember_Role,
  GroupMemberSchema,
  GroupSchema,
} from "@/types/proto-es/v1/group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  UpdateUserRequestSchema,
  UserSchema,
} from "@/types/proto-es/v1/user_service_pb";
import {
  displayRoleDescription,
  displayRoleTitle,
  hasWorkspacePermissionV2,
  isValidEmail,
  sortRoles,
} from "@/utils";

// ============================================================
// usePagedData hook
// ============================================================

function usePagedData<T extends { name: string }>({
  sessionKey,
  fetchList,
  enabled = true,
}: {
  sessionKey: string;
  fetchList: (params: {
    pageSize: number;
    pageToken: string;
  }) => Promise<{ list: T[]; nextPageToken?: string }>;
  enabled?: boolean;
}) {
  const [pageSize, setPageSize] = useSessionPageSize(sessionKey);
  const pageSizeOptions = getPageSizeOptions();

  const [dataList, setDataList] = useState<T[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [hasFetched, setHasFetched] = useState(false);
  const [hasMore, setHasMore] = useState(false);

  const abortRef = useRef<AbortController | null>(null);
  const fetchIdRef = useRef(0);
  const nextPageTokenRef = useRef("");

  const doFetch = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      if (isRefresh) {
        setIsLoading(true);
      } else {
        setIsFetchingMore(true);
      }

      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await fetchList({ pageSize, pageToken: token });
        if (controller.signal.aborted || currentFetchId !== fetchIdRef.current)
          return;
        setDataList((prev) =>
          isRefresh ? result.list : [...prev, ...result.list]
        );
        nextPageTokenRef.current = result.nextPageToken ?? "";
        setHasMore(Boolean(result.nextPageToken));
        setHasFetched(true);
      } catch (e) {
        if (e instanceof Error && e.name === "AbortError") return;
        console.error(e);
      } finally {
        if (currentFetchId === fetchIdRef.current) {
          setIsLoading(false);
          setIsFetchingMore(false);
        }
      }
    },
    [fetchList, pageSize]
  );

  const loadMore = useCallback(() => {
    if (hasMore && !isFetchingMore) {
      doFetch(false);
    }
  }, [hasMore, isFetchingMore, doFetch]);

  const refresh = useCallback(() => {
    doFetch(true);
  }, [doFetch]);

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

  // Fetch on mount and when fetchList/pageSize changes (handles search text reactivity)
  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (!enabled) return;
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      doFetch(true);
      return;
    }
    const timer = setTimeout(() => doFetch(true), 300);
    return () => clearTimeout(timer);
  }, [doFetch, enabled]);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  return {
    dataList,
    isLoading: isLoading || !hasFetched,
    isFetchingMore,
    hasMore,
    loadMore,
    refresh,
    updateCache,
    removeCache,
    pageSize,
    pageSizeOptions,
    onPageSizeChange: setPageSize,
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
  isFetchingMore,
  onLoadMore,
}: {
  pageSize: number;
  pageSizeOptions: number[];
  onPageSizeChange: (size: number) => void;
  hasMore: boolean;
  isFetchingMore: boolean;
  onLoadMore: () => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-end gap-x-2">
      <div className="flex items-center gap-x-2">
        <span className="text-sm text-control-light">
          {t("common.rows-per-page")}
        </span>
        <select
          value={pageSize}
          onChange={(e) => onPageSizeChange(Number(e.target.value))}
          className="border border-control-border rounded-sm text-sm pl-2 pr-6 py-1 min-w-[5rem]"
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
          variant="ghost"
          size="sm"
          disabled={isFetchingMore}
          onClick={onLoadMore}
        >
          <span className="text-sm text-control-light">
            {isFetchingMore ? t("common.loading") : t("common.load-more")}
          </span>
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
  onUserSelected,
  onGroupSelected,
}: {
  users: User[];
  onUserUpdated: (user: User) => void;
  onUserSelected?: (user: User) => void;
  onGroupSelected?: (group: Group) => void;
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
      t("settings.members.action.deactivate-confirm-title")
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

      const updated = create(UserSchema, { ...user, state: State.DELETED });
      onUserUpdated(updated);

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

      const updated = create(UserSchema, { ...user, state: State.ACTIVE });
      onUserUpdated(updated);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    }
  };

  const handleView = (user: User) => {
    const accountType = getAccountTypeByEmail(user.email);
    if (accountType === AccountType.USER && onUserSelected) {
      onUserSelected(user);
    } else {
      router.push({
        name: WORKSPACE_ROUTE_USER_PROFILE,
        params: { principalEmail: user.email },
      });
    }
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
                      </div>
                      <span className="textinfolabel text-xs">
                        {user.email}
                      </span>
                    </div>
                  </div>
                </td>

                {/* Groups column */}
                <td className="px-4 py-2">
                  <UserGroupsCell
                    user={user}
                    onGroupSelected={onGroupSelected}
                  />
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
          })}
        </tbody>
      </table>
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
  const groupStore = useGroupStore();

  if (!user.groups || user.groups.length === 0) {
    return <span className="text-control-light">-</span>;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {user.groups.map((groupName) => {
        const group = groupStore.getGroupByIdentifier(groupName);
        return (
          <Badge
            key={groupName}
            variant="secondary"
            className="text-xs px-1.5 py-0 cursor-pointer"
            onClick={() => {
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
// Highlight helper
// ============================================================

function HighlightText({ text, keyword }: { text: string; keyword: string }) {
  if (!keyword.trim()) return <>{text}</>;
  const idx = text.toLowerCase().indexOf(keyword.toLowerCase());
  if (idx < 0) return <>{text}</>;
  return (
    <>
      {text.slice(0, idx)}
      <span className="bg-yellow-100">
        {text.slice(idx, idx + keyword.length)}
      </span>
      {text.slice(idx + keyword.length)}
    </>
  );
}

// ============================================================
// GroupTable
// ============================================================

function GroupTable({
  groups,
  searchText,
  onGroupSelected,
  onGroupDeleted,
}: {
  groups: Group[];
  searchText: string;
  onGroupSelected: (group: Group) => void;
  onGroupDeleted: (group: Group) => void;
}) {
  const { t } = useTranslation();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const groupStore = useGroupStore();
  const userStore = useUserStore();

  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [memberCache, setMemberCache] = useState<Map<string, User[]>>(
    new Map()
  );
  const memberCacheRef = useRef(memberCache);
  memberCacheRef.current = memberCache;
  const loadingRef = useRef<Set<string>>(new Set());

  const toggleExpand = useCallback(
    (group: Group) => {
      setExpandedGroups((prev) => {
        const next = new Set(prev);
        if (next.has(group.name)) {
          next.delete(group.name);
        } else {
          next.add(group.name);
        }
        return next;
      });

      // Fetch members if not cached (use ref to avoid recreating callback on cache updates)
      if (
        !memberCacheRef.current.has(group.name) &&
        !loadingRef.current.has(group.name)
      ) {
        loadingRef.current.add(group.name);
        const memberNames = group.members.map((m) => m.member);
        userStore.batchGetOrFetchUsers(memberNames).then((users) => {
          loadingRef.current.delete(group.name);
          setMemberCache((prev) => {
            const next = new Map(prev);
            next.set(
              group.name,
              users.filter((u): u is User => !!u)
            );
            return next;
          });
        });
      }
    },
    [userStore]
  );

  const isGroupOwner = useCallback(
    (group: Group) => {
      return (
        group.members.find(
          (m) => extractUserEmail(m.member) === currentUser.email
        )?.role === GroupMember_Role.OWNER
      );
    },
    [currentUser.email]
  );

  const handleDelete = useCallback(
    async (group: Group) => {
      const confirmed = window.confirm(
        t("settings.members.action.deactivate-confirm-title")
      );
      if (!confirmed) return;

      try {
        await groupStore.deleteGroup(group.name);
        onGroupDeleted(group);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.deleted"),
        });
      } catch {
        // error shown by store
      }
    },
    [groupStore, onGroupDeleted, t]
  );

  if (groups.length === 0) {
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
              {t("common.groups")} / {t("common.users")}
            </th>
            <th className="px-4 py-2 text-right font-medium whitespace-nowrap w-16" />
          </tr>
        </thead>
        <tbody>
          {groups.map((group, i) => {
            const isExpanded = expandedGroups.has(group.name);
            const members = memberCache.get(group.name);
            const canEdit =
              isGroupOwner(group) ||
              hasWorkspacePermissionV2("bb.groups.update");
            const canDelete =
              isGroupOwner(group) ||
              hasWorkspacePermissionV2("bb.groups.delete");

            return (
              <GroupRow
                key={group.name}
                group={group}
                index={i}
                isExpanded={isExpanded}
                members={members}
                searchText={searchText}
                canEdit={canEdit}
                canDelete={canDelete}
                onToggle={() => toggleExpand(group)}
                onEdit={() => onGroupSelected(group)}
                onDelete={() => handleDelete(group)}
              />
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function GroupRow({
  group,
  index,
  isExpanded,
  members,
  searchText,
  canEdit,
  canDelete,
  onToggle,
  onEdit,
  onDelete,
}: {
  group: Group;
  index: number;
  isExpanded: boolean;
  members: User[] | undefined;
  searchText: string;
  canEdit: boolean;
  canDelete: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const { t } = useTranslation();
  const stripeBg = index % 2 === 1 ? "bg-gray-50" : "";

  return (
    <>
      <tr className={`border-b last:border-b-0 ${stripeBg}`}>
        <td className="px-4 py-2">
          <div className="flex items-center gap-x-2">
            <button
              className="shrink-0 p-0.5 rounded hover:bg-gray-200"
              onClick={onToggle}
            >
              {isExpanded ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
            </button>
            <Users className="h-4 w-4 shrink-0 text-control-light" />
            <div className="flex flex-col">
              <div className="flex items-center gap-x-1.5">
                <span>
                  <HighlightText text={group.title} keyword={searchText} />
                </span>
                <span className="text-control-light text-xs">
                  (
                  {t("settings.members.groups.n-members", {
                    n: group.members.length,
                  })}
                  )
                </span>
                {group.source && (
                  <Badge className="text-xs px-1.5 py-0">{group.source}</Badge>
                )}
              </div>
              <span className="textinfolabel text-xs">
                <HighlightText text={group.name} keyword={searchText} />
              </span>
            </div>
          </div>
        </td>
        <td className="px-4 py-2">
          <div className="flex justify-end gap-x-1">
            {canEdit && (
              <Tooltip content={t("common.edit")}>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={onEdit}
                >
                  <Pencil className="h-4 w-4" />
                </Button>
              </Tooltip>
            )}
            {canDelete && (
              <Tooltip content={t("common.delete")}>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 text-error hover:text-error"
                  onClick={onDelete}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </Tooltip>
            )}
          </div>
        </td>
      </tr>
      {isExpanded &&
        members &&
        members.map((user) => {
          const memberInfo = group.members.find(
            (m) => extractUserEmail(m.member) === user.email
          );
          const isOwner = memberInfo?.role === GroupMember_Role.OWNER;

          return (
            <tr
              key={user.name}
              className={`border-b last:border-b-0 ${stripeBg}`}
            >
              <td className="px-4 py-2 pl-14">
                <div className="flex items-center gap-x-2">
                  <span>{user.title}</span>
                  <span className="textinfolabel text-xs">{user.email}</span>
                  {isOwner ? (
                    <Badge variant="secondary" className="text-xs px-1.5 py-0">
                      {t("settings.members.groups.form.role.owner")}
                    </Badge>
                  ) : (
                    <Badge variant="default" className="text-xs px-1.5 py-0">
                      {t("settings.members.groups.form.role.member")}
                    </Badge>
                  )}
                </div>
              </td>
              <td />
            </tr>
          );
        })}
      {isExpanded && !members && (
        <tr className={stripeBg}>
          <td colSpan={2} className="px-4 py-2 pl-14">
            <div className="flex items-center gap-x-2 text-control-light text-sm">
              <div className="animate-spin h-4 w-4 border-2 border-accent border-t-transparent rounded-full" />
              {t("common.loading")}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

// ============================================================
// RoleMultiSelect
// ============================================================

function RoleMultiSelect({
  value,
  onChange,
  disabled,
}: {
  value: string[];
  onChange: (roles: string[]) => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const roleStore = useRoleStore();
  const roleList = useVueState(() => [...roleStore.roleList]);
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleClickOutside = useCallback(() => {
    setOpen(false);
    setSearch("");
  }, []);
  useClickOutside(containerRef, open, handleClickOutside);

  const groups = useMemo(() => {
    const kw = search.toLowerCase();
    const matchRole = (name: string) =>
      !kw || displayRoleTitle(name).toLowerCase().includes(kw);

    const workspace = PRESET_WORKSPACE_ROLES.filter(matchRole);
    const project = PRESET_PROJECT_ROLES.filter(matchRole);
    const custom = roleList
      .filter((r) => !PRESET_ROLES.includes(r.name))
      .map((r) => r.name)
      .filter(matchRole);

    const result: { label: string; roles: string[] }[] = [];
    if (workspace.length > 0)
      result.push({
        label: t("role.workspace-roles.self"),
        roles: workspace,
      });
    if (project.length > 0)
      result.push({ label: t("role.project-roles.self"), roles: project });
    if (custom.length > 0)
      result.push({ label: t("role.custom-roles.self"), roles: custom });
    return result;
  }, [roleList, search, t]);

  const toggle = (roleName: string) => {
    if (disabled) return;
    if (value.includes(roleName)) {
      onChange(value.filter((r) => r !== roleName));
    } else {
      onChange([...value, roleName]);
    }
  };

  const remove = (roleName: string) => {
    if (disabled) return;
    onChange(value.filter((r) => r !== roleName));
  };

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger */}
      <div
        className={cn(
          "flex flex-wrap items-center gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm cursor-pointer",
          disabled && "opacity-50 cursor-not-allowed",
          open && "ring-2 ring-accent border-accent"
        )}
        onClick={() => {
          if (!disabled) {
            setOpen(!open);
            requestAnimationFrame(() => inputRef.current?.focus());
          }
        }}
      >
        {value.map((roleName) => (
          <span
            key={roleName}
            className="inline-flex items-center gap-x-1 rounded-xs bg-gray-100 px-1.5 py-0.5 text-xs"
          >
            {displayRoleTitle(roleName)}
            {!disabled && (
              <button
                type="button"
                className="hover:text-error"
                onClick={(e) => {
                  e.stopPropagation();
                  remove(roleName);
                }}
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </span>
        ))}
        {open && (
          <input
            ref={inputRef}
            className="flex-1 min-w-[4rem] outline-hidden text-sm bg-transparent"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={
              value.length === 0
                ? t("settings.members.select-role", { count: 2 })
                : ""
            }
          />
        )}
        {!open && value.length === 0 && (
          <span className="text-control-placeholder">
            {t("settings.members.select-role", { count: 2 })}
          </span>
        )}
        <ChevronDown className="ml-auto h-4 w-4 shrink-0 text-control-light" />
      </div>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1 w-full bg-white border border-control-border rounded-sm shadow-lg max-h-60 overflow-auto">
          {groups.length === 0 && (
            <div className="px-3 py-2 text-sm text-control-light">
              {t("common.no-data")}
            </div>
          )}
          {groups.map((group) => (
            <div key={group.label}>
              <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-gray-50">
                {group.label}
              </div>
              {group.roles.map((roleName) => {
                const selected = value.includes(roleName);
                return (
                  <div
                    key={roleName}
                    className={cn(
                      "flex items-center gap-x-2 px-3 py-1.5 text-sm cursor-pointer hover:bg-gray-50",
                      selected && "bg-accent/5"
                    )}
                    onClick={() => toggle(roleName)}
                  >
                    <div
                      className={cn(
                        "h-4 w-4 rounded-xs border flex items-center justify-center shrink-0",
                        selected
                          ? "bg-accent border-accent text-white"
                          : "border-control-border"
                      )}
                    >
                      {selected && <Check className="h-3 w-3" />}
                    </div>
                    <div className="flex flex-col">
                      <span>{displayRoleTitle(roleName)}</span>
                      {displayRoleDescription(roleName) && (
                        <span className="text-xs text-control-light">
                          {displayRoleDescription(roleName)}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================
// CreateUserDrawer
// ============================================================

function extractUserTitle(email: string): string {
  const atIndex = email.indexOf("@");
  return atIndex !== -1 ? email.substring(0, atIndex) : email;
}

function CreateUserDrawer({
  user,
  onClose,
  onCreated,
  onUpdated,
}: {
  user: User | undefined;
  onClose: () => void;
  onCreated: (user: User) => void;
  onUpdated: (user: User) => void;
}) {
  const { t } = useTranslation();
  const userStore = useUserStore();
  const workspaceStore = useWorkspaceV1Store();
  const settingV1Store = useSettingV1Store();

  const userMapToRoles = useVueState(() => workspaceStore.userMapToRoles);
  const passwordRestriction = useVueState(
    () => settingV1Store.workspaceProfile.passwordRestriction
  );

  const isEditMode =
    !!user && user.name !== "" && !user.name.endsWith("/unknown");

  const allowUpdate =
    !isEditMode || hasWorkspacePermissionV2("bb.users.update");

  const initialRoles = useMemo(() => {
    if (!user || !isEditMode) {
      return [PresetRoleType.WORKSPACE_MEMBER];
    }
    const roles = userMapToRoles.get(getUserFullNameByType(user));
    return roles ? [...roles] : [];
  }, [user, isEditMode, userMapToRoles]);

  const [title, setTitle] = useState(user?.title ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [phone, setPhone] = useState(user?.phone ?? "");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [roles, setRoles] = useState<string[]>(initialRoles);
  const [isRequesting, setIsRequesting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  useEscapeKey(true, onClose);

  // Password validation
  const passwordChecks = useMemo(() => {
    const minLength = passwordRestriction?.minLength ?? 8;
    const checks: { text: string; matched: boolean }[] = [
      {
        text: t("settings.general.workspace.password-restriction.min-length", {
          min: minLength,
        }),
        matched: password.length >= minLength,
      },
    ];
    if (passwordRestriction?.requireNumber) {
      checks.push({
        text: t(
          "settings.general.workspace.password-restriction.require-number"
        ),
        matched: /[0-9]+/.test(password),
      });
    }
    if (passwordRestriction?.requireUppercaseLetter) {
      checks.push({
        text: t(
          "settings.general.workspace.password-restriction.require-uppercase-letter"
        ),
        matched: /[A-Z]+/.test(password),
      });
    } else if (passwordRestriction?.requireLetter) {
      checks.push({
        text: t(
          "settings.general.workspace.password-restriction.require-letter"
        ),
        matched: /[a-zA-Z]+/.test(password),
      });
    }
    if (passwordRestriction?.requireSpecialCharacter) {
      checks.push({
        text: t(
          "settings.general.workspace.password-restriction.require-special-character"
        ),
        matched: /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]+/.test(password),
      });
    }
    return checks;
  }, [password, passwordRestriction, t]);

  const passwordHint =
    password.length > 0 && passwordChecks.some((c) => !c.matched);
  const passwordMismatch = password.length > 0 && password !== passwordConfirm;

  const allowConfirm =
    email.length > 0 &&
    !passwordHint &&
    !passwordMismatch &&
    (isEditMode || password.length > 0);

  const hasPermission = hasWorkspacePermissionV2(
    isEditMode ? "bb.users.update" : "bb.users.create"
  );

  const handleSubmit = async () => {
    if (!allowConfirm || !hasPermission) return;
    setIsRequesting(true);
    try {
      if (isEditMode) {
        await handleUpdate();
      } else {
        await handleCreate();
      }
    } catch {
      // error shown by store
    } finally {
      setIsRequesting(false);
    }
  };

  const handleCreate = async () => {
    const createdUser = await userStore.createUser({
      ...create(UserSchema, {}),
      title: title || extractUserTitle(email),
      email,
      phone,
      password,
    });
    if (roles.length > 0) {
      await workspaceStore.patchIamPolicy([
        {
          member: getUserEmailInBinding(createdUser.email),
          roles,
        },
      ]);
    }
    onCreated(createdUser);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });
    onClose();
  };

  const handleUpdate = async () => {
    if (!user) return;

    const updateMask: string[] = [];
    const payload = create(UserSchema, {
      ...user,
      title,
      phone,
      password,
    });
    if (title !== user.title) updateMask.push("title");
    if (phone !== user.phone) updateMask.push("phone");
    if (password) updateMask.push("password");

    let updatedUser: User = user;

    if (updateMask.length > 0) {
      updatedUser = await userStore.updateUser(
        create(UpdateUserRequestSchema, {
          user: payload,
          updateMask: create(FieldMaskSchema, { paths: updateMask }),
        })
      );
    }

    if (!isEqual([...initialRoles].sort(), [...roles].sort())) {
      await workspaceStore.patchIamPolicy([
        {
          member: getUserEmailInBinding(updatedUser.email),
          roles,
        },
      ]);
    }

    onUpdated(updatedUser);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    onClose();
  };

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 z-40 bg-black/30" onClick={onClose} />

      {/* Drawer */}
      <div
        role="dialog"
        aria-modal="true"
        className="fixed inset-y-0 right-0 z-50 w-[40rem] max-w-[100vw] bg-white shadow-xl flex flex-col"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-medium">
            {isEditMode
              ? t("settings.members.update-user")
              : t("settings.members.add-user")}
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto px-6 py-6">
          <div className="flex flex-col gap-y-6">
            {/* Name */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("common.name")}
              </label>
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder={t("common.name")}
                maxLength={200}
                disabled={!allowUpdate}
              />
            </div>

            {/* Email */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("common.email")}
                <span className="ml-0.5 text-error">*</span>
              </label>
              <Input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={isEditMode}
              />
            </div>

            {/* Roles */}
            {hasWorkspacePermissionV2("bb.workspaces.setIamPolicy") && (
              <div className="flex flex-col gap-y-2">
                <label className="block text-sm font-medium text-control">
                  {t("settings.members.table.roles")}
                </label>
                <RoleMultiSelect
                  value={roles}
                  onChange={setRoles}
                  disabled={false}
                />
              </div>
            )}

            {/* Phone */}
            <div className="flex flex-col gap-y-2">
              <div>
                <label className="block text-sm font-medium text-control">
                  {t("settings.profile.phone")}
                </label>
                <span className="textinfolabel text-sm">
                  {t("settings.profile.phone-tips")}
                </span>
              </div>
              <Input
                type="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                autoComplete="new-password"
                disabled={!allowUpdate}
              />
            </div>

            {/* Password */}
            <div className="flex flex-col gap-y-6">
              <div>
                <label className="block text-sm font-medium text-control">
                  {t("settings.profile.password")}
                  <span className="ml-0.5 text-error">*</span>
                </label>
                <span
                  className={`flex items-center gap-x-1 textinfolabel text-sm ${passwordHint ? "text-error" : ""}`}
                >
                  {t("settings.profile.password-hint")}
                  <Tooltip
                    content={
                      <ul className="list-none text-sm">
                        {passwordChecks.map((check, i) => (
                          <li key={i} className="flex gap-x-1 items-center">
                            {check.matched ? (
                              <CircleCheck className="w-4 text-green-400" />
                            ) : (
                              <CircleAlert className="w-4 text-red-400" />
                            )}
                            {check.text}
                          </li>
                        ))}
                      </ul>
                    }
                  >
                    <CircleAlert className="w-4 cursor-help" />
                  </Tooltip>
                </span>
                <div className="mt-1 relative flex items-center">
                  <Input
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="new-password"
                    placeholder={t("common.sensitive-placeholder")}
                    disabled={!allowUpdate}
                    className={passwordHint ? "border-error" : ""}
                  />
                  <button
                    type="button"
                    className="absolute right-3 cursor-pointer"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? (
                      <Eye className="w-4 h-4" />
                    ) : (
                      <EyeOff className="w-4 h-4" />
                    )}
                  </button>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-control">
                  {t("settings.profile.password-confirm")}
                  <span className="ml-0.5 text-error">*</span>
                </label>
                <div className="mt-1 relative flex items-center">
                  <Input
                    type={showPassword ? "text" : "password"}
                    value={passwordConfirm}
                    onChange={(e) => setPasswordConfirm(e.target.value)}
                    autoComplete="new-password"
                    placeholder={t(
                      "settings.profile.password-confirm-placeholder"
                    )}
                    disabled={!allowUpdate}
                    className={passwordMismatch ? "border-error" : ""}
                  />
                  <button
                    type="button"
                    className="absolute right-3 cursor-pointer"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? (
                      <Eye className="w-4 h-4" />
                    ) : (
                      <EyeOff className="w-4 h-4" />
                    )}
                  </button>
                </div>
                {passwordMismatch && (
                  <span className="text-error text-sm mt-1 pl-1">
                    {t("settings.profile.password-mismatch")}
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-x-2 px-6 py-4 border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={!allowConfirm || !hasPermission || isRequesting}
            onClick={handleSubmit}
          >
            {isEditMode ? t("common.update") : t("common.confirm")}
          </Button>
        </div>
      </div>
    </>
  );
}

// ============================================================
// CreateGroupDrawer
// ============================================================

function deduplicateMembers(members: GroupMember[]): GroupMember[] {
  const map = new Map<string, GroupMember>();
  for (const m of members) {
    const key = m.member;
    const existing = map.get(key);
    if (existing) {
      // Keep the one with OWNER role
      if (m.role === GroupMember_Role.OWNER) {
        map.set(key, m);
      }
    } else {
      map.set(key, m);
    }
  }
  return Array.from(map.values());
}

function CreateGroupDrawer({
  group,
  onClose,
  onUpdated,
  onRemoved,
}: {
  group: Group | undefined;
  onClose: () => void;
  onUpdated: (group: Group) => void;
  onRemoved: (group: Group) => void;
}) {
  const { t } = useTranslation();
  const groupStore = useGroupStore();
  const settingV1Store = useSettingV1Store();
  const currentUser = useVueState(() => useCurrentUserV1().value);

  const isEditMode = !!group;
  const workspaceDomains = useVueState(
    () => settingV1Store.workspaceProfile.domains
  );
  const domainSuffix =
    workspaceDomains.length > 0 ? `@${workspaceDomains[0]}` : "";

  const [email, setEmail] = useState(group?.email ?? "");
  const [title, setTitle] = useState(group?.title ?? "");
  const [description, setDescription] = useState(group?.description ?? "");
  const [members, setMembers] = useState<GroupMember[]>(() => {
    if (group) {
      return group.members.map((m) =>
        create(GroupMemberSchema, { member: m.member, role: m.role })
      );
    }
    return [
      create(GroupMemberSchema, {
        role: GroupMember_Role.OWNER,
        member: currentUser.name,
      }),
    ];
  });
  const [isRequesting, setIsRequesting] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  useEscapeKey(true, onClose);

  const isExternalGroup = !!group?.source;

  const allowEdit = isExternalGroup
    ? false
    : isEditMode
      ? group.members.some(
          (m) =>
            extractUserEmail(m.member) === currentUser.email &&
            m.role === GroupMember_Role.OWNER
        ) || hasWorkspacePermissionV2("bb.groups.update")
      : hasWorkspacePermissionV2("bb.groups.create");

  const fullEmail = useMemo(() => {
    if (isEditMode) return email;
    if (!email) return "";
    if (email.includes("@")) return email;
    return domainSuffix ? `${email}${domainSuffix}` : email;
  }, [email, isEditMode, domainSuffix]);

  const errorMessage = useMemo(() => {
    if (!title.trim())
      return (
        t("settings.members.groups.form.title") + " " + t("common.is-required")
      );
    if (!fullEmail || !isValidEmail(fullEmail))
      return t("settings.members.groups.form.email-tips");
    return "";
  }, [title, fullEmail, t]);

  const hasChanged = useMemo(() => {
    if (!isEditMode) return true;
    if (!group) return true;
    if (title !== group.title) return true;
    if (description !== group.description) return true;
    if (!isEqual(members, group.members)) return true;
    return false;
  }, [isEditMode, group, title, description, members]);

  const allowConfirm = !errorMessage && hasChanged;

  const handleAddMember = () => {
    setMembers((prev) => [
      ...prev,
      create(GroupMemberSchema, { role: GroupMember_Role.MEMBER, member: "" }),
    ]);
  };

  const handleRemoveMember = (index: number) => {
    setMembers((prev) => prev.filter((_, i) => i !== index));
  };

  const handleMemberChange = (
    index: number,
    field: "member" | "role",
    value: string | GroupMember_Role
  ) => {
    setMembers((prev) =>
      prev.map((m, i) => {
        if (i !== index) return m;
        return create(GroupMemberSchema, {
          ...m,
          [field]: value,
        });
      })
    );
  };

  const handleSubmit = async () => {
    if (!allowConfirm || !allowEdit) return;
    setIsRequesting(true);
    try {
      const dedupedMembers = deduplicateMembers(
        members.filter((m) => m.member)
      );
      if (isEditMode && group) {
        const validGroup = create(GroupSchema, {
          name: group.name,
          title,
          description,
          members: dedupedMembers,
        });
        const updated = await groupStore.updateGroup(validGroup);
        onUpdated(updated);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } else {
        const validGroup = create(GroupSchema, {
          name: `${groupNamePrefix}${fullEmail}`,
          title,
          description,
          members: dedupedMembers,
        });
        const created = await groupStore.createGroup(validGroup);
        onUpdated(created);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.created"),
        });
      }
      onClose();
    } catch {
      // error shown by store
    } finally {
      setIsRequesting(false);
    }
  };

  const handleDelete = async () => {
    if (!group) return;
    setIsRequesting(true);
    try {
      await groupStore.deleteGroup(group.name);
      onRemoved(group);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
      onClose();
    } catch {
      // error shown by store
    } finally {
      setIsRequesting(false);
      setShowDeleteConfirm(false);
    }
  };

  const canDelete =
    isEditMode &&
    !isExternalGroup &&
    (group.members.some(
      (m) =>
        extractUserEmail(m.member) === currentUser.email &&
        m.role === GroupMember_Role.OWNER
    ) ||
      hasWorkspacePermissionV2("bb.groups.delete"));

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 z-40 bg-black/30" onClick={onClose} />

      {/* Drawer */}
      <div
        role="dialog"
        aria-modal="true"
        className="fixed inset-y-0 right-0 z-50 w-[40rem] max-w-[100vw] bg-white shadow-xl flex flex-col"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-medium">
            {isEditMode
              ? t("settings.members.groups.update-group")
              : t("settings.members.groups.add-group")}
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto px-6 py-6">
          <div className="flex flex-col gap-y-6">
            {isExternalGroup && (
              <Alert variant="info">
                <AlertDescription>
                  {t("settings.members.groups.external-readonly")}
                </AlertDescription>
              </Alert>
            )}

            {/* Email */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.groups.form.email")}
                <span className="ml-0.5 text-error">*</span>
              </label>
              <span className="textinfolabel text-sm">
                {t("settings.members.groups.form.email-tips")}
              </span>
              <div className="flex items-center gap-x-1">
                <Input
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  disabled={isEditMode || !allowEdit}
                />
                {!isEditMode && domainSuffix && (
                  <span className="text-sm text-control-light whitespace-nowrap">
                    {domainSuffix}
                  </span>
                )}
              </div>
            </div>

            {/* Title */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.groups.form.title")}
                <span className="ml-0.5 text-error">*</span>
              </label>
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                maxLength={200}
                disabled={!allowEdit}
              />
            </div>

            {/* Description */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.groups.form.description")}
              </label>
              <Input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                maxLength={1000}
                disabled={!allowEdit}
              />
            </div>

            {/* Members */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("common.members", { count: 2 })}
              </label>
              <div className="flex flex-col gap-y-2">
                {members.map((member, index) => (
                  <div key={index} className="flex items-center gap-x-2">
                    <Input
                      className="flex-1"
                      value={member.member}
                      onChange={(e) =>
                        handleMemberChange(index, "member", e.target.value)
                      }
                      placeholder="users/hello@example.com"
                      disabled={!allowEdit}
                    />
                    <select
                      value={member.role}
                      onChange={(e) =>
                        handleMemberChange(
                          index,
                          "role",
                          Number(e.target.value) as GroupMember_Role
                        )
                      }
                      className="h-9 rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm"
                      disabled={!allowEdit}
                    >
                      <option value={GroupMember_Role.OWNER}>
                        {t("settings.members.groups.form.role.owner")}
                      </option>
                      <option value={GroupMember_Role.MEMBER}>
                        {t("settings.members.groups.form.role.member")}
                      </option>
                    </select>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 shrink-0 text-error hover:text-error"
                      onClick={() => handleRemoveMember(index)}
                      disabled={!allowEdit}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                {allowEdit && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="self-start"
                    onClick={handleAddMember}
                  >
                    <Plus className="h-4 w-4 mr-1" />
                    {t("settings.members.add-member")}
                  </Button>
                )}
              </div>
            </div>

            {errorMessage && (
              <p className="text-error text-sm">{errorMessage}</p>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t">
          <div>
            {canDelete && (
              <>
                {showDeleteConfirm ? (
                  <div className="flex items-center gap-x-2">
                    <span className="text-sm text-error">
                      {t("settings.members.action.deactivate-confirm-title")}
                    </span>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={handleDelete}
                      disabled={isRequesting}
                    >
                      {t("common.confirm")}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setShowDeleteConfirm(false)}
                    >
                      {t("common.cancel")}
                    </Button>
                  </div>
                ) : (
                  <Button
                    variant="ghost"
                    className="text-error hover:text-error"
                    onClick={() => setShowDeleteConfirm(true)}
                  >
                    <Trash2 className="h-4 w-4 mr-1" />
                    {t("common.delete")}
                  </Button>
                )}
              </>
            )}
          </div>
          <div className="flex items-center gap-x-2">
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button
              disabled={!allowEdit || !allowConfirm || isRequesting}
              onClick={handleSubmit}
            >
              {isEditMode ? t("common.update") : t("common.confirm")}
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}

// ============================================================
// AADSyncDrawer
// ============================================================

function AADSyncDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const settingV1Store = useSettingV1Store();

  const externalUrl = useVueState(
    () => actuatorStore.serverInfo?.externalUrl ?? ""
  );
  const workspaceResourceName = useVueState(
    () => actuatorStore.workspaceResourceName
  );
  const directorySyncToken = useVueState(
    () => settingV1Store.workspaceProfile.directorySyncToken
  );

  useEscapeKey(true, onClose);

  const scimUrl =
    externalUrl && workspaceResourceName
      ? `${externalUrl}/hook/scim/${workspaceResourceName}`
      : "";

  const copyToClipboard = async (value: string) => {
    await navigator.clipboard.writeText(value);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  };

  const handleResetToken = async () => {
    const confirmed = window.confirm(
      t("settings.members.entra-sync.reset-token-warning")
    );
    if (!confirmed) return;

    try {
      await settingV1Store.updateWorkspaceProfile({
        payload: { directorySyncToken: "" },
        updateMask: create(FieldMaskSchema, {
          paths: ["value.workspace_profile.directory_sync_token"],
        }),
      });
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
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 z-40 bg-black/30" onClick={onClose} />

      {/* Drawer */}
      <div
        role="dialog"
        aria-modal="true"
        className="fixed inset-y-0 right-0 z-50 w-[40rem] max-w-[100vw] bg-white shadow-xl flex flex-col"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-medium">
            {t("settings.members.entra-sync.self")}
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto px-6 py-6">
          <div className="flex flex-col gap-y-6">
            {/* Description */}
            <p className="text-sm text-control-light">
              {t("settings.members.entra-sync.description")}{" "}
              <a
                href="https://docs.bytebase.com/administration/scim/overview?source=console"
                target="_blank"
                rel="noopener noreferrer"
                className="text-accent hover:underline"
              >
                {t("common.learn-more")}
              </a>
            </p>

            {/* Missing external URL warning */}
            {!externalUrl && (
              <Alert variant="warning">
                <AlertDescription>{t("banner.external-url")}</AlertDescription>
              </Alert>
            )}

            {/* SCIM Endpoint URL */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.entra-sync.endpoint")}
              </label>
              <span className="textinfolabel text-sm">
                {t("settings.members.entra-sync.endpoint-tip")}
              </span>
              <div className="flex items-center gap-x-2">
                <Input readOnly value={scimUrl} className="flex-1 text-sm" />
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!scimUrl}
                  onClick={() => copyToClipboard(scimUrl)}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>

            {/* Secret Token */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.entra-sync.secret-token")}
              </label>
              <span className="textinfolabel text-sm">
                {t("settings.members.entra-sync.secret-token-tip")}
              </span>
              <div className="flex items-center gap-x-2">
                <Input
                  readOnly
                  type="password"
                  value={directorySyncToken}
                  className="flex-1 text-sm"
                />
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!directorySyncToken}
                  onClick={() => copyToClipboard(directorySyncToken)}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
              {hasWorkspacePermissionV2("bb.settings.setWorkspaceProfile") && (
                <Button
                  variant="outline"
                  size="sm"
                  className="self-start text-error border-error hover:bg-error/10"
                  onClick={handleResetToken}
                >
                  {t("settings.members.entra-sync.reset-token")}
                </Button>
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-x-2 px-6 py-4 border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
        </div>
      </div>
    </>
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

// ============================================================
// MemberTable (view by members)
// ============================================================

function MemberTable({
  bindings,
  allowEdit,
  selectedBindings,
  onSelectionChange,
  onUpdateBinding,
  onRevokeBinding,
}: {
  bindings: MemberBinding[];
  allowEdit: boolean;
  selectedBindings: string[];
  onSelectionChange: (selected: string[]) => void;
  onUpdateBinding: (binding: MemberBinding) => void;
  onRevokeBinding: (binding: MemberBinding) => void;
}) {
  const { t } = useTranslation();

  const allSelected =
    bindings.length > 0 &&
    bindings.every((b) => selectedBindings.includes(b.binding));

  const toggleAll = () => {
    if (allSelected) {
      onSelectionChange([]);
    } else {
      onSelectionChange(bindings.map((b) => b.binding));
    }
  };

  const toggleOne = (binding: string) => {
    onSelectionChange(
      selectedBindings.includes(binding)
        ? selectedBindings.filter((b) => b !== binding)
        : [...selectedBindings, binding]
    );
  };

  const canEdit = (mb: MemberBinding) => {
    if (mb.type === "users") return mb.user?.state !== State.DELETED;
    if (mb.type === "groups") return !mb.group?.deleted;
    return true;
  };

  return (
    <div className="border rounded-sm overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="bg-gray-50 border-b">
            {allowEdit && (
              <th className="w-10 px-3 py-2">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleAll}
                />
              </th>
            )}
            <th className="px-4 py-2 text-left font-medium text-control-light">
              {t("settings.members.table.account")}
            </th>
            <th className="px-4 py-2 text-left font-medium text-control-light">
              {t("settings.members.table.roles")}
            </th>
            <th className="w-24 px-4 py-2 text-left font-medium text-control-light">
              {t("common.operations")}
            </th>
          </tr>
        </thead>
        <tbody>
          {bindings.map((mb) => (
            <tr
              key={mb.binding}
              className="border-b last:border-b-0 hover:bg-gray-50"
            >
              {allowEdit && (
                <td className="px-3 py-2">
                  <input
                    type="checkbox"
                    checked={selectedBindings.includes(mb.binding)}
                    onChange={() => toggleOne(mb.binding)}
                  />
                </td>
              )}
              <td className="px-4 py-2">
                {mb.type === "users" ? (
                  <div className="flex flex-col">
                    <span className="font-medium">{mb.title}</span>
                    <span className="text-control-light text-xs">
                      {mb.user?.email}
                    </span>
                  </div>
                ) : (
                  <div className="flex items-center gap-x-2">
                    <Users className="h-4 w-4 text-control-light" />
                    <span className="font-medium">{mb.title}</span>
                    {mb.group && (
                      <span className="text-control-light text-xs">
                        ({mb.group.members.length}{" "}
                        {t("common.members", {
                          count: mb.group.members.length,
                        })}
                        )
                      </span>
                    )}
                    {mb.group?.deleted && (
                      <Badge variant="destructive" className="text-xs">
                        {t("common.deleted")}
                      </Badge>
                    )}
                  </div>
                )}
              </td>
              <td className="px-4 py-2">
                <div className="flex flex-wrap gap-1">
                  {sortRoles([...mb.workspaceLevelRoles]).map((role) => (
                    <Badge key={role} className="text-xs gap-x-1">
                      <Building2 className="h-3 w-3" />
                      {displayRoleTitle(role)}
                    </Badge>
                  ))}
                </div>
              </td>
              <td className="px-4 py-2">
                <div className="flex items-center gap-x-1">
                  {canEdit(mb) && (
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onUpdateBinding(mb)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                  )}
                  {allowEdit && (
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => {
                        if (
                          window.confirm(
                            t("settings.members.revoke-access-alert")
                          )
                        ) {
                          onRevokeBinding(mb);
                        }
                      }}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </td>
            </tr>
          ))}
          {bindings.length === 0 && (
            <tr>
              <td
                colSpan={allowEdit ? 4 : 3}
                className="px-4 py-8 text-center text-control-light"
              >
                {t("common.no-data")}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

// ============================================================
// MemberTableByRole (view by roles)
// ============================================================

function MemberTableByRole({
  bindings,
  allowEdit,
  onUpdateBinding,
  onRevokeBinding,
}: {
  bindings: MemberBinding[];
  allowEdit: boolean;
  onUpdateBinding: (binding: MemberBinding) => void;
  onRevokeBinding: (binding: MemberBinding) => void;
}) {
  const { t } = useTranslation();
  const [expandedRoles, setExpandedRoles] = useState<Set<string>>(new Set());

  const roleToBindings = useMemo(() => {
    const map = new Map<string, MemberBinding[]>();
    for (const mb of bindings) {
      for (const role of mb.workspaceLevelRoles) {
        if (!map.has(role)) map.set(role, []);
        map.get(role)!.push(mb);
      }
    }
    const sortedRoles = sortRoles([...map.keys()]);
    return sortedRoles.map((role) => ({
      role,
      members: map.get(role) ?? [],
    }));
  }, [bindings]);

  const toggleRole = (role: string) => {
    setExpandedRoles((prev) => {
      const next = new Set(prev);
      if (next.has(role)) next.delete(role);
      else next.add(role);
      return next;
    });
  };

  const canEdit = (mb: MemberBinding) => {
    if (mb.type === "users") return mb.user?.state !== State.DELETED;
    if (mb.type === "groups") return !mb.group?.deleted;
    return true;
  };

  return (
    <div className="border rounded-sm overflow-hidden">
      <table className="w-full text-sm">
        <tbody>
          {roleToBindings.map(({ role, members }) => {
            const expanded = expandedRoles.has(role);
            return (
              <React.Fragment key={role}>
                <tr
                  className="bg-gray-50 border-b cursor-pointer hover:bg-gray-100"
                  onClick={() => toggleRole(role)}
                >
                  <td colSpan={3} className="px-4 py-2">
                    <div className="flex items-center gap-x-2">
                      {expanded ? (
                        <ChevronDown className="h-4 w-4" />
                      ) : (
                        <ChevronRight className="h-4 w-4" />
                      )}
                      <Building2 className="h-4 w-4 text-control-light" />
                      <span className="font-medium">
                        {displayRoleTitle(role)}
                      </span>
                      <span className="text-control-light text-xs">
                        ({members.length})
                      </span>
                    </div>
                  </td>
                </tr>
                {expanded &&
                  members.map((mb) => (
                    <tr
                      key={`${role}-${mb.binding}`}
                      className="border-b last:border-b-0 hover:bg-gray-50"
                    >
                      <td className="px-4 py-2 pl-10">
                        {mb.type === "users" ? (
                          <div className="flex flex-col">
                            <span className="font-medium">{mb.title}</span>
                            <span className="text-control-light text-xs">
                              {mb.user?.email}
                            </span>
                          </div>
                        ) : (
                          <div className="flex items-center gap-x-2">
                            <Users className="h-4 w-4 text-control-light" />
                            <span className="font-medium">{mb.title}</span>
                            {mb.group?.deleted && (
                              <Badge variant="destructive" className="text-xs">
                                {t("common.deleted")}
                              </Badge>
                            )}
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-2" />
                      <td className="w-24 px-4 py-2">
                        <div className="flex items-center gap-x-1">
                          {canEdit(mb) && (
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={() => onUpdateBinding(mb)}
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                          )}
                          {allowEdit && (
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={() => {
                                if (
                                  window.confirm(
                                    t("settings.members.revoke-access-alert")
                                  )
                                ) {
                                  onRevokeBinding(mb);
                                }
                              }}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
              </React.Fragment>
            );
          })}
          {roleToBindings.length === 0 && (
            <tr>
              <td
                colSpan={3}
                className="px-4 py-8 text-center text-control-light"
              >
                {t("common.no-data")}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

// ============================================================
// EditMemberRoleDrawer
// ============================================================

function EditMemberRoleDrawer({
  member,
  onClose,
}: {
  member?: MemberBinding;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const workspaceStore = useWorkspaceV1Store();

  const isEditMode = !!member;

  const [memberInput, setMemberInput] = useState("");
  const [selectedRoles, setSelectedRoles] = useState<string[]>(() =>
    member ? [...member.workspaceLevelRoles] : [PresetRoleType.WORKSPACE_MEMBER]
  );
  const [isRequesting, setIsRequesting] = useState(false);

  useEscapeKey(true, onClose);

  const handleSubmit = async () => {
    setIsRequesting(true);
    try {
      if (isEditMode) {
        await workspaceStore.patchIamPolicy([
          { member: member.binding, roles: selectedRoles },
        ]);
      } else {
        const emails = memberInput
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean);
        const memberList = emails.map((email) => {
          if (email.startsWith("group:") || email.startsWith("user:")) {
            return email;
          }
          return `${userBindingPrefix}${email}`;
        });
        const batchPatch = memberList.map((m) => {
          const existedRoles = workspaceStore.findRolesByMember(m);
          return {
            member: m,
            roles: [...new Set([...selectedRoles, ...existedRoles])],
          };
        });
        await workspaceStore.patchIamPolicy(batchPatch);
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: isEditMode ? t("common.updated") : t("common.created"),
      });
      onClose();
    } catch {
      // error shown by store
    } finally {
      setIsRequesting(false);
    }
  };

  const handleRevoke = async () => {
    if (!member) return;
    const isAllUsers =
      member.binding === `${userBindingPrefix}${ALL_USERS_USER_EMAIL}`;
    const message = isAllUsers
      ? t("settings.members.revoke-allusers-alert")
      : t("settings.members.revoke-access-alert");
    if (!window.confirm(message)) return;

    setIsRequesting(true);
    try {
      await workspaceStore.patchIamPolicy([
        { member: member.binding, roles: [] },
      ]);
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("settings.members.revoked"),
      });
      onClose();
    } catch {
      // error shown by store
    } finally {
      setIsRequesting(false);
    }
  };

  const allowConfirm = isEditMode
    ? selectedRoles.length > 0
    : memberInput.trim().length > 0 && selectedRoles.length > 0;

  return (
    <>
      <div className="fixed inset-0 z-40 bg-black/30" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        className="fixed inset-y-0 right-0 z-50 w-[40rem] max-w-[100vw] bg-white shadow-xl flex flex-col"
      >
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-medium">
            {t("settings.members.grant-access")}
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-5 w-5" />
          </Button>
        </div>

        <div className="flex-1 overflow-auto px-6 py-6">
          <div className="flex flex-col gap-y-6">
            {/* Member input */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.select-account", { count: 1 })}
              </label>
              {isEditMode ? (
                <Input value={member.binding} disabled />
              ) : (
                <>
                  <Input
                    value={memberInput}
                    onChange={(e) => setMemberInput(e.target.value)}
                    placeholder="user:foo@example.com, group:bar@example.com"
                  />
                  <span className="text-xs text-control-light">
                    {t("settings.members.select-account-hint")}
                  </span>
                </>
              )}
            </div>

            {/* Roles */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.select-role", { count: 2 })}
              </label>
              <RoleMultiSelect
                value={selectedRoles}
                onChange={setSelectedRoles}
              />
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between px-6 py-4 border-t">
          <div>
            {isEditMode && (
              <Button
                variant="destructive"
                disabled={isRequesting}
                onClick={handleRevoke}
              >
                {t("settings.members.revoke-access")}
              </Button>
            )}
          </div>
          <div className="flex items-center gap-x-2">
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button
              disabled={!allowConfirm || isRequesting}
              onClick={handleSubmit}
            >
              {t("common.confirm")}
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}

export function UsersPage() {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const settingV1Store = useSettingV1Store();
  const userStore = useUserStore();
  const groupStore = useGroupStore();

  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const activeUserCount = useVueState(() => actuatorStore.activeUserCount);
  const userCountLimit = useVueState(() => subscriptionStore.userCountLimit);

  const [tab, setTab] = useState<TabValue>(() => getInitialTab(isSaaSMode));
  const [userSearchText, setUserSearchText] = useState("");
  const [groupSearchText, setGroupSearchText] = useState("");
  const [showInactiveUsers, setShowInactiveUsers] = useState(false);
  const [inactiveUserSearchText, setInactiveUserSearchText] = useState("");

  const hasUserGroupFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_USER_GROUPS)
  );
  const workspaceDomains = useVueState(
    () => settingV1Store.workspaceProfile.domains
  );
  const hasDirectorySyncFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_DIRECTORY_SYNC)
  );

  // Drawer visibility
  const [showCreateUserDrawer, setShowCreateUserDrawer] = useState(false);
  const [showCreateGroupDrawer, setShowCreateGroupDrawer] = useState(false);
  const [showAadSyncDrawer, setShowAadSyncDrawer] = useState(false);
  const [editingGroup, setEditingGroup] = useState<Group | undefined>(
    undefined
  );
  const [editingUser, setEditingUser] = useState<User | undefined>(undefined);

  // Members tab (SaaS mode)
  const workspaceStore = useWorkspaceV1Store();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const [memberSearchText, setMemberSearchText] = useState("");
  const [memberViewTab, setMemberViewTab] = useState<"MEMBERS" | "ROLES">(
    "MEMBERS"
  );
  const [selectedMembers, setSelectedMembers] = useState<string[]>([]);
  const [showEditMemberDrawer, setShowEditMemberDrawer] = useState(false);
  const [editingMember, setEditingMember] = useState<
    MemberBinding | undefined
  >();

  const memberBindings = useVueState(() =>
    getMemberBindings({
      policies: [
        {
          level: "WORKSPACE" as const,
          policy: workspaceStore.workspaceIamPolicy,
        },
      ],
      searchText: memberSearchText,
      ignoreRoles: new Set([]),
    })
  );

  const canSetIamPolicy = hasWorkspacePermissionV2(
    "bb.workspaces.setIamPolicy"
  );

  const handleRevokeSelected = async () => {
    if (
      selectedMembers.some(
        (m) => m === `${userBindingPrefix}${currentUser.email}`
      )
    ) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("settings.members.cannot-revoke-self"),
      });
      return;
    }
    if (window.confirm(t("settings.members.revoke-access-alert"))) {
      try {
        await workspaceStore.patchIamPolicy(
          selectedMembers.map((m) => ({ member: m, roles: [] }))
        );
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("settings.members.revoked"),
        });
        setSelectedMembers([]);
      } catch {
        // error already shown by store
      }
    }
  };

  const handleMemberUpdateBinding = (binding: MemberBinding) => {
    setEditingMember(binding);
    setShowEditMemberDrawer(true);
  };

  const handleMemberRevokeBinding = async (binding: MemberBinding) => {
    try {
      await workspaceStore.patchIamPolicy([
        { member: binding.binding, roles: [] },
      ]);
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("settings.members.revoked"),
      });
    } catch {
      // error already shown by store
    }
  };

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
    enabled: !isSaaSMode,
  });

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
    enabled: !isSaaSMode,
    fetchList: fetchInactiveUsers,
  });

  // Groups paged data
  const fetchGroups = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const { groups, nextPageToken } = await groupStore.fetchGroupList({
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        filter: { query: groupSearchText },
      });
      return { list: groups, nextPageToken };
    },
    [groupStore, groupSearchText]
  );

  const groupPaged = usePagedData<Group>({
    sessionKey: "bb.paged-group-table",
    fetchList: fetchGroups,
  });

  // Handle query param for group opening
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const name = params.get("name");
    if (name && name.startsWith("groups/")) {
      setTab("GROUPS");
      groupStore.getOrFetchGroupByIdentifier(name).then((group) => {
        if (group) {
          setEditingGroup(group);
          setShowCreateGroupDrawer(true);
        }
      });
    }
  }, []); // mount-only: read query param once
  // Sync tab to URL hash
  useEffect(() => {
    window.location.hash = tab;
  }, [tab]);

  const handleTabChange = (value: TabValue) => {
    if (value) setTab(value);
  };

  const handleActiveUserUpdated = (user: User) => {
    if (user.state === State.DELETED) {
      // Deactivated: remove from active list, add to inactive list
      activeUsers.removeCache(user);
      inactiveUsers.updateCache([user]);
    } else {
      activeUsers.updateCache([user]);
    }
  };

  const handleInactiveUserUpdated = (user: User) => {
    if (user.state === State.ACTIVE) {
      // Restored: remove from inactive list, add to active list
      inactiveUsers.removeCache(user);
      activeUsers.updateCache([user]);
    } else {
      inactiveUsers.updateCache([user]);
    }
  };

  const handleGroupSelected = (group: Group) => {
    setEditingGroup(group);
    setShowCreateGroupDrawer(true);
  };

  const handleGroupDeleted = (group: Group) => {
    groupPaged.removeCache(group);
  };

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {!isSaaSMode && remainingUserCount <= 3 && (
        <Alert variant="warning" className="mb-2">
          <AlertTitle>{t("subscription.usage.user-count.title")}</AlertTitle>
          <AlertDescription>
            {remainingUserCount > 0
              ? t("subscription.usage.user-count.remaining", {
                  total: userCountLimit,
                  count: remainingUserCount,
                })
              : t("subscription.usage.user-count.runoutof", {
                  total: userCountLimit,
                })}{" "}
            {t("subscription.usage.user-count.upgrade")}
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
                <Button
                  variant="outline"
                  disabled={!hasDirectorySyncFeature}
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
                  onClick={() => {
                    setEditingUser(create(UserSchema, { title: "" }));
                    setShowCreateUserDrawer(true);
                  }}
                >
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
                <Button
                  variant="outline"
                  disabled={!hasDirectorySyncFeature}
                  onClick={() => setShowAadSyncDrawer(true)}
                >
                  <Settings className="h-4 w-4 mr-1" />
                  <FeatureBadge
                    feature={PlanFeature.FEATURE_DIRECTORY_SYNC}
                    clickable={false}
                  />
                  {t("settings.members.entra-sync.self")}
                </Button>
                {workspaceDomains.length === 0 ? (
                  <Tooltip
                    content={
                      <span>
                        {t("settings.members.groups.workspace-domain-required")}{" "}
                        <a
                          href="/setting/general#domain-restriction"
                          className="underline text-accent"
                        >
                          {t("common.configure")}
                        </a>
                      </span>
                    }
                  >
                    <Button disabled>
                      <Plus className="h-4 w-4 mr-1" />
                      <FeatureBadge
                        feature={PlanFeature.FEATURE_USER_GROUPS}
                        clickable={false}
                      />
                      {t("settings.members.groups.add-group")}
                    </Button>
                  </Tooltip>
                ) : (
                  <Button
                    disabled={!hasUserGroupFeature}
                    onClick={() => {
                      setEditingGroup(undefined);
                      setShowCreateGroupDrawer(true);
                    }}
                  >
                    <Plus className="h-4 w-4 mr-1" />
                    <FeatureBadge
                      feature={PlanFeature.FEATURE_USER_GROUPS}
                      clickable={false}
                    />
                    {t("settings.members.groups.add-group")}
                  </Button>
                )}
              </>
            )}
            {tab === "MEMBERS" && (
              <>
                <div className="relative">
                  <Input
                    placeholder={t("settings.members.search-member")}
                    value={memberSearchText}
                    onChange={(e) => setMemberSearchText(e.target.value)}
                    className="h-8 text-sm pr-8"
                  />
                  <Search className="absolute right-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
                </div>
                {memberViewTab === "MEMBERS" && (
                  <Button
                    variant="outline"
                    disabled={!canSetIamPolicy || selectedMembers.length === 0}
                    onClick={handleRevokeSelected}
                  >
                    {t("settings.members.revoke-access")}
                  </Button>
                )}
                <Button
                  disabled={!canSetIamPolicy}
                  onClick={() => {
                    setEditingMember(undefined);
                    setShowEditMemberDrawer(true);
                  }}
                >
                  <Plus className="h-4 w-4 mr-1" />
                  {t("settings.members.grant-access")}
                </Button>
              </>
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
                    onUserSelected={(user) => {
                      setEditingUser(user);
                      setShowCreateUserDrawer(true);
                    }}
                    onGroupSelected={(group) => {
                      setTab("GROUPS");
                      handleGroupSelected(group);
                    }}
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
                      {t("settings.members.inactive-users")}
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
                        onUserSelected={(user) => {
                          setEditingUser(user);
                          setShowCreateUserDrawer(true);
                        }}
                        onGroupSelected={(group) => {
                          setTab("GROUPS");
                          handleGroupSelected(group);
                        }}
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
          </TabsPanel>
        )}
        {isSaaSMode && (
          <TabsPanel value="MEMBERS">
            <div className="py-4">
              <Tabs
                value={memberViewTab}
                onValueChange={(v) =>
                  setMemberViewTab(v as "MEMBERS" | "ROLES")
                }
              >
                <TabsList>
                  <TabsTrigger value="MEMBERS">
                    {t("settings.members.view-by-members")}
                  </TabsTrigger>
                  <TabsTrigger value="ROLES">
                    {t("settings.members.view-by-roles")}
                  </TabsTrigger>
                </TabsList>
                <TabsPanel value="MEMBERS">
                  <div className="py-4">
                    <MemberTable
                      bindings={memberBindings}
                      allowEdit={canSetIamPolicy}
                      selectedBindings={selectedMembers}
                      onSelectionChange={setSelectedMembers}
                      onUpdateBinding={handleMemberUpdateBinding}
                      onRevokeBinding={handleMemberRevokeBinding}
                    />
                  </div>
                </TabsPanel>
                <TabsPanel value="ROLES">
                  <div className="py-4">
                    <MemberTableByRole
                      bindings={memberBindings}
                      allowEdit={canSetIamPolicy}
                      onUpdateBinding={handleMemberUpdateBinding}
                      onRevokeBinding={handleMemberRevokeBinding}
                    />
                  </div>
                </TabsPanel>
              </Tabs>
            </div>
          </TabsPanel>
        )}
        <TabsPanel value="GROUPS">
          <div className="py-4 flex flex-col gap-y-4">
            <ComponentPermissionGuard permissions={["bb.groups.list"]}>
              {groupPaged.isLoading && groupPaged.dataList.length === 0 ? (
                <div className="flex items-center justify-center h-32">
                  <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
                </div>
              ) : (
                <>
                  <GroupTable
                    groups={groupPaged.dataList}
                    searchText={groupSearchText}
                    onGroupSelected={handleGroupSelected}
                    onGroupDeleted={handleGroupDeleted}
                  />
                  <PagedTableFooter
                    pageSize={groupPaged.pageSize}
                    pageSizeOptions={groupPaged.pageSizeOptions}
                    onPageSizeChange={groupPaged.onPageSizeChange}
                    hasMore={groupPaged.hasMore}
                    isFetchingMore={groupPaged.isFetchingMore}
                    onLoadMore={groupPaged.loadMore}
                  />
                </>
              )}
            </ComponentPermissionGuard>
          </div>
        </TabsPanel>
      </Tabs>

      {showCreateUserDrawer && (
        <CreateUserDrawer
          user={editingUser}
          onClose={() => {
            setShowCreateUserDrawer(false);
            setEditingUser(undefined);
          }}
          onCreated={(user) => {
            activeUsers.updateCache([user]);
          }}
          onUpdated={(user) => {
            activeUsers.updateCache([user]);
            if (user.state === State.DELETED) {
              activeUsers.removeCache(user);
              inactiveUsers.updateCache([user]);
            }
          }}
        />
      )}

      {showCreateGroupDrawer && (
        <CreateGroupDrawer
          group={editingGroup}
          onClose={() => {
            setShowCreateGroupDrawer(false);
            setEditingGroup(undefined);
          }}
          onUpdated={(group) => {
            groupPaged.updateCache([group]);
          }}
          onRemoved={(group) => {
            groupPaged.removeCache(group);
            setShowCreateGroupDrawer(false);
            setEditingGroup(undefined);
          }}
        />
      )}

      {showAadSyncDrawer && (
        <AADSyncDrawer onClose={() => setShowAadSyncDrawer(false)} />
      )}

      {showEditMemberDrawer && (
        <EditMemberRoleDrawer
          member={editingMember}
          onClose={() => {
            setShowEditMemberDrawer(false);
            setEditingMember(undefined);
          }}
        />
      )}
    </div>
  );
}
