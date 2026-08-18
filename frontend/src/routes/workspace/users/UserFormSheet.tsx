import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { CircleAlert, CircleCheck, Eye, EyeOff } from "lucide-react";
import { useMemo, useRef, useState } from "react";
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
import { getUserFullNameByType } from "@/stores/modules/v1/common";
import { getUserEmailInBinding } from "@/types";
import { PresetRoleType } from "@/types/iam";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  UpdateUserRequestSchema,
  UserSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

function extractUserTitle(email: string): string {
  const atIndex = email.indexOf("@");
  return atIndex !== -1 ? email.substring(0, atIndex) : email;
}

export interface UserFormSheetProps {
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
export function UserFormSheet(props: UserFormSheetProps) {
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
}: Omit<UserFormSheetProps, "open">) {
  const { t } = useTranslation();
  const createUser = useAppStore((state) => state.createUser);
  const updateUser = useAppStore((state) => state.updateUser);
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
    // Read a one-shot snapshot of the workspace role map at mount. The parent
    // page loads the workspace IAM policy, so it is populated by the time an
    // edit sheet opens; reading via getState keeps this baseline mount-only.
    const roles = useAppStore
      .getState()
      .workspaceUserMapToRoles()
      .get(getUserFullNameByType(user));
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
    email.length > 0 && emailDomainValid && !passwordHint && !passwordMismatch;

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
      updatedUser = await updateUser(
        create(UpdateUserRequestSchema, {
          user: payload,
          updateMask: create(FieldMaskSchema, { paths: updateMask }),
        })
      );
    }

    if (!isEqual([...initialRoles].sort(), [...roles].sort())) {
      await patchWorkspaceIamPolicy([
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
        <SheetTitle>
          {isEditMode
            ? t("settings.members.admin.edit-user")
            : t("settings.members.admin.create-user")}
        </SheetTitle>
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
              disabled={!allowUpdate}
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
              disabled={isEditMode}
            />
            {isEditMode && (
              <span className="text-sm text-control-placeholder">
                {t("settings.members.admin.email-immutable-tip")}
              </span>
            )}
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
              disabled={!allowUpdate}
            />
          </FormField>

          {/* Password */}
          <div className="flex flex-col gap-y-6">
            <FormField>
              <FormTitle id="user-form-password-title">
                {isEditMode
                  ? t("settings.members.admin.reset-password")
                  : t("settings.profile.password")}
              </FormTitle>
              {isEditMode && (
                <span className="text-sm text-control-placeholder">
                  {t("settings.members.admin.reset-password-tip")}
                </span>
              )}
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
          {isEditMode ? t("common.update") : t("common.create")}
        </Button>
      </SheetFooter>
    </>
  );
}
