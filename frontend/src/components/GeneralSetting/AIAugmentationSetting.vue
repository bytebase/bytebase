<template>
  <div id="ai" ref="containerRef" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>
    <div class="flex-1 lg:px-4">
      <div class="mt-4 lg:mt-0 space-y-6">
        <!-- AI Providers Section -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <h2 class="text-lg font-medium">
              {{ $t("settings.general.workspace.ai-assistant.providers") }}
            </h2>
            <NButton
              v-if="allowEdit"
              type="primary"
              size="small"
              @click="addProvider"
              :disabled="!canAddProvider"
            >
              <template #icon>
                <heroicons:plus-20-solid class="w-4 h-4" />
              </template>
              {{ $t("settings.general.workspace.ai-assistant.add-provider") }}
            </NButton>
          </div>
          <div class="mb-3 text-sm text-gray-400">
            {{ $t("settings.general.workspace.ai-assistant.description") }}
            <LearnMoreLink
              url="https://docs.bytebase.com/ai-assistant?source=console"
              class="ml-1 text-sm"
            />
          </div>
        </div>

        <!-- No Providers Message -->
        <div
          v-if="state.providers.length === 0"
          class="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center"
        >
          <heroicons:cpu-chip-20-solid
            class="mx-auto h-12 w-12 text-gray-400"
          />
          <h3 class="mt-2 text-sm font-medium text-gray-900">
            {{ $t("settings.general.workspace.ai-assistant.no-providers") }}
          </h3>
          <p class="mt-1 text-sm text-gray-500">
            {{
              $t(
                "settings.general.workspace.ai-assistant.no-providers-description"
              )
            }}
          </p>
          <div class="mt-4">
            <NButton
              v-if="allowEdit && canAddProvider"
              type="primary"
              @click="addProvider"
            >
              <template #icon>
                <heroicons:plus-20-solid class="w-4 h-4" />
              </template>
              {{
                $t("settings.general.workspace.ai-assistant.add-first-provider")
              }}
            </NButton>
          </div>
        </div>

        <!-- Provider Cards -->
        <div v-else class="space-y-4">
          <div
            v-for="(provider, index) in state.providers"
            :key="index"
            class="border rounded-lg p-4 relative"
          >
            <!-- Delete Button in top-right corner -->
            <div v-if="allowEdit" class="absolute top-2 right-2">
              <NTooltip>
                <template #trigger>
                  <NButton
                    type="error"
                    size="small"
                    quaternary
                    circle
                    @click="removeProvider(index)"
                  >
                    <template #icon>
                      <heroicons:trash-20-solid class="w-4 h-4" />
                    </template>
                  </NButton>
                </template>
                {{
                  $t("settings.general.workspace.ai-assistant.remove-provider")
                }}
              </NTooltip>
            </div>

            <div class="space-y-4">
              <!-- Provider Type -->
              <div>
                <label class="flex items-center gap-x-2 mb-2">
                  <span class="font-medium">{{
                    $t("settings.general.workspace.ai-assistant.provider.self")
                  }}</span>
                </label>
                <NSelect
                  style="width: 12rem"
                  v-model:value="provider.type"
                  :options="getProviderOptionsForIndex(index)"
                  :disabled="!allowEdit"
                  :consistent-menu-width="true"
                  @update:value="(value) => onProviderTypeChange(index, value)"
                />
              </div>

              <!-- API Key -->
              <div>
                <label class="flex items-center gap-x-2">
                  <span class="font-medium">{{
                    $t("settings.general.workspace.ai-assistant.api-key.self")
                  }}</span>
                </label>
                <div class="mb-3 text-sm text-gray-400">
                  <i18n-t
                    keypath="settings.general.workspace.ai-assistant.api-key.description"
                  >
                    <template #viewDoc>
                      <a
                        :href="getProviderDefault(provider.type).apiKeyDoc"
                        class="normal-link"
                        target="_blank"
                        >{{
                          $t(
                            "settings.general.workspace.ai-assistant.api-key.find-my-key"
                          )
                        }}
                      </a>
                    </template>
                  </i18n-t>
                </div>
                <NTooltip placement="top-start" :disabled="allowEdit">
                  <template #trigger>
                    <BBTextField
                      v-model:value="provider.apiKey"
                      :disabled="!allowEdit"
                      :placeholder="
                        $t(
                          'settings.general.workspace.ai-assistant.api-key.placeholder'
                        )
                      "
                    />
                  </template>
                  <span class="text-sm text-gray-400 -translate-y-2">
                    {{ $t("settings.general.workspace.only-admin-can-edit") }}
                  </span>
                </NTooltip>
              </div>

              <!-- Endpoint -->
              <div>
                <label class="flex items-center gap-x-2">
                  <span class="font-medium">{{
                    $t("settings.general.workspace.ai-assistant.endpoint.self")
                  }}</span>
                </label>
                <div class="mb-3 text-sm text-gray-400">
                  {{
                    $t(
                      "settings.general.workspace.ai-assistant.endpoint.description"
                    )
                  }}
                </div>
                <NTooltip placement="top-start" :disabled="allowEdit">
                  <template #trigger>
                    <BBTextField
                      v-model:value="provider.endpoint"
                      :required="true"
                      :disabled="!allowEdit"
                      :placeholder="getProviderDefault(provider.type).endpoint"
                    />
                  </template>
                  <span class="text-sm text-gray-400 -translate-y-2">
                    {{ $t("settings.general.workspace.only-admin-can-edit") }}
                  </span>
                </NTooltip>
              </div>

              <!-- Model -->
              <div>
                <label class="flex items-center gap-x-2">
                  <span class="font-medium">{{
                    $t("settings.general.workspace.ai-assistant.model.self")
                  }}</span>
                </label>
                <div class="mb-3 text-sm text-gray-400">
                  {{
                    $t(
                      "settings.general.workspace.ai-assistant.model.description"
                    )
                  }}
                </div>
                <NTooltip placement="top-start" :disabled="allowEdit">
                  <template #trigger>
                    <BBTextField
                      v-model:value="provider.model"
                      :required="true"
                      :disabled="!allowEdit"
                      :placeholder="getProviderDefault(provider.type).model"
                    />
                  </template>
                  <span class="text-sm text-gray-400 -translate-y-2">
                    {{ $t("settings.general.workspace.only-admin-can-edit") }}
                  </span>
                </NTooltip>
              </div>

              <!-- Version (for Claude) -->
              <div v-if="provider.type === AIProvider_Type.CLAUDE">
                <label class="flex items-center gap-x-2">
                  <span class="font-medium">{{
                    $t("settings.general.workspace.ai-assistant.version.self")
                  }}</span>
                </label>
                <div class="mb-3 text-sm text-gray-400">
                  {{
                    $t(
                      "settings.general.workspace.ai-assistant.version.description"
                    )
                  }}
                </div>
                <NTooltip placement="top-start" :disabled="allowEdit">
                  <template #trigger>
                    <BBTextField
                      v-model:value="provider.version"
                      :disabled="!allowEdit"
                      placeholder="2023-06-01"
                    />
                  </template>
                  <span class="text-sm text-gray-400 -translate-y-2">
                    {{ $t("settings.general.workspace.only-admin-can-edit") }}
                  </span>
                </NTooltip>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton, NSelect, NTooltip } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, onMounted, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { BBTextField } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  AISettingSchema,
  AIProviderSchema,
  AIProvider_Type,
  Setting_SettingName,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import type { AIProvider } from "@/types/proto-es/v1/setting_service_pb";

