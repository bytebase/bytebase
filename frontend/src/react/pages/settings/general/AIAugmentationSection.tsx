import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Alert } from "@/react/components/ui/alert";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { useActuatorV1Store } from "@/store";
import { useSettingV1Store } from "@/store/modules";
import {
  AISetting_Provider,
  AISettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
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

const PROVIDER_DEFAULTS: Record<
  AISetting_Provider,
  { apiKeyDoc: string; endpoint: string; model: string }
> = {
  [AISetting_Provider.OPEN_AI]: {
    apiKeyDoc: "https://platform.openai.com/account/api-keys",
    endpoint: "https://api.openai.com/v1/chat/completions",
    model: "gpt-4o",
  },
  [AISetting_Provider.AZURE_OPENAI]: {
    apiKeyDoc: "https://ai.azure.com/",
    endpoint:
      "https://{resource name}.openai.azure.com/openai/deployments/{deployment id}/chat/completions?api-version=2024-06-0",
    model: "gpt-4o",
  },
  [AISetting_Provider.GEMINI]: {
    apiKeyDoc: "https://ai.google.dev/gemini-api/docs",
    endpoint: "https://generativelanguage.googleapis.com/v1beta",
    model: "gemini-2.5-flash",
  },
  [AISetting_Provider.CLAUDE]: {
    apiKeyDoc: "https://docs.anthropic.com/en/api/getting-started",
    endpoint: "https://api.anthropic.com/v1/messages",
    model: "claude-sonnet-4-20250514",
  },
  [AISetting_Provider.PROVIDER_UNSPECIFIED]: {
    apiKeyDoc: "",
    endpoint: "",
    model: "",
  },
};

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
  const settingV1Store = useSettingV1Store();
  const containerRef = useRef<HTMLDivElement>(null);

  const actuatorStore = useActuatorV1Store();
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const canEdit = hasWorkspacePermissionV2("bb.settings.set") && !isSaaSMode;

  const aiSetting = useVueState(() => {
    const setting = settingV1Store.getSettingByName(Setting_SettingName.AI);
    if (setting?.value?.value?.case === "ai") {
      return setting.value.value.value;
    }
    return undefined;
  });

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
    settingV1Store.getOrFetchSettingByName(Setting_SettingName.AI, true);
  }, [settingV1Store]);

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

    await settingV1Store.upsertSetting({
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
  }, [state, getInitialState, aiSetting, settingV1Store]);

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
              <Alert variant="info">
                {t("settings.general.workspace.ai-assistant.enabled-in-saas")}
              </Alert>
            ) : (
              <div className="mt-4 lg:mt-0 flex flex-col gap-y-4">
                {/* Enable toggle */}
                <div>
                  <div className="flex items-center gap-x-2">
                    <input
                      type="checkbox"
                      checked={state.enabled}
                      disabled={!canEdit}
                      onChange={(e) => toggleEnabled(e.target.checked)}
                    />
                    <span className="text-base font-semibold">
                      {t(
                        "settings.general.workspace.ai-assistant.enable-ai-assistant"
                      )}
                    </span>
                  </div>
                  <div className="mt-1 mb-3 text-sm text-gray-400">
                    {t("settings.general.workspace.ai-assistant.description")}{" "}
                    <a
                      href="https://docs.bytebase.com/ai-assistant?source=console"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-accent hover:underline text-sm ml-1"
                    >
                      {t("common.learn-more")}
                    </a>
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
                      <select
                        className="w-48 border border-control-border rounded-xs px-3 py-1.5 text-sm bg-white disabled:opacity-50"
                        value={state.provider}
                        disabled={!canEdit}
                        onChange={(e) =>
                          onProviderChange(
                            Number(e.target.value) as AISetting_Provider
                          )
                        }
                      >
                        {PROVIDER_OPTIONS.map((provider) => (
                          <option key={provider} value={provider}>
                            {provider === AISetting_Provider.OPEN_AI &&
                              t(
                                "settings.general.workspace.ai-assistant.provider.open_ai"
                              )}
                            {provider === AISetting_Provider.AZURE_OPENAI &&
                              t(
                                "settings.general.workspace.ai-assistant.provider.azure_open_ai"
                              )}
                            {provider === AISetting_Provider.GEMINI &&
                              t(
                                "settings.general.workspace.ai-assistant.provider.gemini"
                              )}
                            {provider === AISetting_Provider.CLAUDE &&
                              t(
                                "settings.general.workspace.ai-assistant.provider.claude"
                              )}
                          </option>
                        ))}
                      </select>
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
                      <div className="mb-3 text-sm text-gray-400">
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
                      <div className="mb-3 text-sm text-gray-400">
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
                      <div className="mb-3 text-sm text-gray-400">
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
