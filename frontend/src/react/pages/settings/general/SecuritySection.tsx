import { create } from "@bufbuild/protobuf";
import { DurationSchema, FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { X } from "lucide-react";
import {
  forwardRef,
  type KeyboardEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useId,
  useImperativeHandle,
  useState,
} from "react";
import { Trans, useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  FormField,
  FormFieldGroup,
  FormSection,
} from "@/react/components/ui/form";
import { Input } from "@/react/components/ui/input";
import { usePlanFeature } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { SectionHandle } from "./useSettingSection";

const DEFAULT_EXPIRATION_DAYS = 90;

interface ExpirationState {
  inputValue: number;
  neverExpire: boolean;
}

interface SecurityState {
  enableWatermark: boolean;
  maximumRoleExpiration: ExpirationState;
  maximumRequestExpiration: ExpirationState;
  domains: string[];
  enableRestriction: boolean;
}

interface SecuritySectionProps {
  title: string;
  onDirtyChange: () => void;
}

const getExpirationState = (
  secondsValue: bigint | number | undefined
): ExpirationState => {
  let inputValue = DEFAULT_EXPIRATION_DAYS;
  let neverExpire = true;
  const seconds = secondsValue ? Number(secondsValue) : undefined;
  if (seconds && seconds > 0) {
    inputValue =
      Math.floor(seconds / (60 * 60 * 24)) || DEFAULT_EXPIRATION_DAYS;
    neverExpire = false;
  }
  return { inputValue, neverExpire };
};

const getExpirationDuration = (expiration: ExpirationState) => {
  const seconds = expiration.neverExpire
    ? -1
    : expiration.inputValue * 24 * 60 * 60;
  return create(DurationSchema, {
    seconds: BigInt(seconds),
    nanos: 0,
  });
};

export const SecuritySection = forwardRef<SectionHandle, SecuritySectionProps>(
  function SecuritySection({ title, onDirtyChange }, ref) {
    const { t } = useTranslation();
    const workspaceProfile = useAppStore((s) => s.getWorkspaceProfile());

    const hasWatermarkFeature = usePlanFeature(PlanFeature.FEATURE_WATERMARK);
    const hasDomainRestrictionFeature = usePlanFeature(
      PlanFeature.FEATURE_USER_EMAIL_DOMAIN_RESTRICTION
    );
    const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);

    const getInitialState = useCallback((): SecurityState => {
      const profile = workspaceProfile;

      // Watermark
      const enableWatermark = profile.watermark;

      // Domain restriction
      const domains = Array.isArray(profile.domains)
        ? [...profile.domains]
        : [];
      const enableRestriction = profile.enforceIdentityDomain || false;

      return {
        enableWatermark,
        maximumRoleExpiration: getExpirationState(
          profile.maximumRoleExpiration?.seconds
        ),
        maximumRequestExpiration: getExpirationState(
          profile.maximumRequestExpiration?.seconds
        ),
        domains,
        enableRestriction,
      };
    }, [workspaceProfile]);

    const [state, setState] = useState<SecurityState>(getInitialState);
    const [domainInput, setDomainInput] = useState("");

    const validDomains = state.domains.filter((d) => !!d);
    const canToggleDomainRestriction =
      canEdit && validDomains.length > 0 && hasDomainRestrictionFeature;
    const membersRestrictionLabelId = useId();
    const membersRestrictionDescriptionId = useId();

    const isDirty = useCallback(() => {
      if (domainInput.trim()) return true;
      const init = getInitialState();
      const current = {
        ...state,
        domains: validDomains,
      };
      return !isEqual(current, init);
    }, [state, validDomains, getInitialState, domainInput]);

    const revert = useCallback(() => {
      setState(getInitialState());
      setDomainInput("");
    }, [getInitialState]);

    const update = useCallback(async () => {
      const init = getInitialState();

      // Watermark
      if (state.enableWatermark !== init.enableWatermark) {
        await useAppStore.getState().updateWorkspaceProfile({
          payload: { watermark: state.enableWatermark },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.watermark"],
          }),
        });
      }

      // Maximum role expiration
      if (!isEqual(state.maximumRoleExpiration, init.maximumRoleExpiration)) {
        await useAppStore.getState().updateWorkspaceProfile({
          payload: {
            maximumRoleExpiration: getExpirationDuration(
              state.maximumRoleExpiration
            ),
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.maximum_role_expiration"],
          }),
        });
      }

      // Maximum request expiration
      if (
        !isEqual(state.maximumRequestExpiration, init.maximumRequestExpiration)
      ) {
        await useAppStore.getState().updateWorkspaceProfile({
          payload: {
            maximumRequestExpiration: getExpirationDuration(
              state.maximumRequestExpiration
            ),
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.maximum_request_expiration"],
          }),
        });
      }

      // Domain restriction — include pending input
      const allDomains = domainInput.trim()
        ? [...state.domains, domainInput.trim()]
        : state.domains;
      const currentValidDomains = allDomains.filter((d) => !!d);
      const effectiveRestriction =
        currentValidDomains.length === 0 ? false : state.enableRestriction;

      const domainUpdatePaths: string[] = [];
      if (init.enableRestriction !== effectiveRestriction) {
        domainUpdatePaths.push(
          "value.workspace_profile.enforce_identity_domain"
        );
      }
      if (!isEqual(currentValidDomains, init.domains)) {
        domainUpdatePaths.push("value.workspace_profile.domains");
      }
      if (domainUpdatePaths.length > 0) {
        await useAppStore.getState().updateWorkspaceProfile({
          payload: {
            domains: currentValidDomains,
            enforceIdentityDomain: effectiveRestriction,
          },
          updateMask: create(FieldMaskSchema, {
            paths: domainUpdatePaths,
          }),
        });
      }

      // Pinia and the React app store both cache the workspace profile.
      // Pinia's computed updates automatically; the React store is a
      // load-once cache, so we refresh it here so consumers like
      // <Watermark /> and <BannersWrapper /> pick up the new values.
      await useAppStore.getState().loadWorkspaceProfile(true);
    }, [state, domainInput, getInitialState]);

    useImperativeHandle(ref, () => ({ isDirty, revert, update }));

    useEffect(() => {
      onDirtyChange();
    }, [state, domainInput, onDirtyChange]);

    const addDomain = () => {
      if (!domainInput.trim()) return;
      setState((prev) => ({
        ...prev,
        domains: [...prev.domains, domainInput.trim()],
      }));
      setDomainInput("");
    };

    const handleDomainKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        e.preventDefault();
        addDomain();
      }
    };

    const toggleDomainRestriction = () => {
      if (!canToggleDomainRestriction) return;
      setState((prev) => ({
        ...prev,
        enableRestriction: !prev.enableRestriction,
      }));
    };

    const handleDomainRestrictionTextKeyDown = (
      e: KeyboardEvent<HTMLDivElement>
    ) => {
      if (e.key !== "Enter" && e.key !== " ") return;
      e.preventDefault();
      toggleDomainRestriction();
    };

    const removeDomain = (index: number) => {
      setState((prev) => {
        const newDomains = prev.domains.filter((_, i) => i !== index);
        const newValidDomains = newDomains.filter((d) => !!d);
        return {
          ...prev,
          domains: newDomains,
          enableRestriction:
            newValidDomains.length === 0 ? false : prev.enableRestriction,
        };
      });
    };

    const renderExpirationField = (
      key: "maximumRoleExpiration" | "maximumRequestExpiration",
      title: string,
      description: ReactNode
    ) => {
      const expiration = state[key];
      return (
        <FormField title={title} description={description}>
          <div className="w-full flex flex-row">
            <div className="flex items-center gap-4">
              <div className="relative w-60">
                <Input
                  type="number"
                  className="pr-14"
                  value={expiration.inputValue}
                  min={1}
                  step={1}
                  disabled={!canEdit || expiration.neverExpire}
                  onChange={(e) => {
                    const val = Math.max(1, Math.floor(Number(e.target.value)));
                    if (!Number.isNaN(val)) {
                      setState((prev) => ({
                        ...prev,
                        [key]: {
                          ...prev[key],
                          inputValue: val,
                        },
                      }));
                    }
                  }}
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-500 pointer-events-none">
                  {t("settings.general.workspace.maximum-expiration.days")}
                </span>
              </div>
              <label className="flex items-center gap-x-2">
                <Checkbox
                  checked={expiration.neverExpire}
                  disabled={!canEdit}
                  onCheckedChange={(checked) =>
                    setState((prev) => ({
                      ...prev,
                      [key]: {
                        ...prev[key],
                        neverExpire: checked,
                      },
                    }))
                  }
                />
                <span>
                  {t(
                    "settings.general.workspace.maximum-expiration.never-expires"
                  )}
                </span>
              </label>
            </div>
          </div>
        </FormField>
      );
    };

    return (
      <FormSection id="security" title={title}>
        <PermissionGuard
          permissions={["bb.settings.setWorkspaceProfile"]}
          display="block"
        >
          <FormFieldGroup>
            {/* Watermark */}
            <FormField
              title={
                <span className="flex items-center gap-x-2">
                  <Checkbox
                    checked={state.enableWatermark}
                    disabled={!canEdit || !hasWatermarkFeature}
                    onCheckedChange={(checked) =>
                      setState((prev) => ({
                        ...prev,
                        enableWatermark: checked,
                      }))
                    }
                  />
                  {t("settings.general.workspace.watermark.enable")}
                  <FeatureBadge feature={PlanFeature.FEATURE_WATERMARK} />
                </span>
              }
              description={t(
                "settings.general.workspace.watermark.description"
              )}
            />

            {renderExpirationField(
              "maximumRoleExpiration",
              t("settings.general.workspace.maximum-role-expiration.self"),
              <Trans
                i18nKey="settings.general.workspace.maximum-role-expiration.description"
                components={{
                  highlight: <span className="font-medium text-warning" />,
                }}
              />
            )}

            {renderExpirationField(
              "maximumRequestExpiration",
              t("settings.general.workspace.maximum-request-expiration.self"),
              t(
                "settings.general.workspace.maximum-request-expiration.description"
              )
            )}

            {/* Domain Restriction */}
            <FormField
              id="domain-restriction"
              title={t("settings.general.workspace.domain-restriction.self")}
              description={t(
                "settings.general.workspace.domain-restriction.description"
              )}
            >
              <div className="w-full flex flex-col gap-2">
                {/* Domain tags + input */}
                <div className="flex flex-wrap items-center gap-2">
                  <Input
                    type="text"
                    className="min-w-[20rem]"
                    placeholder={t(
                      "settings.general.workspace.domain-restriction.domain-input-placeholder"
                    )}
                    value={domainInput}
                    disabled={!canEdit}
                    onChange={(e) => setDomainInput(e.target.value)}
                    onKeyDown={handleDomainKeyDown}
                    onBlur={addDomain}
                  />
                  {state.domains.map((domain, index) => (
                    <span
                      key={index}
                      className="inline-flex items-center gap-1 rounded-xs bg-gray-100 px-2 py-1.5 text-sm"
                    >
                      {domain}
                      <button
                        type="button"
                        className="text-gray-500 hover:text-gray-700 disabled:opacity-50"
                        disabled={!canEdit}
                        onClick={() => removeDomain(index)}
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    </span>
                  ))}
                </div>

                {/* Enforce restriction checkbox */}
                <div className="w-full flex flex-row justify-between items-center">
                  <FormField
                    title={
                      <div className="flex items-start gap-x-2">
                        <Checkbox
                          aria-describedby={membersRestrictionDescriptionId}
                          aria-labelledby={membersRestrictionLabelId}
                          checked={state.enableRestriction}
                          className="mt-1"
                          disabled={!canToggleDomainRestriction}
                          onCheckedChange={(checked) =>
                            setState((prev) => ({
                              ...prev,
                              enableRestriction: checked,
                            }))
                          }
                        />
                        <div
                          className="flex flex-col gap-y-0"
                          role="button"
                          tabIndex={canToggleDomainRestriction ? 0 : -1}
                          onClick={toggleDomainRestriction}
                          onKeyDown={handleDomainRestrictionTextKeyDown}
                        >
                          <span
                            id={membersRestrictionLabelId}
                            className="flex items-center gap-x-2"
                          >
                            {t(
                              "settings.general.workspace.domain-restriction.members-restriction.self"
                            )}
                            <FeatureBadge
                              feature={
                                PlanFeature.FEATURE_USER_EMAIL_DOMAIN_RESTRICTION
                              }
                            />
                          </span>
                          <span
                            id={membersRestrictionDescriptionId}
                            className="block text-sm font-normal text-control-placeholder"
                          >
                            {t(
                              "settings.general.workspace.domain-restriction.members-restriction.description"
                            )}
                          </span>
                        </div>
                      </div>
                    }
                  />
                </div>
              </div>
            </FormField>
          </FormFieldGroup>
        </PermissionGuard>
      </FormSection>
    );
  }
);
