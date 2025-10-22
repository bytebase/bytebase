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
            v-model:value="state.tokenDuration"
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

  <div class="mb-7 mt-4 lg:mt-0" @click="handleValueFieldClick">
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">{{
        $t("settings.general.workspace.inactive-timeout.self")
      }}</span>
      <FeatureBadge :feature="PlanFeature.FEATURE_SIGN_IN_FREQUENCY_CONTROL" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.inactive-timeout.description") }}
    </p>
    <NTooltip placement="top-start" :disabled="allowChangeSetting">
      <template #trigger>
        <div class="mt-3 w-full flex flex-row justify-start items-center">
          <NInputNumber
            v-model:value="state.inactiveTimeout"
            class="w-24 mr-4"
            :disabled="!allowChangeSetting"
          />
          {{ $t("settings.general.workspace.inactive-timeout.hours") }}
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
import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { NInputNumber, NRadioGroup, NRadio, NTooltip } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import { featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { defaultTokenDurationInHours } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

interface LocalState {
  tokenDuration: number;
  inactiveTimeout: number;
  timeFormat: "HOURS" | "DAYS";
  showFeatureModal: boolean;
}

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    tokenDuration: defaultTokenDurationInHours / 24,
    inactiveTimeout: -1,
    timeFormat: "DAYS",
    showFeatureModal: false,
  };
  const tokenDurationSeconds = settingV1Store.workspaceProfileSetting
    ?.tokenDuration?.seconds
    ? Number(settingV1Store.workspaceProfileSetting.tokenDuration.seconds)
    : undefined;
  if (tokenDurationSeconds && tokenDurationSeconds > 0) {
    if (tokenDurationSeconds < 60 * 60 * 24) {
      defaultState.tokenDuration =
        Math.floor(tokenDurationSeconds / (60 * 60)) || 1;
      defaultState.timeFormat = "HOURS";
    } else {
      defaultState.tokenDuration =
        Math.floor(tokenDurationSeconds / (60 * 60 * 24)) || 1;
      defaultState.timeFormat = "DAYS";
    }
  }

  const inactiveTimeoutSeconds = Number(
    settingV1Store.workspaceProfileSetting?.inactiveSessionTimeout?.seconds ?? 0
  );
  if (inactiveTimeoutSeconds) {
    defaultState.inactiveTimeout =
      Math.floor(inactiveTimeoutSeconds / (60 * 60)) || 1;
  }
  return defaultState;
};

const props = defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(getInitialState());

const hasSecureTokenFeature = featureToRef(
  PlanFeature.FEATURE_SIGN_IN_FREQUENCY_CONTROL
);

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
      ? state.tokenDuration * 60 * 60
      : state.tokenDuration * 24 * 60 * 60;
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      tokenDuration: create(DurationSchema, {
        seconds: BigInt(seconds),
        nanos: 0,
      }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile_setting_value.token_duration"],
    }),
  });
};

const handleInactivityTimeoutSettingChange = async () => {
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      inactiveSessionTimeout: create(DurationSchema, {
        seconds: BigInt(state.inactiveTimeout * 60 * 60),
        nanos: 0,
      }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile_setting_value.inactive_session_timeout"],
    }),
  });
};

const handleUpdate = async () => {
  const initState = getInitialState();
  if (initState.inactiveTimeout !== state.inactiveTimeout) {
    await handleInactivityTimeoutSettingChange();
  }

  if (
    initState.tokenDuration !== state.tokenDuration ||
    initState.timeFormat !== state.timeFormat
  ) {
    await handleFrequencySettingChange();
  }
};

watch(
  () => [state.timeFormat],
  () => {
    if (state.timeFormat === "HOURS" && state.tokenDuration > 23) {
      state.tokenDuration = 23;
    }
  }
);

defineExpose({
  isDirty: computed(() => !isEqual(getInitialState(), state)),
  update: handleUpdate,
  revert: () => {
    Object.assign(state, getInitialState());
  },
});
</script>
