import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Copy, KeyRound, Plus, Trash2, Undo2 } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { RoleSelect } from "@/react/components/RoleSelect";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
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
import { Tooltip } from "@/react/components/ui/tooltip";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  ensureServiceAccountFullName,
  pushNotification,
  useActuatorV1Store,
  useProjectV1Store,
  useWorkspaceV1Store,
} from "@/store";
import {
  serviceAccountToUser,
  useServiceAccountStore,
} from "@/store/modules/serviceAccount";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useProjectIamPolicyStore } from "@/store/modules/v1/projectIamPolicy";
import {
  getServiceAccountNameInBinding,
  getServiceAccountSuffix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

function execCommandCopy(text: string): boolean {
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  try {
    return document.execCommand("copy");
  } catch {
    return false;
  } finally {
    document.body.removeChild(textarea);
  }
}

async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // fall through to execCommand fallback
    }
  }
  return execCommandCopy(text);
}

// ============================================================
// ServiceAccountTable
// ============================================================

function ServiceAccountTable({
  users,
  project,
  onUserUpdated,
  onUserSelected,
}: {
  users: User[];
  project?: Project;
  onUserUpdated: (user: User) => void;
  onUserSelected?: (user: User) => void;
}) {
  const { t } = useTranslation();
  const serviceAccountStore = useServiceAccountStore();

  const handleDeactivate = async (user: User) => {
    const confirmed = window.confirm(
      t("settings.members.action.deactivate-confirm-title")
    );
    if (!confirmed) return;

    try {
      await serviceAccountStore.deleteServiceAccount(
        ensureServiceAccountFullName(user.email)
      );
      const updated = create(
        (await import("@/types/proto-es/v1/user_service_pb")).UserSchema,
        { ...user, state: State.DELETED }
      );
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
    try {
      await serviceAccountStore.undeleteServiceAccount(
        ensureServiceAccountFullName(user.email)
      );
      const updated = create(
        (await import("@/types/proto-es/v1/user_service_pb")).UserSchema,
        { ...user, state: State.ACTIVE }
      );
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

  const [resetConfirmUser, setResetConfirmUser] = useState<User | undefined>();
  const [copiedKeys, setCopiedKeys] = useState<Set<string>>(new Set());

  const handleResetKey = async (user: User) => {
    setResetConfirmUser(undefined);
    try {
      const sa = await serviceAccountStore.updateServiceAccount(
        { name: ensureServiceAccountFullName(user.email) },
        create(FieldMaskSchema, { paths: ["service_key"] })
      );
      const updated = serviceAccountToUser(sa);
      onUserUpdated(updated);
      if (updated.serviceKey && (await copyToClipboard(updated.serviceKey))) {
        setCopiedKeys((prev) => new Set(prev).add(updated.name));
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("settings.members.service-key-copied"),
        });
      }
    } catch {
      // error shown by store
    }
  };

  const handleCopyKey = async (user: User) => {
    if (!(await copyToClipboard(user.serviceKey))) return;
    setCopiedKeys((prev) => new Set(prev).add(user.name));
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("settings.members.service-key-copied"),
    });
  };

  return (
    <div className="border rounded-sm overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="whitespace-nowrap">
              {t("settings.members.table.account")}
            </TableHead>
            <TableHead className="text-right whitespace-nowrap">
              {t("common.operations")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={2}
                className="py-8 text-center text-control-light text-sm"
              >
                {t("common.no-data")}
              </TableCell>
            </TableRow>
          ) : (
            users.map((user) => {
              const isDeleted = user.state === State.DELETED;
              const canOpenDetail =
                !!onUserSelected &&
                (project
                  ? hasProjectPermissionV2(project, "bb.serviceAccounts.get")
                  : hasWorkspacePermissionV2("bb.serviceAccounts.get"));

              return (
                <TableRow
                  key={user.name}
                  className={cn(
                    canOpenDetail &&
                      "cursor-pointer focus-visible:outline-none focus-visible:bg-control-bg"
                  )}
                  tabIndex={canOpenDetail ? 0 : undefined}
                  role={canOpenDetail ? "button" : undefined}
                  aria-label={
                    canOpenDetail ? user.title || user.email : undefined
                  }
                  onClick={
                    canOpenDetail ? () => onUserSelected(user) : undefined
                  }
                  onKeyDown={
                    canOpenDetail
                      ? (e) => {
                          if (e.key === "Enter" || e.key === " ") {
                            e.preventDefault();
                            onUserSelected(user);
                          }
                        }
                      : undefined
                  }
                >
                  {/* Account column */}
                  <TableCell>
                    <div className="flex items-center gap-x-3">
                      <UserAvatar title={user.title || user.email} />
                      <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-x-1.5">
                          <span
                            className={
                              isDeleted
                                ? "line-through text-control-light"
                                : "font-medium text-main"
                            }
                          >
                            {user.title}
                          </span>
                          <Badge className="text-xs px-1.5 py-0">
                            {t("settings.members.service-account")}
                          </Badge>
                        </div>
                        <span className="textinfolabel text-xs">
                          {user.email}
                        </span>
                      </div>
                      {!isDeleted && (
                        <div className="ml-auto text-xs shrink-0">
                          {user.serviceKey && !copiedKeys.has(user.name) ? (
                            <Button
                              variant="outline"
                              size="xs"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleCopyKey(user);
                              }}
                            >
                              <Copy className="h-3 w-3 mr-1" />
                              {t("settings.members.copy-service-key")}
                            </Button>
                          ) : resetConfirmUser?.name === user.name ? (
                            <div className="flex items-center gap-x-1">
                              <span className="text-xs text-error">
                                {t("settings.members.reset-service-key-alert")}
                              </span>
                              <Button
                                variant="destructive"
                                size="xs"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleResetKey(user);
                                }}
                              >
                                {t("common.reset")}
                              </Button>
                              <Button
                                variant="outline"
                                size="xs"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setResetConfirmUser(undefined);
                                }}
                              >
                                {t("common.cancel")}
                              </Button>
                            </div>
                          ) : (
                            <Button
                              variant="outline"
                              size="xs"
                              onClick={(e) => {
                                e.stopPropagation();
                                setResetConfirmUser(user);
                              }}
                            >
                              <KeyRound className="h-3 w-3 mr-1" />
                              {t("settings.members.reset-service-key")}
                            </Button>
                          )}
                        </div>
                      )}
                    </div>
                  </TableCell>

                  {/* Operations column — destructive/secondary actions only.
                      The row itself is clickable to open the detail sheet. */}
                  <TableCell>
                    <div className="flex justify-end gap-x-1">
                      {!isDeleted &&
                        (project
                          ? hasProjectPermissionV2(
                              project,
                              "bb.serviceAccounts.delete"
                            )
                          : hasWorkspacePermissionV2(
                              "bb.serviceAccounts.delete"
                            )) && (
                          <Tooltip
                            content={t(
                              "settings.members.action.deactivate-confirm-title"
                            )}
                          >
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-error hover:text-error"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleDeactivate(user);
                              }}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </Tooltip>
                        )}
                      {isDeleted &&
                        (project
                          ? hasProjectPermissionV2(
                              project,
                              "bb.serviceAccounts.undelete"
                            )
                          : hasWorkspacePermissionV2(
                              "bb.serviceAccounts.undelete"
                            )) && (
                          <Tooltip
                            content={t(
                              "settings.members.action.reactivate-confirm-title"
                            )}
                          >
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleRestore(user);
                              }}
                            >
                              <Undo2 className="h-4 w-4" />
                            </Button>
                          </Tooltip>
                        )}
                    </div>
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>
    </div>
  );
}