interface LocalProvider {
  type: AIProvider_Type;
  endpoint: string;
  apiKey: string;
  model: string;
  version: string;
}

interface LocalState {
  providers: LocalProvider[];
}

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const containerRef = ref<HTMLDivElement>();
const { t } = useI18n();

const state = reactive<LocalState>({
  providers: [],
});

const aiSetting = computed(() => {
  const setting = settingV1Store.getSettingByName(Setting_SettingName.AI);
  if (setting?.value?.value?.case === "aiSetting") {
    return setting.value.value.value;
  }
  return undefined;
});

const getInitialState = (): LocalState => {
  const providers: LocalProvider[] = [];
  if (aiSetting.value?.providers) {
    for (const provider of aiSetting.value.providers) {
      providers.push({
        type: provider.type ?? AIProvider_Type.OPEN_AI,
        endpoint: provider.endpoint ?? "",
        apiKey: "", // Never expose API key
        model: provider.model ?? "",
        version: provider.version ?? "",
      });
    }
  }
  return { providers };
};

const allProviderTypes = [
  AIProvider_Type.OPEN_AI,
  AIProvider_Type.AZURE_OPENAI,
  AIProvider_Type.GEMINI,
  AIProvider_Type.CLAUDE,
];

const getProviderLabel = (provider: AIProvider_Type): string => {
  switch (provider) {
    case AIProvider_Type.OPEN_AI:
      return t("settings.general.workspace.ai-assistant.provider.open_ai");
    case AIProvider_Type.AZURE_OPENAI:
      return t(
        "settings.general.workspace.ai-assistant.provider.azure_open_ai"
      );
    case AIProvider_Type.GEMINI:
      return t("settings.general.workspace.ai-assistant.provider.gemini");
    case AIProvider_Type.CLAUDE:
      return t("settings.general.workspace.ai-assistant.provider.claude");
    default:
      return "";
  }
};

// Options for the provider dropdown for a specific provider at index
const getProviderOptionsForIndex = (index: number) => {
  const currentProvider = state.providers[index];
  const usedTypes = new Set(
    state.providers.filter((_, i) => i !== index).map((p) => p.type)
  );

  return allProviderTypes
    .filter((type) => type === currentProvider?.type || !usedTypes.has(type))
    .map((provider) => ({
      label: getProviderLabel(provider),
      value: provider,
    }));
};

