import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { Ellipsis } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { ACCOUNT_ROUTE_TWO_FACTOR } from "@/app/router/handles";
import { FeatureBadge } from "@/components/FeatureBadge";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import { getAvatarColor, getInitials } from "@/components/UserAvatar";
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
import { Input } from "@/components/ui/input";
import { WorkspacePageLayout } from "@/components/WorkspacePageLayout";
import { useCurrentUser } from "@/hooks/useAppState";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { RegenerateRecoveryCodesView } from "@/routes/workspace/two-factor/RegenerateRecoveryCodesView";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { AccountType, getAccountTypeByEmail } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2, setDocumentTitle } from "@/utils";
import { SettingsCard, SettingsRow } from "./SettingsCard";
import { getPasswordErrors, UserPasswordSection } from "./UserPasswordSection";

/**
 * Account settings for the signed-in user.
 *
 * Everything here is something a person does to their *own* account: rename
 * themselves, change how they are reached, rotate their password, enroll a
 * second factor. Administering somebody else's account is a different job, done
 * from the Users directory's row menu.
 */
export function AccountSettingsPage() {
  const { t } = useTranslation();

  const legacyCurrentUser = useCurrentUser();
  const [currentUser, setCurrentUser] = useState(legacyCurrentUser);

  const updateUser = useAppStore((state) => state.updateUser);
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
  const isMFAEnabled = user.mfaEnabled;

  // --- Profile section ---
  const [title, setTitle] = useState(user.title);
  const [phone, setPhone] = useState(user.phone);
  const [savingProfile, setSavingProfile] = useState(false);

  // --- Password section ---
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [savingPassword, setSavingPassword] = useState(false);

  // --- Password + 2FA dialogs ---
  const [showChangePassword, setShowChangePassword] = useState(false);
  const [showFeatureModal, setShowFeatureModal] = useState(false);
  const [showDisable2FAConfirm, setShowDisable2FAConfirm] = useState(false);
  const [showRegenerateView, setShowRegenerateView] = useState(false);
  const [disabling2FA, setDisabling2FA] = useState(false);

  useEffect(() => {
    setCurrentUser(legacyCurrentUser);
  }, [legacyCurrentUser]);

  // Re-seed the editable fields only when the signed-in account changes, not
  // on every store write: any self-update reissues `legacyCurrentUser`, and
  // resetting here would discard edits typed while a save was in flight.
  useEffect(() => {
    setTitle(legacyCurrentUser.title);
    setPhone(legacyCurrentUser.phone);
  }, [legacyCurrentUser.name]);

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
      setShowChangePassword(false);
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

  const handleEnable2FA = useCallback(() => {
    if (!has2FAFeature) {
      setShowFeatureModal(true);
      return;
    }
    router.push({ name: ACCOUNT_ROUTE_TWO_FACTOR });
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
    if (disabling2FA) return;
    setDisabling2FA(true);
    try {
      const updated = await updateUser(
        create(UpdateUserRequestSchema, {
          user: { name: user.name, mfaEnabled: false },
          updateMask: create(FieldMaskSchema, { paths: ["mfa_enabled"] }),
        })
      );
      setCurrentUser(updated);
      setShowDisable2FAConfirm(false);
      setShowRegenerateView(false);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("two-factor.messages.2fa-disabled"),
      });
    } catch (error) {
      notifyError(error);
    } finally {
      setDisabling2FA(false);
    }
  }, [disabling2FA, user.name, updateUser, notifyError, t]);

  return (
    <WorkspacePageLayout>
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-y-8">
        <div className="flex items-center gap-x-4">
          <div
            className="flex size-11 shrink-0 items-center justify-center rounded-full text-base font-bold text-white"
            style={{ backgroundColor: getAvatarColor(user.email) }}
          >
            {getInitials(title || user.email)}
          </div>
          <div className="flex min-w-0 flex-col gap-y-0.5">
            <h1 className="text-xl font-medium text-main">
              {t("settings.account.self")}
            </h1>
            <p className="text-sm text-control-light">
              {t("settings.account.description")}
            </p>
          </div>
        </div>

        {/* Who you are. Email leads because it identifies the account; the two
            fields you can actually change follow it.

            Workspace role is deliberately absent. It is not actionable here,
            it is already on the members page, and rendering it made a personal
            settings page read the entire workspace IAM policy — the full
            membership graph — to draw one badge. */}
        <div className="flex flex-col gap-y-3">
          <SettingsCard>
            <SettingsRow
              label={t("settings.profile.email")}
              description={t("settings.account.email-managed")}
            >
              <span className="text-sm text-main">{user.email}</span>
            </SettingsRow>

            <SettingsRow label={t("settings.profile.display-name")}>
              <Input
                value={title}
                autoComplete="off"
                maxLength={200}
                className="w-56"
                placeholder={t("settings.profile.display-name-placeholder")}
                onChange={(e) => setTitle(e.target.value)}
              />
            </SettingsRow>

            {isEndUser && (
              <SettingsRow
                label={t("settings.profile.phone")}
                description={t("settings.profile.phone-tips")}
              >
                <Input
                  value={phone}
                  type="tel"
                  autoComplete="off"
                  className="w-56"
                  onChange={(e) => setPhone(e.target.value)}
                />
              </SettingsRow>
            )}
          </SettingsCard>

          <div className="flex justify-end">
            <Button
              disabled={!profileDirty || savingProfile}
              onClick={handleSaveProfile}
            >
              {t("common.save")}
            </Button>
          </div>
        </div>

        {/* How you prove it is you. Password and two-factor answer the same
            question, so they share a group rather than getting a heading each. */}
        {isEndUser && (
          <div className="flex flex-col gap-y-3">
            <h2 className="text-sm font-medium text-main">
              {t("settings.account.security")}
            </h2>
            <SettingsCard>
              <SettingsRow
                label={t("settings.profile.password")}
                description={t("settings.account.password-notice")}
              >
                <Button
                  appearance="outline"
                  onClick={() => setShowChangePassword(true)}
                >
                  {t("settings.account.update-password")}
                </Button>
              </SettingsRow>

              <SettingsRow
                align="start"
                label={
                  <span className="flex items-center">
                    {t("two-factor.self")}
                    <FeatureBadge
                      feature={PlanFeature.FEATURE_TWO_FA}
                      className="ml-2 inline-flex text-accent"
                    />
                  </span>
                }
                description={
                  <>
                    {t("two-factor.description")}{" "}
                    <LearnMoreLink
                      href="https://docs.bytebase.com/administration/2fa?source=console"
                      className="ml-1 text-accent"
                    />
                  </>
                }
              >
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
              </SettingsRow>

              {isMFAEnabled && (
                <SettingsRow
                  align="start"
                  label={t("two-factor.recovery-codes.self")}
                  description={t("two-factor.recovery-codes.description")}
                >
                  {!showRegenerateView && (
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        aria-label={t("common.more")}
                        className="cursor-pointer rounded-xs p-1 outline-hidden hover:bg-control-bg focus-visible:ring-2 focus-visible:ring-accent"
                      >
                        <Ellipsis className="size-4" />
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
                </SettingsRow>
              )}
            </SettingsCard>

            {/* Gated on MFA too: disabling the second factor while this panel
                is open would otherwise leave it stranded on screen, offering
                to regenerate recovery codes the account no longer has. */}
            {isMFAEnabled && showRegenerateView && (
              <RegenerateRecoveryCodesView
                recoveryCodes={currentUser.tempRecoveryCodes}
                onClose={() => setShowRegenerateView(false)}
              />
            )}
          </div>
        )}

        <FeatureModal
          open={showFeatureModal}
          feature={PlanFeature.FEATURE_TWO_FA}
          onOpenChange={setShowFeatureModal}
        />

        {/* Change my email */}
        <Dialog
          open={showChangePassword}
          onOpenChange={(next) => {
            setShowChangePassword(next);
            if (!next) {
              setPassword("");
              setPasswordConfirm("");
            }
          }}
        >
          <DialogContent className="max-w-md">
            <DialogTitle>{t("settings.account.update-password")}</DialogTitle>
            <DialogDescription>
              {t("settings.account.password-notice")}
            </DialogDescription>
            <div className="mt-4">
              <UserPasswordSection
                password={password}
                passwordConfirm={passwordConfirm}
                onPasswordChange={setPassword}
                onPasswordConfirmChange={setPasswordConfirm}
                passwordRestriction={passwordRestriction}
                required={false}
              />
            </div>
            <div className="mt-4 flex justify-end gap-x-2">
              <Button
                appearance="outline"
                disabled={savingPassword}
                onClick={() => setShowChangePassword(false)}
              >
                {t("common.cancel")}
              </Button>
              <Button
                disabled={!allowSavePassword || savingPassword}
                onClick={handleUpdatePassword}
              >
                {t("settings.account.update-password")}
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
              <Button
                variant="destructive"
                disabled={disabling2FA}
                onClick={handleDisable2FA}
              >
                {t("common.disable")}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </WorkspacePageLayout>
  );
}
