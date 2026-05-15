import { create } from "@bufbuild/protobuf";
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
import { AccountMultiSelect } from "@/react/components/AccountMultiSelect";
import { DatabaseResourceSelector as DatabaseResourceSelectorComponent } from "@/react/components/DatabaseResourceSelector";
import { EnvironmentMultiSelect } from "@/react/components/EnvironmentMultiSelect";
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
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import {
  getRoleEnvironmentLimitationKind,
  roleHasDatabaseLimitation,
} from "@/react/lib/project-member/utils";
import { cn } from "@/react/lib/utils";
import {
  useNavigate,
  WORKSPACE_ROUTE_GROUPS,
  WORKSPACE_ROUTE_USER_PROFILE,
} from "@/react/router";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useRoleStore,
  useSettingV1Store,
  useSubscriptionV1Store,
  useUserStore,
  useWorkspaceV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  ALL_USERS_USER_EMAIL,
  type DatabaseResource,
  isDefaultProject,
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
  displayRoleTitle,
  formatAbsoluteDateTime,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  isBindingPolicyExpired,
  sortRoles,
} from "@/utils";
import {
  buildConditionExpr,
  convertFromExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";
import { getMemberBindings, groupProjectRoleBindings } from "@/utils/v1/member";
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
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const actuatorStore = useActuatorV1Store();
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const userStore = useUserStore();
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
      userStore
        .batchGetOrFetchUsers(memberNames)
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
    [userStore]
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
          {displayRoleTitle(group.role)}
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
          {displayRoleTitle(group.role)}
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
                              {displayRoleTitle(role)}
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
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const actuatorStore = useActuatorV1Store();
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
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
                        {displayRoleTitle(role)}
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
  label: string;
  days?: number;
}

function getExpirationPresets(t: (key: string) => string): ExpirationPreset[] {
  return [
    { label: t("project.members.never-expires") },
    { label: "1 day", days: 1 },
    { label: "3 days", days: 3 },
    { label: "1 week", days: 7 },
    { label: "1 month", days: 30 },
    { label: "3 months", days: 90 },
    { label: "6 months", days: 180 },
    { label: "1 year", days: 365 },
  ];
}

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

// ============================================================
// ProjectRoleBindingForm — one role binding form in create mode
// ============================================================

