<template>
  <!-- Access Token Duration -->
  <div class="mb-7 mt-4 lg:mt-0" @click="handleValueFieldClick">
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">{{
        $t("settings.general.workspace.access-token-duration.self")
      }}</span>
      <FeatureBadge :feature="PlanFeature.FEATURE_TOKEN_DURATION_CONTROL" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.access-token-duration.description") }}
    </p>
    <NTooltip placement="top-start" :disabled="allowChangeSetting">
      <template #trigger>
        <div class="mt-3 w-full flex flex-row justify-start items-center">
          <NInputNumber
            v-model:value="state.accessTokenDuration"
            class="w-24 mr-4"
            :disabled="!allowChangeSetting"
            :min="1"
            :max="state.accessTokenTimeFormat === 'MINUTES' ? 59 : 23"
            :precision="0"
          />
          <NRadioGroup
            v-model:value="state.accessTokenTimeFormat"
            :disabled="!allowChangeSetting"
          >
            <NRadio
              :value="'MINUTES'"
              :label="$t('settings.general.workspace.access-token-duration.minutes')"
            />
            <NRadio
              :value="'HOURS'"
              :label="$t('settings.general.workspace.access-token-duration.hours')"
            />
          </NRadioGroup>
        </div>
      </template>
      <span class="text-sm text-gray-400 -translate-y-2">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </NTooltip>
  </div>

  <!-- Refresh Token Duration -->
  <div class="mb-7 mt-4 lg:mt-0" @click="handleValueFieldClick">
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">{{
        $t("settings.general.workspace.refresh-token-duration.self")
      }}</span>
      <FeatureBadge :feature="PlanFeature.FEATURE_TOKEN_DURATION_CONTROL" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.refresh-token-duration.description") }}
    </p>
    <NTooltip placement="top-start" :disabled="allowChangeSetting">
      <template #trigger>
        <div class="mt-3 w-full flex flex-row justify-start items-center">
          <NInputNumber
            v-model:value="state.refreshTokenDuration"
            class="w-24 mr-4"
            :disabled="!allowChangeSetting"
            :min="1"
            :max="state.refreshTokenTimeFormat === 'HOURS' ? 23 : undefined"
            :precision="0"
          />
          <NRadioGroup
            v-model:value="state.refreshTokenTimeFormat"
            :disabled="!allowChangeSetting"
          >
            <NRadio
              :value="'HOURS'"
              :label="$t('settings.general.workspace.refresh-token-duration.hours')"
            />
            <NRadio
              :value="'DAYS'"
              :label="$t('settings.general.workspace.refresh-token-duration.days')"
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
        $t("settings.general.workspace.inactive-session-timeout.self")
      }}</span>
      <FeatureBadge :feature="PlanFeature.FEATURE_TOKEN_DURATION_CONTROL" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{
        $t("settings.general.workspace.inactive-session-timeout.description")
      }}
      <span class="font-semibold! textinfolabel">
        {{ $t("settings.general.workspace.no-limit") }}
      </span>
    </p>
    <NTooltip placement="top-start" :disabled="allowChangeSetting">
      <template #trigger>
        <div class="mt-3 w-full flex flex-row justify-start items-center">
          <NInputNumber
            v-model:value="state.inactiveTimeout"
            class="w-24 mr-4"
            :min="-1"
            :disabled="!allowChangeSetting"
          />
          <span class="textinfo text-sm">
            {{
              $t("settings.general.workspace.inactive-session-timeout.hours")
            }}
          </span>
        </div>
      </template>
      <span class="text-sm text-gray-400 -translate-y-2">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </NTooltip>
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_TOKEN_DURATION_CONTROL"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { DurationSchema, FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { NInputNumber, NRadio, NRadioGroup, NTooltip } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import { featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  defaultAccessTokenDurationInHours,
  defaultRefreshTokenDurationInHours,
} from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

