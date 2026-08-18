import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import dayjs from "dayjs";
import { ChevronLeft } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import {
  SETTING_ROUTE_PROFILE,
  WORKSPACE_ROUTE_404,
  WORKSPACE_ROUTE_USER_PROFILE,
  WORKSPACE_ROUTE_USERS,
} from "@/app/router/handles";
import { RouterLink } from "@/components/RouterLink";
import { getAvatarColor, getInitials } from "@/components/UserAvatar";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormField, FormSection } from "@/components/ui/form";
import {
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import { useCurrentUser } from "@/hooks/useAppState";
import { displayRoleTitleFromList } from "@/lib/role";
import { EmailInput } from "@/routes/workspace/profile/EmailInput";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  AccountType,
  ALL_USERS_USER_EMAIL,
  getAccountTypeByEmail,
  getDateForPbTimestampProtoEs,
  isValidUserName,
  unknownUser,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2, setDocumentTitle, sortRoles } from "@/utils";
import { migrateUserStorage } from "@/utils/storage-migrate";
import { UserFormSheet } from "./UserFormSheet";

interface UserDetailPageProps {
  principalEmail?: string;
}

const DATE_FORMAT = "YYYY-MM-DD HH:mm";

function formatTimestamp(
  timestamp: Parameters<typeof getDateForPbTimestampProtoEs>[0]
): string | undefined {
  const date = getDateForPbTimestampProtoEs(timestamp);
  return date ? dayjs(date).format(DATE_FORMAT) : undefined;
}

/**
 * Read-only row inside the "details" description list. Admin surfaces state,
 * they don't type into it — every mutation goes through an explicit,
 * confirmable action instead.
 */
function DetailRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-y-1 sm:flex-row sm:gap-x-4 sm:gap-y-0">
      <dt className="w-full shrink-0 text-sm text-control-light sm:w-56">
        {label}
      </dt>
      <dd className="min-w-0 flex-1 text-sm text-main">{children}</dd>
    </div>
  );
}

function EmptyValue() {
  return <span className="text-control-light">-</span>;
}

/**
 * Admin view of another account.
 *
 * This page is deliberately *not* the place where anybody edits their own
 * profile — that lives in Account settings (`SETTING_ROUTE_PROFILE`). Here the
 * account is presented as an administered record: read-only facts plus a short
 * list of named admin operations, each of which confirms before it runs.
 */
