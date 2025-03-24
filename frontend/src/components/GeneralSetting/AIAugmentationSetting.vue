<template>
  <div id="ai" ref="containerRef" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
        <FeatureBadge feature="bb.feature.ai-assistant" />
      </div>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>
    <div class="flex-1 lg:px-4">
      <div class="mt-4 lg:mt-0">
        <div class="flex items-center gap-x-2">
          <Switch
            v-model:value="state.enabled"
            :text="true"
            :disabled="!allowEdit || !hasAIFeature"
            @update:value="toggleAIEnabled"
          />
          <p class="textinfolabel">
          {{ $t("settings.general.workspace.ai-assistant.description") }}
          <LearnMoreLink
            url="https://www.bytebase.com/docs/ai-assistant?source=console"
            class="ml-1 text-sm"
          />
          </p>
        </div>
        <template v-if="state.enabled">
          <label class="flex items-center gap-x-2">
            <span class="font-medium">{{
              $t("settings.general.workspace.ai-assistant.openai-key.self")
            }}</span>
          </label>
          <div class="mb-3 text-sm text-gray-400">
            <i18n-t
              keypath="settings.general.workspace.ai-assistant.openai-key.description"
            >
              <template #viewDoc>
                <a
                  href="https://platform.openai.com/account/api-keys"
                  class="normal-link"
                  target="_blank"
                  >{{
                    $t(
                      "settings.general.workspace.ai-assistant.openai-key.find-my-key"
                    )
                  }}
                </a>
              </template>
            </i18n-t>
          </div>
          <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <BBTextField
              v-model:value="state.openAIKey"
              class="mb-4 w-full"
              :required="true"
              :disabled="!allowEdit || !hasAIFeature"
              :placeholder="
                $t(
                  'settings.general.workspace.ai-assistant.openai-key.placeholder'
                )
              "
            />
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{ $t("settings.general.workspace.only-admin-can-edit") }}
          </span>
          </NTooltip>

          <label class="flex items-center gap-x-2">
            <span class="font-medium">{{
              $t("settings.general.workspace.ai-assistant.openai-endpoint.self")
            }}</span>
          </label>
          <div class="mb-3 text-sm text-gray-400">
            {{
              $t(
                "settings.general.workspace.ai-assistant.openai-endpoint.description"
              )
            }}
          </div>
          <NTooltip placement="top-start" :disabled="allowEdit">
            <template #trigger>
              <BBTextField
                v-model:value="state.openAIEndpoint"
                class="mb-4 w-full"
                :required="true"
                :disabled="!allowEdit || !hasAIFeature"
              />
            </template>
            <span class="text-sm text-gray-400 -translate-y-2">
              {{ $t("settings.general.workspace.only-admin-can-edit") }}
            </span>
          </NTooltip>

          <label class="flex items-center gap-x-2">
            <span class="font-medium">{{
              $t("settings.general.workspace.ai-assistant.openai-model.self")
            }}</span>
          </label>
          <div class="mb-3 text-sm text-gray-400">
            {{
              $t(
                "settings.general.workspace.ai-assistant.openai-model.description"
              )
            }}
          </div>
          <NTooltip placement="top-start" :disabled="allowEdit">
            <template #trigger>
              <BBTextField
                v-model:value="state.openAIModel"
                class="mb-4 w-full"
                :required="true"
                :disabled="!allowEdit || !hasAIFeature"
              />
            </template>
            <span class="text-sm text-gray-400 -translate-y-2">
              {{ $t("settings.general.workspace.only-admin-can-edit") }}
            </span>
          </NTooltip>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { BBTextField } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { Switch } from "@/components/v2";
import { hasFeature } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  AISetting,
  AISetting_Provider,
} from "@/types/proto/v1/setting_service";
import { NTooltip } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, onMounted, reactive, ref, watchEffect } from "vue";
import { FeatureBadge } from "../FeatureGuard";

interface LocalState {
  enabled: boolean;
  openAIKey: string;
  openAIEndpoint: string;
  openAIModel: string;
}

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const containerRef = ref<HTMLDivElement>();

const state = reactive<LocalState>({
  enabled: false,
  openAIKey: "",
  openAIEndpoint: "",
  openAIModel: "",
});

const aiSetting = computed(
  () => settingV1Store.getSettingByName("bb.ai")?.value?.aiSetting
);

const hasAIFeature = computed(() => hasFeature("bb.feature.ai-assistant"));

const getInitialState = (): LocalState => {
  return {
    enabled: aiSetting.value?.enabled ?? false,
    openAIKey: maskKey(aiSetting.value?.apiKey),
    openAIEndpoint: aiSetting.value?.endpoint ?? "",
    openAIModel: aiSetting.value?.model ?? "",
  };
};

watchEffect(() => {
  Object.assign(state, getInitialState());
});

const allowSave = computed((): boolean => {
  const initValue = getInitialState();
  const enabledUpdated = state.enabled !== initValue.enabled;
  const openAIKeyUpdated =
    state.openAIKey !== initValue.openAIKey ||
    (state.openAIKey && !state.openAIKey.includes("***"));
  const openAIEndpointUpdated =
    state.openAIEndpoint !== initValue.openAIEndpoint;
  const openAIModelUpdated = state.openAIModel !== initValue.openAIModel;
  return enabledUpdated || openAIKeyUpdated || openAIEndpointUpdated || openAIModelUpdated;
});

function maskKey(key: string | undefined): string {
  return key ? key.slice(0, 3) + "***" + key.slice(-4) : "";
}

const toggleAIEnabled = (on: boolean) => {
  if (!on) {
    return;
  }
  if (state.openAIEndpoint === "") {
    state.openAIEndpoint = "https://api.openai.com/v1/chat/completions";
  }
  if (state.openAIModel === "") {
    state.openAIModel = "gpt-3.5-turbo";
  }
};

const updateAISetting = async () => {
  let setting = AISetting.fromPartial({});
  if (state.enabled) {
    setting = AISetting.fromPartial({
      ...(aiSetting.value ?? {}),
      enabled: state.enabled,
      apiKey: state.openAIKey,
      endpoint: state.openAIEndpoint,
      model: state.openAIModel,
      // TODO(ed): support change provider.
      provider: AISetting_Provider.OPEN_AI,
    });
  }
  await settingV1Store.upsertSetting({
    name: "bb.ai",
    value: {
      aiSetting: setting,
    },
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
