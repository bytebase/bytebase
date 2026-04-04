import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import {
  ChevronDown,
  ChevronRight,
  Pencil,
  Plus,
  Search,
  Settings,
  Trash2,
  Users,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Alert, AlertDescription } from "@/react/components/ui/alert";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Tooltip } from "@/react/components/ui/tooltip";
import { type ColumnDef, useColumnWidths } from "@/react/hooks/useColumnWidths";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useGroupStore,
  useSettingV1Store,
  useSubscriptionV1Store,
  useUserStore,
} from "@/store";
import { extractUserEmail, groupNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_USER_NAME } from "@/types";
import type { Group, GroupMember } from "@/types/proto-es/v1/group_service_pb";
import {
  GroupMember_Role,
  GroupMemberSchema,
  GroupSchema,
} from "@/types/proto-es/v1/group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2, isValidEmail } from "@/utils";
import { AADSyncDrawer } from "./shared/AADSyncDrawer";
import { PagedTableFooter, usePagedData } from "./shared/usePagedData";

// ============================================================
// Helpers
// ============================================================

// Normalize member identifier to users/{email} format.
// Accepts: "users/foo@bar.com", "foo@bar.com" → "users/foo@bar.com"
function normalizeMemberIdentifier(member: string): string {
  const trimmed = member.trim();
  if (!trimmed) return trimmed;
  if (trimmed.startsWith("users/")) return trimmed;
  return `users/${trimmed}`;
}

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

