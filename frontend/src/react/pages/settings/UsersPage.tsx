import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { ChevronDown, ChevronRight, CircleAlert, CircleCheck, Eye, EyeOff, Pencil, Plus, Search, Settings, Trash2, Undo2, Users, X } from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
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
import { useVueState } from "@/react/hooks/useVueState";
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
import { extractUserEmail, getUserFullNameByType } from "@/store/modules/v1/common";
import { AccountType, getAccountTypeByEmail, getUserEmailInBinding } from "@/types";
import { PresetRoleType } from "@/types/iam";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { GroupMember_Role } from "@/types/proto-es/v1/group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  UpdateUserRequestSchema,
  UserSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { displayRoleTitle, hasWorkspacePermissionV2 } from "@/utils";
import { getDefaultPagination } from "@/utils/pagination";

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
  onUserSelected,
}: {
  users: User[];
  onUserUpdated: (user: User) => void;
  onUserRemoved: (user: User) => void;
  onUserSelected?: (user: User) => void;
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
    const accountType = getAccountTypeByEmail(user.email);
    if (accountType === AccountType.USER && onUserSelected) {
      onUserSelected(user);
    } else {
      window.location.href = `/users/${user.email}`;
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
// Highlight helper
// ============================================================

function HighlightText({ text, keyword }: { text: string; keyword: string }) {
  if (!keyword.trim()) return <>{text}</>;
  const idx = text.toLowerCase().indexOf(keyword.toLowerCase());
  if (idx < 0) return <>{text}</>;
  return (
    <>
      {text.slice(0, idx)}
      <span className="bg-yellow-100">{text.slice(idx, idx + keyword.length)}</span>
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
  const [memberCache, setMemberCache] = useState<Map<string, User[]>>(new Map());
  const loadingRef = useRef<Set<string>>(new Set());

  const toggleExpand = useCallback(async (group: Group) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(group.name)) {
        next.delete(group.name);
      } else {
        next.add(group.name);
        // Fetch members if not cached
        if (!memberCache.has(group.name) && !loadingRef.current.has(group.name)) {
          loadingRef.current.add(group.name);
          const memberNames = group.members.map((m) => m.member);
          userStore.batchGetOrFetchUsers(memberNames).then((users) => {
            loadingRef.current.delete(group.name);
            setMemberCache((prev) => {
              const next = new Map(prev);
              next.set(group.name, users.filter((u): u is User => !!u));
              return next;
            });
          });
        }
      }
      return next;
    });
  }, [memberCache, userStore]);

  const isGroupOwner = useCallback((group: Group) => {
    return group.members.find(
      (m) => extractUserEmail(m.member) === currentUser.email
    )?.role === GroupMember_Role.OWNER;
  }, [currentUser.email]);

  const handleDelete = useCallback(async (group: Group) => {
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
  }, [groupStore, onGroupDeleted, t]);

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
            const canEdit = isGroupOwner(group) || hasWorkspacePermissionV2("bb.groups.update");
            const canDelete = isGroupOwner(group) || hasWorkspacePermissionV2("bb.groups.delete");

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
                  ({t("settings.members.groups.n-members", { n: group.members.length })})
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
      {isExpanded && members && members.map((user) => {
        const memberInfo = group.members.find(
          (m) => extractUserEmail(m.member) === user.email
        );
        const isOwner = memberInfo?.role === GroupMember_Role.OWNER;

        return (
          <tr key={user.name} className={`border-b last:border-b-0 ${stripeBg}`}>
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
// Escape key stack
// ============================================================

const escapeStack: (() => void)[] = [];

function useEscapeKey(onEscape: () => void) {
  useEffect(() => {
    escapeStack.push(onEscape);
    const handler = (e: KeyboardEvent) => {
      if (
        e.key === "Escape" &&
        escapeStack[escapeStack.length - 1] === onEscape
      ) {
        onEscape();
      }
    };
    document.addEventListener("keydown", handler);
    return () => {
      document.removeEventListener("keydown", handler);
      const idx = escapeStack.lastIndexOf(onEscape);
      if (idx >= 0) escapeStack.splice(idx, 1);
    };
  }, [onEscape]);
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
  const roleStore = useRoleStore();
  const settingV1Store = useSettingV1Store();

  const roleList = useVueState(() => roleStore.roleList);
  const userMapToRoles = useVueState(() => workspaceStore.userMapToRoles);
  const passwordRestriction = useVueState(
    () => settingV1Store.workspaceProfile.passwordRestriction
  );

  const isEditMode =
    !!user && user.name !== "" && !user.name.endsWith("/unknown");

  const allowUpdate = !isEditMode || hasWorkspacePermissionV2("bb.users.update");

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

  useEscapeKey(onClose);

  // Password validation
  const passwordChecks = useMemo(() => {
    const minLength = passwordRestriction?.minLength ?? 8;
    const checks: { text: string; matched: boolean }[] = [
      {
        text: t(
          "settings.general.workspace.password-restriction.min-length",
          { min: minLength }
        ),
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

  const passwordHint = password.length > 0 && passwordChecks.some((c) => !c.matched);
  const passwordMismatch =
    password.length > 0 && password !== passwordConfirm;

  const allowConfirm =
    email.length > 0 && !passwordHint && !passwordMismatch;

  const hasPermission = hasWorkspacePermissionV2(
    isEditMode ? "bb.users.update" : "bb.users.create"
  );

  const handleRoleToggle = (roleName: string) => {
    setRoles((prev) =>
      prev.includes(roleName)
        ? prev.filter((r) => r !== roleName)
        : [...prev, roleName]
    );
  };

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
      <div className="fixed inset-y-0 right-0 z-50 w-[40rem] max-w-[100vw] bg-white shadow-xl flex flex-col">
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
                placeholder="Foo"
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
                <div className="border rounded-sm max-h-48 overflow-auto p-2 flex flex-col gap-y-1">
                  {roleList.map((role) => (
                    <label
                      key={role.name}
                      className="flex items-center gap-x-2 px-2 py-1 rounded hover:bg-gray-50 cursor-pointer text-sm"
                    >
                      <input
                        type="checkbox"
                        checked={roles.includes(role.name)}
                        onChange={() => handleRoleToggle(role.name)}
                      />
                      {displayRoleTitle(role.name)}
                    </label>
                  ))}
                </div>
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
  const workspaceDomains = useVueState(() => settingV1Store.workspaceProfile.domains);

  // Drawer visibility
  const [showCreateUserDrawer, setShowCreateUserDrawer] = useState(false);
  const [showCreateGroupDrawer, setShowCreateGroupDrawer] = useState(false);
  const [_showAadSyncDrawer, _setShowAadSyncDrawer] = useState(false);
  const [editingGroup, setEditingGroup] = useState<Group | undefined>(undefined);
  const [editingUser, setEditingUser] = useState<User | undefined>(undefined);

  // Suppress unused warnings for drawers not yet implemented
  void showCreateGroupDrawer;
  void editingGroup;

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

  // Refresh groups when search text changes
  useEffect(() => {
    groupPaged.refresh();
  }, [groupSearchText]); // eslint-disable-line react-hooks/exhaustive-deps

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
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

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
                <Button variant="outline">
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
                    onUserSelected={(user) => {
                      setEditingUser(user);
                      setShowCreateUserDrawer(true);
                    }}
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
                        onUserSelected={(user) => {
                          setEditingUser(user);
                          setShowCreateUserDrawer(true);
                        }}
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
                    isLoading={groupPaged.isLoading}
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
    </div>
  );
}