export function UserDetailPage({ principalEmail }: UserDetailPageProps) {
  const { t } = useTranslation();

  const currentUser = useCurrentUser();
  const getOrFetchUserByIdentifier = useAppStore(
    (state) => state.getOrFetchUserByIdentifier
  );
  const updateUser = useAppStore((state) => state.updateUser);
  const updateEmail = useAppStore((state) => state.updateEmail);
  const archiveUser = useAppStore((state) => state.archiveUser);
  const restoreUser = useAppStore((state) => state.restoreUser);
  const roleList = useAppStore((state) => state.roleList);
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  const getWorkspaceRolesByName = useAppStore(
    (state) => state.getWorkspaceRolesByName
  );
  const getGroupByIdentifier = useAppStore(
    (state) => state.getGroupByIdentifier
  );
  const batchGetOrFetchGroups = useAppStore(
    (state) => state.batchGetOrFetchGroups
  );
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());

  const cachedUser = useAppStore((state) =>
    principalEmail ? state.getUserByIdentifier(principalEmail) : undefined
  );
  const user = cachedUser ?? unknownUser();

  const isSelf = currentUser.name === user.name;
  const isDeactivated = user.state === State.DELETED;
  const isRealUser =
    isValidUserName(user.name) &&
    getAccountTypeByEmail(user.email) === AccountType.USER &&
    user.email !== ALL_USERS_USER_EMAIL;

  const allowGet = isSelf || hasWorkspacePermissionV2("bb.users.get");
  // Everyone with bb.users.get lands here from a member list or an issue
  // byline, so the page doubles as the directory card. Security posture and
  // sign-in history are only meaningful — and only appropriate — for someone
  // who actually administers accounts.
  const isAdminViewer = hasWorkspacePermissionV2("bb.users.update");
  // The backend refuses admin edits of other principals in SaaS mode, where the
  // account is global rather than workspace-owned. Surface that as a read-only
  // page instead of buttons that fail on click.
  const allowAdminActions = isRealUser && !isSaaSMode;
  const allowUpdate =
    allowAdminActions &&
    !isDeactivated &&
    hasWorkspacePermissionV2("bb.users.update");
  const allowUpdateEmail =
    allowAdminActions &&
    !isDeactivated &&
    hasWorkspacePermissionV2("bb.users.updateEmail");
  const allowDeactivate =
    allowAdminActions && !isSelf && hasWorkspacePermissionV2("bb.users.delete");
  const allowReactivate =
    isRealUser && hasWorkspacePermissionV2("bb.users.undelete");

  const userRoles = useMemo(
    () => [...getWorkspaceRolesByName(user.name)],
    // Recompute when the policy changes; getWorkspaceRolesByName reads it.
    [getWorkspaceRolesByName, user.name, workspacePolicy]
  );

  const [showEditSheet, setShowEditSheet] = useState(false);
  const [showChangeEmail, setShowChangeEmail] = useState(false);
  const [showResetMfa, setShowResetMfa] = useState(false);
  const [showLifecycleConfirm, setShowLifecycleConfirm] = useState(false);
  const [newEmail, setNewEmail] = useState("");
  const [processing, setProcessing] = useState(false);

  useEffect(() => {
    if (!principalEmail) {
      router.replace({ name: WORKSPACE_ROUTE_404 });
      return;
    }
    if (getAccountTypeByEmail(principalEmail) !== AccountType.USER) {
      router.replace({ name: WORKSPACE_ROUTE_404 });
      return;
    }
    void (async () => {
      const fetched = await getOrFetchUserByIdentifier({
        identifier: principalEmail,
        fallback: false,
      });
      if (!isValidUserName(fetched.name)) {
        router.replace({ name: WORKSPACE_ROUTE_404 });
      }
    })();
  }, [principalEmail, getOrFetchUserByIdentifier]);

  useEffect(() => {
    if (user.groups.length > 0) {
      void batchGetOrFetchGroups(user.groups);
    }
  }, [user.groups, batchGetOrFetchGroups]);

  useEffect(() => {
    if (user.title) {
      setDocumentTitle(user.title);
    }
  }, [user.title]);

  const notifyError = useCallback((error: unknown) => {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: (error as ConnectError).message,
    });
  }, []);

  const handleChangeEmail = useCallback(async () => {
    if (!newEmail || newEmail === user.email) return;
    setProcessing(true);
    try {
      const oldEmail = user.email;
      await updateEmail(oldEmail, newEmail);
      migrateUserStorage(oldEmail, newEmail);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      setShowChangeEmail(false);
      router.replace({
        name: WORKSPACE_ROUTE_USER_PROFILE,
        params: { principalEmail: newEmail },
      });
    } catch (error) {
      notifyError(error);
    } finally {
      setProcessing(false);
    }
  }, [newEmail, user.email, updateEmail, notifyError, t]);

  const handleResetMfa = useCallback(async () => {
    setProcessing(true);
    try {
      await updateUser(
        create(UpdateUserRequestSchema, {
          user: { name: user.name, mfaEnabled: false },
          updateMask: create(FieldMaskSchema, { paths: ["mfa_enabled"] }),
        })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("two-factor.messages.2fa-disabled"),
      });
      setShowResetMfa(false);
    } catch (error) {
      notifyError(error);
    } finally {
      setProcessing(false);
    }
  }, [user.name, updateUser, notifyError, t]);

  const handleLifecycleChange = useCallback(async () => {
    setProcessing(true);
    try {
      if (isDeactivated) {
        await restoreUser(user.name);
      } else {
        await archiveUser(user.name);
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      setShowLifecycleConfirm(false);
    } catch (error) {
      notifyError(error);
    } finally {
      setProcessing(false);
    }
  }, [isDeactivated, user.name, archiveUser, restoreUser, notifyError, t]);

  if (!allowGet) {
    return (
      <WorkspacePageLayout>
        <Alert
          variant="error"
          title={t("common.missing-permission")}
          description={
            <>
              {t("common.required-permission")}
              <ul className="list-disc pl-4">
                <li>bb.users.get</li>
              </ul>
            </>
          }
        />
      </WorkspacePageLayout>
    );
  }

  // The record arrives from the store cache, so a direct URL hit renders once
  // before the fetch resolves. Hold the frame rather than flashing a header
  // built from the unknown-user placeholder.
  if (!isValidUserName(user.name)) {
    return (
      <WorkspacePageLayout>
        <div className="flex h-32 items-center justify-center">
          <div className="size-6 animate-spin rounded-full border-2 border-accent border-t-transparent" />
        </div>
      </WorkspacePageLayout>
    );
  }

  const lastLogin = formatTimestamp(user.profile?.lastLoginTime);
  const lastPasswordChange = formatTimestamp(
    user.profile?.lastChangePasswordTime
  );

  return (
    <WorkspacePageLayout>
      <WorkspacePageToolbar align="between">
        <RouterLink
          to={{ name: WORKSPACE_ROUTE_USERS }}
          className="flex items-center gap-x-1 text-sm text-control-light hover:text-main"
        >
          <ChevronLeft className="size-4" />
          {t("common.users")}
        </RouterLink>
        <div className="flex items-center gap-x-2">
          {allowUpdate && (
            <Button appearance="outline" onClick={() => setShowEditSheet(true)}>
              {t("common.edit")}
            </Button>
          )}
          {allowUpdateEmail && (
            <Button
              appearance="outline"
              onClick={() => {
                setNewEmail(user.email);
                setShowChangeEmail(true);
              }}
            >
              {t("settings.members.admin.change-email")}
            </Button>
          )}
        </div>
      </WorkspacePageToolbar>

      {/* Identity header */}
      <div className="flex items-center gap-x-4">
        <div
          className="flex size-14 shrink-0 items-center justify-center rounded-full text-lg font-bold text-white"
          style={{ backgroundColor: getAvatarColor(user.email) }}
        >
          {getInitials(user.title)}
        </div>
        <div className="min-w-0 flex flex-col gap-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="truncate text-xl font-medium text-main">
              {user.title}
            </h1>
            {isSelf && (
              <Badge variant="secondary" className="px-1.5 py-0 text-xs">
                {t("common.you")}
              </Badge>
            )}
            <Badge
              variant={isDeactivated ? "destructive" : "success"}
              className="px-1.5 py-0 text-xs"
            >
              {isDeactivated ? t("common.deactivated") : t("common.active")}
            </Badge>
            {user.mfaEnabled && (
              <Badge variant="success" className="px-1.5 py-0 text-xs">
                {t("two-factor.enabled")}
              </Badge>
            )}
            {user.profile?.source && (
              <Badge className="px-1.5 py-0 text-xs">
                {user.profile.source}
              </Badge>
            )}
          </div>
          <span className="truncate text-sm text-control-light">
            {user.email}
          </span>
        </div>
      </div>

      {isSelf && (
        <Alert
          variant="info"
          title={t("settings.members.admin.viewing-self")}
          description={
            <RouterLink
              to={{ name: SETTING_ROUTE_PROFILE }}
              className="normal-link"
            >
              {t("settings.account.self")}
            </RouterLink>
          }
        />
      )}

      {isRealUser && isSaaSMode && !isSelf && (
        <Alert
          variant="info"
          title={t("settings.members.admin.saas-read-only")}
        />
      )}

      {user.profile?.source && (
        <Alert
          variant="info"
          title={t("settings.members.admin.external-source", {
            source: user.profile.source,
          })}
        />
      )}

      {/* Read-only account record */}
      <FormSection title={t("settings.members.admin.account-details")}>
        <dl className="flex flex-col gap-y-4">
          <DetailRow label={t("common.email")}>{user.email}</DetailRow>
          <DetailRow label={t("common.name")}>
            {user.title || <EmptyValue />}
          </DetailRow>
          <DetailRow label={t("settings.profile.phone")}>
            {user.phone || <EmptyValue />}
          </DetailRow>
          <DetailRow label={t("settings.members.table.roles")}>
            {userRoles.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {sortRoles(userRoles).map((role) => (
                  <Badge key={role} className="px-1.5 py-0 text-xs">
                    {displayRoleTitleFromList(role, roleList)}
                  </Badge>
                ))}
              </div>
            ) : (
              <EmptyValue />
            )}
          </DetailRow>
          <DetailRow label={t("common.groups")}>
            {user.groups.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {user.groups.map((groupName) => (
                  <Badge
                    key={groupName}
                    variant="secondary"
                    className="px-1.5 py-0 text-xs"
                  >
                    {getGroupByIdentifier(groupName)?.title ?? groupName}
                  </Badge>
                ))}
              </div>
            ) : (
              <EmptyValue />
            )}
          </DetailRow>
          {isAdminViewer && (
            <>
              <DetailRow label={t("settings.members.admin.last-sign-in")}>
                {lastLogin ?? <EmptyValue />}
              </DetailRow>
              <DetailRow
                label={t("settings.members.admin.last-password-change")}
              >
                {lastPasswordChange ?? <EmptyValue />}
              </DetailRow>
            </>
          )}
        </dl>
      </FormSection>

      {/* Security — the admin can revoke, never enroll on someone's behalf. */}
      {isAdminViewer && isRealUser && (
        <FormSection title={t("settings.account.security")}>
          <FormField
            title={t("two-factor.self")}
            description={t("settings.members.admin.reset-mfa-tip")}
          >
            <div className="flex items-center gap-x-3">
              <span className="text-sm text-main">
                {user.mfaEnabled
                  ? t("two-factor.enabled")
                  : t("settings.members.admin.mfa-not-enrolled")}
              </span>
              {allowUpdate && user.mfaEnabled && (
                <Button
                  appearance="outline"
                  variant="destructive"
                  size="sm"
                  onClick={() => setShowResetMfa(true)}
                >
                  {t("settings.members.admin.reset-mfa")}
                </Button>
              )}
            </div>
          </FormField>
        </FormSection>
      )}

      {/* Lifecycle */}
      {(allowDeactivate || (isDeactivated && allowReactivate)) && (
        <FormSection title={t("settings.members.admin.account-status")}>
          <FormField
            description={
              isDeactivated
                ? t("settings.members.admin.reactivate-tip")
                : t("settings.members.admin.deactivate-tip")
            }
          >
            <div>
              <Button
                appearance="outline"
                variant={isDeactivated ? "default" : "destructive"}
                onClick={() => setShowLifecycleConfirm(true)}
              >
                {isDeactivated
                  ? t("settings.members.admin.reactivate")
                  : t("settings.members.admin.deactivate")}
              </Button>
            </div>
          </FormField>
        </FormSection>
      )}

      <UserFormSheet
        open={showEditSheet}
        user={showEditSheet ? user : undefined}
        onClose={() => setShowEditSheet(false)}
        onCreated={() => {}}
        onUpdated={() => {}}
      />

      {/* Change email — an identity change, so it is its own confirmed step
          rather than a field inside the edit form. */}
      <Dialog open={showChangeEmail} onOpenChange={setShowChangeEmail}>
        <DialogContent className="max-w-md">
          <DialogTitle>{t("settings.members.admin.change-email")}</DialogTitle>
          <DialogDescription>
            {t("settings.members.admin.change-email-description", {
              email: user.email,
            })}
          </DialogDescription>
          <div className="mt-4">
            <EmailInput value={newEmail} onChange={setNewEmail} />
          </div>
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              appearance="outline"
              disabled={processing}
              onClick={() => setShowChangeEmail(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              disabled={processing || !newEmail || newEmail === user.email}
              onClick={handleChangeEmail}
            >
              {t("common.update")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Reset 2FA */}
      <Dialog open={showResetMfa} onOpenChange={setShowResetMfa}>
        <DialogContent className="max-w-md">
          <DialogTitle>{t("settings.members.admin.reset-mfa")}</DialogTitle>
          <DialogDescription>
            {t("settings.members.admin.reset-mfa-description", {
              user: user.title || user.email,
            })}
          </DialogDescription>
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              appearance="outline"
              disabled={processing}
              onClick={() => setShowResetMfa(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={processing}
              onClick={handleResetMfa}
            >
              {t("settings.members.admin.reset-mfa")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Deactivate / reactivate */}
      <Dialog
        open={showLifecycleConfirm}
        onOpenChange={setShowLifecycleConfirm}
      >
        <DialogContent className="max-w-md">
          <DialogTitle>
            {isDeactivated
              ? t("settings.members.action.reactivate-confirm-title")
              : t("settings.members.action.deactivate-confirm-title")}
          </DialogTitle>
          <DialogDescription>
            {isDeactivated
              ? t("settings.members.admin.reactivate-tip")
              : t("settings.members.admin.deactivate-tip")}
          </DialogDescription>
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              appearance="outline"
              disabled={processing}
              onClick={() => setShowLifecycleConfirm(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant={isDeactivated ? "default" : "destructive"}
              disabled={processing}
              onClick={handleLifecycleChange}
            >
              {isDeactivated
                ? t("settings.members.admin.reactivate")
                : t("settings.members.admin.deactivate")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </WorkspacePageLayout>
  );
}
