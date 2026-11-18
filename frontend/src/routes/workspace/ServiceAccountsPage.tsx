import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Check, Copy, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { PermissionGuard } from "@/components/PermissionGuard";
import {
  ProjectPageLayout,
  ProjectPageToolbar,
} from "@/components/ProjectPageLayout";
import { RoleSelect } from "@/components/RoleSelect";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import { PagedTableFooter, usePagedData } from "@/hooks/usePagedData";
import { useProjectByName } from "@/hooks/useProjectByName";
import { writeTextToClipboard } from "@/lib/clipboard";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import {
  getServiceAccountNameInBinding,
  getServiceAccountSuffix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  type ServiceAccount,
  ServiceAccountSchema,
} from "@/types/proto-es/v1/service_account_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";
import {
  AccountCreatedCell,
  AccountEmptyRow,
  AccountListToolbar,
  type AccountMenuItem,
  AccountNameCell,
  AccountPageInfo,
  AccountRoleCell,
  AccountRowMenu,
  AccountSkeletonRows,
  type AccountStateFilter,
  accountListRequest,
  accountMatchesList,
  DeactivateAccountDialog,
  useAccountRoles,
} from "./MachineAccountList";

// ============================================================
// ServiceAccountTable
// ============================================================

const COLUMNS = 4;

