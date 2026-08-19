import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { CircleAlert, CircleCheck, Eye, EyeOff } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { RoleSelect } from "@/components/RoleSelect";
import { Button } from "@/components/ui/button";
import { FormError, FormField, FormTitle } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tooltip } from "@/components/ui/tooltip";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { getUserEmailInBinding } from "@/types";
import { PresetRoleType } from "@/types/iam";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

function extractUserTitle(email: string): string {
  const atIndex = email.indexOf("@");
  return atIndex !== -1 ? email.substring(0, atIndex) : email;
}

export interface UserFormSheetProps {
  open: boolean;
  onClose: () => void;
  onCreated: (user: User) => void;
}

// Provisioning a new account: email, an optional starting password, and the
// roles it should carry. Changing an account that already exists is a
// different job with different consequences — see the Users row menu.
export function UserFormSheet(props: UserFormSheetProps) {
  const { open, onClose } = props;
  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <UserForm onClose={props.onClose} onCreated={props.onCreated} />
      </SheetContent>
    </Sheet>
  );
}

function UserForm({ onClose, onCreated }: Omit<UserFormSheetProps, "open">) {
  const { t } = useTranslation();
  const createUser = useAppStore((state) => state.createUser);
  const patchWorkspaceIamPolicy = useAppStore(
    (state) => state.patchWorkspaceIamPolicy
  );
  const passwordRestriction = useAppStore(
    (s) => s.getWorkspaceProfile().passwordRestriction
  );
  const enforceIdentityDomain = useAppStore(
    (s) => s.getWorkspaceProfile().enforceIdentityDomain
  );
  const workspaceDomains = useAppStore((s) => s.getWorkspaceProfile().domains);

  const [title, setTitle] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [roles, setRoles] = useState<string[]>([
    PresetRoleType.WORKSPACE_MEMBER,
  ]);
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
  const passwordMismatch =
    (password.length > 0 || passwordConfirm.length > 0) &&
    password !== passwordConfirm;

  const emailDomainValid = useMemo(() => {
    if (!enforceIdentityDomain || workspaceDomains.length === 0) return true;
    const atIdx = email.indexOf("@");
    if (atIdx < 0) return false;
    const domain = email.slice(atIdx + 1);
    return workspaceDomains.includes(domain);
  }, [email, enforceIdentityDomain, workspaceDomains]);

  const isFormValid =
    email.length > 0 && emailDomainValid && !passwordHint && !passwordMismatch;

  const allowConfirm = isFormValid;

  const hasPermission = hasWorkspacePermissionV2("bb.users.create");

  const handleSubmit = async () => {
    if (!allowConfirm || !hasPermission) return;
    setIsRequesting(true);
    try {
      await handleCreate();
    } catch {
      // error shown by store
    } finally {
      setIsRequesting(false);
    }
  };

  const handleCreate = async () => {
    // Empty password is valid on create; backend will generate a random password.
    const createdUser = await createUser({
      ...create(UserSchema, {}),
      title: title || extractUserTitle(email),
      email,
      phone,
      password,
    });
    if (roles.length > 0) {
      try {
        await patchWorkspaceIamPolicy([
          {
            member: getUserEmailInBinding(createdUser.email),
            roles,
          },
        ]);
      } catch (error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("settings.members.admin.create-user-roles-failed"),
          description: (error as ConnectError).message,
        });
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

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("settings.members.admin.create-user")}</SheetTitle>
      </SheetHeader>

      <SheetBody>
        <div className="flex flex-col gap-y-6">
          {/* Name */}
          <FormField>
            <FormTitle id="user-form-name-title">{t("common.name")}</FormTitle>
            <Input
              id="user-form-name"
              aria-labelledby="user-form-name-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("common.name")}
              maxLength={200}
            />
          </FormField>

          {/* Email */}
          <FormField>
            <FormTitle id="user-form-email-title">
              {t("common.email")}
              <span className="ml-0.5 text-error">*</span>
            </FormTitle>
            <Input
              id="user-form-email"
              aria-labelledby="user-form-email-title"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </FormField>

          {/* Roles */}
          {hasWorkspacePermissionV2("bb.workspaces.setIamPolicy") && (
            <FormField>
              <FormTitle>{t("settings.members.table.roles")}</FormTitle>
              <RoleSelect value={roles} onChange={setRoles} disabled={false} />
            </FormField>
          )}

          {/* Phone */}
          <FormField>
            <FormTitle id="user-form-phone-title">
              {t("settings.profile.phone")}
            </FormTitle>
            <span className="text-sm text-control-placeholder">
              {t("settings.profile.phone-tips")}
            </span>
            <Input
              id="user-form-phone"
              aria-labelledby="user-form-phone-title"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              autoComplete="new-password"
            />
          </FormField>

          {/* Password */}
          <div className="flex flex-col gap-y-6">
            <FormField>
              <FormTitle id="user-form-password-title">
                {t("settings.profile.password")}
              </FormTitle>
              <span
                className={`flex items-center gap-x-1 text-sm text-control-placeholder ${
                  passwordHint ? "text-error" : ""
                }`}
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
              <div className="relative flex w-full items-center">
                <Input
                  id="user-form-password"
                  aria-labelledby="user-form-password-title"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                  placeholder={t("common.sensitive-placeholder")}
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
            </FormField>

            <FormField>
              <FormTitle id="user-form-password-confirm-title">
                {t("settings.profile.password-confirm")}
              </FormTitle>
              <div className="relative flex w-full items-center">
                <Input
                  id="user-form-password-confirm"
                  aria-labelledby="user-form-password-confirm-title"
                  type={showPassword ? "text" : "password"}
                  value={passwordConfirm}
                  onChange={(e) => setPasswordConfirm(e.target.value)}
                  autoComplete="new-password"
                  placeholder={t(
                    "settings.profile.password-confirm-placeholder"
                  )}
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
                <FormError className="pl-1">
                  {t("settings.profile.password-mismatch")}
                </FormError>
              )}
            </FormField>
          </div>
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
          {t("common.create")}
        </Button>
      </SheetFooter>
    </>
  );
}
