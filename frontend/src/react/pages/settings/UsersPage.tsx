import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import {
  CircleAlert,
  CircleCheck,
  Eye,
  EyeOff,
  Plus,
  Settings,
  Trash2,
  Undo2,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { RoleSelect } from "@/react/components/RoleSelect";
import { UserAvatar } from "@/react/components/UserAvatar";
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
import { Tooltip } from "@/react/components/ui/tooltip";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import {
  WORKSPACE_ROUTE_GROUPS,
  WORKSPACE_ROUTE_USER_PROFILE,
} from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useGroupStore,
  useServiceAccountStore,
  useSettingV1Store,
  useSubscriptionV1Store,
  useUserStore,
  useWorkloadIdentityStore,
  useWorkspaceV1Store,
} from "@/store";
import { getUserFullNameByType } from "@/store/modules/v1/common";
import {
  AccountType,
  getAccountTypeByEmail,
  getUserEmailInBinding,
} from "@/types";
import { PresetRoleType } from "@/types/iam";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  UpdateUserRequestSchema,
  UserSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { AADSyncSheet } from "./shared/AADSyncSheet";

// ============================================================
// UserTable
// ============================================================

function UserTable({
  users,
  onUserUpdated,
  onUserSelected,
  onGroupSelected,
}: {
  users: User[];
  onUserUpdated: (user: User) => void;
  onUserSelected?: (user: User) => void;
  onGroupSelected?: (group: Group) => void;
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
      t("settings.members.action.deactivate-confirm-title")
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

      const updated = create(UserSchema, { ...user, state: State.DELETED });
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

      const updated = create(UserSchema, { ...user, state: State.ACTIVE });
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

  const handleView = (user: User) => {
    const accountType = getAccountTypeByEmail(user.email);
    if (accountType === AccountType.USER && onUserSelected) {
      onUserSelected(user);
    } else {
      router.push({
        name: WORKSPACE_ROUTE_USER_PROFILE,
        params: { principalEmail: user.email },
      });
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
      <Table>
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead className="whitespace-nowrap">
              {t("settings.members.table.account")}
            </TableHead>
            <TableHead className="whitespace-nowrap">
              {t("settings.members.table.groups")}
            </TableHead>
            <TableHead className="text-right whitespace-nowrap">
              {t("common.operations")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.map((user) => {
            const accountType = getAccountTypeByEmail(user.email);
            const isDeleted = user.state === State.DELETED;
            const isSelf = currentUser.name === user.name;

            return (
              <TableRow
                key={user.name}
                className={cn(
                  hasWorkspacePermissionV2(getViewPermission(accountType)) &&
                    "cursor-pointer hover:bg-control-bg focus-visible:outline-none focus-visible:bg-control-bg"
                )}
                tabIndex={
                  hasWorkspacePermissionV2(getViewPermission(accountType))
                    ? 0
                    : undefined
                }
                role={
                  hasWorkspacePermissionV2(getViewPermission(accountType))
                    ? "button"
                    : undefined
                }
                aria-label={
                  hasWorkspacePermissionV2(getViewPermission(accountType))
                    ? user.title || user.email
                    : undefined
                }
                onClick={
                  hasWorkspacePermissionV2(getViewPermission(accountType))
                    ? () => handleView(user)
                    : undefined
                }
                onKeyDown={
                  hasWorkspacePermissionV2(getViewPermission(accountType))
                    ? (e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          handleView(user);
                        }
                      }
                    : undefined
                }
              >
                {/* Account column */}
                <TableCell className="py-2">
                  <div className="flex items-center gap-x-3">
                    <UserAvatar title={user.title || user.email} />
                    <div className="flex flex-col">
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
                        {isSelf && (
                          <Badge
                            variant="secondary"
                            className="text-xs px-1.5 py-0"
                          >
                            {t("common.you")}
                          </Badge>
                        )}
                        {accountType !== AccountType.USER && (
                          <Badge className="text-xs px-1.5 py-0">
                            {getAccountTypeLabel(accountType)}
                          </Badge>
                        )}
                        {user.mfaEnabled && (
                          <Badge
                            variant="success"
                            className="text-xs px-1.5 py-0"
                          >
                            {t("two-factor.enabled")}
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
                </TableCell>

                {/* Groups column */}
                <TableCell className="py-2">
                  <UserGroupsCell
                    user={user}
                    onGroupSelected={onGroupSelected}
                  />
                </TableCell>

                {/* Operations column — destructive/secondary actions only.
                    The row itself is clickable to open the detail sheet. */}
                <TableCell className="py-2">
                  <div className="flex justify-end gap-x-1">
                    {!isDeleted &&
                      hasWorkspacePermissionV2(
                        getDeletePermission(accountType)
                      ) &&
                      !isSelf && (
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
                      hasWorkspacePermissionV2(
                        getUndeletePermission(accountType)
                      ) && (
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
          })}
        </TableBody>
      </Table>
    </div>
  );
}

// ============================================================
// UserGroupsCell
// ============================================================

function UserGroupsCell({
  user,
  onGroupSelected,
}: {
  user: User;
  onGroupSelected?: (group: Group) => void;
}) {
  const groupStore = useGroupStore();

  if (!user.groups || user.groups.length === 0) {
    return <span className="text-control-light">-</span>;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {user.groups.map((groupName) => {
        const group = groupStore.getGroupByIdentifier(groupName);
        return (
          <Badge
            key={groupName}
            variant="secondary"
            className="text-xs px-1.5 py-0 cursor-pointer"
            // The parent TableRow is a row-click button — stop propagation so
            // clicking a group badge navigates to the group without also
            // opening the user detail sheet.
            onClick={(e) => {
              e.stopPropagation();
              if (group && onGroupSelected) {
                onGroupSelected(group);
              }
            }}
          >
            {group?.title || groupName}
          </Badge>
        );
      })}
    </div>
  );
}

// ============================================================
// CreateUserDrawer
// ============================================================

function extractUserTitle(email: string): string {
  const atIndex = email.indexOf("@");
  return atIndex !== -1 ? email.substring(0, atIndex) : email;
}

interface CreateUserSheetProps {
  open: boolean;
  user: User | undefined;
  onClose: () => void;
  onCreated: (user: User) => void;
  onUpdated: (user: User) => void;
}

// Outer wrapper — renders the Sheet container. The actual form lives in
// `UserForm` below, keyed by entity so it remounts cleanly every time a
// different entity is selected, and gated by `open` so it unmounts on close.
// This guarantees that useState initializers always see the latest `user`
// prop and that there's no stale state between opens.
function CreateUserSheet(props: CreateUserSheetProps) {
  const { open, user, onClose } = props;
  // Freeze the entity while open=false so the inner form stays visually
  // stable during the Sheet's close animation; see the long comment on
  // CreateWorkloadIdentitySheet for the full rationale.
  const openEntityRef = useRef(user);
  if (open) {
    openEntityRef.current = user;
  }
  const stableUser = openEntityRef.current;
  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <UserForm
          key={stableUser?.name ?? "new"}
          user={stableUser}
          onClose={props.onClose}
          onCreated={props.onCreated}
          onUpdated={props.onUpdated}
        />
      </SheetContent>
    </Sheet>
  );
}

function UserForm({
  user,
  onClose,
  onCreated,
  onUpdated,
}: Omit<CreateUserSheetProps, "open">) {
  const { t } = useTranslation();
  const userStore = useUserStore();
  const workspaceStore = useWorkspaceV1Store();
  const settingV1Store = useSettingV1Store();

  const userMapToRoles = useVueState(() => workspaceStore.userMapToRoles);
  const passwordRestriction = useVueState(
    () => settingV1Store.workspaceProfile.passwordRestriction
  );
  const enforceIdentityDomain = useVueState(
    () => settingV1Store.workspaceProfile.enforceIdentityDomain
  );
  const workspaceDomains = useVueState(
    () => settingV1Store.workspaceProfile.domains
  );

  const isEditMode =
    !!user && user.name !== "" && !user.name.endsWith("/unknown");

  const allowUpdate =
    !isEditMode || hasWorkspacePermissionV2("bb.users.update");

  // Capture initial values on mount. Because the parent keys this component
  // by user, it remounts fresh every time a different user is edited, so
  // these initial values always reflect the latest `user` prop.
  //
  // The empty deps array is intentional: we want the initial baseline
  // frozen at mount so dirty tracking compares against it. If we added
  // `userMapToRoles` to deps, a Pinia store refresh mid-edit would move
  // the baseline and incorrectly classify untouched fields as dirty
  // (or vice versa). Since the outer CreateUserSheet wrapper remounts
  // this component whenever the edited user changes, "mount-only" is the
  // right scope for the baseline.
  const initialRoles = useMemo(() => {
    if (!user || !isEditMode) {
      return [PresetRoleType.WORKSPACE_MEMBER];
    }
    const roles = userMapToRoles.get(getUserFullNameByType(user));
    return roles ? [...roles] : [];
  }, []);
  const initialTitle = user?.title ?? "";
  const initialEmail = user?.email ?? "";
  const initialPhone = user?.phone ?? "";

  const [title, setTitle] = useState(initialTitle);
  const [email, setEmail] = useState(initialEmail);
  const [phone, setPhone] = useState(initialPhone);
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [roles, setRoles] = useState<string[]>(initialRoles);
  const [isRequesting, setIsRequesting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  // Password validation
  const passwordChecks = useMemo(() => {
    const minLength = passwordRestriction?.minLength ?? 8;
    const checks: { text: string; matched: boolean }[] = [
      {
        text: t("settings.general.workspace.password-restriction.min-length", {
          min: minLength,
        }),
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

  const passwordHint =
    password.length > 0 && passwordChecks.some((c) => !c.matched);
  const passwordMismatch = password.length > 0 && password !== passwordConfirm;

  const emailDomainValid = useMemo(() => {
    if (isEditMode) return true;
    if (!enforceIdentityDomain || workspaceDomains.length === 0) return true;
    const atIdx = email.indexOf("@");
    if (atIdx < 0) return false;
    const domain = email.slice(atIdx + 1);
    return workspaceDomains.includes(domain);
  }, [email, isEditMode, enforceIdentityDomain, workspaceDomains]);

  const isFormValid =
    email.length > 0 &&
    emailDomainValid &&
    !passwordHint &&
    !passwordMismatch &&
    (isEditMode || password.length > 0);

  // Dirty tracking — in edit mode the Update button is disabled unless
  // something actually changed. Create mode is always "dirty".
  const isDirty = useMemo(() => {
    if (!isEditMode) return true;
    if (title !== initialTitle) return true;
    if (phone !== initialPhone) return true;
    if (password.length > 0) return true;
    if (!isEqual([...initialRoles].sort(), [...roles].sort())) return true;
    return false;
  }, [
    isEditMode,
    title,
    phone,
    password,
    roles,
    initialTitle,
    initialPhone,
    initialRoles,
  ]);

  const allowConfirm = isFormValid && isDirty;

  const hasPermission = hasWorkspacePermissionV2(
    isEditMode ? "bb.users.update" : "bb.users.create"
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
    const createdUser = await userStore.createUser({
      ...create(UserSchema, {}),
      title: title || extractUserTitle(email),
      email,
      phone,
      password,
    });
    if (roles.length > 0) {
      try {
        await workspaceStore.patchIamPolicy([
          {
            member: getUserEmailInBinding(createdUser.email),
            roles,
          },
        ]);
      } catch {
        // do nothing
      }
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
      <SheetHeader>
        <SheetTitle>{t("common.user")}</SheetTitle>
      </SheetHeader>

      <SheetBody>
        <div className="flex flex-col gap-y-6">
          {/* Name */}
          <div className="flex flex-col gap-y-2">
            <label className="block text-sm font-medium text-control">
              {t("common.name")}
            </label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("common.name")}
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
              <RoleSelect value={roles} onChange={setRoles} disabled={false} />
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
// UsersPage (main)
// ============================================================

export function UsersPage() {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const userStore = useUserStore();

  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);

  const hasDirectorySyncFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_DIRECTORY_SYNC)
  );
  const canAccessSettings = hasWorkspacePermissionV2("bb.settings.get");

  const [userSearchText, setUserSearchText] = useState("");
  const [showInactiveUsers, setShowInactiveUsers] = useState(false);
  const [inactiveUserSearchText, setInactiveUserSearchText] = useState("");

  // Drawer visibility
  const [showCreateUserDrawer, setShowCreateUserDrawer] = useState(false);
  const [showAadSyncDrawer, setShowAadSyncDrawer] = useState(false);
  const [editingUser, setEditingUser] = useState<User | undefined>(undefined);

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

  const hasUserListPermission = hasWorkspacePermissionV2("bb.users.list");
  const activeUsers = usePagedData<User>({
    sessionKey: "bb.users.active.page-size",
    fetchList: fetchActiveUsers,
    enabled: !isSaaSMode && hasUserListPermission,
  });

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
    enabled: !isSaaSMode && hasUserListPermission && showInactiveUsers,
    fetchList: fetchInactiveUsers,
  });

  const handleActiveUserUpdated = (user: User) => {
    if (user.state === State.DELETED) {
      // Deactivated: remove from active list, add to inactive list
      activeUsers.removeCache(user);
      inactiveUsers.updateCache([user]);
    } else {
      activeUsers.updateCache([user]);
    }
  };

  const handleInactiveUserUpdated = (user: User) => {
    if (user.state === State.ACTIVE) {
      // Restored: remove from inactive list, add to active list
      inactiveUsers.removeCache(user);
      activeUsers.updateCache([user]);
    } else {
      inactiveUsers.updateCache([user]);
    }
  };

  const handleGroupSelected = (group: Group) => {
    router.push({ name: WORKSPACE_ROUTE_GROUPS, query: { name: group.name } });
  };

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {/* Action bar */}
      <div className="flex items-center justify-between gap-x-2 mb-4">
        <SearchInput
          placeholder={t("common.filter-by-name")}
          value={userSearchText}
          onChange={(e) => setUserSearchText(e.target.value)}
        />
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
          <Button
            disabled={!hasWorkspacePermissionV2("bb.users.create")}
            onClick={() => {
              setEditingUser(create(UserSchema, { title: "" }));
              setShowCreateUserDrawer(true);
            }}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("common.create")}
          </Button>
        </div>
      </div>

      <div className="flex flex-col gap-y-4">
        <ComponentPermissionGuard permissions={["bb.users.list"]}>
          {activeUsers.isLoading && activeUsers.dataList.length === 0 ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
            </div>
          ) : (
            <>
              <UserTable
                users={activeUsers.dataList}
                onUserUpdated={handleActiveUserUpdated}
                onUserSelected={(user) => {
                  setEditingUser(user);
                  setShowCreateUserDrawer(true);
                }}
                onGroupSelected={handleGroupSelected}
              />
              <PagedTableFooter
                pageSize={activeUsers.pageSize}
                pageSizeOptions={activeUsers.pageSizeOptions}
                onPageSizeChange={activeUsers.onPageSizeChange}
                hasMore={activeUsers.hasMore}
                isFetchingMore={activeUsers.isFetchingMore}
                onLoadMore={activeUsers.loadMore}
              />
            </>
          )}
        </ComponentPermissionGuard>

        {/* Inactive users toggle (only shown with list permission) */}
        {hasUserListPermission && (
          <label className="flex items-center gap-x-2 text-sm cursor-pointer">
            <Checkbox
              checked={showInactiveUsers}
              onCheckedChange={(checked) => setShowInactiveUsers(checked)}
            />
            {t("settings.members.show-inactive")}
          </label>
        )}

        {showInactiveUsers && (
          <div className="flex flex-col gap-y-4">
            <div className="flex items-center justify-between gap-x-4">
              <h3 className="text-base font-medium">
                {t("settings.members.inactive-users")}
              </h3>
              <SearchInput
                placeholder={t("common.filter-by-name")}
                value={inactiveUserSearchText}
                onChange={(e) => setInactiveUserSearchText(e.target.value)}
              />
            </div>

            {inactiveUsers.isLoading && inactiveUsers.dataList.length === 0 ? (
              <div className="flex items-center justify-center h-32">
                <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
              </div>
            ) : (
              <>
                <UserTable
                  users={inactiveUsers.dataList}
                  onUserUpdated={handleInactiveUserUpdated}
                  onUserSelected={(user) => {
                    setEditingUser(user);
                    setShowCreateUserDrawer(true);
                  }}
                  onGroupSelected={handleGroupSelected}
                />
                <PagedTableFooter
                  pageSize={inactiveUsers.pageSize}
                  pageSizeOptions={inactiveUsers.pageSizeOptions}
                  onPageSizeChange={inactiveUsers.onPageSizeChange}
                  hasMore={inactiveUsers.hasMore}
                  isFetchingMore={inactiveUsers.isFetchingMore}
                  onLoadMore={inactiveUsers.loadMore}
                />
              </>
            )}
          </div>
        )}
      </div>

      <CreateUserSheet
        open={showCreateUserDrawer}
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

      <AADSyncSheet
        open={showAadSyncDrawer}
        onClose={() => setShowAadSyncDrawer(false)}
      />
    </div>
  );
}
