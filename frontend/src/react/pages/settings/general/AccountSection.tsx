import { create } from "@bufbuild/protobuf";
import { DurationSchema, FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { TriangleAlert } from "lucide-react";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useActuatorV1Store,
  useIdentityProviderStore,
  useSubscriptionV1Store,
} from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  defaultAccessTokenDurationInHours,
  defaultRefreshTokenDurationInHours,
} from "@/types";
import type { WorkspaceProfileSetting_PasswordRestriction } from "@/types/proto-es/v1/setting_service_pb";
import { WorkspaceProfileSetting_PasswordRestrictionSchema } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import type { SectionHandle } from "./useSettingSection";

const DEFAULT_MIN_LENGTH = 8;

interface ToggleState {
  disallowSignup: boolean;
  require2fa: boolean;
  disallowPasswordSignin: boolean;
}

interface TokenState {
  accessTokenDuration: number;
  accessTokenTimeFormat: "MINUTES" | "HOURS";
  refreshTokenDuration: number;
  refreshTokenTimeFormat: "HOURS" | "DAYS";
  inactiveTimeout: number;
}

interface AccountSectionProps {
  title: string;
  onDirtyChange: () => void;
}

export const AccountSection = forwardRef<SectionHandle, AccountSectionProps>(
  function AccountSection({ title, onDirtyChange }, ref) {
    const { t } = useTranslation();

    const settingV1Store = useSettingV1Store();
    const idpStore = useIdentityProviderStore();

    const isSaaSMode = useVueState(() => useActuatorV1Store().isSaaSMode);
    const hasDisallowSignupFeature = useVueState(() =>
      useSubscriptionV1Store().hasFeature(
        PlanFeature.FEATURE_DISALLOW_SELF_SERVICE_SIGNUP
      )
    );
    const has2FAFeature = useVueState(() =>
      useSubscriptionV1Store().hasFeature(PlanFeature.FEATURE_TWO_FA)
    );
    const hasDisallowPasswordSigninFeature = useVueState(() =>
      useSubscriptionV1Store().hasFeature(
        PlanFeature.FEATURE_DISALLOW_PASSWORD_SIGNIN
      )
    );
    const hasPasswordFeature = useVueState(() =>
      useSubscriptionV1Store().hasFeature(
        PlanFeature.FEATURE_PASSWORD_RESTRICTIONS
      )
    );
    const hasSecureTokenFeature = useVueState(() =>
      useSubscriptionV1Store().hasFeature(
        PlanFeature.FEATURE_TOKEN_DURATION_CONTROL
      )
    );

    const allowEdit = hasWorkspacePermissionV2(
      "bb.settings.setWorkspaceProfile"
    );

    const existActiveIdentityProvider = useVueState(
      () => idpStore.identityProviderList.length > 0
    );

    // Fetch identity providers on mount
    useEffect(() => {
      idpStore.fetchIdentityProviderList();
    }, []);

    // --- Toggle state ---
    const getInitialToggleState = useCallback((): ToggleState => {
      return {
        disallowSignup: settingV1Store.workspaceProfile.disallowSignup,
        require2fa: settingV1Store.workspaceProfile.require2fa,
        disallowPasswordSignin:
          settingV1Store.workspaceProfile.disallowPasswordSignin,
      };
    }, []);

    const [toggleState, setToggleState] = useState<ToggleState>(
      getInitialToggleState
    );

    // --- Password restriction state ---
    const getInitialPasswordRestriction =
      useCallback((): WorkspaceProfileSetting_PasswordRestriction => {
        return cloneDeep(
          settingV1Store.workspaceProfile.passwordRestriction ??
            create(WorkspaceProfileSetting_PasswordRestrictionSchema, {})
        );
      }, []);

    const [passwordState, setPasswordState] =
      useState<WorkspaceProfileSetting_PasswordRestriction>(
        getInitialPasswordRestriction
      );

    // --- Token duration state ---
    const getInitialTokenState = useCallback((): TokenState => {
      const state: TokenState = {
        accessTokenDuration: defaultAccessTokenDurationInHours,
        accessTokenTimeFormat: "HOURS",
        refreshTokenDuration: defaultRefreshTokenDurationInHours / 24,
        refreshTokenTimeFormat: "DAYS",
        inactiveTimeout: -1,
      };

      const accessTokenSeconds = settingV1Store.workspaceProfile
        .accessTokenDuration?.seconds
        ? Number(settingV1Store.workspaceProfile.accessTokenDuration.seconds)
        : undefined;
      if (accessTokenSeconds && accessTokenSeconds > 0) {
        if (accessTokenSeconds < 60 * 60) {
          state.accessTokenDuration = Math.floor(accessTokenSeconds / 60) || 1;
          state.accessTokenTimeFormat = "MINUTES";
        } else {
          state.accessTokenDuration =
            Math.floor(accessTokenSeconds / (60 * 60)) || 1;
          state.accessTokenTimeFormat = "HOURS";
        }
      }

      const refreshTokenSeconds = settingV1Store.workspaceProfile
        .refreshTokenDuration?.seconds
        ? Number(settingV1Store.workspaceProfile.refreshTokenDuration.seconds)
        : undefined;
      if (refreshTokenSeconds && refreshTokenSeconds > 0) {
        if (refreshTokenSeconds < 60 * 60 * 24) {
          state.refreshTokenDuration =
            Math.floor(refreshTokenSeconds / (60 * 60)) || 1;
          state.refreshTokenTimeFormat = "HOURS";
        } else {
          state.refreshTokenDuration =
            Math.floor(refreshTokenSeconds / (60 * 60 * 24)) || 1;
          state.refreshTokenTimeFormat = "DAYS";
        }
      }

      const inactiveTimeoutSeconds = Number(
        settingV1Store.workspaceProfile.inactiveSessionTimeout?.seconds ?? 0
      );
      if (inactiveTimeoutSeconds) {
        state.inactiveTimeout =
          Math.floor(inactiveTimeoutSeconds / (60 * 60)) || 0;
      }

      return state;
    }, []);

    const [tokenState, setTokenState] =
      useState<TokenState>(getInitialTokenState);

    // Clamp access token when switching to MINUTES
    const prevAccessFormat = useRef(tokenState.accessTokenTimeFormat);
    useEffect(() => {
      if (
        prevAccessFormat.current !== tokenState.accessTokenTimeFormat &&
        tokenState.accessTokenTimeFormat === "MINUTES" &&
        tokenState.accessTokenDuration > 59
      ) {
        setTokenState((s) => ({ ...s, accessTokenDuration: 59 }));
      }
      prevAccessFormat.current = tokenState.accessTokenTimeFormat;
    }, [tokenState.accessTokenTimeFormat, tokenState.accessTokenDuration]);

    // Clamp refresh token when switching to HOURS
    const prevRefreshFormat = useRef(tokenState.refreshTokenTimeFormat);
    useEffect(() => {
      if (
        prevRefreshFormat.current !== tokenState.refreshTokenTimeFormat &&
        tokenState.refreshTokenTimeFormat === "HOURS" &&
        tokenState.refreshTokenDuration > 23
      ) {
        setTokenState((s) => ({ ...s, refreshTokenDuration: 23 }));
      }
      prevRefreshFormat.current = tokenState.refreshTokenTimeFormat;
    }, [tokenState.refreshTokenTimeFormat, tokenState.refreshTokenDuration]);

    // --- Dirty checks ---
    const isToggleDirty = !isEqual(toggleState, getInitialToggleState());
    const isPasswordDirty = !isEqual(
      passwordState,
      getInitialPasswordRestriction()
    );
    const isTokenDirty = !isEqual(tokenState, getInitialTokenState());
    const isDirty = isToggleDirty || isPasswordDirty || isTokenDirty;

    // --- Update ---
    const handleUpdate = useCallback(async () => {
      // Password restriction
      if (!isEqual(passwordState, getInitialPasswordRestriction())) {
        await settingV1Store.updateWorkspaceProfile({
          payload: {
            passwordRestriction: create(
              WorkspaceProfileSetting_PasswordRestrictionSchema,
              { ...passwordState }
            ),
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.password_restriction"],
          }),
        });
      }

      // Token durations
      const initToken = getInitialTokenState();
      if (
        initToken.accessTokenDuration !== tokenState.accessTokenDuration ||
        initToken.accessTokenTimeFormat !== tokenState.accessTokenTimeFormat
      ) {
        const seconds =
          tokenState.accessTokenTimeFormat === "MINUTES"
            ? tokenState.accessTokenDuration * 60
            : tokenState.accessTokenDuration * 60 * 60;
        await settingV1Store.updateWorkspaceProfile({
          payload: {
            accessTokenDuration: create(DurationSchema, {
              seconds: BigInt(seconds),
              nanos: 0,
            }),
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.access_token_duration"],
          }),
        });
      }

      if (
        initToken.refreshTokenDuration !== tokenState.refreshTokenDuration ||
        initToken.refreshTokenTimeFormat !== tokenState.refreshTokenTimeFormat
      ) {
        const seconds =
          tokenState.refreshTokenTimeFormat === "HOURS"
            ? tokenState.refreshTokenDuration * 60 * 60
            : tokenState.refreshTokenDuration * 24 * 60 * 60;
        await settingV1Store.updateWorkspaceProfile({
          payload: {
            refreshTokenDuration: create(DurationSchema, {
              seconds: BigInt(seconds),
              nanos: 0,
            }),
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.refresh_token_duration"],
          }),
        });
      }

      if (initToken.inactiveTimeout !== tokenState.inactiveTimeout) {
        await settingV1Store.updateWorkspaceProfile({
          payload: {
            inactiveSessionTimeout: create(DurationSchema, {
              seconds: BigInt(tokenState.inactiveTimeout * 60 * 60),
              nanos: 0,
            }),
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.inactive_session_timeout"],
          }),
        });
      }

      // Toggle fields
      const initToggle = getInitialToggleState();
      const updateMaskPaths: string[] = [];
      if (toggleState.disallowSignup !== initToggle.disallowSignup) {
        updateMaskPaths.push("value.workspace_profile.disallow_signup");
      }
      if (toggleState.require2fa !== initToggle.require2fa) {
        updateMaskPaths.push("value.workspace_profile.require_2fa");
      }
      if (
        toggleState.disallowPasswordSignin !== initToggle.disallowPasswordSignin
      ) {
        updateMaskPaths.push(
          "value.workspace_profile.disallow_password_signin"
        );
      }
      if (updateMaskPaths.length > 0) {
        await settingV1Store.updateWorkspaceProfile({
          payload: {
            disallowSignup: toggleState.disallowSignup,
            require2fa: toggleState.require2fa,
            disallowPasswordSignin: toggleState.disallowPasswordSignin,
          },
          updateMask: create(FieldMaskSchema, { paths: updateMaskPaths }),
        });
      }
    }, [
      toggleState,
      passwordState,
      tokenState,
      getInitialToggleState,
      getInitialPasswordRestriction,
      getInitialTokenState,
    ]);

    // --- Revert ---
    const handleRevert = useCallback(() => {
      setToggleState(getInitialToggleState());
      setPasswordState(getInitialPasswordRestriction());
      setTokenState(getInitialTokenState());
    }, [
      getInitialToggleState,
      getInitialPasswordRestriction,
      getInitialTokenState,
    ]);

    useImperativeHandle(ref, () => ({
      isDirty: () => isDirty,
      update: handleUpdate,
      revert: handleRevert,
    }));

    useEffect(() => {
      onDirtyChange();
    }, [toggleState, passwordState, tokenState, onDirtyChange]);

    // --- Password restriction helpers ---
    const passwordRotationDays = useMemo(() => {
      if (!passwordState.passwordRotation) return 0;
      return Number(passwordState.passwordRotation.seconds) / (24 * 60 * 60);
    }, [passwordState.passwordRotation]);

    const updatePasswordField = useCallback(
      (update: Partial<WorkspaceProfileSetting_PasswordRestriction>) => {
        if (!hasPasswordFeature) return;
        setPasswordState((prev) => ({ ...prev, ...update }));
      },
      [hasPasswordFeature]
    );

    const disabled = !allowEdit;

    return (
      <div id="account" className="py-6 lg:flex">
        <div className="text-left lg:w-1/4">
          <h1 className="text-2xl font-bold">{title}</h1>
        </div>
        <div className="flex-1 lg:px-4">
          {/* Sub-section 1: Disallow signup (non-SaaS only) */}
          {!isSaaSMode && (
            <>
              <div className="mb-7 mt-4 lg:mt-0">
                <div className="flex items-center gap-x-2">
                  <input
                    type="checkbox"
                    checked={toggleState.disallowSignup}
                    disabled={disabled || !hasDisallowSignupFeature}
                    onChange={(e) =>
                      setToggleState((s) => ({
                        ...s,
                        disallowSignup: e.target.checked,
                      }))
                    }
                  />
                  <span className="font-medium">
                    {t("settings.general.workspace.disallow-signup.enable")}
                  </span>
                </div>
                <div className="mt-1 mb-3 text-sm text-gray-400">
                  {t("settings.general.workspace.disallow-signup.description")}
                </div>
              </div>
              <hr />
            </>
          )}

          {/* Sub-section 2: Password Restriction */}
          <div className="mb-7 mt-4 lg:mt-0">
            <p className="font-medium flex flex-row justify-start items-center mb-2 gap-x-2">
              {t("settings.general.workspace.password-restriction.self")}
            </p>
            <div className="flex flex-col gap-y-3">
              <div className="flex items-center">
                <input
                  type="number"
                  className="w-24 mr-2 rounded border border-control-border px-2 py-1 text-sm"
                  value={passwordState.minLength || DEFAULT_MIN_LENGTH}
                  min={DEFAULT_MIN_LENGTH}
                  disabled={disabled || !hasPasswordFeature}
                  onChange={(e) => {
                    const val = parseInt(e.target.value, 10);
                    updatePasswordField({
                      minLength: Math.max(
                        Number.isNaN(val) ? DEFAULT_MIN_LENGTH : val,
                        DEFAULT_MIN_LENGTH
                      ),
                    });
                  }}
                />
                <span>
                  {t(
                    "settings.general.workspace.password-restriction.min-length",
                    {
                      min: passwordState.minLength || DEFAULT_MIN_LENGTH,
                    }
                  )}
                </span>
              </div>
              <label className="flex items-center gap-x-2">
                <input
                  type="checkbox"
                  checked={passwordState.requireNumber}
                  disabled={disabled || !hasPasswordFeature}
                  onChange={(e) =>
                    updatePasswordField({
                      requireNumber: e.target.checked,
                    })
                  }
                />
                {t(
                  "settings.general.workspace.password-restriction.require-number"
                )}
              </label>
              <label className="flex items-center gap-x-2">
                <input
                  type="checkbox"
                  checked={passwordState.requireLetter}
                  disabled={disabled || !hasPasswordFeature}
                  onChange={(e) =>
                    updatePasswordField({
                      requireLetter: e.target.checked,
                    })
                  }
                />
                {t(
                  "settings.general.workspace.password-restriction.require-letter"
                )}
              </label>
              <label className="flex items-center gap-x-2">
                <input
                  type="checkbox"
                  checked={passwordState.requireUppercaseLetter}
                  disabled={disabled || !hasPasswordFeature}
                  onChange={(e) =>
                    updatePasswordField({
                      requireUppercaseLetter: e.target.checked,
                    })
                  }
                />
                {t(
                  "settings.general.workspace.password-restriction.require-uppercase-letter"
                )}
              </label>
              <label className="flex items-center gap-x-2">
                <input
                  type="checkbox"
                  checked={passwordState.requireSpecialCharacter}
                  disabled={disabled || !hasPasswordFeature}
                  onChange={(e) =>
                    updatePasswordField({
                      requireSpecialCharacter: e.target.checked,
                    })
                  }
                />
                {t(
                  "settings.general.workspace.password-restriction.require-special-character"
                )}
              </label>
              <label className="flex items-center gap-x-2">
                <input
                  type="checkbox"
                  checked={passwordState.requireResetPasswordForFirstLogin}
                  disabled={disabled || !hasPasswordFeature}
                  onChange={(e) =>
                    updatePasswordField({
                      requireResetPasswordForFirstLogin: e.target.checked,
                    })
                  }
                />
                {t(
                  "settings.general.workspace.password-restriction.require-reset-password-for-first-login"
                )}
              </label>
              <label className="flex items-center gap-x-2">
                <input
                  type="checkbox"
                  checked={!!passwordState.passwordRotation}
                  disabled={disabled || !hasPasswordFeature}
                  onChange={(e) => {
                    if (e.target.checked) {
                      updatePasswordField({
                        passwordRotation: create(DurationSchema, {
                          seconds: BigInt(7 * 24 * 60 * 60),
                          nanos: 0,
                        }),
                      });
                    } else {
                      setPasswordState((prev) => ({
                        ...prev,
                        passwordRotation: undefined,
                      }));
                    }
                  }}
                />
                <span className="flex items-center gap-x-2">
                  {
                    t(
                      "settings.general.workspace.password-restriction.password-rotation"
                    ).split("{day}")[0]
                  }
                  {passwordState.passwordRotation ? (
                    <input
                      type="number"
                      className="w-24 mx-2 rounded border border-control-border px-2 py-1 text-sm"
                      value={passwordRotationDays}
                      min={1}
                      disabled={disabled || !hasPasswordFeature}
                      onClick={(e) => e.stopPropagation()}
                      onChange={(e) => {
                        const val = parseInt(e.target.value, 10);
                        updatePasswordField({
                          passwordRotation: create(DurationSchema, {
                            seconds: BigInt(
                              (Number.isNaN(val) || val < 1 ? 1 : val) *
                                24 *
                                60 *
                                60
                            ),
                            nanos: 0,
                          }),
                        });
                      }}
                    />
                  ) : (
                    <span className="mx-1">N</span>
                  )}
                  {
                    t(
                      "settings.general.workspace.password-restriction.password-rotation"
                    ).split("{day}")[1]
                  }
                </span>
              </label>
            </div>
          </div>

          <hr />

          {/* Sub-section 3: Require 2FA */}
          <div className="flex flex-col gap-y-7">
            <div className="mt-4 lg:mt-0">
              <div className="flex items-center gap-x-2">
                <input
                  type="checkbox"
                  checked={toggleState.require2fa}
                  disabled={disabled || !has2FAFeature}
                  onChange={(e) =>
                    setToggleState((s) => ({
                      ...s,
                      require2fa: e.target.checked,
                    }))
                  }
                />
                <span className="font-medium">
                  {t("settings.general.workspace.require-2fa.enable")}
                </span>
              </div>
              <div className="mt-1 text-sm text-gray-400">
                {t("settings.general.workspace.require-2fa.description")}
              </div>
            </div>

            {/* Sub-section 4: Disallow password signin (non-SaaS only) */}
            {!isSaaSMode && (
              <div className="lg:mt-0">
                <div className="flex items-center gap-x-2">
                  <input
                    type="checkbox"
                    checked={toggleState.disallowPasswordSignin}
                    disabled={
                      disabled ||
                      !hasDisallowPasswordSigninFeature ||
                      (!toggleState.disallowPasswordSignin &&
                        !existActiveIdentityProvider)
                    }
                    onChange={(e) =>
                      setToggleState((s) => ({
                        ...s,
                        disallowPasswordSignin: e.target.checked,
                      }))
                    }
                  />
                  <span className="font-medium flex items-center gap-x-2">
                    {t(
                      "settings.general.workspace.disallow-password-signin.enable"
                    )}
                    {hasDisallowPasswordSigninFeature &&
                      !toggleState.disallowPasswordSignin &&
                      !existActiveIdentityProvider && (
                        <span
                          title={t(
                            "settings.general.workspace.disallow-password-signin.require-sso-setup"
                          )}
                        >
                          <TriangleAlert className="w-4 text-warning" />
                        </span>
                      )}
                  </span>
                </div>
                <div className="mt-1 text-sm text-gray-400">
                  {t(
                    "settings.general.workspace.disallow-password-signin.description"
                  )}
                </div>
              </div>
            )}
          </div>

          <hr />

          {/* Sub-section 5: Token Duration */}
          {/* Access Token Duration */}
          <div className="mb-7 mt-4 lg:mt-0">
            <p className="font-medium flex flex-row justify-start items-center">
              <span className="mr-2">
                {t("settings.general.workspace.access-token-duration.self")}
              </span>
            </p>
            <p className="text-sm text-gray-400 mt-1">
              {t(
                "settings.general.workspace.access-token-duration.description"
              )}
            </p>
            <div className="mt-3 flex flex-row justify-start items-center gap-x-4">
              <input
                type="number"
                className="w-24 rounded border border-control-border px-2 py-1 text-sm"
                value={tokenState.accessTokenDuration}
                min={1}
                max={tokenState.accessTokenTimeFormat === "MINUTES" ? 59 : 23}
                disabled={disabled || !hasSecureTokenFeature}
                onChange={(e) => {
                  const val = parseInt(e.target.value, 10);
                  if (!Number.isNaN(val)) {
                    const max =
                      tokenState.accessTokenTimeFormat === "MINUTES" ? 59 : 23;
                    setTokenState((s) => ({
                      ...s,
                      accessTokenDuration: Math.max(1, Math.min(val, max)),
                    }));
                  }
                }}
              />
              <label className="flex items-center gap-x-1">
                <input
                  type="radio"
                  name="accessTokenTimeFormat"
                  value="MINUTES"
                  checked={tokenState.accessTokenTimeFormat === "MINUTES"}
                  disabled={disabled || !hasSecureTokenFeature}
                  onChange={() =>
                    setTokenState((s) => ({
                      ...s,
                      accessTokenTimeFormat: "MINUTES",
                    }))
                  }
                />
                {t("settings.general.workspace.access-token-duration.minutes")}
              </label>
              <label className="flex items-center gap-x-1">
                <input
                  type="radio"
                  name="accessTokenTimeFormat"
                  value="HOURS"
                  checked={tokenState.accessTokenTimeFormat === "HOURS"}
                  disabled={disabled || !hasSecureTokenFeature}
                  onChange={() =>
                    setTokenState((s) => ({
                      ...s,
                      accessTokenTimeFormat: "HOURS",
                    }))
                  }
                />
                {t("settings.general.workspace.access-token-duration.hours")}
              </label>
            </div>
          </div>

          {/* Refresh Token Duration */}
          <div className="mb-7 mt-4 lg:mt-0">
            <p className="font-medium flex flex-row justify-start items-center">
              <span className="mr-2">
                {t("settings.general.workspace.refresh-token-duration.self")}
              </span>
            </p>
            <p className="text-sm text-gray-400 mt-1">
              {t(
                "settings.general.workspace.refresh-token-duration.description"
              )}
            </p>
            <div className="mt-3 flex flex-row justify-start items-center gap-x-4">
              <input
                type="number"
                className="w-24 rounded border border-control-border px-2 py-1 text-sm"
                value={tokenState.refreshTokenDuration}
                min={1}
                max={
                  tokenState.refreshTokenTimeFormat === "HOURS" ? 23 : undefined
                }
                disabled={disabled || !hasSecureTokenFeature}
                onChange={(e) => {
                  const val = parseInt(e.target.value, 10);
                  if (!Number.isNaN(val)) {
                    const max =
                      tokenState.refreshTokenTimeFormat === "HOURS"
                        ? 23
                        : undefined;
                    setTokenState((s) => ({
                      ...s,
                      refreshTokenDuration: Math.max(
                        1,
                        max ? Math.min(val, max) : val
                      ),
                    }));
                  }
                }}
              />
              <label className="flex items-center gap-x-1">
                <input
                  type="radio"
                  name="refreshTokenTimeFormat"
                  value="HOURS"
                  checked={tokenState.refreshTokenTimeFormat === "HOURS"}
                  disabled={disabled || !hasSecureTokenFeature}
                  onChange={() =>
                    setTokenState((s) => ({
                      ...s,
                      refreshTokenTimeFormat: "HOURS",
                    }))
                  }
                />
                {t("settings.general.workspace.refresh-token-duration.hours")}
              </label>
              <label className="flex items-center gap-x-1">
                <input
                  type="radio"
                  name="refreshTokenTimeFormat"
                  value="DAYS"
                  checked={tokenState.refreshTokenTimeFormat === "DAYS"}
                  disabled={disabled || !hasSecureTokenFeature}
                  onChange={() =>
                    setTokenState((s) => ({
                      ...s,
                      refreshTokenTimeFormat: "DAYS",
                    }))
                  }
                />
                {t("settings.general.workspace.refresh-token-duration.days")}
              </label>
            </div>
          </div>

          {/* Inactive Session Timeout */}
          <div className="mb-7 mt-4 lg:mt-0">
            <p className="font-medium flex flex-row justify-start items-center">
              <span className="mr-2">
                {t("settings.general.workspace.inactive-session-timeout.self")}
              </span>
            </p>
            <p className="text-sm text-gray-400 mt-1">
              {t(
                "settings.general.workspace.inactive-session-timeout.description"
              )}{" "}
              <span className="font-semibold">
                {t("settings.general.workspace.no-limit")}
              </span>
            </p>
            <div className="mt-3 flex flex-row justify-start items-center gap-x-4">
              <input
                type="number"
                className="w-24 rounded border border-control-border px-2 py-1 text-sm"
                value={tokenState.inactiveTimeout}
                min={-1}
                disabled={disabled || !hasSecureTokenFeature}
                onChange={(e) => {
                  const val = parseInt(e.target.value, 10);
                  if (!Number.isNaN(val)) {
                    setTokenState((s) => ({
                      ...s,
                      inactiveTimeout: Math.max(-1, val),
                    }));
                  }
                }}
              />
              <span className="text-sm text-gray-500">
                {t("settings.general.workspace.inactive-session-timeout.hours")}
              </span>
            </div>
          </div>
        </div>
      </div>
    );
  }
);