interface LocalState {
  accessTokenDuration: number;
  accessTokenTimeFormat: "MINUTES" | "HOURS";
  refreshTokenDuration: number;
  refreshTokenTimeFormat: "HOURS" | "DAYS";
  inactiveTimeout: number;
  showFeatureModal: boolean;
}

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    accessTokenDuration: defaultAccessTokenDurationInHours * 60,
    accessTokenTimeFormat: "MINUTES",
    refreshTokenDuration: defaultRefreshTokenDurationInHours / 24,
    refreshTokenTimeFormat: "DAYS",
    inactiveTimeout: -1,
    showFeatureModal: false,
  };

  // Access token duration
  const accessTokenSeconds = settingV1Store.workspaceProfileSetting
    ?.accessTokenDuration?.seconds
    ? Number(settingV1Store.workspaceProfileSetting.accessTokenDuration.seconds)
    : undefined;
  if (accessTokenSeconds && accessTokenSeconds > 0) {
    if (accessTokenSeconds < 60 * 60) {
      defaultState.accessTokenDuration =
        Math.floor(accessTokenSeconds / 60) || 1;
      defaultState.accessTokenTimeFormat = "MINUTES";
    } else {
      defaultState.accessTokenDuration =
        Math.floor(accessTokenSeconds / (60 * 60)) || 1;
      defaultState.accessTokenTimeFormat = "HOURS";
    }
  }

  // Refresh token duration
  const refreshTokenSeconds = settingV1Store.workspaceProfileSetting
    ?.refreshTokenDuration?.seconds
    ? Number(
        settingV1Store.workspaceProfileSetting.refreshTokenDuration.seconds
      )
    : undefined;
  if (refreshTokenSeconds && refreshTokenSeconds > 0) {
    if (refreshTokenSeconds < 60 * 60 * 24) {
      defaultState.refreshTokenDuration =
        Math.floor(refreshTokenSeconds / (60 * 60)) || 1;
      defaultState.refreshTokenTimeFormat = "HOURS";
    } else {
      defaultState.refreshTokenDuration =
        Math.floor(refreshTokenSeconds / (60 * 60 * 24)) || 1;
      defaultState.refreshTokenTimeFormat = "DAYS";
    }
  }

  // Inactive timeout
  const inactiveTimeoutSeconds = Number(
    settingV1Store.workspaceProfileSetting?.inactiveSessionTimeout?.seconds ?? 0
  );
  if (inactiveTimeoutSeconds) {
    defaultState.inactiveTimeout =
      Math.floor(inactiveTimeoutSeconds / (60 * 60)) || 0;
  }

  return defaultState;
};

const props = defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(getInitialState());

const hasSecureTokenFeature = featureToRef(
  PlanFeature.FEATURE_TOKEN_DURATION_CONTROL
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

const handleInactivityTimeoutSettingChange = async () => {
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      inactiveSessionTimeout: create(DurationSchema, {
        seconds: BigInt(state.inactiveTimeout * 60 * 60),
        nanos: 0,
      }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.inactive_session_timeout"],
    }),
  });
};

const handleAccessTokenDurationChange = async () => {
  const seconds =
    state.accessTokenTimeFormat === "MINUTES"
      ? state.accessTokenDuration * 60
      : state.accessTokenDuration * 60 * 60;
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      accessTokenDuration: create(DurationSchema, {
        seconds: BigInt(seconds),
        nanos: 0,
      }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.access_token_duration"],
    }),
  });
};

const handleRefreshTokenDurationChange = async () => {
  const seconds =
    state.refreshTokenTimeFormat === "HOURS"
      ? state.refreshTokenDuration * 60 * 60
      : state.refreshTokenDuration * 24 * 60 * 60;
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      refreshTokenDuration: create(DurationSchema, {
        seconds: BigInt(seconds),
        nanos: 0,
      }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.refresh_token_duration"],
    }),
  });
};

const handleUpdate = async () => {
  const initState = getInitialState();

  if (initState.inactiveTimeout !== state.inactiveTimeout) {
    await handleInactivityTimeoutSettingChange();
  }

  if (
    initState.accessTokenDuration !== state.accessTokenDuration ||
    initState.accessTokenTimeFormat !== state.accessTokenTimeFormat
  ) {
    await handleAccessTokenDurationChange();
  }

  if (
    initState.refreshTokenDuration !== state.refreshTokenDuration ||
    initState.refreshTokenTimeFormat !== state.refreshTokenTimeFormat
  ) {
    await handleRefreshTokenDurationChange();
  }
};

watch(
  () => [state.accessTokenTimeFormat],
  () => {
    if (
      state.accessTokenTimeFormat === "MINUTES" &&
      state.accessTokenDuration > 59
    ) {
      state.accessTokenDuration = 59;
    }
  }
);

watch(
  () => [state.refreshTokenTimeFormat],
  () => {
    if (
      state.refreshTokenTimeFormat === "HOURS" &&
      state.refreshTokenDuration > 23
    ) {
      state.refreshTokenDuration = 23;
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