// ============================================================
// CreateServiceAccountDrawer
// ============================================================

interface CreateServiceAccountSheetProps {
  open: boolean;
  serviceAccount: ServiceAccount | undefined;
  project?: string;
  onClose: () => void;
  onCreated: (sa: ServiceAccount) => void;
  onUpdated: (sa: ServiceAccount) => void;
}

function CreateServiceAccountSheet(props: CreateServiceAccountSheetProps) {
  const { open, serviceAccount, onClose } = props;
  // Freeze the entity while open=false so the inner form stays visually
  // stable during the Sheet's close animation.
  const openEntityRef = useRef(serviceAccount);
  if (open) {
    openEntityRef.current = serviceAccount;
  }
  const stableServiceAccount = openEntityRef.current;
  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <ServiceAccountForm
          key={stableServiceAccount?.name ?? "new"}
          serviceAccount={stableServiceAccount}
          project={props.project}
          onClose={props.onClose}
          onCreated={props.onCreated}
          onUpdated={props.onUpdated}
        />
      </SheetContent>
    </Sheet>
  );
}

function ServiceAccountForm({
  serviceAccount,
  project,
  onClose,
  onCreated,
  onUpdated,
}: Omit<CreateServiceAccountSheetProps, "open">) {
  const { t } = useTranslation();
  const serviceAccountStore = useServiceAccountStore();
  const workspaceStore = useWorkspaceV1Store();
  const actuatorStore = useActuatorV1Store();
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  const projectEntity = useVueState(() =>
    project ? projectStore.getProjectByName(project) : undefined
  );

  const parent = useVueState(
    () => project ?? actuatorStore.workspaceResourceName
  );

  const isEditMode = !!serviceAccount && !!serviceAccount.email;
  const emailSuffix = useMemo(() => {
    const pid = project ? project.replace(/^projects\//, "") : "";
    return getServiceAccountSuffix(pid || undefined);
  }, [project]);

  // Capture initial values on mount — parent keys by serviceAccount so
  // these reflect the latest props.
  const initialTitle = serviceAccount?.title ?? "";
  const initialEmailPrefix = serviceAccount?.email
    ? serviceAccount.email.split("@")[0]
    : "";

  const [title, setTitle] = useState(initialTitle);
  const [emailPrefix, setEmailPrefix] = useState(initialEmailPrefix);
  const [roles, setRoles] = useState<string[]>([]);
  const [isRequesting, setIsRequesting] = useState(false);

  const isFormValid = isEditMode ? true : emailPrefix.trim().length > 0;

  // Dirty tracking — Update button disabled unless something changed.
  const isDirty = useMemo(() => {
    if (!isEditMode) return true;
    if (title !== initialTitle) return true;
    // Roles are not part of edit mode for service accounts (create-only).
    return false;
  }, [isEditMode, title, initialTitle]);

  const allowConfirm = isFormValid && isDirty;

  const requiredPermission = isEditMode
    ? "bb.serviceAccounts.update"
    : "bb.serviceAccounts.create";
  const hasPermission = projectEntity
    ? hasProjectPermissionV2(projectEntity, requiredPermission)
    : hasWorkspacePermissionV2(requiredPermission);

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

  const updateProjectIamPolicyForMember = async (
    projectName: string,
    member: string,
    newRoles: string[]
  ) => {
    const policy = structuredClone(
      projectIamPolicyStore.getProjectIamPolicy(projectName)
    );
    for (const binding of policy.bindings) {
      binding.members = binding.members.filter((m) => m !== member);
    }
    policy.bindings = policy.bindings.filter(
      (binding) => binding.members.length > 0
    );
    for (const role of newRoles) {
      const existing = policy.bindings.find((b) => b.role === role);
      if (existing) {
        if (!existing.members.includes(member)) {
          existing.members.push(member);
        }
      } else {
        policy.bindings.push(
          create(BindingSchema, { role, members: [member] })
        );
      }
    }
    await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
  };

  const handleCreate = async () => {
    const sa = await serviceAccountStore.createServiceAccount(
      emailPrefix.trim(),
      { title: title.trim() || emailPrefix.trim() },
      parent
    );

    if (roles.length > 0) {
      const member = getServiceAccountNameInBinding(sa.email);
      if (projectEntity) {
        await updateProjectIamPolicyForMember(
          projectEntity.name,
          member,
          roles
        );
      } else {
        await workspaceStore.patchIamPolicy([{ member, roles }]);
      }
    }

    onCreated(sa);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });
    onClose();
  };

  const handleUpdate = async () => {
    if (!serviceAccount) return;

    const updateMask: string[] = [];
    if (title !== serviceAccount.title) {
      updateMask.push("title");
    }

    let updatedSa: ServiceAccount = serviceAccount;
    if (updateMask.length > 0) {
      updatedSa = await serviceAccountStore.updateServiceAccount(
        {
          name: ensureServiceAccountFullName(serviceAccount.email),
          title,
        },
        create(FieldMaskSchema, { paths: [...updateMask] })
      );
    }

    onUpdated(updatedSa);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    onClose();
  };

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("settings.members.service-account")}</SheetTitle>
      </SheetHeader>

      <SheetBody>
        <div className="flex flex-col gap-y-6">
          {/* Name */}
          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium text-control">
              {t("common.name")}
            </label>
            <Input
              autoComplete="off"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Foo"
              maxLength={200}
            />
          </div>

          {/* Email */}
          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium text-control">
              {t("common.email")}
              <span className="ml-0.5 text-error">*</span>
            </label>
            {isEditMode ? (
              <Input value={serviceAccount?.email ?? ""} disabled />
            ) : (
              <div className="px-1 flex items-center border border-control-border rounded-xs overflow-hidden focus-within:border-accent">
                <input
                  type="text"
                  autoComplete="off"
                  value={emailPrefix}
                  onChange={(e) => setEmailPrefix(e.target.value)}
                  className="flex-1 h-9 px-3 py-1 text-sm bg-transparent border-none outline-none ring-0 shadow-none focus:border-none focus:outline-none focus:ring-0 focus:shadow-none"
                />
                <span className="px-2 py-1 text-sm text-control-light bg-control-bg whitespace-nowrap">
                  @{emailSuffix}
                </span>
              </div>
            )}
          </div>

          {/* Roles (create mode only) */}
          {!isEditMode &&
            (projectEntity
              ? hasProjectPermissionV2(
                  projectEntity,
                  "bb.projects.setIamPolicy"
                )
              : hasWorkspacePermissionV2("bb.workspaces.setIamPolicy")) && (
              <div className="flex flex-col gap-y-2">
                <label className="block text-sm font-medium text-control">
                  {t("settings.members.table.roles")}
                </label>
                <RoleSelect
                  value={roles}
                  onChange={setRoles}
                  disabled={false}
                  scope={project ? "project" : undefined}
                />
              </div>
            )}
        </div>
      </SheetBody>

      <SheetFooter>
        <Button variant="outline" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <Button
          disabled={!allowConfirm || !hasPermission || isRequesting}
          onClick={handleSubmit}
        >
          {isEditMode ? t("common.update") : t("common.create")}
        </Button>
      </SheetFooter>
    </>
  );
}

