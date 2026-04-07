import { create } from "@bufbuild/protobuf";
import {
  Building2,
  Check,
  ChevronDown,
  ChevronRight,
  Info,
  Pencil,
  Plus,
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
import type { MemberBinding } from "@/components/Member/types";
import { getMemberBindings } from "@/components/Member/utils";
import {
  roleHasDatabaseLimitation,
  roleHasEnvironmentLimitation,
} from "@/components/ProjectMember/utils";
import { AccountMultiSelect } from "@/react/components/AccountMultiSelect";
import { DatabaseResourceSelector as DatabaseResourceSelectorComponent } from "@/react/components/DatabaseResourceSelector";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { RoleSelect } from "@/react/components/RoleSelect";
import { UserAvatar } from "@/react/components/UserAvatar";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/react/components/ui/alert";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useEnvironmentV1Store,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useRoleStore,
  useSubscriptionV1Store,
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
import {
  displayRoleTitle,
  formatAbsoluteDateTime,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  sortRoles,
} from "@/utils";
import {
  buildConditionExpr,
  convertFromExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";

const EMPTY_ROLE_SET = new Set<string>();

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

  const selectableBindings = useMemo(
    () =>
      bindings.filter(
        (b) => scope !== "project" || b.projectRoleBindings.length > 0
      ),
    [bindings, scope]
  );

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
                    disabled={isSelectDisabled(mb)}
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
                  {scope === "project"
                    ? sortRoles(mb.projectRoleBindings.map((b) => b.role)).map(
                        (role) => (
                          <Badge key={role} className="text-xs gap-x-1">
                            {displayRoleTitle(role)}
                          </Badge>
                        )
                      )
                    : sortRoles([...mb.workspaceLevelRoles]).map((role) => (
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
  scope,
}: {
  bindings: MemberBinding[];
  allowEdit: boolean;
  onUpdateBinding: (binding: MemberBinding) => void;
  onRevokeBinding: (binding: MemberBinding) => void;
  scope: "workspace" | "project";
}) {
  const { t } = useTranslation();
  const [expandedRoles, setExpandedRoles] = useState<Set<string>>(new Set());
  const initializedRef = useRef(false);

  const roleToBindings = useMemo(() => {
    const map = new Map<string, MemberBinding[]>();
    for (const mb of bindings) {
      const roles =
        scope === "project"
          ? mb.projectRoleBindings.map((b) => b.role)
          : [...mb.workspaceLevelRoles];
      for (const role of roles) {
        if (!map.has(role)) map.set(role, []);
        map.get(role)!.push(mb);
      }
    }
    const sortedRoles = sortRoles([...map.keys()]);
    return sortedRoles.map((role) => ({
      role,
      members: map.get(role) ?? [],
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

type DatabaseMode = "ALL" | "EXPRESSION" | "SELECT";

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
// EnvironmentMultiSelect
// ============================================================

function EnvironmentMultiSelect({
  value,
  onChange,
}: {
  value: string[];
  onChange: (envs: string[]) => void;
}) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const environmentList = useVueState(
    () => environmentStore.environmentList ?? []
  );
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback(() => setOpen(false), []);
  useClickOutside(containerRef, open, handleClickOutside);

  const toggle = (name: string) => {
    onChange(
      value.includes(name) ? value.filter((v) => v !== name) : [...value, name]
    );
  };

  const remove = (name: string) => {
    onChange(value.filter((v) => v !== name));
  };

  return (
    <div ref={containerRef} className="relative">
      <div
        className={cn(
          "flex items-center flex-wrap gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm cursor-pointer",
          open && "ring-2 ring-accent border-accent"
        )}
        onClick={() => setOpen(!open)}
      >
        {value.length === 0 && (
          <span className="text-control-placeholder">
            {t("environment.select")}
          </span>
        )}
        {value.map((name) => (
          <span
            key={name}
            className="inline-flex items-center gap-x-1 rounded-xs border border-control-border px-1 py-0.5 text-xs"
          >
            <EnvironmentLabel environmentName={name} className="text-xs" />
            <button
              type="button"
              className="text-control-light hover:text-control"
              onClick={(e) => {
                e.stopPropagation();
                remove(name);
              }}
            >
              <X className="h-3 w-3" />
            </button>
          </span>
        ))}
        <ChevronDown className="ml-auto h-4 w-4 shrink-0 text-control-light" />
      </div>

      {open && (
        <div className="absolute z-50 mt-1 w-full bg-white border border-control-border rounded-sm shadow-lg max-h-60 overflow-auto">
          {environmentList.length === 0 && (
            <div className="px-3 py-2 text-sm text-control-light">
              {t("common.no-data")}
            </div>
          )}
          {environmentList.map((env) => {
            const selected = value.includes(env.name);
            return (
              <div
                key={env.name}
                className="flex items-center gap-x-2 px-3 py-1.5 text-sm hover:bg-gray-50 cursor-pointer"
                onClick={() => toggle(env.name)}
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
                <EnvironmentLabel environment={env} />
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
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
  const showEnvironments = useMemo(
    () => form.role && roleHasEnvironmentLimitation(form.role),
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
          <div className="max-h-32 overflow-auto border rounded-sm bg-gray-50 p-2">
            <div className="flex flex-wrap gap-1">
              {permissions.map((perm) => (
                <span
                  key={perm}
                  className="inline-block rounded-xs bg-gray-200 px-1.5 py-0.5 text-xs text-control-light"
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
      {showEnvironments && (
        <div className="flex flex-col gap-y-2">
          <div>
            <label className="block text-sm font-medium text-control">
              {t("common.environments")}
            </label>
            <span className="text-xs text-control-light">
              {t("project.members.allow-ddl")}
            </span>
          </div>
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
                    ? "bg-accent text-white border-accent"
                    : "bg-white text-control border-control-border hover:bg-gray-50"
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

  const isEditMode = !!member;
  const isProjectCreateMode = !!projectName && !isEditMode;
  const isProjectEditMode = !!projectName && isEditMode;

  // Live project role bindings for the member (reactively updated when IAM policy changes)
  const liveProjectRoleBindings = useVueState(() => {
    if (!isProjectEditMode || !member || !projectName) return [];
    const policy = projectIamPolicyStore.getProjectIamPolicy(projectName);
    return policy.bindings.filter((b) => b.members.includes(member.binding));
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

  const getEnvironmentLimitation = useCallback((binding: Binding): string[] => {
    if (!binding.parsedExpr) return [];
    return convertFromExpr(binding.parsedExpr).environments ?? [];
  }, []);

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
            const environments = roleHasEnvironmentLimitation(form.role)
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
        <div className="fixed inset-0 z-40 bg-black/30" onClick={onClose} />
        <div
          role="dialog"
          aria-modal="true"
          className="fixed inset-y-0 right-0 z-50 w-[40rem] max-w-[100vw] bg-white shadow-xl flex flex-col"
        >
          {/* Header */}
          <div className="flex items-center justify-between px-6 py-4 border-b">
            <h2 className="text-lg font-medium">
              {t("project.members.edit-member", {
                member: `${member.title} (${memberEmail})`,
              })}
            </h2>
            <div className="flex items-center gap-x-2">
              <Button
                variant="outline"
                onClick={() => setShowNestedGrant(true)}
              >
                <Plus className="h-4 w-4 mr-1" />
                {t("settings.members.grant-access")}
              </Button>
              <Button variant="ghost" size="icon" onClick={onClose}>
                <X className="h-5 w-5" />
              </Button>
            </div>
          </div>

          {/* Body — Role Bindings */}
          <div className="flex-1 overflow-auto px-6 py-6">
            <div className="flex flex-col gap-y-6">
              {liveProjectRoleBindings.length === 0 && (
                <div className="text-center text-control-light py-8">
                  {t("common.no-data")}
                </div>
              )}
              {liveProjectRoleBindings.map((binding, idx) => {
                const rows = getSingleBindingRows(binding);
                const envs = roleHasEnvironmentLimitation(binding.role)
                  ? getEnvironmentLimitation(binding)
                  : [];
                const showEnvBanner = roleHasEnvironmentLimitation(
                  binding.role
                );
                return (
                  <div
                    key={`${binding.role}-${idx}`}
                    className="border rounded-sm"
                  >
                    {/* Role header */}
                    <div className="flex items-center justify-between px-4 py-3 bg-gray-50 border-b">
                      <span className="font-medium text-sm">
                        {displayRoleTitle(binding.role)}
                      </span>
                      <div className="flex items-center gap-x-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          title={t("common.edit")}
                          onClick={() => setShowNestedGrant(true)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title={t("common.delete")}
                          disabled={isRequesting}
                          onClick={() => handleDeleteRole(binding)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>

                    {/* Environment info banner */}
                    {showEnvBanner && (
                      <div className="mx-4 mt-3 flex items-start gap-x-2 rounded-sm bg-blue-50 border border-blue-200 px-3 py-2 text-xs text-blue-700">
                        <Info className="h-4 w-4 shrink-0 mt-0.5" />
                        <div>
                          {envs.length > 0 ? (
                            <>
                              <span>{t("project.members.allow-ddl")}</span>
                              <div className="flex flex-wrap gap-1 mt-1">
                                {envs.map((env) => (
                                  <Badge
                                    key={env}
                                    variant="secondary"
                                    className="text-xs"
                                  >
                                    <EnvironmentLabel
                                      environmentName={env}
                                      className="text-xs"
                                    />
                                  </Badge>
                                ))}
                              </div>
                            </>
                          ) : (
                            <span>
                              {t(
                                "project.members.disallow-ddl-all-environments"
                              )}
                            </span>
                          )}
                        </div>
                      </div>
                    )}

                    {/* Database resources table */}
                    <div className="p-4">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b">
                            <th className="px-2 py-1.5 text-left font-medium text-control-light">
                              {t("common.database")}
                            </th>
                            <th className="px-2 py-1.5 text-left font-medium text-control-light">
                              {t("common.schema")}
                            </th>
                            <th className="px-2 py-1.5 text-left font-medium text-control-light">
                              {t("common.table")}
                            </th>
                            <th className="px-2 py-1.5 text-left font-medium text-control-light">
                              {t("common.expiration")}
                            </th>
                          </tr>
                        </thead>
                        <tbody>
                          {rows.map((row, rowIdx) => (
                            <tr
                              key={rowIdx}
                              className="border-b last:border-b-0"
                            >
                              <td className="px-2 py-1.5 text-sm">
                                {row.databaseResource?.databaseFullName ?? "*"}
                              </td>
                              <td className="px-2 py-1.5 text-sm">
                                {row.databaseResource?.schema ?? "*"}
                              </td>
                              <td className="px-2 py-1.5 text-sm">
                                {row.databaseResource?.table ?? "*"}
                              </td>
                              <td className="px-2 py-1.5 text-sm">
                                {row.expiration
                                  ? formatAbsoluteDateTime(
                                      row.expiration.getTime()
                                    )
                                  : t("project.members.never-expires")}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-between px-6 py-4 border-t">
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
          </div>
        </div>

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

export function MembersPage({ projectId }: { projectId?: string }) {
  const { t } = useTranslation();
  const workspaceStore = useWorkspaceV1Store();
  const actuatorStore = useActuatorV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  const activeUserCount = useVueState(() => actuatorStore.activeUserCount);
  const userCountLimit = useVueState(() => subscriptionStore.userCountLimit);
  const remainingUserCount = useMemo(
    () => Math.max(0, userCountLimit - activeUserCount),
    [userCountLimit, activeUserCount]
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

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {!projectName && remainingUserCount <= 3 && (
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
      {projectName && (
        <div className="textinfolabel px-4 pt-4">
          {t("project.members.description")}{" "}
          <a
            href="https://docs.bytebase.com/administration/roles/?source=console#project-roles"
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent hover:underline"
          >
            {t("common.learn-more")}
          </a>
        </div>
      )}

      <div className="flex items-center justify-between gap-x-2 mb-4">
        <SearchInput
          placeholder={t("settings.members.search-member")}
          value={memberSearchText}
          onChange={(e) => setMemberSearchText(e.target.value)}
        />
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
