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
import { ComponentPermissionGuard } from "@/components/ComponentPermissionGuard";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import { PermissionGuard } from "@/components/PermissionGuard";
import { Alert } from "@/components/ui/alert";
import { Checkbox } from "@/components/ui/checkbox";
import { FormField, FormFieldGroup, FormSection } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useServerState } from "@/hooks/useAppState";
import {
  AI_ASSISTANT_PRODUCT_INTRO,
  useProductIntro,
} from "@/lib/productIntro";
import { useAppStore } from "@/stores/app";
import {
  AISetting_Provider,
  AISettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { PROVIDER_DEFAULTS, PROVIDER_MODELS } from "./aiProviderDefaults";
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

const CUSTOM_MODEL = "__custom__";

function getSelectedModel(state: AIState): string {
  return (
    (PROVIDER_MODELS[state.provider] ?? []).find(
      (model) =>
        model.value === state.model && model.endpoint === state.endpoint
    )?.value ?? CUSTOM_MODEL
  );
}

export const AIAugmentationSection = forwardRef<
  SectionHandle,
  AIAugmentationSectionProps
>(function AIAugmentationSection({ title, onDirtyChange }, ref) {
  const { t } = useTranslation();
  const containerRef = useRef<HTMLElement>(null);

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
  const [selectedModel, setSelectedModel] = useState(() =>
    getSelectedModel(getInitialState())
  );

  // Re-sync state when the store value changes (e.g. after initial fetch).
  const prevAiSettingRef = useRef(aiSetting);
  useEffect(() => {
    if (prevAiSettingRef.current !== aiSetting) {
      prevAiSettingRef.current = aiSetting;
      setState(getInitialState());
      setSelectedModel(getSelectedModel(getInitialState()));
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

  useProductIntro({
    id: AI_ASSISTANT_PRODUCT_INTRO,
    title: t("settings.general.workspace.ai-assistant.self"),
    description: t("settings.general.workspace.ai-assistant.description"),
  });

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
    setSelectedModel(getSelectedModel(getInitialState()));
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
    setSelectedModel(defaults.model);
    setState((s) => ({
      ...s,
      provider,
      apiKey: "",
      endpoint: defaults.endpoint,
      model: defaults.model,
    }));
  };

  const onModelChange = (modelName: string | null) => {
    if (!modelName) {
      return;
    }
    setSelectedModel(modelName);
    if (modelName === CUSTOM_MODEL) {
      return;
    }
    const model = (PROVIDER_MODELS[state.provider] ?? []).find(
      (model) => model.value === modelName
    );
    if (!model) {
      return;
    }
    setState((s) => ({
      ...s,
      endpoint: model.endpoint,
      model: model.value,
    }));
  };

  const toggleEnabled = (enabled: boolean) => {
    if (enabled && (!state.endpoint || !state.model)) {
      setSelectedModel(
        state.endpoint || state.model
          ? CUSTOM_MODEL
          : PROVIDER_DEFAULTS[state.provider].model
      );
    }
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

  const isCustomModel = selectedModel === CUSTOM_MODEL;
  const isAzureOpenAI = state.provider === AISetting_Provider.AZURE_OPENAI;
  const isOpenAICompatible =
    state.provider === AISetting_Provider.OPEN_AI || isAzureOpenAI;

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
    <FormSection
      id="ai"
      ref={containerRef}
      title={title}
      data-product-intro-target={AI_ASSISTANT_PRODUCT_INTRO}
    >
      <ComponentPermissionGuard permissions={["bb.settings.get"]}>
        <PermissionGuard permissions={["bb.settings.set"]} display="block">
          <FormFieldGroup>
            {isSaaSMode ? (
              <Alert
                variant="info"
                description={t(
                  "settings.general.workspace.ai-assistant.enabled-in-saas"
                )}
              />
            ) : (
              <>
                {/* Enable toggle */}
                <FormField
                  title={
                    <span className="flex items-center gap-x-2">
                      <Checkbox
                        checked={state.enabled}
                        disabled={!canEdit}
                        onCheckedChange={(checked) => toggleEnabled(checked)}
                      />
                      {t(
                        "settings.general.workspace.ai-assistant.enable-ai-assistant"
                      )}
                    </span>
                  }
                  description={
                    <>
                      {t("settings.general.workspace.ai-assistant.description")}{" "}
                      <LearnMoreLink
                        href="https://docs.bytebase.com/ai-assistant?source=console"
                        className="text-accent text-sm ml-1"
                      />
                    </>
                  }
                />

                {/* Collapsible fields when enabled */}
                {state.enabled && (
                  <>
                    {/* Provider */}
                    <FormField
                      title={t(
                        "settings.general.workspace.ai-assistant.provider.self"
                      )}
                    >
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
                    </FormField>

                    {/* API Key */}
                    <FormField
                      title={t(
                        "settings.general.workspace.ai-assistant.api-key.self"
                      )}
                      description={t(
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
                    >
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
                    </FormField>

                    {/* Model */}
                    <FormField
                      title={t(
                        "settings.general.workspace.ai-assistant.model.self"
                      )}
                      description={t(
                        "settings.general.workspace.ai-assistant.model.description"
                      )}
                    >
                      <Select
                        value={selectedModel}
                        disabled={!canEdit}
                        onValueChange={onModelChange}
                      >
                        <SelectTrigger className="w-48">
                          <SelectValue>
                            {selectedModel === CUSTOM_MODEL
                              ? t(
                                  "settings.general.workspace.ai-assistant.model.custom"
                                )
                              : selectedModel}
                          </SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          {(PROVIDER_MODELS[state.provider] ?? []).map(
                            (model) => (
                              <SelectItem key={model.value} value={model.value}>
                                {model.value}
                              </SelectItem>
                            )
                          )}
                          <SelectItem value={CUSTOM_MODEL}>
                            {t(
                              "settings.general.workspace.ai-assistant.model.custom"
                            )}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </FormField>

                    {(isCustomModel || isAzureOpenAI) && (
                      <FormField
                        title={t(
                          isAzureOpenAI
                            ? "settings.general.workspace.ai-assistant.model.deployment"
                            : "settings.general.workspace.ai-assistant.model.name"
                        )}
                        description={
                          isAzureOpenAI
                            ? t(
                                "settings.general.workspace.ai-assistant.model.deployment-description"
                              )
                            : undefined
                        }
                      >
                        <Input
                          value={state.model}
                          required
                          disabled={!canEdit}
                          onChange={(e) =>
                            setState((s) => ({ ...s, model: e.target.value }))
                          }
                        />
                      </FormField>
                    )}

                    {isCustomModel && (
                      <FormField
                        title={t(
                          "settings.general.workspace.ai-assistant.endpoint.self"
                        )}
                        description={t(
                          isOpenAICompatible
                            ? "settings.general.workspace.ai-assistant.endpoint.openai-description"
                            : "settings.general.workspace.ai-assistant.endpoint.description"
                        )}
                      >
                        <Input
                          value={state.endpoint}
                          required
                          disabled={!canEdit}
                          placeholder={providerDefault.endpoint}
                          onChange={(e) =>
                            setState((s) => ({
                              ...s,
                              endpoint: e.target.value,
                            }))
                          }
                        />
                      </FormField>
                    )}
                  </>
                )}
              </>
            )}
          </FormFieldGroup>
        </PermissionGuard>
      </ComponentPermissionGuard>
    </FormSection>
  );
});