// ============================================================
// HighlightText
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

  const columns: ColumnDef[] = useMemo(
    () => [
      { key: "groups-users", defaultWidth: 500, minWidth: 200 },
      { key: "actions", defaultWidth: 100, minWidth: 60, resizable: false },
    ],
    []
  );

  const { widths, totalWidth, onResizeStart } = useColumnWidths(
    columns,
    "bb.groups-table-widths"
  );

  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [memberCache, setMemberCache] = useState<Map<string, User[]>>(
    new Map()
  );
  const memberCacheRef = useRef(memberCache);
  memberCacheRef.current = memberCache;
  const loadingRef = useRef<Set<string>>(new Set());

  const fetchGroupMembers = useCallback(
    (group: Group) => {
      if (loadingRef.current.has(group.name)) return;
      loadingRef.current.add(group.name);
      const memberNames = group.members.map((m) => m.member);
      userStore
        .batchGetOrFetchUsers(memberNames)
        .then((users) => {
          setMemberCache((prev) => {
            const next = new Map(prev);
            next.set(
              group.name,
              users.filter((u): u is User => !!u)
            );
            return next;
          });
        })
        .catch(() => {
          // Allow retry on next expand
        })
        .finally(() => {
          loadingRef.current.delete(group.name);
        });
    },
    [userStore]
  );

  // Invalidate member cache when groups data changes (e.g. after editing membership)
  // and refetch members for currently expanded groups
  const prevGroupsRef = useRef(groups);
  const expandedGroupsRef = useRef(expandedGroups);
  expandedGroupsRef.current = expandedGroups;
  const fetchGroupMembersRef = useRef(fetchGroupMembers);
  fetchGroupMembersRef.current = fetchGroupMembers;
  useEffect(() => {
    if (prevGroupsRef.current !== groups) {
      prevGroupsRef.current = groups;
      setMemberCache(new Map());
      loadingRef.current = new Set();
      // Refetch members for currently expanded groups
      for (const groupName of expandedGroupsRef.current) {
        const group = groups.find((g) => g.name === groupName);
        if (group) {
          fetchGroupMembersRef.current(group);
        }
      }
    }
  }, [groups]);

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

      // Fetch members if not cached
      if (!memberCacheRef.current.has(group.name)) {
        fetchGroupMembers(group);
      }
    },
    [fetchGroupMembers]
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
    <div className="border rounded-sm overflow-x-auto">
      <Table style={{ width: `${totalWidth}px` }}>
        <colgroup>
          {widths.map((w, i) => (
            <col key={columns[i].key} style={{ width: `${w}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead resizable onResizeStart={(e) => onResizeStart(0, e)}>
              {t("common.groups")} / {t("common.users")}
            </TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
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
        </TableBody>
      </Table>
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
      <TableRow className={stripeBg}>
        <TableCell>
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
        </TableCell>
        <TableCell>
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
        </TableCell>
      </TableRow>
      {isExpanded &&
        members &&
        members.map((user) => {
          const memberInfo = group.members.find(
            (m) => extractUserEmail(m.member) === user.email
          );
          const isOwner = memberInfo?.role === GroupMember_Role.OWNER;

          return (
            <TableRow key={user.name} className={stripeBg}>
              <TableCell className="pl-14">
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
              </TableCell>
              <TableCell />
            </TableRow>
          );
        })}
      {isExpanded && !members && (
        <TableRow className={stripeBg}>
          <TableCell colSpan={2} className="pl-14">
            <div className="flex items-center gap-x-2 text-control-light text-sm">
              <div className="animate-spin h-4 w-4 border-2 border-accent border-t-transparent rounded-full" />
              {t("common.loading")}
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

// ============================================================
// CreateGroupDrawer
// ============================================================

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
  const actuatorStore = useActuatorV1Store();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const userStore = useUserStore();

  const isEditMode = !!group;
  const workspaceDomains = useVueState(
    () => settingV1Store.workspaceProfile.domains
  );
  const domainOptions = workspaceDomains.filter((d) => d.trim());

  const [email, setEmail] = useState(() => {
    if (!group) return "";
    return group.email ?? "";
  });
  const [selectedDomain, setSelectedDomain] = useState(() => {
    if (group?.email) {
      const atIdx = group.email.indexOf("@");
      if (atIdx >= 0) {
        const domain = group.email.slice(atIdx + 1);
        if (domainOptions.includes(domain)) return domain;
      }
    }
    return domainOptions[0] ?? "";
  });
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
    // When workspace domains are configured, enforce by using selected domain
    if (domainOptions.length > 0 && selectedDomain) {
      const localPart = email.split("@")[0];
      return `${localPart}@${selectedDomain}`;
    }
    return email;
  }, [email, isEditMode, domainOptions, selectedDomain]);

  const errorMessage = useMemo(() => {
    if (!title.trim())
      return (
        t("settings.members.groups.form.title") + " " + t("common.is-required")
      );
    if (!fullEmail || !isValidEmail(fullEmail))
      return t("settings.members.groups.form.email-tips");
    // Validate member identifiers — extract email part and check format
    for (const m of members) {
      const raw = m.member.trim();
      if (!raw) continue;
      const memberEmail = raw.startsWith("users/") ? raw.slice(6) : raw;
      if (!isValidEmail(memberEmail)) {
        return `Invalid member: ${raw}`;
      }
    }
    return "";
  }, [title, fullEmail, members, t]);

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
      const normalizedMembers = members
        .filter((m) => m.member.trim())
        .map((m) =>
          create(GroupMemberSchema, {
            ...m,
            member: normalizeMemberIdentifier(m.member),
          })
        );

      // In self-hosted mode, verify all members exist via backend lookup
      if (!isSaaSMode) {
        const notFound: string[] = [];
        for (const m of normalizedMembers) {
          const user = await userStore.getOrFetchUserByIdentifier({
            identifier: m.member,
            silent: true,
            fallback: false,
          });
          if (!user || user.name === UNKNOWN_USER_NAME) {
            notFound.push(m.member.replace("users/", ""));
          }
        }
        if (notFound.length > 0) {
          pushNotification({
            module: "bytebase",
            style: "WARN",
            title: `User not found: ${notFound.join(", ")}`,
          });
          setIsRequesting(false);
          return;
        }
      }

      const dedupedMembers = deduplicateMembers(normalizedMembers);
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
                  value={isEditMode ? email : email.split("@")[0]}
                  onChange={(e) => setEmail(e.target.value)}
                  disabled={isEditMode || !allowEdit}
                />
                {!isEditMode && domainOptions.length > 0 && (
                  <>
                    <span className="text-sm text-control-light">@</span>
                    {domainOptions.length === 1 ? (
                      <span className="text-sm text-control-light whitespace-nowrap">
                        {domainOptions[0]}
                      </span>
                    ) : (
                      <select
                        value={selectedDomain}
                        onChange={(e) => setSelectedDomain(e.target.value)}
                        className="border border-control-border rounded-sm text-sm pl-2 pr-6 py-1"
                        disabled={!allowEdit}
                      >
                        {domainOptions.map((d) => (
                          <option key={d} value={d}>
                            {d}
                          </option>
                        ))}
                      </select>
                    )}
                  </>
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
// GroupsPage (main)
// ============================================================

export function GroupsPage() {
  const { t } = useTranslation();
  const subscriptionStore = useSubscriptionV1Store();
  const settingV1Store = useSettingV1Store();
  const groupStore = useGroupStore();

  const hasUserGroupFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_USER_GROUPS)
  );
  const workspaceDomains = useVueState(
    () => settingV1Store.workspaceProfile.domains
  );
  const hasDirectorySyncFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_DIRECTORY_SYNC)
  );
  const canAccessSettings = hasWorkspacePermissionV2("bb.settings.get");

  const [groupSearchText, setGroupSearchText] = useState("");
  const [showCreateGroupDrawer, setShowCreateGroupDrawer] = useState(false);
  const [showAadSyncDrawer, setShowAadSyncDrawer] = useState(false);
  const [editingGroup, setEditingGroup] = useState<Group | undefined>(
    undefined
  );

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

  const hasGroupListPermission = hasWorkspacePermissionV2("bb.groups.list");
  const groupPaged = usePagedData<Group>({
    sessionKey: "bb.paged-group-table",
    fetchList: fetchGroups,
    enabled: hasGroupListPermission,
  });

  // Handle query param for group opening: ?name=groups/...
  useEffect(() => {
    const handleRouteChange = () => {
      const params = new URLSearchParams(window.location.search);
      const name = params.get("name");
      if (name && name.startsWith("groups/")) {
        // Clear the query param so the drawer doesn't reopen on refresh/navigation
        router.replace({
          ...router.currentRoute.value,
          query: {},
        });
        groupStore
          .getOrFetchGroupByIdentifier(name)
          .then((group) => {
            if (group) {
              setEditingGroup(group);
              setShowCreateGroupDrawer(true);
            }
          })
          .catch(() => {
            // Group not found or fetch failed — silently ignore
          });
      }
    };
    // Run on mount
    handleRouteChange();
    // Listen for in-app navigation via Vue router (pushState/replaceState)
    const unregister = router.afterEach(() => {
      handleRouteChange();
    });
    return () => unregister();
  }, []); // mount-only setup, router.afterEach handles subsequent changes

  const handleGroupSelected = (group: Group) => {
    setEditingGroup(group);
    setShowCreateGroupDrawer(true);
  };

  const handleGroupDeleted = (group: Group) => {
    groupPaged.removeCache(group);
  };

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {/* Action bar */}
      <div className="flex items-center justify-between mb-4">
        <div className="relative">
          <Input
            placeholder={t("common.filter-by-name")}
            value={groupSearchText}
            onChange={(e) => setGroupSearchText(e.target.value)}
            className="h-8 text-sm pr-8"
          />
          <Search className="absolute right-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
        </div>
        <div className="flex items-center gap-x-2">
          <Button
            variant="outline"
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
          {workspaceDomains.length === 0 ? (
            <Tooltip
              content={
                <span>
                  {t("settings.members.groups.workspace-domain-required")}{" "}
                  <a
                    href={
                      router.resolve({
                        name: SETTING_ROUTE_WORKSPACE_GENERAL,
                        hash: "#domain-restriction",
                      }).href
                    }
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
              disabled={
                !hasUserGroupFeature ||
                !hasWorkspacePermissionV2("bb.groups.create")
              }
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
        </div>
      </div>

      {/* Groups table */}
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
    </div>
  );
}
