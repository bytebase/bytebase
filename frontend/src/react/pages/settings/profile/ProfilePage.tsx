import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { cloneDeep, isEqual } from "lodash-es";
import { Ellipsis, ShieldAlert } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { t as vueT } from "@/plugins/i18n";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useVueState } from "@/react/hooks/useVueState";
import {
  getAvatarColor,
  getInitials,
} from "@/react/pages/settings/shared/UserAvatar";
import { RegenerateRecoveryCodesView } from "@/react/pages/settings/two-factor/RegenerateRecoveryCodesView";
import { router } from "@/router";
import {
  WORKSPACE_ROUTE_404,
  WORKSPACE_ROUTE_USER_PROFILE,
} from "@/router/dashboard/workspaceRoutes";
import {
  SETTING_ROUTE_PROFILE_TWO_FACTOR,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/router/dashboard/workspaceSetting";
import {
  hasFeature,
  pushNotification,
  useActuatorV1Store,
  useAuthStore,
  useCurrentUserV1,
  useSettingV1Store,
  useUserStore,
  useWorkspaceV1Store,
} from "@/store";
import {
  AccountType,
  ALL_USERS_USER_EMAIL,
  getAccountTypeByEmail,
  isValidUserName,
  unknownUser,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import {
  displayRoleTitle,
  hasWorkspacePermissionV2,
  setDocumentTitle,
  sortRoles,
} from "@/utils";
import { migrateUserStorage } from "@/utils/storage-migrate";
import { EmailInput } from "./EmailInput";
import { getPasswordErrors, UserPasswordSection } from "./UserPasswordSection";

interface ProfilePageProps {
  principalEmail?: string;
}

export function ProfilePage({ principalEmail }: ProfilePageProps) {
  const { t } = useTranslation();

  const authStore = useAuthStore();
  const settingV1Store = useSettingV1Store();
  const userStore = useUserStore();
  const workspaceStore = useWorkspaceV1Store();
  const actuatorStore = useActuatorV1Store();

  // --- Reactive Vue state ---
  const currentUser = useVueState(() => useCurrentUserV1().value);

  const user = useVueState(() => {
    if (principalEmail) {
      return userStore.getUserByIdentifier(principalEmail) ?? unknownUser();
    }
    return useCurrentUserV1().value;
  });

  const userRoles = useVueState(() => [
    ...workspaceStore.getWorkspaceRolesByName(user.name),
  ]);

  const passwordRestriction = useVueState(
    () => settingV1Store.workspaceProfile.passwordRestriction
  );

  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);

  const has2FAFeature = useVueState(() =>
    hasFeature(PlanFeature.FEATURE_TWO_FA)
  );

  const require2fa = useVueState(
    () => settingV1Store.workspaceProfile.require2fa
  );

  const tempRecoveryCodes = useVueState(() => currentUser.tempRecoveryCodes);

  // --- Derived values ---
  const isSelf = currentUser.name === user.name;

  const allowGet = isSelf || hasWorkspacePermissionV2("bb.users.get");

  const allowEdit = useMemo(() => {
    if (
      user.email === ALL_USERS_USER_EMAIL ||
      getAccountTypeByEmail(user.email) !== AccountType.USER
    ) {
      return false;
    }
    if (user.state !== State.ACTIVE) {
      return false;
    }
    if (isSelf) {
      return true;
    }
    return !isSaaSMode && hasWorkspacePermissionV2("bb.users.update");
  }, [user, isSelf, isSaaSMode]);

  const allowEditEmail = hasWorkspacePermissionV2("bb.users.updateEmail");

  const isMFAEnabled = user.mfaEnabled;
  const showRegenerateRecoveryCodes =
    isMFAEnabled && user.name === currentUser.name;

  // --- Local state ---
  const [editing, setEditing] = useState(false);
  const [editingUser, setEditingUser] = useState<User | undefined>(undefined);
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [saving, setSaving] = useState(false);
  const [showFeatureModal, setShowFeatureModal] = useState(false);
  const [showDisable2FAConfirm, setShowDisable2FAConfirm] = useState(false);
  const [showRegenerateView, setShowRegenerateView] = useState(false);
  const [showEllipsisMenu, setShowEllipsisMenu] = useState(false);

  const editNameRef = useRef<HTMLInputElement>(null);
  const ellipsisMenuRef = useRef<HTMLDivElement>(null);
  useClickOutside(ellipsisMenuRef, showEllipsisMenu, () =>
    setShowEllipsisMenu(false)
  );

  // --- Password validity ---
  const passwordErrors = useMemo(() => {
    if (!editingUser) return { hasHint: false, hasMismatch: false };
    return getPasswordErrors(
      editingUser.password ?? "",
      passwordConfirm,
      passwordRestriction
    );
  }, [editingUser, passwordConfirm, passwordRestriction]);

  const allowSaveEdit = useMemo(() => {
    if (!editingUser) return false;
    return (
      !isEqual(user, editingUser) &&
      !passwordErrors.hasHint &&
      !passwordErrors.hasMismatch
    );
  }, [user, editingUser, passwordErrors]);

  // --- Effects ---

  // On mount: validate account type and fetch user
  useEffect(() => {
    if (principalEmail) {
      const userType = getAccountTypeByEmail(principalEmail);
      if (userType !== AccountType.USER) {
        router.replace({ name: WORKSPACE_ROUTE_404 });
        return;
      }
      (async () => {
        const fetched = await userStore.getOrFetchUserByIdentifier({
          identifier: principalEmail,
          fallback: false,
        });
        if (!isValidUserName(fetched.name)) {
          router.replace({ name: WORKSPACE_ROUTE_404 });
        }
      })();
    }
  }, [principalEmail]);

  // Keyboard shortcuts — refs are assigned after callbacks are defined below
  const saveEditRef = useRef<() => void>(() => {});
  const cancelEditRef = useRef<() => void>(() => {});

  // Route change guard
  useEffect(() => {
    const shouldGuard = editing && allowSaveEdit;

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (shouldGuard) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", handleBeforeUnload);

    const removeGuard = router.beforeEach((_to, _from, next) => {
      if (shouldGuard) {
        if (!window.confirm(vueT("common.leave-without-saving"))) {
          next(false);
          return;
        }
      }
      next();
    });

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
      removeGuard();
    };
  }, [editing, allowSaveEdit]);

  // Document title
  useEffect(() => {
    setDocumentTitle(user.title);
  }, [user.title]);

  // --- Actions ---
  const editUser = useCallback(() => {
    setEditingUser(cloneDeep(user));
    setEditing(true);
    setPasswordConfirm("");
    setTimeout(() => editNameRef.current?.focus(), 0);
  }, [user]);

  const cancelEdit = useCallback(() => {
    setEditingUser(undefined);
    setEditing(false);
  }, []);

  const updateEditingUser = useCallback(
    <K extends keyof User>(field: K, value: User[K]) => {
      setEditingUser((prev) => {
        if (!prev) return prev;
        return { ...prev, [field]: value };
      });
    },
    []
  );

  const saveEdit = useCallback(async () => {
    if (!editingUser) return;

    const emailChanged = editingUser.email !== user.email;
    const updateMaskPaths: string[] = [];

    if (editingUser.title !== user.title) {
      updateMaskPaths.push("title");
    }
    if (editingUser.phone !== user.phone) {
      updateMaskPaths.push("phone");
    }
    if (editingUser.password !== "") {
      updateMaskPaths.push("password");
    }

    setSaving(true);
    try {
      if (emailChanged) {
        const oldEmail = user.email;
        const updatedUser = await userStore.updateEmail(
          oldEmail,
          editingUser.email
        );
        migrateUserStorage(oldEmail, editingUser.email);
        if (isSelf) {
          authStore.updateCurrentUserNameForEmailChange(updatedUser.name);
        }
      }

      if (updateMaskPaths.length > 0) {
        await userStore.updateUser(
          create(UpdateUserRequestSchema, {
            user: editingUser,
            updateMask: create(FieldMaskSchema, {
              paths: updateMaskPaths,
            }),
            regenerateRecoveryCodes: false,
            regenerateTempMfaSecret: false,
          })
        );
      }

      if (emailChanged || updateMaskPaths.length > 0) {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      }
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: (error as ConnectError).message || "Failed to update user",
      });
      return;
    } finally {
      setSaving(false);
    }

    setEditingUser(undefined);
    setEditing(false);

    if (emailChanged && principalEmail) {
      router.replace({
        name: WORKSPACE_ROUTE_USER_PROFILE,
        params: { principalEmail: editingUser.email },
      });
    }
  }, [editingUser, user, isSelf, principalEmail, t]);

  // Assign refs for keyboard shortcuts now that callbacks are defined
  saveEditRef.current = saveEdit;
  cancelEditRef.current = cancelEdit;

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.isComposing) return;
      if (editing) {
        if (e.code === "Escape") {
          cancelEditRef.current();
        } else if (e.code === "Enter" && e.metaKey) {
          if (allowSaveEdit) {
            saveEditRef.current();
          }
        }
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [editing, allowSaveEdit]);

  const enable2FA = useCallback(() => {
    if (!has2FAFeature) {
      setShowFeatureModal(true);
      return;
    }
    router.push({ name: SETTING_ROUTE_PROFILE_TWO_FACTOR });
  }, [has2FAFeature]);

  const disable2FA = useCallback(() => {
    if (require2fa && !hasWorkspacePermissionV2("bb.policies.update")) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("two-factor.messages.cannot-disable"),
      });
    } else {
      setShowDisable2FAConfirm(true);
    }
  }, [require2fa, t]);

  const handleDisable2FA = useCallback(async () => {
    await userStore.updateUser(
      create(UpdateUserRequestSchema, {
        user: {
          name: user.name,
          mfaEnabled: false,
        },
        updateMask: create(FieldMaskSchema, {
          paths: ["mfa_enabled"],
        }),
      })
    );
    setShowDisable2FAConfirm(false);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("two-factor.messages.2fa-disabled"),
    });
  }, [user.name, t]);

  // --- No permission ---
  if (!allowGet) {
    return (
      <main className="pt-4 flex-1 h-full relative pb-8 focus:outline-hidden xl:order-last px-4">
        <div
          role="alert"
          className="relative w-full rounded-xs border px-4 py-3 text-sm flex gap-x-3 items-start border-error/30 bg-error/5 text-error"
        >
          <ShieldAlert className="h-5 w-5 shrink-0 mt-0.5" />
          <div className="flex flex-col gap-2">
            <h5 className="font-medium leading-tight">
              {t("common.missing-required-permission")}
            </h5>
            <div>
              {t("common.required-permission")}
              <ul className="list-disc pl-4">
                <li>bb.users.get</li>
              </ul>
            </div>
          </div>
        </div>
      </main>
    );
  }

  // --- Main render ---
  return (
    <main
      className="pt-4 flex-1 h-full relative pb-8 focus:outline-hidden xl:order-last"
      tabIndex={0}
    >
      <div>
        {/* Profile header */}
        <div>
          <div className="-mt-4 h-32 bg-accent lg:h-48" />
          <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="-mt-20 sm:flex sm:items-end sm:gap-x-5">
              {/* Large avatar */}
              <div
                className="rounded-full flex items-center justify-center text-white font-bold shrink-0 h-32 w-32 text-4xl ring-4 ring-white"
                style={{ backgroundColor: getAvatarColor(user.title) }}
              >
                {getInitials(user.title)}
              </div>
              <div className="mt-6 sm:flex-1 sm:min-w-0 sm:flex sm:items-center sm:justify-end sm:gap-x-6 sm:pb-1">
                <div className="mt-6 flex flex-row justify-stretch gap-x-2">
                  {allowEdit && (
                    <>
                      {editing ? (
                        <>
                          <Button
                            variant="ghost"
                            disabled={saving}
                            onClick={cancelEdit}
                          >
                            {t("common.cancel")}
                          </Button>
                          <Button
                            disabled={!allowSaveEdit || saving}
                            onClick={saveEdit}
                          >
                            {t("common.save")}
                          </Button>
                        </>
                      ) : (
                        <Button variant="outline" onClick={editUser}>
                          {t("common.edit")}
                        </Button>
                      )}
                    </>
                  )}
                </div>
              </div>
            </div>
            <div className="block mt-6 min-w-0 flex-1">
              {editing ? (
                <Input
                  ref={editNameRef}
                  autoComplete="off"
                  value={editingUser?.title ?? ""}
                  className="w-64 text-lg"
                  onChange={(e) => updateEditingUser("title", e.target.value)}
                />
              ) : (
                <h1 className="pb-1.5 text-2xl font-bold text-main truncate">
                  {user.title}
                </h1>
              )}
            </div>
          </div>
        </div>

        {/* Description list */}
        <div className="mt-6 mb-2 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
          <dl className="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-3">
            {/* Role */}
            <div className="sm:col-span-1">
              <dt className="text-sm font-medium text-control-light">
                {t("settings.profile.role")}
              </dt>
              <dd className="mt-1 text-sm text-main">
                <div className="flex flex-row justify-start items-start flex-wrap gap-2">
                  {sortRoles(userRoles).map((role) => (
                    <span
                      key={role}
                      className="inline-flex items-center rounded-full px-3 py-0.5 text-sm font-medium bg-control-bg text-control"
                    >
                      {displayRoleTitle(role)}
                    </span>
                  ))}
                </div>
                {!hasFeature(PlanFeature.FEATURE_IAM) && (
                  <a
                    href="#"
                    className="normal-link"
                    onClick={(e) => {
                      e.preventDefault();
                      router.push({
                        name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
                      });
                    }}
                  >
                    {t("settings.profile.subscription")}
                  </a>
                )}
              </dd>
            </div>

            {/* Email */}
            <div className="sm:col-span-1">
              <dt className="text-sm font-medium text-control-light">
                {t("settings.profile.email")}
              </dt>
              <dd className="mt-1 text-sm text-main">
                {editing && allowEditEmail ? (
                  <EmailInput
                    value={editingUser?.email ?? ""}
                    onChange={(val) => updateEditingUser("email", val)}
                  />
                ) : (
                  user.email
                )}
              </dd>
            </div>

            {/* Phone */}
            {getAccountTypeByEmail(user.email) === AccountType.USER && (
              <div className="sm:col-span-1">
                <dt className="text-sm font-medium text-control-light">
                  {t("settings.profile.phone")}
                </dt>
                <dd className="mt-1 text-sm text-main">
                  {editing ? (
                    <Input
                      value={editingUser?.phone ?? ""}
                      placeholder={t("settings.profile.phone-tips")}
                      autoComplete="off"
                      type="tel"
                      onChange={(e) =>
                        updateEditingUser("phone", e.target.value)
                      }
                    />
                  ) : (
                    user.phone
                  )}
                </dd>
              </div>
            )}

            {/* Password */}
            {editing && editingUser && (
              <div className="col-span-2">
                <UserPasswordSection
                  password={editingUser.password ?? ""}
                  passwordConfirm={passwordConfirm}
                  onPasswordChange={(val) => updateEditingUser("password", val)}
                  onPasswordConfirmChange={setPasswordConfirm}
                  passwordRestriction={passwordRestriction}
                />
              </div>
            )}
          </dl>
        </div>

        {/* 2FA section */}
        {allowEdit && (
          <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 border-t mt-16 pt-8 pb-4">
            <div className="w-full flex flex-row justify-between items-center">
              <span className="text-lg font-medium flex flex-row justify-start items-center">
                {t("two-factor.self")}
                <FeatureBadge
                  feature={PlanFeature.FEATURE_TWO_FA}
                  className="ml-2 text-accent inline-flex"
                />
              </span>
              <div className="flex gap-x-2">
                {isMFAEnabled && (
                  <Button variant="destructive" onClick={disable2FA}>
                    {t("common.disable")}
                  </Button>
                )}
                {user.email === currentUser.email && (
                  <Button variant="outline" onClick={enable2FA}>
                    {isMFAEnabled ? t("common.edit") : t("common.enable")}
                  </Button>
                )}
              </div>
            </div>
            <p className="mt-4 text-sm text-gray-500">
              {t("two-factor.description")}{" "}
              <a
                href="https://docs.bytebase.com/administration/2fa?source=console"
                target="_blank"
                rel="noopener noreferrer"
                className="text-accent hover:underline ml-1"
              >
                {t("common.learn-more")}
              </a>
            </p>

            {showRegenerateRecoveryCodes && (
              <>
                <div className="w-full flex flex-row justify-between items-center mt-8">
                  <span className="text-lg font-medium">
                    {t("two-factor.recovery-codes.self")}
                  </span>
                  {!showRegenerateView && (
                    <div className="relative" ref={ellipsisMenuRef}>
                      <button
                        type="button"
                        className="p-1 rounded hover:bg-control-bg"
                        onClick={() => setShowEllipsisMenu((v) => !v)}
                      >
                        <Ellipsis className="w-8" />
                      </button>
                      {showEllipsisMenu && (
                        <div className="absolute right-0 mt-1 z-10 bg-white border border-control-border rounded shadow-md py-1 min-w-36">
                          <button
                            type="button"
                            className="w-full text-left px-3 py-1.5 text-sm hover:bg-control-bg"
                            onClick={() => {
                              setShowRegenerateView(true);
                              setShowEllipsisMenu(false);
                            }}
                          >
                            {t("common.regenerate")}
                          </button>
                        </div>
                      )}
                    </div>
                  )}
                </div>
                <p className="mt-4 text-sm text-gray-500">
                  {t("two-factor.recovery-codes.description")}
                </p>
                {showRegenerateView && (
                  <RegenerateRecoveryCodesView
                    recoveryCodes={tempRecoveryCodes}
                    onClose={() => setShowRegenerateView(false)}
                  />
                )}
              </>
            )}
          </div>
        )}
      </div>

      {/* Feature modal for 2FA */}
      <Dialog open={showFeatureModal} onOpenChange={setShowFeatureModal}>
        <DialogContent className="p-6">
          <DialogTitle>{t("subscription.disabled-feature")}</DialogTitle>
          <DialogDescription className="mt-2">
            {t("subscription.require-subscription", {
              requiredPlan: t("subscription.plan.enterprise.title"),
            })}
          </DialogDescription>
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              variant="outline"
              onClick={() => setShowFeatureModal(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              onClick={() => {
                setShowFeatureModal(false);
                router.push({
                  name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
                });
              }}
            >
              {t("common.learn-more")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Disable 2FA confirm dialog */}
      <Dialog
        open={showDisable2FAConfirm}
        onOpenChange={setShowDisable2FAConfirm}
      >
        <DialogContent className="p-6">
          <DialogTitle>{t("two-factor.disable.self")}</DialogTitle>
          <DialogDescription className="mt-2">
            {t("two-factor.disable.description")}
          </DialogDescription>
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              variant="outline"
              onClick={() => setShowDisable2FAConfirm(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDisable2FA}>
              {t("common.confirm")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </main>
  );
}
