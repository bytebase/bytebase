import { create } from "@bufbuild/protobuf";
import { DurationSchema, FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { X } from "lucide-react";
import {
  forwardRef,
  type KeyboardEvent,
  useCallback,
  useEffect,
  useImperativeHandle,
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
import { usePlanFeature } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { SectionHandle } from "./useSettingSection";

const DEFAULT_EXPIRATION_DAYS = 90;

interface SecurityState {
  enableWatermark: boolean;
  inputValue: number;
  neverExpire: boolean;
  domains: string[];
  enableRestriction: boolean;
}

interface SecuritySectionProps {
  title: string;
  onDirtyChange: () => void;
}

export const SecuritySection = forwardRef<SectionHandle, SecuritySectionProps>(
  function SecuritySection({ title, onDirtyChange }, ref) {
    const { t } = useTranslation();
    const settingV1Store = useSettingV1Store();

    const hasWatermarkFeature = usePlanFeature(PlanFeature.FEATURE_WATERMARK);
    const hasDomainRestrictionFeature = usePlanFeature(
      PlanFeature.FEATURE_USER_EMAIL_DOMAIN_RESTRICTION
    );
    const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);

    const getInitialState = useCallback((): SecurityState => {
      const profile = settingV1Store.workspaceProfile;

      // Watermark
      const enableWatermark = profile.watermark;

      // Maximum role expiration
      let inputValue = DEFAULT_EXPIRATION_DAYS;
      let neverExpire = true;
      const seconds = profile.maximumRoleExpiration?.seconds
        ? Number(profile.maximumRoleExpiration.seconds)
        : undefined;
      if (seconds && seconds > 0) {
        inputValue =
          Math.floor(seconds / (60 * 60 * 24)) || DEFAULT_EXPIRATION_DAYS;
        neverExpire = false;
      }

      // Domain restriction
      const domains = Array.isArray(profile.domains)
        ? [...profile.domains]
        : [];
      const enableRestriction = profile.enforceIdentityDomain || false;

      return {
        enableWatermark,
        inputValue,
        neverExpire,
        domains,
        enableRestriction,
      };
    }, [settingV1Store]);

    const [state, setState] = useState<SecurityState>(getInitialState);
    const [domainInput, setDomainInput] = useState("");

    const validDomains = state.domains.filter((d) => !!d);

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
        await settingV1Store.updateWorkspaceProfile({
          payload: { watermark: state.enableWatermark },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.watermark"],
          }),
        });
      }

      // Maximum role expiration
      if (
        state.neverExpire !== init.neverExpire ||
        state.inputValue !== init.inputValue
      ) {
        let seconds = -1;
        if (!state.neverExpire) {
          seconds = state.inputValue * 24 * 60 * 60;
        }
        await settingV1Store.updateWorkspaceProfile({
          payload: {
            maximumRoleExpiration: create(DurationSchema, {
              seconds: BigInt(seconds),
              nanos: 0,
            }),
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.maximum_role_expiration"],
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
        await settingV1Store.updateWorkspaceProfile({
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
    }, [state, domainInput, settingV1Store, getInitialState]);

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

    return (
      <div id="security" className="py-6 lg:flex gap-y-4 lg:gap-y-0">
        <div className="text-left lg:w-1/4">
          <h1 className="text-2xl font-bold">{title}</h1>
        </div>
        <PermissionGuard
          permissions={["bb.settings.setWorkspaceProfile"]}
          display="block"
        >
          <div className="flex-1 lg:px-4 flex flex-col gap-y-6">
            {/* Watermark */}
            <div>
              <div className="flex items-center gap-x-2">
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
                <span className="text-base font-semibold">
                  {t("settings.general.workspace.watermark.enable")}
                </span>
                <FeatureBadge feature={PlanFeature.FEATURE_WATERMARK} />
              </div>
              <div className="mt-1 text-sm text-gray-400">
                {t("settings.general.workspace.watermark.description")}
              </div>
            </div>

            {/* Maximum Role Expiration */}
            <div>
              <p className="text-base font-semibold flex flex-row justify-start items-center">
                <span className="mr-2">
                  {t("settings.general.workspace.maximum-role-expiration.self")}
                </span>
              </p>
              <p className="text-sm text-gray-400 mt-1">
                {t(
                  "settings.general.workspace.maximum-role-expiration.description"
                )}
              </p>
              <div className="mt-3 w-full flex flex-row">
                <div className="flex items-center gap-4">
                  <div className="relative w-60">
                    <Input
                      type="number"
                      className="pr-14"
                      value={state.inputValue}
                      min={1}
                      step={1}
                      disabled={!canEdit || state.neverExpire}
                      onChange={(e) => {
                        const val = Math.max(
                          1,
                          Math.floor(Number(e.target.value))
                        );
                        if (!Number.isNaN(val)) {
                          setState((prev) => ({ ...prev, inputValue: val }));
                        }
                      }}
                    />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-500 pointer-events-none">
                      {t(
                        "settings.general.workspace.maximum-role-expiration.days"
                      )}
                    </span>
                  </div>
                  <label className="flex items-center gap-x-2">
                    <Checkbox
                      checked={state.neverExpire}
                      disabled={!canEdit}
                      onCheckedChange={(checked) =>
                        setState((prev) => ({
                          ...prev,
                          neverExpire: checked,
                        }))
                      }
                    />
                    <span>
                      {t(
                        "settings.general.workspace.maximum-role-expiration.never-expires"
                      )}
                    </span>
                  </label>
                </div>
              </div>
            </div>

            {/* Domain Restriction */}
            <div>
              <h3
                id="domain-restriction"
                className="text-base font-semibold flex flex-row justify-start items-center"
              >
                <span className="mr-2">
                  {t("settings.general.workspace.domain-restriction.self")}
                </span>
              </h3>
              <p className="text-sm text-gray-400 mt-1">
                {t("settings.general.workspace.domain-restriction.description")}
              </p>
              <div className="w-full flex flex-col gap-2 mt-2">
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
                  <label className="flex items-start gap-x-2">
                    <Checkbox
                      checked={state.enableRestriction}
                      className="mt-1"
                      disabled={
                        !canEdit ||
                        validDomains.length === 0 ||
                        !hasDomainRestrictionFeature
                      }
                      onCheckedChange={(checked) =>
                        setState((prev) => ({
                          ...prev,
                          enableRestriction: checked,
                        }))
                      }
                    />
                    <div>
                      <div className="text-base font-semibold flex items-center gap-x-2">
                        {t(
                          "settings.general.workspace.domain-restriction.members-restriction.self"
                        )}
                        <FeatureBadge
                          feature={
                            PlanFeature.FEATURE_USER_EMAIL_DOMAIN_RESTRICTION
                          }
                        />
                      </div>
                      <p className="text-sm text-gray-400 leading-tight">
                        {t(
                          "settings.general.workspace.domain-restriction.members-restriction.description"
                        )}
                      </p>
                    </div>
                  </label>
                </div>
              </div>
            </div>
          </div>
        </PermissionGuard>
      </div>
    );
  }
);
