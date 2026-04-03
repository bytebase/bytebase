import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Pencil, Plus, Trash2, Undo2, X } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import {
  ensureServiceAccountFullName,
  pushNotification,
  useActuatorV1Store,
  useWorkspaceV1Store,
} from "@/store";
import {
  serviceAccountToUser,
  useServiceAccountStore,
} from "@/store/modules/serviceAccount";
import {
  getServiceAccountNameInBinding,
  getServiceAccountSuffix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { RoleMultiSelect } from "./shared/RoleMultiSelect";
import { UserAvatar } from "./shared/UserAvatar";
import { PagedTableFooter, usePagedData } from "./shared/usePagedData";

// ============================================================
// ServiceAccountTable
// ============================================================

function ServiceAccountTable({
  users,
  onUserUpdated,
  onUserSelected,
}: {
  users: User[];
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
            <th className="px-4 py-2 text-right font-medium whitespace-nowrap">
              {t("common.operations")}
            </th>
          </tr>
        </thead>
        <tbody>
          {users.map((user, i) => {
            const isDeleted = user.state === State.DELETED;

            return (
              <tr
                key={user.name}
                className={`border-b last:border-b-0 ${i % 2 === 1 ? "bg-gray-50" : ""}`}
              >
                {/* Account column */}
                <td className="px-4 py-2">
                  <div className="flex items-center gap-x-3">
                    <UserAvatar title={user.title || user.email} />
                    <div className="flex flex-col">
                      <div className="flex items-center gap-x-1.5">
                        <span
                          className={
                            isDeleted
                              ? "line-through text-control-light"
                              : "font-medium text-accent"
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
                  </div>
                </td>

                {/* Operations column */}
                <td className="px-4 py-2">
                  <div className="flex justify-end gap-x-1">
                    {!isDeleted && (
                      <>
                        {hasWorkspacePermissionV2(
                          "bb.serviceAccounts.delete"
                        ) && (
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
                        {hasWorkspacePermissionV2("bb.serviceAccounts.get") && (
                          <Tooltip content={t("common.edit")}>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7"
                              onClick={() => onUserSelected?.(user)}
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                          </Tooltip>
                        )}
                      </>
                    )}
                    {isDeleted &&
                      hasWorkspacePermissionV2(
                        "bb.serviceAccounts.undelete"
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
// CreateServiceAccountDrawer
// ============================================================

function CreateServiceAccountDrawer({
  serviceAccount,
  onClose,
  onCreated,
  onUpdated,
}: {
  serviceAccount: ServiceAccount | undefined;
  onClose: () => void;
  onCreated: (sa: ServiceAccount) => void;
  onUpdated: (sa: ServiceAccount) => void;
}) {
  const { t } = useTranslation();
  const serviceAccountStore = useServiceAccountStore();
  const workspaceStore = useWorkspaceV1Store();
  const actuatorStore = useActuatorV1Store();

  const parent = useVueState(() => actuatorStore.workspaceResourceName);

  const isEditMode = !!serviceAccount && !!serviceAccount.email;
  const emailSuffix = getServiceAccountSuffix();

  const [title, setTitle] = useState(serviceAccount?.title ?? "");
  const [emailPrefix, setEmailPrefix] = useState(
    serviceAccount?.email ? serviceAccount.email.split("@")[0] : ""
  );
  const [roles, setRoles] = useState<string[]>([]);
  const [isRequesting, setIsRequesting] = useState(false);

  useEscapeKey(true, onClose);

  const allowConfirm = isEditMode ? true : emailPrefix.trim().length > 0;

  const hasPermission = hasWorkspacePermissionV2(
    isEditMode ? "bb.serviceAccounts.update" : "bb.serviceAccounts.create"
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
    const sa = await serviceAccountStore.createServiceAccount(
      emailPrefix.trim(),
      { title: title.trim() || emailPrefix.trim() },
      parent
    );

    if (roles.length > 0) {
      await workspaceStore.patchIamPolicy([
        {
          member: getServiceAccountNameInBinding(sa.email),
          roles,
        },
      ]);
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
              ? t("settings.members.update-service-account")
              : t("settings.members.add-service-account")}
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
              <input
                type="text"
                autoComplete="off"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Foo"
                maxLength={200}
                className="flex h-9 w-full rounded-xs border border-control-border bg-transparent px-3 py-1 text-sm outline-hidden placeholder:text-control-placeholder focus:ring-2 focus:ring-accent focus:border-accent"
              />
            </div>

            {/* Email */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("common.email")}
                <span className="ml-0.5 text-error">*</span>
              </label>
              {isEditMode ? (
                <input
                  type="text"
                  value={serviceAccount?.email ?? ""}
                  disabled
                  className="flex h-9 w-full rounded-xs border border-control-border bg-gray-50 px-3 py-1 text-sm opacity-60 cursor-not-allowed"
                />
              ) : (
                <div className="flex items-center border border-control-border rounded-xs overflow-hidden focus-within:ring-2 focus-within:ring-accent focus-within:border-accent">
                  <input
                    type="text"
                    autoComplete="off"
                    value={emailPrefix}
                    onChange={(e) => setEmailPrefix(e.target.value)}
                    className="flex-1 h-9 px-3 py-1 text-sm outline-hidden bg-transparent"
                  />
                  <span className="px-2 py-1 text-sm text-control-light bg-control-bg border-l border-control-border whitespace-nowrap">
                    @{emailSuffix}
                  </span>
                </div>
              )}
            </div>

            {/* Roles (create mode only) */}
            {!isEditMode &&
              hasWorkspacePermissionV2("bb.workspaces.setIamPolicy") && (
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
// ServiceAccountsPage (main)
// ============================================================

export function ServiceAccountsPage() {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const serviceAccountStore = useServiceAccountStore();

  const parent = useVueState(() => actuatorStore.workspaceResourceName);

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
    sessionKey: "bb.service-accounts.active.page-size",
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
    sessionKey: "bb.service-accounts.inactive.page-size",
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
        <Button
          disabled={!hasWorkspacePermissionV2("bb.serviceAccounts.create")}
          onClick={() => {
            setEditingSa(undefined);
            setShowDrawer(true);
          }}
        >
          <Plus className="h-4 w-4 mr-1" />
          {t("settings.members.add-service-account")}
        </Button>
      </div>

      <div className="flex flex-col gap-y-4">
        {/* Active list */}
        {activeData.isLoading && activeData.dataList.length === 0 ? (
          <div className="flex items-center justify-center h-32">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        ) : (
          <>
            <ServiceAccountTable
              users={activeData.dataList}
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
          <div className="flex flex-col gap-y-3">
            <h3 className="text-base font-medium">
              {t("settings.members.inactive-service-accounts")}
            </h3>

            {inactiveData.isLoading && inactiveData.dataList.length === 0 ? (
              <div className="flex items-center justify-center h-32">
                <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
              </div>
            ) : (
              <>
                <ServiceAccountTable
                  users={inactiveData.dataList}
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

      {showDrawer && (
        <CreateServiceAccountDrawer
          serviceAccount={editingSa}
          onClose={() => {
            setShowDrawer(false);
            setEditingSa(undefined);
          }}
          onCreated={handleCreated}
          onUpdated={handleUpdated}
        />
      )}
    </div>
  );
}
