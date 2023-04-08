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

          <FeatureBadge feature="bb.feature.watermark" class="text-accent" />

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
      <div v-if="!isSaaSMode" class="mb-7 mt-5 lg:mt-0">
        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <BBCheckbox
            :disabled="!allowEdit"
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
            {{ $t("settings.general.workspace.only-owner-can-edit") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.disallow-signup.description") }}
        </div>
      </div>
      <!-- TODO(steven): remove release guard later -->
      <div v-if="isDev" class="mb-7 mt-5 lg:mt-0">
        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <BBCheckbox
            :disabled="!allowEdit"
            :value="require2FAEnabled"
            @toggle="handleRequire2FAToggle"
          />
          <span class="font-medium">{{
            $t("settings.general.workspace.require-2fa.enable")
          }}</span>
          <FeatureBadge feature="bb.feature.2fa" class="text-accent" />
          <span
            v-if="!allowEdit"
            class="text-sm text-gray-400 -translate-y-2 tooltip"
          >
            {{ $t("settings.general.workspace.only-owner-can-edit") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.require-2fa.description") }}
        </div>
      </div>
    </div>
  </div>

  <FeatureModal
    v-if="state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { storeToRefs } from "pinia";
import {
  featureToRef,
  pushNotification,
  useCurrentUser,
  useSettingByName,
  useSettingStore,
  useActuatorStore,
  useUserStore,
} from "@/store";
import { BBCheckbox } from "@/bbkit";
import { hasWorkspacePermission } from "@/utils";
import { useI18n } from "vue-i18n";
import { FeatureType } from "@/types";
import { UserType } from "@/types/proto/v1/auth_service";

interface LocalState {
  featureNameForModal?: FeatureType;
}
const state = reactive<LocalState>({});
const { t } = useI18n();
const settingStore = useSettingStore();
const currentUser = useCurrentUser();
const userStore = useUserStore();
const actuatorStore = useActuatorStore();

const { isSaaSMode } = storeToRefs(actuatorStore);
const hasWatermarkFeature = featureToRef("bb.feature.branding");
const watermarkSetting = useSettingByName("bb.workspace.watermark");
const has2FAFeature = featureToRef("bb.feature.2fa");

const allowEdit = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-general",
    currentUser.value.role
  );
});
const watermarkEnabled = computed((): boolean => {
  return watermarkSetting.value?.value === "1";
});
const disallowSignupEnabled = computed((): boolean => {
  return settingStore.workspaceSetting?.disallowSignup ?? false;
});
const require2FAEnabled = computed((): boolean => {
  return settingStore.workspaceSetting?.require2fa ?? false;
});

const handleDisallowSignupToggle = async (on: boolean) => {
  await settingStore.updateWorkspaceProfile({
    disallowSignup: on,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};

const handleRequire2FAToggle = async (on: boolean) => {
  if (!has2FAFeature.value) {
    state.featureNameForModal = "bb.feature.2fa";
    return;
  }

  if (on) {
    // Only allow to enable this when all users have enabled 2FA.
    const userList = userStore.userList
      .filter((user) => user.userType === UserType.USER)
      .filter((user) => !user.mfaEnabled);
    if (userList.length > 0) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t(
          "settings.general.workspace.require-2fa.need-all-user-2fa-enabled"
        ),
      });
      return;
    }
  }

  await settingStore.updateWorkspaceProfile({
    require2fa: on,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};

const handleWatermarkToggle = async (on: boolean) => {
  if (!hasWatermarkFeature.value) {
    state.featureNameForModal = "bb.feature.watermark";
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
