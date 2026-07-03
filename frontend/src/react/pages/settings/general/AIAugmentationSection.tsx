import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
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
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
import { LearnMoreLink } from "@/react/components/LearnMoreLink";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Alert } from "@/react/components/ui/alert";
import { Checkbox } from "@/react/components/ui/checkbox";
import { Input } from "@/react/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { useServerState } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import {
  AISetting_Provider,
  AISettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { PROVIDER_DEFAULTS } from "./aiProviderDefaults";
import type { SectionHandle } from "./useSettingSection";

interface AIAugmentationSectionProps {
  title: string;
  onDirtyChange: () => void;
}

interface AIState {
  enabled: boolean;
  apiKey: string;
  endpoint: string;
  model: string;
  provider: AISetting_Provider;
}

const PROVIDER_OPTIONS = [
  AISetting_Provider.OPEN_AI,
  AISetting_Provider.AZURE_OPENAI,
  AISetting_Provider.GEMINI,
  AISetting_Provider.CLAUDE,
] as const;

export const AIAugmentationSection = forwardRef<
  SectionHandle,
  AIAugmentationSectionProps
>(function AIAugmentationSection({ title, onDirtyChange }, ref) {
  const { t } = useTranslation();
  const containerRef = useRef<HTMLDivElement>(null);

  const { isSaaSMode } = useServerState();
  const canEdit = hasWorkspacePermissionV2("bb.settings.set") && !isSaaSMode;

  const settingsByName = useAppStore((s) => s.settingsByName);
  const aiSetting = useMemo(() => {
    const setting = useAppStore
      .getState()
      .getSettingByName(Setting_SettingName.AI);
    if (setting?.value?.value?.case === "ai") {
      return setting.value.value.value;
    }
    return undefined;
  }, [settingsByName]);

  const getInitialState = useCallback((): AIState => {
    return {
      enabled: aiSetting?.enabled ?? false,
      apiKey: "",
      endpoint: aiSetting?.endpoint ?? "",
      model: aiSetting?.model ?? "",
      provider: aiSetting?.provider ?? AISetting_Provider.OPEN_AI,
    };
  }, [aiSetting]);

  const [state, setState] = useState<AIState>(getInitialState);

  // Re-sync state when the store value changes (e.g. after initial fetch).
  const prevAiSettingRef = useRef(aiSetting);
  useEffect(() => {
    if (prevAiSettingRef.current !== aiSetting) {
      prevAiSettingRef.current = aiSetting;
      setState(getInitialState());
    }
  }, [aiSetting, getInitialState]);

  // Fetch setting on mount.
  useEffect(() => {
    useAppStore
      .getState()
      .getOrFetchSettingByName(Setting_SettingName.AI, true);
  }, []);

  // Scroll into view on mount.
  useEffect(() => {
    if (location.hash === "#ai-assistant" && containerRef.current) {
      containerRef.current.scrollIntoView({ block: "nearest" });
    }
  }, []);

  const providerDefault =
    PROVIDER_DEFAULTS[state.provider] ??
    PROVIDER_DEFAULTS[AISetting_Provider.OPEN_AI];

  const isDirty = useCallback(() => {
    const init = getInitialState();
    return (
      state.enabled !== init.enabled ||
      state.provider !== init.provider ||
      !!state.apiKey ||
      state.endpoint !== init.endpoint ||
      state.model !== init.model
    );
  }, [state, getInitialState]);

  const revert = useCallback(() => {
    setState(getInitialState());
  }, [getInitialState]);

  const update = useCallback(async () => {
    const init = getInitialState();
    const paths: string[] = [];
    if (state.enabled !== init.enabled) paths.push("value.ai.enabled");
    if (state.provider !== init.provider) paths.push("value.ai.provider");
    if (state.endpoint !== init.endpoint) paths.push("value.ai.endpoint");
    if (state.model !== init.model) paths.push("value.ai.model");
    if (state.apiKey || state.provider !== init.provider)
      paths.push("value.ai.api_key");

    await useAppStore.getState().upsertSetting({
      name: Setting_SettingName.AI,
      value: create(SettingValueSchema, {
        value: {
          case: "ai",
          value: create(AISettingSchema, {
            enabled: state.enabled,
            apiKey: state.apiKey,
            endpoint: state.endpoint,
            model: state.model,
            provider: state.provider,
            version: aiSetting?.version ?? "",
          }),
        },
      }),
      updateMask: create(FieldMaskSchema, { paths }),
    });
  }, [state, getInitialState, aiSetting]);

  useImperativeHandle(ref, () => ({ isDirty, revert, update }));

  useEffect(() => {
    onDirtyChange();
  }, [state, onDirtyChange]);

  const onProviderChange = (provider: AISetting_Provider) => {
    const defaults = PROVIDER_DEFAULTS[provider];
    setState((s) => ({
      ...s,
      provider,
      apiKey: "",
      endpoint: defaults.endpoint,
      model: defaults.model,
    }));
  };

  const toggleEnabled = (enabled: boolean) => {
    setState((s) => {
      if (enabled) {
        const defaults = PROVIDER_DEFAULTS[s.provider];
        return {
          ...s,
          enabled,
          endpoint: s.endpoint || defaults.endpoint,
          model: s.model || defaults.model,
        };
      }
      return { ...s, enabled };
    });
  };

  const providerLabel = (provider: AISetting_Provider) => {
    switch (provider) {
      case AISetting_Provider.OPEN_AI:
        return t("settings.general.workspace.ai-assistant.provider.open_ai");
      case AISetting_Provider.AZURE_OPENAI:
        return t(
          "settings.general.workspace.ai-assistant.provider.azure_open_ai"
        );
      case AISetting_Provider.GEMINI:
        return t("settings.general.workspace.ai-assistant.provider.gemini");
      case AISetting_Provider.CLAUDE:
        return t("settings.general.workspace.ai-assistant.provider.claude");
      default:
        return "";
    }
  };

  return (
    <div id="ai" ref={containerRef} className="py-6 lg:flex">
      <div className="text-left lg:w-1/4">
        <div className="flex items-center gap-x-2">
          <h1 className="text-2xl font-bold">{title}</h1>
        </div>
      </div>
      <ComponentPermissionGuard
        permissions={["bb.settings.get"]}
        className="flex-1 lg:px-4"
      >
        <PermissionGuard permissions={["bb.settings.set"]} display="block">
          <div className="flex-1 lg:px-4">
            {isSaaSMode ? (
              <Alert
                variant="info"
                description={t(
                  "settings.general.workspace.ai-assistant.enabled-in-saas"
                )}
              />
            ) : (
              <div className="mt-4 lg:mt-0 flex flex-col gap-y-4">
                {/* Enable toggle */}
                <div>
                  <div className="flex items-center gap-x-2">
                    <Checkbox
                      checked={state.enabled}
                      disabled={!canEdit}
                      onCheckedChange={(checked) => toggleEnabled(checked)}
                    />
                    <span className="text-base font-semibold">
                      {t(
                        "settings.general.workspace.ai-assistant.enable-ai-assistant"
                      )}
                    </span>
                  </div>
                  <div className="mt-1 mb-3 text-sm text-control-placeholder">
                    {t("settings.general.workspace.ai-assistant.description")}{" "}
                    <LearnMoreLink
                      href="https://docs.bytebase.com/ai-assistant?source=console"
                      className="text-accent text-sm ml-1"
                    />
                  </div>
                </div>

                {/* Collapsible fields when enabled */}
                {state.enabled && (
                  <>
                    {/* Provider */}
                    <div>
                      <label className="flex items-center gap-x-2 mb-2">
                        <span className="text-base font-semibold">
                          {t(
                            "settings.general.workspace.ai-assistant.provider.self"
                          )}
                        </span>
                      </label>
                      <Select
                        value={String(state.provider)}
                        disabled={!canEdit}
                        onValueChange={(value) =>
                          onProviderChange(Number(value) as AISetting_Provider)
                        }
                      >
                        <SelectTrigger className="w-48">
                          <SelectValue>
                            {providerLabel(state.provider)}
                          </SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          {PROVIDER_OPTIONS.map((provider) => (
                            <SelectItem key={provider} value={String(provider)}>
                              {providerLabel(provider)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    {/* API Key */}
                    <div>
                      <label className="flex items-center gap-x-2">
                        <span className="text-base font-semibold">
                          {t(
                            "settings.general.workspace.ai-assistant.api-key.self"
                          )}
                        </span>
                      </label>
                      <div className="mb-3 text-sm text-control-placeholder">
                        {t(
                          "settings.general.workspace.ai-assistant.api-key.description",
                          {
                            viewDoc: "__LINK__",
                            interpolation: { escapeValue: false },
                          }
                        )
                          .split("__LINK__")
                          .map((part, i) => (
                            <span key={i}>
                              {part}
                              {i === 0 && (
                                <a
                                  href={providerDefault.apiKeyDoc}
                                  className="normal-link"
                                  target="_blank"
                                  rel="noopener noreferrer"
                                >
                                  {t(
                                    "settings.general.workspace.ai-assistant.api-key.find-my-key"
                                  )}
                                </a>
                              )}
                            </span>
                          ))}
                      </div>
                      <Input
                        value={state.apiKey}
                        disabled={!canEdit}
                        placeholder={t(
                          "settings.general.workspace.ai-assistant.api-key.placeholder"
                        )}
                        onChange={(e) =>
                          setState((s) => ({ ...s, apiKey: e.target.value }))
                        }
                      />
                    </div>

                    {/* Endpoint */}
                    <div>
                      <label className="flex items-center gap-x-2">
                        <span className="text-base font-semibold">
                          {t(
                            "settings.general.workspace.ai-assistant.endpoint.self"
                          )}
                        </span>
                      </label>
                      <div className="mb-3 text-sm text-control-placeholder">
                        {t(
                          "settings.general.workspace.ai-assistant.endpoint.description"
                        )}
                      </div>
                      <Input
                        value={state.endpoint}
                        required
                        disabled={!canEdit}
                        placeholder={providerDefault.endpoint}
                        onChange={(e) =>
                          setState((s) => ({ ...s, endpoint: e.target.value }))
                        }
                      />
                    </div>

                    {/* Model */}
                    <div>
                      <label className="flex items-center gap-x-2">
                        <span className="text-base font-semibold">
                          {t(
                            "settings.general.workspace.ai-assistant.model.self"
                          )}
                        </span>
                      </label>
                      <div className="mb-3 text-sm text-control-placeholder">
                        {t(
                          "settings.general.workspace.ai-assistant.model.description"
                        )}
                      </div>
                      <Input
                        value={state.model}
                        required
                        disabled={!canEdit}
                        onChange={(e) =>
                          setState((s) => ({ ...s, model: e.target.value }))
                        }
                      />
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        </PermissionGuard>
      </ComponentPermissionGuard>
    </div>
  );
});
