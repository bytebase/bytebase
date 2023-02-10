<template>
  <div class="px-4 py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <h1 class="text-2xl font-bold">
        {{ $t("settings.general.workspace.security") }}
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
          <BBCheckbox
            :disabled="!allowEdit"
            :value="watermarkEnabled"
            @toggle="handleWatermarkToggle"
          />
          <span class="font-medium">{{
            $t("settings.general.workspace.watermark.enable")
          }}</span>

          <span
            v-if="!allowEdit"
            class="text-sm text-gray-400 -translate-y-2 tooltip"
          >
            {{ $t("settings.general.workspace.watermark.only-owner-can-edit") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.watermark.description") }}
        </div>
      </div>
      <div class="mb-7 mt-5 lg:mt-0">
        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <BBCheckbox
            :value="disallowSignupEnabled"
            @toggle="handleDisallowSignupToggle"
          />
          <span class="font-medium">{{
            $t("settings.general.workspace.disallow-signup.enable")
          }}</span>

          <span
            v-if="!allowEdit"
            class="text-sm text-gray-400 -translate-y-2 tooltip"
          >
            {{
              $t(
                "settings.general.workspace.disallow-signup.only-owner-can-edit"
              )
            }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.disallow-signup.description") }}
        </div>
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
  useActuatorStore,
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
const actuatorStore = useActuatorStore();

const hasWatermarkFeature = featureToRef("bb.feature.branding");
const setting = useSettingByName("bb.workspace.watermark");

const allowEdit = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-general",
    currentUser.value.role
  );
});

const watermarkEnabled = computed((): boolean => {
  return setting.value?.value === "1";
});

const disallowSignupEnabled = computed((): boolean => {
  return actuatorStore.disallowSignup;
});

const handleDisallowSignupToggle = async (on: boolean) => {
  await actuatorStore.patchServerInfo({
    disallowSignup: on,
  });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};

const handleWatermarkToggle = async (on: boolean) => {
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
