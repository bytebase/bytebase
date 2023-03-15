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

          <FeatureBadge
            feature="bb.feature.plugin.openai"
            class="text-accent"
          />

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
                href="https://platform.openai.com/docs/introduction/overview"
                class="normal-link"
                target="_blank"
                >{{ $t("common.view-doc") }}</a
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
        <div class="flex">
          <button
            type="button"
            class="btn-primary ml-auto"
            :disabled="!allowSave"
            @click.prevent="updateOpenAIKey"
          >
            {{ $t("common.update") }}
          </button>
        </div>
      </div>
    </div>

    <FeatureModal
      v-if="state.showFeatureModal"
      feature="bb.feature.plugin.openai"
      @cancel="state.showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import {
  hasFeature,
  pushNotification,
  useCurrentUser,
  useSettingByName,
  useSettingStore,
} from "@/store";
import { hasWorkspacePermission } from "@/utils";
import FeatureBadge from "@/components/FeatureBadge.vue";
import FeatureModal from "@/components/FeatureModal.vue";
import scrollIntoView from "scroll-into-view-if-needed";

interface LocalState {
  openAIKey: string;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const settingStore = useSettingStore();
const currentUser = useCurrentUser();
const containerRef = ref<HTMLDivElement>();

const state = reactive<LocalState>({
  openAIKey: "",
  showFeatureModal: false,
});

const openAIKeySetting = useSettingByName("bb.plugin.openai.key");

watchEffect(() => {
  state.openAIKey = openAIKeySetting.value?.value ?? "";
});

const allowEdit = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-general",
    currentUser.value.role
  );
});

const allowSave = computed((): boolean => {
  return allowEdit.value && state.openAIKey !== openAIKeySetting.value?.value;
});

const handleOpenAIKeyChange = (event: InputEvent) => {
  state.openAIKey = (event.target as HTMLInputElement).value;
};

const updateOpenAIKey = async () => {
  if (!hasFeature("bb.feature.plugin.openai")) {
    state.showFeatureModal = true;
    return;
  }

  if (!allowSave.value) {
    return;
  }

  await settingStore.updateSettingByName({
    name: "bb.plugin.openai.key",
    value: state.openAIKey,
  });
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