function ServiceAccountTable({
  serviceAccounts,
  loading,
  project,
  projectName,
  filtered,
  onClearFilters,
  onUpdated,
  onSelected,
}: {
  serviceAccounts: ServiceAccount[];
  loading: boolean;
  project?: Project;
  projectName?: string;
  filtered: boolean;
  onClearFilters: () => void;
  onUpdated: (sa: ServiceAccount) => void;
  onSelected: (sa: ServiceAccount) => void;
}) {
  const { t } = useTranslation();
  const deleteServiceAccount = useAppStore(
    (state) => state.deleteServiceAccount
  );
  const undeleteServiceAccount = useAppStore(
    (state) => state.undeleteServiceAccount
  );
  const updateServiceAccount = useAppStore(
    (state) => state.updateServiceAccount
  );
  const rolesOf = useAccountRoles(projectName);

  const can = (permission: Parameters<typeof hasWorkspacePermissionV2>[0]) =>
    project
      ? hasProjectPermissionV2(project, permission)
      : hasWorkspacePermissionV2(permission);
  const canOpenDetail = can("bb.serviceAccounts.get");
  const canDelete = can("bb.serviceAccounts.delete");
  const canRestore = can("bb.serviceAccounts.undelete");
  const canUpdate = can("bb.serviceAccounts.update");

  const [deactivating, setDeactivating] = useState<ServiceAccount>();
  const [isDeactivating, setIsDeactivating] = useState(false);
  const handleDeactivate = async () => {
    if (!deactivating) return;
    setIsDeactivating(true);
    try {
      await deleteServiceAccount(deactivating.name);
      onUpdated(
        create(ServiceAccountSchema, { ...deactivating, state: State.DELETED })
      );
      setDeactivating(undefined);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    } finally {
      setIsDeactivating(false);
    }
  };

  const handleRestore = async (sa: ServiceAccount) => {
    try {
      const updated = await undeleteServiceAccount(sa.name);
      onUpdated(updated);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    }
  };

  const [resetConfirmSa, setResetConfirmSa] = useState<
    ServiceAccount | undefined
  >();
  const [isResetting, setIsResetting] = useState(false);
  const [keyFeedback, setKeyFeedback] = useState<Map<string, "reset" | "copy">>(
    new Map()
  );
  const copiedKeyTimers = useRef(
    new Map<string, ReturnType<typeof setTimeout>>()
  );

  useEffect(
    () => () => {
      for (const timer of copiedKeyTimers.current.values()) {
        clearTimeout(timer);
      }
    },
    []
  );

  const markKeyActionComplete = (name: string, action: "reset" | "copy") => {
    setKeyFeedback((prev) => new Map(prev).set(name, action));
    clearTimeout(copiedKeyTimers.current.get(name));
    copiedKeyTimers.current.set(
      name,
      setTimeout(() => {
        setKeyFeedback((prev) => {
          const next = new Map(prev);
          next.delete(name);
          return next;
        });
        copiedKeyTimers.current.delete(name);
      }, 2000)
    );
  };

  const redactedServiceAccount = (sa: ServiceAccount) =>
    create(ServiceAccountSchema, { ...sa, serviceKey: "" });

  const handleResetKey = async (sa: ServiceAccount) => {
    setIsResetting(true);
    try {
      const updated = await updateServiceAccount(
        { name: sa.name },
        create(FieldMaskSchema, { paths: ["service_key"] })
      );
      setResetConfirmSa(undefined);
      const copied =
        !!updated.serviceKey &&
        (await writeTextToClipboard(updated.serviceKey));
      if (copied) {
        onUpdated(redactedServiceAccount(updated));
        markKeyActionComplete(updated.name, "reset");
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.copied"),
        });
        return;
      }

      onUpdated(updated);
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.copy-failed"),
      });
    } catch {
      // error shown by store
    } finally {
      setIsResetting(false);
    }
  };

  const handleCopyKey = async (sa: ServiceAccount) => {
    if (!(await writeTextToClipboard(sa.serviceKey))) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.copy-failed"),
      });
      return;
    }
    onUpdated(redactedServiceAccount(sa));
    markKeyActionComplete(sa.name, "copy");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  };

  // Shown only while a new or reset key waits to be copied; the key is
  // returned once.
  const renderKeyAction = (sa: ServiceAccount) => {
    const feedback = keyFeedback.get(sa.name);
    if (feedback) {
      return (
        <Button
          appearance="outline"
          size="xs"
          disabled
          className="text-success disabled:opacity-100"
        >
          <Check className="size-3" />
          {feedback === "reset"
            ? t("settings.members.service-key-reset-and-copied")
            : t("common.copied")}
        </Button>
      );
    }
    if (!sa.serviceKey) return null;
    return (
      <Button
        appearance="outline"
        size="xs"
        onClick={() => void handleCopyKey(sa)}
      >
        <Copy className="size-3" />
        {t("settings.members.copy-service-key")}
      </Button>
    );
  };

  const menuItems = (sa: ServiceAccount): AccountMenuItem[] => {
    if (sa.state === State.DELETED) {
      return canRestore
        ? [
            {
              key: "restore",
              label: t("common.restore"),
              onSelect: () => void handleRestore(sa),
            },
          ]
        : [];
    }
    const items: AccountMenuItem[] = [];
    if (canUpdate) {
      items.push({
        key: "reset-key",
        label: t("settings.members.reset-service-key"),
        onSelect: () => setResetConfirmSa(sa),
      });
    }
    if (canOpenDetail) {
      items.push({
        key: "edit",
        label: t("common.edit"),
        onSelect: () => onSelected(sa),
      });
    }
    if (canDelete) {
      items.push({
        key: "deactivate",
        label: t("common.deactivate"),
        destructive: true,
        onSelect: () => setDeactivating(sa),
      });
    }
    return items;
  };

  return (
    <div className="overflow-x-auto rounded-sm border">
      <Table>
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead className="min-w-50 whitespace-nowrap">
              {t("common.name")}
            </TableHead>
            <TableHead className="whitespace-nowrap">
              {t("settings.members.table.roles")}
            </TableHead>
            <TableHead className="whitespace-nowrap">
              {t("common.created")}
            </TableHead>
            <TableHead>
              <span className="sr-only">{t("common.actions")}</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading ? (
            <AccountSkeletonRows columns={COLUMNS} />
          ) : serviceAccounts.length === 0 ? (
            <AccountEmptyRow
              colSpan={COLUMNS}
              filtered={filtered}
              emptyText={t("account-list.no-service-accounts")}
              onClearFilters={onClearFilters}
            />
          ) : (
            serviceAccounts.map((sa) => {
              const isDeleted = sa.state === State.DELETED;
              return (
                <TableRow key={sa.name}>
                  <TableCell className="w-full max-w-0">
                    <AccountNameCell
                      title={sa.title}
                      email={sa.email}
                      inactive={isDeleted}
                      onOpen={
                        canOpenDetail && !isDeleted
                          ? () => onSelected(sa)
                          : undefined
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <AccountRoleCell
                      roles={rolesOf(getServiceAccountNameInBinding(sa.email))}
                    />
                  </TableCell>
                  <TableCell>
                    <AccountCreatedCell createTime={sa.createTime} />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-x-2">
                      {!isDeleted && renderKeyAction(sa)}
                      <AccountRowMenu
                        label={sa.title || sa.email}
                        items={menuItems(sa)}
                      />
                    </div>
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      <AlertDialog
        open={!!resetConfirmSa}
        onOpenChange={(open) => {
          if (!open && !isResetting) setResetConfirmSa(undefined);
        }}
      >
        <AlertDialogContent className="max-w-lg">
          <AlertDialogTitle>
            {t("settings.members.reset-service-key")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("settings.members.reset-service-key-alert")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              appearance="outline"
              disabled={isResetting}
              onClick={() => setResetConfirmSa(undefined)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={isResetting}
              onClick={() => {
                if (resetConfirmSa) void handleResetKey(resetConfirmSa);
              }}
            >
              {t("common.reset")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <DeactivateAccountDialog
        account={deactivating}
        pending={isDeactivating}
        onCancel={() => setDeactivating(undefined)}
        onConfirm={() => void handleDeactivate()}
      />
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
  const createServiceAccount = useAppStore(
    (state) => state.createServiceAccount
  );
  const updateServiceAccount = useAppStore(
    (state) => state.updateServiceAccount
  );
  const patchWorkspaceIamPolicy = useAppStore(
    (state) => state.patchWorkspaceIamPolicy
  );
  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());
  const projectsByName = useAppStore((s) => s.projectsByName);
  const getProjectIamPolicy = useAppStore((state) => state.getProjectIamPolicy);
  const updateProjectIamPolicy = useAppStore(
    (state) => state.updateProjectIamPolicy
  );

  // subscribe to re-render on project cache change
  void projectsByName;
  const projectEntityFromName = useProjectByName(project ?? "");
  const projectEntity = project ? projectEntityFromName : undefined;

  const parent = project ?? workspaceResourceName;

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
    const policy = structuredClone(getProjectIamPolicy(projectName));
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
    await updateProjectIamPolicy(projectName, policy);
  };

  const handleCreate = async () => {
    const sa = await createServiceAccount(
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
        await patchWorkspaceIamPolicy([{ member, roles }]);
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
      updatedSa = await updateServiceAccount(
        {
          name: serviceAccount.name,
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
          <FormField title={<>{t("common.name")}</>}>
            <Input
              autoComplete="off"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Foo"
              maxLength={200}
            />
          </FormField>

          {/* Email */}
          <FormField
            title={
              <>
                {t("common.email")}
                <span className="ml-0.5 text-error">*</span>
              </>
            }
          >
            {isEditMode ? (
              <Input value={serviceAccount?.email ?? ""} disabled />
            ) : (
              <div className="px-1 flex items-center border border-control-border rounded-xs overflow-hidden focus-within:border-accent">
                <Input
                  size="md"
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
          </FormField>

          {/* Roles (create mode only) */}
          {!isEditMode &&
            (projectEntity
              ? hasProjectPermissionV2(
                  projectEntity,
                  "bb.projects.setIamPolicy"
                )
              : hasWorkspacePermissionV2("bb.workspaces.setIamPolicy")) && (
              <FormField title={<>{t("settings.members.table.roles")}</>}>
                <RoleSelect
                  value={roles}
                  onChange={setRoles}
                  disabled={false}
                  scope={project ? "project" : undefined}
                />
              </FormField>
            )}
        </div>
      </SheetBody>

      <SheetFooter>
        <Button appearance="outline" onClick={onClose}>
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
  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());
  const projectsByName = useAppStore((s) => s.projectsByName);
  const listServiceAccounts = useAppStore((state) => state.listServiceAccounts);

  const projectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : undefined;
  // subscribe to re-render on project cache change
  void projectsByName;
  const projectFromName = useProjectByName(projectName ?? "");
  const project = projectName ? projectFromName : undefined;

  const parent = projectName ?? workspaceResourceName;

  const [query, setQuery] = useState("");
  const [stateFilter, setStateFilter] = useState<AccountStateFilter>("active");
  const [showDrawer, setShowDrawer] = useState(false);
  const [editingSa, setEditingSa] = useState<ServiceAccount | undefined>(
    undefined
  );

  const fetchList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const response = await listServiceAccounts({
        parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        ...accountListRequest(stateFilter, query),
      });
      return {
        list: response.serviceAccounts,
        nextPageToken: response.nextPageToken,
      };
    },
    [listServiceAccounts, parent, stateFilter, query]
  );

  const data = usePagedData<ServiceAccount>({
    sessionKey: `bb.service-accounts${projectName ? `.${projectName}` : ""}.page-size`,
    fetchList,
  });

  // Keeps a row only while it matches the filters. Removing the last loaded
  // row refetches, so remaining pages stay reachable.
  const handleUpdated = (sa: ServiceAccount) => {
    if (accountMatchesList(sa, stateFilter, query)) {
      data.updateCache([sa]);
      return;
    }
    data.removeCache(sa);
    if (data.hasMore && data.dataList.every((row) => row.name === sa.name)) {
      data.refresh();
    }
  };

  const handleOpenEdit = (sa: ServiceAccount) => {
    setEditingSa(sa);
    setShowDrawer(true);
  };

  const canCreate = project
    ? project.state === State.ACTIVE &&
      hasProjectPermissionV2(project, "bb.serviceAccounts.create")
    : hasWorkspacePermissionV2("bb.serviceAccounts.create");
  const openCreate = () => {
    setEditingSa(undefined);
    setShowDrawer(true);
  };
  const filtered = query.trim() !== "" || stateFilter !== "active";

  const PageLayout = projectName ? ProjectPageLayout : WorkspacePageLayout;
  const PageToolbar = projectName ? ProjectPageToolbar : WorkspacePageToolbar;

  return (
    <PageLayout>
      <AccountPageInfo
        isProject={!!projectName}
        description={t("account-list.service-accounts-description")}
      />

      <PageToolbar>
        <AccountListToolbar
          query={query}
          onQueryChange={setQuery}
          state={stateFilter}
          onStateChange={setStateFilter}
          action={
            <PermissionGuard
              permissions={["bb.serviceAccounts.create"]}
              project={project}
            >
              <Button disabled={!canCreate} onClick={openCreate}>
                <Plus className="size-4" />
                {t("common.create")}
              </Button>
            </PermissionGuard>
          }
        />
      </PageToolbar>

      <div className="flex flex-col gap-y-4">
        <ServiceAccountTable
          serviceAccounts={data.dataList}
          loading={data.isLoading && data.dataList.length === 0}
          project={project}
          projectName={projectName}
          filtered={filtered}
          onClearFilters={() => {
            setQuery("");
            setStateFilter("active");
          }}
          onUpdated={handleUpdated}
          onSelected={handleOpenEdit}
        />
        {(data.dataList.length > 0 || data.hasMore) && (
          <PagedTableFooter
            pageSize={data.pageSize}
            pageSizeOptions={data.pageSizeOptions}
            onPageSizeChange={data.onPageSizeChange}
            hasMore={data.hasMore}
            isFetchingMore={data.isFetchingMore}
            onLoadMore={data.loadMore}
          />
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
        onCreated={handleUpdated}
        onUpdated={handleUpdated}
      />
    </PageLayout>
  );
}