// ============================================================
// ServiceAccountsPage (main)
// ============================================================

export function ServiceAccountsPage({ projectId }: { projectId?: string }) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const serviceAccountStore = useServiceAccountStore();
  const projectStore = useProjectV1Store();

  const projectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : undefined;
  const project = useVueState(() =>
    projectName ? projectStore.getProjectByName(projectName) : undefined
  );

  const parent = useVueState(
    () => projectName ?? actuatorStore.workspaceResourceName
  );

  const [showInactive, setShowInactive] = useState(false);
  const [showDrawer, setShowDrawer] = useState(false);
  const [editingSa, setEditingSa] = useState<ServiceAccount | undefined>(
    undefined
  );

  // Active service accounts
  const fetchActive = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const response = await serviceAccountStore.listServiceAccounts({
        parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        showDeleted: false,
      });
      const list: User[] = response.serviceAccounts.map(serviceAccountToUser);
      return { list, nextPageToken: response.nextPageToken };
    },
    [serviceAccountStore, parent]
  );

  const activeData = usePagedData<User>({
    sessionKey: `bb.service-accounts${projectName ? `.${projectName}` : ""}.active.page-size`,
    fetchList: fetchActive,
  });

  // Inactive service accounts
  const fetchInactive = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const response = await serviceAccountStore.listServiceAccounts({
        parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        showDeleted: true,
        filter: { state: State.DELETED },
      });
      const list: User[] = response.serviceAccounts.map(serviceAccountToUser);
      return { list, nextPageToken: response.nextPageToken };
    },
    [serviceAccountStore, parent]
  );

  const inactiveData = usePagedData<User>({
    sessionKey: `bb.service-accounts${projectName ? `.${projectName}` : ""}.inactive.page-size`,
    enabled: showInactive,
    fetchList: fetchInactive,
  });

  const handleActiveUserUpdated = (user: User) => {
    if (user.state === State.DELETED) {
      activeData.removeCache(user);
      inactiveData.updateCache([user]);
    } else {
      activeData.updateCache([user]);
    }
  };

  const handleInactiveUserUpdated = (user: User) => {
    if (user.state === State.ACTIVE) {
      inactiveData.removeCache(user);
      activeData.updateCache([user]);
    } else {
      inactiveData.updateCache([user]);
    }
  };

  const handleOpenEdit = (user: User) => {
    const sa = serviceAccountStore.getServiceAccount(user.email);
    setEditingSa(sa);
    setShowDrawer(true);
  };

  const handleCreated = (sa: ServiceAccount) => {
    activeData.updateCache([serviceAccountToUser(sa)]);
  };

  const handleUpdated = (sa: ServiceAccount) => {
    activeData.updateCache([serviceAccountToUser(sa)]);
  };

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-medium leading-7 text-main">
          {t("settings.members.service-accounts")}
        </h2>
        <PermissionGuard
          permissions={["bb.serviceAccounts.create"]}
          project={project}
        >
          <Button
            disabled={
              project
                ? project.state !== State.ACTIVE ||
                  !hasProjectPermissionV2(project, "bb.serviceAccounts.create")
                : !hasWorkspacePermissionV2("bb.serviceAccounts.create")
            }
            onClick={() => {
              setEditingSa(undefined);
              setShowDrawer(true);
            }}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("common.create")}
          </Button>
        </PermissionGuard>
      </div>

      <div className="flex flex-col gap-y-4">
        {/* Active list */}
        {activeData.isLoading && activeData.dataList.length === 0 ? (
          <div className="flex items-center justify-center h-32">
            <div className="animate-spin size-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        ) : (
          <>
            <ServiceAccountTable
              users={activeData.dataList}
              project={project}
              onUserUpdated={handleActiveUserUpdated}
              onUserSelected={handleOpenEdit}
            />
            <PagedTableFooter
              pageSize={activeData.pageSize}
              pageSizeOptions={activeData.pageSizeOptions}
              onPageSizeChange={activeData.onPageSizeChange}
              hasMore={activeData.hasMore}
              isFetchingMore={activeData.isFetchingMore}
              onLoadMore={activeData.loadMore}
            />
          </>
        )}

        {/* Inactive toggle */}
        <label className="flex items-center gap-x-2 text-sm cursor-pointer">
          <input
            type="checkbox"
            checked={showInactive}
            onChange={(e) => setShowInactive(e.target.checked)}
          />
          <span className="textinfolabel">
            {t("settings.members.show-inactive")}
          </span>
        </label>

        {/* Inactive list */}
        {showInactive && (
          <div className="flex flex-col gap-y-4">
            <h3 className="text-base font-medium">
              {t("settings.members.inactive-service-accounts")}
            </h3>

            {inactiveData.isLoading && inactiveData.dataList.length === 0 ? (
              <div className="flex items-center justify-center h-32">
                <div className="animate-spin size-6 border-2 border-accent border-t-transparent rounded-full" />
              </div>
            ) : (
              <>
                <ServiceAccountTable
                  users={inactiveData.dataList}
                  project={project}
                  onUserUpdated={handleInactiveUserUpdated}
                />
                <PagedTableFooter
                  pageSize={inactiveData.pageSize}
                  pageSizeOptions={inactiveData.pageSizeOptions}
                  onPageSizeChange={inactiveData.onPageSizeChange}
                  hasMore={inactiveData.hasMore}
                  isFetchingMore={inactiveData.isFetchingMore}
                  onLoadMore={inactiveData.loadMore}
                />
              </>
            )}
          </div>
        )}
      </div>

      <CreateServiceAccountSheet
        open={showDrawer}
        serviceAccount={editingSa}
        project={projectName}
        onClose={() => {
          setShowDrawer(false);
          setEditingSa(undefined);
        }}
        onCreated={handleCreated}
        onUpdated={handleUpdated}
      />
    </div>
  );
}
