import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { Ellipsis } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
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
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { AccountType, getAccountTypeByEmail } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { EmailInput } from "../profile/EmailInput";
import { getPasswordErrors } from "../profile/UserPasswordSection";
import { generatePassword } from "./generatePassword";

type OpenDialog =
  | "none"
  | "reset-password"
  | "reset-mfa"
  | "change-email"
  | "lifecycle";

interface UserRowMenuProps {
  user: User;
  isSelf: boolean;
  /** `replaces` names an identity the update supersedes, if any. */
  onUserUpdated: (user: User, replaces?: User) => void;
  onEdit: (user: User) => void;
  onDeactivate: (user: User) => Promise<void>;
  onReactivate: (user: User) => Promise<void>;
}

/**
 * Administrative actions for one account, from the directory row.
 *
 * Recovery leads because it is why an admin opens this menu: somebody cannot
 * sign in. Everything below the separator is ordinary bookkeeping.
 */
export function UserRowMenu({
  user,
  isSelf,
  onUserUpdated,
  onEdit,
  onDeactivate,
  onReactivate,
}: UserRowMenuProps) {
  const { t } = useTranslation();
  const updateUser = useAppStore((state) => state.updateUser);
  const updateEmail = useAppStore((state) => state.updateEmail);
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());
  const requireMfa = useAppStore((s) => s.getWorkspaceProfile().requireMfa);
  const passwordRestriction = useAppStore(
    (s) => s.getWorkspaceProfile().passwordRestriction
  );

  const [dialog, setDialog] = useState<OpenDialog>("none");
  const [processing, setProcessing] = useState(false);

  // Reset-password state. `issuedPassword` doubles as the dialog's phase: once
  // set, the dialog shows the result instead of the form.
  const [password, setPassword] = useState("");
  const [issuedPassword, setIssuedPassword] = useState("");

  const [newEmail, setNewEmail] = useState("");

  const accountType = getAccountTypeByEmail(user.email);
  const isEndUser = accountType === AccountType.USER;
  const isDeactivated = user.state === State.DELETED;

  // Cloud owns the account record, so only role management remains — and that
  // lives on the members page, not here.
  //
  // A deactivated account is also off limits: UpdateUser and UpdateEmail both
  // reject a deleted user outright, so every one of these would fail. The only
  // thing to do with such an account is bring it back.
  const canAdminister = !isSaaSMode && isEndUser && !isDeactivated && !isSelf;
  const canUpdate =
    canAdminister && hasWorkspacePermissionV2("bb.users.update");
  const canUpdateEmail =
    canAdminister && hasWorkspacePermissionV2("bb.users.updateEmail");
  // DeleteUser and UndeleteUser are unimplemented in cloud, so offering them
  // for an end user there would only produce an error. Service accounts and
  // workload identities go through their own RPCs and are unaffected.
  const canChangeLifecycle = !isEndUser || !isSaaSMode;
  const canDeactivate =
    canChangeLifecycle &&
    !isSelf &&
    hasWorkspacePermissionV2(
      accountType === AccountType.SERVICE_ACCOUNT
        ? "bb.serviceAccounts.delete"
        : accountType === AccountType.WORKLOAD_IDENTITY
          ? "bb.workloadIdentities.delete"
          : "bb.users.delete"
    );
  const canReactivate =
    canChangeLifecycle &&
    hasWorkspacePermissionV2(
      accountType === AccountType.SERVICE_ACCOUNT
        ? "bb.serviceAccounts.undelete"
        : accountType === AccountType.WORKLOAD_IDENTITY
          ? "bb.workloadIdentities.undelete"
          : "bb.users.undelete"
    );

  // Recovery restores sign-in, which only means something for a live account.
  const showRecovery = canUpdate;
  const hasAnyAction =
    showRecovery ||
    canUpdate ||
    canUpdateEmail ||
    (isDeactivated ? canReactivate : canDeactivate);

  // In cloud the account itself is out of the workspace's hands, so say so
  // rather than rendering nothing and leaving the admin to wonder.
  const saasNotice = isSaaSMode && isEndUser;

  const closeDialog = () => {
    setDialog("none");
    setPassword("");
    setIssuedPassword("");
    setNewEmail("");
  };

  const passwordErrors = getPasswordErrors(
    password,
    password,
    passwordRestriction
  );

  const notifyFailure = (error: unknown) => {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: (error as ConnectError).message || t("common.error"),
    });
  };

  const handleResetPassword = async () => {
    setProcessing(true);
    try {
      const updated = await updateUser(
        create(UpdateUserRequestSchema, {
          user: { ...user, password },
          updateMask: create(FieldMaskSchema, { paths: ["password"] }),
        })
      );
      onUserUpdated(updated);
      setIssuedPassword(password);
    } catch (error) {
      notifyFailure(error);
    } finally {
      setProcessing(false);
    }
  };

  const handleResetMfa = async () => {
    setProcessing(true);
    try {
      const updated = await updateUser(
        create(UpdateUserRequestSchema, {
          user: { name: user.name, mfaEnabled: false },
          updateMask: create(FieldMaskSchema, { paths: ["mfa_enabled"] }),
        })
      );
      onUserUpdated(updated);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("two-factor.messages.2fa-disabled"),
      });
      closeDialog();
    } catch (error) {
      notifyFailure(error);
    } finally {
      setProcessing(false);
    }
  };

  const handleChangeEmail = async () => {
    setProcessing(true);
    try {
      const updated = await updateEmail(user.email, newEmail);
      // The resource name moved with the email, so the old row has to go.
      onUserUpdated(updated, user);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      closeDialog();
    } catch (error) {
      notifyFailure(error);
    } finally {
      setProcessing(false);
    }
  };

  const handleLifecycle = async () => {
    setProcessing(true);
    try {
      await (isDeactivated ? onReactivate(user) : onDeactivate(user));
      closeDialog();
    } catch {
      // Leave the dialog open on failure — closing it reads as success. The
      // caller has already surfaced the error, so no second toast here.
    } finally {
      setProcessing(false);
    }
  };

  if (!hasAnyAction && !saasNotice) return null;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              appearance="secondary"
              size="sm"
              aria-label={t("common.more")}
            >
              <Ellipsis className="size-4" />
            </Button>
          }
        />
        <DropdownMenuContent className="min-w-52">
          {saasNotice && (
            <div className="px-3 py-2 text-sm text-control-light">
              {t("settings.members.admin.saas-read-only")}
            </div>
          )}
          {showRecovery && (
            <>
              <DropdownMenuItem onClick={() => setDialog("reset-password")}>
                {t("settings.members.admin.reset-password")}
              </DropdownMenuItem>
              {user.mfaEnabled && (
                <DropdownMenuItem onClick={() => setDialog("reset-mfa")}>
                  {t("settings.members.admin.reset-mfa")}
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
            </>
          )}
          {canUpdate && (
            <DropdownMenuItem onClick={() => onEdit(user)}>
              {t("settings.members.admin.edit-user")}
            </DropdownMenuItem>
          )}
          {canUpdateEmail && (
            <DropdownMenuItem
              onClick={() => {
                setNewEmail(user.email);
                setDialog("change-email");
              }}
            >
              {t("settings.members.admin.change-email")}
            </DropdownMenuItem>
          )}
          {isDeactivated
            ? canReactivate && (
                <DropdownMenuItem onClick={() => setDialog("lifecycle")}>
                  {t("settings.members.admin.reactivate")}
                </DropdownMenuItem>
              )
            : canDeactivate && (
                <DropdownMenuItem
                  className="text-error"
                  onClick={() => setDialog("lifecycle")}
                >
                  {t("settings.members.admin.deactivate")}
                </DropdownMenuItem>
              )}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Reset password — two phases in one dialog: choose a password, then
          read it back once. There is no mail server to fall back on in a
          self-hosted deployment, so the admin has to carry it themselves. */}
      <Dialog
        open={dialog === "reset-password"}
        onOpenChange={(next) => {
          // From the moment the request goes out, closing loses a credential
          // that is about to become the account's only working one: the reset
          // lands server-side regardless, so an Escape or backdrop click mid
          // flight revokes the old password and discards the new one unseen.
          // Closing requires the explicit Done button.
          if (next || processing || issuedPassword) return;
          closeDialog();
        }}
      >
        <DialogContent className="max-w-md">
          <DialogTitle>
            {t("settings.members.admin.reset-password")}
          </DialogTitle>
          {issuedPassword ? (
            <>
              <DialogDescription>
                {t("settings.members.admin.reset-password-issued", {
                  user: user.title || user.email,
                })}
              </DialogDescription>
              <div className="mt-4 flex items-center gap-x-2">
                <Input
                  readOnly
                  value={issuedPassword}
                  className="font-mono"
                  aria-label={t("common.password")}
                />
                <CopyButton content={issuedPassword} size="md" />
              </div>
              <Alert
                className="mt-4"
                title={t("settings.members.admin.reset-password-deliver")}
              />
              <div className="mt-4 flex justify-end gap-x-2">
                <Button onClick={closeDialog}>{t("common.done")}</Button>
              </div>
            </>
          ) : (
            <>
              <DialogDescription>
                {t("settings.members.admin.reset-password-description", {
                  user: user.title || user.email,
                })}
              </DialogDescription>
              <div className="mt-4 flex items-center gap-x-2">
                <Input
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                  className="font-mono"
                  aria-label={t("common.password")}
                />
                <Button
                  appearance="outline"
                  onClick={() =>
                    setPassword(generatePassword(passwordRestriction))
                  }
                >
                  {t("settings.members.admin.generate")}
                </Button>
              </div>
              {password && passwordErrors.hasHint && (
                <p className="mt-2 text-sm text-error">
                  {t("settings.profile.password-hint")}
                </p>
              )}
              <div className="mt-4 flex justify-end gap-x-2">
                <Button
                  appearance="outline"
                  disabled={processing}
                  onClick={closeDialog}
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  disabled={processing || !password || passwordErrors.hasHint}
                  onClick={handleResetPassword}
                >
                  {t("settings.members.admin.reset-password")}
                </Button>
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>

      {/* Reset 2FA — the consequence differs by workspace policy, so the copy
          does too. */}
      <Dialog
        open={dialog === "reset-mfa"}
        onOpenChange={(next) => !next && closeDialog()}
      >
        <DialogContent className="max-w-md">
          <DialogTitle>{t("settings.members.admin.reset-mfa")}</DialogTitle>
          <DialogDescription>
            {t("settings.members.admin.reset-mfa-recovery-first")}
          </DialogDescription>
          <p className="mt-3 text-sm text-control">
            {requireMfa
              ? t("settings.members.admin.reset-mfa-required", {
                  user: user.title || user.email,
                })
              : t("settings.members.admin.reset-mfa-optional", {
                  user: user.title || user.email,
                })}
          </p>
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              appearance="outline"
              disabled={processing}
              onClick={closeDialog}
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

      {/* Change email — an identity change, so it confirms on its own. */}
      <Dialog
        open={dialog === "change-email"}
        onOpenChange={(next) => !next && closeDialog()}
      >
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
              onClick={closeDialog}
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

      {/* Deactivate / reactivate */}
      <Dialog
        open={dialog === "lifecycle"}
        onOpenChange={(next) => !next && closeDialog()}
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
              onClick={closeDialog}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant={isDeactivated ? "default" : "destructive"}
              disabled={processing}
              onClick={handleLifecycle}
            >
              {isDeactivated
                ? t("settings.members.admin.reactivate")
                : t("settings.members.admin.deactivate")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
