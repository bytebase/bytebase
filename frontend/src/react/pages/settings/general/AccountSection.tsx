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
import { FeatureBadge } from "@/react/components/FeatureBadge";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { Checkbox } from "@/react/components/ui/checkbox";
import { Input } from "@/react/components/ui/input";
import { NumberInput } from "@/react/components/ui/number-input";
import { usePlanFeature, useServerState } from "@/react/hooks/useAppState";
import { useVueState } from "@/react/hooks/useVueState";
import { useIdentityProviderStore } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  defaultAccessTokenDurationInHours,
  defaultRefreshTokenDurationInHours,
} from "@/types";
import type {
  WorkspaceProfileSetting,
  WorkspaceProfileSetting_PasswordRestriction,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  Setting_SettingName,
  WorkspaceProfileSetting_PasswordRestrictionSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { isDev } from "@/utils";
import type { SectionHandle } from "./useSettingSection";

const DEFAULT_MIN_LENGTH = 8;

interface ToggleState {
  disallowSignup: boolean;
  requireMfa: boolean;
  disallowPasswordSignin: boolean;
  allowEmailCodeSignin: boolean;
}

interface TokenState {
  // `null` represents an empty input while the user is typing; coerced to
  // sensible defaults (1 for durations, -1 for inactiveTimeout = "no limit")
  // on save and dirty comparison.
  accessTokenDuration: number | null;
  accessTokenTimeFormat: "MINUTES" | "HOURS";
  refreshTokenDuration: number | null;
  refreshTokenTimeFormat: "HOURS" | "DAYS";
  inactiveTimeout: number | null;
}

// Defaults used when the user leaves a duration field empty.
const EMPTY_TOKEN_DURATION_DEFAULT = 1;
const EMPTY_INACTIVE_TIMEOUT_DEFAULT = -1;

// Canonicalizes TokenState for both isDirty comparison and save: coerces
// transient `null` inputs to defaults and floors any fractional values (paste
// of "1.5" etc.) so the two paths always agree on what counts as a change.
function normalizeTokenState(state: TokenState): TokenState {
  const floorOr = (v: number | null, fallback: number): number =>
    Math.floor(v ?? fallback);
  return {
    ...state,
    accessTokenDuration: floorOr(
      state.accessTokenDuration,
      EMPTY_TOKEN_DURATION_DEFAULT
    ),
    refreshTokenDuration: floorOr(
      state.refreshTokenDuration,
      EMPTY_TOKEN_DURATION_DEFAULT
    ),
    inactiveTimeout: floorOr(
      state.inactiveTimeout,
      EMPTY_INACTIVE_TIMEOUT_DEFAULT
    ),
  };
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

    const { isSaaSMode, workspaceResourceName } = useServerState();
    const hasDisallowSignupFeature = usePlanFeature(
      PlanFeature.FEATURE_DISALLOW_SELF_SERVICE_SIGNUP
    );
    const has2FAFeature = usePlanFeature(PlanFeature.FEATURE_TWO_FA);
    const hasDisallowPasswordSigninFeature = usePlanFeature(
      PlanFeature.FEATURE_DISALLOW_PASSWORD_SIGNIN
    );
    const hasPasswordFeature = usePlanFeature(
      PlanFeature.FEATURE_PASSWORD_RESTRICTIONS
    );
    const hasSecureTokenFeature = usePlanFeature(
      PlanFeature.FEATURE_TOKEN_DURATION_CONTROL
    );

    const [allowEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);

    const existActiveIdentityProvider = useVueState(
      () => idpStore.identityProviderList.length > 0
    );

    // Track whether the EMAIL setting is configured (required to enable email-code signin).
    // Reactively reads from the Vue store so the value updates when EMAIL is (un)configured.
    const hasEmailSetting = useVueState(
      () => !!settingV1Store.getSettingByName(Setting_SettingName.EMAIL)
    );

    // Fetch identity providers after the workspace resource name is ready.
    useEffect(() => {
      if (workspaceResourceName) {
        idpStore.fetchIdentityProviderList(workspaceResourceName);
      }
    }, [idpStore, workspaceResourceName]);

    // Fetch EMAIL setting on mount.
    useEffect(() => {
      // Populate the EMAIL setting into the store cache; useVueState above
      // picks up the change reactively.
      settingV1Store.getOrFetchSettingByName(Setting_SettingName.EMAIL, true);
    }, [settingV1Store]);

    // --- Toggle state ---
    const getInitialToggleState = useCallback((): ToggleState => {
      return {
        disallowSignup: settingV1Store.workspaceProfile.disallowSignup,
        requireMfa: settingV1Store.workspaceProfile.requireMfa,
        disallowPasswordSignin:
          settingV1Store.workspaceProfile.disallowPasswordSignin,
        allowEmailCodeSignin:
          settingV1Store.workspaceProfile.allowEmailCodeSignin,
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
        tokenState.accessTokenDuration !== null &&
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
        tokenState.refreshTokenDuration !== null &&
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
    // Normalize transient empty inputs (`null`) before comparing so a
    // cleared-then-saved field doesn't leave the section permanently dirty.
    const isTokenDirty = !isEqual(
      normalizeTokenState(tokenState),
      getInitialTokenState()
    );
    const isDirty = isToggleDirty || isPasswordDirty || isTokenDirty;

    // --- Update ---
    const handleUpdate = useCallback(async () => {
      const updateMaskPaths: string[] = [];
      const payload: Partial<WorkspaceProfileSetting> = {};

      // Password restriction
      if (!isEqual(passwordState, getInitialPasswordRestriction())) {
        payload.passwordRestriction = create(
          WorkspaceProfileSetting_PasswordRestrictionSchema,
          { ...passwordState }
        );
        updateMaskPaths.push("value.workspace_profile.password_restriction");
      }

      // Token durations — `normalizeTokenState` coerces transient `null`
      // inputs to defaults and floors fractional values, so the resolved
      // shape matches what isDirty compared against.
      const initToken = getInitialTokenState();
      const resolvedToken = normalizeTokenState(tokenState);
      const accessTokenDuration = resolvedToken.accessTokenDuration as number;
      const refreshTokenDuration = resolvedToken.refreshTokenDuration as number;
      const inactiveTimeout = resolvedToken.inactiveTimeout as number;

      if (
        initToken.accessTokenDuration !== accessTokenDuration ||
        initToken.accessTokenTimeFormat !== tokenState.accessTokenTimeFormat
      ) {
        const seconds =
          tokenState.accessTokenTimeFormat === "MINUTES"
            ? accessTokenDuration * 60
            : accessTokenDuration * 60 * 60;
        payload.accessTokenDuration = create(DurationSchema, {
          seconds: BigInt(seconds),
          nanos: 0,
        });
        updateMaskPaths.push("value.workspace_profile.access_token_duration");
      }

      if (
        initToken.refreshTokenDuration !== refreshTokenDuration ||
        initToken.refreshTokenTimeFormat !== tokenState.refreshTokenTimeFormat
      ) {
        const seconds =
          tokenState.refreshTokenTimeFormat === "HOURS"
            ? refreshTokenDuration * 60 * 60
            : refreshTokenDuration * 24 * 60 * 60;
        payload.refreshTokenDuration = create(DurationSchema, {
          seconds: BigInt(seconds),
          nanos: 0,
        });
        updateMaskPaths.push("value.workspace_profile.refresh_token_duration");
      }

      if (initToken.inactiveTimeout !== inactiveTimeout) {
        payload.inactiveSessionTimeout = create(DurationSchema, {
          seconds: BigInt(inactiveTimeout * 60 * 60),
          nanos: 0,
        });
        updateMaskPaths.push(
          "value.workspace_profile.inactive_session_timeout"
        );
      }

      // Toggle fields
      const initToggle = getInitialToggleState();
      if (toggleState.disallowSignup !== initToggle.disallowSignup) {
        payload.disallowSignup = toggleState.disallowSignup;
        updateMaskPaths.push("value.workspace_profile.disallow_signup");
      }
      if (toggleState.requireMfa !== initToggle.requireMfa) {
        payload.requireMfa = toggleState.requireMfa;
        updateMaskPaths.push("value.workspace_profile.require_mfa");
      }
      if (
        toggleState.disallowPasswordSignin !== initToggle.disallowPasswordSignin
      ) {
        payload.disallowPasswordSignin = toggleState.disallowPasswordSignin;
        updateMaskPaths.push(
          "value.workspace_profile.disallow_password_signin"
        );
      }
      if (
        toggleState.allowEmailCodeSignin !== initToggle.allowEmailCodeSignin
      ) {
        payload.allowEmailCodeSignin = toggleState.allowEmailCodeSignin;
        updateMaskPaths.push("value.workspace_profile.allow_email_code_signin");
      }

      if (updateMaskPaths.length === 0) return;

      await settingV1Store.updateWorkspaceProfile({
        payload,
        updateMask: create(FieldMaskSchema, { paths: updateMaskPaths }),
      });

      // Reset local state from the (now-updated) Vue store so isDirty clears
      // and the parent's bottom bar disappears.
      setToggleState(getInitialToggleState());
      setPasswordState(getInitialPasswordRestriction());
      setTokenState(getInitialTokenState());
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
        <PermissionGuard
          permissions={["bb.settings.setWorkspaceProfile"]}
          display="block"
        >
          <div className="flex-1 lg:px-4">
            {/* Sub-section 1: Disallow signup (non-SaaS only) */}
            {!isSaaSMode && (
              <>
                <div className="mt-4 lg:mt-0">
                  <div className="flex items-center gap-x-2">
                    <Checkbox
                      checked={toggleState.disallowSignup}
                      disabled={disabled || !hasDisallowSignupFeature}
                      onCheckedChange={(checked) =>
                        setToggleState((s) => ({
                          ...s,
                          disallowSignup: checked,
                        }))
                      }
                    />
                    <span className="text-base font-semibold">
                      {t("settings.general.workspace.disallow-signup.enable")}
                    </span>
                    <FeatureBadge
                      feature={PlanFeature.FEATURE_DISALLOW_SELF_SERVICE_SIGNUP}
                    />
                  </div>
                  <div className="mt-1 mb-3 text-sm text-gray-400">
                    {t(
                      "settings.general.workspace.disallow-signup.description"
                    )}
                  </div>
                </div>
                <hr className="my-6" />
              </>
            )}

            {/* Sub-section 2: Password Restriction */}
            <div className="mt-4 lg:mt-0">
              <p className="text-base font-semibold flex flex-row justify-start items-center mb-2 gap-x-2">
                {t("settings.general.workspace.password-restriction.self")}
                <FeatureBadge
                  feature={PlanFeature.FEATURE_PASSWORD_RESTRICTIONS}
                />
              </p>
              <div className="flex flex-col gap-y-3">
                <div className="flex items-center">
                  <Input
                    type="number"
                    className="w-24 mr-2"
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
                  <Checkbox
                    checked={passwordState.requireNumber}
                    disabled={disabled || !hasPasswordFeature}
                    onCheckedChange={(checked) =>
                      updatePasswordField({
                        requireNumber: checked,
                      })
                    }
                  />
                  {t(
                    "settings.general.workspace.password-restriction.require-number"
                  )}
                </label>
                <label className="flex items-center gap-x-2">
                  <Checkbox
                    checked={passwordState.requireLetter}
                    disabled={disabled || !hasPasswordFeature}
                    onCheckedChange={(checked) =>
                      updatePasswordField({
                        requireLetter: checked,
                      })
                    }
                  />
                  {t(
                    "settings.general.workspace.password-restriction.require-letter"
                  )}
                </label>
                <label className="flex items-center gap-x-2">
                  <Checkbox
                    checked={passwordState.requireUppercaseLetter}
                    disabled={disabled || !hasPasswordFeature}
                    onCheckedChange={(checked) =>
                      updatePasswordField({
                        requireUppercaseLetter: checked,
                      })
                    }
                  />
                  {t(
                    "settings.general.workspace.password-restriction.require-uppercase-letter"
                  )}
                </label>
                <label className="flex items-center gap-x-2">
                  <Checkbox
                    checked={passwordState.requireSpecialCharacter}
                    disabled={disabled || !hasPasswordFeature}
                    onCheckedChange={(checked) =>
                      updatePasswordField({
                        requireSpecialCharacter: checked,
                      })
                    }
                  />
                  {t(
                    "settings.general.workspace.password-restriction.require-special-character"
                  )}
                </label>
                <label className="flex items-center gap-x-2">
                  <Checkbox
                    checked={passwordState.requireResetPasswordForFirstLogin}
                    disabled={disabled || !hasPasswordFeature}
                    onCheckedChange={(checked) =>
                      updatePasswordField({
                        requireResetPasswordForFirstLogin: checked,
                      })
                    }
                  />
                  {t(
                    "settings.general.workspace.password-restriction.require-reset-password-for-first-login"
                  )}
                </label>
                <label className="flex items-center gap-x-2">
                  <Checkbox
                    checked={!!passwordState.passwordRotation}
                    disabled={disabled || !hasPasswordFeature}
                    onCheckedChange={(checked) => {
                      if (checked) {
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
                      ).split("{{day}}")[0]
                    }
                    {passwordState.passwordRotation ? (
                      <Input
                        type="number"
                        className="w-24 mx-2"
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
                      ).split("{{day}}")[1]
                    }
                  </span>
                </label>
              </div>
            </div>

            <hr className="my-6" />

            {/* Sub-section 3: Require 2FA */}
            <div className="flex flex-col gap-y-7">
              <div className="mt-4 lg:mt-0">
                <div className="flex items-center gap-x-2">
                  <Checkbox
                    checked={toggleState.requireMfa}
                    disabled={disabled || !has2FAFeature}
                    onCheckedChange={(checked) =>
                      setToggleState((s) => ({
                        ...s,
                        requireMfa: checked,
                      }))
                    }
                  />
                  <span className="text-base font-semibold">
                    {t("settings.general.workspace.require-2fa.enable")}
                  </span>
                  <FeatureBadge feature={PlanFeature.FEATURE_TWO_FA} />
                </div>
                <div className="mt-1 text-sm text-gray-400">
                  {t("settings.general.workspace.require-2fa.description")}
                </div>
              </div>

              {/* Sub-section 4: Disallow password signin (non-SaaS only) */}
              {!isSaaSMode && (
                <div className="lg:mt-0">
                  <div className="flex items-center gap-x-2">
                    <Checkbox
                      checked={toggleState.disallowPasswordSignin}
                      disabled={
                        disabled ||
                        !hasDisallowPasswordSigninFeature ||
                        (!toggleState.disallowPasswordSignin &&
                          !existActiveIdentityProvider)
                      }
                      onCheckedChange={(checked) =>
                        setToggleState((s) => ({
                          ...s,
                          disallowPasswordSignin: checked,
                        }))
                      }
                    />
                    <span className="text-base font-semibold flex items-center gap-x-2">
                      {t(
                        "settings.general.workspace.disallow-password-signin.enable"
                      )}
                      <FeatureBadge
                        feature={PlanFeature.FEATURE_DISALLOW_PASSWORD_SIGNIN}
                      />
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

              {/* Sub-section 5: Allow email-code signin (non-SaaS only; requires EMAIL setting).
                  Hidden in production until the email configuration UI is built. */}
              {!isSaaSMode && isDev() && (
                <div className="lg:mt-0">
                  <div className="flex items-center gap-x-2">
                    <Checkbox
                      checked={toggleState.allowEmailCodeSignin}
                      disabled={
                        disabled ||
                        (!toggleState.allowEmailCodeSignin && !hasEmailSetting)
                      }
                      onCheckedChange={(checked) =>
                        setToggleState((s) => ({
                          ...s,
                          allowEmailCodeSignin: checked,
                        }))
                      }
                    />
                    <span className="text-base font-semibold flex items-center gap-x-2">
                      {t(
                        "settings.general.workspace.allow-email-code-signin.enable"
                      )}
                      {!toggleState.allowEmailCodeSignin &&
                        !hasEmailSetting && (
                          <span
                            title={t(
                              "settings.general.workspace.allow-email-code-signin.require-email-setting"
                            )}
                          >
                            <TriangleAlert className="w-4 text-warning" />
                          </span>
                        )}
                    </span>
                  </div>
                  <div className="mt-1 text-sm text-gray-400">
                    {t(
                      "settings.general.workspace.allow-email-code-signin.description"
                    )}
                  </div>
                </div>
              )}
            </div>

            <hr className="my-6" />

            {/* Sub-section 5: Token Duration */}
            {/* Access Token Duration */}
            <div className="mb-7 mt-4 lg:mt-0">
              <p className="text-base font-semibold flex flex-row justify-start items-center gap-x-2">
                <span>
                  {t("settings.general.workspace.access-token-duration.self")}
                </span>
                <FeatureBadge
                  feature={PlanFeature.FEATURE_TOKEN_DURATION_CONTROL}
                />
              </p>
              <p className="text-sm text-gray-400 mt-1">
                {t(
                  "settings.general.workspace.access-token-duration.description"
                )}
              </p>
              <div className="mt-3 flex flex-row justify-start items-center gap-x-4">
                <NumberInput
                  className="w-24"
                  value={tokenState.accessTokenDuration}
                  min={1}
                  max={tokenState.accessTokenTimeFormat === "MINUTES" ? 59 : 23}
                  step={1}
                  disabled={disabled || !hasSecureTokenFeature}
                  onValueChange={(v) =>
                    setTokenState((s) => ({ ...s, accessTokenDuration: v }))
                  }
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
                  {t(
                    "settings.general.workspace.access-token-duration.minutes"
                  )}
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
              <p className="text-base font-semibold flex flex-row justify-start items-center gap-x-2">
                <span>
                  {t("settings.general.workspace.refresh-token-duration.self")}
                </span>
                <FeatureBadge
                  feature={PlanFeature.FEATURE_TOKEN_DURATION_CONTROL}
                />
              </p>
              <p className="text-sm text-gray-400 mt-1">
                {t(
                  "settings.general.workspace.refresh-token-duration.description"
                )}
              </p>
              <div className="mt-3 flex flex-row justify-start items-center gap-x-4">
                <NumberInput
                  className="w-24"
                  value={tokenState.refreshTokenDuration}
                  min={1}
                  max={
                    tokenState.refreshTokenTimeFormat === "HOURS"
                      ? 23
                      : undefined
                  }
                  step={1}
                  disabled={disabled || !hasSecureTokenFeature}
                  onValueChange={(v) =>
                    setTokenState((s) => ({ ...s, refreshTokenDuration: v }))
                  }
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
            <div className="mt-4 lg:mt-0">
              <p className="text-base font-semibold flex flex-row justify-start items-center gap-x-2">
                <span>
                  {t(
                    "settings.general.workspace.inactive-session-timeout.self"
                  )}
                </span>
                <FeatureBadge
                  feature={PlanFeature.FEATURE_TOKEN_DURATION_CONTROL}
                />
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
                <NumberInput
                  className="w-24"
                  value={tokenState.inactiveTimeout}
                  min={-1}
                  step={1}
                  disabled={disabled || !hasSecureTokenFeature}
                  onValueChange={(v) =>
                    setTokenState((s) => ({ ...s, inactiveTimeout: v }))
                  }
                />
                <span className="text-sm text-gray-500">
                  {t(
                    "settings.general.workspace.inactive-session-timeout.hours"
                  )}
                </span>
              </div>
            </div>
          </div>
        </PermissionGuard>
      </div>
    );
  }
);