// Available provider types that haven't been used yet
const availableProviderTypes = computed(() => {
  const usedTypes = new Set(state.providers.map((p) => p.type));
  return allProviderTypes.filter((type) => !usedTypes.has(type));
});

// Can we add more providers?
const canAddProvider = computed(() => {
  return availableProviderTypes.value.length > 0 && state.providers.length < 4;
});

watchEffect(() => {
  Object.assign(state, getInitialState());
});

const allowSave = computed((): boolean => {
  const initValue = getInitialState();

  // Check if providers count changed
  if (state.providers.length !== initValue.providers.length) {
    return true;
  }

  // Check if any provider was modified
  for (let i = 0; i < state.providers.length; i++) {
    const current = state.providers[i];
    const initial = initValue.providers[i];

    if (!initial) return true;

    if (current.type !== initial.type) return true;
    if (current.endpoint !== initial.endpoint) return true;
    if (current.model !== initial.model) return true;
    if (current.version !== initial.version) return true;
    if (current.apiKey !== "") return true; // API key was entered
  }

  return false;
});

const getProviderDefault = (type: AIProvider_Type) => {
  switch (type) {
    case AIProvider_Type.OPEN_AI:
      return {
        apiKey: "",
        apiKeyDoc: "https://platform.openai.com/account/api-keys",
        endpoint: "https://api.openai.com/v1/chat/completions",
        model: "gpt-3.5-turbo",
      };
    case AIProvider_Type.AZURE_OPENAI:
      return {
        apiKey: "",
        apiKeyDoc: "https://ai.azure.com/",
        endpoint:
          "https://{resource name}.openai.azure.com/openai/deployments/{deployment id}/chat/completions?api-version=2024-06-01",
        model: "gpt-4o",
      };
    case AIProvider_Type.GEMINI:
      return {
        apiKey: "",
        apiKeyDoc: "https://ai.google.dev/gemini-api/docs",
        endpoint: "https://generativelanguage.googleapis.com/v1beta",
        model: "gemini-2.0-flash-exp",
      };
    case AIProvider_Type.CLAUDE:
      return {
        apiKey: "",
        apiKeyDoc: "https://docs.anthropic.com/en/api/getting-started",
        endpoint: "https://api.anthropic.com/v1/messages",
        model: "claude-3-5-sonnet-20241022",
      };
    default:
      return {
        apiKey: "",
        apiKeyDoc: "",
        endpoint: "",
        model: "",
      };
  }
};

const addProvider = () => {
  // Use the first available provider type
  const availableType = availableProviderTypes.value[0];
  if (!availableType) return;

  const defaults = getProviderDefault(availableType);
  state.providers.push({
    type: availableType,
    endpoint: defaults.endpoint,
    apiKey: "",
    model: defaults.model,
    version: availableType === AIProvider_Type.CLAUDE ? "2023-06-01" : "",
  });
};

const removeProvider = (index: number) => {
  state.providers.splice(index, 1);
};

const onProviderTypeChange = (index: number, type: AIProvider_Type) => {
  const defaults = getProviderDefault(type);
  const provider = state.providers[index];
  provider.endpoint = defaults.endpoint;
  provider.model = defaults.model;
  provider.version = type === AIProvider_Type.CLAUDE ? "2023-06-01" : "";
};

const updateAISetting = async () => {
  // Build providers list for the API
  const providers: AIProvider[] = [];
  for (const provider of state.providers) {
    // Get the existing provider to preserve API key if not changed
    let apiKey = provider.apiKey;
    if (!apiKey && aiSetting.value?.providers) {
      const existing = aiSetting.value.providers.find(
        (p) => p.type === provider.type
      );
      if (existing) {
        apiKey = existing.apiKey;
      }
    }

    providers.push(
      create(AIProviderSchema, {
        type: provider.type,
        endpoint: provider.endpoint,
        apiKey: apiKey,
        model: provider.model,
        version: provider.version,
      })
    );
  }

  await settingV1Store.upsertSetting({
    name: Setting_SettingName.AI,
    value: create(SettingValueSchema, {
      value: {
        case: "aiSetting",
        value: create(AISettingSchema, {
          providers: providers,
        }),
      },
    }),
  });

  Object.assign(state, getInitialState());
};

onMounted(() => {
  if (location.hash === "#ai-assistant") {
    const container = containerRef.value;
    if (!container) return;
    scrollIntoView(container, {
      scrollMode: "if-needed",
    });
  }
});

defineExpose({
  isDirty: allowSave,
  title: props.title,
  update: updateAISetting,
  revert: () => {
    Object.assign(state, getInitialState());
  },
});
</script>
