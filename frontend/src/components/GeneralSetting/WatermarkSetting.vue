<template>
  <div class="px-4 py-6">
    <div>
      <h1 class="text-2xl font-bold">
        {{ $t("settings.general.workspace.watermark.self") }}
      </h1>
    </div>
    <div class="mt-4 space-y-2">
      <div>
        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <BBCheckbox
            :disabled="!allowEdit"
            :value="enabled"
            @toggle="handleToggle"
          />
          <span class="font-medium">{{
            $t("settings.general.workspace.enable-watermark")
          }}</span>

          <span
            v-if="!allowEdit"
            class="text-sm text-gray-400 -translate-y-2 tooltip"
          >
            {{ $t("settings.general.workspace.watermark.only-owner-can-edit") }}
          </span>
        </label>
      </div>

      <div class="mb-3 text-sm text-gray-400">
        {{ $t("settings.general.workspace.watermark.description") }}
      </div>
    </div>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.watermark"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";

import {
  featureToRef,
  pushNotification,
  useCurrentUser,
  useSettingByName,
  useSettingStore,
} from "@/store";
import { BBCheckbox } from "@/bbkit";
import { hasWorkspacePermission } from "@/utils";
import { useI18n } from "vue-i18n";

interface LocalState {
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  showFeatureModal: false,
});

const { t } = useI18n();
const settingStore = useSettingStore();
const currentUser = useCurrentUser();

const hasWatermarkFeature = featureToRef("bb.feature.branding");
const setting = useSettingByName("bb.workspace.watermark");

const allowEdit = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-general",
    currentUser.value.role
  );
});

const enabled = computed((): boolean => {
  return setting.value?.value === "1";
});

const handleToggle = async (on: boolean) => {
  if (!hasWatermarkFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  const value = on ? "1" : "0";
  await settingStore.updateSettingByName({
    name: "bb.workspace.watermark",
    value,
  });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.watermark.update-success"),
  });
};
</script>
