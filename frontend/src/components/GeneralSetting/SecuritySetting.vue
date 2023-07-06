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
          <NCheckbox
            :disabled="!allowEdit"
            :checked="watermarkEnabled"
            @update:checked="handleWatermarkToggle"
          />
          <span class="font-medium">{{
            $t("settings.general.workspace.watermark.enable")
          }}</span>

          <FeatureBadge feature="bb.feature.watermark" />

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
          <NCheckbox
            :disabled="!allowEdit"
            :checked="disallowSignupEnabled"
            @update:checked="handleDisallowSignupToggle"
          />
          <span class="font-medium">{{
            $t("settings.general.workspace.disallow-signup.enable")
          }}</span>

          <FeatureBadge feature="bb.feature.disallow-signup" />

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
      <div class="mb-7 mt-5 lg:mt-0">
        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <NCheckbox
            :disabled="!allowEdit"
            :checked="require2FAEnabled"
            @update:checked="handleRequire2FAToggle"
          />
          <span class="font-medium">{{
            $t("settings.general.workspace.require-2fa.enable")
          }}</span>
          <FeatureBadge feature="bb.feature.2fa" />
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
    :open="state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { storeToRefs } from "pinia";
import { NCheckbox } from "naive-ui";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  useActuatorV1Store,
  useUserStore,
} from "@/store";
import { hasWorkspacePermissionV1 } from "@/utils";
import { useI18n } from "vue-i18n";
import { FeatureType } from "@/types";
import { UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { useSettingV1Store } from "@/store/modules/v1/setting";

interface LocalState {
  featureNameForModal?: FeatureType;
}
const state = reactive<LocalState>({});
const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();
const actuatorStore = useActuatorV1Store();

const { isSaaSMode } = storeToRefs(actuatorStore);
const hasWatermarkFeature = featureToRef("bb.feature.branding");
const has2FAFeature = featureToRef("bb.feature.2fa");
const hasDisallowSignupFeature = featureToRef("bb.feature.disallow-signup");

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-general",
    currentUserV1.value.userRole
  );
});
const watermarkEnabled = computed((): boolean => {
  return (
    settingV1Store.getSettingByName("bb.workspace.watermark")?.value
      ?.stringValue === "1"
  );
});
const disallowSignupEnabled = computed((): boolean => {
  return settingV1Store.workspaceProfileSetting?.disallowSignup ?? false;
});
const require2FAEnabled = computed((): boolean => {
  return settingV1Store.workspaceProfileSetting?.require2fa ?? false;
});

const handleDisallowSignupToggle = async (on: boolean) => {
  if (!hasDisallowSignupFeature.value) {
    state.featureNameForModal = "bb.feature.disallow-signup";
    return;
  }
  await settingV1Store.updateWorkspaceProfile({
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
    const userList = (await userStore.fetchUserList())
      .filter(
        (user) => user.userType === UserType.USER && user.state === State.ACTIVE
      )
      .filter((user) => !user.mfaEnabled);
    if (userList.length > 0) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t(
          "settings.general.workspace.require-2fa.need-all-user-2fa-enabled",
          {
            count: userList.length,
            users: userList.map((user) => user.email).join(", "),
          }
        ),
      });
      return;
    }
  }

  await settingV1Store.updateWorkspaceProfile({
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
  await settingV1Store.upsertSetting({
    name: "bb.workspace.watermark",
    value: {
      stringValue: value,
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.watermark.update-success"),
  });
};
</script>
