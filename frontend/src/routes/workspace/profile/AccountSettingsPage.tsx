import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { Ellipsis } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import {
  SETTING_ROUTE_PROFILE_TWO_FACTOR,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/app/router/handles";
import { FeatureBadge } from "@/components/FeatureBadge";
import { LearnMoreLink } from "@/components/LearnMoreLink";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { FeatureModal } from "@/components/ui/feature-modal";
import { FormField, FormFieldGroup, FormSection } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { WorkspacePageLayout } from "@/components/WorkspacePageLayout";
import { useCurrentUser } from "@/hooks/useAppState";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { displayRoleTitleFromList } from "@/lib/role";
import { RegenerateRecoveryCodesView } from "@/routes/workspace/two-factor/RegenerateRecoveryCodesView";
import { hasFeature, pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { AccountType, getAccountTypeByEmail } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2, setDocumentTitle, sortRoles } from "@/utils";
import { migrateUserStorage } from "@/utils/storage-migrate";
import { EmailInput } from "./EmailInput";
import { getPasswordErrors, UserPasswordSection } from "./UserPasswordSection";

/**
 * Account settings for the signed-in user.
 *
 * Everything here is something a person does to their *own* account: rename
 * themselves, change how they are reached, rotate their password, enroll a
 * second factor. Administering somebody else's account is a different job with
 * a different page — see `UserDetailPage`.
 */
export function AccountSettingsPage() {
  const { t } = useTranslation();

  const legacyCurrentUser = useCurrentUser();
  const [currentUser, setCurrentUser] = useState(legacyCurrentUser);

  const updateUser = useAppStore((state) => state.updateUser);
  const updateEmail = useAppStore((state) => state.updateEmail);
  const updateCurrentUserNameForEmailChange = useAppStore(
    (state) => state.updateCurrentUserNameForEmailChange
  );
  const roleList = useAppStore((state) => state.roleList);
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  const getWorkspaceRolesByName = useAppStore(
    (state) => state.getWorkspaceRolesByName
  );
  const passwordRestriction = useAppStore(
    (s) => s.getWorkspaceProfile().passwordRestriction
  );
  const requireMfa = useAppStore((s) => s.getWorkspaceProfile().requireMfa);
  const has2FAFeature = useAppStore((s) =>
    s.hasFeature(PlanFeature.FEATURE_TWO_FA)
  );

  const user = currentUser;
  const isEndUser = getAccountTypeByEmail(user.email) === AccountType.USER;
  // Email is a workspace-managed identity: the server gates UpdateEmail on
  // bb.users.updateEmail, so most people see it as a read-only fact here.
  const allowChangeEmail = hasWorkspacePermissionV2("bb.users.updateEmail");
  const isMFAEnabled = user.mfaEnabled;

  const userRoles = useMemo(
    () => [...getWorkspaceRolesByName(user.name)],
    // Recompute when the policy changes; getWorkspaceRolesByName reads it.
    [getWorkspaceRolesByName, user.name, workspacePolicy]
  );

  // --- Profile section ---
  const [title, setTitle] = useState(user.title);
  const [phone, setPhone] = useState(user.phone);
  const [savingProfile, setSavingProfile] = useState(false);

  // --- Password section ---
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [savingPassword, setSavingPassword] = useState(false);

  // --- Email + 2FA dialogs ---
  const [showChangeEmail, setShowChangeEmail] = useState(false);
  const [newEmail, setNewEmail] = useState("");
  const [savingEmail, setSavingEmail] = useState(false);
  const [showFeatureModal, setShowFeatureModal] = useState(false);
  const [showDisable2FAConfirm, setShowDisable2FAConfirm] = useState(false);
  const [showRegenerateView, setShowRegenerateView] = useState(false);

  useEffect(() => {
    setCurrentUser(legacyCurrentUser);
    setTitle(legacyCurrentUser.title);
    setPhone(legacyCurrentUser.phone);
  }, [legacyCurrentUser]);

  useEffect(() => {
    setDocumentTitle(t("settings.account.self"));
  }, [t]);

  const profileDirty = title !== user.title || phone !== user.phone;
  const passwordErrors = useMemo(
    () => getPasswordErrors(password, passwordConfirm, passwordRestriction),
    [password, passwordConfirm, passwordRestriction]
  );
  const allowSavePassword =
    password.length > 0 &&
    !passwordErrors.hasHint &&
    !passwordErrors.hasMismatch;

  useUnsavedChangesGuard(profileDirty);

  const notifyError = useCallback((error: unknown) => {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: (error as ConnectError).message,
    });
  }, []);

  const notifyUpdated = useCallback(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }, [t]);

  const handleSaveProfile = useCallback(async () => {
    if (!profileDirty) return;
    const paths: string[] = [];
    if (title !== user.title) paths.push("title");
    if (phone !== user.phone) paths.push("phone");
    setSavingProfile(true);
    try {
      const updated = await updateUser(
        create(UpdateUserRequestSchema, {
          user: { ...user, title, phone },
          updateMask: create(FieldMaskSchema, { paths }),
        })
      );
      setCurrentUser(updated);
      notifyUpdated();
    } catch (error) {
      notifyError(error);
    } finally {
      setSavingProfile(false);
    }
  }, [
    profileDirty,
    title,
    phone,
    user,
    updateUser,
    notifyUpdated,
    notifyError,
  ]);

  const handleUpdatePassword = useCallback(async () => {
    if (!allowSavePassword) return;
    setSavingPassword(true);
    try {
      const updated = await updateUser(
        create(UpdateUserRequestSchema, {
          user: { ...user, password },
          updateMask: create(FieldMaskSchema, { paths: ["password"] }),
        })
      );
      setCurrentUser(updated);
      setPassword("");
      setPasswordConfirm("");
      notifyUpdated();
    } catch (error) {
      notifyError(error);
    } finally {
      setSavingPassword(false);
    }
  }, [
    allowSavePassword,
    password,
    user,
    updateUser,
    notifyUpdated,
    notifyError,
  ]);

  const handleChangeEmail = useCallback(async () => {
    if (!newEmail || newEmail === user.email) return;
    setSavingEmail(true);
    try {
      const oldEmail = user.email;
      const updated = await updateEmail(oldEmail, newEmail);
      migrateUserStorage(oldEmail, newEmail);
      updateCurrentUserNameForEmailChange(updated.name);
      setCurrentUser(updated);
      setShowChangeEmail(false);
      notifyUpdated();
    } catch (error) {
      notifyError(error);
    } finally {
      setSavingEmail(false);
    }
  }, [
    newEmail,
    user.email,
    updateEmail,
    updateCurrentUserNameForEmailChange,
    notifyUpdated,
    notifyError,
  ]);

  const handleEnable2FA = useCallback(() => {
    if (!has2FAFeature) {
      setShowFeatureModal(true);
      return;
    }
    router.push({ name: SETTING_ROUTE_PROFILE_TWO_FACTOR });
  }, [has2FAFeature]);

  const handleRequestDisable2FA = useCallback(() => {
    // A workspace-wide 2FA requirement outranks the individual preference;
    // only an admin who can relax the policy may turn their own factor off.
    if (requireMfa && !hasWorkspacePermissionV2("bb.policies.update")) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("two-factor.messages.cannot-disable"),
      });
      return;
    }
    setShowDisable2FAConfirm(true);
  }, [requireMfa, t]);

  const handleDisable2FA = useCallback(async () => {
    try {
      const updated = await updateUser(
        create(UpdateUserRequestSchema, {
          user: { name: user.name, mfaEnabled: false },
          updateMask: create(FieldMaskSchema, { paths: ["mfa_enabled"] }),
        })
      );
      setCurrentUser(updated);
      setShowDisable2FAConfirm(false);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("two-factor.messages.2fa-disabled"),
      });
    } catch (error) {
      notifyError(error);
    }
  }, [user.name, updateUser, notifyError, t]);

  return (
    <WorkspacePageLayout>
      <div className="flex flex-col gap-y-1">
        <h1 className="text-xl font-medium text-main">
          {t("settings.account.self")}
        </h1>
        <p className="text-sm text-control-light">
          {t("settings.account.description")}
        </p>
      </div>

      {/* Profile */}
      <FormSection title={t("settings.account.profile")}>
        <div className="flex items-start gap-x-4">
          <div
            className="flex size-14 shrink-0 items-center justify-center rounded-full text-lg font-bold text-white"
            style={{ backgroundColor: getAvatarColor(user.email) }}
          >
            {getInitials(title || user.email)}
          </div>
          <FormFieldGroup className="flex-1">
            <FormField title={t("settings.profile.display-name")}>
              <Input
                value={title}
                autoComplete="off"
                maxLength={200}
                placeholder={t("settings.profile.display-name-placeholder")}
                onChange={(e) => setTitle(e.target.value)}
              />
            </FormField>
            {isEndUser && (
              <FormField
                title={t("settings.profile.phone")}
                description={t("settings.profile.phone-tips")}
              >
                <Input
                  value={phone}
                  type="tel"
                  autoComplete="off"
                  onChange={(e) => setPhone(e.target.value)}
                />
              </FormField>
            )}
            <div className="flex justify-end">
              <Button
                disabled={!profileDirty || savingProfile}
                onClick={handleSaveProfile}
              >
                {t("common.save")}
              </Button>
            </div>
          </FormFieldGroup>
        </div>
      </FormSection>

      {/* Sign-in identity */}
      <FormSection title={t("settings.account.sign-in")}>
        <FormFieldGroup>
          <FormField
            title={t("settings.profile.email")}
            description={
              allowChangeEmail ? undefined : t("settings.account.email-managed")
            }
          >
            <div className="flex items-center gap-x-3">
              <span className="text-sm text-main">{user.email}</span>
              {allowChangeEmail && (
                <Button
                  appearance="outline"
                  size="sm"
                  onClick={() => {
                    setNewEmail(user.email);
                    setShowChangeEmail(true);
                  }}
                >
                  {t("settings.account.change-email")}
                </Button>
              )}
            </div>
          </FormField>

          <FormField title={t("settings.profile.role")}>
            <div className="flex flex-wrap items-center gap-2">
              {userRoles.length > 0 ? (
                sortRoles(userRoles).map((role) => (
                  <Badge key={role}>
                    {displayRoleTitleFromList(role, roleList)}
                  </Badge>
                ))
              ) : (
                <span className="text-sm text-control-light">-</span>
              )}
            </div>
            {!hasFeature(PlanFeature.FEATURE_IAM) && (
              <RouterLink
                to={{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }}
                className="normal-link"
              >
                {t("settings.profile.subscription")}
              </RouterLink>
            )}
          </FormField>
        </FormFieldGroup>
      </FormSection>

      {/* Password */}
      {isEndUser && (
        <FormSection title={t("settings.profile.password")}>
          <FormFieldGroup>
            <Alert
              variant="info"
              title={t("settings.account.password-notice")}
            />
            <UserPasswordSection
              password={password}
              passwordConfirm={passwordConfirm}
              onPasswordChange={setPassword}
              onPasswordConfirmChange={setPasswordConfirm}
              passwordRestriction={passwordRestriction}
              required={false}
            />
            <div className="flex justify-end">
              <Button
                disabled={!allowSavePassword || savingPassword}
                onClick={handleUpdatePassword}
              >
                {t("settings.account.update-password")}
              </Button>
            </div>
          </FormFieldGroup>
        </FormSection>
      )}

      {/* Two-factor authentication */}
      {isEndUser && (
        <FormSection
          title={
            <span className="flex flex-row items-center justify-start">
              {t("two-factor.self")}
              <FeatureBadge
                feature={PlanFeature.FEATURE_TWO_FA}
                className="ml-2 inline-flex text-accent"
              />
            </span>
          }
        >
          <FormFieldGroup>
            <div className="flex items-start justify-between gap-x-4">
              <p className="text-sm text-control-light">
                {t("two-factor.description")}{" "}
                <LearnMoreLink
                  href="https://docs.bytebase.com/administration/2fa?source=console"
                  className="ml-1 text-accent"
                />
              </p>
              <div className="flex shrink-0 gap-x-2">
                {isMFAEnabled && (
                  <Button
                    variant="destructive"
                    onClick={handleRequestDisable2FA}
                  >
                    {t("common.disable")}
                  </Button>
                )}
                <Button appearance="outline" onClick={handleEnable2FA}>
                  {isMFAEnabled ? t("common.edit") : t("common.enable")}
                </Button>
              </div>
            </div>

            {isMFAEnabled && (
              <div className="flex flex-col gap-y-2 border-t pt-4">
                <div className="flex w-full flex-row items-center justify-between">
                  <span className="font-medium">
                    {t("two-factor.recovery-codes.self")}
                  </span>
                  {!showRegenerateView && (
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        aria-label={t("common.more")}
                        className="cursor-pointer rounded-xs p-1 outline-hidden hover:bg-control-bg focus-visible:ring-2 focus-visible:ring-accent"
                      >
                        <Ellipsis className="w-8" />
                      </DropdownMenuTrigger>
                      <DropdownMenuContent className="min-w-36">
                        <DropdownMenuItem
                          className="py-1.5"
                          onClick={() => setShowRegenerateView(true)}
                        >
                          {t("common.regenerate")}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                </div>
                <p className="text-sm text-control-light">
                  {t("two-factor.recovery-codes.description")}
                </p>
                {showRegenerateView && (
                  <RegenerateRecoveryCodesView
                    recoveryCodes={currentUser.tempRecoveryCodes}
                    onClose={() => setShowRegenerateView(false)}
                  />
                )}
              </div>
            )}
          </FormFieldGroup>
        </FormSection>
      )}

      <FeatureModal
        open={showFeatureModal}
        feature={PlanFeature.FEATURE_TWO_FA}
        onOpenChange={setShowFeatureModal}
      />

      {/* Change my email */}
      <Dialog open={showChangeEmail} onOpenChange={setShowChangeEmail}>
        <DialogContent className="max-w-md">
          <DialogTitle>{t("settings.account.change-email")}</DialogTitle>
          <DialogDescription>
            {t("settings.account.change-email-description")}
          </DialogDescription>
          <div className="mt-4">
            <EmailInput value={newEmail} onChange={setNewEmail} />
          </div>
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              appearance="outline"
              disabled={savingEmail}
              onClick={() => setShowChangeEmail(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              disabled={savingEmail || !newEmail || newEmail === user.email}
              onClick={handleChangeEmail}
            >
              {t("common.update")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Disable my 2FA */}
      <Dialog
        open={showDisable2FAConfirm}
        onOpenChange={setShowDisable2FAConfirm}
      >
        <DialogContent className="max-w-md">
          <DialogTitle>{t("two-factor.disable.self")}</DialogTitle>
          <DialogDescription>
            {t("two-factor.disable.description")}
          </DialogDescription>
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              appearance="outline"
              onClick={() => setShowDisable2FAConfirm(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDisable2FA}>
              {t("common.disable")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </WorkspacePageLayout>
  );
}
