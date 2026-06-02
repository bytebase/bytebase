import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import {
  Building2,
  ChevronDown,
  ChevronRight,
  Pencil,
  Plus,
  ShieldUser,
  Trash2,
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
import { v4 as uuidv4 } from "uuid";
import type { ConditionGroupExpr, Factor, Operator } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { AccountMultiSelect } from "@/react/components/AccountMultiSelect";
import { DatabaseResourceSelector as DatabaseResourceSelectorComponent } from "@/react/components/DatabaseResourceSelector";
import { EnvironmentSelect } from "@/react/components/EnvironmentSelect";
import { ExprEditor, type OptionConfig } from "@/react/components/ExprEditor";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { LearnMoreLink } from "@/react/components/LearnMoreLink";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { RoleSelect } from "@/react/components/RoleSelect";
import { DDLWarningCallout } from "@/react/components/role-grant/DDLWarningCallout";
import { UserCell } from "@/react/components/UserCell";
import { Alert } from "@/react/components/ui/alert";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { ExpirationPicker } from "@/react/components/ui/expiration-picker";
import { Input } from "@/react/components/ui/input";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import {
  getMemberBindings,
  groupProjectRoleBindings,
} from "@/react/lib/memberBindings";
import {
  getRoleEnvironmentLimitationKind,
  roleHasDatabaseLimitation,
} from "@/react/lib/project-member/utils";
import { displayRoleTitleFromList } from "@/react/lib/role";
import { cn } from "@/react/lib/utils";
import {
  useNavigate,
  WORKSPACE_ROUTE_GROUPS,
  WORKSPACE_ROUTE_USER_PROFILE,
} from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  ALL_USERS_USER_EMAIL,
  type DatabaseResource,
  isDefaultProject,
  PresetRoleType,
  userBindingPrefix,
} from "@/types";
import { ExprSchema as ConditionExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import { type Binding, BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { GroupBinding, MemberBinding } from "@/types/v1/member";
import { AccountType, getAccountTypeByEmail } from "@/types/v1/user";
import {
  batchConvertParsedExprToCELString,
  formatAbsoluteDateTime,
  getDatabaseNameOptionConfig,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  isBindingPolicyExpired,
  sortRoles,
} from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";
import {
  buildConditionExpr,
  convertFromExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";
import { MemberBindingEnvironmentBanner } from "./MemberBindingEnvironmentBanner";
import { MemberDatabaseResourceName } from "./MemberDatabaseResourceName";
import { getSetIamPolicyPermissionGuardConfig } from "./membersPageActions";
import { getProjectRoleBindingEnvironmentLimitationState } from "./membersPageEnvironment";
import { RequestRoleSheet } from "./RequestRoleSheet";
import {
  getRequestRoleButtonState,
  REQUEST_ROLE_REQUIRED_PERMISSIONS,
} from "./requestRoleButton";
import type { DatabaseMode } from "./types";

const EMPTY_ROLE_SET = new Set<string>();

const assertNever = (value: never): never => {
  throw new Error(`Unexpected value: ${String(value)}`);
};

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
  scope,
}: {
  bindings: MemberBinding[];
  allowEdit: boolean;
  selectedBindings: string[];
  onSelectionChange: (selected: string[]) => void;
  onUpdateBinding: (binding: MemberBinding) => void;
  onRevokeBinding: (binding: MemberBinding) => void;
  scope: "workspace" | "project";
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const isSaaSMode = useVueState(() => useAppStore.getState().isSaaSMode());
  const batchGetOrFetchUsers = useAppStore(
    (state) => state.batchGetOrFetchUsers
  );
  const roleList = useAppStore((state) => state.roleList);
  const navigate = useNavigate();
  const canGetGroups = hasWorkspacePermissionV2("bb.groups.get");
  const canGetUsers = hasWorkspacePermissionV2("bb.users.get");

  // Group expand state. Cache is keyed by group name and invalidated
  // when the group-binding *content* changes — not on `bindings`
  // reference change, because the parent rebuilds `memberBindings` via
  // `useVueState(() => getMemberBindings(...))` and gets a new array
  // identity on every render. We can't use the `prevBindingsRef.current
  // !== bindings` shortcut that `GroupsPage` uses (its `groups` comes
  // from a reducer-backed `usePagedData` with stable identity).
  // Comparing a content-derived signature instead lets the cache only
  // reset on real membership changes — same effect, different trigger.
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [memberCache, setMemberCache] = useState<Map<string, User[]>>(
    new Map()
  );
  const memberCacheRef = useRef(memberCache);
  memberCacheRef.current = memberCache;
  const loadingRef = useRef<Set<string>>(new Set());

  const fetchGroupMembers = useCallback(
    (group: GroupBinding) => {
      if (loadingRef.current.has(group.name)) return;
      loadingRef.current.add(group.name);
      const memberNames = group.members.map((m) => m.member);
      batchGetOrFetchUsers(memberNames)
        .then((users: (User | undefined)[]) => {
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
        .finally(() => loadingRef.current.delete(group.name));
    },
    [batchGetOrFetchUsers]
  );

  // Signature of just the group bindings — that's all the cache cares
  // about. Recomputed on every render (because `bindings` reference
  // changes), but yields a stable string when group content is
  // unchanged, so the effect below only fires on real membership
  // changes.
  const groupBindingsSignature = useMemo(() => {
    const parts: string[] = [];
    for (const b of bindings) {
      if (b.type !== "groups" || !b.group) continue;
      const members = b.group.members
        .map((m) => m.member)
        .sort()
        .join(",");
      parts.push(`${b.group.name}:${members}`);
    }
    return parts.join("|");
  }, [bindings]);

  // Reset the cache and refetch currently-expanded groups when the
  // group-bindings signature changes (membership edit, IAM refresh).
  const expandedGroupsRef = useRef(expandedGroups);
  expandedGroupsRef.current = expandedGroups;
  const bindingsRef = useRef(bindings);
  bindingsRef.current = bindings;
  const fetchGroupMembersRef = useRef(fetchGroupMembers);
  fetchGroupMembersRef.current = fetchGroupMembers;
  useEffect(() => {
    setMemberCache(new Map());
    loadingRef.current = new Set();
    for (const groupName of expandedGroupsRef.current) {
      const mb = bindingsRef.current.find(
        (b) => b.type === "groups" && b.group?.name === groupName
      );
      if (mb?.group) {
        fetchGroupMembersRef.current(mb.group);
      }
    }
  }, [groupBindingsSignature]);

  const toggleGroupExpand = useCallback(
    (group: GroupBinding) => {
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

  const selectableBindings = useMemo(
    () =>
      bindings.filter(
        (b) => scope !== "project" || b.projectRoleBindings.length > 0
      ),
    [bindings, scope]
  );

  // Drop selections that are no longer rendered. The parent filters
  // `bindings` by search text, so without this a user could select rows,
  // type a search that hides them, and still bulk-revoke hidden members.
  useEffect(() => {
    const visibleNames = new Set(bindings.map((b) => b.binding));
    const next = selectedBindings.filter((b) => visibleNames.has(b));
    if (next.length !== selectedBindings.length) {
      onSelectionChange(next);
    }
  }, [bindings, selectedBindings, onSelectionChange]);

  const allSelected =
    selectableBindings.length > 0 &&
    selectableBindings.every((b) => selectedBindings.includes(b.binding));

  const toggleAll = () => {
    if (allSelected) {
      onSelectionChange([]);
    } else {
      onSelectionChange(selectableBindings.map((b) => b.binding));
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

  const isSelectDisabled = (mb: MemberBinding) => {
    return scope === "project" && mb.projectRoleBindings.length === 0;
  };

  const renderProjectRoleSummary = (bindings: Binding[]) => {
    // Show all roles, active first then expired. Expired chips get
    // line-through + reduced opacity so the count of stale bindings is
    // visible — exposing the issue rather than hiding it.
    const active = bindings.filter((b) => !isBindingPolicyExpired(b));
    const expired = bindings.filter((b) => isBindingPolicyExpired(b));
    return [
      ...groupProjectRoleBindings(active).map((group) => (
        <Badge key={`active-${group.role}`} className="text-xs gap-x-1">
          {displayRoleTitleFromList(group.role, roleList)}
          {group.bindings.length > 1 && (
            <span className="text-control-light">
              ({group.bindings.length})
            </span>
          )}
        </Badge>
      )),
      ...groupProjectRoleBindings(expired).map((group) => (
        <Badge
          key={`expired-${group.role}`}
          className="text-xs gap-x-1 line-through opacity-60"
        >
          {displayRoleTitleFromList(group.role, roleList)}
          {group.bindings.length > 1 && (
            <span className="text-control-light">
              ({group.bindings.length})
            </span>
          )}
        </Badge>
      )),
    ];
  };

  return (
    <div className="border rounded-sm overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow className="bg-control-bg">
            {allowEdit && (
              <TableHead className="w-10">
                <Checkbox checked={allSelected} onCheckedChange={toggleAll} />
              </TableHead>
            )}
            <TableHead>{t("settings.members.table.account")}</TableHead>
            <TableHead>{t("settings.members.table.roles")}</TableHead>
            <TableHead className="w-24">{t("common.operations")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {bindings.map((mb) => {
            const isGroupExpanded =
              mb.type === "groups" &&
              mb.group &&
              expandedGroups.has(mb.group.name);
            const groupMembers = mb.group
              ? memberCache.get(mb.group.name)
              : undefined;

            return (
              <React.Fragment key={mb.binding}>
                <TableRow>
                  {allowEdit && (
                    <TableCell>
                      <Checkbox
                        checked={selectedBindings.includes(mb.binding)}
                        disabled={isSelectDisabled(mb)}
                        onCheckedChange={() => toggleOne(mb.binding)}
                      />
                    </TableCell>
                  )}
                  <TableCell>
                    <UserCell
                      title={mb.title}
                      subtitle={
                        mb.type === "users"
                          ? mb.user?.email
                          : mb.binding.replace("group:", "groups/")
                      }
                      avatar={
                        mb.type === "groups" ? (
                          <>
                            <button
                              type="button"
                              className="flex size-5 shrink-0 items-center justify-center cursor-pointer"
                              onClick={() =>
                                mb.group && toggleGroupExpand(mb.group)
                              }
                            >
                              {isGroupExpanded ? (
                                <ChevronDown className="size-4 text-control-light" />
                              ) : (
                                <ChevronRight className="size-4 text-control-light" />
                              )}
                            </button>
                            <div className="size-9 rounded-full bg-control-bg-hover flex items-center justify-center shrink-0">
                              <Users className="size-4 text-control-light" />
                            </div>
                          </>
                        ) : undefined
                      }
                      nameLink={
                        mb.type === "users" && canGetUsers && mb.user?.email
                          ? {
                              onClick: () =>
                                void navigate.push({
                                  name: WORKSPACE_ROUTE_USER_PROFILE,
                                  params: {
                                    principalEmail: mb.user!.email,
                                  },
                                }),
                            }
                          : mb.type === "groups" && canGetGroups
                            ? {
                                onClick: () =>
                                  void navigate.push({
                                    name: WORKSPACE_ROUTE_GROUPS,
                                  }),
                              }
                            : undefined
                      }
                      badges={
                        <>
                          {mb.type === "users" &&
                            mb.user?.name === currentUser.name && (
                              <Badge className="text-xs">
                                {t("common.you")}
                              </Badge>
                            )}
                          {isSaaSMode && mb.type === "users" && mb.pending && (
                            <Badge variant="warning" className="text-xs">
                              {t("settings.members.pending-invite")}
                            </Badge>
                          )}
                          {mb.type === "users" &&
                            mb.user?.email &&
                            getAccountTypeByEmail(mb.user.email) ===
                              AccountType.SERVICE_ACCOUNT && (
                              <Badge variant="secondary" className="text-xs">
                                {t("settings.members.service-account")}
                              </Badge>
                            )}
                          {mb.type === "users" &&
                            mb.user?.email &&
                            getAccountTypeByEmail(mb.user.email) ===
                              AccountType.WORKLOAD_IDENTITY && (
                              <Badge variant="secondary" className="text-xs">
                                {t("settings.members.workload-identity")}
                              </Badge>
                            )}
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
                        </>
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {scope === "project"
                        ? renderProjectRoleSummary(mb.projectRoleBindings)
                        : sortRoles([...mb.workspaceLevelRoles]).map((role) => (
                            <Badge key={role} className="text-xs gap-x-1">
                              <Building2 className="h-3 w-3" />
                              {displayRoleTitleFromList(role, roleList)}
                            </Badge>
                          ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-x-1">
                      {allowEdit && canEdit(mb) && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => onUpdateBinding(mb)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                      )}
                      {allowEdit && (
                        <Button
                          variant="ghost"
                          size="sm"
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
                  </TableCell>
                </TableRow>
                {/* Expanded group members */}
                {isGroupExpanded &&
                  (groupMembers ? (
                    groupMembers.map((user) => (
                      <TableRow key={`${mb.binding}-${user.name}`}>
                        {allowEdit && <TableCell />}
                        <TableCell>
                          <UserCell
                            title={user.title}
                            subtitle={user.email}
                            size="sm"
                            className="pl-12"
                            nameLink={
                              canGetUsers
                                ? {
                                    onClick: () =>
                                      void navigate.push({
                                        name: WORKSPACE_ROUTE_USER_PROFILE,
                                        params: {
                                          principalEmail: user.email,
                                        },
                                      }),
                                  }
                                : undefined
                            }
                            badges={
                              user.name === currentUser.name ? (
                                <Badge className="text-xs">
                                  {t("common.you")}
                                </Badge>
                              ) : undefined
                            }
                          />
                        </TableCell>
                        <TableCell />
                        <TableCell />
                      </TableRow>
                    ))
                  ) : (
                    <TableRow>
                      {allowEdit && <TableCell />}
                      <TableCell>
                        <div className="flex items-center gap-x-3 pl-12">
                          <span className="text-sm text-control-light">
                            {t("common.loading")}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell />
                      <TableCell />
                    </TableRow>
                  ))}
              </React.Fragment>
            );
          })}
          {bindings.length === 0 && (
            <TableRow>
              <TableCell
                colSpan={allowEdit ? 4 : 3}
                className="px-4 py-8 text-center text-control-light"
              >
                {t("common.no-data")}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
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
  scope,
}: {
  bindings: MemberBinding[];
  allowEdit: boolean;
  onUpdateBinding: (binding: MemberBinding) => void;
  onRevokeBinding: (binding: MemberBinding) => void;
  scope: "workspace" | "project";
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const isSaaSMode = useVueState(() => useAppStore.getState().isSaaSMode());
  const roleList = useAppStore((state) => state.roleList);
  const [expandedRoles, setExpandedRoles] = useState<Set<string>>(new Set());
  const initializedRef = useRef(false);

  type RoleMember = { member: MemberBinding; allExpired: boolean };
  const roleToBindings = useMemo(() => {
    const map = new Map<string, RoleMember[]>();
    const appendToRole = (role: string, entry: RoleMember) => {
      const arr = map.get(role) ?? [];
      arr.push(entry);
      map.set(role, arr);
    };
    for (const mb of bindings) {
      if (scope === "project") {
        // Group this member's bindings by role so we can mark a row expired
        // only when ALL of its bindings for that role are expired.
        const bindingsByRole = new Map<string, Binding[]>();
        for (const b of mb.projectRoleBindings) {
          const arr = bindingsByRole.get(b.role) ?? [];
          arr.push(b);
          bindingsByRole.set(b.role, arr);
        }
        for (const [role, roleBindings] of bindingsByRole) {
          const allExpired = roleBindings.every((b) =>
            isBindingPolicyExpired(b)
          );
          appendToRole(role, { member: mb, allExpired });
        }
      } else {
        for (const role of mb.workspaceLevelRoles) {
          appendToRole(role, { member: mb, allExpired: false });
        }
      }
    }
    const sortedRoles = sortRoles([...map.keys()]);
    return sortedRoles.map((role) => ({
      role,
      // Active members first, expired-only members last.
      members: (map.get(role) ?? []).slice().sort((a, b) => {
        return Number(a.allExpired) - Number(b.allExpired);
      }),
    }));
  }, [bindings, scope]);

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
      <Table>
        <TableBody striped={false}>
          {roleToBindings.map(({ role, members }) => {
            const expanded = expandedRoles.has(role);
            return (
              <React.Fragment key={role}>
                <TableRow
                  className="bg-control-bg cursor-pointer"
                  onClick={() => toggleRole(role)}
                >
                  <TableCell colSpan={3}>
                    <div className="flex items-center gap-x-2">
                      {expanded ? (
                        <ChevronDown className="h-4 w-4" />
                      ) : (
                        <ChevronRight className="h-4 w-4" />
                      )}
                      {scope === "workspace" && (
                        <Building2 className="h-4 w-4 text-control-light" />
                      )}
                      <span className="font-medium">
                        {displayRoleTitleFromList(role, roleList)}
                      </span>
                      <span className="text-control-light text-xs">
                        ({members.length})
                      </span>
                    </div>
                  </TableCell>
                </TableRow>
                {expanded &&
                  members.map(({ member: mb, allExpired }) => (
                    <TableRow
                      key={`${role}-${mb.binding}`}
                      className={cn(allExpired && "opacity-60")}
                    >
                      <TableCell className="pl-10">
                        <UserCell
                          title={mb.title}
                          subtitle={
                            mb.type === "users"
                              ? mb.user?.email
                              : mb.binding.replace("group:", "groups/")
                          }
                          size="sm"
                          avatar={
                            mb.type === "groups" ? (
                              <div className="size-7 rounded-full bg-control-bg-hover flex items-center justify-center shrink-0">
                                <Users className="size-3.5 text-control-light" />
                              </div>
                            ) : undefined
                          }
                          nameClassName={
                            allExpired ? "line-through" : undefined
                          }
                          badges={
                            <>
                              {allExpired && (
                                <Badge
                                  variant="destructive"
                                  className="text-xs"
                                >
                                  {t("common.expired")}
                                </Badge>
                              )}
                              {mb.type === "users" &&
                                mb.user?.name === currentUser.name && (
                                  <Badge className="text-xs">
                                    {t("common.you")}
                                  </Badge>
                                )}
                              {isSaaSMode &&
                                mb.type === "users" &&
                                mb.pending && (
                                  <Badge variant="warning" className="text-xs">
                                    {t("settings.members.pending-invite")}
                                  </Badge>
                                )}
                              {mb.type === "users" &&
                                mb.user?.email &&
                                getAccountTypeByEmail(mb.user.email) ===
                                  AccountType.SERVICE_ACCOUNT && (
                                  <Badge
                                    variant="secondary"
                                    className="text-xs"
                                  >
                                    {t("settings.members.service-account")}
                                  </Badge>
                                )}
                              {mb.type === "users" &&
                                mb.user?.email &&
                                getAccountTypeByEmail(mb.user.email) ===
                                  AccountType.WORKLOAD_IDENTITY && (
                                  <Badge
                                    variant="secondary"
                                    className="text-xs"
                                  >
                                    {t("settings.members.workload-identity")}
                                  </Badge>
                                )}
                              {mb.group?.deleted && (
                                <Badge
                                  variant="destructive"
                                  className="text-xs"
                                >
                                  {t("common.deleted")}
                                </Badge>
                              )}
                            </>
                          }
                        />
                      </TableCell>
                      <TableCell />
                      <TableCell className="w-24">
                        <div className="flex items-center gap-x-1">
                          {allowEdit && canEdit(mb) && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => onUpdateBinding(mb)}
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                          )}
                          {allowEdit && (
                            <Button
                              variant="ghost"
                              size="sm"
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
                      </TableCell>
                    </TableRow>
                  ))}
              </React.Fragment>
            );
          })}
          {roleToBindings.length === 0 && (
            <TableRow>
              <TableCell
                colSpan={3}
                className="px-4 py-8 text-center text-control-light"
              >
                {t("common.no-data")}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}

// ============================================================
// Expiration presets
// ============================================================

interface ExpirationPreset {
  labelKey: string;
  days: number;
}

// A small set of common presets. Users needing any other duration pick the
// "Custom" option, which reveals a datetime picker.
const EXPIRATION_PRESETS: ExpirationPreset[] = [
  { labelKey: "project.members.expiration-presets.one-week", days: 7 },
  { labelKey: "project.members.expiration-presets.one-month", days: 30 },
  { labelKey: "project.members.expiration-presets.three-months", days: 90 },
  { labelKey: "project.members.expiration-presets.one-year", days: 365 },
];

function computeExpirationTimestamp(days?: number): number | undefined {
  if (days === undefined) return undefined;
  return Date.now() + days * 86400000;
}

function formatExpirationDate(timestampMs?: number): string {
  if (!timestampMs) return "";
  return new Date(timestampMs).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

// Validates the form's expiration against the workspace cap. "Never" (no
// timestamp) is only allowed when no cap is configured; a chosen timestamp
// must be in the future and within the cap.
function isExpirationValid(
  form: RoleBindingFormState,
  maximumRoleExpirationDays: number | undefined
): boolean {
  if (form.expirationTimestampInMS === undefined) {
    return maximumRoleExpirationDays === undefined;
  }
  const now = Date.now();
  if (form.expirationTimestampInMS <= now) return false;
  if (
    maximumRoleExpirationDays !== undefined &&
    form.expirationTimestampInMS > now + maximumRoleExpirationDays * 86400000
  ) {
    return false;
  }
  return true;
}

function ExpirationChip({
  label,
  selected,
  onClick,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={cn(
        "px-2.5 py-1 text-xs rounded-sm border transition-colors",
        selected
          ? "bg-accent text-accent-text border-accent"
          : "bg-background text-control border-control-border hover:bg-control-bg"
      )}
      onClick={onClick}
    >
      {label}
    </button>
  );
}

// ============================================================
// ProjectRoleBindingForm — one role binding form in create mode
// ============================================================

interface RoleBindingFormState {
  id: string;
  role: string;
  reason: string;
  expirationDays: number | undefined;
  expirationTimestampInMS: number | undefined;
  // When true, the user picked a custom datetime instead of a preset.
  expirationCustom: boolean;
  databaseMode: DatabaseMode;
  databaseResources: DatabaseResource[];
  exprGroup: ConditionGroupExpr;
  environments: string[];
}

// ============================================================
// DatabaseResourceSection — radio modes + selector
// ============================================================

function DatabaseResourceSection({
  projectName,
  mode,
  onModeChange,
  databaseResources,
  onDatabaseResourcesChange,
  exprGroup,
  onExprGroupChange,
  factorList,
  factorOptionConfigMap,
  factorOperatorOverrideMap,
  formId,
}: {
  projectName: string;
  mode: DatabaseMode;
  onModeChange: (mode: DatabaseMode) => void;
  databaseResources: DatabaseResource[];
  onDatabaseResourcesChange: (resources: DatabaseResource[]) => void;
  exprGroup: ConditionGroupExpr;
  onExprGroupChange: (expr: ConditionGroupExpr) => void;
  factorList: Factor[];
  factorOptionConfigMap: Map<Factor, OptionConfig>;
  factorOperatorOverrideMap: Map<Factor, Operator[]>;
  formId: string;
}) {
  const { t } = useTranslation();

  const modes: { value: DatabaseMode; label: string }[] = [
    { value: "ALL", label: t("common.all") },
    { value: "EXPRESSION", label: "CEL Expression" },
    { value: "SELECT", label: t("common.manually-select") },
  ];

  return (
    <div className="flex flex-col gap-y-2">
      <label className="block text-sm font-medium text-control">
        {t("common.databases")}
        <span className="ml-0.5 text-error">*</span>
      </label>
      <div className="flex items-center gap-x-4">
        {modes.map((m) => (
          <label
            key={m.value}
            className="flex items-center gap-x-2 text-sm cursor-pointer"
          >
            <input
              type="radio"
              name={`db-mode-${formId}`}
              checked={mode === m.value}
              onChange={() => onModeChange(m.value)}
            />
            {m.label}
          </label>
        ))}
      </div>

      {mode === "EXPRESSION" && (
        <ExprEditor
          expr={exprGroup}
          factorList={factorList}
          optionConfigMap={factorOptionConfigMap}
          factorOperatorOverrideMap={factorOperatorOverrideMap}
          onUpdate={onExprGroupChange}
        />
      )}

      {mode === "SELECT" && (
        <DatabaseResourceSelectorComponent
          projectName={projectName}
          value={databaseResources}
          onChange={onDatabaseResourcesChange}
        />
      )}
    </div>
  );
}

function ProjectRoleBindingForm({
  form,
  onChange,
  onRemove,
  canRemove,
  projectName,
  maximumRoleExpirationDays,
}: {
  form: RoleBindingFormState;
  onChange: (updated: RoleBindingFormState) => void;
  onRemove: () => void;
  canRemove: boolean;
  projectName: string;
  maximumRoleExpirationDays: number | undefined;
}) {
  const { t } = useTranslation();

  const roleList = useAppStore((state) => state.roleList);

  const visibleExpirationPresets = useMemo(
    () =>
      EXPIRATION_PRESETS.filter(
        (preset) =>
          maximumRoleExpirationDays === undefined ||
          preset.days <= maximumRoleExpirationDays
      ),
    [maximumRoleExpirationDays]
  );
  const minDatetime = dayjs().format("YYYY-MM-DDTHH:mm");
  const maxDatetime =
    maximumRoleExpirationDays !== undefined
      ? dayjs()
          .add(maximumRoleExpirationDays, "days")
          .format("YYYY-MM-DDTHH:mm")
      : undefined;
  const now = Date.now();
  const expirationIsInPast =
    form.expirationCustom &&
    form.expirationTimestampInMS !== undefined &&
    form.expirationTimestampInMS <= now;
  const expirationExceedsMax =
    form.expirationTimestampInMS !== undefined &&
    maximumRoleExpirationDays !== undefined &&
    form.expirationTimestampInMS > now + maximumRoleExpirationDays * 86400000;
  const factorList = useMemo<Factor[]>(
    () => [
      CEL_ATTRIBUTE_RESOURCE_DATABASE,
      CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
      CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
    ],
    []
  );
  const factorOperatorOverrideMap = useMemo(
    () =>
      new Map<Factor, Operator[]>([
        [CEL_ATTRIBUTE_RESOURCE_DATABASE, ["_==_", "@in"]],
        [CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME, ["_==_"]],
        [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, ["_==_", "@in"]],
      ]),
    []
  );
  const factorOptionConfigMap = useMemo(
    () =>
      factorList.reduce((map, factor) => {
        if (factor === CEL_ATTRIBUTE_RESOURCE_DATABASE) {
          map.set(factor, {
            ...getDatabaseNameOptionConfig(projectName),
            supportMultiple: false,
          });
        } else {
          map.set(factor, { options: [] });
        }
        return map;
      }, new Map<Factor, OptionConfig>()),
    [factorList, projectName]
  );

  const permissions = useMemo(() => {
    if (!form.role) return [];
    const presetRole = roleList.find((r) => r.name === form.role);
    return presetRole?.permissions ?? [];
  }, [form.role, roleList]);

  const showDatabases = useMemo(
    () => form.role && roleHasDatabaseLimitation(form.role),
    [form.role]
  );
  const envKind = useMemo(
    () => (form.role ? getRoleEnvironmentLimitationKind(form.role) : undefined),
    [form.role]
  );

  const handleRoleChange = (role: string) => {
    onChange({
      ...form,
      role,
      databaseMode: "ALL",
      databaseResources: [],
      exprGroup: wrapAsGroup(emptySimpleExpr()),
      environments: [],
    });
  };

  const handleReasonChange = (reason: string) => {
    onChange({ ...form, reason });
  };

  const handlePresetChange = (days: number | undefined) => {
    onChange({
      ...form,
      expirationCustom: false,
      expirationDays: days,
      expirationTimestampInMS: computeExpirationTimestamp(days),
    });
  };

  const handleCustomClick = () => {
    // Seed the picker with the current timestamp, or a default 1 week
    // (clamped to the cap) when switching from "Never".
    const seedDays =
      maximumRoleExpirationDays !== undefined
        ? Math.min(7, maximumRoleExpirationDays)
        : 7;
    onChange({
      ...form,
      expirationCustom: true,
      expirationDays: undefined,
      expirationTimestampInMS:
        form.expirationTimestampInMS ?? computeExpirationTimestamp(seedDays),
    });
  };

  const handleCustomDateChange = (value: string | undefined) => {
    onChange({
      ...form,
      expirationCustom: true,
      expirationDays: undefined,
      expirationTimestampInMS: value ? dayjs(value).valueOf() : undefined,
    });
  };

  return (
    <div className="border rounded-sm p-4 flex flex-col gap-y-4 relative">
      {canRemove && (
        <button
          type="button"
          className="absolute top-2 right-2 text-control-light hover:text-error"
          onClick={onRemove}
        >
          <X className="h-4 w-4" />
        </button>
      )}

      {/* Role select */}
      <div className="flex flex-col gap-y-2">
        <label className="block text-sm font-medium text-control">
          {t("settings.members.assign-role")}
        </label>
        <RoleSelect
          value={form.role ? [form.role] : []}
          onChange={(roles) => handleRoleChange(roles[0] ?? "")}
          multiple={false}
          scope="project"
        />
      </div>

      {/* Permissions display */}
      {permissions.length > 0 && (
        <div className="flex flex-col gap-y-2">
          <label className="block text-sm font-medium text-control">
            {t("common.permissions")}
          </label>
          <div className="max-h-32 overflow-auto border rounded-sm bg-control-bg p-2">
            <div className="flex flex-wrap gap-1">
              {permissions.map((perm) => (
                <span
                  key={perm}
                  className="inline-block rounded-xs bg-control-bg-hover px-1.5 py-0.5 text-xs text-control-light"
                >
                  {perm}
                </span>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Reason */}
      <div className="flex flex-col gap-y-2">
        <label className="block text-sm font-medium text-control">
          {t("common.reason")}{" "}
          <span className="text-control-light font-normal">
            ({t("common.optional")})
          </span>
        </label>
        <textarea
          className="w-full rounded-xs border border-control-border bg-transparent px-3 py-2 text-sm resize-none"
          rows={2}
          value={form.reason}
          onChange={(e) => handleReasonChange(e.target.value)}
        />
      </div>

      {/* Databases (conditional on role) */}
      {showDatabases && (
        <DatabaseResourceSection
          projectName={projectName}
          mode={form.databaseMode}
          onModeChange={(mode: DatabaseMode) =>
            onChange({
              ...form,
              databaseMode: mode,
              databaseResources: [],
              exprGroup: wrapAsGroup(emptySimpleExpr()),
            })
          }
          databaseResources={form.databaseResources}
          onDatabaseResourcesChange={(resources: DatabaseResource[]) =>
            onChange({ ...form, databaseResources: resources })
          }
          exprGroup={form.exprGroup}
          onExprGroupChange={(expr: ConditionGroupExpr) =>
            onChange({ ...form, exprGroup: expr })
          }
          factorList={factorList}
          factorOptionConfigMap={factorOptionConfigMap}
          factorOperatorOverrideMap={factorOperatorOverrideMap}
          formId={form.id}
        />
      )}

      {/* Environments (conditional on role) */}
      {envKind && (
        <div className="flex flex-col gap-y-2">
          <label className="block text-sm font-medium text-control">
            {t("common.environments")}
          </label>
          <DDLWarningCallout type="drawer" kind={envKind} />
          <EnvironmentSelect
            multiple
            portal
            value={form.environments}
            onChange={(envs) => onChange({ ...form, environments: envs })}
          />
        </div>
      )}

      {/* Expiration */}
      <div className="flex flex-col gap-y-2">
        <label className="block text-sm font-medium text-control">
          {t("common.expiration")}
          <span className="ml-0.5 text-error">*</span>
        </label>
        <div className="flex flex-wrap gap-1.5">
          {/* "Never" is only offered when the workspace sets no cap. */}
          {maximumRoleExpirationDays === undefined && (
            <ExpirationChip
              label={t("project.members.never-expires")}
              selected={
                !form.expirationCustom && form.expirationDays === undefined
              }
              onClick={() => handlePresetChange(undefined)}
            />
          )}
          {visibleExpirationPresets.map((preset) => (
            <ExpirationChip
              key={preset.days}
              label={t(preset.labelKey)}
              selected={
                !form.expirationCustom && form.expirationDays === preset.days
              }
              onClick={() => handlePresetChange(preset.days)}
            />
          ))}
          <ExpirationChip
            label={t("common.custom")}
            selected={form.expirationCustom}
            onClick={handleCustomClick}
          />
        </div>
        {form.expirationCustom && (
          <ExpirationPicker
            value={
              form.expirationTimestampInMS
                ? dayjs(form.expirationTimestampInMS).format("YYYY-MM-DDTHH:mm")
                : undefined
            }
            onChange={handleCustomDateChange}
            minDate={minDatetime}
            maxDate={maxDatetime}
          />
        )}
        {maximumRoleExpirationDays !== undefined && (
          <p className="text-xs text-control-light">
            {t("project.members.request-role.max-expiration-hint", {
              days: maximumRoleExpirationDays,
            })}
          </p>
        )}
        {expirationIsInPast && (
          <p className="text-xs text-error">
            {t("project.members.request-role.expiration-must-be-future")}
          </p>
        )}
        {expirationExceedsMax && (
          <p className="text-xs text-error">
            {t("project.members.request-role.expiration-exceeds-max", {
              days: maximumRoleExpirationDays,
            })}
          </p>
        )}
        {!form.expirationCustom && form.expirationTimestampInMS && (
          <span className="text-xs text-control-light">
            {t("project.members.expires-at", {
              date: formatExpirationDate(form.expirationTimestampInMS),
            })}
          </span>
        )}
      </div>
    </div>
  );
}

// ============================================================
// EditMemberRoleDrawer
// ============================================================

function EditMemberRoleDrawer({
  member,
  onClose,
  projectName,
  initialBindings,
}: {
  member?: MemberBinding;
  onClose: () => void;
  projectName?: string;
  initialBindings?: string[];
}) {
  const { t } = useTranslation();
  const patchWorkspaceIamPolicy = useAppStore(
    (state) => state.patchWorkspaceIamPolicy
  );
  const findWorkspaceRolesByMember = useAppStore(
    (state) => state.findWorkspaceRolesByMember
  );
  const getProjectIamPolicy = useAppStore((state) => state.getProjectIamPolicy);
  const updateProjectIamPolicy = useAppStore(
    (state) => state.updateProjectIamPolicy
  );
  const isSaaSMode = useVueState(() => useAppStore.getState().isSaaSMode());
  const roleList = useAppStore((state) => state.roleList);
  const settingsByName = useAppStore((s) => s.settingsByName);
  const hasEmailSetting = useMemo(
    () => !!useAppStore.getState().getSettingByName(Setting_SettingName.EMAIL),
    [settingsByName]
  );
  useEffect(() => {
    useAppStore
      .getState()
      .getOrFetchSettingByName(Setting_SettingName.EMAIL, true);
  }, []);

  const isEditMode = !!member;
  const isProjectCreateMode = !!projectName && !isEditMode;
  const isProjectEditMode = !!projectName && isEditMode;

  // Live project role bindings for the member (reactively updated when IAM policy changes).
  // Active bindings come first, expired ones last; original order is preserved within each group.
  const liveProjectRoleBindings = useVueState(() => {
    if (!isProjectEditMode || !member || !projectName) return [];
    const policy = getProjectIamPolicy(projectName);
    const matching = policy.bindings.filter((b) =>
      b.members.includes(member.binding)
    );
    return [...matching].sort((a, b) => {
      const aExpired = isBindingPolicyExpired(a) ? 1 : 0;
      const bExpired = isBindingPolicyExpired(b) ? 1 : 0;
      return aExpired - bExpired;
    });
  });

  const [selectedBindings, setSelectedBindings] = useState<string[]>(
    initialBindings ?? []
  );
  const [selectedRoles, setSelectedRoles] = useState<string[]>(() => {
    if (!member) return [];
    if (projectName) return member.projectRoleBindings.map((b) => b.role);
    return [...member.workspaceLevelRoles];
  });
  const [isRequesting, setIsRequesting] = useState(false);
  const [showNestedGrant, setShowNestedGrant] = useState(false);

  const [form, setForm] = useState<RoleBindingFormState>(() => ({
    id: uuidv4(),
    role: "",
    reason: "",
    expirationDays: 7,
    expirationTimestampInMS: computeExpirationTimestamp(7),
    expirationCustom: false,
    databaseMode: "ALL",
    databaseResources: [],
    exprGroup: wrapAsGroup(emptySimpleExpr()),
    environments: [],
  }));

  useEscapeKey(true, onClose);

  // Workspace-configured maximum role expiration, in days. PROJECT_OWNER
  // grants are exempt; returns undefined when no cap is set. Mirrors
  // RequestRoleSheet so direct grants and role requests behave the same.
  const workspaceProfile = useAppStore((s) => s.getWorkspaceProfile());
  const maximumRoleExpirationDays = useMemo(() => {
    if (form.role === PresetRoleType.PROJECT_OWNER) return undefined;
    const seconds = workspaceProfile.maximumRoleExpiration?.seconds;
    if (!seconds) return undefined;
    return Math.floor(Number(seconds) / (60 * 60 * 24));
  }, [workspaceProfile, form.role]);

  // Helpers for project edit mode
  const getSingleBindingRows = useCallback(
    (
      binding: Binding
    ): {
      databaseResource?: DatabaseResource;
      expiration?: Date;
    }[] => {
      if (!binding.parsedExpr) return [{}];
      const conditionExpr = convertFromExpr(binding.parsedExpr);
      const base: { expiration?: Date } = {};
      if (conditionExpr.expiredTime)
        base.expiration = new Date(conditionExpr.expiredTime);
      if (
        conditionExpr.databaseResources &&
        conditionExpr.databaseResources.length > 0
      ) {
        return conditionExpr.databaseResources.map((r) => ({
          ...base,
          databaseResource: r,
        }));
      }
      return [base];
    },
    []
  );

  const handleDeleteRole = async (roleBinding: Binding) => {
    if (!member || !projectName) return;
    const roleName = displayRoleTitleFromList(roleBinding.role, roleList);
    if (
      !window.confirm(
        t("project.members.revoke-role-from-member", {
          role: roleName,
          member: member.title,
        })
      )
    )
      return;
    setIsRequesting(true);
    try {
      const policy = structuredClone(getProjectIamPolicy(projectName));
      const match = policy.bindings.find(
        (b) =>
          b.role === roleBinding.role &&
          (b.condition?.expression ?? "") ===
            (roleBinding.condition?.expression ?? "")
      );
      if (match) {
        match.members = match.members.filter((m) => m !== member.binding);
      }
      policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
      await updateProjectIamPolicy(projectName, policy);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
      // Close the drawer only when the member has no remaining roles.
      const remaining = policy.bindings.some((b) =>
        b.members.includes(member.binding)
      );
      if (!remaining) {
        onClose();
      }
    } catch {
      // error shown by store
    } finally {
      setIsRequesting(false);
    }
  };

  const handleSubmit = async () => {
    setIsRequesting(true);
    try {
      if (projectName) {
        const policy = structuredClone(getProjectIamPolicy(projectName));
        if (isEditMode) {
          // Remove member from unconditional bindings only;
          // preserve conditional bindings (expiration, database scope)
          for (const binding of policy.bindings) {
            if (!binding.condition?.expression) {
              binding.members = binding.members.filter(
                (m) => m !== member.binding
              );
            }
          }
          policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
          // Add member to selected roles (unconditional)
          for (const role of selectedRoles) {
            const existing = policy.bindings.find(
              (b) => b.role === role && !b.condition?.expression
            );
            if (existing) {
              if (!existing.members.includes(member.binding)) {
                existing.members.push(member.binding);
              }
            } else {
              policy.bindings.push(
                create(BindingSchema, {
                  role,
                  members: [member.binding],
                })
              );
            }
          }
        } else {
          // Create mode with role binding form
          {
            if (!form.role) throw new Error("role is required");
            const databaseResources =
              form.databaseMode === "SELECT" &&
              form.databaseResources.length > 0
                ? form.databaseResources
                : undefined;
            const environments =
              form.role &&
              getRoleEnvironmentLimitationKind(form.role) !== undefined
                ? form.environments
                : undefined;
            const hasCondition =
              form.expirationTimestampInMS !== undefined ||
              form.reason !== "" ||
              databaseResources !== undefined ||
              environments !== undefined ||
              (form.databaseMode === "EXPRESSION" &&
                validateSimpleExpr(form.exprGroup));
            if (
              form.databaseMode === "EXPRESSION" &&
              validateSimpleExpr(form.exprGroup)
            ) {
              let parsedExpr;
              try {
                parsedExpr = await buildCELExpr(form.exprGroup);
              } catch {
                parsedExpr = undefined;
              }
              if (!parsedExpr) {
                pushNotification({
                  module: "bytebase",
                  style: "CRITICAL",
                  title: t(
                    "project.members.request-role.failed-to-build-expression"
                  ),
                });
                return;
              }
              const [exprString] = await batchConvertParsedExprToCELString([
                parsedExpr,
              ]);
              if (!exprString?.trim()) {
                pushNotification({
                  module: "bytebase",
                  style: "CRITICAL",
                  title: t(
                    "project.members.request-role.failed-to-build-expression"
                  ),
                });
                return;
              }
              const extraParts = stringifyConditionExpression({
                expirationTimestampInMS: form.expirationTimestampInMS,
                environments,
              });
              const fullExpression = extraParts
                ? `(${exprString}) && ${extraParts}`
                : exprString;
              const condition = create(ConditionExprSchema, {
                expression: fullExpression,
                description: form.reason,
              });
              const existingConditioned = policy.bindings.find(
                (b) =>
                  b.role === form.role &&
                  b.condition?.expression === condition.expression
              );
              if (existingConditioned) {
                for (const m of selectedBindings) {
                  if (!existingConditioned.members.includes(m)) {
                    existingConditioned.members.push(m);
                  }
                }
              } else {
                policy.bindings.push(
                  create(BindingSchema, {
                    role: form.role,
                    members: [...selectedBindings],
                    condition,
                  })
                );
              }
            } else if (hasCondition) {
              const condition = buildConditionExpr({
                role: form.role,
                description: form.reason,
                expirationTimestampInMS: form.expirationTimestampInMS,
                databaseResources,
                environments,
              });
              const existingConditioned = policy.bindings.find(
                (b) =>
                  b.role === form.role &&
                  b.condition?.expression === condition.expression
              );
              if (existingConditioned) {
                for (const m of selectedBindings) {
                  if (!existingConditioned.members.includes(m)) {
                    existingConditioned.members.push(m);
                  }
                }
              } else {
                policy.bindings.push(
                  create(BindingSchema, {
                    role: form.role,
                    members: [...selectedBindings],
                    condition,
                  })
                );
              }
            } else {
              // No condition: merge into existing unconditional binding
              const existing = policy.bindings.find(
                (b) => b.role === form.role && !b.condition?.expression
              );
              if (existing) {
                for (const binding of selectedBindings) {
                  if (!existing.members.includes(binding)) {
                    existing.members.push(binding);
                  }
                }
              } else {
                policy.bindings.push(
                  create(BindingSchema, {
                    role: form.role,
                    members: [...selectedBindings],
                  })
                );
              }
            }
          }
        }
        await updateProjectIamPolicy(projectName, policy);
      } else {
        if (isEditMode) {
          await patchWorkspaceIamPolicy([
            { member: member.binding, roles: selectedRoles },
          ]);
        } else {
          const batchPatch = selectedBindings.map((binding) => {
            const existedRoles = findWorkspaceRolesByMember(binding);
            return {
              member: binding,
              roles: [...new Set([...selectedRoles, ...existedRoles])],
            };
          });
          await patchWorkspaceIamPolicy(batchPatch);
        }
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
      if (projectName) {
        const policy = structuredClone(getProjectIamPolicy(projectName));
        for (const binding of policy.bindings) {
          binding.members = binding.members.filter((m) => m !== member.binding);
        }
        policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
        await updateProjectIamPolicy(projectName, policy);
      } else {
        await patchWorkspaceIamPolicy([{ member: member.binding, roles: [] }]);
      }
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

  const allowConfirm = isProjectCreateMode
    ? selectedBindings.length > 0 &&
      !!form.role &&
      isExpirationValid(form, maximumRoleExpirationDays) &&
      !(
        roleHasDatabaseLimitation(form.role) &&
        form.databaseMode === "SELECT" &&
        form.databaseResources.length === 0
      ) &&
      !(
        roleHasDatabaseLimitation(form.role) &&
        form.databaseMode === "EXPRESSION" &&
        !validateSimpleExpr(form.exprGroup)
      )
    : isEditMode
      ? selectedRoles.length > 0
      : selectedBindings.length > 0 && selectedRoles.length > 0;

  // Project edit mode: show detailed role bindings view
  if (isProjectEditMode && member) {
    const memberEmail = member.user?.email ?? member.binding;
    return (
      <>
        <Sheet open onOpenChange={(nextOpen) => !nextOpen && onClose()}>
          <SheetContent width="medium">
            <SheetHeader>
              <div className="flex min-w-0 items-center justify-between gap-x-2">
                <SheetTitle className="truncate">
                  {t("project.members.edit-member", {
                    member: `${member.title} (${memberEmail})`,
                  })}
                </SheetTitle>
                <Button
                  variant="outline"
                  onClick={() => setShowNestedGrant(true)}
                >
                  <Plus className="h-4 w-4 mr-1" />
                  {t("settings.members.grant-access")}
                </Button>
              </div>
            </SheetHeader>

            {/* Body — Role Bindings */}
            <SheetBody className="px-6 py-6">
              <div className="flex flex-col gap-y-6">
                {liveProjectRoleBindings.length === 0 && (
                  <div className="text-center text-control-light py-8">
                    {t("common.no-data")}
                  </div>
                )}
                {liveProjectRoleBindings.map((binding, idx) => {
                  const rows = getSingleBindingRows(binding);
                  const envLimitation =
                    getProjectRoleBindingEnvironmentLimitationState(binding);
                  const bindingKind = getRoleEnvironmentLimitationKind(
                    binding.role
                  );
                  const isExpired = isBindingPolicyExpired(binding);
                  return (
                    <div
                      key={`${binding.role}-${idx}`}
                      className={cn(
                        "border rounded-sm",
                        isExpired && "opacity-60"
                      )}
                    >
                      {/* Role header */}
                      <div className="flex items-center justify-between px-4 py-3 bg-control-bg border-b">
                        <div className="flex items-center gap-x-2">
                          <span
                            className={cn(
                              "font-medium text-sm",
                              isExpired && "line-through"
                            )}
                          >
                            {displayRoleTitleFromList(binding.role, roleList)}
                          </span>
                          {isExpired && (
                            <Badge variant="destructive" className="text-xs">
                              {t("common.expired")}
                            </Badge>
                          )}
                        </div>
                        <div className="flex items-center gap-x-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            title={t("common.edit")}
                            onClick={() => setShowNestedGrant(true)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            title={t("common.delete")}
                            disabled={isRequesting}
                            onClick={() => handleDeleteRole(binding)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>

                      {/* Environment info banner */}
                      {envLimitation && bindingKind && (
                        <div className="mx-4 mt-3">
                          <MemberBindingEnvironmentBanner
                            envLimitation={envLimitation}
                            bindingKind={bindingKind}
                          />
                        </div>
                      )}

                      {/* Database resources table */}
                      <div className="p-4">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>{t("common.database")}</TableHead>
                              <TableHead>{t("common.schema")}</TableHead>
                              <TableHead>{t("common.table")}</TableHead>
                              <TableHead>{t("common.expiration")}</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {rows.map((row, rowIdx) => (
                              <TableRow key={rowIdx}>
                                <TableCell>
                                  <MemberDatabaseResourceName
                                    resource={row.databaseResource}
                                  />
                                </TableCell>
                                <TableCell>
                                  {row.databaseResource?.schema ?? "*"}
                                </TableCell>
                                <TableCell>
                                  {row.databaseResource?.table ?? "*"}
                                </TableCell>
                                <TableCell>
                                  {row.expiration
                                    ? formatAbsoluteDateTime(
                                        row.expiration.getTime()
                                      )
                                    : t("project.members.never-expires")}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    </div>
                  );
                })}
              </div>
            </SheetBody>

            {/* Footer */}
            <SheetFooter className="justify-between">
              <div>
                <Button
                  variant="destructive"
                  disabled={isRequesting}
                  onClick={handleRevoke}
                >
                  {t("settings.members.revoke-access")}
                </Button>
              </div>
              <div className="flex items-center gap-x-2">
                <Button variant="outline" onClick={onClose}>
                  {t("common.cancel")}
                </Button>
                <Button onClick={onClose}>{t("common.ok")}</Button>
              </div>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Nested grant access drawer (stacked on top, member pre-selected) */}
        {showNestedGrant && (
          <EditMemberRoleDrawer
            projectName={projectName}
            initialBindings={[member.binding]}
            onClose={() => setShowNestedGrant(false)}
          />
        )}
      </>
    );
  }

  // Workspace edit mode, workspace/project create mode — original UI
  return (
    <Sheet open onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <SheetContent width="medium">
        <SheetHeader>
          <SheetTitle>{t("common.members", { count: 1 })}</SheetTitle>
        </SheetHeader>

        <SheetBody className="px-6 py-6">
          <div className="flex flex-col gap-y-6">
            {!isEditMode && !projectName && hasEmailSetting && (
              <Alert
                variant="info"
                description={t("settings.members.invite-email-hint")}
              />
            )}
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

            {/* Roles — project create mode uses rich form, otherwise simple multi-select */}
            {isProjectCreateMode ? (
              <div className="flex flex-col gap-y-4">
                <ProjectRoleBindingForm
                  form={form}
                  onChange={setForm}
                  onRemove={() => {}}
                  canRemove={false}
                  projectName={projectName}
                  maximumRoleExpirationDays={maximumRoleExpirationDays}
                />
              </div>
            ) : (
              <div className="flex flex-col gap-y-2">
                <label className="block text-sm font-medium text-control">
                  {t("settings.members.select-role", { count: 2 })}
                </label>
                <RoleSelect
                  value={selectedRoles}
                  onChange={setSelectedRoles}
                  scope={projectName ? "project" : undefined}
                />
              </div>
            )}
          </div>
        </SheetBody>

        <SheetFooter className="justify-between">
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
              {isEditMode ? t("common.update") : t("common.create")}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ============================================================
// MembersPage
// ============================================================

export function MembersPage({ projectId }: { projectId?: string }) {
  const { t } = useTranslation();
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  const patchWorkspaceIamPolicy = useAppStore(
    (state) => state.patchWorkspaceIamPolicy
  );
  const currentUser = useCurrentUser();
  const projectsByName = useAppStore((s) => s.projectsByName);
  const updateProjectIamPolicy = useAppStore(
    (state) => state.updateProjectIamPolicy
  );
  // subscribe to re-render on project cache change
  void projectsByName;

  const userCountInIam = useAppStore((s) => s.userCountInIam());
  const userCountLimit = useAppStore((s) => s.userCountLimit());
  const remainingUserCount = useMemo(
    () => Math.max(0, userCountLimit - userCountInIam),
    [userCountLimit, userCountInIam]
  );

  const projectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : undefined;
  const project = useVueState(() =>
    projectName
      ? useAppStore.getState().getProjectByName(projectName)
      : undefined
  );

  const [memberSearchText, setMemberSearchText] = useState("");
  const [memberViewTab, setMemberViewTab] = useState<"MEMBERS" | "ROLES">(
    "MEMBERS"
  );
  const [selectedMembers, setSelectedMembers] = useState<string[]>([]);
  const [showEditMemberDrawer, setShowEditMemberDrawer] = useState(false);
  const [editingMember, setEditingMember] = useState<
    MemberBinding | undefined
  >();
  const [showRequestRoleDialog, setShowRequestRoleDialog] = useState(false);

  const hasRequestRoleFeature = useVueState(() =>
    useAppStore.getState().hasFeature(PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW)
  );

  // IAM policy loads are owned by the parent shells: ProjectRouteShell
  // loads project IAM on /projects/:projectId/members, and
  // DashboardFrameShell's useEnsureWorkspaceCommonData loads workspace IAM
  // (+ referenced groups) on /settings/members. This page just reads them.
  // Subscribe directly to the Zustand projectPoliciesByName slice so the
  // member table re-renders when loadProjectIamPolicy / updateProjectIamPolicy
  // writes to the app store. Wrapping `getProjectIamPolicy()` in
  // `useVueState` would only re-render on Vue reactivity changes and miss
  // these Zustand writes.
  const projectIamPolicy = useAppStore((state) =>
    projectName ? state.projectPoliciesByName[projectName] : undefined
  );

  // `useVueState` ensures we re-render whenever any reactive dep
  // `getMemberBindings(...)` reads from changes — IAM policies, but
  // also the group / user / service-account / workload-identity stores
  // it pulls metadata from. The result array reference changes on every
  // render; the table component handles that with content-based change
  // detection (a group-bindings signature) so its expand-cache only
  // resets on real membership changes.
  // Keep this in useVueState so it re-runs when the Pinia user/group/
  // service-account/workload-identity stores that getMemberBindings reads
  // for member metadata change. The workspace IAM policy itself now comes
  // from the app store: subscribing to `workspacePolicy` above re-renders
  // this component on policy changes, and useVueState reads the latest getter
  // each render, so both reactivity sources are covered.
  const memberBindings = useVueState(() =>
    getMemberBindings({
      policies:
        projectName && projectIamPolicy
          ? [{ level: "PROJECT" as const, policy: projectIamPolicy }]
          : workspacePolicy
            ? [{ level: "WORKSPACE" as const, policy: workspacePolicy }]
            : [],
      searchText: memberSearchText,
      ignoreRoles: EMPTY_ROLE_SET,
    })
  );

  const canSetIamPolicy = project
    ? !isDefaultProject(project.name) &&
      project.state !== State.DELETED &&
      hasProjectPermissionV2(project, "bb.projects.setIamPolicy")
    : hasWorkspacePermissionV2("bb.workspaces.setIamPolicy");

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
        if (projectName && projectIamPolicy) {
          const policy = structuredClone(projectIamPolicy);
          for (const binding of policy.bindings) {
            binding.members = binding.members.filter(
              (member) => !selectedMembers.includes(member)
            );
          }
          policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
          await updateProjectIamPolicy(projectName, policy);
        } else {
          await patchWorkspaceIamPolicy(
            selectedMembers.map((m) => ({ member: m, roles: [] }))
          );
        }
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
      if (projectName && projectIamPolicy) {
        const policy = structuredClone(projectIamPolicy);
        for (const b of policy.bindings) {
          b.members = b.members.filter((member) => member !== binding.binding);
        }
        policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
        await updateProjectIamPolicy(projectName, policy);
      } else {
        await patchWorkspaceIamPolicy([{ member: binding.binding, roles: [] }]);
      }
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("settings.members.revoked"),
      });
    } catch {
      // error already shown by store
    }
  };

  const scope = projectName ? "project" : "workspace";

  const requestRoleButtonState = useMemo(
    () =>
      getRequestRoleButtonState({
        projectName,
        projectReady: !!project,
        allowRequestRole: project?.allowRequestRole ?? false,
        canSetIamPolicy,
        hasRequestRoleFeature,
      }),
    [projectName, project, canSetIamPolicy, hasRequestRoleFeature]
  );

  const requestRoleDisabledReason = useMemo(() => {
    const reason = requestRoleButtonState.disabledReason;
    if (!reason) return undefined;

    switch (reason.kind) {
      case "loading":
        return t("common.loading");
      case "allow-request-role-disabled":
        return t(
          "project.members.request-role.disabled-reason.allow-request-role-disabled"
        );
      case "can-grant-access-directly":
        return t(
          "project.members.request-role.disabled-reason.can-grant-access-directly",
          {
            permission: reason.permission,
          }
        );
      case "feature-unavailable":
        return t(
          "project.members.request-role.disabled-reason.feature-unavailable"
        );
      default:
        return assertNever(reason);
    }
  }, [requestRoleButtonState.disabledReason, t]);

  const setIamPolicyPermissionGuard = useMemo(
    () => getSetIamPolicyPermissionGuardConfig(project),
    [project]
  );

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {!projectName && remainingUserCount <= 3 && (
        <Alert
          variant="warning"
          className="mb-2"
          title={t("subscription.usage.user-count.title")}
          description={
            <>
              {remainingUserCount > 0
                ? t("subscription.usage.user-count.remaining", {
                    total: userCountLimit,
                    count: remainingUserCount,
                  })
                : t("subscription.usage.user-count.runoutof", {
                    total: userCountLimit,
                  })}{" "}
              {t("subscription.usage.user-count.upgrade")}
            </>
          }
        />
      )}
      {projectName && (
        <div className="textinfolabel mb-4">
          {t("project.members.description")}{" "}
          <LearnMoreLink
            href="https://docs.bytebase.com/administration/roles/?source=console#project-roles"
            className="text-accent"
          />
        </div>
      )}

      <div className="flex items-center justify-between gap-x-2 mb-4">
        <SearchInput
          placeholder={t("settings.members.search-member")}
          value={memberSearchText}
          onChange={(e) => setMemberSearchText(e.target.value)}
        />
        <div className="flex items-center gap-x-2">
          <PermissionGuard {...setIamPolicyPermissionGuard}>
            {({ disabled }) => (
              <div className="flex items-center gap-x-2">
                {memberViewTab === "MEMBERS" && (
                  <Button
                    variant="outline"
                    disabled={
                      disabled ||
                      !canSetIamPolicy ||
                      selectedMembers.length === 0
                    }
                    onClick={handleRevokeSelected}
                  >
                    {t("settings.members.revoke-access")}
                  </Button>
                )}
                <Button
                  disabled={disabled || !canSetIamPolicy}
                  onClick={() => {
                    setEditingMember(undefined);
                    setShowEditMemberDrawer(true);
                  }}
                >
                  <Plus className="h-4 w-4 mr-1" />
                  {t("settings.members.grant-access")}
                </Button>
              </div>
            )}
          </PermissionGuard>
          {requestRoleButtonState.visible &&
            (requestRoleDisabledReason ? (
              <Tooltip content={requestRoleDisabledReason}>
                <span className="inline-flex">
                  <Button
                    disabled
                    onClick={() => setShowRequestRoleDialog(true)}
                  >
                    {hasRequestRoleFeature ? (
                      <ShieldUser className="size-4 mr-1" />
                    ) : (
                      <FeatureBadge
                        feature={PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW}
                        clickable={false}
                        className="mr-1"
                      />
                    )}
                    {t("issue.title.request-role")}
                  </Button>
                </span>
              </Tooltip>
            ) : (
              <PermissionGuard
                permissions={[...REQUEST_ROLE_REQUIRED_PERMISSIONS]}
                project={project}
              >
                {({ disabled }) => (
                  <Button
                    disabled={disabled}
                    onClick={() => setShowRequestRoleDialog(true)}
                  >
                    <ShieldUser className="size-4 mr-1" />
                    {t("issue.title.request-role")}
                  </Button>
                )}
              </PermissionGuard>
            ))}
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
              scope={scope}
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
              scope={scope}
            />
          </div>
        </TabsPanel>
      </Tabs>

      {project && (
        <RequestRoleSheet
          open={showRequestRoleDialog}
          project={project}
          onClose={() => setShowRequestRoleDialog(false)}
        />
      )}

      {showEditMemberDrawer && (
        <EditMemberRoleDrawer
          member={editingMember}
          projectName={projectName}
          onClose={() => {
            setShowEditMemberDrawer(false);
            setEditingMember(undefined);
          }}
        />
      )}
    </div>
  );
}
