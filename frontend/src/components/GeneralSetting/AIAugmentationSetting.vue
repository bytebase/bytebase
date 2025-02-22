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
        <p class="mb-2 textinfolabel">
          {{ $t("settings.general.workspace.ai-assistant.description") }}
          <LearnMoreLink
            url="https://www.bytebase.com/docs/ai-assistant?source=console"
            class="ml-1 text-sm"
          />
        </p>

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
                }}</a
              >
            </template>
          </i18n-t>
        </div>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <NInput
              v-model:value="state.openAIKey"
              class="mb-4 w-full"
              :disabled="!allowEdit"
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
            <NInput
              v-model:value="state.openAIEndpoint"
              class="mb-4 w-full"
              :disabled="!allowEdit"
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
            <NInput
              v-model:value="state.openAIModel"
              class="mb-4 w-full"
              :disabled="!allowEdit"
            />
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{ $t("settings.general.workspace.only-admin-can-edit") }}
          </span>
        </NTooltip>
      </div>
    </div>

    <FeatureModal
      feature="bb.feature.ai-assistant"
      :open="state.showFeatureModal"
      @cancel="state.showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { NInput, NTooltip } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, onMounted, reactive, ref, watchEffect } from "vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { hasFeature } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";

interface LocalState {
  openAIKey: string;
  openAIEndpoint: string;
  openAIModel: string;
  showFeatureModal: boolean;
}

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const containerRef = ref<HTMLDivElement>();

const state = reactive<LocalState>({
  openAIKey: "",
  openAIEndpoint: "",
  openAIModel: "",
  showFeatureModal: false,
});

const openAIKeySetting = settingV1Store.getSettingByName(
  "bb.plugin.openai.key"
);
const openAIEndpointSetting = settingV1Store.getSettingByName(
  "bb.plugin.openai.endpoint"
);
const openAIModelSetting = settingV1Store.getSettingByName(
  "bb.plugin.openai.model"
);

watchEffect(() => {
  state.openAIKey = maskKey(openAIKeySetting?.value?.stringValue);
  state.openAIEndpoint = openAIEndpointSetting?.value?.stringValue ?? "";
  state.openAIModel = openAIModelSetting?.value?.stringValue ?? "";
});

const allowSave = computed((): boolean => {
  const openAIKeyUpdated =
    state.openAIKey !== maskKey(openAIKeySetting?.value?.stringValue) ||
    (state.openAIKey && !state.openAIKey.includes("***"));
  const openAIEndpointUpdated =
    state.openAIEndpoint !== openAIEndpointSetting?.value?.stringValue;
  const openAIModelUpdated =
    state.openAIModel !== openAIModelSetting?.value?.stringValue;
  return openAIKeyUpdated || openAIEndpointUpdated || openAIModelUpdated;
});

function maskKey(key: string | undefined): string {
  return key ? key.slice(0, 3) + "***" + key.slice(-4) : "";
}

const updateOpenAIKeyEndpoint = async () => {
  // Always allow to unset the key.
  const isUnset = state.openAIKey === "" && state.openAIEndpoint === "";
  if (!isUnset && !hasFeature("bb.feature.ai-assistant")) {
    state.showFeatureModal = true;
    return;
  }

  if (
    state.openAIKey !== maskKey(openAIKeySetting?.value?.stringValue) ||
    !state.openAIKey.includes("***")
  ) {
    await settingV1Store.upsertSetting({
      name: "bb.plugin.openai.key",
      value: {
        stringValue: state.openAIKey,
      },
    });
  }
  if (state.openAIEndpoint !== openAIEndpointSetting?.value?.stringValue) {
    await settingV1Store.upsertSetting({
      name: "bb.plugin.openai.endpoint",
      value: {
        stringValue: state.openAIEndpoint,
      },
    });
  }
  if (state.openAIEndpoint !== openAIModelSetting?.value?.stringValue) {
    await settingV1Store.upsertSetting({
      name: "bb.plugin.openai.model",
      value: {
        stringValue: state.openAIModel,
      },
    });
  }
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
  update: updateOpenAIKeyEndpoint,
});
</script>
