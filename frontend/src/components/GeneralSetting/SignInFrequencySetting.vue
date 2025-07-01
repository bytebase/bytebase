<template>
  <div class="mb-7 mt-4 lg:mt-0" @click="handleValueFieldClick">
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">{{
        $t("settings.general.workspace.sign-in-frequency.self")
      }}</span>
      <FeatureBadge :feature="PlanFeature.FEATURE_SIGN_IN_FREQUENCY_CONTROL" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.sign-in-frequency.description") }}
    </p>
    <NTooltip placement="top-start" :disabled="allowChangeSetting">
      <template #trigger>
        <div class="mt-3 w-full flex flex-row justify-start items-center">
          <NInputNumber
            v-model:value="state.inputValue"
            class="w-24 mr-4"
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
    :feature="PlanFeature.FEATURE_SIGN_IN_FREQUENCY_CONTROL"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { NInputNumber, NRadioGroup, NRadio, NTooltip } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { defaultTokenDurationInHours } from "@/types";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { create } from "@bufbuild/protobuf";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    inputValue: defaultTokenDurationInHours / 24,
    timeFormat: "DAYS",
    showFeatureModal: false,
  };
  const seconds =
    settingV1Store.workspaceProfileSetting?.tokenDuration?.seconds
      ? Number(settingV1Store.workspaceProfileSetting.tokenDuration.seconds)
      : undefined;
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
  showFeatureModal: boolean;
}

const props = defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(getInitialState());

const hasSecureTokenFeature = featureToRef(PlanFeature.FEATURE_SIGN_IN_FREQUENCY_CONTROL);

const allowChangeSetting = computed(() => {
  return hasSecureTokenFeature.value && props.allowEdit;
});

const handleValueFieldClick = () => {
  if (!hasSecureTokenFeature.value) {
    state.showFeatureModal = true;
    return;
  }
};

const handleFrequencySettingChange = async () => {
  const seconds =
    state.timeFormat === "HOURS"
      ? state.inputValue * 60 * 60
      : state.inputValue * 24 * 60 * 60;
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      tokenDuration: create(DurationSchema, { seconds: BigInt(seconds), nanos: 0 }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile_setting_value.token_duration"]
    }),
  });
};

watch(
  () => [state.timeFormat],
  () => {
    if (state.timeFormat === "HOURS" && state.inputValue > 23) {
      state.inputValue = 23;
    }
  }
);

defineExpose({
  isDirty: computed(() => !isEqual(getInitialState(), state)),
  update: handleFrequencySettingChange,
  revert: () => {
    Object.assign(state, getInitialState());
  },
});
</script>
