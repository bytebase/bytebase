<template>
  <div class="mb-7 mt-4 lg:mt-0" @click="handleValueFieldClick">
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">{{
        $t("settings.general.workspace.sign-in-frequency.self")
      }}</span>
      <FeatureBadge feature="bb.feature.secure-token" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.sign-in-frequency.description") }}
    </p>
    <NTooltip placement="top-start" :disabled="allowChangeSetting">
      <template #trigger>
        <div class="mt-3 w-full flex flex-row justify-start items-center">
          <NInputNumber
            v-model:value="state.inputValue"
            class="w-20 mr-4"
            :disabled="!allowChangeSetting"
            :min="1"
            :max="state.timeFormat === 'HOURS' ? 23 : undefined"
            :precision="0"
          />
          <NRadioGroup
            v-model:value="state.timeFormat"
            :disabled="!allowChangeSetting"
          >
            <NRadio
              :value="'HOURS'"
              :label="$t('settings.general.workspace.sign-in-frequency.hours')"
            />
            <NRadio
              :value="'DAYS'"
              :label="$t('settings.general.workspace.sign-in-frequency.days')"
            />
          </NRadioGroup>
        </div>
      </template>
      <span class="text-sm text-gray-400 -translate-y-2">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </NTooltip>
  </div>

  <FeatureModal
    :open="!!state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { NInputNumber, NRadioGroup, NRadio } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { featureToRef, pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { FeatureType, defaultTokenDurationInHours } from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    inputValue: defaultTokenDurationInHours / 24,
    timeFormat: "DAYS",
  };
  const seconds =
    settingV1Store.workspaceProfileSetting?.tokenDuration?.seconds?.toNumber();
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

const props = defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(getInitialState());

const hasSecureTokenFeature = featureToRef("bb.feature.secure-token");

const allowChangeSetting = computed(() => {
  return hasSecureTokenFeature.value && props.allowEdit;
});

const handleValueFieldClick = () => {
  if (!hasSecureTokenFeature.value) {
    state.featureNameForModal = "bb.feature.secure-token";
    return;
  }
};

const handleFrequencySettingChange = useDebounceFn(async () => {
  const seconds =
    state.timeFormat === "HOURS"
      ? state.inputValue * 60 * 60
      : state.inputValue * 24 * 60 * 60;
  await settingV1Store.updateWorkspaceProfile({
    tokenDuration: Duration.fromPartial({ seconds, nanos: 0 }),
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.sign-in-frequency.config-updated"),
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
    if (!hasSecureTokenFeature.value) {
      state.featureNameForModal = "bb.feature.secure-token";
      return;
    }

    handleFrequencySettingChange();
  }
);
</script>
