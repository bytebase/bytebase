import {
  Building2,
  ChevronDown,
  ChevronRight,
  Pencil,
  Plus,
  Search,
  Trash2,
  Users,
  X,
} from "lucide-react";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { MemberBinding } from "@/components/Member/types";
import { getMemberBindings } from "@/components/Member/utils";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useWorkspaceV1Store,
} from "@/store";
import { ALL_USERS_USER_EMAIL, userBindingPrefix } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { displayRoleTitle, hasWorkspacePermissionV2, sortRoles } from "@/utils";
import { AccountMultiSelect } from "./shared/AccountMultiSelect";
import { RoleMultiSelect } from "./shared/RoleMultiSelect";
import { UserAvatar } from "./shared/UserAvatar";

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
            <th className="w-24 px-4 py-2 text-left font-medium text-control-light" />
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
                <div className="flex items-center gap-x-3">
                  {mb.type === "users" ? (
                    <UserAvatar title={mb.title || mb.user?.email || "?"} />
                  ) : (
                    <div className="h-9 w-9 rounded-full bg-gray-200 flex items-center justify-center shrink-0">
                      <Users className="h-4 w-4 text-gray-500" />
                    </div>
                  )}
                  <div className="flex flex-col">
                    <div className="flex items-center gap-x-1.5">
                      <span className="font-medium text-accent">
                        {mb.title}
                      </span>
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
                    <span className="text-control-light text-xs">
                      {mb.type === "users"
                        ? mb.user?.email
                        : mb.binding.replace("group:", "groups/")}
                    </span>
                  </div>
                </div>
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
                  {allowEdit && canEdit(mb) && (
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
  const initializedRef = useRef(false);

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

  // Expand all roles by default on first load
  useEffect(() => {
    if (!initializedRef.current && roleToBindings.length > 0) {
      initializedRef.current = true;
      setExpandedRoles(new Set(roleToBindings.map((r) => r.role)));
    }
  }, [roleToBindings]);

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
                        <div className="flex items-center gap-x-3">
                          {mb.type === "users" ? (
                            <UserAvatar
                              title={mb.title || mb.user?.email || "?"}
                              size="sm"
                            />
                          ) : (
                            <div className="h-7 w-7 rounded-full bg-gray-200 flex items-center justify-center shrink-0">
                              <Users className="h-3.5 w-3.5 text-gray-500" />
                            </div>
                          )}
                          <div className="flex flex-col">
                            <span className="font-medium text-accent">
                              {mb.title}
                            </span>
                            <span className="text-control-light text-xs">
                              {mb.type === "users"
                                ? mb.user?.email
                                : mb.binding.replace("group:", "groups/")}
                            </span>
                          </div>
                          {mb.group?.deleted && (
                            <Badge variant="destructive" className="text-xs">
                              {t("common.deleted")}
                            </Badge>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-2" />
                      <td className="w-24 px-4 py-2">
                        <div className="flex items-center gap-x-1">
                          {allowEdit && canEdit(mb) && (
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
  const isSaaSMode = useVueState(() => useActuatorV1Store().isSaaSMode);

  const isEditMode = !!member;

  const [selectedBindings, setSelectedBindings] = useState<string[]>([]);
  const [selectedRoles, setSelectedRoles] = useState<string[]>(() =>
    member ? [...member.workspaceLevelRoles] : []
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
        const batchPatch = selectedBindings.map((binding) => {
          const existedRoles = workspaceStore.findRolesByMember(binding);
          return {
            member: binding,
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
      member.binding === ALL_USERS_USER_EMAIL ||
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
    : selectedBindings.length > 0 && selectedRoles.length > 0;

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
                <AccountMultiSelect
                  value={selectedBindings}
                  onChange={setSelectedBindings}
                  includeAllUsers={!isSaaSMode}
                />
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

// ============================================================
// MembersPage
// ============================================================

export function MembersPage() {
  const { t } = useTranslation();
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

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      <div className="flex items-center justify-between mb-4">
        <div className="relative">
          <Input
            placeholder={t("settings.members.search-member")}
            value={memberSearchText}
            onChange={(e) => setMemberSearchText(e.target.value)}
            className="h-8 text-sm pr-8"
          />
          <Search className="absolute right-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
        </div>
        <div className="flex items-center gap-x-2">
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
        </div>
      </div>

      <Tabs
        value={memberViewTab}
        onValueChange={(v) => setMemberViewTab(v as "MEMBERS" | "ROLES")}
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
