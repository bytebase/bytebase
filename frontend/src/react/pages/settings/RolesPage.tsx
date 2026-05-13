import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { cloneDeep, sortBy, uniq } from "lodash-es";
import { Pencil, Plus, Trash2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { LearnMoreLink } from "@/react/components/LearnMoreLink";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import {
  ResourceIdField,
  type ResourceIdFieldRef,
} from "@/react/components/ResourceIdField";
import { RoleSelect } from "@/react/components/RoleSelect";
import { Alert } from "@/react/components/ui/alert";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
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
import { Textarea } from "@/react/components/ui/textarea";
import { BlockTooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useRoleStore,
  useSubscriptionV1Store,
  useWorkspaceV1Store,
} from "@/store";
import { roleNamePrefix } from "@/store/modules/v1/common";
import {
  BASIC_WORKSPACE_PERMISSIONS,
  PERMISSIONS,
  PRESET_ROLES,
} from "@/types";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  displayRoleDescription,
  displayRoleTitle,
  extractRoleResourceName,
  hasWorkspacePermissionV2,
  isCustomRole,
} from "@/utils";
import { extractGrpcErrorMessage, getErrorCode } from "@/utils/connect";

// ============================================================
// PermissionTransfer
// ============================================================

function PermissionTransfer({
  value,
  onChange,
  disabled,
}: {
  value: string[];
  onChange: (permissions: string[]) => void;
  disabled: boolean;
}) {
  const { t } = useTranslation();
  const [sourceFilter, setSourceFilter] = useState("");

  const allPermissions = useMemo(() => [...PERMISSIONS].sort(), []);

  const valueSet = useMemo(() => new Set(value), [value]);

  const sourceItems = useMemo(() => {
    const items = allPermissions.filter((p) => !valueSet.has(p));
    if (!sourceFilter.trim()) return items;
    const kw = sourceFilter.trim().toLowerCase();
    return items.filter((p) => p.toLowerCase().includes(kw));
  }, [allPermissions, valueSet, sourceFilter]);

  const totalSourceCount = allPermissions.length - value.length;

  const selectItem = (p: string) => {
    if (disabled) return;
    onChange(uniq([...value, p]));
  };

  const removeItem = (p: string) => {
    if (disabled) return;
    onChange(value.filter((v) => v !== p));
  };

  const selectAll = () => {
    if (disabled) return;
    onChange(uniq([...value, ...sourceItems]));
    setSourceFilter("");
  };

  const clearAll = () => {
    if (disabled) return;
    onChange([]);
  };

  return (
    <div className="flex h-[28rem] border rounded-sm overflow-hidden">
      {/* Source */}
      <div className="flex-1 flex flex-col border-r min-w-0">
        <div className="flex items-center gap-x-2 px-3 py-2 border-b">
          <button
            type="button"
            className="text-sm text-main hover:text-accent disabled:opacity-50"
            disabled={disabled || sourceItems.length === 0}
            onClick={selectAll}
          >
            {t("common.select-all")}
          </button>
          <span className="text-xs text-control-light">
            {t("common.total-n-items", { n: totalSourceCount })}
          </span>
        </div>
        <div className="px-3 py-2 border-b">
          <SearchInput
            placeholder={t("common.search")}
            value={sourceFilter}
            onChange={(e) => setSourceFilter(e.target.value)}
            className="h-8"
            wrapperClassName="flex-none"
            disabled={disabled}
          />
        </div>
        <div className="flex-1 overflow-auto">
          {sourceItems.map((p) => (
            <div
              key={p}
              className={`group flex items-center justify-between px-3 py-1.5 text-sm hover:bg-gray-50 ${disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
              onClick={() => selectItem(p)}
            >
              <span className="truncate">{p}</span>
              {!disabled && (
                <Plus className="h-3.5 w-3.5 shrink-0 text-control-light opacity-0 group-hover:opacity-100 group-hover:text-accent" />
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Target */}
      <div className="flex-1 flex flex-col min-w-0">
        <div className="flex items-center gap-x-2 px-3 py-2 border-b">
          <button
            type="button"
            className="text-sm text-main hover:text-accent disabled:opacity-50"
            disabled={disabled || value.length === 0}
            onClick={clearAll}
          >
            {t("common.clear")}
          </button>
          <span className="text-xs text-control-light">
            {t("common.n-items-selected", { n: value.length })}
          </span>
        </div>
        <div className="flex-1 overflow-auto">
          {[...value].sort().map((p) => (
            <div
              key={p}
              className={`group flex items-center justify-between px-3 py-1.5 text-sm hover:bg-gray-50 ${disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
              onClick={() => removeItem(p)}
            >
              <span className="truncate">{p}</span>
              {!disabled && (
                <X className="h-3.5 w-3.5 shrink-0 text-control-light opacity-0 group-hover:opacity-100 group-hover:text-error" />
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ============================================================
// ImportPermissionModal
// ============================================================

function ImportPermissionModal({
  onCancel,
  onImport,
}: {
  onCancel: () => void;
  onImport: (permissions: string[]) => void;
}) {
  const { t } = useTranslation();
  const roleStore = useRoleStore();
  const roleList = useVueState(() => [...roleStore.roleList]);
  const [selectedRoleName, setSelectedRoleName] = useState<string>("");

  const selectedRole = useMemo(
    () => roleList.find((r) => r.name === selectedRoleName),
    [roleList, selectedRoleName]
  );

  const permissions = selectedRole?.permissions ?? [];

  return (
    <Dialog open onOpenChange={(nextOpen) => !nextOpen && onCancel()}>
      <DialogContent className="w-[28rem] max-w-[calc(100vw-2rem)] p-6">
        <DialogTitle>{t("role.import-from-role")}</DialogTitle>

        <div className="mt-4 flex flex-col gap-y-3">
          <div>
            <label className="textlabel mb-1 block">
              {t("role.select-role")}
            </label>
            <RoleSelect
              value={selectedRoleName ? [selectedRoleName] : []}
              onChange={(roles) => setSelectedRoleName(roles[0] ?? "")}
              multiple={false}
            />
          </div>

          {selectedRole && (
            <>
              <p className="textinfolabel">
                {displayRoleDescription(selectedRole.name)}
              </p>
              <div>
                <label className="textlabel mb-1 block">
                  {t("common.permissions")} ({permissions.length})
                </label>
                <div className="max-h-40 overflow-auto border rounded-sm p-2">
                  {permissions.map((p) => (
                    <p key={p} className="text-sm leading-5">
                      {p}
                    </p>
                  ))}
                </div>
              </div>
            </>
          )}

          <div className="flex justify-end gap-x-2 mt-2">
            <Button variant="outline" onClick={onCancel}>
              {t("common.cancel")}
            </Button>
            <Button
              disabled={!selectedRole}
              onClick={() => onImport(permissions)}
            >
              {t("common.import")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ============================================================
// DeleteConfirmModal
// ============================================================

function DeleteConfirmModal({
  roleName,
  occupiedResources,
  onConfirm,
  onCancel,
}: {
  roleName: string;
  occupiedResources: string[];
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const { t } = useTranslation();
  const canDelete = occupiedResources.length === 0;

  return (
    <AlertDialog open onOpenChange={(nextOpen) => !nextOpen && onCancel()}>
      <AlertDialogContent>
        <AlertDialogTitle>
          {t("common.delete")} - {displayRoleTitle(roleName)}
        </AlertDialogTitle>

        {occupiedResources.length > 0 ? (
          <div className="mt-4">
            <Alert
              variant="warning"
              description={
                <>
                  <p className="mb-2">
                    {t("resource.delete-warning-with-resources", {
                      name: displayRoleTitle(roleName),
                    })}
                  </p>
                  <ul className="list-disc pl-4 text-sm">
                    {occupiedResources.map((r) => (
                      <li key={r}>{r}</li>
                    ))}
                  </ul>
                  <p className="mt-2">{t("resource.delete-warning-retry")}</p>
                </>
              }
            />
          </div>
        ) : (
          <AlertDialogDescription className="mt-2">
            {t("resource.delete-warning", {
              name: displayRoleTitle(roleName),
            })}
          </AlertDialogDescription>
        )}

        <div className="mt-6 flex justify-end gap-x-2">
          <Button variant="outline" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          {canDelete && (
            <Button variant="destructive" onClick={onConfirm}>
              {t("common.delete")}
            </Button>
          )}
        </div>
      </AlertDialogContent>
    </AlertDialog>
  );
}

// ============================================================
// RoleDrawer
// ============================================================

function RoleSheet({
  open,
  role,
  mode,
  hasCustomRoleFeature,
  onClose,
  onShowFeatureModal,
}: {
  open: boolean;
  role: Role | undefined;
  mode: "ADD" | "EDIT";
  hasCustomRoleFeature: boolean;
  onClose: () => void;
  onShowFeatureModal: () => void;
}) {
  const { t } = useTranslation();
  const roleStore = useRoleStore();

  const [editRole, setEditRole] = useState<Role>(
    create(RoleSchema, { permissions: [...BASIC_WORKSPACE_PERMISSIONS] })
  );
  const [dirty, setDirty] = useState(false);
  const [loading, setLoading] = useState(false);
  const [showImportModal, setShowImportModal] = useState(false);
  const [resourceId, setResourceId] = useState("");
  const [resourceIdValid, setResourceIdValid] = useState(true);
  const resourceIdFieldRef = useRef<ResourceIdFieldRef>(null);

  const isBuiltin = useMemo(
    () => (editRole.name ? !isCustomRole(editRole.name) : false),
    [editRole.name]
  );

  const missedBasicPermissions = useMemo(
    () =>
      BASIC_WORKSPACE_PERMISSIONS.filter(
        (p) => !editRole.permissions.includes(p)
      ),
    [editRole.permissions]
  );

  const canCreate = hasWorkspacePermissionV2("bb.roles.create");
  const canUpdate = hasWorkspacePermissionV2("bb.roles.update");
  const allowSave = useMemo(() => {
    if (!dirty) return false;
    if (!editRole.title || editRole.title.length === 0) return false;
    if (mode === "ADD") {
      if (!resourceId) return false;
      if (!resourceIdValid) return false;
    }
    if (editRole.permissions.length === 0) return false;
    return true;
  }, [
    dirty,
    editRole.title,
    editRole.permissions,
    mode,
    resourceId,
    resourceIdValid,
  ]);

  const validateResourceId = useCallback(
    async (val: string) => {
      if (
        val &&
        roleStore.roleList.find((r) => r.name === `${roleNamePrefix}${val}`)
      ) {
        return [
          {
            type: "error" as const,
            message: t("resource-id.validation.duplicated", {
              resource: t("role.self"),
            }),
          },
        ];
      }
      return [];
    },
    [roleStore.roleList, t]
  );

  // Sync incoming role prop to local state
  useEffect(() => {
    if (role) {
      const cloned = cloneDeep(role);
      if (!cloned.title) {
        cloned.title = extractRoleResourceName(cloned.name);
      }
      setEditRole(cloned);
      setResourceId(extractRoleResourceName(cloned.name));
      setResourceIdValid(true);
      requestAnimationFrame(() => {
        setDirty(false);
      });
    }
  }, [role]);

  const handleTitleChange = (title: string) => {
    setEditRole((prev) => {
      const next = cloneDeep(prev);
      next.title = title;
      return next;
    });
    setDirty(true);
  };

  const handleResourceIdChange = (val: string) => {
    setResourceId(val);
    setEditRole((prev) => {
      const next = cloneDeep(prev);
      next.name = `${roleNamePrefix}${val}`;
      return next;
    });
    setDirty(true);
  };

  const handleDescriptionChange = (description: string) => {
    setEditRole((prev) => {
      const next = cloneDeep(prev);
      next.description = description;
      return next;
    });
    setDirty(true);
  };

  const handlePermissionsChange = (permissions: string[]) => {
    setEditRole((prev) => {
      const next = cloneDeep(prev);
      next.permissions = permissions;
      return next;
    });
    setDirty(true);
  };

  const addMissingPermissions = () => {
    handlePermissionsChange(
      uniq([...editRole.permissions, ...missedBasicPermissions])
    );
  };

  const handleImportPermissions = (permissions: string[]) => {
    handlePermissionsChange(uniq([...editRole.permissions, ...permissions]));
    setShowImportModal(false);
  };

  const handleSave = async () => {
    if (!hasCustomRoleFeature) {
      onShowFeatureModal();
      onClose();
      return;
    }

    setLoading(true);
    try {
      await roleStore.upsertRole(editRole);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: mode === "ADD" ? t("common.added") : t("common.updated"),
      });
      onClose();
    } catch (error) {
      if (error instanceof ConnectError && error.code === Code.AlreadyExists) {
        resourceIdFieldRef.current?.addValidationError(error.message);
      } else {
        throw error;
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("common.role.self")}</SheetTitle>
        </SheetHeader>

        <SheetBody>
          <div className="flex flex-col gap-y-5">
            {/* Title */}
            <div className="flex flex-col gap-y-1.5">
              <label className="text-sm font-medium text-main">
                {t("role.title")}
                <span className="ml-0.5 text-error">*</span>
              </label>
              <Input
                value={editRole.title}
                onChange={(e) => handleTitleChange(e.target.value)}
                placeholder={t("role.setting.title-placeholder")}
                maxLength={200}
                disabled={isBuiltin}
                className={
                  dirty && editRole.title?.length === 0 ? "border-error" : ""
                }
              />

              {/* Resource ID — inline below title */}
              {mode === "ADD" && (
                <ResourceIdField
                  ref={resourceIdFieldRef}
                  value={resourceId}
                  resourceName={t("role.self")}
                  resourceTitle={editRole.title}
                  suffix
                  validate={validateResourceId}
                  onChange={handleResourceIdChange}
                  onValidationChange={setResourceIdValid}
                />
              )}
              {mode === "EDIT" && !isBuiltin && (
                <ResourceIdField
                  value={extractRoleResourceName(editRole.name)}
                  resourceName={t("role.self")}
                  readonly
                />
              )}
            </div>

            {/* Description */}
            <div className="flex flex-col gap-y-1.5">
              <label className="text-sm font-medium text-main">
                {t("common.description")}
              </label>
              <Textarea
                value={editRole.description}
                onChange={(e) => handleDescriptionChange(e.target.value)}
                placeholder={t("role.setting.description-placeholder")}
                maxLength={1000}
                disabled={isBuiltin}
                className="min-h-[80px]"
              />
            </div>

            {/* Permissions */}
            <div className="flex flex-col gap-y-2">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium text-main">
                  {t("common.permissions")}
                  <span className="ml-0.5 text-error">*</span>
                </label>
                {!isBuiltin && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowImportModal(true)}
                  >
                    <Plus className="w-4 h-4 mr-1" />
                    {t("role.import-from-role")}
                  </Button>
                )}
              </div>

              {missedBasicPermissions.length > 0 && !isBuiltin && (
                <Alert
                  variant="error"
                  title={t("common.missing-required-permission")}
                  description={
                    <>
                      <p>{t("common.required-workspace-permission")}</p>
                      <ul className="list-disc pl-4 mt-1">
                        {missedBasicPermissions.map((p) => (
                          <li key={p}>{p}</li>
                        ))}
                      </ul>
                      <div className="mt-2">
                        <Button size="sm" onClick={addMissingPermissions}>
                          {t("common.add-permissions")}
                        </Button>
                      </div>
                    </>
                  }
                />
              )}

              <PermissionTransfer
                value={editRole.permissions}
                onChange={handlePermissionsChange}
                disabled={isBuiltin}
              />
            </div>
          </div>
        </SheetBody>

        {!isBuiltin && (
          <SheetFooter>
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button
              disabled={
                !allowSave || (mode === "ADD" ? !canCreate : !canUpdate)
              }
              onClick={handleSave}
            >
              {mode === "ADD" ? t("common.create") : t("common.update")}
            </Button>
          </SheetFooter>
        )}

        {loading && (
          <div className="absolute inset-0 z-10 bg-background/50 flex items-center justify-center">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        )}

        {/* Import modal */}
        {showImportModal && (
          <ImportPermissionModal
            onCancel={() => setShowImportModal(false)}
            onImport={handleImportPermissions}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

// ============================================================
// RolesPage (main)
// ============================================================

export function RolesPage() {
  const { t } = useTranslation();
  const roleStore = useRoleStore();
  const workspaceStore = useWorkspaceV1Store();
  const subscriptionStore = useSubscriptionV1Store();

  const [ready, setReady] = useState(false);
  const [detailRole, setDetailRole] = useState<Role | undefined>();
  const [detailMode, setDetailMode] = useState<"ADD" | "EDIT">("ADD");
  const [showFeatureModal, setShowFeatureModal] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Role | undefined>();
  const [deleteResources, setDeleteResources] = useState<string[]>([]);

  const hasCustomRoleFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_CUSTOM_ROLES)
  );

  const canCreate = hasWorkspacePermissionV2("bb.roles.create");
  const canDelete = hasWorkspacePermissionV2("bb.roles.delete");

  const roleList = useVueState(() => [...roleStore.roleList]);

  const filteredRoleList = useMemo(() => {
    return sortBy(roleList, (role) =>
      PRESET_ROLES.includes(role.name)
        ? PRESET_ROLES.indexOf(role.name)
        : roleList.length
    );
  }, [roleList]);

  // Fetch roles on mount and handle query param
  useEffect(() => {
    roleStore
      .fetchRoleList()
      .then(() => {
        const urlParams = new URLSearchParams(window.location.search);
        const name = urlParams.get("role");
        if (name?.startsWith(roleNamePrefix)) {
          const role = roleStore.getRoleByName(name);
          if (role) {
            setDetailRole(role);
            setDetailMode("EDIT");
          }
        }
      })
      .finally(() => setReady(true));
  }, []);

  const addRole = () => {
    setDetailRole(
      create(RoleSchema, { permissions: [...BASIC_WORKSPACE_PERMISSIONS] })
    );
    setDetailMode("ADD");
  };

  const editRole = (role: Role) => {
    setDetailRole(role);
    setDetailMode("EDIT");
  };

  const handleDeleteRole = (role: Role) => {
    if (!hasCustomRoleFeature) {
      setShowFeatureModal(true);
      return;
    }

    const usersWithRole = [
      ...(workspaceStore.roleMapToUsers.get(role.name) ?? new Set()),
    ];
    setDeleteResources(usersWithRole);
    setDeleteTarget(role);
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    try {
      await roleStore.deleteRole(deleteTarget);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
      setDeleteTarget(undefined);
    } catch (error) {
      if (getErrorCode(error) === Code.FailedPrecondition) {
        const message = extractGrpcErrorMessage(error);
        const resources =
          message.split("used by resources: ")[1]?.split(",") ?? [];
        if (resources.length > 0) {
          setDeleteResources(resources);
        }
      }
    }
  };

  return (
    <div className="w-full px-4 py-4 flex flex-col gap-y-4">
      {showFeatureModal && (
        <FeatureAttention feature={PlanFeature.FEATURE_CUSTOM_ROLES} />
      )}

      <div className="textinfolabel">
        {t("role.setting.description")}{" "}
        <LearnMoreLink
          href="https://docs.bytebase.com/administration/roles?source=console"
          className="text-accent"
        />
      </div>

      <div className="w-full flex justify-end">
        <PermissionGuard permissions={["bb.roles.create"]}>
          <Button disabled={!canCreate} onClick={addRole}>
            <FeatureBadge
              feature={PlanFeature.FEATURE_CUSTOM_ROLES}
              clickable={false}
              className="mr-1 text-white inline-flex"
              fallback={<Plus className="h-4 w-4 mr-1" />}
            />
            {t("common.create")}
          </Button>
        </PermissionGuard>
      </div>

      {/* Roles Table */}
      {ready ? (
        <div className="border rounded-sm overflow-hidden">
          <Table className="table-fixed">
            <colgroup>
              <col className="w-1/3" />
              <col />
              <col className="w-24" />
            </colgroup>
            <TableHeader>
              <TableRow className="bg-control-bg">
                <TableHead className="whitespace-nowrap">
                  {t("role.title")}
                </TableHead>
                <TableHead className="whitespace-nowrap">
                  {t("common.description")}
                </TableHead>
                <TableHead className="text-right whitespace-nowrap" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredRoleList.map((role) => (
                <TableRow key={role.name}>
                  <TableCell className="whitespace-nowrap">
                    <span>{displayRoleTitle(role.name)}</span>
                    {!isCustomRole(role.name) && (
                      <Badge variant="secondary" className="ml-2 text-xs">
                        {t("common.system")}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-control-light overflow-hidden">
                    <BlockTooltip content={displayRoleDescription(role.name)}>
                      <span className="block truncate">
                        {displayRoleDescription(role.name)}
                      </span>
                    </BlockTooltip>
                  </TableCell>
                  <TableCell>
                    {isCustomRole(role.name) && (
                      <div className="flex justify-end gap-x-2">
                        {canDelete && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-error hover:text-error"
                            onClick={() => handleDeleteRole(role)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => editRole(role)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                      </div>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="flex items-center justify-center h-32">
          <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
        </div>
      )}

      {/* Role Sheet */}
      <RoleSheet
        open={detailRole !== undefined}
        role={detailRole}
        mode={detailMode}
        hasCustomRoleFeature={hasCustomRoleFeature}
        onClose={() => setDetailRole(undefined)}
        onShowFeatureModal={() => setShowFeatureModal(true)}
      />

      {/* Delete confirm */}
      {deleteTarget && (
        <DeleteConfirmModal
          roleName={deleteTarget.name}
          occupiedResources={deleteResources}
          onConfirm={confirmDelete}
          onCancel={() => setDeleteTarget(undefined)}
        />
      )}
    </div>
  );
}
