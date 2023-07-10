<template>
  <div ref="containerRef" class="px-4 py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <h1 class="text-2xl font-bold">
        {{ $t("settings.general.workspace.plugin.openai.ai-augmentation") }}
      </h1>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-owner-can-edit") }}
      </span>
    </div>
    <div class="flex-1 lg:px-5">
      <div class="mb-7 mt-5 lg:mt-0">
        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <span class="font-medium">{{
            $t("settings.general.workspace.plugin.openai.openai-key.self")
          }}</span>

          <FeatureBadge feature="bb.feature.plugin.openai" />

          <span
            v-if="!allowEdit"
            class="text-sm text-gray-400 -translate-y-2 tooltip"
          >
            {{ $t("settings.general.workspace.only-owner-can-edit") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          <i18n-t
            keypath="settings.general.workspace.plugin.openai.openai-key.description"
          >
            <template #viewDoc>
              <a
                href="https://platform.openai.com/account/api-keys"
                class="normal-link"
                target="_blank"
                >{{
                  $t(
                    "settings.general.workspace.plugin.openai.openai-key.find-my-key"
                  )
                }}</a
              >
            </template>
          </i18n-t>
        </div>
        <BBTextField
          class="mb-5 w-full"
          :disabled="!allowEdit"
          :value="state.openAIKey"
          :placeholder="
            $t(
              'settings.general.workspace.plugin.openai.openai-key.placeholder'
            )
          "
          @input="handleOpenAIKeyChange"
        />
        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <span class="font-medium">{{
            $t("settings.general.workspace.plugin.openai.openai-endpoint.self")
          }}</span>

          <FeatureBadge feature="bb.feature.plugin.openai" />

          <span
            v-if="!allowEdit"
            class="text-sm text-gray-400 -translate-y-2 tooltip"
          >
            {{ $t("settings.general.workspace.only-owner-can-edit") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{
            $t(
              "settings.general.workspace.plugin.openai.openai-endpoint.description"
            )
          }}
        </div>
        <BBTextField
          class="mb-5 w-full"
          :disabled="!allowEdit"
          :value="state.openAIEndpoint"
          @input="handleOpenAIEndpointChange"
        />
        <div class="flex">
          <button
            type="button"
            class="btn-primary ml-auto"
            :disabled="!allowSave"
            @click.prevent="updateOpenAIKeyEndpoint"
          >
            {{ $t("common.update") }}
          </button>
        </div>
      </div>
    </div>

    <FeatureModal
      feature="bb.feature.plugin.openai"
      :open="state.showFeatureModal"
      @cancel="state.showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { hasFeature, pushNotification, useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV1 } from "@/utils";
import scrollIntoView from "scroll-into-view-if-needed";
import { useSettingV1Store } from "@/store/modules/v1/setting";

interface LocalState {
  openAIKey: string;
  openAIEndpoint: string;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const currentUserV1 = useCurrentUserV1();
const containerRef = ref<HTMLDivElement>();

const state = reactive<LocalState>({
  openAIKey: "",
  openAIEndpoint: "",
  showFeatureModal: false,
});

const openAIKeySetting = settingV1Store.getSettingByName(
  "bb.plugin.openai.key"
);
const openAIEndpointSetting = settingV1Store.getSettingByName(
  "bb.plugin.openai.endpoint"
);

watchEffect(() => {
  state.openAIKey = maskKey(openAIKeySetting?.value?.stringValue);
  state.openAIEndpoint = openAIEndpointSetting?.value?.stringValue ?? "";
});

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-general",
    currentUserV1.value.userRole
  );
});

const allowSave = computed((): boolean => {
  const openAIKeyUpdated =
    state.openAIKey !== maskKey(openAIKeySetting?.value?.stringValue) ||
    (state.openAIKey && !state.openAIKey.includes("***"));
  return (
    allowEdit.value &&
    (openAIKeyUpdated ||
      state.openAIEndpoint !== openAIEndpointSetting?.value?.stringValue)
  );
});

function maskKey(key: string | undefined): string {
  return key ? key.slice(0, 3) + "***" + key.slice(-4) : "";
}

const handleOpenAIKeyChange = (event: InputEvent) => {
  state.openAIKey = (event.target as HTMLInputElement).value;
};

const handleOpenAIEndpointChange = (event: InputEvent) => {
  state.openAIEndpoint = (event.target as HTMLInputElement).value;
};

const updateOpenAIKeyEndpoint = async () => {
  if (!hasFeature("bb.feature.plugin.openai")) {
    state.showFeatureModal = true;
    return;
  }

  if (!allowSave.value) {
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
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};

onMounted(() => {
  if (location.hash === "#ai-augmentation") {
    const container = containerRef.value;
    if (!container) return;
    scrollIntoView(container, {
      scrollMode: "if-needed",
    });
  }
});
</script>