interface RoleBindingFormState {
  id: string;
  role: string;
  reason: string;
  expirationDays: number | undefined;
  expirationTimestampInMS: number | undefined;
  databaseMode: DatabaseMode;
  databaseResources: DatabaseResource[];
  celExpression: string;
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
  celExpression,
  onCelExpressionChange,
  formId,
}: {
  projectName: string;
  mode: DatabaseMode;
  onModeChange: (mode: DatabaseMode) => void;
  databaseResources: DatabaseResource[];
  onDatabaseResourcesChange: (resources: DatabaseResource[]) => void;
  celExpression: string;
  onCelExpressionChange: (expr: string) => void;
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
        <textarea
          className="w-full rounded-xs border border-control-border bg-transparent px-3 py-2 text-sm font-mono resize-none"
          rows={3}
          placeholder='e.g. resource.database_name.startsWith("employee_")'
          value={celExpression}
          onChange={(e) => onCelExpressionChange(e.target.value)}
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
}: {
  form: RoleBindingFormState;
  onChange: (updated: RoleBindingFormState) => void;
  onRemove: () => void;
  canRemove: boolean;
  projectName: string;
}) {
  const { t } = useTranslation();
  const roleStore = useRoleStore();
  const roleList = useVueState(() => [...roleStore.roleList]);

  const expirationPresets = useMemo(() => getExpirationPresets(t), [t]);

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
      celExpression: "",
      environments: [],
    });
  };

  const handleReasonChange = (reason: string) => {
    onChange({ ...form, reason });
  };

  const handleExpirationChange = (days: number | undefined) => {
    onChange({
      ...form,
      expirationDays: days,
      expirationTimestampInMS: computeExpirationTimestamp(days),
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
              celExpression: "",
            })
          }
          databaseResources={form.databaseResources}
          onDatabaseResourcesChange={(resources: DatabaseResource[]) =>
            onChange({ ...form, databaseResources: resources })
          }
          celExpression={form.celExpression}
          onCelExpressionChange={(expr: string) =>
            onChange({ ...form, celExpression: expr })
          }
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
          <EnvironmentMultiSelect
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
          {expirationPresets.map((preset) => {
            const isSelected = form.expirationDays === preset.days;
            return (
              <button
                key={preset.label}
                type="button"
                className={cn(
                  "px-2.5 py-1 text-xs rounded-sm border transition-colors",
                  isSelected
                    ? "bg-accent text-accent-text border-accent"
                    : "bg-background text-control border-control-border hover:bg-control-bg"
                )}
                onClick={() => handleExpirationChange(preset.days)}
              >
                {preset.label}
              </button>
            );
          })}
        </div>
        {form.expirationTimestampInMS && (
          <span className="text-xs text-control-light">
            Expires: {formatExpirationDate(form.expirationTimestampInMS)}
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
  const workspaceStore = useWorkspaceV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const isSaaSMode = useVueState(() => useActuatorV1Store().isSaaSMode);
  const settingV1Store = useSettingV1Store();
  const hasEmailSetting = useVueState(
    () => !!settingV1Store.getSettingByName(Setting_SettingName.EMAIL)
  );
  useEffect(() => {
    settingV1Store.getOrFetchSettingByName(Setting_SettingName.EMAIL, true);
  }, [settingV1Store]);

  const isEditMode = !!member;
  const isProjectCreateMode = !!projectName && !isEditMode;
  const isProjectEditMode = !!projectName && isEditMode;

  // Live project role bindings for the member (reactively updated when IAM policy changes).
  // Active bindings come first, expired ones last; original order is preserved within each group.
  const liveProjectRoleBindings = useVueState(() => {
    if (!isProjectEditMode || !member || !projectName) return [];
    const policy = projectIamPolicyStore.getProjectIamPolicy(projectName);
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
    id: crypto.randomUUID(),
    role: "",
    reason: "",
    expirationDays: 7,
    expirationTimestampInMS: computeExpirationTimestamp(7),
    databaseMode: "ALL",
    databaseResources: [],
    celExpression: "",
    environments: [],
  }));

  useEscapeKey(true, onClose);

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
    const roleName = displayRoleTitle(roleBinding.role);
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
      const policy = structuredClone(
        projectIamPolicyStore.getProjectIamPolicy(projectName)
      );
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
      await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
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
        const policy = structuredClone(
          projectIamPolicyStore.getProjectIamPolicy(projectName)
        );
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
              (form.databaseMode === "EXPRESSION" && form.celExpression !== "");
            if (form.databaseMode === "EXPRESSION" && form.celExpression) {
              const extraParts = stringifyConditionExpression({
                expirationTimestampInMS: form.expirationTimestampInMS,
                environments,
              });
              const fullExpression = extraParts
                ? `(${form.celExpression}) && ${extraParts}`
                : form.celExpression;
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
        await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
      } else {
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
        const policy = structuredClone(
          projectIamPolicyStore.getProjectIamPolicy(projectName)
        );
        for (const binding of policy.bindings) {
          binding.members = binding.members.filter((m) => m !== member.binding);
        }
        policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
        await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
      } else {
        await workspaceStore.patchIamPolicy([
          { member: member.binding, roles: [] },
        ]);
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
      !(
        roleHasDatabaseLimitation(form.role) &&
        form.databaseMode === "SELECT" &&
        form.databaseResources.length === 0
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
                            {displayRoleTitle(binding.role)}
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
  const workspaceStore = useWorkspaceV1Store();
  const actuatorStore = useActuatorV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  const userCountInIam = useVueState(() => actuatorStore.userCountInIam);
  const userCountLimit = useVueState(() => subscriptionStore.userCountLimit);
  const remainingUserCount = useMemo(
    () => Math.max(0, userCountLimit - userCountInIam),
    [userCountLimit, userCountInIam]
  );

  const projectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : undefined;
  const project = useVueState(() =>
    projectName ? projectStore.getProjectByName(projectName) : undefined
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
    subscriptionStore.hasFeature(PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW)
  );

  // Fetch project IAM policy on mount
  useEffect(() => {
    if (projectName) {
      projectIamPolicyStore.getOrFetchProjectIamPolicy(projectName);
    }
  }, [projectName, projectIamPolicyStore]);

  const projectIamPolicy = useVueState(() =>
    projectName
      ? projectIamPolicyStore.getProjectIamPolicy(projectName)
      : undefined
  );

  // `useVueState` ensures we re-render whenever any reactive dep
  // `getMemberBindings(...)` reads from changes — IAM policies, but
  // also the group / user / service-account / workload-identity stores
  // it pulls metadata from. The result array reference changes on every
  // render; the table component handles that with content-based change
  // detection (a group-bindings signature) so its expand-cache only
  // resets on real membership changes.
  const memberBindings = useVueState(() =>
    getMemberBindings({
      policies:
        projectName && projectIamPolicy
          ? [{ level: "PROJECT" as const, policy: projectIamPolicy }]
          : [
              {
                level: "WORKSPACE" as const,
                policy: workspaceStore.workspaceIamPolicy,
              },
            ],
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
          await projectIamPolicyStore.updateProjectIamPolicy(
            projectName,
            policy
          );
        } else {
          await workspaceStore.patchIamPolicy(
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
        await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
      } else {
        await workspaceStore.patchIamPolicy([
          { member: binding.binding, roles: [] },
        ]);
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
