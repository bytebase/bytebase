<template>
  <div class="mb-7 mt-5 lg:mt-0">
    <p class="font-medium">{{ "Sign-in frequency" }}</p>
    <p class="text-sm text-gray-400">
      {{ "Sign-in frequency is the period before a user must sign in again." }}
    </p>
    <div class="mt-4 w-full flex flex-row justify-start items-center">
      <NInputNumber
        v-model:value="state.inputValue"
        class="w-20 mr-4"
        :disabled="!allowEdit"
        :min="1"
        :max="state.timeFormat === 'HOURS' ? 23 : undefined"
        :precision="0"
      />
      <NRadioGroup v-model:value="state.timeFormat" :disabled="!allowEdit">
        <NRadio :value="'HOURS'" :label="'Hour(s)'" />
        <NRadio :value="'DAYS'" :label="'Day(s)'" />
      </NRadioGroup>
    </div>
  </div>

  <FeatureModal
    :open="state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { NInputNumber, NRadioGroup, NRadio } from "naive-ui";
import { featureToRef, pushNotification, useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV1 } from "@/utils";
import { FeatureType } from "@/types";
import { useSettingV1Store } from "@/store/modules/v1/setting";

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    inputValue: 7,
    timeFormat: "DAYS",
  };
  const seconds =
    settingV1Store.workspaceProfileSetting?.refreshTokenDuration?.seconds;
  if (seconds && seconds > 0) {
    if (seconds < 60 * 60 * 24) {
      defaultState.inputValue = Math.floor(seconds / (60 * 60)) || 1;
      defaultState.timeFormat = "HOURS";
    } else {
      defaultState.inputValue = Math.floor(seconds / (60 * 60 * 24)) || 1;
      defaultState.timeFormat = "DAYS";
    }
  }
  return defaultState;
};

interface LocalState {
  inputValue: number;
  timeFormat: "HOURS" | "DAYS";
  featureNameForModal?: FeatureType;
}

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const currentUserV1 = useCurrentUserV1();
const state = reactive<LocalState>(getInitialState());

// TODO(steven): update feature name.
const hasSecureTokenFeature = featureToRef("bb.feature.secure-token");

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-general",
    currentUserV1.value.userRole
  );
});

const handleFrequencySettingChange = useDebounceFn(async () => {
  if (!hasSecureTokenFeature.value) {
    state.featureNameForModal = "bb.feature.secure-token";
    return;
  }

  const seconds =
    state.timeFormat === "HOURS"
      ? state.inputValue * 60 * 60
      : state.inputValue * 24 * 60 * 60;
  await settingV1Store.updateWorkspaceProfile({
    refreshTokenDuration: { seconds: seconds, nanos: 0 },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
}, 2000);

watch(
  () => [state.timeFormat],
  () => {
    if (state.timeFormat === "HOURS" && state.inputValue > 23) {
      state.inputValue = 23;
    }
  }
);

watch(
  () => [state.inputValue, state.timeFormat],
  () => {
    handleFrequencySettingChange();
  }
);
</script>
